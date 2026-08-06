package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mkoteknik/ospanel/internal/adapter/system"
	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// DomainHandler domain yönetimi
type DomainHandler struct {
	store store.Store
	log   *logger.Logger
}

// NewDomainHandler yeni DomainHandler
func NewDomainHandler(s store.Store, log *logger.Logger) *DomainHandler {
	return &DomainHandler{store: s, log: log}
}

// List kullanıcının domainlerini listeler
func (h *DomainHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	domains, err := h.store.ListDomains(r.Context(), userID)
	if err != nil {
		h.log.Errorw("domain listesi alınamadı", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Domainler listelenemedi"})
		return
	}
	if domains == nil {
		domains = []*model.Domain{}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domains": domains,
		"total":   len(domains),
	})
}

// Create yeni domain oluşturur
func (h *DomainHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	var req model.CreateDomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek formatı"})
		return
	}

	if req.Domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Domain adı gerekli"})
		return
	}

	// Basit domain validasyonu
	if !isValidDomain(req.Domain) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz domain formatı (örn: site.com)"})
		return
	}

	if req.PHPVersion == "" {
		req.PHPVersion = "8.3"
	}

	// Domain zaten var mı?
	existing, _ := h.store.GetDomainByName(r.Context(), req.Domain)
	if existing != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Bu domain zaten kayıtlı"})
		return
	}

	user, err := h.store.GetUser(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Kullanıcı bilgisi alınamadı"})
		return
	}

	// Document root oluştur
	homeDir := user.HomeDir
	if homeDir == "" {
		homeDir = system.GetHomeDir(user.Username)
	}

	docRoot, err := system.CreateDocumentRoot(homeDir, req.Domain)
	if err != nil {
		h.log.Warnw("document root oluşturulamadı, devam ediliyor", "error", err, "path", docRoot)
	}

	domain := &model.Domain{
		UserID:       userID,
		Domain:       req.Domain,
		DocumentRoot: docRoot,
		PHPVersion:   req.PHPVersion,
		ForceHTTPS:   true,
		Status:       model.DomainActive,
	}

	if err := h.store.CreateDomain(r.Context(), domain); err != nil {
		h.log.Errorw("domain oluşturulamadı", "error", err, "domain", req.Domain)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Domain oluşturulamadı"})
		return
	}

	h.log.Infow("domain oluşturuldu", "domain", req.Domain, "user_id", userID, "docroot", docRoot)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"domain":  domain,
		"message": "Domain başarıyla oluşturuldu",
	})
}

// Get domain detayı
func (h *DomainHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz domain ID"})
		return
	}

	domain, err := h.store.GetDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}

	// Ek bilgiler
	docRootExists := system.PathExists(domain.DocumentRoot)
	var diskUsage int64
	if docRootExists {
		diskUsage, _ = system.DiskUsage(domain.DocumentRoot)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":          domain,
		"docroot_exists":  docRootExists,
		"disk_usage_mb":   diskUsage / (1024 * 1024),
		"site_url":        "http://" + domain.Domain,
	})
}

// Update domain günceller
func (h *DomainHandler) Update(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	domain, err := h.store.GetDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if v, ok := updates["php_version"]; ok {
		if phpv, ok := v.(string); ok && isValidPHPVersion(phpv) {
			domain.PHPVersion = phpv
		}
	}
	if v, ok := updates["force_https"]; ok {
		if b, ok := v.(bool); ok {
			domain.ForceHTTPS = b
		}
	}
	if v, ok := updates["status"]; ok {
		if s, ok := v.(string); ok {
			domain.Status = model.DomainStatus(s)
		}
	}
	if v, ok := updates["bandwidth_limit"]; ok {
		domain.BandwidthLimit = int64(v.(float64))
	}

	domain.UpdatedAt = time.Now()
	if err := h.store.UpdateDomain(r.Context(), domain); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Domain güncellenemedi"})
		return
	}

	writeJSON(w, http.StatusOK, domain)
}

// Delete domain siler
func (h *DomainHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	domain, err := h.store.GetDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}

	if err := h.store.DeleteDomain(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Domain silinemedi"})
		return
	}

	h.log.Infow("domain silindi", "domain", domain.Domain, "domain_id", id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Domain başarıyla silindi"})
}

// InstallSSL domain'e SSL kurar
func (h *DomainHandler) InstallSSL(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	domain, err := h.store.GetDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}

	var req struct {
		Type string `json:"type"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Type == "" {
		req.Type = "lets_encrypt"
	}

	h.log.Infow("ssl kurulumu başlatıldı", "domain", domain.Domain, "type", req.Type)

	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "SSL kurulumu başlatıldı - " + req.Type,
	})
}

// isValidDomain basit domain validasyonu
func isValidDomain(domain string) bool {
	if len(domain) < 4 || len(domain) > 253 {
		return false
	}
	// Nokta içermeli ve en az bir noktadan sonra 2+ karakter olmalı
	lastDot := -1
	for i := len(domain) - 1; i >= 0; i-- {
		if domain[i] == '.' {
			lastDot = i
			break
		}
	}
	return lastDot > 0 && lastDot < len(domain)-2
}

// isValidPHPVersion geçerli PHP sürümü mü?
func isValidPHPVersion(v string) bool {
	valid := map[string]bool{
		"7.4": true, "8.0": true, "8.1": true, "8.2": true, "8.3": true, "8.4": true,
	}
	return valid[v]
}
