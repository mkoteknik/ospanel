package sqlite

import (
	"context"
	"time"

	"github.com/mkoteknik/ospanel/internal/model"
)

// CreateAlias yeni domain alias/parked domain olusturur
func (db *DB) CreateAlias(ctx context.Context, alias *model.Alias) error {
	now := time.Now().UTC().Format(time.RFC3339)
	alias.CreatedAt = time.Now().UTC()

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO domain_aliases (domain_id, alias, type, target, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		alias.DomainID, alias.Alias, alias.Type, alias.Target, now)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	alias.ID = id
	return nil
}

// ListAliasesByDomain domain'e ait alias'lari listeler
func (db *DB) ListAliasesByDomain(ctx context.Context, domainID int64) ([]*model.Alias, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, domain_id, alias, type, target, created_at
		FROM domain_aliases WHERE domain_id = ? ORDER BY id`, domainID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var aliases []*model.Alias
	for rows.Next() {
		a := &model.Alias{}
		var createdAt string
		if err := rows.Scan(&a.ID, &a.DomainID, &a.Alias, &a.Type, &a.Target, &createdAt); err != nil {
			return nil, err
		}
		a.CreatedAt = scanTime(createdAt)
		aliases = append(aliases, a)
	}
	if aliases == nil {
		aliases = []*model.Alias{}
	}
	return aliases, rows.Err()
}

// DeleteAlias alias siler
func (db *DB) DeleteAlias(ctx context.Context, id int64) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM domain_aliases WHERE id=?", id)
	return err
}
