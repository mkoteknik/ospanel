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
	"syscall"
	"time"
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

// getCPUUsage /proc/stat'tan CPU kullanım yüzdesini okur
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
			// cpu user nice system idle iowait irq softirq steal
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

			if total == 0 {
				return 0
			}

			// Kısa bir bekleme ve ikinci okuma ile anlık kullanım hesapla
			time.Sleep(100 * time.Millisecond)

			// Dosyayı başa sar ve ikinci okumayı yap
			f.Seek(0, 0)
			scanner2 := bufio.NewScanner(f)
			for scanner2.Scan() {
				line2 := scanner2.Text()
				if strings.HasPrefix(line2, "cpu ") {
					f2 := strings.Fields(line2)
					if len(f2) < 8 {
						break
					}
					u2, _ := strconv.ParseUint(f2[1], 10, 64)
					n2, _ := strconv.ParseUint(f2[2], 10, 64)
					s2, _ := strconv.ParseUint(f2[3], 10, 64)
					id2, _ := strconv.ParseUint(f2[4], 10, 64)
					iw2, _ := strconv.ParseUint(f2[5], 10, 64)
					ir2, _ := strconv.ParseUint(f2[6], 10, 64)
					si2, _ := strconv.ParseUint(f2[7], 10, 64)
					st2 := uint64(0)
					if len(f2) > 8 {
						st2, _ = strconv.ParseUint(f2[8], 10, 64)
					}

					idle2 := id2 + iw2
					nonIdle2 := u2 + n2 + s2 + ir2 + si2 + st2
					total2 := idle2 + nonIdle2

					totalDelta := total2 - total
					idleDelta := idle2 - idleTotal

					if totalDelta == 0 {
						return 0
					}
					return math.Round((1.0-float64(idleDelta)/float64(totalDelta))*10000) / 100
				}
			}
			break
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
