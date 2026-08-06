package cache

import (
	"fmt"
	"os/exec"
	"strings"
)

// RedisClient Redis yönetim istemcisi
type RedisClient struct {
	installed bool
	socket    string
	cli       string
}

// NewRedisClient yeni Redis client oluşturur
func NewRedisClient() *RedisClient {
	rc := &RedisClient{
		socket: "/var/run/redis/redis-server.sock",
		cli:    "redis-cli",
	}

	// Redis kurulu mu?
	if path, err := exec.LookPath("redis-cli"); err == nil {
		rc.cli = path
		rc.installed = true
	}

	return rc
}

// IsAvailable Redis kullanılabilir mi?
func (r *RedisClient) IsAvailable() bool {
	return r.installed
}

// GetInfo Redis INFO çıktısını döndürür
func (r *RedisClient) GetInfo() map[string]interface{} {
	if !r.installed {
		return map[string]interface{}{"installed": false}
	}

	out, err := exec.Command(r.cli, "INFO").CombinedOutput()
	if err != nil {
		return map[string]interface{}{"installed": true, "error": err.Error()}
	}

	info := map[string]interface{}{"installed": true}
	section := "server"
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "#") {
			section = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "#")))
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			info[section+"."+parts[0]] = strings.TrimSpace(parts[1])
		}
	}

	// Özet metrikler
	info["summary"] = map[string]string{
		"version":        fmt.Sprint(info["server.redis_version"]),
		"uptime_days":    fmt.Sprint(info["server.uptime_in_days"]),
		"used_memory":    fmt.Sprint(info["memory.used_memory_human"]),
		"connected":      fmt.Sprint(info["clients.connected_clients"]),
		"total_keys":     fmt.Sprint(info["keyspace.total_keys"]),
		"hit_rate":       fmt.Sprint(info["stats.instantaneous_ops_per_sec"]),
	}

	return info
}

// GetStats özet Redis istatistikleri
func (r *RedisClient) GetStats() map[string]interface{} {
	info := r.GetInfo()
	summary, _ := info["summary"].(map[string]string)
	if summary == nil {
		return map[string]interface{}{"installed": r.installed}
	}
	return map[string]interface{}{
		"installed":    r.installed,
		"version":      summary["version"],
		"uptime_days":  summary["uptime_days"],
		"used_memory":  summary["used_memory"],
		"connected":    summary["connected"],
		"total_keys":   summary["total_keys"],
		"ops_per_sec":  summary["hit_rate"],
	}
}

// FlushCache tüm cache'i temizler
func (r *RedisClient) FlushCache() error {
	if !r.installed {
		return fmt.Errorf("Redis kurulu değil")
	}
	return exec.Command(r.cli, "FLUSHDB").Run()
}

// FlushAll tüm veritabanlarını temizler
func (r *RedisClient) FlushAll() error {
	if !r.installed {
		return fmt.Errorf("Redis kurulu değil")
	}
	return exec.Command(r.cli, "FLUSHALL").Run()
}

// MemoryUsage bellek kullanımını döndürür (byte)
func (r *RedisClient) MemoryUsage() (int64, error) {
	if !r.installed {
		return 0, fmt.Errorf("Redis kurulu değil")
	}
	out, err := exec.Command(r.cli, "INFO", "memory").CombinedOutput()
	if err != nil {
		return 0, err
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "used_memory:") {
			var bytes int64
			fmt.Sscanf(line, "used_memory:%d", &bytes)
			return bytes, nil
		}
	}
	return 0, nil
}
