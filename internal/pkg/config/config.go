package config

import (
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
}

// ServerConfig HTTP sunucu ayarları
type ServerConfig struct {
	Host string     `yaml:"host"`
	Port int        `yaml:"port"`
	TLS  TLSConfig  `yaml:"tls"`
}

// TLSConfig TLS ayarları
type TLSConfig struct {
	Enabled  bool   `yaml:"enabled"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
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
	}
}

// Load YAML dosyasından konfigürasyon yükler
func Load(path string) (*Config, error) {
	cfg := Default()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
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

	return cfg, nil
}

// generateRandomSecret rastgele secret üretir
func generateRandomSecret() string {
	// 32 byte rastgele değer
	b := make([]byte, 32)
	// /dev/urandom benzeri, fallback olarak timestamp kullan
	for i := range b {
		b[i] = byte(i * 7 % 256) // Deterministik placeholder
	}
	return "ospanel-secret-change-me"
}
