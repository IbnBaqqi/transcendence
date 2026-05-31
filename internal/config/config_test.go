package config

import (
	"os"
	"testing"
)

func TestLoad(t *testing.T) {
	os.Setenv("ENV", "test")
	os.Setenv("PORT", "9090")
	os.Setenv("DB_URL", "test://localhost")
	defer func() {
		os.Unsetenv("ENV")
		os.Unsetenv("PORT")
		os.Unsetenv("DB_URL")
		}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if cfg.Env != "test" {
		t.Errorf("expected env 'test', got '%s'", cfg.Env)
	}

	if cfg.Server.Port != "9090" {
		t.Errorf("expected port '9090', got '%s'", cfg.Server.Port)
	}
}
