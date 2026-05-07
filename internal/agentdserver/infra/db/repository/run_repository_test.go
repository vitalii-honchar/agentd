package repository

import (
	"context"
	"testing"
	"time"

	"agentd/internal/agentdserver/domain"
)

func TestAgentRunRepositoryCreateUpdateQuery(t *testing.T) {
	t.Parallel()

	fixture := newRuntimeRepositoryFixture(t, "release-notes-helper")
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	run := domain.AgentRun{
		ID:            "run-1",
		AgentName:     "release-notes-helper",
		AgentRevision: "rev-1",
		Trigger:       domain.RunTriggerManual,
		Status:        domain.AgentRunStatusQueued,
		StartedAt:     &now,
		WorkDir:       "/tmp/run-1",
		LogPath:       "/tmp/run-1/run.log",
	}

	if err := fixture.Runs.Create(context.Background(), run); err != nil {
		t.Fatalf("Create: %v", err)
	}
	found, err := fixture.Runs.FindByID(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.ID != run.ID || found.Status != domain.AgentRunStatusQueued {
		t.Fatalf("found run: %#v", found)
	}

	run.Status = domain.AgentRunStatusRunning
	if err := fixture.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("Update: %v", err)
	}
	active, err := fixture.Runs.FindActive(context.Background())
	if err != nil {
		t.Fatalf("FindActive: %v", err)
	}
	if active.Status != domain.AgentRunStatusRunning {
		t.Fatalf("active status: got %q", active.Status)
	}

	completedAt := now.Add(time.Minute)
	run.Status = domain.AgentRunStatusCompleted
	run.CompletedAt = &completedAt
	if err := fixture.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("Update completed: %v", err)
	}
	activeRuns, err := fixture.Runs.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if len(activeRuns) != 0 {
		t.Fatalf("active runs: %#v", activeRuns)
	}
}

func TestRuntimeEventRepositoryAppendAndQuery(t *testing.T) {
	t.Parallel()

	fixture := newRuntimeRepositoryFixture(t, "release-notes-helper")
	run := domain.AgentRun{
		ID:            "run-1",
		AgentName:     "release-notes-helper",
		AgentRevision: "rev-1",
		Trigger:       domain.RunTriggerManual,
		Status:        domain.AgentRunStatusRunning,
		WorkDir:       "/tmp/run-1",
		LogPath:       "/tmp/run-1/run.log",
	}
	if err := fixture.Runs.Create(context.Background(), run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	event := domain.RuntimeEvent{
		ID:             "event-1",
		AgentName:      "release-notes-helper",
		RunID:          "run-1",
		EventType:      "agent.run.started",
		Level:          domain.EventLevelInfo,
		Message:        "run started",
		AttributesJSON: "{}",
		CreatedAt:      time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC),
	}
	if err := fixture.Events.Append(context.Background(), event); err != nil {
		t.Fatalf("Append: %v", err)
	}

	byRun, err := fixture.Events.ListByRun(context.Background(), "run-1", 10)
	if err != nil {
		t.Fatalf("ListByRun: %v", err)
	}
	if len(byRun) != 1 || byRun[0].EventType != "agent.run.started" {
		t.Fatalf("events by run: %#v", byRun)
	}

	recent, err := fixture.Events.ListRecent(context.Background(), 10)
	if err != nil {
		t.Fatalf("ListRecent: %v", err)
	}
	if len(recent) != 1 || recent[0].ID != "event-1" {
		t.Fatalf("recent events: %#v", recent)
	}
}
