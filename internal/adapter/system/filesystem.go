package system

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
)

// EnsureDir bir dizinin var olduğundan emin olur, yoksa oluşturur
func EnsureDir(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("dizin oluşturulamadı %s: %w", path, err)
	}
	return nil
}

// CreateDocumentRoot domain için document root oluşturur
func CreateDocumentRoot(homeDir, domain string) (string, error) {
	// Linux'ta: /home/user/public_html/domain.com
	// Windows'ta: C:\Users\user\public_html\domain.com
	publicHTML := filepath.Join(homeDir, "public_html")
	docRoot := filepath.Join(publicHTML, domain)

	if err := EnsureDir(docRoot, 0755); err != nil {
		return "", err
	}

	// index.html oluştur
	indexPath := filepath.Join(docRoot, "index.html")
	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		indexContent := fmt.Sprintf(`<!DOCTYPE html>
<html lang="tr">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - OpenSpeed Panel</title>
    <style>
        body { font-family: system-ui, sans-serif; display: flex; justify-content: center; align-items: center; min-height: 100vh; margin: 0; background: #f5f6fa; }
        .card { background: white; padding: 40px; border-radius: 16px; box-shadow: 0 4px 20px rgba(0,0,0,0.08); text-align: center; }
        h1 { color: #1a1a2e; }
        p { color: #666; }
    </style>
</head>
<body>
    <div class="card">
        <h1>🚀 %s</h1>
        <p>Bu site OpenSpeed Panel ile oluşturuldu.</p>
        <p>Powered by OpenLiteSpeed</p>
    </div>
</body>
</html>`, domain, domain)
		os.WriteFile(indexPath, []byte(indexContent), 0644)
	}

	return docRoot, nil
}

// GetHomeDir kullanıcının home dizinini döndürür
func GetHomeDir(username string) string {
	// Linux'ta /home/username, Windows'ta C:\Users\username
	if runtime.GOOS == "windows" {
		return filepath.Join("C:\\Users", username)
	}

	u, err := user.Lookup(username)
	if err == nil {
		return u.HomeDir
	}
	return filepath.Join("/home", username)
}

// PathExists dosya/dizin var mı kontrol eder
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// DiskUsage dizin için disk kullanımını hesaplar (byte)
func DiskUsage(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// GetSystemStats sistem istatistiklerini döndürür
func GetSystemStats() map[string]interface{} {
	stats := map[string]interface{}{
		"cpu": map[string]interface{}{
			"usage_percent": 0,
			"cores":         runtime.NumCPU(),
		},
		"memory": map[string]interface{}{
			"total_gb":      0,
			"used_gb":       0,
			"usage_percent": 0,
		},
		"disk": map[string]interface{}{
			"total_gb":      0,
			"used_gb":       0,
			"usage_percent": 0,
		},
		"go_version":    runtime.Version(),
		"os":            runtime.GOOS,
		"arch":          runtime.GOARCH,
		"goroutines":    runtime.NumGoroutine(),
	}
	return stats
}
