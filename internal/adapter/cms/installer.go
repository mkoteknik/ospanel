package cms

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CMSInfo kurulum sonucu
type CMSInfo struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	AdminURL     string `json:"admin_url"`
	AdminUser    string `json:"admin_user"`
	AdminPass    string `json:"admin_pass"`
	Database     string `json:"database"`
	DBUser       string `json:"db_user"`
	DBPass       string `json:"db_pass"`
	InstalledAt  string `json:"installed_at"`
}

// InstallerConfig CMS kurulum konfigürasyonu
type InstallerConfig struct {
	Domain       string
	DocumentRoot string
	DBHost       string
	DBName       string
	DBUser       string
	DBPass       string
	SiteTitle    string
	AdminUser    string
	AdminPass    string
	AdminEmail   string
}

// InstallResult kurulum sonucu
type InstallResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	CMS     *CMSInfo `json:"cms,omitempty"`
	Error   string  `json:"error,omitempty"`
}

// WordPress kaynak URL
const wordpressURL = "https://wordpress.org/latest.zip"

// Joomla kaynak URL
const joomlaURL = "https://downloads.joomla.org/cms/joomla5/5-2-4/Joomla_5-2-4-Stable-Full_Package.zip?format=zip"

// GenRandomString rastgele string uretir (exported)
func GenRandomString(length int) string {
	b := make([]byte, length/2+1)
	rand.Read(b)
	return hex.EncodeToString(b)[:length]
}

// downloadFile URL'den dosya indirir, temp path döner
func downloadFile(url, destPath string) error {
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("indirme hatası: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("indirme başarısız: HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("dosya oluşturulamadı: %w", err)
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractZip zip dosyasını hedef dizine çıkarır (ZipSlip korumalı)
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("zip açılamadı: %w", err)
	}
	defer r.Close()

	// Önce ilk seviye dizini bul (WordPress wordpress/, Joomla Joomla_xxx/)
	var basePrefix string
	for _, f := range r.File {
		parts := strings.SplitN(f.Name, "/", 2)
		if len(parts) > 0 && parts[0] != "" {
			if basePrefix == "" {
				basePrefix = parts[0] + "/"
			}
		}
	}

	for _, f := range r.File {
		// basePrefix'i kaldır
		relPath := f.Name
		if basePrefix != "" && strings.HasPrefix(f.Name, basePrefix) {
			relPath = f.Name[len(basePrefix):]
		}
		if relPath == "" {
			continue
		}

		// ZipSlip koruması
		cleanName := filepath.Clean(relPath)
		if strings.Contains(cleanName, "..") {
			continue
		}

		targetPath := filepath.Join(destDir, cleanName)
		if !strings.HasPrefix(filepath.Clean(targetPath), filepath.Clean(destDir)+string(filepath.Separator)) && targetPath != destDir {
			continue
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(targetPath, 0755)
			continue
		}

		// Parent dizini oluştur
		os.MkdirAll(filepath.Dir(targetPath), 0755)

		// Dosyayı çıkar
		src, err := f.Open()
		if err != nil {
			continue
		}

		dst, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			src.Close()
			continue
		}

		io.Copy(dst, src)
		src.Close()
		dst.Close()
	}

	return nil
}

// chownR dizin altındaki tüm dosyaları recursive chown yapar (best-effort)
func chownR(path string, uid, gid int) {
	filepath.Walk(path, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		os.Chown(p, uid, gid)
		return nil
	})
}

// InstallWordPress WordPress kurulumu
func InstallWordPress(cfg InstallerConfig) (*InstallResult, error) {
	if cfg.SiteTitle == "" {
		cfg.SiteTitle = cfg.Domain
	}
	if cfg.AdminUser == "" {
		cfg.AdminUser = "admin"
	}
	if cfg.AdminPass == "" {
		cfg.AdminPass = GenRandomString(16)
	}
	if cfg.AdminEmail == "" {
		cfg.AdminEmail = "admin@" + cfg.Domain
	}

	docRoot := cfg.DocumentRoot
	tmpZip := filepath.Join(os.TempDir(), "wordpress-"+GenRandomString(8)+".zip")

	// 1. WordPress indir
	if err := downloadFile(wordpressURL, tmpZip); err != nil {
		return &InstallResult{Success: false, Error: err.Error()}, err
	}
	defer os.Remove(tmpZip)

	// 2. Zip'i document root'a çıkar
	if err := extractZip(tmpZip, docRoot); err != nil {
		return &InstallResult{Success: false, Error: err.Error()}, err
	}

	// 3. wp-config.php oluştur
	wpConfig := generateWPConfig(cfg)
	configPath := filepath.Join(docRoot, "wp-config.php")
	if err := os.WriteFile(configPath, []byte(wpConfig), 0644); err != nil {
		return &InstallResult{Success: false, Error: fmt.Sprintf("wp-config.php oluşturulamadı: %v", err)}, err
	}

	// 4. Dosya izinleri
	chownR(docRoot, -1, -1) // best-effort, root değilse sessizce geçer

	// 5. WordPress'i CLI olmadan kur (wp-admin/install.php endpoint'i yoksa)
	// Manuel kurulum: Kullanici ilk girişte WordPress kurulum ekranını görecek
	// Otomatik kurulum için wp-cli gerekir, o yoksa manuel adıma yönlendirelim

	result := &InstallResult{
		Success: true,
		Message: "WordPress dosyaları başarıyla yüklendi. Siteye giderek kurulumu tamamlayın.",
		CMS: &CMSInfo{
			Name:        "WordPress",
			Version:     "latest",
			AdminURL:    "https://" + cfg.Domain + "/wp-admin",
			AdminUser:   cfg.AdminUser,
			AdminPass:   cfg.AdminPass,
			Database:    cfg.DBName,
			DBUser:      cfg.DBUser,
			DBPass:      cfg.DBPass,
			InstalledAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	return result, nil
}

// InstallJoomla Joomla kurulumu
func InstallJoomla(cfg InstallerConfig) (*InstallResult, error) {
	if cfg.SiteTitle == "" {
		cfg.SiteTitle = cfg.Domain
	}
	if cfg.AdminUser == "" {
		cfg.AdminUser = "admin"
	}
	if cfg.AdminPass == "" {
		cfg.AdminPass = GenRandomString(16)
	}
	if cfg.AdminEmail == "" {
		cfg.AdminEmail = "admin@" + cfg.Domain
	}

	docRoot := cfg.DocumentRoot
	tmpZip := filepath.Join(os.TempDir(), "joomla-"+GenRandomString(8)+".zip")

	// 1. Joomla indir
	if err := downloadFile(joomlaURL, tmpZip); err != nil {
		return &InstallResult{Success: false, Error: err.Error()}, err
	}
	defer os.Remove(tmpZip)

	// 2. Zip'i document root'a çıkar
	if err := extractZip(tmpZip, docRoot); err != nil {
		return &InstallResult{Success: false, Error: err.Error()}, err
	}

	// 3. configuration.php için hazırlık
	// Joomla kurulumu web arayüzü üzerinden tamamlanır
	// DB bilgilerini kaydet ki kullanıcıya gösterelim

	result := &InstallResult{
		Success: true,
		Message: "Joomla dosyaları başarıyla yüklendi. Siteye giderek kurulumu tamamlayın.",
		CMS: &CMSInfo{
			Name:        "Joomla",
			Version:     "5.2.4",
			AdminURL:    "https://" + cfg.Domain + "/administrator",
			AdminUser:   cfg.AdminUser,
			AdminPass:   cfg.AdminPass,
			Database:    cfg.DBName,
			DBUser:      cfg.DBUser,
			DBPass:      cfg.DBPass,
			InstalledAt: time.Now().UTC().Format(time.RFC3339),
		},
	}

	return result, nil
}

// generateWPConfig wp-config.php içeriğini oluşturur
func generateWPConfig(cfg InstallerConfig) string {
	salts := generateWPSalts()

	return fmt.Sprintf(`<?php
/**
 * WordPress yapılandırma dosyası - OpenSpeed Panel tarafından oluşturuldu
 * Tarih: %s
 */

// ** Veritabanı ayarları **
define('DB_NAME', '%s');
define('DB_USER', '%s');
define('DB_PASSWORD', '%s');
define('DB_HOST', '%s');
define('DB_CHARSET', 'utf8mb4');
define('DB_COLLATE', 'utf8mb4_unicode_ci');

// ** Auth key'ler ve salt'lar (rastgele üretildi) **
%s

// ** Tablo öneki **
$table_prefix = 'wp_';

// ** Debug modu (production'da kapalı) **
define('WP_DEBUG', false);
define('WP_DEBUG_LOG', false);
define('WP_DEBUG_DISPLAY', false);

// ** Bellek limiti **
define('WP_MEMORY_LIMIT', '256M');
define('WP_MAX_MEMORY_LIMIT', '512M');

// ** Otomatik güncellemeler **
define('WP_AUTO_UPDATE_CORE', true);

// ** Post revizyonları (5 ile sınırla) **
define('WP_POST_REVISIONS', 5);

// ** Dosya düzenleme (güvenlik için kapalı) **
define('DISALLOW_FILE_EDIT', true);

// ** Absolute path **/
if (!defined('ABSPATH')) {
    define('ABSPATH', __DIR__ . '/');
}
require_once ABSPATH . 'wp-settings.php';
`, time.Now().Format("2006-01-02 15:04:05"),
		cfg.DBName, cfg.DBUser, cfg.DBPass, cfg.DBHost,
		salts)
}

// generateWPSalts WordPress salt'larını rastgele üretir
func generateWPSalts() string {
	saltKeys := []string{
		"AUTH_KEY", "SECURE_AUTH_KEY", "LOGGED_IN_KEY", "NONCE_KEY",
		"AUTH_SALT", "SECURE_AUTH_SALT", "LOGGED_IN_SALT", "NONCE_SALT",
	}
	var lines []string
	for _, key := range saltKeys {
		lines = append(lines, fmt.Sprintf("define('%s', '%s');", key, GenRandomString(64)))
	}
	return strings.Join(lines, "\n")
}
