package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/mkoteknik/ospanel/internal/model"
)

// scanTime RFC3339 string'i time.Time'e çevirir
func scanTime(s string) time.Time {
	t, _ := time.Parse(time.RFC3339, s)
	return t
}

// userSelectCols ortak SELECT kolonlari (tekrar eden kod azaltmak icin)
const userSelectCols = `id, username, email, password_hash, role, reseller_id,
	totp_secret, totp_enabled, home_dir, shell, quota_limit,
	max_domains, max_emails, max_databases,
	login_attempts, locked_until, last_login_at, last_login_ip,
	status, created_at, updated_at`

// userInsertCols ortak INSERT kolonlari
const userInsertCols = `username, email, password_hash, role, reseller_id,
	totp_secret, totp_enabled, home_dir, shell, quota_limit,
	max_domains, max_emails, max_databases, status, created_at, updated_at`

// scanUserRow bir kullanıcı satırını tarar (reseller destegi ile)
func scanUserRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.User, error) {
	user := &model.User{}
	var resellerID sql.NullInt64
	var lockedUntil sql.NullString
	var lastLogin sql.NullString
	var createdAt, updatedAt string

	err := scanner.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &resellerID,
		&user.TOTPSecret, &user.TOTPEnabled,
		&user.HomeDir, &user.Shell, &user.QuotaLimit,
		&user.MaxDomains, &user.MaxEmails, &user.MaxDatabases,
		&user.LoginAttempts, &lockedUntil, &lastLogin, &user.LastLoginIP,
		&user.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}

	if resellerID.Valid {
		user.ResellerID = &resellerID.Int64
	}
	if lockedUntil.Valid {
		t := scanTime(lockedUntil.String)
		user.LockedUntil = &t
	}
	if lastLogin.Valid {
		t := scanTime(lastLogin.String)
		user.LastLoginAt = &t
	}
	user.CreatedAt = scanTime(createdAt)
	user.UpdatedAt = scanTime(updatedAt)

	return user, nil
}

// CreateUser yeni kullanıcı oluşturur (reseller destegi ile)
func (db *DB) CreateUser(ctx context.Context, user *model.User) error {
	now := time.Now().UTC().Format(time.RFC3339)
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO users (`+userInsertCols+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Username, user.Email, user.PasswordHash, string(user.Role),
		user.ResellerID, user.TOTPSecret, user.TOTPEnabled,
		user.HomeDir, user.Shell, user.QuotaLimit,
		user.MaxDomains, user.MaxEmails, user.MaxDatabases,
		string(user.Status), now, now,
	)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	user.ID = id
	return nil
}

// GetUser ID'ye göre kullanıcı getirir
func (db *DB) GetUser(ctx context.Context, id int64) (*model.User, error) {
	return scanUserRow(db.conn.QueryRowContext(ctx, `
		SELECT `+userSelectCols+` FROM users WHERE id = ?`, id))
}

// GetUserByUsername kullanıcı adına göre kullanıcı getirir
func (db *DB) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return scanUserRow(db.conn.QueryRowContext(ctx, `
		SELECT `+userSelectCols+` FROM users WHERE username = ?`, username))
}

// GetUserByEmail email'e göre kullanıcı getirir
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return scanUserRow(db.conn.QueryRowContext(ctx, `
		SELECT `+userSelectCols+` FROM users WHERE email = ?`, email))
}

// ListUsers tüm kullanıcıları listeler
func (db *DB) ListUsers(ctx context.Context) ([]*model.User, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT `+userSelectCols+` FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserRows(rows)
}

// ListUsersByReseller reseller'a bagli kullanicilari listeler
func (db *DB) ListUsersByReseller(ctx context.Context, resellerID int64) ([]*model.User, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT `+userSelectCols+` FROM users WHERE reseller_id = ? ORDER BY id`, resellerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanUserRows(rows)
}

// CountUsersByReseller reseller'a bagli kullanici sayisini dondurur
func (db *DB) CountUsersByReseller(ctx context.Context, resellerID int64) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM users WHERE reseller_id = ? AND status != 'inactive'", resellerID).Scan(&count)
	return count, err
}

// CountDomainsByUser kullaniciya ait domain sayisini dondurur
func (db *DB) CountDomainsByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM domains WHERE user_id = ? AND status != 'inactive'", userID).Scan(&count)
	return count, err
}

// CountDatabasesByUser kullaniciya ait veritabani sayisini dondurur
func (db *DB) CountDatabasesByUser(ctx context.Context, userID int64) (int, error) {
	var count int
	err := db.conn.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM databases WHERE user_id = ? AND status != 'inactive'", userID).Scan(&count)
	return count, err
}

// scanUserRows rows'dan kullanici listesini tarar
func scanUserRows(rows *sql.Rows) ([]*model.User, error) {
	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		var resellerID sql.NullInt64
		var lockedUntil sql.NullString
		var lastLogin sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.Role, &resellerID,
			&user.TOTPSecret, &user.TOTPEnabled,
			&user.HomeDir, &user.Shell, &user.QuotaLimit,
			&user.MaxDomains, &user.MaxEmails, &user.MaxDatabases,
			&user.LoginAttempts, &lockedUntil, &lastLogin, &user.LastLoginIP,
			&user.Status, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		if resellerID.Valid {
			user.ResellerID = &resellerID.Int64
		}
		if lockedUntil.Valid {
			t := scanTime(lockedUntil.String)
			user.LockedUntil = &t
		}
		if lastLogin.Valid {
			t := scanTime(lastLogin.String)
			user.LastLoginAt = &t
		}
		user.CreatedAt = scanTime(createdAt)
		user.UpdatedAt = scanTime(updatedAt)
		users = append(users, user)
	}
	return users, rows.Err()
}

// UpdateUser kullanıcı günceller (reseller destegi ile)
func (db *DB) UpdateUser(ctx context.Context, user *model.User) error {
	now := time.Now().UTC().Format(time.RFC3339)
	user.UpdatedAt = time.Now().UTC()

	var lockedUntil interface{}
	if user.LockedUntil != nil {
		lockedUntil = user.LockedUntil.Format(time.RFC3339)
	}

	var lastLogin interface{}
	if user.LastLoginAt != nil {
		lastLogin = user.LastLoginAt.Format(time.RFC3339)
	}

	_, err := db.conn.ExecContext(ctx, `
		UPDATE users SET email=?, role=?, reseller_id=?, totp_secret=?, totp_enabled=?,
			home_dir=?, shell=?, quota_limit=?, max_domains=?, max_emails=?,
			max_databases=?, login_attempts=?,
			locked_until=?, last_login_at=?, last_login_ip=?, status=?, updated_at=?
		WHERE id=?`,
		user.Email, string(user.Role), user.ResellerID,
		user.TOTPSecret, user.TOTPEnabled,
		user.HomeDir, user.Shell, user.QuotaLimit,
		user.MaxDomains, user.MaxEmails, user.MaxDatabases,
		user.LoginAttempts,
		lockedUntil, lastLogin, user.LastLoginIP, string(user.Status), now,
		user.ID,
	)
	return err
}

// DeleteUser kullanıcı siler
func (db *DB) DeleteUser(ctx context.Context, id int64) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM users WHERE id=?", id)
	return err
}

// UpdateLoginAttempts giriş denemesi sayısını günceller
func (db *DB) UpdateLoginAttempts(ctx context.Context, id int64, attempts int) error {
	_, err := db.conn.ExecContext(ctx, "UPDATE users SET login_attempts=? WHERE id=?", attempts, id)
	return err
}

// LockUser kullanıcıyı kilitler
func (db *DB) LockUser(ctx context.Context, id int64, until interface{}) error {
	var untilStr interface{}
	if t, ok := until.(time.Time); ok {
		untilStr = t.Format(time.RFC3339)
	}
	_, err := db.conn.ExecContext(ctx,
		"UPDATE users SET status='locked', locked_until=? WHERE id=?", untilStr, id)
	return err
}
