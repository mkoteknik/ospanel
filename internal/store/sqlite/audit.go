package sqlite

import (
	"context"
	"time"

	"github.com/openspeed-panel/ospanel/internal/model"
)

// CreateAuditLog denetim kaydı oluşturur
func (db *DB) CreateAuditLog(ctx context.Context, log *model.AuditLog) error {
	now := time.Now().UTC().Format(time.RFC3339)
	log.CreatedAt = time.Now().UTC()

	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO audit_logs (user_id, action, resource, details, ip, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		log.UserID, log.Action, log.Resource, log.Details, log.IP, now,
	)
	return err
}

// ListAuditLogs denetim kayıtlarını listeler
func (db *DB) ListAuditLogs(ctx context.Context, limit, offset int) ([]*model.AuditLog, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, user_id, action, resource, details, ip, created_at
		FROM audit_logs ORDER BY id DESC LIMIT ? OFFSET ?`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*model.AuditLog
	for rows.Next() {
		l := &model.AuditLog{}
		if err := rows.Scan(&l.ID, &l.UserID, &l.Action, &l.Resource,
			&l.Details, &l.IP, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, nil
}

// GetSetting ayar değerini getirir
func (db *DB) GetSetting(ctx context.Context, key string) (*model.Setting, error) {
	s := &model.Setting{}
	err := db.conn.QueryRowContext(ctx, `
		SELECT key, value, description, updated_at FROM settings WHERE key=?`, key,
	).Scan(&s.Key, &s.Value, &s.Description, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// SetSetting ayar değerini günceller
func (db *DB) SetSetting(ctx context.Context, setting *model.Setting) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.ExecContext(ctx, `
		INSERT INTO settings (key, value, description, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at`,
		setting.Key, setting.Value, setting.Description, now,
	)
	return err
}

// ListSettings tüm ayarları listeler
func (db *DB) ListSettings(ctx context.Context) ([]*model.Setting, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT key, value, description, updated_at FROM settings ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []*model.Setting
	for rows.Next() {
		s := &model.Setting{}
		if err := rows.Scan(&s.Key, &s.Value, &s.Description, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, nil
}
