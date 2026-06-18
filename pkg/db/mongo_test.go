package db

import (
	"context"
	"testing"
	"time"

	"github.com/innacy/finance-agent/pkg/config"
)

func TestNewClientReturnsClient(t *testing.T) {
	cfg := &config.DBConfig{
		URI:      "mongodb://localhost:27017",
		Database: "test-finance-agent",
		Timeout:  5 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient should not error with valid config: %v", err)
	}
	if client == nil {
		t.Fatal("NewClient returned nil")
	}
	if client.DatabaseName() != "test-finance-agent" {
		t.Errorf("expected database name 'test-finance-agent', got %q", client.DatabaseName())
	}
}

func TestNewClientRejectsEmptyURI(t *testing.T) {
	cfg := &config.DBConfig{
		URI:      "",
		Database: "test",
		Timeout:  5 * time.Second,
	}

	_, err := NewClient(cfg)
	if err == nil {
		t.Error("expected error for empty URI")
	}
}

func TestPingFailsWithBadURI(t *testing.T) {
	cfg := &config.DBConfig{
		URI:      "mongodb://192.0.2.1:27017",
		Database: "test",
		Timeout:  1 * time.Second,
	}

	client, err := NewClient(cfg)
	if err != nil {
		t.Fatalf("NewClient should not error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err = client.Ping(ctx)
	if err == nil {
		t.Error("Ping should fail with unreachable host")
	}
}

func TestDisconnectWithoutConnect(t *testing.T) {
	cfg := &config.DBConfig{
		URI:      "mongodb://localhost:27017",
		Database: "test",
		Timeout:  5 * time.Second,
	}

	client, _ := NewClient(cfg)
	err := client.Disconnect(context.Background())
	if err != nil {
		t.Errorf("Disconnect should not error: %v", err)
	}
}
