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

// CreateUser yeni kullanıcı oluşturur
func (db *DB) CreateUser(ctx context.Context, user *model.User) error {
	now := time.Now().UTC().Format(time.RFC3339)
	user.CreatedAt = time.Now().UTC()
	user.UpdatedAt = time.Now().UTC()

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO users (username, email, password_hash, role, totp_secret, totp_enabled,
			home_dir, shell, quota_limit, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		user.Username, user.Email, user.PasswordHash, string(user.Role),
		user.TOTPSecret, user.TOTPEnabled, user.HomeDir, user.Shell,
		user.QuotaLimit, string(user.Status), now, now,
	)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	user.ID = id
	return nil
}

// scanUserRow bir kullanıcı satırını tarar
func scanUserRow(scanner interface {
	Scan(dest ...interface{}) error
}) (*model.User, error) {
	user := &model.User{}
	var lockedUntil sql.NullString
	var lastLogin sql.NullString
	var createdAt, updatedAt string

	err := scanner.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.TOTPSecret, &user.TOTPEnabled,
		&user.HomeDir, &user.Shell, &user.QuotaLimit, &user.LoginAttempts,
		&lockedUntil, &lastLogin, &user.LastLoginIP, &user.Status,
		&createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
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

// GetUser ID'ye göre kullanıcı getirir
func (db *DB) GetUser(ctx context.Context, id int64) (*model.User, error) {
	return scanUserRow(db.conn.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, role, totp_secret, totp_enabled,
			home_dir, shell, quota_limit, login_attempts, locked_until,
			last_login_at, last_login_ip, status, created_at, updated_at
		FROM users WHERE id = ?`, id))
}

// GetUserByUsername kullanıcı adına göre kullanıcı getirir
func (db *DB) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	return scanUserRow(db.conn.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, role, totp_secret, totp_enabled,
			home_dir, shell, quota_limit, login_attempts, locked_until,
			last_login_at, last_login_ip, status, created_at, updated_at
		FROM users WHERE username = ?`, username))
}

// GetUserByEmail email'e göre kullanıcı getirir
func (db *DB) GetUserByEmail(ctx context.Context, email string) (*model.User, error) {
	return scanUserRow(db.conn.QueryRowContext(ctx, `
		SELECT id, username, email, password_hash, role, totp_secret, totp_enabled,
			home_dir, shell, quota_limit, login_attempts, locked_until,
			last_login_at, last_login_ip, status, created_at, updated_at
		FROM users WHERE email = ?`, email))
}

// ListUsers tüm kullanıcıları listeler
func (db *DB) ListUsers(ctx context.Context) ([]*model.User, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, username, email, password_hash, role, totp_secret, totp_enabled,
			home_dir, shell, quota_limit, login_attempts, locked_until,
			last_login_at, last_login_ip, status, created_at, updated_at
		FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*model.User
	for rows.Next() {
		user := &model.User{}
		var lockedUntil sql.NullString
		var lastLogin sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.PasswordHash,
			&user.Role, &user.TOTPSecret, &user.TOTPEnabled,
			&user.HomeDir, &user.Shell, &user.QuotaLimit, &user.LoginAttempts,
			&lockedUntil, &lastLogin, &user.LastLoginIP, &user.Status,
			&createdAt, &updatedAt,
		); err != nil {
			return nil, err
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

// UpdateUser kullanıcı günceller
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
		UPDATE users SET email=?, role=?, totp_secret=?, totp_enabled=?,
			home_dir=?, shell=?, quota_limit=?, login_attempts=?,
			locked_until=?, last_login_at=?, last_login_ip=?, status=?, updated_at=?
		WHERE id=?`,
		user.Email, string(user.Role), user.TOTPSecret, user.TOTPEnabled,
		user.HomeDir, user.Shell, user.QuotaLimit, user.LoginAttempts,
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
