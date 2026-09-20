package config

import (
	"testing"
	"time"
)

func TestLoad_Defaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":8080")
	}
	if cfg.ShutdownTimeout != 10*time.Second {
		t.Errorf("ShutdownTimeout = %s, want 10s", cfg.ShutdownTimeout)
	}
	if cfg.DatabaseURL == "" {
		t.Error("DatabaseURL default is empty, want a usable local connection string")
	}
}

func TestLoad_AddrFromEnv(t *testing.T) {
	t.Setenv("SHIPYARD_API_ADDR", ":9090")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned unexpected error: %v", err)
	}
	if cfg.Addr != ":9090" {
		t.Errorf("Addr = %q, want %q", cfg.Addr, ":9090")
	}
}

// Table-driven test: each case is one row, run as its own subtest via
// t.Run so a failure names exactly which case failed.
func TestValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "valid config",
			cfg:     Config{Addr: ":8080", ShutdownTimeout: time.Second, DatabaseURL: "postgres://x"},
			wantErr: false,
		},
		{
			name:    "empty addr",
			cfg:     Config{Addr: "", ShutdownTimeout: time.Second, DatabaseURL: "postgres://x"},
			wantErr: true,
		},
		{
			name:    "zero shutdown timeout",
			cfg:     Config{Addr: ":8080", ShutdownTimeout: 0, DatabaseURL: "postgres://x"},
			wantErr: true,
		},
		{
			name:    "negative shutdown timeout",
			cfg:     Config{Addr: ":8080", ShutdownTimeout: -time.Second, DatabaseURL: "postgres://x"},
			wantErr: true,
		},
		{
			name:    "empty database url",
			cfg:     Config{Addr: ":8080", ShutdownTimeout: time.Second, DatabaseURL: ""},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

func TestLoadWorker_Defaults(t *testing.T) {
	cfg, err := LoadWorker()
	if err != nil {
		t.Fatalf("LoadWorker() returned unexpected error: %v", err)
	}
	if cfg.Concurrency != 4 {
		t.Errorf("Concurrency = %d, want 4", cfg.Concurrency)
	}
	if cfg.ID == "" {
		t.Error("ID default is empty, want <hostname>-<pid>")
	}
	if cfg.Lease != 30*time.Second || cfg.PollInterval != 2*time.Second {
		t.Errorf("Lease/PollInterval = %s/%s, want 30s/2s", cfg.Lease, cfg.PollInterval)
	}
}

func TestLoadWorker_FromEnv(t *testing.T) {
	t.Setenv("SHIPYARD_WORKER_CONCURRENCY", "9")
	t.Setenv("SHIPYARD_WORKER_LEASE", "750ms")
	t.Setenv("SHIPYARD_WORKER_ID", "w-test")

	cfg, err := LoadWorker()
	if err != nil {
		t.Fatalf("LoadWorker() returned unexpected error: %v", err)
	}
	if cfg.Concurrency != 9 || cfg.Lease != 750*time.Millisecond || cfg.ID != "w-test" {
		t.Errorf("got %+v, want concurrency 9, lease 750ms, id w-test", cfg)
	}
}

func TestLoadWorker_RejectsBadValues(t *testing.T) {
	cases := []struct {
		name, key, value string
	}{
		{"non-numeric concurrency", "SHIPYARD_WORKER_CONCURRENCY", "many"},
		{"zero concurrency", "SHIPYARD_WORKER_CONCURRENCY", "0"},
		{"unparseable duration", "SHIPYARD_WORKER_LEASE", "soon"},
		{"negative duration", "SHIPYARD_WORKER_DRAIN_TIMEOUT", "-5s"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(tc.key, tc.value)
			if _, err := LoadWorker(); err == nil {
				t.Errorf("LoadWorker() with %s=%q returned nil error, want a validation error", tc.key, tc.value)
			}
		})
	}
}
