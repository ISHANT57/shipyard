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
			cfg:     Config{Addr: ":8080", ShutdownTimeout: time.Second},
			wantErr: false,
		},
		{
			name:    "empty addr",
			cfg:     Config{Addr: "", ShutdownTimeout: time.Second},
			wantErr: true,
		},
		{
			name:    "zero shutdown timeout",
			cfg:     Config{Addr: ":8080", ShutdownTimeout: 0},
			wantErr: true,
		},
		{
			name:    "negative shutdown timeout",
			cfg:     Config{Addr: ":8080", ShutdownTimeout: -time.Second},
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
