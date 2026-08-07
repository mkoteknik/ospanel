package sqlite

import (
	"context"
	"time"

	"github.com/mkoteknik/ospanel/internal/model"
)

// CreateDNSRecord DNS kaydı oluşturur
func (db *DB) CreateDNSRecord(ctx context.Context, record *model.DNSRecord) error {
	now := time.Now().UTC().Format(time.RFC3339)
	record.CreatedAt = time.Now().UTC()
	record.UpdatedAt = time.Now().UTC()

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO dns_records (domain_id, type, name, value, ttl, priority, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		record.DomainID, record.Type, record.Name, record.Value,
		record.TTL, record.Priority, "active", now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	record.ID = id
	return nil
}

// ListDNSRecords domain'in DNS kayıtlarını listeler
func (db *DB) ListDNSRecords(ctx context.Context, domainID int64) ([]*model.DNSRecord, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, domain_id, type, name, value, ttl, priority, status, created_at, updated_at
		FROM dns_records WHERE domain_id=? ORDER BY type, name`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []*model.DNSRecord
	for rows.Next() {
		r := &model.DNSRecord{}
		if err := rows.Scan(&r.ID, &r.DomainID, &r.Type, &r.Name, &r.Value,
			&r.TTL, &r.Priority, &r.Status, &r.CreatedAt, &r.UpdatedAt,
		); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// GetDNSRecord tek DNS kaydı getirir
func (db *DB) GetDNSRecord(ctx context.Context, id int64) (*model.DNSRecord, error) {
	r := &model.DNSRecord{}
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, domain_id, type, name, value, ttl, priority, status, created_at, updated_at
		FROM dns_records WHERE id=?`, id,
	).Scan(&r.ID, &r.DomainID, &r.Type, &r.Name, &r.Value,
		&r.TTL, &r.Priority, &r.Status, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return r, nil
}

// UpdateDNSRecord DNS kaydı günceller
func (db *DB) UpdateDNSRecord(ctx context.Context, record *model.DNSRecord) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.ExecContext(ctx, `
		UPDATE dns_records SET value=?, ttl=?, priority=?, updated_at=?
		WHERE id=?`,
		record.Value, record.TTL, record.Priority, now, record.ID,
	)
	return err
}

// DeleteDNSRecord DNS kaydı siler
func (db *DB) DeleteDNSRecord(ctx context.Context, id int64) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM dns_records WHERE id=?", id)
	return err
}
