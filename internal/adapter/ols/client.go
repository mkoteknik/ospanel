package ols

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

// Client OpenLiteSpeed yönetim istemcisi
type Client struct {
	adminURL  string
	vhostsDir string
	confDir   string
	binPath   string
}

// NewClient yeni OLS client oluşturur
func NewClient(adminURL, vhostsDir, confDir, binPath string) *Client {
	return &Client{
		adminURL:  adminURL,
		vhostsDir: vhostsDir,
		confDir:   confDir,
		binPath:   binPath,
	}
}

// IsAvailable OLS'in kurulu olup olmadığını kontrol eder
func (c *Client) IsAvailable() bool {
	if _, err := os.Stat(c.binPath); err != nil {
		// Windows'ta veya OLS kurulu değilse
		return false
	}
	return true
}

// CreateVHost yeni bir virtual host oluşturur
func (c *Client) CreateVHost(domain, documentRoot, phpVersion string) error {
	if !c.IsAvailable() {
		return fmt.Errorf("OLS kurulu değil, vhost oluşturulamadı (simülasyon)")
	}

	// 1. Vhost dizinini oluştur
	vhostDir := filepath.Join(c.vhostsDir, domain)
	if err := os.MkdirAll(vhostDir, 0755); err != nil {
		return fmt.Errorf("vhost dizini oluşturulamadı: %w", err)
	}

	// 2. Document root oluştur
	if err := os.MkdirAll(documentRoot, 0755); err != nil {
		return fmt.Errorf("document root oluşturulamadı: %w", err)
	}

	// 3. vhconf.xml oluştur
	vhconfPath := filepath.Join(vhostDir, "vhconf.xml")
	vhconf := generateVHostConfig(domain, documentRoot, phpVersion)
	if err := os.WriteFile(vhconfPath, []byte(vhconf), 0644); err != nil {
		return fmt.Errorf("vhconf.xml yazılamadı: %w", err)
	}

	// 4. Ana konfigürasyona vhost'u ekle
	if err := c.addVHostToMainConfig(domain); err != nil {
		return fmt.Errorf("ana konfigürasyona eklenemedi: %w", err)
	}

	// 5. Graceful restart
	if err := c.Reload(); err != nil {
		return fmt.Errorf("OLS reload başarısız: %w", err)
	}

	return nil
}

// DeleteVHost virtual host'u siler
func (c *Client) DeleteVHost(domain string) error {
	if !c.IsAvailable() {
		return fmt.Errorf("OLS kurulu değil")
	}

	// 1. Vhost dizinini sil
	vhostDir := filepath.Join(c.vhostsDir, domain)
	if err := os.RemoveAll(vhostDir); err != nil {
		return fmt.Errorf("vhost dizini silinemedi: %w", err)
	}

	// 2. Ana konfigürasyondan kaldır
	if err := c.removeVHostFromMainConfig(domain); err != nil {
		return fmt.Errorf("ana konfigürasyondan kaldırılamadı: %w", err)
	}

	// 3. Graceful restart
	if err := c.Reload(); err != nil {
		return fmt.Errorf("OLS reload başarısız: %w", err)
	}

	return nil
}

// SetPHPVersion domain için PHP sürümünü değiştirir
func (c *Client) SetPHPVersion(domain, phpVersion string) error {
	if !c.IsAvailable() {
		return nil
	}

	vhostDir := filepath.Join(c.vhostsDir, domain)
	vhconfPath := filepath.Join(vhostDir, "vhconf.xml")

	// Mevcut konfigürasyonu oku
	content, err := os.ReadFile(vhconfPath)
	if err != nil {
		return err
	}

	// PHP sürüm referansını değiştir
	old := `lsphp` + extractPHPVersion(string(content))
	new := `lsphp` + strings.ReplaceAll(phpVersion, ".", "")
	newContent := strings.ReplaceAll(string(content), old, new)

	if err := os.WriteFile(vhconfPath, []byte(newContent), 0644); err != nil {
		return err
	}

	return c.Reload()
}

// Reload OLS'i graceful restart yapar
func (c *Client) Reload() error {
	if !c.IsAvailable() {
		return nil
	}

	// Konfigürasyon testi
	testCmd := exec.Command(c.binPath, "-t")
	if out, err := testCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("konfigürasyon hatası: %s - %w", string(out), err)
	}

	// Graceful restart
	cmd := exec.Command(c.binPath, "-r")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("reload başarısız: %s - %w", string(out), err)
	}

	return nil
}

// GetStatus OLS durumunu döndürür
func (c *Client) GetStatus() map[string]interface{} {
	status := map[string]interface{}{
		"installed":   c.IsAvailable(),
		"bin_path":    c.binPath,
		"vhosts_dir":  c.vhostsDir,
		"conf_dir":    c.confDir,
		"admin_url":   c.adminURL,
	}

	if c.IsAvailable() {
		cmd := exec.Command(c.binPath, "-v")
		out, err := cmd.CombinedOutput()
		if err == nil {
			status["version"] = strings.TrimSpace(string(out))
		}
	}

	return status
}

// addVHostToMainConfig ana httpd_config.xml'e vhost ekler
func (c *Client) addVHostToMainConfig(domain string) error {
	configPath := filepath.Join(c.confDir, "httpd_config.xml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	vhostEntry := fmt.Sprintf(`<virtualHostConfig>
      <name>%s</name>
      <vhRoot>%s</vhRoot>
      <configFile>%s</configFile>
      <allowSymbolLink>1</allowSymbolLink>
      <enableScript>1</enableScript>
      <restrained>1</restrained>
    </virtualHostConfig>`, domain, c.vhostsDir+"/"+domain+"/", "$VH_ROOT/conf/vhconf.xml")

	// <virtualHostList> içine ekle
	if strings.Contains(string(content), "<name>"+domain+"</name>") {
		return nil // Zaten var
	}

	newContent := strings.Replace(string(content),
		"</virtualHostList>",
		"  "+vhostEntry+"\n    </virtualHostList>",
		1)

	return os.WriteFile(configPath, []byte(newContent), 0644)
}

// removeVHostFromMainConfig ana konfigürasyondan vhost'u kaldırır
func (c *Client) removeVHostFromMainConfig(domain string) error {
	configPath := filepath.Join(c.confDir, "httpd_config.xml")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	// <virtualHostConfig>...</virtualHostConfig> bloğunu kaldır
	// Basit string işlemi - production'da XML parser kullanılmalı
	lines := strings.Split(string(content), "\n")
	var newLines []string
	skip := false
	for _, line := range lines {
		if strings.Contains(line, "<name>"+domain+"</name>") {
			skip = true
			continue
		}
		if skip && strings.Contains(line, "</virtualHostConfig>") {
			skip = false
			continue
		}
		if !skip {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644)
}

// generateVHostConfig vhost XML konfigürasyonu üretir
func generateVHostConfig(domain, documentRoot, phpVersion string) string {
	tmpl := `<?xml version="1.0" encoding="UTF-8"?>
<virtualHostConfig>
  <docRoot>{{.DocRoot}}</docRoot>
  <enableGzip>1</enableGzip>
  <cgroups>0</cgroups>
  <enableBr>1</enableBr>

  <index>
    <useServer>1</useServer>
    <indexFiles>index.html,index.htm,index.php</indexFiles>
  </index>

  <accessControl>
    <allow>*</allow>
  </accessControl>

  <scriptHandlerList>
    <scriptHandler>
      <suffix>php</suffix>
      <type>lsapi</type>
      <handler>lsphp{{.PHPVer}}</handler>
    </scriptHandler>
  </scriptHandlerList>

  <rewrite>
    <enable>1</enable>
    <autoLoadHtaccess>1</autoLoadHtaccess>
  </rewrite>

  <accessLog>
    <fileName>$VH_ROOT/logs/access.log</fileName>
    <logHeaders>7</logHeaders>
    <rollingSize>10M</rollingSize>
    <keepDays>30</keepDays>
  </accessLog>
</virtualHostConfig>`

	data := map[string]string{
		"Domain": domain,
		"DocRoot":  documentRoot,
		"PHPVer": strings.ReplaceAll(phpVersion, ".", ""),
	}

	t := template.Must(template.New("vhost").Parse(tmpl))
	var buf strings.Builder
	t.Execute(&buf, data)
	return buf.String()
}

func extractPHPVersion(content string) string {
	// vhconf.xml'den PHP sürümünü çıkarır
	for _, ver := range []string{"84", "83", "82", "81", "80", "74"} {
		if strings.Contains(content, "lsphp"+ver) {
			return ver
		}
	}
	return "83" // varsayılan
}
