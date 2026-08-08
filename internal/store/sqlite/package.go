package sqlite

import (
	"context"
	"time"

	"github.com/mkoteknik/ospanel/internal/model"
)

// ListPackages tum hosting paketlerini listeler
func (db *DB) ListPackages(ctx context.Context) ([]*model.HostingPackage, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, name, cpu_shares, memory_mb, nproc, disk_mb, max_domains, max_emails, max_db, created_at
		FROM hosting_packages ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pkgs []*model.HostingPackage
	for rows.Next() {
		p := &model.HostingPackage{}
		var createdAt string
		if err := rows.Scan(&p.ID, &p.Name, &p.CPUShares, &p.MemoryMB, &p.Nproc,
			&p.DiskMB, &p.MaxDomains, &p.MaxEmails, &p.MaxDB, &createdAt); err != nil {
			return nil, err
		}
		p.CreatedAt = scanTime(createdAt)
		pkgs = append(pkgs, p)
	}
	if pkgs == nil {
		pkgs = []*model.HostingPackage{}
	}
	return pkgs, rows.Err()
}

// GetPackage ID'ye gore paket getirir
func (db *DB) GetPackage(ctx context.Context, id int64) (*model.HostingPackage, error) {
	p := &model.HostingPackage{}
	var createdAt string
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, name, cpu_shares, memory_mb, nproc, disk_mb, max_domains, max_emails, max_db, created_at
		FROM hosting_packages WHERE id = ?`, id).Scan(
		&p.ID, &p.Name, &p.CPUShares, &p.MemoryMB, &p.Nproc,
		&p.DiskMB, &p.MaxDomains, &p.MaxEmails, &p.MaxDB, &createdAt)
	if err != nil {
		return nil, err
	}
	p.CreatedAt = scanTime(createdAt)
	return p, nil
}

// UpdatePackage paket gunceller
func (db *DB) UpdatePackage(ctx context.Context, p *model.HostingPackage) error {
	_, err := db.conn.ExecContext(ctx, `
		UPDATE hosting_packages SET name=?, cpu_shares=?, memory_mb=?, nproc=?, disk_mb=?,
		max_domains=?, max_emails=?, max_db=? WHERE id=?`,
		p.Name, p.CPUShares, p.MemoryMB, p.Nproc, p.DiskMB,
		p.MaxDomains, p.MaxEmails, p.MaxDB, p.ID)
	return err
}

// CreatePackage yeni paket olusturur
func (db *DB) CreatePackage(ctx context.Context, p *model.HostingPackage) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO hosting_packages (name, cpu_shares, memory_mb, nproc, disk_mb, max_domains, max_emails, max_db, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.Name, p.CPUShares, p.MemoryMB, p.Nproc, p.DiskMB, p.MaxDomains, p.MaxEmails, p.MaxDB, now)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	p.ID = id
	p.CreatedAt = time.Now().UTC()
	return nil
}

// DeletePackage paket siler
func (db *DB) DeletePackage(ctx context.Context, id int64) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM hosting_packages WHERE id=?", id)
	return err
}

// UpdateUserPackage kullaniciya paket atar
func (db *DB) UpdateUserPackage(ctx context.Context, userID, packageID int64) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.ExecContext(ctx,
		"UPDATE users SET package_id=?, updated_at=? WHERE id=?", packageID, now, userID)
	return err
}
