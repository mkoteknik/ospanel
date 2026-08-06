package database

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	_ "github.com/lib/pq"
)

// PGClient PostgreSQL yönetim istemcisi
type PGClient struct {
	installed bool
	conn      *sql.DB
}

// PGDatabase PostgreSQL veritabanı
type PGDatabase struct {
	Name     string `json:"name"`
	Owner    string `json:"owner"`
	Encoding string `json:"encoding"`
	Size     string `json:"size"`
}

// NewPGClient yeni PostgreSQL client
func NewPGClient() *PGClient {
	// PostgreSQL kurulu mu?
	if _, err := exec.LookPath("psql"); err != nil {
		return &PGClient{installed: false}
	}

	// Bağlantı dene
	conn, err := sql.Open("postgres", "host=127.0.0.1 user=postgres password=postgres dbname=postgres sslmode=disable")
	if err != nil {
		return &PGClient{installed: true}
	}
	if err := conn.Ping(); err != nil {
		conn.Close()
		return &PGClient{installed: true}
	}

	return &PGClient{installed: true, conn: conn}
}

// IsAvailable PostgreSQL kullanılabilir mi?
func (p *PGClient) IsAvailable() bool {
	return p.installed && p.conn != nil
}

// ListDatabases veritabanlarını listeler
func (p *PGClient) ListDatabases() ([]PGDatabase, error) {
	if !p.IsAvailable() {
		return nil, fmt.Errorf("PostgreSQL kullanılabilir değil")
	}

	rows, err := p.conn.Query(`
		SELECT d.datname, pg_catalog.pg_get_userbyid(d.datdba) as owner,
			pg_catalog.pg_encoding_to_char(d.encoding) as encoding,
			pg_size_pretty(pg_database_size(d.datname)) as size
		FROM pg_catalog.pg_database d
		WHERE d.datname NOT IN ('template0', 'template1')
		ORDER BY d.datname`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []PGDatabase
	for rows.Next() {
		var db PGDatabase
		if err := rows.Scan(&db.Name, &db.Owner, &db.Encoding, &db.Size); err != nil {
			continue
		}
		dbs = append(dbs, db)
	}
	return dbs, rows.Err()
}

// CreateDatabase veritabanı oluşturur
func (p *PGClient) CreateDatabase(name, owner, password string) error {
	if !p.IsAvailable() {
		return fmt.Errorf("PostgreSQL kullanılabilir değil")
	}

	// Kullanıcı oluştur
	p.conn.Exec(fmt.Sprintf("CREATE USER \"%s\" WITH PASSWORD '%s'", owner, password))
	p.conn.Exec(fmt.Sprintf("ALTER ROLE \"%s\" WITH LOGIN", owner))

	// Veritabanı oluştur
	_, err := p.conn.Exec(fmt.Sprintf("CREATE DATABASE \"%s\" OWNER \"%s\" ENCODING 'UTF8'", name, owner))
	if err != nil {
		return err
	}

	// Yetkilendir
	p.conn.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON DATABASE \"%s\" TO \"%s\"", name, owner))

	return nil
}

// DeleteDatabase veritabanı siler
func (p *PGClient) DeleteDatabase(name string) error {
	if !p.IsAvailable() {
		return nil
	}
	_, err := p.conn.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS \"%s\"", name))
	return err
}

// GetStats PostgreSQL istatistikleri
func (p *PGClient) GetStats() map[string]interface{} {
	stats := map[string]interface{}{"installed": p.installed}

	if !p.IsAvailable() {
		return stats
	}

	var version string
	p.conn.QueryRow("SELECT version()").Scan(&version)
	stats["version"] = strings.Split(version, ",")[0]

	dbs, err := p.ListDatabases()
	stats["databases"] = len(dbs)
	stats["error"] = fmt.Sprint(err)

	return stats
}

// Close bağlantıyı kapatır
func (p *PGClient) Close() {
	if p.conn != nil {
		p.conn.Close()
	}
}

// InstallPostgreSQL PostgreSQL kurulum komutu
func InstallPostgreSQL() error {
	// OS detection
	data, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return fmt.Errorf("OS tespit edilemedi")
	}

	if strings.Contains(string(data), "ubuntu") || strings.Contains(string(data), "debian") {
		exec.Command("apt-get", "update").Run()
		cmd := exec.Command("apt-get", "install", "-y", "-qq", "postgresql", "postgresql-contrib")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("apt kurulum başarısız: %s - %w", string(out), err)
		}
	} else {
		cmd := exec.Command("dnf", "install", "-y", "postgresql-server", "postgresql-contrib")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("dnf kurulum başarısız: %s - %w", string(out), err)
		}
		// Init DB
		exec.Command("postgresql-setup", "--initdb").Run()
	}

	// Başlat
	exec.Command("systemctl", "enable", "postgresql").Run()
	exec.Command("systemctl", "start", "postgresql").Run()

	return nil
}
