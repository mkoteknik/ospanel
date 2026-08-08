package crypto

import (
	"os"
	"testing"
)

func TestEncryptDecrypt(t *testing.T) {
	keyPath := "/tmp/test_master.key"
	// Onceki test kalintisini temizle ve yeni key olustur
	os.Remove(keyPath)
	if err := EnsureMasterKey(keyPath); err != nil {
		t.Fatalf("ensure master key: %v", err)
	}

	plain := "hello-aura-123"
	enc, err := Encrypt(plain, keyPath)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	// Encrypted format "enc:" prefix'i ile baslamali
	if len(enc) < 4 || enc[:4] != "enc:" {
		t.Errorf("encrypted text 'enc:' prefix'i ile baslamali, got: %s", enc[:min(len(enc), 20)])
	}

	dec, err := Decrypt(enc, keyPath)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if dec != plain {
		t.Errorf("roundtrip %q != %q", dec, plain)
	}
}
