package cgroups

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Limiter cgroups v2 kaynak limitleri
type Limiter struct{}

// NewLimiter yeni cgroups limiter
func NewLimiter() *Limiter { return &Limiter{} }

// IsAvailable cgroups v2 kullanilabilir mi?
func (l *Limiter) IsAvailable() bool {
	return l.detectVersion() == 2
}

func (l *Limiter) detectVersion() int {
	if _, err := os.Stat("/sys/fs/cgroup/cgroup.controllers"); err == nil {
		return 2
	}
	if _, err := os.Stat("/sys/fs/cgroup/cpu"); err == nil {
		return 1
	}
	return 0
}

// ApplyLimits kullaniciya cgroups limitleri uygular (sudo gerekebilir)
func (l *Limiter) ApplyLimits(username string, cpuShares, memoryMB, nproc int) error {
	if !l.IsAvailable() {
		return fmt.Errorf("cgroups v2 kullanilabilir degil")
	}

	cgPath := filepath.Join("/sys/fs/cgroup", "ospanel-"+username)

	// 1. cgroup olustur
	os.MkdirAll(cgPath, 0755)

	// 2. CPU limiti (cpu.max: $MAX $PERIOD)
	if cpuShares > 0 {
		period := 100000  // 100ms
		quota := cpuShares * period / 1024
		if quota < 1000 {
			quota = 1000
		}
		cpuMax := fmt.Sprintf("%d %d", quota, period)
		os.WriteFile(filepath.Join(cgPath, "cpu.max"), []byte(cpuMax), 0644)
	}

	// 3. RAM limiti (memory.max)
	if memoryMB > 0 {
		memBytes := memoryMB * 1024 * 1024
		os.WriteFile(filepath.Join(cgPath, "memory.max"), []byte(strconv.Itoa(memBytes)), 0644)
		// memory.high = max (soft limit)
		os.WriteFile(filepath.Join(cgPath, "memory.high"), []byte(strconv.Itoa(memBytes)), 0644)
	}

	// 4. Process limiti (pids.max)
	if nproc > 0 {
		os.WriteFile(filepath.Join(cgPath, "pids.max"), []byte(strconv.Itoa(nproc)), 0644)
	}

	// 5. Kullanici process'lerini cgroup'a ekle
	l.classifyUser(username, cgPath)

	return nil
}

// classifyUser kullanici process'lerini cgroup'a ekler
func (l *Limiter) classifyUser(username, cgPath string) {
	// cgroup.procs dosyasina PID'leri yaz
	cgProcs := filepath.Join(cgPath, "cgroup.procs")

	// Mevcut process'leri ekle
	cmd := exec.Command("pgrep", "-u", username)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return
	}
	for _, pid := range strings.Fields(string(out)) {
		os.WriteFile(cgProcs, []byte(pid), 0644)
	}

	// Yeni process'lerin otomatik eklenmesi icin subtree_control
	subtree := filepath.Join(cgPath, "cgroup.subtree_control")
	os.WriteFile(subtree, []byte("+cpu +memory +pids"), 0644)
}

// RemoveLimits kullanici cgroup'unu kaldirir
func (l *Limiter) RemoveLimits(username string) error {
	cgPath := filepath.Join("/sys/fs/cgroup", "ospanel-"+username)
	return os.RemoveAll(cgPath)
}

// GetUsage kullanicinin kaynak kullanimini dondurur
func (l *Limiter) GetUsage(username string) map[string]interface{} {
	usage := map[string]interface{}{
		"cpu_usage":    0,
		"memory_usage": 0,
		"nproc_usage":  0,
	}

	cgPath := filepath.Join("/sys/fs/cgroup", "ospanel-"+username)

	if data, err := os.ReadFile(filepath.Join(cgPath, "cpu.stat")); err == nil {
		usec := parseCPUStat(string(data))
		usage["cpu_usage_usec"] = usec
		if usec > 0 {
			usage["cpu_usage"] = float64(usec) / 1000000.0 // saniye
		}
	}

	if data, err := os.ReadFile(filepath.Join(cgPath, "memory.current")); err == nil {
		bytes, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		usage["memory_usage"] = bytes / (1024 * 1024) // MB
	}

	if data, err := os.ReadFile(filepath.Join(cgPath, "pids.current")); err == nil {
		nproc, _ := strconv.Atoi(strings.TrimSpace(string(data)))
		usage["nproc_usage"] = nproc
	}

	return usage
}

func parseCPUStat(data string) int64 {
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, "usage_usec ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				val, _ := strconv.ParseInt(parts[1], 10, 64)
				return val
			}
		}
	}
	return 0
}
