package repository

import (
	"context"
	"testing"

	"agentd/internal/agentdserver/infra/db"
)

type settingsRepositoryFixture struct {
	DB     *db.DB
	Agents *AgentRepository
}

type runtimeRepositoryFixture struct {
	Manager *RuntimeDBManager
	Runs    *AgentRunRepository
	Events  *RuntimeEventRepository
}

func TestRepositoryFixturesOpenMigratedStores(t *testing.T) {
	t.Parallel()

	settings := newSettingsRepositoryFixture(t)
	if settings.DB == nil {
		t.Fatal("settings DB is nil")
	}
	if settings.Agents == nil {
		t.Fatal("settings Agent repository is nil")
	}

	runtime := newRuntimeRepositoryFixture(t, "fixture-agent")
	if runtime.Manager == nil {
		t.Fatal("runtime manager is nil")
	}
	if runtime.Runs == nil {
		t.Fatal("runtime run repository is nil")
	}
	if runtime.Events == nil {
		t.Fatal("runtime event repository is nil")
	}
}

func newSettingsRepositoryFixture(t *testing.T) settingsRepositoryFixture {
	t.Helper()

	database, err := db.New("settings", db.Config{
		Path:         t.TempDir() + "/settings.db",
		MaxOpenConns: 1,
		Pragmas:      db.PragmasSettings,
	})
	if err != nil {
		t.Fatalf("New settings DB: %v", err)
	}
	t.Cleanup(func() {
		if err := database.Stop(context.Background()); err != nil {
			t.Fatalf("Stop settings DB: %v", err)
		}
	})
	if err := database.Start(context.Background()); err != nil {
		t.Fatalf("Start settings DB: %v", err)
	}

	agents, err := NewAgentRepository(database)
	if err != nil {
		t.Fatalf("NewAgentRepository: %v", err)
	}

	return settingsRepositoryFixture{DB: database, Agents: agents}
}

func newRuntimeRepositoryFixture(t *testing.T, agentName string) runtimeRepositoryFixture {
	t.Helper()

	manager, err := NewRuntimeDBManager(t.TempDir(), 1)
	if err != nil {
		t.Fatalf("NewRuntimeDBManager: %v", err)
	}
	t.Cleanup(func() {
		if err := manager.Close(context.Background()); err != nil {
			t.Fatalf("Close runtime manager: %v", err)
		}
	})
	if err := manager.EnsureAgent(context.Background(), agentName); err != nil {
		t.Fatalf("EnsureAgent: %v", err)
	}

	runs, ok := manager.Runs(agentName).(*AgentRunRepository)
	if !ok {
		t.Fatalf("Runs returned %T", manager.Runs(agentName))
	}
	events, ok := manager.Events(agentName).(*RuntimeEventRepository)
	if !ok {
		t.Fatalf("Events returned %T", manager.Events(agentName))
	}

	return runtimeRepositoryFixture{
		Manager: manager,
		Runs:    runs,
		Events:  events,
	}
}
