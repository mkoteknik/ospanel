package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/openspeed-panel/ospanel/internal/model"
	"github.com/openspeed-panel/ospanel/internal/pkg/logger"
	"github.com/openspeed-panel/ospanel/internal/store"
)

// AdminHandler admin panel işlemleri
type AdminHandler struct {
	store store.Store
	log   *logger.Logger
}

// NewAdminHandler yeni AdminHandler oluşturur
func NewAdminHandler(s store.Store, log *logger.Logger) *AdminHandler {
	return &AdminHandler{store: s, log: log}
}

// ListUsers tüm kullanıcıları listeler
func (h *AdminHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.store.ListUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Kullanıcılar listelenemedi"})
		return
	}

	if users == nil {
		users = []*model.User{}
	}

	// Hassas bilgileri temizle
	for _, u := range users {
		u.PasswordHash = ""
		u.TOTPSecret = ""
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"users": users,
		"total": len(users),
	})
}

// CreateUser yeni kullanıcı oluşturur
func (h *AdminHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if req.Role == "" {
		req.Role = "user"
	}

	hashedPassword, err := hashPassword(req.Password)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Şifre hashlenemedi"})
		return
	}

	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		Role:         model.UserRole(req.Role),
		HomeDir:      "/home/" + req.Username,
		Status:       model.StatusActive,
	}

	if err := h.store.CreateUser(r.Context(), user); err != nil {
		h.log.Errorw("kullanıcı oluşturulamadı", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Kullanıcı oluşturulamadı"})
		return
	}

	h.log.Infow("kullanıcı oluşturuldu", "username", req.Username)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"user": user,
		"message": "Kullanıcı başarıyla oluşturuldu",
	})
}

// UpdateUser kullanıcı günceller
func (h *AdminHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	user, err := h.store.GetUser(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Kullanıcı bulunamadı"})
		return
	}

	var updates map[string]interface{}
	json.NewDecoder(r.Body).Decode(&updates)

	if v, ok := updates["status"]; ok {
		user.Status = model.UserStatus(v.(string))
	}
	if v, ok := updates["role"]; ok {
		user.Role = model.UserRole(v.(string))
	}
	if v, ok := updates["quota_limit"]; ok {
		user.QuotaLimit = int64(v.(float64))
	}

	user.UpdatedAt = time.Now()
	h.store.UpdateUser(r.Context(), user)

	writeJSON(w, http.StatusOK, user)
}

// DeleteUser kullanıcı siler
func (h *AdminHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.store.DeleteUser(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Kullanıcı silinemedi"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Kullanıcı silindi"})
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
