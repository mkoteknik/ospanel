package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"os"
	"strings"
)

// getMasterKey master key'i dosyadan veya env'den alır, yoksa nil döner.
// Sırasıyla dener: env var, verilen yol, /etc/ospanel/master.key, $OSPANEL_DATA_DIR/master.key
func getMasterKey(masterKeyPath string) []byte {
	// 1. Env'de direkt key var mi?
	if path := os.Getenv("OSPANEL_MASTER_KEY"); path != "" && len(path) >= 32 {
		if len(path) == 64 {
			return []byte(path)[:32]
		}
		h := sha256.Sum256([]byte(path))
		return h[:]
	}

	// 2. Denenecek yollar
	searchPaths := []string{}
	if masterKeyPath != "" {
		searchPaths = append(searchPaths, masterKeyPath)
	}
	searchPaths = append(searchPaths, "/etc/ospanel/master.key")
	if dataDir := os.Getenv("OSPANEL_DATA_DIR"); dataDir != "" {
		searchPaths = append(searchPaths, dataDir+"/master.key")
	}

	for _, p := range searchPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		trimmed := strings.TrimSpace(string(data))
		if len(trimmed) < 16 {
			continue
		}
		h := sha256.Sum256([]byte(trimmed))
		return h[:]
	}
	return nil
}

// Encrypt şifreyi AES-GCM ile şifreler, base64 döner. Master key yoksa HATA döner.
func Encrypt(plaintext, masterKeyPath string) (string, error) {
	key := getMasterKey(masterKeyPath)
	if key == nil {
		return "", errors.New("master key bulunamadı - önce EnsureMasterKey() çağırın")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return "enc:" + base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt şifreyi çözer
func Decrypt(ciphertext, masterKeyPath string) (string, error) {
	if ciphertext == "" {
		return "", errors.New("boş şifre")
	}
	if len(ciphertext) > 6 && ciphertext[:6] == "plain:" {
		b, err := base64.StdEncoding.DecodeString(ciphertext[6:])
		if err != nil {
			return "", err
		}
		return string(b), nil
	}
	encPart := ciphertext
	if len(ciphertext) > 4 && ciphertext[:4] == "enc:" {
		encPart = ciphertext[4:]
	} else {
		// Eski plaintext formatı
		return ciphertext, nil
	}
	key := getMasterKey(masterKeyPath)
	if key == nil {
		return "", errors.New("master key bulunamadı")
	}
	data, err := base64.StdEncoding.DecodeString(encPart)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("geçersiz şifre")
	}
	nonce, ct := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

// EnsureMasterKey master key yoksa oluşturur (0600)
func EnsureMasterKey(masterKeyPath string) error {
	if masterKeyPath == "" {
		masterKeyPath = "/etc/ospanel/master.key"
	}
	if _, err := os.Stat(masterKeyPath); err == nil {
		return nil
	}
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	// Ensure dir
	if err := os.MkdirAll("/etc/ospanel", 0700); err != nil {
		return err
	}
	// Store as hex
	enc := make([]byte, 64)
	const hexChars = "0123456789abcdef"
	for i, v := range b {
		enc[i*2] = hexChars[v>>4]
		enc[i*2+1] = hexChars[v&0x0F]
	}
	return os.WriteFile(masterKeyPath, enc, 0600)
}
