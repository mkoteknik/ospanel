package handler

import (
	"encoding/json"
	"net/http"

	"github.com/mkoteknik/ospanel/internal/adapter/cloudflare"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
)

// CFHandler CloudFlare yönetimi
type CFHandler struct {
	cf  *cloudflare.Client
	log *logger.Logger
}

// NewCFHandler yeni CloudFlare handler
func NewCFHandler(log *logger.Logger) *CFHandler {
	return &CFHandler{cf: cloudflare.NewClient(), log: log}
}

// Status CF durumunu döndürür
func (h *CFHandler) Status(w http.ResponseWriter, r *http.Request) {
	stats := h.cf.GetStats()
	writeJSON(w, http.StatusOK, stats)
}

// Configure CF bilgilerini kaydeder
func (h *CFHandler) Configure(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email  string `json:"email"`
		APIKey string `json:"api_key"`
		ZoneID string `json:"zone_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if err := h.cf.Configure(req.Email, req.APIKey, req.ZoneID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Kaydedilemedi"})
		return
	}

	h.log.Infow("CloudFlare yapılandırıldı", "email", req.Email)
	writeJSON(w, http.StatusOK, map[string]string{"message": "CloudFlare yapılandırıldı"})
}

// ListDNS CF DNS kayıtlarını listeler
func (h *CFHandler) ListDNS(w http.ResponseWriter, r *http.Request) {
	records, err := h.cf.ListDNSRecords()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if records == nil { records = []cloudflare.CFDNSRecord{} }
	writeJSON(w, http.StatusOK, map[string]interface{}{"records": records, "total": len(records)})
}

// CreateDNS CF DNS kaydı oluşturur
func (h *CFHandler) CreateDNS(w http.ResponseWriter, r *http.Request) {
	var record cloudflare.CFDNSRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz kayıt"})
		return
	}
	r2, err := h.cf.CreateDNSRecord(record)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.log.Infow("CF DNS kaydı oluşturuldu", "name", record.Name, "type", record.Type)
	writeJSON(w, http.StatusCreated, r2)
}

// DeleteDNS CF DNS kaydı siler
func (h *CFHandler) DeleteDNS(w http.ResponseWriter, r *http.Request) {
	recordID := r.URL.Query().Get("id")
	if err := h.cf.DeleteDNSRecord(recordID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Silindi"})
}

// PurgeCache CF cache temizler
func (h *CFHandler) PurgeCache(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URLs []string `json:"urls,omitempty"`
	}
	json.NewDecoder(r.Body).Decode(&req)

	var err error
	if len(req.URLs) > 0 {
		err = h.cf.PurgeURL(req.URLs)
	} else {
		err = h.cf.PurgeCache()
	}

	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	h.log.Infow("CF cache purge")
	writeJSON(w, http.StatusOK, map[string]string{"message": "Cache temizlendi"})
}

// Analytics CF analitik verileri
func (h *CFHandler) Analytics(w http.ResponseWriter, r *http.Request) {
	data, err := h.cf.GetAnalytics()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, data)
}

// SSLMode CF SSL modunu değiştirir
func (h *CFHandler) SSLMode(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode string `json:"mode"` // off, flexible, full, strict
	}
	json.NewDecoder(r.Body).Decode(&req)
	if err := h.cf.SetSSLMode(req.Mode); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "SSL modu: " + req.Mode})
}
