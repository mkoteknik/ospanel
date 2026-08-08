package database

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLClient MariaDB/MySQL yönetim istemcisi
type MySQLClient struct {
	db  *sql.DB
	dsn string
}

// NewMySQLClient yeni MySQL client oluşturur
func NewMySQLClient() *MySQLClient {
	// Bağlantı bilgilerini /etc/ospanel/mysql_root_pass veya db.conf dosyasından oku
	data, err := os.ReadFile("/etc/ospanel/db.conf")
	if err != nil {
		// Config yoksa baglanmadan bos client don (guvenli default)
		return &MySQLClient{}
	}

	config := parseConfig(string(data))
	user := config["DB_USER"]
	pass := config["DB_PASS"]
	if user == "" {
		user = "root"
	}

	dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/?charset=utf8mb4&parseTime=true", user, pass)
	return newMySQLWithDSN(dsn)
}

func newMySQLWithDSN(dsn string) *MySQLClient {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return &MySQLClient{dsn: dsn}
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	return &MySQLClient{db: db, dsn: dsn}
}

// IsAvailable MySQL kullanılabilir mi?
func (m *MySQLClient) IsAvailable() bool {
	if m.db == nil {
		return false
	}
	return m.db.Ping() == nil
}

// CreateDatabase yeni veritabanı oluşturur
func (m *MySQLClient) CreateDatabase(dbName, userName, password string) error {
	if !m.IsAvailable() {
		return fmt.Errorf("MySQL sunucusuna bağlanılamadı")
	}

	// Veritabanı oluştur
	_, err := m.db.Exec(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci", sanitizeIdent(dbName)))
	if err != nil {
		return fmt.Errorf("veritabanı oluşturulamadı: %w", err)
	}

	// Kullanıcı oluştur
	_, err = m.db.Exec(fmt.Sprintf("CREATE USER IF NOT EXISTS '%s'@'localhost' IDENTIFIED BY ?", sanitizeIdent(userName)), password)
	if err != nil {
		// Kullanıcı zaten varsa hata verme, devam et
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("kullanıcı oluşturulamadı: %w", err)
		}
	}

	// Yetkilendir
	_, err = m.db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON `%s`.* TO '%s'@'localhost'", sanitizeIdent(dbName), sanitizeIdent(userName)))
	if err != nil {
		return fmt.Errorf("yetkilendirme yapılamadı: %w", err)
	}

	_, _ = m.db.Exec("FLUSH PRIVILEGES")
	return nil
}

// DeleteDatabase veritabanı ve kullanıcıyı siler
func (m *MySQLClient) DeleteDatabase(dbName, userName string) error {
	if !m.IsAvailable() {
		return fmt.Errorf("MySQL sunucusuna bağlanılamadı")
	}

	_, err := m.db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", sanitizeIdent(dbName)))
	if err != nil {
		return fmt.Errorf("veritabanı silinemedi: %w", err)
	}

	_, err = m.db.Exec(fmt.Sprintf("DROP USER IF EXISTS '%s'@'localhost'", sanitizeIdent(userName)))
	if err != nil {
		// Kullanıcı silinemezse logla ama hata döndürme
		return nil
	}

	_, _ = m.db.Exec("FLUSH PRIVILEGES")
	return nil
}

// ListDatabases tüm veritabanlarını listeler
func (m *MySQLClient) ListDatabases() ([]string, error) {
	if !m.IsAvailable() {
		return nil, fmt.Errorf("MySQL sunucusuna bağlanılamadı")
	}

	rows, err := m.db.Query("SHOW DATABASES")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			continue
		}
		// Sistem veritabanlarını atla
		if name == "information_schema" || name == "mysql" || name == "performance_schema" || name == "sys" {
			continue
		}
		databases = append(databases, name)
	}
	return databases, nil
}

// GetDatabaseSize veritabanı boyutunu döndürür (bytes)
func (m *MySQLClient) GetDatabaseSize(dbName string) (int64, error) {
	if !m.IsAvailable() {
		return 0, fmt.Errorf("MySQL sunucusuna bağlanılamadı")
	}

	var size int64
	err := m.db.QueryRow(`
		SELECT COALESCE(SUM(data_length + index_length), 0)
		FROM information_schema.tables
		WHERE table_schema = ?`, dbName,
	).Scan(&size)
	return size, err
}

// sanitizeIdent SQL identifier'ı güvenli hale getirir (backtick'leri escape et)
func sanitizeIdent(s string) string {
	return strings.ReplaceAll(s, "`", "``")
}

// parseConfig basit key=value konfigürasyon parser'ı
func parseConfig(data string) map[string]string {
	config := map[string]string{}
	for _, line := range strings.Split(data, "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return config
}

// Close bağlantıyı kapatır
func (m *MySQLClient) Close() error {
	if m.db != nil {
		return m.db.Close()
	}
	return nil
}
