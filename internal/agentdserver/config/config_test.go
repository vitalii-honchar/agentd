package config

import (
	"testing"
	"time"
)

func TestFromLookupDefaults(t *testing.T) {
	cfg, err := FromLookup(emptyLookup)
	if err != nil {
		t.Fatalf("from lookup: %v", err)
	}

	if cfg.Server.Address() != "127.0.0.1:18080" {
		t.Fatalf("unexpected address: %s", cfg.Server.Address())
	}
	if cfg.Storage.SettingsDBPath != DefaultSettingsDBPath {
		t.Fatalf("unexpected settings db path: %s", cfg.Storage.SettingsDBPath)
	}
	if cfg.Storage.SQLiteMaxConns != DefaultSQLiteMaxConns {
		t.Fatalf("unexpected sqlite max conns: %d", cfg.Storage.SQLiteMaxConns)
	}
	if cfg.Runtime.StartupTimeout != DefaultStartupTimeout {
		t.Fatalf("unexpected startup timeout: %s", cfg.Runtime.StartupTimeout)
	}
	if cfg.Codex.Command != DefaultCodexCommand {
		t.Fatalf("unexpected codex command: %s", cfg.Codex.Command)
	}
	if cfg.Codex.Timeout != DefaultCodexTimeout {
		t.Fatalf("unexpected codex timeout: %s", cfg.Codex.Timeout)
	}
}

func TestFromLookupOverrides(t *testing.T) {
	values := map[string]string{
		"AGENTD_PRODUCTION":         "true",
		"AGENTD_SERVER_HOST":        "0.0.0.0",
		"AGENTD_SERVER_PORT":        "19090",
		"AGENTD_DATA_DIR":           "/tmp/agentd",
		"AGENTD_SETTINGS_DB_PATH":   "/tmp/agentd/settings.db",
		"AGENTD_RUNTIME_DB_DIR":     "/tmp/agentd/agents",
		"AGENTD_RUN_LOG_DIR":        "/tmp/agentd/logs",
		"AGENTD_SQLITE_MAX_CONNS":   "8",
		"AGENTD_STARTUP_TIMEOUT":    "5s",
		"AGENTD_SHUTDOWN_TIMEOUT":   "6s",
		"AGENTD_RUN_STOP_TIMEOUT":   "7s",
		"AGENTD_HTTP_READ_TIMEOUT":  "8s",
		"AGENTD_HTTP_WRITE_TIMEOUT": "9s",
		"AGENTD_CODEX_COMMAND":      "/opt/bin/codex",
		"AGENTD_CODEX_MODEL":        "gpt-5.4-mini",
		"AGENTD_CODEX_PROFILE":      "agentd",
		"AGENTD_CODEX_TIMEOUT":      "2m",
		"OPENAI_API_KEY":            "test-key",
	}

	cfg, err := FromLookup(mapLookup(values))
	if err != nil {
		t.Fatalf("from lookup: %v", err)
	}

	if !cfg.Production {
		t.Fatalf("expected production mode")
	}
	if cfg.Server.Address() != "0.0.0.0:19090" {
		t.Fatalf("unexpected address: %s", cfg.Server.Address())
	}
	if cfg.Storage.SQLiteMaxConns != 8 {
		t.Fatalf("unexpected sqlite max conns: %d", cfg.Storage.SQLiteMaxConns)
	}
	if cfg.Runtime.RunStopTimeout != 7*time.Second {
		t.Fatalf("unexpected run stop timeout: %s", cfg.Runtime.RunStopTimeout)
	}
	if cfg.OpenAI.APIKey != "test-key" {
		t.Fatalf("unexpected OpenAI API key")
	}
	if cfg.Codex.Command != "/opt/bin/codex" {
		t.Fatalf("unexpected codex command: %s", cfg.Codex.Command)
	}
	if cfg.Codex.Model != "gpt-5.4-mini" || cfg.Codex.Profile != "agentd" {
		t.Fatalf("unexpected codex model/profile: %#v", cfg.Codex)
	}
	if cfg.Codex.Timeout != 2*time.Minute {
		t.Fatalf("unexpected codex timeout: %s", cfg.Codex.Timeout)
	}
}

func TestFromLookupInvalidNumbersFallBack(t *testing.T) {
	values := map[string]string{
		"AGENTD_SQLITE_MAX_CONNS":  "bad",
		"AGENTD_STARTUP_TIMEOUT":   "bad",
		"AGENTD_SHUTDOWN_TIMEOUT":  "bad",
		"AGENTD_RUN_STOP_TIMEOUT":  "bad",
		"AGENTD_HTTP_READ_TIMEOUT": "bad",
		"AGENTD_CODEX_TIMEOUT":     "bad",
	}

	cfg, err := FromLookup(mapLookup(values))
	if err != nil {
		t.Fatalf("from lookup: %v", err)
	}

	if cfg.Storage.SQLiteMaxConns != DefaultSQLiteMaxConns {
		t.Fatalf("unexpected sqlite max conns: %d", cfg.Storage.SQLiteMaxConns)
	}
	if cfg.Runtime.StartupTimeout != DefaultStartupTimeout {
		t.Fatalf("unexpected startup timeout: %s", cfg.Runtime.StartupTimeout)
	}
	if cfg.Codex.Timeout != DefaultCodexTimeout {
		t.Fatalf("unexpected codex timeout: %s", cfg.Codex.Timeout)
	}
}

func TestValidateRejectsMissingRequiredFields(t *testing.T) {
	cfg, err := FromLookup(emptyLookup)
	if err != nil {
		t.Fatalf("from lookup: %v", err)
	}
	cfg.Storage.RuntimeDBDir = ""

	if err := cfg.Validate(); err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestValidateRejectsInvalidCodexConfig(t *testing.T) {
	t.Parallel()

	cfg, err := FromLookup(emptyLookup)
	if err != nil {
		t.Fatalf("from lookup: %v", err)
	}
	cfg.Codex.Command = ""
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected missing codex command validation error")
	}

	cfg, err = FromLookup(emptyLookup)
	if err != nil {
		t.Fatalf("from lookup: %v", err)
	}
	cfg.Codex.Timeout = 0
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected invalid codex timeout validation error")
	}
}

func emptyLookup(string) (string, bool) {
	return "", false
}

func mapLookup(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		value, ok := values[key]

		return value, ok
	}
}
