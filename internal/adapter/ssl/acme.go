package ssl

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ACMEClient Let's Encrypt certbot istemcisi
type ACMEClient struct {
	installed bool
	certbot   string
}

// CertInfo SSL sertifika bilgisi
type CertInfo struct {
	Domain    string    `json:"domain"`
	ExpiresAt time.Time `json:"expires_at"`
	Issuer    string    `json:"issuer"`
	DaysLeft  int       `json:"days_left"`
	Active    bool      `json:"active"`
}

// NewACMEClient yeni ACME client
func NewACMEClient() *ACMEClient {
	c := &ACMEClient{}
	if path, err := exec.LookPath("certbot"); err == nil {
		c.certbot = path
		c.installed = true
	}
	return c
}

// IsAvailable certbot kullanılabilir mi?
func (c *ACMEClient) IsAvailable() bool {
	return c.installed
}

// IssueCertificate domain için SSL sertifikası oluşturur
func (c *ACMEClient) IssueCertificate(domain, docRoot, email string) error {
	if !c.installed {
		return fmt.Errorf("certbot kurulu değil")
	}

	// certbot certonly --webroot -w DOCROOT -d DOMAIN --non-interactive --agree-tos -m EMAIL
	args := []string{
		"certonly", "--webroot",
		"-w", docRoot,
		"-d", domain,
		"-d", "www." + domain,
		"--non-interactive",
		"--agree-tos",
		"-m", email,
		"--deploy-hook", "/usr/local/lsws/bin/lshttpd -r",
	}

	cmd := exec.Command(c.certbot, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("sertifika alınamadı: %s - %w", string(out), err)
	}

	// OLS'e SSL kur
	return c.installToOLS(domain)
}

// IssueWildcard wildcard SSL sertifikası (DNS challenge)
func (c *ACMEClient) IssueWildcard(domain, email, dnsProvider string) error {
	if !c.installed {
		return fmt.Errorf("certbot kurulu değil")
	}

	args := []string{
		"certonly",
		"--dns-cloudflare",
		"--dns-cloudflare-credentials", "/etc/ospanel/cf.ini",
		"-d", domain,
		"-d", "*." + domain,
		"--non-interactive",
		"--agree-tos",
		"-m", email,
		"--deploy-hook", "/usr/local/lsws/bin/lshttpd -r",
	}

	cmd := exec.Command(c.certbot, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wildcard sertifika alınamadı: %s - %w", string(out), err)
	}

	return c.installToOLS(domain)
}

// installToOLS sertifikayı OLS'e yükler
func (c *ACMEClient) installToOLS(domain string) error {
	certDir := fmt.Sprintf("/etc/letsencrypt/live/%s", domain)
	olsSSL := fmt.Sprintf("/usr/local/lsws/conf/cert/%s", domain)

	os.MkdirAll(olsSSL, 0755)

	// Sertifika dosyalarını OLS dizinine kopyala
	copyFile(certDir+"/fullchain.pem", olsSSL+"/fullchain.pem")
	copyFile(certDir+"/privkey.pem", olsSSL+"/privkey.pem")

	// OLS reload
	exec.Command("/usr/local/lsws/bin/lshttpd", "-r").Run()

	return nil
}

// CheckCertificate domain SSL durumunu kontrol eder
func (c *ACMEClient) CheckCertificate(domain string) (*CertInfo, error) {
	certDir := fmt.Sprintf("/etc/letsencrypt/live/%s", domain)
	certFile := certDir + "/fullchain.pem"

	if _, err := os.Stat(certFile); os.IsNotExist(err) {
		return &CertInfo{Domain: domain, Active: false}, nil
	}

	// openssl ile sertifika bilgisi al
	cmd := exec.Command("openssl", "x509", "-in", certFile, "-dates", "-issuer", "-noout")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	info := &CertInfo{Domain: domain, Active: true}
	output := string(out)

	// Son kullanma tarihi
	if idx := strings.Index(output, "notAfter="); idx != -1 {
		dateStr := strings.TrimSpace(output[idx+9:])
		if t, err := time.Parse("Jan 2 15:04:05 2006 MST", dateStr); err == nil {
			info.ExpiresAt = t
			info.DaysLeft = int(time.Until(t).Hours() / 24)
		}
	}

	// Issuer
	if idx := strings.Index(output, "issuer="); idx != -1 {
		info.Issuer = strings.TrimSpace(output[idx+7:])
		// Sadece ilk satırı al
		if nl := strings.Index(info.Issuer, "\n"); nl != -1 {
			info.Issuer = info.Issuer[:nl]
		}
	}

	return info, nil
}

// RenewCertificate sertifikayı yeniler
func (c *ACMEClient) RenewCertificate(domain string) error {
	if !c.installed {
		return fmt.Errorf("certbot kurulu değil")
	}
	cmd := exec.Command(c.certbot, "renew", "--cert-name", domain, "--deploy-hook", "/usr/local/lsws/bin/lshttpd -r")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("yenileme başarısız: %s - %w", string(out), err)
	}
	return nil
}

// ListCertificates tüm sertifikaları listeler
func (c *ACMEClient) ListCertificates() ([]CertInfo, error) {
	certDir := "/etc/letsencrypt/live"
	entries, err := os.ReadDir(certDir)
	if err != nil {
		return nil, nil
	}

	var certs []CertInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		info, err := c.CheckCertificate(entry.Name())
		if err == nil && info.Active {
			certs = append(certs, *info)
		}
	}
	return certs, nil
}

// SetupAutoRenew otomatik yenileme cron job'u oluşturur
func (c *ACMEClient) SetupAutoRenew() error {
	cronJob := "0 3 * * * root certbot renew --quiet --deploy-hook '/usr/local/lsws/bin/lshttpd -r'"
	return os.WriteFile("/etc/cron.d/ospanel-ssl-renew", []byte(cronJob+"\n"), 0644)
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0600)
}
