package handler

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mkoteknik/ospanel/internal/adapter/system"
	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/pkg/safepath"
	"github.com/mkoteknik/ospanel/internal/store"
)

// FileHandler dosya yöneticisi
type FileHandler struct {
	store store.Store
	log   *logger.Logger
}

// FileInfo dosya/dizin bilgisi
type FileInfo struct {
	Name     string    `json:"name"`
	Path     string    `json:"path"`
	Type     string    `json:"type"` // "file" veya "dir"
	Size     int64     `json:"size"`
	Mode     string    `json:"mode"`
	Modified time.Time `json:"modified"`
}

// NewFileHandler yeni FileHandler
func NewFileHandler(s store.Store, log *logger.Logger) *FileHandler {
	return &FileHandler{store: s, log: log}
}

// checkDiskQuota kullanıcının disk kotasını kontrol eder, aşılırsa hata döner
func (h *FileHandler) checkDiskQuota(r *http.Request, additionalBytes int64) error {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		return nil
	}
	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil || user.QuotaLimit <= 0 {
		return nil // Kota limiti yoksa veya kullanici bulunamazsa izin ver
	}

	homeDir := user.HomeDir
	if homeDir == "" {
		return nil
	}

	usage, err := system.DiskUsage(homeDir)
	if err != nil {
		return nil // Hesaplanamazsa engelleme, sadece logla
	}

	quotaBytes := user.QuotaLimit * 1024 * 1024 // MB -> bytes
	if usage+additionalBytes > quotaBytes {
		usedMB := usage / (1024 * 1024)
		limitMB := user.QuotaLimit
		return fmt.Errorf("disk kotası aşıldı: %d MB / %d MB kullanım", usedMB, limitMB)
	}
	return nil
}

// getJail kullanıcının jail dizinini döndürür (homeDir)
func (h *FileHandler) getJail(r *http.Request) (string, error) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		return "", fmt.Errorf("yetkilendirme gerekli")
	}
	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		return "", err
	}
	// Admin için de kendi homeDir, global FS erişimi yok (güvenlik)
	jail := user.HomeDir
	if jail == "" {
		// Fallback: DataDir altında user klasörü
		jail = filepath.Join("/var/lib/ospanel", "homes", user.Username)
		_ = os.MkdirAll(jail, 0750)
	}
	// Jail yoksa oluştur
	_ = os.MkdirAll(jail, 0750)
	// Jail'in gerçek yolu (symlink çöz)
	if real, err := filepath.EvalSymlinks(jail); err == nil {
		jail = real
	}
	return filepath.Clean(jail), nil
}

// resolve kullanıcı girdisini jail içinde çözer
func (h *FileHandler) resolve(r *http.Request, userPath string) (string, error) {
	jail, err := h.getJail(r)
	if err != nil {
		return "", err
	}
	// Boş ise jail
	if userPath == "" {
		return jail, nil
	}
	// Özel: "/" jail root demek
	if userPath == "/" || userPath == "." {
		return jail, nil
	}
	return safepath.Resolve(jail, userPath)
}

// List dizin içeriğini listeler
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	absPath, err := h.resolve(r, path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol: " + err.Error()})
		return
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"path":        absPath,
			"files":       []FileInfo{},
			"breadcrumbs": getBreadcrumbs(absPath),
		})
		return
	}

	entries, err := os.ReadDir(absPath)
	if err != nil {
		h.log.Errorw("dizin okunamadı", "path", absPath, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dizin okunamadı: " + err.Error()})
		return
	}

	// Limit: max 5000 dosya
	if len(entries) > 5000 {
		entries = entries[:5000]
	}

	var files []FileInfo
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			continue
		}
		fileType := "file"
		if entry.IsDir() {
			fileType = "dir"
		}
		files = append(files, FileInfo{
			Name:     entry.Name(),
			Path:     filepath.Join(absPath, entry.Name()),
			Type:     fileType,
			Size:     info.Size(),
			Mode:     info.Mode().String(),
			Modified: info.ModTime(),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].Type != files[j].Type {
			return files[i].Type == "dir"
		}
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":        absPath,
		"files":       files,
		"breadcrumbs": getBreadcrumbs(absPath),
	})
}

// ReadFile dosya içeriğini okur
func (h *FileHandler) ReadFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	absPath, err := h.resolve(r, req.Path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Dosya bulunamadı"})
		return
	}
	if info.IsDir() {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dizin okunamaz, dosya seçin"})
		return
	}
	if info.Size() > 2*1024*1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dosya çok büyük (max 2MB)"})
		return
	}
	// Sadece text dosyalar
	if isBinaryFile(absPath) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Binary dosya okunamaz, indirin"})
		return
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya okunamadı"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":    absPath,
		"content": string(content),
		"size":    info.Size(),
	})
}

// WriteFile dosya içeriğini yazar
func (h *FileHandler) WriteFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}
	if len(req.Content) > 2*1024*1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "İçerik çok büyük (max 2MB)"})
		return
	}

	absPath, err := h.resolve(r, req.Path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	// Dosya jail içinde mi ve parent var mı?
	if err := os.MkdirAll(filepath.Dir(absPath), 0750); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dizin oluşturulamadı"})
		return
	}

	// Disk kota kontrolu
	if err := h.checkDiskQuota(r, int64(len(req.Content))); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	if err := os.WriteFile(absPath, []byte(req.Content), 0644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya yazılamadı"})
		return
	}

	h.log.Infow("dosya kaydedildi", "path", absPath)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Dosya kaydedildi"})
}

// DeleteFile dosya/dizin siler
func (h *FileHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Yol gerekli"})
		return
	}

	absPath, err := h.resolve(r, path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol: " + err.Error()})
		return
	}

	// Jail root silinemez
	jail, _ := h.getJail(r)
	if absPath == jail {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Kök dizin silinemez"})
		return
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Bulunamadı: " + absPath})
		} else if os.IsPermission(err) {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Erişim reddedildi (yetkisiz): " + absPath})
		} else {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya bilgisi alınamadı: " + err.Error()})
		}
		return
	}

	if info.IsDir() {
		if err := os.RemoveAll(absPath); err != nil {
			if os.IsPermission(err) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "Silme yetkisi yok: " + absPath + " — dizin başka kullanıcıya ait olabilir. Terminalden şunu çalıştırın: sudo chown -R metin:metin \"" + absPath + "\""})
			} else {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dizin silinemedi: " + err.Error()})
			}
			return
		}
	} else {
		if err := os.Remove(absPath); err != nil {
			if os.IsPermission(err) {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "Silme yetkisi yok: " + absPath + " — dosya başka kullanıcıya ait olabilir. Terminalden şunu çalıştırın: sudo chown metin:metin \"" + absPath + "\""})
			} else {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya silinemedi: " + err.Error()})
			}
			return
		}
	}

	h.log.Infow("silindi", "path", absPath)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Başarıyla silindi"})
}

// CreateDir yeni dizin oluşturur
func (h *FileHandler) CreateDir(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Parent string `json:"parent"`
		Name   string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}
	if req.Name == "" || strings.Contains(req.Name, "/") || strings.Contains(req.Name, "\\") || strings.Contains(req.Name, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz dizin adı"})
		return
	}

	parentAbs, err := h.resolve(r, req.Parent)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	newPath := filepath.Join(parentAbs, req.Name)
	// newPath de jail içinde mi kontrol
	if _, err := h.resolve(r, newPath); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	if err := os.MkdirAll(newPath, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dizin oluşturulamadı"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Dizin oluşturuldu", "path": newPath})
}

// Upload dosya yükleme
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(50 << 20); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Form çok büyük (max 50MB)"})
		return
	}

	dir := r.FormValue("dir")
	if dir == "" {
		dir = "/"
	}
	absDir, err := h.resolve(r, dir)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz dizin"})
		return
	}
	if _, err := os.Stat(absDir); os.IsNotExist(err) {
		_ = os.MkdirAll(absDir, 0755)
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dosya alınamadı"})
		return
	}
	defer file.Close()

	// Dosya adı temizle
	filename := filepath.Base(header.Filename)
	if filename == "." || filename == "/" || strings.Contains(filename, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz dosya adı"})
		return
	}

	destPath := filepath.Join(absDir, filename)
	if _, err := h.resolve(r, destPath); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	// Dosya boyutu limiti
	if header.Size > 50*1024*1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dosya çok büyük (max 50MB)"})
		return
	}

	// Disk kota kontrolu
	if err := h.checkDiskQuota(r, header.Size); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": err.Error()})
		return
	}

	dst, err := os.Create(destPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya oluşturulamadı"})
		return
	}
	defer dst.Close()

	// LimitReader ile kopyala
	limited := io.LimitReader(file, 50*1024*1024+1)
	n, _ := io.Copy(dst, limited)
	if n > 50*1024*1024 {
		_ = os.Remove(destPath)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dosya çok büyük (max 50MB)"})
		return
	}

	h.log.Infow("dosya yüklendi", "path", destPath, "size", header.Size)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Dosya yüklendi: " + filename})
}

// CreateArchive zip arşivi oluşturur
func (h *FileHandler) CreateArchive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Paths []string `json:"paths"`
		Name  string   `json:"name"`
		Dir   string   `json:"dir"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}
	if len(req.Paths) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "En az bir yol gerekli"})
		return
	}
	if req.Name == "" {
		req.Name = "archive.zip"
	}
	if !strings.HasSuffix(req.Name, ".zip") {
		req.Name += ".zip"
	}
	// Name temizle
	req.Name = filepath.Base(req.Name)

	baseDir := req.Dir
	if baseDir == "" {
		baseDir = "/"
	}
	baseAbs, err := h.resolve(r, baseDir)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz dizin"})
		return
	}
	zipPath := filepath.Join(baseAbs, req.Name)
	if _, err := h.resolve(r, zipPath); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz arşiv yolu"})
		return
	}

	zipFile, err := os.Create(zipPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Arşiv oluşturulamadı"})
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, p := range req.Paths {
		absP, err := h.resolve(r, p)
		if err != nil {
			continue
		}
		info, err := os.Stat(absP)
		if err != nil {
			continue
		}
		if info.IsDir() {
			_ = filepath.Walk(absP, func(path string, fi os.FileInfo, err error) error {
				if err != nil {
					return nil
				}
				relPath, _ := filepath.Rel(baseAbs, path)
				if relPath == "." {
					return nil
				}
				_ = addToZip(zipWriter, path, relPath, fi)
				return nil
			})
		} else {
			relName, _ := filepath.Rel(baseAbs, absP)
			_ = addToZip(zipWriter, absP, relName, info)
		}
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Arşiv oluşturuldu: " + req.Name})
}

// ExtractArchive zip/tar.gz arşivi açar
func (h *FileHandler) ExtractArchive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Dest string `json:"dest"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	absPath, err := h.resolve(r, req.Path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz arşiv yolu"})
		return
	}
	absDest, err := h.resolve(r, req.Dest)
	if err != nil {
		// Dest boşsa arşivin bulunduğu dizin
		absDest = filepath.Dir(absPath)
	}
	if _, err := os.Stat(absPath); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Arşiv bulunamadı"})
		return
	}

	if strings.HasSuffix(absPath, ".zip") {
		if err := extractZipSafe(absPath, absDest); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Arşiv açılamadı: " + err.Error()})
			return
		}
	} else if strings.HasSuffix(absPath, ".tar.gz") || strings.HasSuffix(absPath, ".tgz") {
		if err := extractTarGz(absPath, absDest); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Arşiv açılamadı: " + err.Error()})
			return
		}
	} else {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Desteklenmeyen format (zip, tar.gz)"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Arşiv açıldı"})
}

// Download dosya indirir
func (h *FileHandler) Download(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Yol gerekli"})
		return
	}
	absPath, err := h.resolve(r, path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	info, err := os.Stat(absPath)
	if err != nil || info.IsDir() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Dosya bulunamadı"})
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+info.Name())
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", info.Size()))
	http.ServeFile(w, r, absPath)
}

// Chmod dosya/dizin izinlerini değiştirir
func (h *FileHandler) Chmod(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Mode string `json:"mode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	absPath, err := h.resolve(r, req.Path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	var mode os.FileMode
	if _, err := fmt.Sscanf(req.Mode, "%o", &mode); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz izin formatı (örn: 0755)"})
		return
	}
	// Sadece 0644,0755,0600 gibi güvenli modlar
	if mode > 0777 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz izin"})
		return
	}

	if err := os.Chmod(absPath, mode); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "İzinler değiştirilemedi"})
		return
	}

	h.log.Infow("chmod", "path", absPath, "mode", req.Mode)
	writeJSON(w, http.StatusOK, map[string]string{"message": "İzinler güncellendi"})
}

// Rename dosya/dizin adını değiştirir
func (h *FileHandler) Rename(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path    string `json:"path"`
		NewName string `json:"new_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}
	if req.NewName == "" || strings.Contains(req.NewName, "/") || strings.Contains(req.NewName, "\\") || strings.Contains(req.NewName, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz yeni ad"})
		return
	}

	absPath, err := h.resolve(r, req.Path)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	newPath := filepath.Join(filepath.Dir(absPath), req.NewName)
	if _, err := h.resolve(r, newPath); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz hedef"})
		return
	}

	if err := os.Rename(absPath, newPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Yeniden adlandırılamadı"})
		return
	}

	h.log.Infow("rename", "from", absPath, "to", newPath)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Yeniden adlandırıldı", "new_path": newPath})
}

// CreateFile yeni boş dosya oluşturur
func (h *FileHandler) CreateFile(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Dir  string `json:"dir"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}
	if req.Name == "" || strings.Contains(req.Name, "/") || strings.Contains(req.Name, "\\") || strings.Contains(req.Name, "..") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz dosya adı"})
		return
	}

	absDir, err := h.resolve(r, req.Dir)
	if err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz dizin"})
		return
	}

	newPath := filepath.Join(absDir, req.Name)
	if _, err := h.resolve(r, newPath); err != nil {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	if _, err := os.Stat(newPath); err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Dosya zaten var"})
		return
	}

	if err := os.WriteFile(newPath, []byte(""), 0644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya oluşturulamadı"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Dosya oluşturuldu", "path": newPath})
}

// --- Yardımcılar ---

func getBreadcrumbs(path string) []map[string]string {
	parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
	var crumbs []map[string]string
	accum := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		if filepath.IsAbs(path) && accum == "" {
			accum = string(filepath.Separator) + part
		} else {
			accum = filepath.Join(accum, part)
		}
		crumbs = append(crumbs, map[string]string{
			"name": part,
			"path": accum,
		})
	}
	return crumbs
}

func addToZip(zw *zip.Writer, filePath, zipPath string, info os.FileInfo) error {
	if info.IsDir() {
		_, _ = zw.CreateHeader(&zip.FileHeader{Name: zipPath + "/", Method: zip.Store})
		return nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}
	_, err = io.Copy(writer, f)
	return err
}

func extractZipSafe(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		// ZipSlip koruması
		cleanName := filepath.Clean(f.Name)
		if strings.Contains(cleanName, "..") {
			continue
		}
		path := filepath.Join(dest, cleanName)
		// Jail içinde mi?
		if !strings.HasPrefix(filepath.Clean(path), filepath.Clean(dest)) {
			continue
		}
		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(path, f.Mode())
			continue
		}
		_ = os.MkdirAll(filepath.Dir(path), 0755)
		rc, err := f.Open()
		if err != nil {
			continue
		}
		dst, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			continue
		}
		_, _ = io.CopyN(dst, rc, 20*1024*1024) // max 20MB per file
		rc.Close()
		dst.Close()
	}
	return nil
}

func extractZip(src, dest string) error {
	return extractZipSafe(src, dest)
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar okuma hatası: %w", err)
		}

		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(dest)+string(filepath.Separator)) {
			return fmt.Errorf("güvenlik: geçersiz arşiv yolu: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("dizin oluşturulamadı %s: %w", target, err)
			}
		case tar.TypeReg:
			_ = os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("dosya oluşturulamadı %s: %w", target, err)
			}
			if _, err := io.CopyN(outFile, tr, header.Size); err != nil && err != io.EOF {
				outFile.Close()
				return fmt.Errorf("dosya yazma hatası %s: %w", target, err)
			}
			outFile.Close()
		case tar.TypeSymlink:
			// Symlink atla (güvenlik)
			continue
		case tar.TypeLink:
			continue
		}
	}
	return nil
}

func isBinaryFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()
	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	if n == 0 {
		return false
	}
	// Null byte varsa binary
	for _, b := range buf[:n] {
		if b == 0 {
			return true
		}
	}
	return false
}
