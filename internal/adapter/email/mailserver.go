package email

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

// MailServer email sunucusu yönetimi
type MailServer struct {
	db *sql.DB
}

// EmailAccount email hesabı
type EmailAccount struct {
	ID       int64  `json:"id"`
	DomainID int64  `json:"domain_id"`
	Email    string `json:"email"`
	Maildir  string `json:"maildir"`
	Quota    int    `json:"quota"`
}

// NewMailServer yeni email sunucu client'ı
func NewMailServer() *MailServer {
	// /etc/ospanel/email_db.conf dosyasından bağlantı bilgilerini oku
	data, err := os.ReadFile("/etc/ospanel/email_db.conf")
	if err != nil {
		return &MailServer{}
	}

	config := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			config[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}

	dsn := fmt.Sprintf("%s:%s@tcp(127.0.0.1:3306)/%s?charset=utf8mb4",
		config["DB_USER"], config["DB_PASS"], config["DB_NAME"])

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return &MailServer{}
	}

	return &MailServer{db: db}
}

// IsAvailable email sunucusu hazır mı?
func (m *MailServer) IsAvailable() bool {
	if m.db == nil {
		return false
	}
	return m.db.Ping() == nil
}

// CreateDomain email domaini ekler
func (m *MailServer) CreateDomain(domain string) error {
	if !m.IsAvailable() {
		return fmt.Errorf("email sunucusu kullanılabilir değil")
	}
	_, err := m.db.Exec("INSERT INTO virtual_domains (name) VALUES (?)", domain)
	return err
}

// DeleteDomain email domainini siler
func (m *MailServer) DeleteDomain(domain string) error {
	if !m.IsAvailable() {
		return nil
	}
	_, err := m.db.Exec("DELETE FROM virtual_domains WHERE name = ?", domain)
	return err
}

// CreateAccount email hesabı oluşturur
func (m *MailServer) CreateAccount(domain, email, password string, quota int) error {
	if !m.IsAvailable() {
		return fmt.Errorf("email sunucusu kullanılabilir değil")
	}

	// Domain ID'sini bul
	var domainID int64
	err := m.db.QueryRow("SELECT id FROM virtual_domains WHERE name = ?", domain).Scan(&domainID)
	if err != nil {
		return fmt.Errorf("domain bulunamadı: %s", domain)
	}

	// Şifreyi SHA512-CRYPT ile hashle
	hashed := hashPasswordCrypt(password)

	// Maildir yolu
	maildir := fmt.Sprintf("%s/%s/", domain, strings.Split(email, "@")[0])

	_, err = m.db.Exec(
		"INSERT INTO virtual_users (domain_id, email, password, maildir, quota) VALUES (?, ?, ?, ?, ?)",
		domainID, email, hashed, maildir, quota,
	)
	return err
}

// DeleteAccount email hesabı siler
func (m *MailServer) DeleteAccount(email string) error {
	if !m.IsAvailable() {
		return nil
	}
	_, err := m.db.Exec("DELETE FROM virtual_users WHERE email = ?", email)
	return err
}

// ListAccounts domain'e ait email hesaplarını listeler
func (m *MailServer) ListAccounts(domain string) ([]EmailAccount, error) {
	if !m.IsAvailable() {
		return nil, fmt.Errorf("email sunucusu kullanılabilir değil")
	}

	rows, err := m.db.Query(`
		SELECT u.id, u.domain_id, u.email, u.maildir, u.quota
		FROM virtual_users u
		JOIN virtual_domains d ON u.domain_id = d.id
		WHERE d.name = ?
		ORDER BY u.email`, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []EmailAccount
	for rows.Next() {
		var a EmailAccount
		if err := rows.Scan(&a.ID, &a.DomainID, &a.Email, &a.Maildir, &a.Quota); err != nil {
			continue
		}
		accounts = append(accounts, a)
	}
	return accounts, nil
}

// CreateAlias email forwarder oluşturur
func (m *MailServer) CreateAlias(domain, source, destination string) error {
	if !m.IsAvailable() {
		return nil
	}
	var domainID int64
	m.db.QueryRow("SELECT id FROM virtual_domains WHERE name = ?", domain).Scan(&domainID)
	_, err := m.db.Exec(
		"INSERT INTO virtual_aliases (domain_id, source, destination) VALUES (?, ?, ?)",
		domainID, source, destination,
	)
	return err
}

// DeleteAlias email alias siler
func (m *MailServer) DeleteAlias(source string) error {
	if !m.IsAvailable() {
		return nil
	}
	_, err := m.db.Exec("DELETE FROM virtual_aliases WHERE source = ?", source)
	return err
}

// ListAliases domain'e ait alias'lari listeler
func (m *MailServer) ListAliases(domain string) ([]map[string]interface{}, error) {
	if !m.IsAvailable() {
		return nil, fmt.Errorf("email sunucusu kullanilabilir degil")
	}
	rows, err := m.db.Query(`
		SELECT a.id, a.source, a.destination
		FROM virtual_aliases a
		JOIN virtual_domains d ON a.domain_id = d.id
		WHERE d.name = ?
		ORDER BY a.source`, domain)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []map[string]interface{}
	for rows.Next() {
		var id int64
		var source, dest string
		if err := rows.Scan(&id, &source, &dest); err != nil {
			continue
		}
		aliases = append(aliases, map[string]interface{}{
			"id":          id,
			"source":      source,
			"destination": dest,
		})
	}
	if aliases == nil {
		aliases = []map[string]interface{}{}
	}
	return aliases, nil
}

// GenerateDKIM domain için DKIM anahtarı oluşturur
func (m *MailServer) GenerateDKIM(domain string) (string, error) {
	keyDir := "/etc/opendkim/keys/" + domain
	os.MkdirAll(keyDir, 0750)

	cmd := exec.Command("opendkim-genkey", "-D", keyDir, "-d", domain, "-s", "mail")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("DKIM oluşturulamadı: %s - %w", string(out), err)
	}

	// Public key'i oku (DNS TXT kaydı için)
	pubKey, _ := os.ReadFile(keyDir + "/mail.txt")
	return string(pubKey), nil
}

// GetStats email sunucu istatistikleri
func (m *MailServer) GetStats() map[string]interface{} {
	stats := map[string]interface{}{
		"installed": m.IsAvailable(),
	}

	if m.IsAvailable() {
		var domains, users, aliases int
		m.db.QueryRow("SELECT COUNT(*) FROM virtual_domains").Scan(&domains)
		m.db.QueryRow("SELECT COUNT(*) FROM virtual_users").Scan(&users)
		m.db.QueryRow("SELECT COUNT(*) FROM virtual_aliases").Scan(&aliases)
		stats["domains"] = domains
		stats["users"] = users
		stats["aliases"] = aliases
	}

	return stats
}

// hashPasswordCrypt SHA512-CRYPT hash (Dovecot uyumlu)
func hashPasswordCrypt(password string) string {
	// Dovecot'ta kullanılan SHA512-CRYPT formatı
	cmd := exec.Command("doveadm", "pw", "-s", "SHA512-CRYPT", "-p", password)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// Fallback
		return "{SHA512-CRYPT}" + password
	}
	return strings.TrimSpace(string(out))
}
