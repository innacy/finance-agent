package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load with no file should use defaults: %v", err)
	}

	if cfg.DB.URI != "mongodb://localhost:27017" {
		t.Errorf("expected default DB URI, got %q", cfg.DB.URI)
	}
	if cfg.DB.Database != "finance-agent" {
		t.Errorf("expected default DB name, got %q", cfg.DB.Database)
	}
	if cfg.DB.Timeout != 10*time.Second {
		t.Errorf("expected 10s timeout, got %v", cfg.DB.Timeout)
	}
	if cfg.CLI.CurrencySymbol != "₹" {
		t.Errorf("expected ₹ currency symbol, got %q", cfg.CLI.CurrencySymbol)
	}
	if cfg.CLI.DateFormat != "02 Jan 2006" {
		t.Errorf("expected date format '02 Jan 2006', got %q", cfg.CLI.DateFormat)
	}
	if cfg.AI.Provider != "nvidia" {
		t.Errorf("expected nvidia provider, got %q", cfg.AI.Provider)
	}
	if cfg.Daemon.PollInterval != 5*time.Minute {
		t.Errorf("expected 5m poll interval, got %v", cfg.Daemon.PollInterval)
	}
	if cfg.Categories.MinConfidence != 0.8 {
		t.Errorf("expected min_confidence 0.8, got %f", cfg.Categories.MinConfidence)
	}
	if cfg.Categories.AIThreshold != 0.6 {
		t.Errorf("expected ai_threshold 0.6, got %f", cfg.Categories.AIThreshold)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := filepath.Join(dir, "config.yaml")

	content := []byte(`
db:
  uri: "mongodb://custom:27017"
  database: "test-db"
  timeout: 5s
cli:
  currency_symbol: "$"
  theme: "dark"
`)
	if err := os.WriteFile(cfgFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(cfgFile)
	if err != nil {
		t.Fatalf("Load from file failed: %v", err)
	}

	if cfg.DB.URI != "mongodb://custom:27017" {
		t.Errorf("expected custom URI, got %q", cfg.DB.URI)
	}
	if cfg.DB.Database != "test-db" {
		t.Errorf("expected test-db, got %q", cfg.DB.Database)
	}
	if cfg.DB.Timeout != 5*time.Second {
		t.Errorf("expected 5s timeout, got %v", cfg.DB.Timeout)
	}
	if cfg.CLI.CurrencySymbol != "$" {
		t.Errorf("expected $ symbol, got %q", cfg.CLI.CurrencySymbol)
	}
	if cfg.CLI.Theme != "dark" {
		t.Errorf("expected dark theme, got %q", cfg.CLI.Theme)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("FINANCE_AGENT_DB_URI", "mongodb://env-host:27017")
	t.Setenv("FINANCE_AGENT_AI_PROVIDER", "ollama")

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load with env vars failed: %v", err)
	}

	if cfg.DB.URI != "mongodb://env-host:27017" {
		t.Errorf("expected env URI, got %q", cfg.DB.URI)
	}
	if cfg.AI.Provider != "ollama" {
		t.Errorf("expected ollama provider, got %q", cfg.AI.Provider)
	}
}

func TestValidateRequiresDBURI(t *testing.T) {
	cfg := &Config{
		DB: DBConfig{URI: "", Database: "test"},
	}
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for empty DB URI")
	}
}

func TestValidateRequiresDBName(t *testing.T) {
	cfg := &Config{
		DB: DBConfig{URI: "mongodb://localhost:27017", Database: ""},
	}
	err := Validate(cfg)
	if err == nil {
		t.Error("expected validation error for empty DB name")
	}
}

func TestValidatePassesWithDefaults(t *testing.T) {
	cfg, _ := Load("")
	err := Validate(cfg)
	if err != nil {
		t.Errorf("default config should validate: %v", err)
	}
}
