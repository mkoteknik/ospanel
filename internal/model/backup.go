package model

import "time"

// BackupJob yedekleme işi
type BackupJob struct {
	ID          int64      `json:"id"`
	UserID      int64      `json:"user_id"`
	DomainID    *int64     `json:"domain_id,omitempty"`
	Type        string     `json:"type"` // full, incremental, differential
	Destination string     `json:"destination"` // local, s3, ftp, sftp
	DestConfig  string     `json:"-"`
	Schedule    string     `json:"schedule"` // cron formatı
	Retention   int        `json:"retention"` // gün
	LastRun     *time.Time `json:"last_run"`
	NextRun     *time.Time `json:"next_run"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// AuditLog denetim kaydı
type AuditLog struct {
	ID        int64     `json:"id"`
	UserID    *int64    `json:"user_id,omitempty"`
	Action    string    `json:"action"`
	Resource  string    `json:"resource"`
	Details   string    `json:"details"`
	IP        string    `json:"ip"`
	CreatedAt time.Time `json:"created_at"`
}

// Setting sistem ayarı
type Setting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}
