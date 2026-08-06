package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

// DB SQLite veritabanı bağlantısı
type DB struct {
	conn *sql.DB
	path string
}

// New yeni bir SQLite veritabanı bağlantısı oluşturur
func New(path string) (*DB, error) {
	// Dizin yoksa oluştur
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("veritabanı dizini oluşturulamadı: %w", err)
	}

	// WAL modu ve performans ayarları ile bağlan
	dsn := path + "?_journal_mode=WAL&_busy_timeout=5000&_foreign_keys=on&_synchronous=NORMAL&_cache_size=-32000"

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("veritabanı açılamadı: %w", err)
	}

	// Bağlantı havuzu ayarları
	conn.SetMaxOpenConns(1) // SQLite için tek yazıcı
	conn.SetMaxIdleConns(1)

	// Bağlantıyı test et
	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("veritabanı bağlantısı test edilemedi: %w", err)
	}

	return &DB{conn: conn, path: path}, nil
}

// Conn ham *sql.DB bağlantısını döndürür (migration için)
func (db *DB) Conn() *sql.DB {
	return db.conn
}

// Close veritabanı bağlantısını kapatır
func (db *DB) Close() error {
	return db.conn.Close()
}
