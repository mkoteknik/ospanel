package handler

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/mkoteknik/ospanel/internal/adapter/database"
	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/pkg/crypto"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

var (
	dbNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,64}$`)
	dbUserRegex = regexp.MustCompile(`^[a-zA-Z0-9_]{3,32}$`)
)

// DatabaseHandler veritabanı yönetimi işlemleri
type DatabaseHandler struct {
	store  store.Store
	log    *logger.Logger
	mysql  *database.MySQLClient
}

// NewDatabaseHandler yeni DatabaseHandler oluşturur
func NewDatabaseHandler(s store.Store, log *logger.Logger) *DatabaseHandler {
	return &DatabaseHandler{store: s, log: log, mysql: database.NewMySQLClient()}
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

	if !dbNameRegex.MatchString(req.Name) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz veritabanı adı (3-64 karakter, harf/rakam/_)"})
		return
	}
	if !dbUserRegex.MatchString(req.Username) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz kullanıcı adı (3-32 karakter, harf/rakam/_)"})
		return
	}
	if len(req.Password) < 12 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Şifre en az 12 karakter olmalı"})
		return
	}

	encPass, err := crypto.Encrypt(req.Password, "")
	if err != nil {
		h.log.Errorw("veritabani sifresi sifrelenemedi", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Şifre şifrelenemedi - master key eksik"})
		return
	}

	db := &model.Database{
		UserID:      userID,
		Name:        req.Name,
		Username:    req.Username,
		PasswordEnc: encPass,
		Charset:     req.Charset,
		Collation:   "utf8mb4_unicode_ci",
		Status:      "active",
	}

	if err := h.store.CreateDatabase(r.Context(), db); err != nil {
		h.log.Errorw("veritabanı oluşturulamadı", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Veritabanı oluşturulamadı"})
		return
	}

	// Gerçek MySQL/MariaDB veritabanı ve kullanıcı oluştur
	mysqlCreated := false
	if h.mysql != nil && h.mysql.IsAvailable() {
		if err := h.mysql.CreateDatabase(req.Name, req.Username, req.Password); err != nil {
			h.log.Warnw("MySQL veritabanı oluşturulamadı, panel kaydı yapıldı", "db", req.Name, "error", err)
		} else {
			mysqlCreated = true
			h.log.Infow("MySQL veritabanı oluşturuldu", "db", req.Name, "user", req.Username)
		}
	}

	h.log.Infow("veritabanı oluşturuldu", "name", req.Name, "user_id", userID)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"database": db,
		"message":  "Veritabanı başarıyla oluşturuldu",
		"mysql_created": mysqlCreated,
	})
}

// Delete veritabanı siler
func (h *DatabaseHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	// Silmeden önce veritabanı bilgisini al
	db, err := h.store.GetDatabase(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Veritabanı bulunamadı"})
		return
	}

	// Gerçek MySQL veritabanını ve kullanıcısını sil
	if h.mysql != nil && h.mysql.IsAvailable() {
		if err := h.mysql.DeleteDatabase(db.Name, db.Username); err != nil {
			h.log.Warnw("MySQL veritabanı silinemedi", "db", db.Name, "error", err)
		} else {
			h.log.Infow("MySQL veritabanı silindi", "db", db.Name)
		}
	}

	if err := h.store.DeleteDatabase(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Veritabanı silinemedi"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Veritabanı başarıyla silindi"})
}
