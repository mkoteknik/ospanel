package model

import "time"

// Database bir veritabanı
type Database struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	Name            string    `json:"name"`
	Username        string    `json:"username"`
	PasswordEnc     string    `json:"-"`
	Charset         string    `json:"charset"`
	Collation       string    `json:"collation"`
	RemoteAccess    bool      `json:"remote_access"`
	Status          string    `json:"status"`
	CreatedAt       time.Time `json:"created_at"`
}

// CreateDatabaseRequest veritabanı oluşturma isteği
type CreateDatabaseRequest struct {
	Name      string `json:"name" validate:"required,min=1,max=64"`
	Username  string `json:"username" validate:"required,min=1,max=32"`
	Password  string `json:"password" validate:"required,min=8"`
	Charset   string `json:"charset"`
}
