package ssl

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

// DNSChallenge DNS doğrulama bilgisi
type DNSChallenge struct {
	Domain  string `json:"domain"`
	Record  string `json:"record"`  // _acme-challenge.example.com
	Type    string `json:"type"`    // TXT
	Value   string `json:"value"`   // doğrulama değeri
	Status  string `json:"status"`  // pending, verified, failed
	Message string `json:"message"` // kullanıcıya gösterilecek mesaj
}

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

// IssueCertificate domain için SSL sertifikası oluşturur (HTTP challenge)
func (c *ACMEClient) IssueCertificate(domain, docRoot, email string) error {
	if !c.installed {
		return fmt.Errorf("certbot kurulu değil")
	}

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

	return c.installToOLS(domain)
}

// IssueWildcard wildcard SSL - Cloudflare API ile (mevcut)
func (c *ACMEClient) IssueWildcard(domain, email, dnsProvider string) error {
	if !c.installed {
		return fmt.Errorf("certbot kurulu değil")
	}

	var args []string

	switch dnsProvider {
	case "powerdns":
		// PowerDNS API üzerinden otomatik DNS challenge
		return c.issueWildcardManual(domain, email, "")
	case "manual":
		// Manuel DNS - kullanıcı TXT kaydını eliyle ekler
		return fmt.Errorf("manuel DNS modu için StartDNSChallenge kullanın")
	default:
		// Cloudflare
		args = []string{
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
	}

	cmd := exec.Command(c.certbot, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("wildcard sertifika alınamadı: %s - %w", string(out), err)
	}

	return c.installToOLS(domain)
}

// StartDNSChallenge manuel DNS wildcard SSL sürecini başlatır
// TXT kaydını döndürür, kullanıcı bunu DNS'ine ekler
func (c *ACMEClient) StartDNSChallenge(domain, email string) (*DNSChallenge, error) {
	if !c.installed {
		return nil, fmt.Errorf("certbot kurulu değil")
	}

	// Benzersiz challenge ID'si
	challengeID := genRandomHex(8)
	challengeDir := "/tmp/ospanel-dns-challenge"
	os.MkdirAll(challengeDir, 0755)

	// Manuel auth hook script'i oluştur
	authHook := fmt.Sprintf(`#!/bin/bash
# OpenSpeed Panel - DNS Challenge Auth Hook
echo "CHALLENGE_START"
echo "DOMAIN=$CERTBOT_DOMAIN"
echo "VALIDATION=$CERTBOT_VALIDATION"
echo "TOKEN=$CERTBOT_TOKEN"
echo "CHALLENGE_END"

# Challenge bilgisini dosyaya yaz
cat > %s/%s.json << EOF
{
  "domain": "$CERTBOT_DOMAIN",
  "validation": "$CERTBOT_VALIDATION",
  "token": "$CERTBOT_TOKEN",
  "record": "_acme-challenge.$CERTBOT_DOMAIN",
  "status": "pending"
}
EOF

# Panel'in onaylamasını bekle (max 5 dakika)
for i in $(seq 1 60); do
  if grep -q '"verified"' %s/%s.json 2>/dev/null; then
    # TXT kaydının yayılması için 30 saniye bekle
    sleep 30
    exit 0
  fi
  sleep 5
done
# Timeout - panel onaylamadı
exit 1
`, challengeDir, challengeID, challengeDir, challengeID)

	cleanupHook := fmt.Sprintf(`#!/bin/bash
# OpenSpeed Panel - DNS Challenge Cleanup Hook
rm -f %s/%s.json
`, challengeDir, challengeID)

	authHookPath := fmt.Sprintf("%s/auth-%s.sh", challengeDir, challengeID)
	cleanupHookPath := fmt.Sprintf("%s/cleanup-%s.sh", challengeDir, challengeID)

	if err := os.WriteFile(authHookPath, []byte(authHook), 0755); err != nil {
		return nil, fmt.Errorf("auth hook oluşturulamadı: %w", err)
	}
	if err := os.WriteFile(cleanupHookPath, []byte(cleanupHook), 0755); err != nil {
		return nil, fmt.Errorf("cleanup hook oluşturulamadı: %w", err)
	}

	// certbot'u arka planda başlat
	args := []string{
		"certonly",
		"--manual",
		"--preferred-challenges", "dns",
		"-d", domain,
		"-d", "*." + domain,
		"--non-interactive",
		"--agree-tos",
		"-m", email,
		"--manual-auth-hook", authHookPath,
		"--manual-cleanup-hook", cleanupHookPath,
		"--deploy-hook", "/usr/local/lsws/bin/lshttpd -r",
	}

	cmd := exec.Command(c.certbot, args...)

	// stdout'tan challenge bilgisini yakala
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("certbot başlatılamadı: %w", err)
	}

	// Challenge dosyasının oluşmasını bekle
	challengeFile := fmt.Sprintf("%s/%s.json", challengeDir, challengeID)
	var challenge *DNSChallenge

	for i := 0; i < 30; i++ {
		if data, err := os.ReadFile(challengeFile); err == nil {
			// Challenge dosyasını parse et
			lines := strings.Split(string(data), "\n")
			challenge = &DNSChallenge{
				Domain:  domain,
				Type:    "TXT",
				Record:  "_acme-challenge." + domain,
				Status:  "pending",
				Message: "Bu TXT kaydını DNS yöneticinize ekleyin, ardından Doğrula'ya tıklayın.",
			}
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, `"validation": "`) {
					challenge.Value = strings.Trim(line[15:], `",`)
				}
			}
			break
		}
		time.Sleep(2 * time.Second)
	}

	// stdout'u arka planda oku
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				break
			}
			// Challenge bilgisini yakala
			output := string(buf[:n])
			if strings.Contains(output, "CERTBOT_DOMAIN") {
				// zaten dosyadan okuduk
			}
		}
	}()

	if challenge == nil {
		cmd.Process.Kill()
		return nil, fmt.Errorf("DNS challenge başlatılamadı - certbot çıktı okunamadı")
	}

	// Challenge ID'sini challenge'a göm
	challenge.Message = fmt.Sprintf("DNS yöneticinize şu TXT kaydını ekleyin:\n\nAd: %s\nTür: TXT\nDeğer: %s\n\nEkledikten sonra Doğrula butonuna tıklayın. (ID: %s)",
		challenge.Record, challenge.Value, challengeID)

	return challenge, nil
}

// CompleteDNSChallenge manuel DNS challenge'ı tamamlar
func (c *ACMEClient) CompleteDNSChallenge(challengeID string) error {
	challengeFile := fmt.Sprintf("/tmp/ospanel-dns-challenge/%s.json", challengeID)
	data, err := os.ReadFile(challengeFile)
	if err != nil {
		return fmt.Errorf("challenge bulunamadı - süre dolmuş olabilir, tekrar deneyin")
	}

	// Challenge'ı verified olarak işaretle - auth hook bunu bekliyor
	content := strings.Replace(string(data), `"pending"`, `"verified"`, 1)
	if err := os.WriteFile(challengeFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("challenge güncellenemedi: %w", err)
	}

	return nil
}

// GetDNSChallengeStatus challenge durumunu kontrol eder
func (c *ACMEClient) GetDNSChallengeStatus(challengeID string) (*DNSChallenge, error) {
	challengeFile := fmt.Sprintf("/tmp/ospanel-dns-challenge/%s.json", challengeID)
	data, err := os.ReadFile(challengeFile)
	if err != nil {
		return nil, fmt.Errorf("challenge bulunamadı")
	}

	challenge := &DNSChallenge{Status: "pending"}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, `"domain":`) {
			challenge.Domain = strings.Trim(line[10:], `",`)
		}
		if strings.Contains(line, `"validation":`) {
			challenge.Value = strings.Trim(line[15:], `",`)
		}
		if strings.Contains(line, `"status":`) {
			challenge.Status = strings.Trim(line[11:], `",`)
		}
		if strings.Contains(line, `"verified"`) {
			challenge.Status = "verified"
		}
	}

	return challenge, nil
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

	cmd := exec.Command("openssl", "x509", "-in", certFile, "-dates", "-issuer", "-noout")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	info := &CertInfo{Domain: domain, Active: true}
	output := string(out)

	if idx := strings.Index(output, "notAfter="); idx != -1 {
		dateStr := strings.TrimSpace(output[idx+9:])
		if t, err := time.Parse("Jan 2 15:04:05 2006 MST", dateStr); err == nil {
			info.ExpiresAt = t
			info.DaysLeft = int(time.Until(t).Hours() / 24)
		}
	}

	if idx := strings.Index(output, "issuer="); idx != -1 {
		info.Issuer = strings.TrimSpace(output[idx+7:])
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

func genRandomHex(n int) string {
	b := make([]byte, n/2+1)
	rand.Read(b)
	return hex.EncodeToString(b)[:n]
}

// issueWildcardManual internal: PowerDNS otomatik veya manuel
func (c *ACMEClient) issueWildcardManual(domain, email, powerDNSAPI string) error {
	if powerDNSAPI != "" {
		// PowerDNS API ile otomatik
		return c.issueWithDNS(domain, email)
	}
	// Manuel mod - StartDNSChallenge + CompleteDNSChallenge akışı
	return fmt.Errorf("manuel DNS için StartDNSChallenge/CompleteDNSChallenge kullanın")
}

func (c *ACMEClient) issueWithDNS(domain, email string) error {
	args := []string{
		"certonly",
		"--manual",
		"--preferred-challenges", "dns",
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
		return fmt.Errorf("sertifika alınamadı: %s - %w", string(out), err)
	}
	return c.installToOLS(domain)
}
