package db

import (
	"context"
	"testing"
)

func TestEmbeddedSettingsMigrations(t *testing.T) {
	t.Parallel()

	database, err := New("settings", Config{
		Path:         t.TempDir() + "/settings.db",
		MaxOpenConns: 1,
		Pragmas:      PragmasSettings,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer stopDB(t, database)

	if err := database.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	assertObjectExists(t, database, "table", "agents")
	assertObjectExists(t, database, "table", "agent_tools")
	assertObjectExists(t, database, "table", "agent_mcp_servers")
	assertObjectExists(t, database, "index", "idx_agents_enabled_schedule")
	assertObjectExists(t, database, "index", "idx_agent_tools_kind")
	assertObjectExists(t, database, "index", "idx_agents_vendor")
}

func TestEmbeddedRuntimeMigrations(t *testing.T) {
	t.Parallel()

	database, err := New("runtime", Config{
		Path:         t.TempDir() + "/runtime.db",
		MaxOpenConns: 1,
		Pragmas:      PragmasRuntime,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	defer stopDB(t, database)

	if err := database.Start(context.Background()); err != nil {
		t.Fatalf("Start: %v", err)
	}

	assertObjectExists(t, database, "table", "agent_runs")
	assertObjectExists(t, database, "table", "runtime_events")
	assertObjectExists(t, database, "index", "idx_agent_runs_agent_due")
	assertObjectExists(t, database, "index", "idx_agent_runs_latest_logs")
	assertObjectExists(t, database, "index", "idx_runtime_events_run_created")
	assertObjectExists(t, database, "index", "idx_runtime_events_type_created")
}

func assertObjectExists(t *testing.T, database *DB, objectType, name string) {
	t.Helper()

	var count int
	if err := database.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type = ? AND name = ?",
		objectType,
		name,
	).Scan(&count); err != nil {
		t.Fatalf("query sqlite_master for %s %s: %v", objectType, name, err)
	}
	if count != 1 {
		t.Fatalf("%s %s count: got %d want 1", objectType, name, count)
	}
}

func stopDB(t *testing.T, database *DB) {
	t.Helper()

	if err := database.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
