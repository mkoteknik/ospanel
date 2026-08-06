package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// DatabaseHandler veritabanı yönetimi işlemleri
type DatabaseHandler struct {
	store store.Store
	log   *logger.Logger
}

// NewDatabaseHandler yeni DatabaseHandler oluşturur
func NewDatabaseHandler(s store.Store, log *logger.Logger) *DatabaseHandler {
	return &DatabaseHandler{store: s, log: log}
}

// List kullanıcının veritabanlarını listeler
func (h *DatabaseHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	dbs, err := h.store.ListDatabases(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Veritabanları listelenemedi"})
		return
	}

	if dbs == nil {
		dbs = []*model.Database{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"databases": dbs,
		"total":     len(dbs),
	})
}

// Create yeni veritabanı oluşturur
func (h *DatabaseHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req model.CreateDatabaseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if req.Charset == "" {
		req.Charset = "utf8mb4"
	}

	db := &model.Database{
		UserID:      userID,
		Name:        req.Name,
		Username:    req.Username,
		PasswordEnc: req.Password, // Production'da şifrelenmeli
		Charset:     req.Charset,
		Collation:   "utf8mb4_unicode_ci",
		Status:      "active",
	}

	if err := h.store.CreateDatabase(r.Context(), db); err != nil {
		h.log.Errorw("veritabanı oluşturulamadı", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Veritabanı oluşturulamadı"})
		return
	}

	h.log.Infow("veritabanı oluşturuldu", "name", req.Name, "user_id", userID)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"database": db,
		"message":  "Veritabanı başarıyla oluşturuldu",
	})
}

// Delete veritabanı siler
func (h *DatabaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := h.store.DeleteDatabase(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Veritabanı silinemedi"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Veritabanı başarıyla silindi"})
}
