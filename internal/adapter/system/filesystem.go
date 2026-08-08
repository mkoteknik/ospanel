package system

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	cpuMu       sync.Mutex
	cpuPrevIdle uint64
	cpuPrevTotal uint64
	cpuPrevTime time.Time
	cpuCache    float64
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

	// .htaccess güvenlik dosyası oluştur (OLS Apache uyumlu)
	htaccessPath := filepath.Join(docRoot, ".htaccess")
	if _, err := os.Stat(htaccessPath); os.IsNotExist(err) {
		os.WriteFile(htaccessPath, []byte(GenerateHtaccess()), 0644)
	}

	return docRoot, nil
}

// GenerateHtaccess domain document root için güvenlik .htaccess içeriği
func GenerateHtaccess() string {
	return `# OpenSpeed Panel - Otomatik Guvenlik .htaccess
# OLS tarafindan Apache uyumlu olarak islenir

# === Dizin listelemeyi kapat ===
Options -Indexes

# === Hassas dosyalari engelle ===
<FilesMatch "\.(htaccess|htpasswd|ini|phps|log|sh|bak|sql|tar|gz|zip)$">
    Require all denied
</FilesMatch>

# === WordPress ozel: wp-config.php, xmlrpc.php koruma ===
<FilesMatch "^(wp-config\.php|xmlrpc\.php|wp-login\.php)$">
    Require all granted
</FilesMatch>

# === WordPress xmlrpc saldirilarini sinirla ===
<IfModule mod_rewrite.c>
    RewriteEngine On
    RewriteRule ^xmlrpc\.php$ - [F,L]
</IfModule>

# === PHP guvenlik (Apache mod_php / OLS LSAPI) ===
<IfModule php_module>
    php_value open_basedir "%{DOCUMENT_ROOT}:/tmp:/dev/urandom"
    php_value upload_max_filesize 64M
    php_value post_max_size 80M
    php_value max_execution_time 30
    php_value max_input_time 60
    php_value memory_limit 256M
    php_flag display_errors Off
    php_flag allow_url_include Off
    php_flag session.cookie_httponly On
    php_flag session.cookie_secure On
</IfModule>

# === Guvenlik header'lari ===
<IfModule mod_headers.c>
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set Referrer-Policy "strict-origin-when-cross-origin"
    Header always set Permissions-Policy "camera=(), microphone=(), geolocation=()"
    Header set X-XSS-Protection "1; mode=block"
</IfModule>

# === Statik dosya cache ===
<IfModule mod_expires.c>
    ExpiresActive On
    ExpiresByType image/jpg "access plus 1 year"
    ExpiresByType image/jpeg "access plus 1 year"
    ExpiresByType image/gif "access plus 1 year"
    ExpiresByType image/png "access plus 1 year"
    ExpiresByType image/svg+xml "access plus 1 year"
    ExpiresByType image/webp "access plus 1 year"
    ExpiresByType text/css "access plus 1 month"
    ExpiresByType text/javascript "access plus 1 month"
    ExpiresByType application/javascript "access plus 1 month"
    ExpiresByType application/x-font-ttf "access plus 1 year"
    ExpiresByType application/x-font-woff "access plus 1 year"
    ExpiresByType font/woff2 "access plus 1 year"
</IfModule>

# === Gzip sikistirma ===
<IfModule mod_deflate.c>
    AddOutputFilterByType DEFLATE text/html text/plain text/css text/javascript
    AddOutputFilterByType DEFLATE application/javascript application/json application/xml
    AddOutputFilterByType DEFLATE image/svg+xml
</IfModule>

# === WordPress SEO URL (varsa) ===
<IfModule mod_rewrite.c>
    RewriteEngine On
    RewriteRule ^index\.php$ - [L]
    RewriteCond %{REQUEST_FILENAME} !-f
    RewriteCond %{REQUEST_FILENAME} !-d
    RewriteRule . /index.php [L]
</IfModule>
`
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

// GetSystemStats sistem istatistiklerini döndürür (gerçek Linux /proc)
func GetSystemStats() map[string]interface{} {
	cpuUsage := getCPUUsage()
	memTotal, memUsed, memPercent := getMemoryInfo()
	diskTotal, diskUsed, diskPercent := getDiskInfo("/")

	// Uptime
	uptime, _ := getUptime()

	// Load average
	load1, load5, load15 := getLoadAvg()

	// Network
	rxBytes, txBytes := getNetworkTraffic()

	return map[string]interface{}{
		"cpu": map[string]interface{}{
			"usage_percent": cpuUsage,
			"cores":          runtime.NumCPU(),
		},
		"memory": map[string]interface{}{
			"total_gb":      roundGB(memTotal),
			"used_gb":       roundGB(memUsed),
			"free_gb":       roundGB(memTotal - memUsed),
			"usage_percent": memPercent,
		},
		"disk": map[string]interface{}{
			"total_gb":      roundGB(diskTotal),
			"used_gb":       roundGB(diskUsed),
			"free_gb":       roundGB(diskTotal - diskUsed),
			"usage_percent": diskPercent,
		},
		"load": map[string]interface{}{
			"load1":  load1,
			"load5":  load5,
			"load15": load15,
		},
		"network": map[string]interface{}{
			"rx_bytes": rxBytes,
			"tx_bytes": txBytes,
			"rx_mb":    roundGB(rxBytes),
			"tx_mb":    roundGB(txBytes),
		},
		"uptime_seconds": uptime,
		"go_version":     runtime.Version(),
		"os":             runtime.GOOS,
		"arch":           runtime.GOARCH,
		"goroutines":     runtime.NumGoroutine(),
	}
}

// getCPUUsage /proc/stat'tan CPU kullanım yüzdesini okur — non-blocking, cache'li
func getCPUUsage() float64 {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "cpu ") {
			fields := strings.Fields(line)
			if len(fields) < 8 {
				return 0
			}
			user, _ := strconv.ParseUint(fields[1], 10, 64)
			nice, _ := strconv.ParseUint(fields[2], 10, 64)
			system, _ := strconv.ParseUint(fields[3], 10, 64)
			idle, _ := strconv.ParseUint(fields[4], 10, 64)
			iowait, _ := strconv.ParseUint(fields[5], 10, 64)
			irq, _ := strconv.ParseUint(fields[6], 10, 64)
			softirq, _ := strconv.ParseUint(fields[7], 10, 64)
			steal := uint64(0)
			if len(fields) > 8 {
				steal, _ = strconv.ParseUint(fields[8], 10, 64)
			}
			idleTotal := idle + iowait
			nonIdle := user + nice + system + irq + softirq + steal
			total := idleTotal + nonIdle

			cpuMu.Lock()
			defer cpuMu.Unlock()

			// İlk çağrı — cache yok, 0 dön ve state sakla
			if cpuPrevTime.IsZero() {
				cpuPrevIdle = idleTotal
				cpuPrevTotal = total
				cpuPrevTime = time.Now()
				return cpuCache
			}

			// 500ms'den sık çağrılırsa cache dön (throttle)
			if time.Since(cpuPrevTime) < 500*time.Millisecond {
				return cpuCache
			}

			totalDelta := total - cpuPrevTotal
			idleDelta := idleTotal - cpuPrevIdle

			cpuPrevIdle = idleTotal
			cpuPrevTotal = total
			cpuPrevTime = time.Now()

			if totalDelta == 0 {
				return cpuCache
			}
			usage := math.Round((1.0-float64(idleDelta)/float64(totalDelta))*10000) / 100
			// Clamp 0-100
			if usage < 0 {
				usage = 0
			}
			if usage > 100 {
				usage = 100
			}
			cpuCache = usage
			return usage
		}
	}
	return 0
}

// getMemoryInfo /proc/meminfo'dan bellek bilgisi okur (bytes)
func getMemoryInfo() (total, used uint64, percent float64) {
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, 0
	}
	defer f.Close()

	var memTotal, memFree, buffers, cached uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		val, _ := strconv.ParseUint(fields[1], 10, 64)
		switch fields[0] {
		case "MemTotal:":
			memTotal = val * 1024 // kB -> bytes
		case "MemFree:":
			memFree = val * 1024
		case "Buffers:":
			buffers = val * 1024
		case "Cached:":
			cached = val * 1024
		}
	}

	total = memTotal
	used = memTotal - memFree - buffers - cached
	if total > 0 {
		percent = math.Round(float64(used)/float64(total)*10000) / 100
	}
	return
}

// getDiskInfo disk kullanım bilgisi (bytes)
func getDiskInfo(path string) (total, used uint64, percent float64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, 0
	}

	total = stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used = total - available

	if total > 0 {
		percent = math.Round(float64(used)/float64(total)*10000) / 100
	}
	return
}

// getUptime /proc/uptime'dan sistem uptime'ını okur
func getUptime() (float64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, fmt.Errorf("geçersiz uptime")
	}
	return strconv.ParseFloat(fields[0], 64)
}

// getLoadAvg /proc/loadavg'dan yük ortalamasını okur
func getLoadAvg() (load1, load5, load15 float64) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return 0, 0, 0
	}
	fields := strings.Fields(string(data))
	if len(fields) >= 3 {
		load1, _ = strconv.ParseFloat(fields[0], 64)
		load5, _ = strconv.ParseFloat(fields[1], 64)
		load15, _ = strconv.ParseFloat(fields[2], 64)
	}
	return
}

// getNetworkTraffic /proc/net/dev'dan ağ trafiğini okur
func getNetworkTraffic() (rxBytes, txBytes uint64) {
	f, err := os.Open("/proc/net/dev")
	if err != nil {
		return 0, 0
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// eth0, ens, enp, wlan gibi fiziksel arayüzleri topla, lo'yu atla
		if strings.Contains(line, "lo:") || !strings.Contains(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "Inter") || strings.HasPrefix(line, "face") {
			continue
		}

		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}

		fields := strings.Fields(line[idx+1:])
		if len(fields) < 10 {
			continue
		}

		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		rxBytes += rx
		txBytes += tx
	}
	return
}

// roundGB byte değerini GB cinsine yuvarlar (2 decimal)
func roundGB(bytes uint64) float64 {
	gb := float64(bytes) / (1024 * 1024 * 1024)
	return math.Round(gb*100) / 100
}
