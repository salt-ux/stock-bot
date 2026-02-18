package config

import "testing"

func TestLoadDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "")
	t.Setenv("APP_PORT", "")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_PORT", "")
	t.Setenv("DB_USER", "")
	t.Setenv("DB_PASSWORD", "")
	t.Setenv("DB_NAME", "")
	t.Setenv("REDIS_HOST", "")
	t.Setenv("REDIS_PORT", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.App.Env != "local" {
		t.Fatalf("expected local env, got %s", cfg.App.Env)
	}
	if cfg.App.Port != 8080 {
		t.Fatalf("expected app port 8080, got %d", cfg.App.Port)
	}
	if cfg.DB.Port != 3306 {
		t.Fatalf("expected db port 3306, got %d", cfg.DB.Port)
	}
	if cfg.Redis.Port != 6379 {
		t.Fatalf("expected redis port 6379, got %d", cfg.Redis.Port)
	}
}

func TestLoadInvalidPortRange(t *testing.T) {
	t.Setenv("APP_PORT", "70000")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for out-of-range APP_PORT")
	}
}
