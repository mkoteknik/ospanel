package sqlite

import (
	"context"
	"time"

	"github.com/openspeed-panel/ospanel/internal/model"
)

// CreateDatabase yeni veritabanı kaydı oluşturur
func (db *DB) CreateDatabase(ctx context.Context, database *model.Database) error {
	now := time.Now().UTC().Format(time.RFC3339)
	database.CreatedAt = time.Now().UTC()

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO databases (user_id, name, username, password_enc, charset, collation,
			remote_access, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		database.UserID, database.Name, database.Username, database.PasswordEnc,
		database.Charset, database.Collation, database.RemoteAccess, database.Status, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	database.ID = id
	return nil
}

// GetDatabase ID'ye göre veritabanı getirir
func (db *DB) GetDatabase(ctx context.Context, id int64) (*model.Database, error) {
	d := &model.Database{}
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, user_id, name, username, password_enc, charset, collation,
			remote_access, status, created_at
		FROM databases WHERE id=?`, id,
	).Scan(&d.ID, &d.UserID, &d.Name, &d.Username, &d.PasswordEnc,
		&d.Charset, &d.Collation, &d.RemoteAccess, &d.Status, &d.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return d, nil
}

// ListDatabases kullanıcının veritabanlarını listeler
func (db *DB) ListDatabases(ctx context.Context, userID int64) ([]*model.Database, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, user_id, name, username, password_enc, charset, collation,
			remote_access, status, created_at
		FROM databases WHERE user_id=? ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dbs []*model.Database
	for rows.Next() {
		d := &model.Database{}
		if err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.Username, &d.PasswordEnc,
			&d.Charset, &d.Collation, &d.RemoteAccess, &d.Status, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		dbs = append(dbs, d)
	}
	return dbs, nil
}

// DeleteDatabase veritabanı siler
func (db *DB) DeleteDatabase(ctx context.Context, id int64) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM databases WHERE id=?", id)
	return err
}
