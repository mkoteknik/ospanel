package config

import (
	"crypto/rand"
	"encoding/hex"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config ana konfigürasyon yapısı
type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Auth     AuthConfig     `yaml:"auth"`
	Database DatabaseConfig `yaml:"database"`
	Log      LogConfig      `yaml:"log"`
	OLS      OLSConfig      `yaml:"ols"`
	DataDir  string         `yaml:"data_dir"`
	Security SecurityConfig `yaml:"security"`
}

// ServerConfig HTTP sunucu ayarları
type ServerConfig struct {
	Host           string   `yaml:"host"`
	Port           int      `yaml:"port"`
	TLS            TLSConfig `yaml:"tls"`
	AllowedOrigins []string `yaml:"allowed_origins"`
	TrustedProxies []string `yaml:"trusted_proxies"`
}

// TLSConfig TLS ayarları
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// SecurityConfig güvenlik ayarları
type SecurityConfig struct {
	StrictFileJail bool   `yaml:"strict_file_jail"`
	MasterKeyPath  string `yaml:"master_key_path"`
	JWTKeyPath     string `yaml:"jwt_key_path"`
}

// AuthConfig kimlik doğrulama ayarları
type AuthConfig struct {
	JWTSecret           string `yaml:"jwt_secret"`
	AccessTokenExpiry   int    `yaml:"access_token_expiry"`   // dakika
	RefreshTokenExpiry  int    `yaml:"refresh_token_expiry"`  // dakika
	MaxLoginAttempts    int    `yaml:"max_login_attempts"`
	LockoutDuration     int    `yaml:"lockout_duration"`      // dakika
}

// DatabaseConfig veritabanı ayarları
type DatabaseConfig struct {
	Path           string `yaml:"path"`
	MaxConnections int    `yaml:"max_connections"`
	WALMode        bool   `yaml:"wal_mode"`
}

// LogConfig loglama ayarları
type LogConfig struct {
	Level  string `yaml:"level"`  // debug, info, warn, error
	Output string `yaml:"output"` // stdout, file
	Path   string `yaml:"path"`   // dosya yolu (output=file ise)
}

// OLSConfig OpenLiteSpeed ayarları
type OLSConfig struct {
	AdminURL  string `yaml:"admin_url"`
	AdminUser string `yaml:"admin_user"`
	AdminPass string `yaml:"admin_pass"`
	VHostsDir string `yaml:"vhosts_dir"`
	ConfDir   string `yaml:"conf_dir"`
	BinPath   string `yaml:"bin_path"`
}

// Default varsayılan konfigürasyonu döndürür
func Default() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8090,
			TLS: TLSConfig{
				Enabled: false,
			},
			AllowedOrigins: []string{},
			TrustedProxies: []string{"127.0.0.1", "::1"},
		},
		Auth: AuthConfig{
			JWTSecret:          generateRandomSecret(),
			AccessTokenExpiry:  15,   // 15 dakika
			RefreshTokenExpiry: 10080, // 7 gün
			MaxLoginAttempts:   5,
			LockoutDuration:    30,
		},
		Database: DatabaseConfig{
			MaxConnections: 10,
			WALMode:        true,
		},
		Log: LogConfig{
			Level:  "info",
			Output: "stdout",
		},
		OLS: OLSConfig{
			AdminURL:  "http://localhost:7080",
			VHostsDir: "/usr/local/lsws/conf/vhosts",
			ConfDir:   "/usr/local/lsws/conf",
			BinPath:   "/usr/local/lsws/bin/lshttpd",
		},
		DataDir: "/var/lib/ospanel",
		Security: SecurityConfig{
			StrictFileJail: true,
			MasterKeyPath:  "/etc/ospanel/master.key",
			JWTKeyPath:     "/etc/ospanel/jwt.key",
		},
	}
}

// Load YAML dosyasından konfigürasyon yükler
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Config yoksa JWT/ master key persistence'ini hazırla
			ensureJWTSecret(cfg)
			return cfg, nil
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	// Dizin yollarını mutlak yap
	if !filepath.IsAbs(cfg.DataDir) {
		abs, err := filepath.Abs(cfg.DataDir)
		if err == nil {
			cfg.DataDir = abs
		}
	}

	// JWT secret persistence: boşsa veya default ise diskten yükle/kaydet
	ensureJWTSecret(cfg)

	// Security default'ları doldur
	if cfg.Security.MasterKeyPath == "" {
		cfg.Security.MasterKeyPath = "/etc/ospanel/master.key"
	}
	if cfg.Security.JWTKeyPath == "" {
		cfg.Security.JWTKeyPath = "/etc/ospanel/jwt.key"
	}

	return cfg, nil
}

// ensureJWTSecret JWT secret'ın restart'ta değişmemesi için diskte kalıcı tutar
func ensureJWTSecret(cfg *Config) {
	// Eğer config'te explicit secret varsa diske yazma, aynen kullan
	// Ancak boş veya 64 hex (32 byte) default gibiyse ve dosya varsa oradan oku
	keyPath := cfg.Security.JWTKeyPath
	if keyPath == "" {
		keyPath = "/etc/ospanel/jwt.key"
	}

	// Env override en yüksek öncelik
	if envKey := os.Getenv("OSPANEL_JWT_SECRET"); envKey != "" {
		cfg.Auth.JWTSecret = envKey
		return
	}

	// Dosyadan oku
	if data, err := os.ReadFile(keyPath); err == nil {
		trimmed := string(data)
		// Trim newline/space
		trimmed = trimSpace(trimmed)
		if len(trimmed) >= 32 {
			cfg.Auth.JWTSecret = trimmed
			return
		}
	}

	// Config'te valid secret varsa onu persist et
	if cfg.Auth.JWTSecret != "" && len(cfg.Auth.JWTSecret) >= 32 {
		// Persist et (best effort, permission 0600)
		_ = os.MkdirAll(filepath.Dir(keyPath), 0700)
		_ = os.WriteFile(keyPath, []byte(cfg.Auth.JWTSecret), 0600)
		return
	}

	// Hiçbiri yoksa yeni üret ve persist et
	newSecret := generateRandomSecret()
	cfg.Auth.JWTSecret = newSecret
	_ = os.MkdirAll(filepath.Dir(keyPath), 0700)
	_ = os.WriteFile(keyPath, []byte(newSecret), 0600)
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\r' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\r' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

// generateRandomSecret crypto/rand ile gerçek rastgele 32-byte secret üretir
func generateRandomSecret() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}
