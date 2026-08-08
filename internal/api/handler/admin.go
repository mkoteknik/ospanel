package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mkoteknik/ospanel/internal/adapter/cgroups"
	"github.com/mkoteknik/ospanel/internal/adapter/jail"
	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// AdminHandler admin panel işlemleri
type AdminHandler struct {
	store store.Store
	log   *logger.Logger
	jail  *jail.SSHManager
	cg    *cgroups.Limiter
}

// NewAdminHandler yeni AdminHandler oluşturur
func NewAdminHandler(s store.Store, log *logger.Logger) *AdminHandler {
	return &AdminHandler{store: s, log: log, jail: jail.NewSSHManager(), cg: cgroups.NewLimiter()}
}

// GetHostname sunucu hostname bilgisini döndürür
func (h *AdminHandler) GetHostname(w http.ResponseWriter, r *http.Request) {
	out, _ := exec.Command("hostnamectl", "--static").CombinedOutput()
	out2, _ := exec.Command("hostname", "-f").CombinedOutput()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"hostname":   strings.TrimSpace(string(out)),
		"fqdn":       strings.TrimSpace(string(out2)),
		"ip":         getServerIP(),
	})
}

// SetHostname sunucu hostname'ini değiştirir
func (h *AdminHandler) SetHostname(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Hostname string `json:"hostname"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Hostname == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz hostname"})
		return
	}

	// IP kontrolü - hostname'in DNS'te bu IP'ye çözümlendiğini kontrol et
	// (production'da daha sıkı kontrol yapılabilir)

	cmd := exec.Command("hostnamectl", "set-hostname", req.Hostname)
	if out, err := cmd.CombinedOutput(); err != nil {
		h.log.Errorw("hostname değiştirilemedi", "error", err, "out", string(out))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Hostname değiştirilemedi: " + string(out)})
		return
	}

	// Postfix'i güncelle
	domain := req.Hostname
	if idx := strings.Index(domain, "."); idx != -1 {
		domain = domain[idx+1:]
		exec.Command("postconf", "-e", "myhostname = "+req.Hostname).Run()
		exec.Command("postconf", "-e", "mydomain = "+domain).Run()
		exec.Command("systemctl", "reload", "postfix").Run()
	}

	h.log.Infow("hostname değiştirildi", "hostname", req.Hostname)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Hostname değiştirildi: " + req.Hostname})
}

func getServerIP() string {
	out, _ := exec.Command("hostname", "-I").CombinedOutput()
	ips := strings.Fields(string(out))
	if len(ips) > 0 {
		return ips[0]
	}
	return "127.0.0.1"
}

// ListUsers tüm kullanıcıları listeler (admin: herkes, reseller: sadece kendi kullanicilari)
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())
	callerID, _ := middleware.GetUserID(r.Context())

	var users []*model.User
	var err error

	if callerRole == "admin" {
		users, err = h.store.ListUsers(r.Context())
	} else if callerRole == "reseller" {
		users, err = h.store.ListUsersByReseller(r.Context(), callerID)
	} else {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu islem icin yetkiniz yok"})
		return
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Kullanıcılar listelenemedi"})
		return
	}

	if users == nil {
		users = []*model.User{}
	}

	// Hassas bilgileri temizle
	jailStatuses := make(map[int64]bool)
	for _, u := range users {
		u.PasswordHash = ""
		u.TOTPSecret = ""
		jailStatuses[u.ID] = h.jail.IsJailed(u.Username)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users":  users,
		"total":  len(users),
		"jailed": jailStatuses,
	})
}

// CreateUser yeni kullanıcı oluşturur (admin/reseller, quota kontrollu)
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())
	callerID, _ := middleware.GetUserID(r.Context())

	var req struct {
		Username     string `json:"username"`
		Email        string `json:"email"`
		Password     string `json:"password"`
		Role         string `json:"role"`
		MaxDomains   int    `json:"max_domains"`
		MaxEmails    int    `json:"max_emails"`
		MaxDatabases int    `json:"max_databases"`
		QuotaLimit   int64  `json:"quota_limit"` // MB disk
		EnableJail   bool   `json:"enable_jail"` // SFTP chroot jail
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	// Reseller sadece 'user' rolunde kullanici olusturabilir
	if callerRole == "reseller" {
		if req.Role != "user" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Reseller sadece 'user' kullanicisi olusturabilir"})
			return
		}
		// Kota kontrolu
		count, _ := h.store.CountUsersByReseller(r.Context(), callerID)
		maxUsers := h.getSettingInt(r.Context(), "reseller_max_users", 50)
		if count >= maxUsers {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": fmt.Sprintf("Kullanici limitine ulastiniz (max %d)", maxUsers)})
			return
		}
	}

	// Kullanici adi validasyonu
	if len(req.Username) < 3 || len(req.Username) > 32 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Kullanici adi 3-32 karakter olmali"})
		return
	}
	if len(req.Password) < 8 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Sifre en az 8 karakter olmali"})
		return
	}

	// Kullanici adi zaten var mi?
	if existing, _ := h.store.GetUserByUsername(r.Context(), req.Username); existing != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Bu kullanici adi zaten kullaniliyor"})
		return
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Şifre hashlenemedi"})
		return
	}

	// Varsayilan degerler
	if req.MaxDomains <= 0 { req.MaxDomains = 10 }
	if req.MaxEmails <= 0 { req.MaxEmails = 20 }
	if req.MaxDatabases <= 0 { req.MaxDatabases = 10 }
	if req.QuotaLimit <= 0 { req.QuotaLimit = 5120 } // 5GB

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         model.UserRole(req.Role),
		HomeDir:      "/home/" + req.Username,
		Status:       model.StatusActive,
		MaxDomains:   req.MaxDomains,
		MaxEmails:    req.MaxEmails,
		MaxDatabases: req.MaxDatabases,
		QuotaLimit:   req.QuotaLimit,
	}

	// Reseller kendi ID'sini parent olarak ekler
	if callerRole == "reseller" {
		user.ResellerID = &callerID
	}

	if err := h.store.CreateUser(r.Context(), user); err != nil {
		h.log.Errorw("kullanıcı oluşturulamadı", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Kullanıcı oluşturulamadı: " + err.Error()})
		return
	}

	// SFTP Jail kurulumu
	jailStatus := "disabled"
	if req.EnableJail {
		if err := h.jail.SetupChrootJail(user.Username, user.HomeDir); err != nil {
			h.log.Warnw("jail kurulumu basarisiz", "username", user.Username, "error", err)
			jailStatus = "failed: " + err.Error()
		} else {
			jailStatus = "enabled"
		}
	}

	h.log.Infow("kullanıcı oluşturuldu", "username", req.Username, "role", req.Role, "created_by", callerRole, "jail", jailStatus)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user":    user,
		"message": "Kullanıcı başarıyla oluşturuldu",
		"jail":    jailStatus,
	})
}

// UpdateUser kullanıcı günceller (reseller: sadece kendi kullanicilarini)
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())
	callerID, _ := middleware.GetUserID(r.Context())

	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	user, err := h.store.GetUser(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	// Reseller sadece kendi alt kullanicilarini guncelleyebilir
	if callerRole == "reseller" {
		if user.ResellerID == nil || *user.ResellerID != callerID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu kullaniciyi guncelleme yetkiniz yok"})
			return
		}
	}

	var updates map[string]interface{}
	json.NewDecoder(r.Body).Decode(&updates)

	if v, ok := updates["status"]; ok {
		if s, ok := v.(string); ok {
			user.Status = model.UserStatus(s)
		}
	}
	if v, ok := updates["role"]; ok {
		if s, ok := v.(string); ok {
			// Reseller kullaniciyi admin/reseller yapamaz
			if callerRole == "reseller" && (s == "admin" || s == "reseller") {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu rol ataması için yetkiniz yok"})
				return
			}
			user.Role = model.UserRole(s)
		}
	}
	if v, ok := updates["quota_limit"]; ok {
		user.QuotaLimit = int64(v.(float64))
	}
	if v, ok := updates["max_domains"]; ok {
		user.MaxDomains = int(v.(float64))
	}
	if v, ok := updates["max_emails"]; ok {
		user.MaxEmails = int(v.(float64))
	}
	if v, ok := updates["max_databases"]; ok {
		user.MaxDatabases = int(v.(float64))
	}

	user.UpdatedAt = time.Now()
	h.store.UpdateUser(r.Context(), user)

	writeJSON(w, http.StatusOK, user)
}

// DeleteUser kullanıcı siler (reseller: sadece kendi kullanicilarini)
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	callerRole, _ := middleware.GetUserRole(r.Context())
	callerID, _ := middleware.GetUserID(r.Context())

	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	// Reseller sadece kendi alt kullanicilarini silebilir
	if callerRole == "reseller" {
		user, err := h.store.GetUser(r.Context(), id)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
			return
		}
		if user.ResellerID == nil || *user.ResellerID != callerID {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu kullaniciyi silme yetkiniz yok"})
			return
		}
	}

	if err := h.store.DeleteUser(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Kullanıcı silinemedi"})
		return
	}

	h.log.Infow("kullanici silindi", "user_id", id, "deleted_by", callerRole)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Kullanıcı silindi"})
}

// getSettingInt ayar degerini int olarak okur
// ToggleJail kullanici SFTP jail durumunu acar/kapatir
func (h *AdminHandler) ToggleJail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	user, err := h.store.GetUser(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	// Sahiplik kontrolü (reseller sadece kendi kullanicilari)
	callerRole, _ := middleware.GetUserRole(r.Context())
	callerID, _ := middleware.GetUserID(r.Context())
	if callerRole == "reseller" && (user.ResellerID == nil || *user.ResellerID != callerID) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu kullaniciyi yonetme yetkiniz yok"})
		return
	}

	if h.jail.IsJailed(user.Username) {
		h.jail.RemoveChrootJail(user.Username, user.HomeDir)
		h.log.Infow("jail kaldirildi", "username", user.Username)
		writeJSON(w, http.StatusOK, map[string]interface{}{"message": "SFTP Jail kaldırıldı", "jailed": false})
	} else {
		if err := h.jail.SetupChrootJail(user.Username, user.HomeDir); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Jail kurulamadı: " + err.Error()})
			return
		}
		h.log.Infow("jail aktif", "username", user.Username)
		writeJSON(w, http.StatusOK, map[string]interface{}{"message": "SFTP Jail aktifleştirildi", "jailed": true})
	}
}

// ListPackages hosting paketlerini listeler
func (h *AdminHandler) ListPackages(w http.ResponseWriter, r *http.Request) {
	pkgs, err := h.store.ListPackages(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Paketler listelenemedi"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"packages": pkgs, "total": len(pkgs)})
}

// AssignPackage kullaniciya paket atar + cgroups limitlerini uygular
func (h *AdminHandler) AssignPackage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	user, err := h.store.GetUser(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	var req struct {
		PackageID int64 `json:"package_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	pkg, err := h.store.GetPackage(r.Context(), req.PackageID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Paket bulunamadı"})
		return
	}

	if err := h.store.UpdateUserPackage(r.Context(), user.ID, pkg.ID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Paket atanamadı"})
		return
	}

	cgResult := "skipped"
	if h.cg.IsAvailable() {
		if err := h.cg.ApplyLimits(user.Username, pkg.CPUShares, pkg.MemoryMB, pkg.Nproc); err != nil {
			cgResult = "failed: " + err.Error()
		} else {
			cgResult = "applied"
		}
	}

	user.QuotaLimit = int64(pkg.DiskMB)
	user.MaxDomains = pkg.MaxDomains
	user.MaxEmails = pkg.MaxEmails
	user.MaxDatabases = pkg.MaxDB
	h.store.UpdateUser(r.Context(), user)

	h.log.Infow("paket atandi", "username", user.Username, "package", pkg.Name, "cgroups", cgResult)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Paket atandı: " + pkg.Name, "package": pkg, "cgroups": cgResult,
	})
}

// CreatePackage yeni hosting paketi olusturur
func (h *AdminHandler) CreatePackage(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name       string `json:"name"`
		CPUShares  int    `json:"cpu_shares"`
		MemoryMB   int    `json:"memory_mb"`
		Nproc      int    `json:"nproc"`
		DiskMB     int    `json:"disk_mb"`
		MaxDomains int    `json:"max_domains"`
		MaxEmails  int    `json:"max_emails"`
		MaxDB      int    `json:"max_db"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek - paket adı gerekli"})
		return
	}

	pkg := &model.HostingPackage{
		Name: req.Name, CPUShares: req.CPUShares, MemoryMB: req.MemoryMB,
		Nproc: req.Nproc, DiskMB: req.DiskMB, MaxDomains: req.MaxDomains,
		MaxEmails: req.MaxEmails, MaxDB: req.MaxDB,
	}
	if pkg.CPUShares == 0 { pkg.CPUShares = 1024 }
	if pkg.MemoryMB == 0 { pkg.MemoryMB = 1024 }
	if pkg.Nproc == 0 { pkg.Nproc = 50 }

	if err := h.store.CreatePackage(r.Context(), pkg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Paket oluşturulamadı: " + err.Error()})
		return
	}
	h.log.Infow("paket olusturuldu", "name", pkg.Name)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"package": pkg, "message": "Paket oluşturuldu"})
}

// UpdatePackage paket gunceller
func (h *AdminHandler) UpdatePackage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	pkg, err := h.store.GetPackage(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Paket bulunamadı"})
		return
	}
	var updates map[string]interface{}
	json.NewDecoder(r.Body).Decode(&updates)
	if v, ok := updates["name"]; ok { pkg.Name = v.(string) }
	if v, ok := updates["cpu_shares"]; ok { pkg.CPUShares = int(v.(float64)) }
	if v, ok := updates["memory_mb"]; ok { pkg.MemoryMB = int(v.(float64)) }
	if v, ok := updates["nproc"]; ok { pkg.Nproc = int(v.(float64)) }
	if v, ok := updates["disk_mb"]; ok { pkg.DiskMB = int(v.(float64)) }
	if v, ok := updates["max_domains"]; ok { pkg.MaxDomains = int(v.(float64)) }
	if v, ok := updates["max_emails"]; ok { pkg.MaxEmails = int(v.(float64)) }
	if v, ok := updates["max_db"]; ok { pkg.MaxDB = int(v.(float64)) }
	if err := h.store.UpdatePackage(r.Context(), pkg); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Paket güncellenemedi"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"package": pkg, "message": "Paket güncellendi"})
}

// DeletePackage paket siler
func (h *AdminHandler) DeletePackage(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.store.DeletePackage(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Paket silinemedi"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Paket silindi"})
}

func (h *AdminHandler) getSettingInt(ctx context.Context, key string, defaultVal int) int {
	s, err := h.store.GetSetting(ctx, key)
	if err != nil || s == nil {
		return defaultVal
	}
	val, err := strconv.Atoi(s.Value)
	if err != nil {
		return defaultVal
	}
	return val
}

// GetSettings sistem ayarlarını getirir
func (h *AdminHandler) GetSettings(w http.ResponseWriter, r *http.Request) {
	settings, err := h.store.ListSettings(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Ayarlar alınamadı"})
		return
	}

	if settings == nil {
		settings = []*model.Setting{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"settings": settings,
	})
}

// UpdateSettings sistem ayarlarını günceller
func (h *AdminHandler) UpdateSettings(w http.ResponseWriter, r *http.Request) {
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	for key, value := range updates {
		h.store.SetSetting(r.Context(), &model.Setting{
			Key:   key,
			Value: value,
		})
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Ayarlar güncellendi"})
}

// AuditLogs denetim kayıtlarını listeler
func (h *AdminHandler) AuditLogs(w http.ResponseWriter, r *http.Request) {
	logs, err := h.store.ListAuditLogs(r.Context(), 100, 0)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Denetim kayıtları alınamadı"})
		return
	}

	if logs == nil {
		logs = []*model.AuditLog{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"logs":  logs,
		"total": len(logs),
	})
}
