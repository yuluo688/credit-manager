package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultStaleReservationTimeout(t *testing.T) {
	if got := Default().Stream.StaleReservationTimeout; got != 2*time.Hour {
		t.Fatalf("default stale timeout = %s, want 2h", got)
	}
}

func TestStaleReservationTimeoutValidation(t *testing.T) {
	for _, timeout := range []time.Duration{time.Minute, 2 * time.Hour, 24 * time.Hour} {
		cfg := Default()
		cfg.DataDir = t.TempDir()
		cfg.Stream.StaleReservationTimeout = timeout
		if err := cfg.Validate(); err != nil {
			t.Fatalf("timeout %s should be valid: %v", timeout, err)
		}
	}
	for _, timeout := range []time.Duration{0, 30 * time.Second, 25 * time.Hour} {
		cfg := Default()
		cfg.DataDir = t.TempDir()
		cfg.Stream.StaleReservationTimeout = timeout
		if err := cfg.Validate(); err == nil {
			t.Fatalf("timeout %s should be invalid", timeout)
		}
	}
}

func TestExampleConfigParses(t *testing.T) {
	path := filepath.Join("..", "..", "config.example.yaml")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read example config: %v", err)
	}
	cfg, err := parseYAML(raw, true)
	if err != nil {
		t.Fatalf("parse example config: %v", err)
	}
	if cfg.Stream.StaleReservationTimeout != 2*time.Hour {
		t.Fatalf("example stale timeout = %s, want 2h", cfg.Stream.StaleReservationTimeout)
	}
}
