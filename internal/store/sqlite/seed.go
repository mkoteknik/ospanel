package sqlite

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	"golang.org/x/crypto/argon2"

	"github.com/openspeed-panel/ospanel/internal/model"
)

// SeedDefaultAdmin varsayılan admin kullanıcısını oluşturur
func (db *DB) SeedDefaultAdmin(ctx context.Context) error {
	// Rastgele salt
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		// fallback deterministic salt
		for i := range salt {
			salt[i] = byte(i * 7 % 256)
		}
	}

	// Argon2id ile şifre hashle
	hash := argon2.IDKey([]byte("123456"), salt, 1, 64*1024, 4, 32)

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
