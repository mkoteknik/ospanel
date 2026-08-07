package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	ssl2 "github.com/mkoteknik/ospanel/internal/adapter/ssl"
	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// SSLHandler SSL sertifika yönetimi
type SSLHandler struct {
	store store.Store
	log   *logger.Logger
	acme  *ssl2.ACMEClient
}

// NewSSLHandler yeni SSLHandler
func NewSSLHandler(s store.Store, log *logger.Logger) *SSLHandler {
	return &SSLHandler{store: s, log: log, acme: ssl2.NewACMEClient()}
}

// List tüm SSL sertifikalarını listeler
func (h *SSLHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())

	// Kullanıcının domain'lerine ait SSL sertifikalarını getir
	domains, err := h.store.ListDomains(r.Context(), userID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Domain listesi alınamadı"})
		return
	}

	type SSLDomainInfo struct {
		Domain     string              `json:"domain"`
		DomainID   int64               `json:"domain_id"`
		SSLEnabled bool                `json:"ssl_enabled"`
		Cert       *ssl2.CertInfo      `json:"cert,omitempty"`
	}

	var list []SSLDomainInfo
	for _, d := range domains {
		info := SSLDomainInfo{
			Domain:     d.Domain,
			DomainID:   d.ID,
			SSLEnabled: d.SSLenabled,
		}

		// Sertifika durumunu kontrol et
		if h.acme != nil && h.acme.IsAvailable() {
			cert, err := h.acme.CheckCertificate(d.Domain)
			if err == nil && cert != nil && cert.Active {
				info.Cert = cert
				info.SSLEnabled = true
			}
		}

		list = append(list, info)
	}

	if list == nil {
		list = []SSLDomainInfo{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"ssl_domains": list,
		"total":       len(list),
	})
}

// Renew sertifika yeniler
func (h *SSLHandler) Renew(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	// Domain'i bul
	cert, err := h.store.GetSSLCert(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Sertifika bulunamadı"})
		return
	}

	if h.acme == nil || !h.acme.IsAvailable() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Certbot kullanılabilir değil"})
		return
	}

	if err := h.acme.RenewCertificate(cert.CommonName); err != nil {
		h.log.Errorw("SSL yenileme başarısız", "domain", cert.CommonName, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Sertifika yenilenemedi: " + err.Error()})
		return
	}

	certInfo, _ := h.acme.CheckCertificate(cert.CommonName)

	h.log.Infow("SSL yenilendi", "domain", cert.CommonName)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":   "Sertifika yenilendi",
		"cert_info": certInfo,
	})
}

// Delete SSL sertifikası siler
func (h *SSLHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	if err := h.store.DeleteSSLCert(r.Context(), id); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Sertifika silinemedi"})
		return
	}

	h.log.Infow("SSL sertifikası silindi", "id", id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "SSL sertifikası silindi"})
}

// Get sertifika detayı
func (h *SSLHandler) Get(w http.ResponseWriter, r *http.Request) {
	count := r.URL.Query().Get("count")
	if count == "true" {
		// Tüm sertifikaları say
		certs, err := h.acme.ListCertificates()
		if err != nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{"total": 0, "active": 0, "expiring_soon": 0})
			return
		}

		active := 0
		expiring := 0
		for _, c := range certs {
			if c.Active {
				active++
				if c.DaysLeft < 30 {
					expiring++
				}
			}
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"total":         len(certs),
			"active":        active,
			"expiring_soon": expiring,
		})
		return
	}

	// Tek sertifika detayı
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	cert, err := h.store.GetSSLCert(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Sertifika bulunamadı"})
		return
	}

	var certInfo *ssl2.CertInfo
	if h.acme != nil {
		certInfo, _ = h.acme.CheckCertificate(cert.CommonName)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"cert":      cert,
		"cert_info": certInfo,
	})
}

// SetupAutoRenew otomatik SSL yenileme cron'unu kurar
func (h *SSLHandler) SetupAutoRenew(w http.ResponseWriter, r *http.Request) {
	if h.acme == nil || !h.acme.IsAvailable() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "Certbot kullanılabilir değil"})
		return
	}

	if err := h.acme.SetupAutoRenew(); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Otomatik yenileme kurulamadı"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Otomatik SSL yenileme kuruldu (her gün 03:00)"})
}

// IssueWildcard wildcard SSL sertifikası oluşturur
func (h *SSLHandler) IssueWildcard(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DomainID int64  `json:"domain_id"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	domain, err := h.store.GetDomain(r.Context(), req.DomainID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}

	if req.Email == "" {
		req.Email = "admin@" + domain.Domain
	}

	if h.acme != nil && h.acme.IsAvailable() {
		if err := h.acme.IssueWildcard(domain.Domain, req.Email, "cloudflare"); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Wildcard SSL alınamadı: " + err.Error()})
			return
		}
	}

	h.log.Infow("wildcard SSL kuruldu", "domain", domain.Domain)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Wildcard SSL kuruldu: *." + domain.Domain})
}
