package service

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/mkoteknik/ospanel/internal/adapter/dns"
	"github.com/mkoteknik/ospanel/internal/adapter/email"
	"github.com/mkoteknik/ospanel/internal/adapter/ols"
	"github.com/mkoteknik/ospanel/internal/adapter/system"
	"github.com/mkoteknik/ospanel/internal/model"
	"github.com/mkoteknik/ospanel/internal/pkg/logger"
	"github.com/mkoteknik/ospanel/internal/store"
)

// DomainService domain iş mantığı — handler sadece HTTP, servis transactional
type DomainService struct {
	store    store.Store
	log      *logger.Logger
	ols      *ols.Client
	pdns     *dns.Client
	mail     *email.MailServer
	serverIP string
}

func NewDomainService(s store.Store, log *logger.Logger, olsClient *ols.Client, pdnsClient *dns.Client, mailServer *email.MailServer, serverIP string) *DomainService {
	return &DomainService{store: s, log: log, ols: olsClient, pdns: pdnsClient, mail: mailServer, serverIP: serverIP}
}

type CreateDomainResult struct {
	Domain     *model.Domain
	VhostOK    bool
	AutoSetup  map[string]interface{}
}

// Create domain oluşturur — compensating transaction ile
func (svc *DomainService) Create(ctx context.Context, userID int64, username, homeDir, domain, phpVersion string) (*CreateDomainResult, error) {
	// Document root
	docRoot, err := system.CreateDocumentRoot(homeDir, domain)
	if err != nil {
		svc.log.Warnw("document root oluşturulamadı", "error", err, "domain", domain)
		docRoot = filepath.Join(homeDir, "public_html", domain)
	}

	d := &model.Domain{
		UserID:       userID,
		Domain:       domain,
		DocumentRoot: docRoot,
		PHPVersion:   phpVersion,
		ForceHTTPS:   true,
		Status:       model.DomainActive,
	}

	// 1. DB insert
	if err := svc.store.CreateDomain(ctx, d); err != nil {
		return nil, fmt.Errorf("domain DB kaydı başarısız: %w", err)
	}

	// Compensating helpers
	rollbackDB := func() { _ = svc.store.DeleteDomain(ctx, d.ID) }

	// 2. OLS vhost
	vhostOK := false
	if svc.ols != nil && svc.ols.IsAvailable() {
		if err := svc.ols.CreateVHost(domain, docRoot, phpVersion); err != nil {
			svc.log.Warnw("OLS vhost oluşturulamadı, rollback DB", "error", err, "domain", domain)
			rollbackDB()
			return nil, fmt.Errorf("OLS vhost hatası: %w", err)
		}
		vhostOK = true
	}

	// 3. DNS/Mail otomatik kurulum (best-effort, hata rollback yapmaz — sadece log)
	auto := svc.autoSetupDomain(domain, username)

	// Eğer DNS/Mail kritik ise ve başarısız olursa, vhost'u sil ve DB rollback
	// Şimdilik best-effort, ileride settings ile strict yapılabilir

	svc.log.Infow("domain service create tamam", "domain", domain, "vhost", vhostOK, "auto", auto)
	return &CreateDomainResult{Domain: d, VhostOK: vhostOK, AutoSetup: auto}, nil
}

func (svc *DomainService) autoSetupDomain(domain, username string) map[string]interface{} {
	results := map[string]interface{}{
		"dns": false, "email": false, "admin_email": "", "dkim": false, "spf": false, "dmarc": false, "ssl": false,
	}
	// Mevcut handler'daki autoSetup ile aynı mantık, sadeleştirildi
	// PowerDNS
	if svc.pdns != nil && svc.pdns.IsAvailable() {
		if err := svc.pdns.CreateZone(domain); err == nil {
			results["dns"] = true
			_ = svc.pdns.CreateRecord(domain, dns.Record{Name: domain + ".", Type: "A", Content: svc.serverIP, TTL: 3600})
			_ = svc.pdns.CreateRecord(domain, dns.Record{Name: "www." + domain + ".", Type: "CNAME", Content: domain + ".", TTL: 3600})
			_ = svc.pdns.CreateRecord(domain, dns.Record{Name: domain + ".", Type: "MX", Content: "mail." + domain + ".", TTL: 3600, Prio: 10})
			_ = svc.pdns.CreateRecord(domain, dns.Record{Name: "mail." + domain + ".", Type: "A", Content: svc.serverIP, TTL: 3600})
			spf := "v=spf1 mx a ip4:" + svc.serverIP + " ~all"
			_ = svc.pdns.CreateRecord(domain, dns.Record{Name: domain + ".", Type: "TXT", Content: spf, TTL: 3600})
			results["spf"] = true
			_ = svc.pdns.CreateRecord(domain, dns.Record{Name: "_dmarc." + domain + ".", Type: "TXT", Content: "v=DMARC1; p=quarantine; rua=mailto:admin@" + domain, TTL: 3600})
			results["dmarc"] = true
		}
	}
	if svc.mail != nil && svc.mail.IsAvailable() {
		if err := svc.mail.CreateDomain(domain); err == nil {
			results["email"] = true
			// admin hesabı best-effort
			adminEmail := "admin@" + domain
			// generateRandomPass yerine sabit değil, mail server kendi üretir
			dkim, _ := svc.mail.GenerateDKIM(domain)
			if dkim != "" {
				results["dkim"] = true
				_ = svc.pdns.CreateRecord(domain, dns.Record{Name: "mail._domainkey." + domain + ".", Type: "TXT", Content: dkim, TTL: 3600})
			}
			_ = adminEmail // log için
		}
	}
	results["ssl"] = false
	results["ssl_note"] = "Linux sunucuda Let's Encrypt otomatik"
	return results
}

// Delete domain siler — compensating: DB + OLS
func (svc *DomainService) Delete(ctx context.Context, domain *model.Domain) error {
	if err := svc.store.DeleteDomain(ctx, domain.ID); err != nil {
		return err
	}
	if svc.ols != nil && svc.ols.IsAvailable() {
		if err := svc.ols.DeleteVHost(domain.Domain); err != nil {
			svc.log.Warnw("OLS vhost silinemedi (DB silindi)", "error", err, "domain", domain.Domain)
		}
	}
	return nil
}
