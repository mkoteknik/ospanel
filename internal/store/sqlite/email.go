package sqlite

import (
	"context"
	"time"

	"github.com/openspeed-panel/ospanel/internal/model"
)

// CreateEmail yeni email hesabı oluşturur
func (db *DB) CreateEmail(ctx context.Context, email *model.Email) error {
	now := time.Now().UTC().Format(time.RFC3339)
	email.CreatedAt = time.Now().UTC()
	email.UpdatedAt = time.Now().UTC()

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO emails (domain_id, email, password_hash, quota, forward_to,
			autoresponder_msg, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		email.DomainID, email.Email, email.PasswordHash, email.Quota,
		email.ForwardTo, email.AutoresponderMsg, email.Status, now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	email.ID = id
	return nil
}

// GetEmail ID'ye göre email getirir
func (db *DB) GetEmail(ctx context.Context, id int64) (*model.Email, error) {
	e := &model.Email{}
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, domain_id, email, password_hash, quota, forward_to,
			autoresponder_msg, status, created_at, updated_at
		FROM emails WHERE id=?`, id,
	).Scan(&e.ID, &e.DomainID, &e.Email, &e.PasswordHash, &e.Quota,
		&e.ForwardTo, &e.AutoresponderMsg, &e.Status, &e.CreatedAt, &e.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return e, nil
}

// ListEmails domain'e ait emailleri listeler
func (db *DB) ListEmails(ctx context.Context, domainID int64) ([]*model.Email, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, domain_id, email, password_hash, quota, forward_to,
			autoresponder_msg, status, created_at, updated_at
		FROM emails WHERE domain_id=? ORDER BY id`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var emails []*model.Email
	for rows.Next() {
		e := &model.Email{}
		if err := rows.Scan(&e.ID, &e.DomainID, &e.Email, &e.PasswordHash, &e.Quota,
			&e.ForwardTo, &e.AutoresponderMsg, &e.Status, &e.CreatedAt, &e.UpdatedAt,
		); err != nil {
			return nil, err
		}
		emails = append(emails, e)
	}
	return emails, nil
}

// UpdateEmail email günceller
func (db *DB) UpdateEmail(ctx context.Context, email *model.Email) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.ExecContext(ctx, `
		UPDATE emails SET quota=?, forward_to=?, autoresponder_msg=?, status=?, updated_at=?
		WHERE id=?`,
		email.Quota, email.ForwardTo, email.AutoresponderMsg, email.Status, now, email.ID,
	)
	return err
}

// DeleteEmail email siler
func (db *DB) DeleteEmail(ctx context.Context, id int64) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM emails WHERE id=?", id)
	return err
}
