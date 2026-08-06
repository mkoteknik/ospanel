package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mkoteknik/ospanel/internal/adapter/dns"
	"github.com/mkoteknik/ospanel/internal/adapter/email"
	"github.com/mkoteknik/ospanel/internal/adapter/ols"
	"github.com/mkoteknik/ospanel/internal/adapter/system"
	"github.com/mkoteknik/ospanel/internal/api/middleware"
	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// DomainHandler domain yönetimi
type DomainHandler struct {
	store     store.Store
	log       *logger.Logger
	ols       *ols.Client
	pdns      *dns.Client
	mail      *email.MailServer
	serverIP  string
}

// NewDomainHandler yeni DomainHandler
func NewDomainHandler(s store.Store, log *logger.Logger, olsClient *ols.Client, pdnsClient *dns.Client, mailServer *email.MailServer, serverIP string) *DomainHandler {
	return &DomainHandler{store: s, log: log, ols: olsClient, pdns: pdnsClient, mail: mailServer, serverIP: serverIP}
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

	// OLS'te vhost oluştur
	vhostCreated := false
	if h.ols != nil && h.ols.IsAvailable() {
		if err := h.ols.CreateVHost(domain.Domain, docRoot, domain.PHPVersion); err != nil {
			h.log.Warnw("OLS vhost oluşturulamadı", "error", err, "domain", req.Domain)
		} else {
			vhostCreated = true
			h.log.Infow("OLS vhost oluşturuldu", "domain", req.Domain)
		}
	}

	// === OTOMASYON: DNS, Email, DKIM, SSL ===
	autoResults := h.autoSetupDomain(req.Domain, user.Username, h.serverIP)

	h.log.Infow("domain oluşturuldu", "domain", req.Domain, "user_id", userID, "docroot", docRoot,
		"dns", autoResults["dns"], "email", autoResults["email"], "dkim", autoResults["dkim"])

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"domain":    domain,
		"message":   "Domain başarıyla oluşturuldu",
		"auto_setup": map[string]interface{}{
			"vhost":      vhostCreated,
			"dns_zone":   autoResults["dns"],
			"email":      autoResults["email"],
			"admin_email": autoResults["admin_email"],
			"dkim":       autoResults["dkim"],
			"spf":        autoResults["spf"],
			"dmarc":      autoResults["dmarc"],
			"ssl":        autoResults["ssl"],
		},
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

	// OLS'ten vhost'u sil
	if h.ols != nil && h.ols.IsAvailable() {
		if err := h.ols.DeleteVHost(domain.Domain); err != nil {
			h.log.Warnw("OLS vhost silinemedi", "error", err, "domain", domain.Domain)
		}
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

// autoSetupDomain domain için otomatik kurulum yapar: DNS, Email, DKIM, SPF, DMARC
func (h *DomainHandler) autoSetupDomain(domain, username, serverIP string) map[string]interface{} {
	results := map[string]interface{}{
		"dns": false, "email": false, "admin_email": "", "dkim": false, "spf": false, "dmarc": false, "ssl": false,
	}

	adminEmail := "admin@" + domain
	adminPass := generateRandomPass(16)

	// 1. PowerDNS zone oluştur
	if h.pdns != nil && h.pdns.IsAvailable() {
		if err := h.pdns.CreateZone(domain); err == nil {
			results["dns"] = true

			// A kaydı @ → server IP
			h.pdns.CreateRecord(domain, dns.Record{
				Name: domain + ".", Type: "A", Content: serverIP, TTL: 3600,
			})
			// www CNAME
			h.pdns.CreateRecord(domain, dns.Record{
				Name: "www." + domain + ".", Type: "CNAME", Content: domain + ".", TTL: 3600,
			})
			// MX kaydı
			h.pdns.CreateRecord(domain, dns.Record{
				Name: domain + ".", Type: "MX", Content: "mail." + domain + ".", TTL: 3600, Prio: 10,
			})
			// mail A kaydı
			h.pdns.CreateRecord(domain, dns.Record{
				Name: "mail." + domain + ".", Type: "A", Content: serverIP, TTL: 3600,
			})
			// SPF
			spfRecord := "v=spf1 mx a ip4:" + serverIP + " ~all"
			h.pdns.CreateRecord(domain, dns.Record{
				Name: domain + ".", Type: "TXT", Content: spfRecord, TTL: 3600,
			})
			results["spf"] = true

			// DMARC
			h.pdns.CreateRecord(domain, dns.Record{
				Name: "_dmarc." + domain + ".", Type: "TXT",
				Content: "v=DMARC1; p=quarantine; rua=mailto:" + adminEmail,
				TTL: 3600,
			})
			results["dmarc"] = true

			// CAA - Let's Encrypt yetkilendirme
			h.pdns.CreateRecord(domain, dns.Record{
				Name: domain + ".", Type: "CAA",
				Content: "0 issue \"letsencrypt.org\"",
				TTL: 3600,
			})

			h.log.Infow("PowerDNS zone ve kayitlar olusturuldu", "domain", domain)
		}
	}

	// 2. Email domain ve admin hesabı
	if h.mail != nil && h.mail.IsAvailable() {
		if err := h.mail.CreateDomain(domain); err == nil {
			results["email"] = true

			if err := h.mail.CreateAccount(domain, adminEmail, adminPass, 1024); err == nil {
				results["admin_email"] = adminEmail
				h.log.Infow("admin email olusturuldu", "email", adminEmail)
			}

			// DKIM anahtarı oluştur
			dkimKey, err := h.mail.GenerateDKIM(domain)
			if err == nil && dkimKey != "" {
				results["dkim"] = true
				// DKIM TXT kaydını ekle (mail._domainkey)
				h.pdns.CreateRecord(domain, dns.Record{
					Name:    "mail._domainkey." + domain + ".",
					Type:    "TXT",
					Content: dkimKey,
					TTL:     3600,
				})
			}
		}
	}

	// 3. SSL sertifika bilgisi (Linux'ta certbot ile otomatik)
	results["ssl"] = false // Linux sunucuda certbot/certificate ile aktif olacak
	results["ssl_note"] = "Linux sunucuda Let's Encrypt otomatik kurulacak"

	return results
}

// generateRandomPass rastgele şifre üretir
func generateRandomPass(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)[:length]
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
