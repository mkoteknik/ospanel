package sqlite

import (
	"context"
	"time"

	"github.com/mkoteknik/ospanel/internal/model"
)

// CreateDomain yeni domain oluşturur
func (db *DB) CreateDomain(ctx context.Context, domain *model.Domain) error {
	now := time.Now().UTC().Format(time.RFC3339)
	domain.CreatedAt = time.Now().UTC()
	domain.UpdatedAt = time.Now().UTC()

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO domains (user_id, domain, document_root, php_version,
			ssl_enabled, force_https, bandwidth_limit, disk_limit, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		domain.UserID, domain.Domain, domain.DocumentRoot, domain.PHPVersion,
		domain.SSLenabled, domain.ForceHTTPS, domain.BandwidthLimit,
		domain.DiskLimit, string(domain.Status), now, now,
	)
	if err != nil {
		return err
	}

	id, _ := result.LastInsertId()
	domain.ID = id
	return nil
}

// scanDomainRow domain satırını tarar
func scanDomainRow(scanner interface{ Scan(...interface{}) error }) (*model.Domain, error) {
	d := &model.Domain{}
	var createdAt, updatedAt string
	err := scanner.Scan(&d.ID, &d.UserID, &d.Domain, &d.DocumentRoot, &d.PHPVersion,
		&d.SSLenabled, &d.ForceHTTPS, &d.BandwidthLimit, &d.DiskLimit,
		&d.Status, &createdAt, &updatedAt,
	)
	if err != nil {
		return nil, err
	}
	d.CreatedAt = scanTime(createdAt)
	d.UpdatedAt = scanTime(updatedAt)
	return d, nil
}

// GetDomain ID'ye göre domain getirir
func (db *DB) GetDomain(ctx context.Context, id int64) (*model.Domain, error) {
	return scanDomainRow(db.conn.QueryRowContext(ctx, `
		SELECT id, user_id, domain, document_root, php_version,
			ssl_enabled, force_https, bandwidth_limit, disk_limit, status, created_at, updated_at
		FROM domains WHERE id = ?`, id))
}

// GetDomainByName domain adına göre getirir
func (db *DB) GetDomainByName(ctx context.Context, name string) (*model.Domain, error) {
	return scanDomainRow(db.conn.QueryRowContext(ctx, `
		SELECT id, user_id, domain, document_root, php_version,
			ssl_enabled, force_https, bandwidth_limit, disk_limit, status, created_at, updated_at
		FROM domains WHERE domain = ?`, name))
}

// ListDomains kullanıcının domainlerini listeler
func (db *DB) ListDomains(ctx context.Context, userID int64) ([]*model.Domain, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, user_id, domain, document_root, php_version,
			ssl_enabled, force_https, bandwidth_limit, disk_limit, status, created_at, updated_at
		FROM domains WHERE user_id = ? ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var domains []*model.Domain
	for rows.Next() {
		d := &model.Domain{}
		var createdAt, updatedAt string
		if err := rows.Scan(&d.ID, &d.UserID, &d.Domain, &d.DocumentRoot, &d.PHPVersion,
			&d.SSLenabled, &d.ForceHTTPS, &d.BandwidthLimit, &d.DiskLimit,
			&d.Status, &createdAt, &updatedAt,
		); err != nil {
			return nil, err
		}
		d.CreatedAt = scanTime(createdAt)
		d.UpdatedAt = scanTime(updatedAt)
		domains = append(domains, d)
	}
	return domains, rows.Err()
}

// UpdateDomain domain günceller
func (db *DB) UpdateDomain(ctx context.Context, domain *model.Domain) error {
	now := time.Now().UTC().Format(time.RFC3339)
	domain.UpdatedAt = time.Now().UTC()

	_, err := db.conn.ExecContext(ctx, `
		UPDATE domains SET php_version=?, ssl_enabled=?, force_https=?,
			bandwidth_limit=?, disk_limit=?, status=?, updated_at=?
		WHERE id=?`,
		domain.PHPVersion, domain.SSLenabled, domain.ForceHTTPS,
		domain.BandwidthLimit, domain.DiskLimit, string(domain.Status), now,
		domain.ID,
	)
	return err
}

// DeleteDomain domain siler
func (db *DB) DeleteDomain(ctx context.Context, id int64) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM domains WHERE id=?", id)
	return err
}
