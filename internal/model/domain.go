package model

import "time"

// DomainStatus domain durumu
type DomainStatus string

const (
	DomainActive   DomainStatus = "active"
	DomainInactive DomainStatus = "inactive"
	DomainError    DomainStatus = "error"
)

// Domain bir web sitesi/domain
type Domain struct {
	ID              int64        `json:"id"`
	UserID          int64        `json:"user_id"`
	ParentID        *int64       `json:"parent_id,omitempty"` // Subdomain ise parent domain ID
	Domain          string       `json:"domain"`
	DocumentRoot    string       `json:"document_root"`
	PHPVersion      string       `json:"php_version"`
	SSLenabled      bool         `json:"ssl_enabled"`
	ForceHTTPS      bool         `json:"force_https"`
	BandwidthLimit  int64        `json:"bandwidth_limit"` // MB
	DiskLimit       int64        `json:"disk_limit"`      // MB
	Status          DomainStatus `json:"status"`
	CreatedAt       time.Time    `json:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at"`
}

// CreateSubdomainRequest subdomain oluşturma isteği
type CreateSubdomainRequest struct {
	ParentID int64  `json:"parent_id"`
	Subdomain string `json:"subdomain"` // sadece "blog" kısmı
	PHPVersion string `json:"php_version"`
}

// CreateDomainRequest domain oluşturma isteği
type CreateDomainRequest struct {
	Domain     string `json:"domain" validate:"required,hostname"`
	PHPVersion string `json:"php_version" validate:"required,oneof=7.4 8.0 8.1 8.2 8.3 8.4"`
	UserID     int64  `json:"user_id"`
}

// Alias domain alias/park domain
type Alias struct {
	ID        int64     `json:"id"`
	DomainID  int64     `json:"domain_id"`
	Alias     string    `json:"alias"`
	Type      string    `json:"type"` // park, redirect
	Target    string    `json:"target,omitempty"` // redirect URL
	CreatedAt time.Time `json:"created_at"`
}
