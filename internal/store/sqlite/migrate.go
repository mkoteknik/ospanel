package sqlite

import (
	"context"
	"fmt"
)

// Migrate veritabanı migration'larını çalıştırır
func (db *DB) Migrate(ctx context.Context) error {
	migrations := []struct {
		version int
		query   string
	}{
		{1, createUsersTable},
		{2, createDomainsTable},
		{3, createEmailsTable},
		{4, createDatabasesTable},
		{5, createSSLCertsTable},
		{6, createDNSRecordsTable},
		{7, createBackupJobsTable},
		{8, createAuditLogsTable},
		{9, createSettingsTable},
		{10, insertDefaultSettings},
		{11, addSubdomainSupport},
		{12, addPerformanceIndices},
		{13, addResellerSupport},
		{14, addAliasTable},
		{15, addPackagesTable},
	}

	// Migration versiyon tablosunu oluştur
	if _, err := db.conn.ExecContext(ctx, createMigrationsTable); err != nil {
		return fmt.Errorf("migration tablosu oluşturulamadı: %w", err)
	}

	for _, m := range migrations {
		// Bu migration zaten uygulandı mı?
		var exists bool
		err := db.conn.QueryRowContext(ctx,
			"SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = ?)", m.version,
		).Scan(&exists)

		if err != nil {
			return fmt.Errorf("migration kontrol hatası (v%d): %w", m.version, err)
		}

		if exists {
			continue
		}

		// Migration'ı çalıştır
		if _, err := db.conn.ExecContext(ctx, m.query); err != nil {
			return fmt.Errorf("migration hatası (v%d): %w", m.version, err)
		}

		// Migration kaydını ekle
		if _, err := db.conn.ExecContext(ctx,
			"INSERT INTO schema_migrations (version) VALUES (?)", m.version,
		); err != nil {
			return fmt.Errorf("migration kayıt hatası (v%d): %w", m.version, err)
		}
	}

	return nil
}

const createMigrationsTable = `
CREATE TABLE IF NOT EXISTS schema_migrations (
    version   INTEGER PRIMARY KEY,
    applied_at TEXT NOT NULL DEFAULT (datetime('now'))
)`

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    username        TEXT NOT NULL UNIQUE,
    email           TEXT NOT NULL UNIQUE,
    password_hash   TEXT NOT NULL,
    role            TEXT NOT NULL DEFAULT 'user' CHECK(role IN ('admin','reseller','user')),
    totp_secret     TEXT NOT NULL DEFAULT '',
    totp_enabled    INTEGER NOT NULL DEFAULT 0,
    home_dir        TEXT NOT NULL DEFAULT '',
    shell           TEXT NOT NULL DEFAULT '/bin/bash',
    quota_limit     INTEGER NOT NULL DEFAULT 0,
    login_attempts  INTEGER NOT NULL DEFAULT 0,
    locked_until    TEXT,
    last_login_at   TEXT,
    last_login_ip   TEXT NOT NULL DEFAULT '',
    status          TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','inactive','locked')),
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now'))
)`

const createDomainsTable = `
CREATE TABLE IF NOT EXISTS domains (
    id               INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id          INTEGER NOT NULL,
    domain           TEXT NOT NULL UNIQUE,
    document_root    TEXT NOT NULL,
    php_version      TEXT NOT NULL DEFAULT '8.3',
    ssl_enabled      INTEGER NOT NULL DEFAULT 0,
    force_https      INTEGER NOT NULL DEFAULT 1,
    bandwidth_limit  INTEGER NOT NULL DEFAULT 0,
    disk_limit       INTEGER NOT NULL DEFAULT 0,
    status           TEXT NOT NULL DEFAULT 'active' CHECK(status IN ('active','inactive','error')),
    created_at       TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at       TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
)`

const createEmailsTable = `
CREATE TABLE IF NOT EXISTS emails (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id          INTEGER NOT NULL,
    email              TEXT NOT NULL UNIQUE,
    password_hash      TEXT NOT NULL,
    quota              INTEGER NOT NULL DEFAULT 1024,
    forward_to         TEXT NOT NULL DEFAULT '',
    autoresponder_msg  TEXT NOT NULL DEFAULT '',
    status             TEXT NOT NULL DEFAULT 'active',
    created_at         TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at         TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
)`

const createDatabasesTable = `
CREATE TABLE IF NOT EXISTS databases (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id       INTEGER NOT NULL,
    name          TEXT NOT NULL,
    username      TEXT NOT NULL,
    password_enc  TEXT NOT NULL,
    charset       TEXT NOT NULL DEFAULT 'utf8mb4',
    collation     TEXT NOT NULL DEFAULT 'utf8mb4_unicode_ci',
    remote_access INTEGER NOT NULL DEFAULT 0,
    status        TEXT NOT NULL DEFAULT 'active',
    created_at    TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE(name)
)`

const createSSLCertsTable = `
CREATE TABLE IF NOT EXISTS ssl_certs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id   INTEGER NOT NULL UNIQUE,
    type        TEXT NOT NULL DEFAULT 'lets_encrypt',
    common_name TEXT NOT NULL,
    certificate TEXT NOT NULL,
    private_key TEXT NOT NULL,
    chain       TEXT NOT NULL DEFAULT '',
    issuer      TEXT NOT NULL DEFAULT '',
    expires_at  TEXT NOT NULL,
    auto_renew  INTEGER NOT NULL DEFAULT 1,
    status      TEXT NOT NULL DEFAULT 'active',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
)`

const createDNSRecordsTable = `
CREATE TABLE IF NOT EXISTS dns_records (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id  INTEGER NOT NULL,
    type       TEXT NOT NULL CHECK(type IN ('A','AAAA','CNAME','MX','TXT','NS','SRV','CAA')),
    name       TEXT NOT NULL DEFAULT '',
    value      TEXT NOT NULL,
    ttl        INTEGER NOT NULL DEFAULT 3600,
    priority   INTEGER NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
)`

const createBackupJobsTable = `
CREATE TABLE IF NOT EXISTS backup_jobs (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id     INTEGER NOT NULL,
    domain_id   INTEGER,
    type        TEXT NOT NULL DEFAULT 'full',
    destination TEXT NOT NULL DEFAULT 'local',
    dest_config TEXT NOT NULL DEFAULT '{}',
    schedule    TEXT NOT NULL DEFAULT '',
    retention   INTEGER NOT NULL DEFAULT 7,
    last_run    TEXT,
    next_run    TEXT,
    status      TEXT NOT NULL DEFAULT 'pending',
    created_at  TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at  TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE SET NULL
)`

const createAuditLogsTable = `
CREATE TABLE IF NOT EXISTS audit_logs (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id    INTEGER,
    action     TEXT NOT NULL,
    resource   TEXT NOT NULL,
    details    TEXT NOT NULL DEFAULT '{}',
    ip         TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
)`

const createSettingsTable = `
CREATE TABLE IF NOT EXISTS settings (
    key         TEXT PRIMARY KEY,
    value       TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
)`

const addSubdomainSupport = `
ALTER TABLE domains ADD COLUMN parent_id INTEGER DEFAULT NULL REFERENCES domains(id) ON DELETE CASCADE;
CREATE INDEX IF NOT EXISTS idx_domains_parent ON domains(parent_id);
`
const addPerformanceIndices = `
CREATE INDEX IF NOT EXISTS idx_domains_user ON domains(user_id);
CREATE INDEX IF NOT EXISTS idx_domains_user_domain ON domains(user_id, domain);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at);
CREATE INDEX IF NOT EXISTS idx_audit_user_created ON audit_logs(user_id, created_at);
CREATE INDEX IF NOT EXISTS idx_dns_domain ON dns_records(domain_id);
CREATE INDEX IF NOT EXISTS idx_backup_user ON backup_jobs(user_id);
CREATE INDEX IF NOT EXISTS idx_ssl_expires ON ssl_certs(expires_at);
CREATE INDEX IF NOT EXISTS idx_databases_user ON databases(user_id);
CREATE INDEX IF NOT EXISTS idx_emails_domain ON emails(domain_id);
`


const insertDefaultSettings = `
INSERT OR IGNORE INTO settings (key, value, description) VALUES
    ('site_name', 'Aura Panel', 'Panel adı'),
    ('site_description', 'Modern Hosting Control Panel', 'Panel açıklaması'),
    ('default_php_version', '8.3', 'Varsayılan PHP sürümü'),
    ('default_charset', 'utf8mb4', 'Varsayılan veritabanı karakter seti'),
    ('max_domains_per_user', '100', 'Kullanıcı başına maksimum domain'),
    ('max_emails_per_domain', '100', 'Domain başına maksimum email'),
    ('max_databases_per_user', '50', 'Kullanıcı başına maksimum veritabanı'),
    ('backup_retention_days', '30', 'Yedek saklama süresi (gün)'),
    ('ssl_auto_renew_days', '30', 'SSL otomatik yenileme (gün kala)'),
    ('monitoring_interval', '60', 'Monitoring veri toplama aralığı (saniye)')
`

const addResellerSupport = `
ALTER TABLE users ADD COLUMN reseller_id INTEGER DEFAULT NULL REFERENCES users(id) ON DELETE SET NULL;
ALTER TABLE users ADD COLUMN max_domains INTEGER NOT NULL DEFAULT 10;
ALTER TABLE users ADD COLUMN max_emails INTEGER NOT NULL DEFAULT 20;
ALTER TABLE users ADD COLUMN max_databases INTEGER NOT NULL DEFAULT 10;
CREATE INDEX IF NOT EXISTS idx_users_reseller ON users(reseller_id);

-- Reseller quota ayarlari
INSERT OR IGNORE INTO settings (key, value, description) VALUES
    ('reseller_max_users', '50', 'Reseller basina maksimum kullanici'),
    ('reseller_max_domains', '200', 'Reseller basina maksimum domain'),
    ('reseller_max_disk_gb', '50', 'Reseller basina maksimum disk (GB)'),
    ('user_max_domains', '10', 'Kullanici basina varsayilan maksimum domain'),
    ('user_max_emails', '20', 'Kullanici basina varsayilan maksimum email'),
    ('user_max_databases', '10', 'Kullanici basina varsayilan maksimum veritabani'),
    ('user_max_disk_gb', '5', 'Kullanici basina varsayilan maksimum disk (GB)');
`

const addAliasTable = `
CREATE TABLE IF NOT EXISTS domain_aliases (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    domain_id  INTEGER NOT NULL,
    alias      TEXT NOT NULL UNIQUE,
    type       TEXT NOT NULL DEFAULT 'park' CHECK(type IN ('park','redirect')),
    target     TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    FOREIGN KEY (domain_id) REFERENCES domains(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_aliases_domain ON domain_aliases(domain_id);
`

const addPackagesTable = `
CREATE TABLE IF NOT EXISTS hosting_packages (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    cpu_shares  INTEGER NOT NULL DEFAULT 512,   -- CPU payi (1024 = 1 core)
    memory_mb   INTEGER NOT NULL DEFAULT 512,   -- RAM limiti (MB)
    nproc       INTEGER NOT NULL DEFAULT 50,    -- Max process sayisi
    disk_mb     INTEGER NOT NULL DEFAULT 2048,  -- Disk limiti (MB)
    max_domains INTEGER NOT NULL DEFAULT 5,     -- Max domain
    max_emails  INTEGER NOT NULL DEFAULT 10,    -- Max email
    max_db      INTEGER NOT NULL DEFAULT 5,     -- Max veritabani
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

ALTER TABLE users ADD COLUMN package_id INTEGER DEFAULT NULL REFERENCES hosting_packages(id);

-- Varsayilan paketler
INSERT OR IGNORE INTO hosting_packages (id, name, cpu_shares, memory_mb, nproc, disk_mb, max_domains, max_emails, max_db) VALUES
    (1, 'Basic', 512, 512, 20, 2048, 3, 5, 3),
    (2, 'Pro', 1024, 1024, 50, 5120, 10, 20, 10),
    (3, 'Enterprise', 2048, 4096, 100, 20480, 50, 100, 50),
    (4, 'Custom', 1024, 1024, 50, 5120, 10, 20, 10);
`
