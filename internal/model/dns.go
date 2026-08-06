package model

import "time"

// DNSRecord DNS kaydı
type DNSRecord struct {
	ID        int64     `json:"id"`
	DomainID  int64     `json:"domain_id"`
	Type      string    `json:"type"` // A, AAAA, CNAME, MX, TXT, NS, SRV, CAA
	Name      string    `json:"name"`
	Value     string    `json:"value"`
	TTL       int       `json:"ttl"`
	Priority  int       `json:"priority,omitempty"` // MX, SRV için
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
