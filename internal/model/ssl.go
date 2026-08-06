package model

import "time"

// SSLCertificate SSL sertifikası
type SSLCertificate struct {
	ID             int64     `json:"id"`
	DomainID       int64     `json:"domain_id"`
	Type           string    `json:"type"` // lets_encrypt, custom, self_signed
	CommonName     string    `json:"common_name"`
	Certificate    string    `json:"-"`
	PrivateKey     string    `json:"-"`
	Chain          string    `json:"-"`
	Issuer         string    `json:"issuer"`
	ExpiresAt      time.Time `json:"expires_at"`
	AutoRenew      bool      `json:"auto_renew"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
