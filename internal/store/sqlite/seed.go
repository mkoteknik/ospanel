package sqlite

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/mkoteknik/ospanel/internal/model"
)

// SeedDefaultAdmin varsayılan admin kullanıcısını oluşturur
func (db *DB) SeedDefaultAdmin(ctx context.Context) error {
	// Şifre: env'den oku, yoksa rastgele üret
	password := os.Getenv("OSPANEL_ADMIN_PASS")
	if password == "" {
		password = "admin123" // fallback (development)
	}

	// Rastgele salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		for i := range salt {
			salt[i] = byte(i * 7 % 256)
		}
	}

	// Argon2id ile şifre hashle
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	// Salt + hash birleştir ve encode et
	encoded := "ospanel$v1$" + hex.EncodeToString(salt) + "$" + hex.EncodeToString(hash)

	now := time.Now().UTC()
	user := &model.User{
		Username:     "admin",
		Email:        "admin@localhost",
		PasswordHash: encoded,
		Role:         model.RoleAdmin,
		HomeDir:      "/home/admin",
		Shell:        "/bin/bash",
		Status:       model.StatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	return db.CreateUser(ctx, user)
}
