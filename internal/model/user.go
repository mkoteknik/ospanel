package model

import "time"

// UserRole kullanıcı rolü
type UserRole string

const (
	RoleAdmin    UserRole = "admin"
	RoleReseller UserRole = "reseller"
	RoleUser     UserRole = "user"
)

// UserStatus kullanıcı durumu
type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusInactive UserStatus = "inactive"
	StatusLocked   UserStatus = "locked"
)

// User sistem kullanıcısı
type User struct {
	ID             int64      `json:"id"`
	Username       string     `json:"username"`
	Email          string     `json:"email"`
	PasswordHash   string     `json:"-"` // JSON'da gözükmez
	Role           UserRole   `json:"role"`
	ResellerID     *int64     `json:"reseller_id,omitempty"` // Bagli oldugu reseller
	TOTPSecret     string     `json:"-"` // JSON'da gözükmez
	TOTPEnabled    bool       `json:"totp_enabled"`
	HomeDir        string     `json:"home_dir"`
	Shell          string     `json:"shell"`
	QuotaLimit     int64      `json:"quota_limit"` // MB disk
	MaxDomains     int        `json:"max_domains"`
	MaxEmails      int        `json:"max_emails"`
	MaxDatabases   int        `json:"max_databases"`
	LoginAttempts  int        `json:"-"`
	LockedUntil    *time.Time `json:"-"`
	LastLoginAt    *time.Time `json:"last_login_at"`
	LastLoginIP    string     `json:"last_login_ip"`
	Status         UserStatus `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

// LoginRequest giriş isteği
type LoginRequest struct {
	Username string `json:"username" validate:"required,min=3,max=64"`
	Password string `json:"password" validate:"required,min=8"`
	TOTPCode string `json:"totp_code,omitempty"`
}

// LoginResponse giriş yanıtı
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
	User         *User  `json:"user"`
}

// ChangePasswordRequest şifre değiştirme isteği
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}
