package agentdserver

import (
	"context"
	"testing"
	"time"

	"agentd/internal/agentdserver/config"
)

func TestNewWithConfigStartStop(t *testing.T) {
	t.Parallel()

	cfg := testConfig(t)
	server, err := NewWithConfig(cfg)
	if err != nil {
		t.Fatalf("NewWithConfig: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := server.Stop(ctx); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}

func TestNewWithConfigRejectsNilConfig(t *testing.T) {
	t.Parallel()

	_, err := NewWithConfig(nil)
	if err == nil {
		t.Fatal("NewWithConfig returned nil error")
	}
}

func testConfig(t *testing.T) *config.Config {
	t.Helper()

	dir := t.TempDir()

	return &config.Config{
		Server: config.ServerConfig{
			Host:         "127.0.0.1",
			Port:         "0",
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
		},
		Storage: config.StorageConfig{
			DataDir:        dir,
			SettingsDBPath: dir + "/settings.db",
			RuntimeDBDir:   dir + "/agents",
			RunLogDir:      dir + "/logs",
			SQLiteMaxConns: 1,
		},
		Runtime: config.RuntimeConfig{
			StartupTimeout:  time.Second,
			ShutdownTimeout: time.Second,
			RunStopTimeout:  time.Second,
		},
	}
}
