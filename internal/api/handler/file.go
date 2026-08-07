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

	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
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

// List dizin içeriğini listeler
func (h *FileHandler) List(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		path = "/"
	}

	// Güvenlik: path traversal engelle
	path = filepath.Clean(path)
	if strings.Contains(path, "..") {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Geçersiz yol"})
		return
	}

	// Mutlak yola çevir (Windows/Linux uyumlu)
	var absPath string
	if filepath.IsAbs(path) {
		absPath = path
	} else {
		// Varsayılan: kullanıcının home dizini
		userID, _ := middleware.GetUserID(r.Context())
		user, err := h.store.GetUser(r.Context(), userID)
		if err == nil && user.HomeDir != "" {
			absPath = filepath.Join(user.HomeDir, "public_html")
		} else {
			absPath, _ = os.Getwd()
		}
	}

	// Dizin yoksa oluşturmayı dene
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"path":       absPath,
			"files":      []FileInfo{},
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

	// Dizinler önce, sonra dosyalar; alfabetik
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

	// Dosya boyutu kontrolü (max 1MB)
	info, err := os.Stat(req.Path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Dosya bulunamadı"})
		return
	}
	if info.Size() > 1024*1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dosya çok büyük (max 1MB)"})
		return
	}

	content, err := os.ReadFile(req.Path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya okunamadı"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"path":    req.Path,
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

	if err := os.WriteFile(req.Path, []byte(req.Content), 0644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya yazılamadı"})
		return
	}

	h.log.Infow("dosya kaydedildi", "path", req.Path)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Dosya kaydedildi"})
}

// DeleteFile dosya/dizin siler
func (h *FileHandler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	if path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Yol gerekli"})
		return
	}

	info, err := os.Stat(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Bulunamadı"})
		return
	}

	if info.IsDir() {
		if err := os.RemoveAll(path); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dizin silinemedi"})
			return
		}
	} else {
		if err := os.Remove(path); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya silinemedi"})
			return
		}
	}

	h.log.Infow("silindi", "path", path)
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

	newPath := filepath.Join(req.Parent, req.Name)
	if err := os.MkdirAll(newPath, 0755); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dizin oluşturulamadı"})
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"message": "Dizin oluşturuldu", "path": newPath})
}

// Upload dosya yükleme
func (h *FileHandler) Upload(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(50 << 20) // 50MB max

	dir := r.FormValue("dir")
	if dir == "" {
		dir = "."
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Dosya alınamadı"})
		return
	}
	defer file.Close()

	destPath := filepath.Join(dir, header.Filename)
	dst, err := os.Create(destPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Dosya oluşturulamadı"})
		return
	}
	defer dst.Close()

	io.Copy(dst, file)

	h.log.Infow("dosya yüklendi", "path", destPath, "size", header.Size)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Dosya yüklendi: " + header.Filename})
}

// CreateArchive zip arşivi oluşturur
func (h *FileHandler) CreateArchive(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Paths []string `json:"paths"`
		Name  string   `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if req.Name == "" {
		req.Name = "archive.zip"
	}
	if !strings.HasSuffix(req.Name, ".zip") {
		req.Name += ".zip"
	}

	zipFile, err := os.Create(req.Name)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Arşiv oluşturulamadı"})
		return
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, path := range req.Paths {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}

		if info.IsDir() {
			filepath.Walk(path, func(p string, fi os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				relPath, _ := filepath.Rel(filepath.Dir(path), p)
				addToZip(zipWriter, p, relPath, fi)
				return nil
			})
		} else {
			addToZip(zipWriter, path, info.Name(), info)
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

	if strings.HasSuffix(req.Path, ".zip") {
		if err := extractZip(req.Path, req.Dest); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Arşiv açılamadı: " + err.Error()})
			return
		}
	} else if strings.HasSuffix(req.Path, ".tar.gz") || strings.HasSuffix(req.Path, ".tgz") {
		if err := extractTarGz(req.Path, req.Dest); err != nil {
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

	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Dosya bulunamadı"})
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+info.Name())
	w.Header().Set("Content-Type", "application/octet-stream")
	http.ServeFile(w, r, path)
}

// Chmod dosya/dizin izinlerini değiştirir
func (h *FileHandler) Chmod(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Path string `json:"path"`
		Mode string `json:"mode"` // "0755", "0644" gibi
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	var mode os.FileMode
	if _, err := fmt.Sscanf(req.Mode, "%o", &mode); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz izin formatı (örn: 0755)"})
		return
	}

	if err := os.Chmod(req.Path, mode); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "İzinler değiştirilemedi"})
		return
	}

	h.log.Infow("chmod", "path", req.Path, "mode", req.Mode)
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

	newPath := filepath.Join(filepath.Dir(req.Path), req.NewName)
	if err := os.Rename(req.Path, newPath); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Yeniden adlandırılamadı"})
		return
	}

	h.log.Infow("rename", "from", req.Path, "to", newPath)
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

	newPath := filepath.Join(req.Dir, req.Name)
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

func addToZip(zw *zip.Writer, filePath, zipPath string, info os.FileInfo) {
	if info.IsDir() {
		zw.CreateHeader(&zip.FileHeader{Name: zipPath + "/", Method: zip.Store})
		return
	}

	f, err := os.Open(filePath)
	if err != nil {
		return
	}
	defer f.Close()

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return
	}
	header.Name = zipPath
	header.Method = zip.Deflate

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return
	}
	io.Copy(writer, f)
}

func extractZip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		path := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(path, f.Mode())
			continue
		}
		os.MkdirAll(filepath.Dir(path), 0755)
		rc, _ := f.Open()
		dst, _ := os.Create(path)
		io.Copy(dst, rc)
		rc.Close()
		dst.Close()
	}
	return nil
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

		// Path traversal koruması
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
			os.MkdirAll(filepath.Dir(target), 0755)
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("dosya oluşturulamadı %s: %w", target, err)
			}
			if _, err := io.CopyN(outFile, tr, header.Size); err != nil {
				outFile.Close()
				return fmt.Errorf("dosya yazma hatası %s: %w", target, err)
			}
			outFile.Close()
		case tar.TypeSymlink:
			os.Symlink(header.Linkname, target)
		case tar.TypeLink:
			os.Link(filepath.Join(dest, header.Linkname), target)
		}
	}
	return nil
}
