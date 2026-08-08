package atomicfile

import (
	"fmt"
	"os"
	"path/filepath"
)

// WriteFileAtomic dosyayı atomik olarak yazar: tmp dosya + fsync + rename.
// Üzerine yazılacak dosya varsa önce .bak.<ts> yedeği alınır (opsiyonel).
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("dizin oluşturulamadı: %w", err)
	}

	// Backup
	if _, err := os.Stat(path); err == nil {
		bak := path + ".bak"
		if b, err := os.ReadFile(path); err == nil {
			_ = os.WriteFile(bak, b, perm)
		}
	}

	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return fmt.Errorf("tmp oluşturulamadı: %w", err)
	}
	tmpName := tmp.Name()
	// Cleanup on failure
	defer func() {
		tmp.Close()
		os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		return fmt.Errorf("tmp yazılamadı: %w", err)
	}
	if err := tmp.Chmod(perm); err != nil {
		return fmt.Errorf("chmod hatası: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("fsync hatası: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("tmp kapatılamadı: %w", err)
	}

	// fsync parent dir
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename hatası: %w", err)
	}
	// Dir fsync (best effort)
	if d, err := os.Open(dir); err == nil {
		_ = d.Sync()
		d.Close()
	}
	return nil
}
