package exec

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// Runner komut çalıştırma yardımcısı
type Runner struct {
	timeout time.Duration
}

// NewRunner yeni bir Runner oluşturur
func NewRunner(timeout time.Duration) *Runner {
	return &Runner{timeout: timeout}
}

// Result komut çalıştırma sonucu
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Duration time.Duration
}

// Run bir komut çalıştırır
func (r *Runner) Run(ctx context.Context, command string, args ...string) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)

	start := time.Now()

	stdout, err := cmd.Output()

	duration := time.Since(start)

	result := &Result{
		Stdout:   strings.TrimSpace(string(stdout)),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Stderr = strings.TrimSpace(string(exitErr.Stderr))
			return result, fmt.Errorf("komut başarısız: %s %s: %w", command, strings.Join(args, " "), err)
		}
		return result, fmt.Errorf("komut çalıştırılamadı: %s %s: %w", command, strings.Join(args, " "), err)
	}

	return result, nil
}

// RunWithStdin stdin ile komut çalıştırır
func (r *Runner) RunWithStdin(ctx context.Context, stdin string, command string, args ...string) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Stdin = strings.NewReader(stdin)

	start := time.Now()
	stdout, err := cmd.Output()
	duration := time.Since(start)

	result := &Result{
		Stdout:   strings.TrimSpace(string(stdout)),
		Duration: duration,
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Stderr = strings.TrimSpace(string(exitErr.Stderr))
			return result, fmt.Errorf("komut başarısız: %w", err)
		}
		return result, fmt.Errorf("komut çalıştırılamadı: %w", err)
	}

	return result, nil
}

// DefaultRunner varsayılan 30 saniye timeout'lu runner
var DefaultRunner = NewRunner(30 * time.Second)
