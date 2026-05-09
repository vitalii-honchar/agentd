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

func TestSettingsMigrationsCreateRevisionMetadataTables(t *testing.T) {
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

	assertObjectExists(t, database, "table", "agent_revisions")
	assertObjectExists(t, database, "table", "agent_revision_tools")
	assertObjectExists(t, database, "table", "agent_revision_artifact_files")
	assertObjectExists(t, database, "table", "agent_revision_environment")
	assertObjectExists(t, database, "index", "idx_agent_revisions_agent_digest")
	assertObjectExists(t, database, "index", "idx_agent_revisions_status")
	assertObjectExists(t, database, "index", "idx_agent_revision_tools_kind")
	assertObjectExists(t, database, "index", "idx_agent_revision_environment_key")
}

func TestSettingsMigrationsEnforceRevisionDigestUniquenessPerAgent(t *testing.T) {
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
	insertMigrationTestAgent(t, database, "digest-agent")
	insertMigrationTestRevision(t, database, "digest-agent", "revision-1", "sha256:abc")

	if _, err := database.ExecContext(context.Background(), `INSERT INTO agent_revisions (
	    agent_name, revision_id, content_digest, source_path, artifact_path,
	    prompt, vendor_name, vendor_model, schedule_type, status, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"digest-agent",
		"revision-2",
		"sha256:abc",
		"/tmp/agent.md",
		"/tmp/data/work/digest-agent/revision-2",
		"Prompt",
		"openai",
		"gpt-test",
		"manual",
		"finalized",
		"2026-05-08T00:00:00Z",
	); err == nil {
		t.Fatal("expected duplicate revision digest to fail")
	}
}

func TestSettingsMigrationsAddNullableContractMetadata(t *testing.T) {
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

	for _, column := range []string{
		"contract_input_schema_raw",
		"contract_output_schema_raw",
		"contract_input_schema_digest",
		"contract_output_schema_digest",
	} {
		assertNullableColumnExists(t, database, "agents", column)
	}
	for _, column := range []string{
		"contract_input_schema_raw",
		"contract_output_schema_raw",
		"contract_input_schema_digest",
		"contract_output_schema_digest",
		"contract_digest",
	} {
		assertNullableColumnExists(t, database, "agent_revisions", column)
	}

	insertMigrationTestAgent(t, database, "legacy-contract-agent")
	insertMigrationTestRevision(t, database, "legacy-contract-agent", "revision-1", "sha256:legacy")

	var inputSchema any
	if err := database.QueryRowContext(
		context.Background(),
		"SELECT contract_input_schema_raw FROM agents WHERE name = ?",
		"legacy-contract-agent",
	).Scan(&inputSchema); err != nil {
		t.Fatalf("select legacy agent contract column: %v", err)
	}
	if inputSchema != nil {
		t.Fatalf("legacy agent contract column: got %#v want nil", inputSchema)
	}
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
	assertObjectExists(t, database, "index", "idx_agent_runs_terminal_results")
	assertObjectExists(t, database, "table", "tool_executions")
	assertObjectExists(t, database, "index", "idx_tool_executions_run_started")
	assertObjectExists(t, database, "index", "idx_runtime_events_run_created")
	assertObjectExists(t, database, "index", "idx_runtime_events_type_created")
}

func insertMigrationTestAgent(t *testing.T, database *DB, agentName string) {
	t.Helper()

	if _, err := database.ExecContext(context.Background(), `INSERT INTO agents (
	    name, revision, definition_source_path, definition_markdown, prompt, enabled,
	    vendor_name, vendor_model, schedule_type, status, created_at, updated_at, applied_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agentName,
		"source-revision",
		"/tmp/agent.md",
		"# Agent",
		"Prompt",
		1,
		"openai",
		"gpt-test",
		"manual",
		"active",
		"2026-05-08T00:00:00Z",
		"2026-05-08T00:00:00Z",
		"2026-05-08T00:00:00Z",
	); err != nil {
		t.Fatalf("insert migration test agent: %v", err)
	}
}

func insertMigrationTestRevision(t *testing.T, database *DB, agentName, revisionID, digest string) {
	t.Helper()

	if _, err := database.ExecContext(context.Background(), `INSERT INTO agent_revisions (
	    agent_name, revision_id, content_digest, source_path, artifact_path,
	    prompt, vendor_name, vendor_model, schedule_type, status, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		agentName,
		revisionID,
		digest,
		"/tmp/agent.md",
		"/tmp/data/work/"+agentName+"/"+revisionID,
		"Prompt",
		"openai",
		"gpt-test",
		"manual",
		"finalized",
		"2026-05-08T00:00:00Z",
	); err != nil {
		t.Fatalf("insert migration test revision: %v", err)
	}
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

func assertNullableColumnExists(t *testing.T, database *DB, table string, column string) {
	t.Helper()

	rows, err := database.QueryContext(context.Background(), "PRAGMA table_info("+table+")")
	if err != nil {
		t.Fatalf("query table info for %s: %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid        int
			name       string
			columnType string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultVal, &pk); err != nil {
			t.Fatalf("scan table info for %s: %v", table, err)
		}
		if name == column {
			if notNull != 0 {
				t.Fatalf("column %s.%s should be nullable", table, column)
			}
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate table info for %s: %v", table, err)
	}
	t.Fatalf("column %s.%s does not exist", table, column)
}

func stopDB(t *testing.T, database *DB) {
	t.Helper()

	if err := database.Stop(context.Background()); err != nil {
		t.Fatalf("Stop: %v", err)
	}
}
