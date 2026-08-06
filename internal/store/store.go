package store

import (
	"context"

	"github.com/openspeed-panel/ospanel/internal/model"
)

// Store tüm veri erişim işlemlerini tanımlayan arayüz
type Store interface {
	// Migration
	Migrate(ctx context.Context) error

	// SeedDefaultAdmin ilk admin kullanıcısını oluşturur
	SeedDefaultAdmin(ctx context.Context) error

	// User
	CreateUser(ctx context.Context, user *model.User) error
	GetUser(ctx context.Context, id int64) (*model.User, error)
	GetUserByUsername(ctx context.Context, username string) (*model.User, error)
	GetUserByEmail(ctx context.Context, email string) (*model.User, error)
	ListUsers(ctx context.Context) ([]*model.User, error)
	UpdateUser(ctx context.Context, user *model.User) error
	DeleteUser(ctx context.Context, id int64) error
	UpdateLoginAttempts(ctx context.Context, id int64, attempts int) error
	LockUser(ctx context.Context, id int64, until interface{}) error

	// Domain
	CreateDomain(ctx context.Context, domain *model.Domain) error
	GetDomain(ctx context.Context, id int64) (*model.Domain, error)
	GetDomainByName(ctx context.Context, name string) (*model.Domain, error)
	ListDomains(ctx context.Context, userID int64) ([]*model.Domain, error)
	UpdateDomain(ctx context.Context, domain *model.Domain) error
	DeleteDomain(ctx context.Context, id int64) error

	// Email
	CreateEmail(ctx context.Context, email *model.Email) error
	GetEmail(ctx context.Context, id int64) (*model.Email, error)
	ListEmails(ctx context.Context, domainID int64) ([]*model.Email, error)
	UpdateEmail(ctx context.Context, email *model.Email) error
	DeleteEmail(ctx context.Context, id int64) error

	// Database
	CreateDatabase(ctx context.Context, db *model.Database) error
	GetDatabase(ctx context.Context, id int64) (*model.Database, error)
	ListDatabases(ctx context.Context, userID int64) ([]*model.Database, error)
	DeleteDatabase(ctx context.Context, id int64) error

	// SSL
	CreateSSLCert(ctx context.Context, cert *model.SSLCertificate) error
	GetSSLCert(ctx context.Context, domainID int64) (*model.SSLCertificate, error)
	UpdateSSLCert(ctx context.Context, cert *model.SSLCertificate) error
	DeleteSSLCert(ctx context.Context, id int64) error
	ListExpiringCerts(ctx context.Context, days int) ([]*model.SSLCertificate, error)

	// DNS
	CreateDNSRecord(ctx context.Context, record *model.DNSRecord) error
	ListDNSRecords(ctx context.Context, domainID int64) ([]*model.DNSRecord, error)
	UpdateDNSRecord(ctx context.Context, record *model.DNSRecord) error
	DeleteDNSRecord(ctx context.Context, id int64) error

	// Backup
	CreateBackupJob(ctx context.Context, job *model.BackupJob) error
	GetBackupJob(ctx context.Context, id int64) (*model.BackupJob, error)
	ListBackupJobs(ctx context.Context, userID int64) ([]*model.BackupJob, error)
	UpdateBackupJob(ctx context.Context, job *model.BackupJob) error
	DeleteBackupJob(ctx context.Context, id int64) error

	// Audit
	CreateAuditLog(ctx context.Context, log *model.AuditLog) error
	ListAuditLogs(ctx context.Context, limit, offset int) ([]*model.AuditLog, error)

	// Settings
	GetSetting(ctx context.Context, key string) (*model.Setting, error)
	SetSetting(ctx context.Context, setting *model.Setting) error
	ListSettings(ctx context.Context) ([]*model.Setting, error)

	// Close veritabanı bağlantısını kapatır
	Close() error
}
