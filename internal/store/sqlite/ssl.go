package sqlite

import (
	"context"
	"time"

	"github.com/openspeed-panel/ospanel/internal/model"
)

// CreateSSLCert SSL sertifikası oluşturur
func (db *DB) CreateSSLCert(ctx context.Context, cert *model.SSLCertificate) error {
	now := time.Now().UTC().Format(time.RFC3339)
	cert.CreatedAt = time.Now().UTC()
	cert.UpdatedAt = time.Now().UTC()

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO ssl_certs (domain_id, type, common_name, certificate, private_key,
			chain, issuer, expires_at, auto_renew, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		cert.DomainID, cert.Type, cert.CommonName, cert.Certificate, cert.PrivateKey,
		cert.Chain, cert.Issuer, cert.ExpiresAt.Format(time.RFC3339),
		cert.AutoRenew, cert.Status, now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	cert.ID = id
	return nil
}

// GetSSLCert domain'in SSL sertifikasını getirir
func (db *DB) GetSSLCert(ctx context.Context, domainID int64) (*model.SSLCertificate, error) {
	c := &model.SSLCertificate{}
	var expiresAt string
	err := db.conn.QueryRowContext(ctx, `
		SELECT id, domain_id, type, common_name, certificate, private_key,
			chain, issuer, expires_at, auto_renew, status, created_at, updated_at
		FROM ssl_certs WHERE domain_id=?`, domainID,
	).Scan(&c.ID, &c.DomainID, &c.Type, &c.CommonName, &c.Certificate,
		&c.PrivateKey, &c.Chain, &c.Issuer, &expiresAt, &c.AutoRenew,
		&c.Status, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	c.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
	return c, nil
}

// UpdateSSLCert SSL sertifikası günceller
func (db *DB) UpdateSSLCert(ctx context.Context, cert *model.SSLCertificate) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.ExecContext(ctx, `
		UPDATE ssl_certs SET certificate=?, private_key=?, chain=?,
			expires_at=?, auto_renew=?, status=?, updated_at=?
		WHERE id=?`,
		cert.Certificate, cert.PrivateKey, cert.Chain,
		cert.ExpiresAt.Format(time.RFC3339), cert.AutoRenew, cert.Status, now,
		cert.ID,
	)
	return err
}

// DeleteSSLCert SSL sertifikası siler
func (db *DB) DeleteSSLCert(ctx context.Context, id int64) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM ssl_certs WHERE id=?", id)
	return err
}

// ListExpiringCerts yakında süresi dolacak sertifikaları listeler
func (db *DB) ListExpiringCerts(ctx context.Context, days int) ([]*model.SSLCertificate, error) {
	cutoff := time.Now().UTC().Add(time.Duration(days) * 24 * time.Hour).Format(time.RFC3339)
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, domain_id, type, common_name, certificate, private_key,
			chain, issuer, expires_at, auto_renew, status, created_at, updated_at
		FROM ssl_certs WHERE expires_at <= ? AND auto_renew = 1`, cutoff)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var certs []*model.SSLCertificate
	for rows.Next() {
		c := &model.SSLCertificate{}
		var expiresAt string
		if err := rows.Scan(&c.ID, &c.DomainID, &c.Type, &c.CommonName, &c.Certificate,
			&c.PrivateKey, &c.Chain, &c.Issuer, &expiresAt, &c.AutoRenew,
			&c.Status, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		c.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)
		certs = append(certs, c)
	}
	return certs, nil
}
