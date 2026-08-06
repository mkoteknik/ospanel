package model

import "time"

// Email bir email hesabı
type Email struct {
	ID               int64     `json:"id"`
	DomainID         int64     `json:"domain_id"`
	Email            string    `json:"email"`
	PasswordHash     string    `json:"-"`
	Quota            int64     `json:"quota"` // MB
	ForwardTo        string    `json:"forward_to,omitempty"`
	AutoresponderMsg string    `json:"autoresponder_msg,omitempty"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}
