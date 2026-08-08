package handler

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mkoteknik/ospanel/internal/adapter/cms"
	"github.com/mkoteknik/ospanel/internal/adapter/database"
	"github.com/mkoteknik/ospanel/internal/adapter/dns"
	"github.com/mkoteknik/ospanel/internal/adapter/email"
	"github.com/mkoteknik/ospanel/internal/adapter/ols"
	ssl2 "github.com/mkoteknik/ospanel/internal/adapter/ssl"
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
	mysql     *database.MySQLClient
	serverIP  string
}

// NewDomainHandler yeni DomainHandler
func NewDomainHandler(s store.Store, log *logger.Logger, olsClient *ols.Client, pdnsClient *dns.Client, mailServer *email.MailServer, serverIP string) *DomainHandler {
	return &DomainHandler{store: s, log: log, ols: olsClient, pdns: pdnsClient, mail: mailServer, mysql: database.NewMySQLClient(), serverIP: serverIP}
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

	// Kota kontrolu: max_domains
	domainCount, _ := h.store.CountDomainsByUser(r.Context(), userID)
	if user.MaxDomains > 0 && domainCount >= user.MaxDomains {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"error": fmt.Sprintf("Domain limitine ulastiniz (max %d). Mevcut: %d", user.MaxDomains, domainCount),
		})
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

	// IDOR koruması: sadece sahibi veya admin görebilir
	if callerID, ok := middleware.GetUserID(r.Context()); ok {
		if domain.UserID != callerID {
			if role, ok := middleware.GetUserRole(r.Context()); !ok || role != "admin" {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu domain için yetkiniz yok"})
				return
			}
		}
	}

	// Ek bilgiler
	docRootExists := system.PathExists(domain.DocumentRoot)
	var diskUsage int64
	if docRootExists {
		diskUsage, _ = system.DiskUsage(domain.DocumentRoot)
	}

	// Kullanici kota bilgisi
	user, _ := h.store.GetUser(r.Context(), domain.UserID)
	var quotaMB int64
	var homeUsage int64
	if user != nil && user.HomeDir != "" {
		quotaMB = user.QuotaLimit
		homeUsage, _ = system.DiskUsage(user.HomeDir)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"domain":          domain,
		"docroot_exists":  docRootExists,
		"disk_usage_mb":   diskUsage / (1024 * 1024),
		"home_usage_mb":   homeUsage / (1024 * 1024),
		"quota_limit_mb":  quotaMB,
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

	// IDOR koruması
	if callerID, ok := middleware.GetUserID(r.Context()); ok {
		if domain.UserID != callerID {
			if role, ok := middleware.GetUserRole(r.Context()); !ok || role != "admin" {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu domain için yetkiniz yok"})
				return
			}
		}
	}

	var updates map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if v, ok := updates["php_version"]; ok {
		if phpv, ok := v.(string); ok && isValidPHPVersion(phpv) {
			if domain.PHPVersion != phpv {
				oldVersion := domain.PHPVersion
				domain.PHPVersion = phpv
				// OLS'te de PHP sürümünü değiştir
				if h.ols != nil && h.ols.IsAvailable() {
					if err := h.ols.SetPHPVersion(domain.Domain, phpv); err != nil {
						h.log.Warnw("OLS PHP sürümü değiştirilemedi", "domain", domain.Domain, "old", oldVersion, "new", phpv, "error", err)
					} else {
						h.log.Infow("OLS PHP sürümü değiştirildi", "domain", domain.Domain, "version", phpv)
					}
				}
			}
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

	// IDOR koruması
	if callerID, ok := middleware.GetUserID(r.Context()); ok {
		if domain.UserID != callerID {
			if role, ok := middleware.GetUserRole(r.Context()); !ok || role != "admin" {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu domain için yetkiniz yok"})
				return
			}
		}
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
		Type  string `json:"type"`
		Email string `json:"email"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Type == "" {
		req.Type = "lets_encrypt"
	}
	if req.Email == "" {
		req.Email = "admin@" + domain.Domain
	}

	// Let's Encrypt ile sertifika al
	acme := ssl2.NewACMEClient()
	if acme.IsAvailable() {
		if err := acme.IssueCertificate(domain.Domain, domain.DocumentRoot, req.Email); err != nil {
			h.log.Errorw("ssl kurulumu başarısız", "domain", domain.Domain, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "SSL kurulamadı: " + err.Error()})
			return
		}

		// Domain SSL durumunu güncelle
		domain.SSLenabled = true
		h.store.UpdateDomain(r.Context(), domain)

		// OLS reload (certbot hook yapar ama yine de)
		if h.ols != nil {
			h.ols.Reload()
		}

		certInfo, _ := acme.CheckCertificate(domain.Domain)

		h.log.Infow("ssl kuruldu", "domain", domain.Domain, "issuer", "Let's Encrypt")
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"message":    "SSL başarıyla kuruldu!",
			"domain":     domain.Domain,
			"cert_info":  certInfo,
			"site_url":   "https://" + domain.Domain,
		})
		return
	}

	h.log.Warnw("certbot kurulu değil, SSL kurulamadı", "domain", domain.Domain)
	writeJSON(w, http.StatusAccepted, map[string]string{
		"message": "SSL kurulumu Linux sunucuda aktif olacak (certbot gerekli)",
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

// CreateSubdomain alt domain oluşturur
func (h *DomainHandler) CreateSubdomain(w http.ResponseWriter, r *http.Request) {
	userID, _ := middleware.GetUserID(r.Context())
	parentID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	var req model.CreateSubdomainRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	// URL'den gelen parent_id'yi kullan
	req.ParentID = parentID

	// Parent domain'i bul
	parent, err := h.store.GetDomain(r.Context(), req.ParentID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Ana domain bulunamadı: " + err.Error()})
		return
	}

	subDomain := req.Subdomain + "." + parent.Domain
	docRoot := parent.DocumentRoot + "_sub/" + req.Subdomain

	// Alt domain zaten var mı?
	existing, _ := h.store.GetDomainByName(r.Context(), subDomain)
	if existing != nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Bu subdomain zaten kayıtlı"})
		return
	}

	// Document root oluştur
	system.CreateDocumentRoot(docRoot, req.Subdomain)

	if req.PHPVersion == "" { req.PHPVersion = parent.PHPVersion }

	domain := &model.Domain{
		UserID:       userID,
		ParentID:     &req.ParentID,
		Domain:       subDomain,
		DocumentRoot: docRoot,
		PHPVersion:   req.PHPVersion,
		ForceHTTPS:   true,
		Status:       model.DomainActive,
	}

	if err := h.store.CreateDomain(r.Context(), domain); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Subdomain oluşturulamadı"})
		return
	}

	// OLS vhost
	if h.ols != nil && h.ols.IsAvailable() {
		h.ols.CreateVHost(subDomain, docRoot, req.PHPVersion)
	}

	// Progress tracking
	steps := []map[string]interface{}{}

	// Adım 1: Document root
	steps = append(steps, map[string]interface{}{"step": 1, "name": "Dosya dizini oluşturuldu", "status": "done", "detail": docRoot})

	// Adım 2: OLS vhost
	vhostStatus := "skipped"
	if h.ols != nil && h.ols.IsAvailable() {
		if err := h.ols.CreateVHost(subDomain, docRoot, req.PHPVersion); err != nil {
			vhostStatus = "failed"
			steps = append(steps, map[string]interface{}{"step": 2, "name": "OLS Sanal Host", "status": "failed", "detail": err.Error()})
		} else {
			vhostStatus = "done"
			steps = append(steps, map[string]interface{}{"step": 2, "name": "OLS Sanal Host oluşturuldu", "status": "done", "detail": "PHP " + req.PHPVersion})
		}
	} else {
		steps = append(steps, map[string]interface{}{"step": 2, "name": "OLS Sanal Host", "status": "skipped", "detail": "OLS kullanılabilir değil"})
	}

	// Adım 3: DNS kaydı
	dnsStatus := "skipped"
	if h.pdns != nil && h.pdns.IsAvailable() {
		if err := h.pdns.CreateRecord(parent.Domain, dns.Record{
			Name: req.Subdomain + "." + parent.Domain + ".", Type: "CNAME",
			Content: parent.Domain + ".", TTL: 3600,
		}); err != nil {
			dnsStatus = "failed"
			steps = append(steps, map[string]interface{}{"step": 3, "name": "DNS kaydı", "status": "failed", "detail": err.Error()})
		} else {
			dnsStatus = "done"
			steps = append(steps, map[string]interface{}{"step": 3, "name": "DNS kaydı eklendi", "status": "done", "detail": req.Subdomain + "." + parent.Domain + " → CNAME → " + parent.Domain})
		}
	} else {
		steps = append(steps, map[string]interface{}{"step": 3, "name": "DNS kaydı", "status": "skipped", "detail": "PowerDNS kullanılabilir değil"})
	}

	// Adım 4: SSL (ana domain wildcard varsa otomatik kapsar)
	sslStatus := "skipped"
	sslNote := ""
	if parent.SSLenabled {
		sslStatus = "done"
		sslNote = "Ana domain SSL'i tüm alt domainleri kapsar (wildcard)"
	} else {
		sslNote = "SSL henüz kurulmadı - Domain sayfasından kurabilirsiniz"
	}
	steps = append(steps, map[string]interface{}{"step": 4, "name": "SSL Sertifikası", "status": sslStatus, "detail": sslNote})

	// Adım 5: Tamamlandı
	steps = append(steps, map[string]interface{}{"step": 5, "name": "Alt domain hazır", "status": "done", "detail": "http://" + subDomain})

	h.log.Infow("subdomain oluşturuldu", "subdomain", subDomain, "parent", parent.Domain,
		"vhost", vhostStatus, "dns", dnsStatus, "ssl", sslStatus)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"domain":  domain,
		"message": "Alt domain başarıyla oluşturuldu",
		"steps":   steps,
		"url":     "http://" + subDomain,
	})
}

// ListSubdomains domain'in alt domainlerini listeler
func (h *DomainHandler) ListSubdomains(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	subs, err := h.store.ListSubdomains(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Subdomainler listelenemedi"})
		return
	}
	if subs == nil { subs = []*model.Domain{} }
	writeJSON(w, http.StatusOK, map[string]interface{}{"subdomains": subs, "total": len(subs)})
}

// ListEmails domain email hesaplarini listeler
func (h *DomainHandler) ListEmails(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Domain gerekli"})
		return
	}
	accounts, _ := h.mail.ListAccounts(domain)
	if accounts == nil { accounts = []email.EmailAccount{} }
	writeJSON(w, http.StatusOK, map[string]interface{}{"emails": accounts, "total": len(accounts)})
}

// CreateEmailAccount email hesabi olusturur
func (h *DomainHandler) CreateEmailAccount(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Quota    int    `json:"quota"`
		Domain   string `json:"domain"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}
	if h.mail != nil && h.mail.IsAvailable() {
		if err := h.mail.CreateAccount(req.Domain, req.Email, req.Password, req.Quota); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	h.log.Infow("email hesabi olusturuldu", "email", req.Email)
	writeJSON(w, http.StatusCreated, map[string]string{"message": "Email oluşturuldu: " + req.Email})
}

// DeleteEmailAccount email hesabi siler
func (h *DomainHandler) DeleteEmailAccount(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	email := r.URL.Query().Get("email")
	if email == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Email adresi gerekli"})
		return
	}

	// Mail sunucusundan sil
	if h.mail != nil && h.mail.IsAvailable() {
		if err := h.mail.DeleteAccount(email); err != nil {
			h.log.Errorw("email silinemedi", "email", email, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Email silinemedi: " + err.Error()})
			return
		}
	}

	// SQLite store'dan da sil
	if err := h.store.DeleteEmail(r.Context(), id); err != nil {
		h.log.Warnw("email store'dan silinemedi", "id", id, "error", err)
	}

	h.log.Infow("email silindi", "email", email, "id", id)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Email silindi: " + email})
}

// UpdateEmail email hesabi gunceller (quota, forward, autoresponder)
func (h *DomainHandler) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var req struct {
		Quota            int    `json:"quota"`
		ForwardTo        string `json:"forward_to"`
		AutoresponderMsg string `json:"autoresponder_msg"`
		Password         string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	email, err := h.store.GetEmail(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Email bulunamadi"})
		return
	}
	if req.Quota > 0 {
		email.Quota = int64(req.Quota)
	}
	if req.ForwardTo != "" {
		email.ForwardTo = req.ForwardTo
	}
	if req.AutoresponderMsg != "" {
		email.AutoresponderMsg = req.AutoresponderMsg
	}
	if err := h.store.UpdateEmail(r.Context(), email); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Email guncellenemedi"})
		return
	}
	if req.Password != "" && h.mail != nil && h.mail.IsAvailable() {
		parts := strings.Split(email.Email, "@")
		if len(parts) == 2 {
			h.mail.DeleteAccount(email.Email)
			h.mail.CreateAccount(parts[1], email.Email, req.Password, int(email.Quota))
		}
	}
	h.log.Infow("email guncellendi", "email", email.Email, "id", id)
	writeJSON(w, http.StatusOK, map[string]interface{}{"email": email, "message": "Email guncellendi"})
}

// ListAliases domain email alias/forwarder listesi
func (h *DomainHandler) ListAliases(w http.ResponseWriter, r *http.Request) {
	domain := r.URL.Query().Get("domain")
	if domain == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Domain gerekli"})
		return
	}
	if h.mail != nil && h.mail.IsAvailable() {
		aliases, err := h.mail.ListAliases(domain)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"aliases": aliases, "total": len(aliases)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"aliases": []interface{}{}, "total": 0})
}

// CreateAlias email alias/forwarder olusturur
func (h *DomainHandler) CreateAlias(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Domain      string `json:"domain"`
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Gecersiz istek"})
		return
	}
	if h.mail != nil && h.mail.IsAvailable() {
		if err := h.mail.CreateAlias(req.Domain, req.Source, req.Destination); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	h.log.Infow("email alias olusturuldu", "domain", req.Domain, "source", req.Source)
	writeJSON(w, http.StatusCreated, map[string]string{"message": "Alias olusturuldu: " + req.Source})
}

// DeleteAlias email alias siler
func (h *DomainHandler) DeleteAlias(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	email := r.URL.Query().Get("email")
	if h.mail != nil && h.mail.IsAvailable() && email != "" {
		if err := h.mail.DeleteAlias(email); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
	}
	h.log.Infow("email alias silindi", "id", id, "email", email)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Alias silindi"})
}

// isValidDomain RFC1035 uyumlu domain validasyonu
func isValidDomain(domain string) bool {
	if len(domain) < 4 || len(domain) > 253 {
		return false
	}
	// Toplam uzunluk ve label kontrolleri
	if strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") || strings.HasSuffix(domain, "-") {
		return false
	}
	if strings.Contains(domain, "..") || strings.Contains(domain, "--") {
		return false
	}
	// En az bir nokta ve TLD >=2
	lastDot := strings.LastIndex(domain, ".")
	if lastDot <= 0 || lastDot >= len(domain)-2 {
		return false
	}
	tld := domain[lastDot+1:]
	if len(tld) < 2 {
		return false
	}
	// TLD sadece harf
	for _, c := range tld {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')) {
			return false
		}
	}
	// Her label 1-63, alfanum + hyphen, hyphen başta/sonda olamaz
	labels := strings.Split(domain, ".")
	for _, lbl := range labels {
		if len(lbl) == 0 || len(lbl) > 63 {
			return false
		}
		if lbl[0] == '-' || lbl[len(lbl)-1] == '-' {
			return false
		}
		for _, c := range lbl {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-') {
				return false
			}
		}
	}
	// Reserved
	reserved := map[string]bool{"localhost": true, "test": true, "invalid": true, "example": true}
	if reserved[strings.ToLower(domain)] {
		return false
	}
	return true
}

// isValidPHPVersion geçerli PHP sürümü mü?
func isValidPHPVersion(v string) bool {
	valid := map[string]bool{"8.2": true, "8.3": true, "8.4": true}
	return valid[v]
}

// InstallCMS domain'e CMS kurar (WordPress, Joomla)
func (h *DomainHandler) InstallCMS(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	domain, err := h.store.GetDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}

	// Sahiplik kontrolü
	callerID, _ := middleware.GetUserID(r.Context())
	callerRole, _ := middleware.GetUserRole(r.Context())
	if domain.UserID != callerID && callerRole != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu domain için yetkiniz yok"})
		return
	}

	var req struct {
		CMS        string `json:"cms"`         // "wordpress" veya "joomla"
		SiteTitle  string `json:"site_title"`
		AdminUser  string `json:"admin_user"`
		AdminPass  string `json:"admin_pass"`
		AdminEmail string `json:"admin_email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz istek"})
		return
	}

	if req.CMS != "wordpress" && req.CMS != "joomla" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Desteklenen CMS: wordpress, joomla"})
		return
	}

	// Document root kontrolü
	docRoot := domain.DocumentRoot
	if _, err := os.Stat(docRoot); os.IsNotExist(err) {
		if err := os.MkdirAll(docRoot, 0755); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Document root oluşturulamadı"})
			return
		}
	}

	// Dizin boş mu kontrol et
	entries, _ := os.ReadDir(docRoot)
	if len(entries) > 0 {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "Dizin boş değil. CMS kurulumu için boş bir dizin gerekli."})
		return
	}

	// Veritabanı oluştur
	dbName := "cms_" + strings.ReplaceAll(strings.ReplaceAll(domain.Domain, ".", "_"), "-", "_")
	dbName = dbName[:min(len(dbName), 64)]
	dbUser := "usr_" + cms.GenRandomString(10)
	dbPass := cms.GenRandomString(20)

	if h.mysql == nil || !h.mysql.IsAvailable() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "MySQL sunucusuna bağlanılamadı. CMS kurulumu için veritabanı gerekli."})
		return
	}

	if err := h.mysql.CreateDatabase(dbName, dbUser, dbPass); err != nil {
		h.log.Errorw("CMS veritabanı oluşturulamadı", "error", err, "domain", domain.Domain)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Veritabanı oluşturulamadı: " + err.Error()})
		return
	}

	// CMS kurulumu
	cfg := cms.InstallerConfig{
		Domain:       domain.Domain,
		DocumentRoot: docRoot,
		DBHost:       "127.0.0.1",
		DBName:       dbName,
		DBUser:       dbUser,
		DBPass:       dbPass,
		SiteTitle:    req.SiteTitle,
		AdminUser:    req.AdminUser,
		AdminPass:    req.AdminPass,
		AdminEmail:   req.AdminEmail,
	}

	var result *cms.InstallResult
	switch req.CMS {
	case "wordpress":
		result, err = cms.InstallWordPress(cfg)
	case "joomla":
		result, err = cms.InstallJoomla(cfg)
	}

	if err != nil || !result.Success {
		h.log.Errorw("CMS kurulumu başarısız", "error", err, "cms", req.CMS, "domain", domain.Domain)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "CMS kurulumu başarısız: " + result.Error})
		return
	}

	// Veritabanı kaydını store'a ekle
	dbRecord := &model.Database{
		UserID:      domain.UserID,
		Name:        dbName,
		Username:    dbUser,
		PasswordEnc: dbPass,
		Charset:     "utf8mb4",
		Status:      "active",
	}
	h.store.CreateDatabase(r.Context(), dbRecord)

	h.log.Infow("CMS kuruldu", "cms", req.CMS, "domain", domain.Domain, "admin_url", result.CMS.AdminURL)
	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "CMS başarıyla kuruldu! Siteye giderek kurulumu tamamlayın.",
		"cms":     result.CMS,
		"next_step": map[string]string{
			"site_url":  "http://" + domain.Domain,
			"admin_url": result.CMS.AdminURL,
			"note":      "Siteye giderek kurulumu tamamlayın. Veritabanı bilgileri otomatik girilecektir.",
		},
	})
}

// SecureSite document root'a .htaccess guvenlik dosyasi ekler/yeniler
func (h *DomainHandler) SecureSite(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	domain, err := h.store.GetDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}
	callerID, _ := middleware.GetUserID(r.Context())
	callerRole, _ := middleware.GetUserRole(r.Context())
	if domain.UserID != callerID && callerRole != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu domain için yetkiniz yok"})
		return
	}

	htaccessPath := filepath.Join(domain.DocumentRoot, ".htaccess")
	if err := os.WriteFile(htaccessPath, []byte(system.GenerateHtaccess()), 0644); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": ".htaccess yazılamadı"})
		return
	}
	h.log.Infow(".htaccess yenilendi", "domain", domain.Domain)
	writeJSON(w, http.StatusOK, map[string]string{"message": ".htaccess güvenlik dosyası oluşturuldu"})
}

// UploadCustomSSL manuel SSL sertifikasi yukleme
func (h *DomainHandler) UploadCustomSSL(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	domain, err := h.store.GetDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}
	callerID, _ := middleware.GetUserID(r.Context())
	callerRole, _ := middleware.GetUserRole(r.Context())
	if domain.UserID != callerID && callerRole != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu domain için yetkiniz yok"})
		return
	}

	var req struct {
		Certificate string `json:"certificate"` // PEM format
		PrivateKey  string `json:"private_key"`  // PEM format
		Chain       string `json:"chain"`        // opsiyonel intermediate
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Certificate == "" || req.PrivateKey == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Sertifika ve private key gerekli"})
		return
	}

	// OLS cert dizini
	certDir := "/usr/local/lsws/conf/cert/" + domain.Domain
	os.MkdirAll(certDir, 0755)

	fullchain := req.Certificate
	if req.Chain != "" {
		fullchain += "\n" + req.Chain
	}

	if err := os.WriteFile(certDir+"/fullchain.pem", []byte(fullchain), 0600); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Sertifika kaydedilemedi"})
		return
	}
	if err := os.WriteFile(certDir+"/privkey.pem", []byte(req.PrivateKey), 0600); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Private key kaydedilemedi"})
		return
	}

	// OLS reload
	exec.Command("/usr/local/lsws/bin/lshttpd", "-r").Run()

	// SSL durumunu guncelle
	domain.SSLenabled = true
	h.store.UpdateDomain(r.Context(), domain)

	h.log.Infow("ozel SSL yuklendi", "domain", domain.Domain)
	writeJSON(w, http.StatusOK, map[string]string{"message": "SSL sertifikası başarıyla yüklendi"})
}

// GetPHPExtensions domain PHP surumu icin yuklu extensions listesini dondurur
func (h *DomainHandler) GetPHPExtensions(w http.ResponseWriter, r *http.Request) {
	version := r.URL.Query().Get("version")
	if version == "" {
		version = "8.3"
	}
	if h.ols != nil {
		exts := h.ols.GetPHPExtensions(version)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"extensions": exts,
			"version":    version,
			"available":  h.ols.GetAvailablePHPVersions(),
		})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"extensions": []string{},
		"version":    version,
		"available":  []string{"8.3"},
	})
}

// ListAliases domain alias/parked domainleri listeler
func (h *DomainHandler) ListDomainAliases(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	domain, err := h.store.GetDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}
	// Sahiplik kontrolü
	callerID, _ := middleware.GetUserID(r.Context())
	callerRole, _ := middleware.GetUserRole(r.Context())
	if domain.UserID != callerID && callerRole != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu domain için yetkiniz yok"})
		return
	}

	aliases, err := h.store.ListAliasesByDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Aliaslar listelenemedi"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"aliases": aliases, "total": len(aliases)})
}

// CreateAlias domain'e alias/parked domain ekler
func (h *DomainHandler) CreateDomainAlias(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	domain, err := h.store.GetDomain(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}
	callerID, _ := middleware.GetUserID(r.Context())
	callerRole, _ := middleware.GetUserRole(r.Context())
	if domain.UserID != callerID && callerRole != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu domain için yetkiniz yok"})
		return
	}

	var req struct {
		Alias  string `json:"alias"`
		Type   string `json:"type"` // "park" veya "redirect"
		Target string `json:"target,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Alias == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Geçersiz alias adı"})
		return
	}
	if req.Type == "" {
		req.Type = "park"
	}
	if req.Type != "park" && req.Type != "redirect" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Tip 'park' veya 'redirect' olmalı"})
		return
	}
	if req.Type == "redirect" && req.Target == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Redirect için hedef URL gerekli"})
		return
	}

	// Aynı alias zaten var mı?
	existing, _ := h.store.ListAliasesByDomain(r.Context(), id)
	for _, a := range existing {
		if a.Alias == req.Alias {
			writeJSON(w, http.StatusConflict, map[string]string{"error": "Bu alias zaten ekli"})
			return
		}
	}

	alias := &model.Alias{
		DomainID: id,
		Alias:    req.Alias,
		Type:     req.Type,
		Target:   req.Target,
	}
	if err := h.store.CreateAlias(r.Context(), alias); err != nil {
		h.log.Errorw("alias oluşturulamadı", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Alias oluşturulamadı"})
		return
	}

	// OLS vhost'a alias ekle
	if h.ols != nil && h.ols.IsAvailable() {
		h.ols.CreateVHost(alias.Alias, domain.DocumentRoot, domain.PHPVersion)
	}

	h.log.Infow("alias eklendi", "domain", domain.Domain, "alias", req.Alias)
	writeJSON(w, http.StatusCreated, map[string]interface{}{"alias": alias, "message": "Alias başarıyla eklendi"})
}

// DeleteAlias alias siler
func (h *DomainHandler) DeleteDomainAlias(w http.ResponseWriter, r *http.Request) {
	aliasID, _ := strconv.ParseInt(chi.URLParam(r, "aliasId"), 10, 64)
	domainID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)

	domain, err := h.store.GetDomain(r.Context(), domainID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Domain bulunamadı"})
		return
	}
	callerID, _ := middleware.GetUserID(r.Context())
	callerRole, _ := middleware.GetUserRole(r.Context())
	if domain.UserID != callerID && callerRole != "admin" {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Bu domain için yetkiniz yok"})
		return
	}

	if err := h.store.DeleteAlias(r.Context(), aliasID); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Alias silinemedi"})
		return
	}
	h.log.Infow("alias silindi", "alias_id", aliasID)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Alias silindi"})
}
