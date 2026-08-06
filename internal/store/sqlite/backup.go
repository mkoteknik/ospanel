package sqlite

import (
	"context"
	"database/sql"
	"time"

	"github.com/mkoteknik/ospanel/internal/model"
)

// CreateBackupJob yedekleme işi oluşturur
func (db *DB) CreateBackupJob(ctx context.Context, job *model.BackupJob) error {
	now := time.Now().UTC().Format(time.RFC3339)
	job.CreatedAt = time.Now().UTC()
	job.UpdatedAt = time.Now().UTC()

	result, err := db.conn.ExecContext(ctx, `
		INSERT INTO backup_jobs (user_id, domain_id, type, destination, dest_config,
			schedule, retention, status, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.UserID, job.DomainID, job.Type, job.Destination, job.DestConfig,
		job.Schedule, job.Retention, "pending", now, now,
	)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	job.ID = id
	return nil
}

// GetBackupJob ID'ye göre yedekleme işi getirir
func (db *DB) GetBackupJob(ctx context.Context, id int64) (*model.BackupJob, error) {
	j := &model.BackupJob{}
	var lastRun, nextRun sql.NullString
	var domainID sql.NullInt64

	err := db.conn.QueryRowContext(ctx, `
		SELECT id, user_id, domain_id, type, destination, dest_config,
			schedule, retention, last_run, next_run, status, created_at, updated_at
		FROM backup_jobs WHERE id=?`, id,
	).Scan(&j.ID, &j.UserID, &domainID, &j.Type, &j.Destination, &j.DestConfig,
		&j.Schedule, &j.Retention, &lastRun, &nextRun, &j.Status,
		&j.CreatedAt, &j.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if domainID.Valid {
		j.DomainID = &domainID.Int64
	}
	if lastRun.Valid {
		t, _ := time.Parse(time.RFC3339, lastRun.String)
		j.LastRun = &t
	}
	if nextRun.Valid {
		t, _ := time.Parse(time.RFC3339, nextRun.String)
		j.NextRun = &t
	}

	return j, nil
}

// ListBackupJobs kullanıcının yedekleme işlerini listeler
func (db *DB) ListBackupJobs(ctx context.Context, userID int64) ([]*model.BackupJob, error) {
	rows, err := db.conn.QueryContext(ctx, `
		SELECT id, user_id, domain_id, type, destination, dest_config,
			schedule, retention, last_run, next_run, status, created_at, updated_at
		FROM backup_jobs WHERE user_id=? ORDER BY id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []*model.BackupJob
	for rows.Next() {
		j := &model.BackupJob{}
		var lastRun, nextRun sql.NullString
		var domainID sql.NullInt64
		if err := rows.Scan(&j.ID, &j.UserID, &domainID, &j.Type, &j.Destination,
			&j.DestConfig, &j.Schedule, &j.Retention, &lastRun, &nextRun,
			&j.Status, &j.CreatedAt, &j.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if domainID.Valid {
			j.DomainID = &domainID.Int64
		}
		if lastRun.Valid {
			t, _ := time.Parse(time.RFC3339, lastRun.String)
			j.LastRun = &t
		}
		if nextRun.Valid {
			t, _ := time.Parse(time.RFC3339, nextRun.String)
			j.NextRun = &t
		}
		jobs = append(jobs, j)
	}
	return jobs, nil
}

// UpdateBackupJob yedekleme işi günceller
func (db *DB) UpdateBackupJob(ctx context.Context, job *model.BackupJob) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.conn.ExecContext(ctx, `
		UPDATE backup_jobs SET schedule=?, retention=?, status=?, last_run=?, next_run=?, updated_at=?
		WHERE id=?`,
		job.Schedule, job.Retention, job.Status, job.LastRun, job.NextRun, now, job.ID,
	)
	return err
}

// DeleteBackupJob yedekleme işi siler
func (db *DB) DeleteBackupJob(ctx context.Context, id int64) error {
	_, err := db.conn.ExecContext(ctx, "DELETE FROM backup_jobs WHERE id=?", id)
	return err
}
