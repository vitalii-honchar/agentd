package repository

import (
	"context"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
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

func TestAgentRunRepositoryPersistsTerminalResults(t *testing.T) {
	t.Parallel()

	fixture := newRuntimeRepositoryFixture(t, "cybersecurity-reddit-watch")
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(time.Minute)
	run := domain.AgentRun{
		ID:            "run-result-1",
		AgentName:     "cybersecurity-reddit-watch",
		AgentRevision: "rev-1",
		Trigger:       domain.RunTriggerManual,
		Status:        domain.AgentRunStatusRunning,
		StartedAt:     &now,
		WorkDir:       "/tmp/run-result-1",
		LogPath:       "/tmp/run-result-1/run.log",
	}
	if err := fixture.Runs.Create(context.Background(), run); err != nil {
		t.Fatalf("Create: %v", err)
	}

	run.Status = domain.AgentRunStatusFailed
	run.CompletedAt = &completedAt
	run.Result = "found likely credential exposure in one post"
	run.ResultSummary = "likely credential exposure"
	run.ErrorCode = "agent_failed"
	run.ErrorMessage = "analysis failed after tool output"
	if err := fixture.Runs.Update(context.Background(), run); err != nil {
		t.Fatalf("Update failed result: %v", err)
	}

	found, err := fixture.Runs.FindByID(context.Background(), run.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Result != run.Result {
		t.Fatalf("result: got %q want %q", found.Result, run.Result)
	}
	if found.ResultSummary != run.ResultSummary {
		t.Fatalf("result summary: got %q want %q", found.ResultSummary, run.ResultSummary)
	}
	if found.ErrorCode != run.ErrorCode || found.ErrorMessage != run.ErrorMessage {
		t.Fatalf("failure fields: %#v", found)
	}

	terminal, err := fixture.Runs.ListTerminal(context.Background())
	if err != nil {
		t.Fatalf("ListTerminal: %v", err)
	}
	if len(terminal) != 1 || terminal[0].ID != run.ID || terminal[0].Result != run.Result {
		t.Fatalf("terminal results: %#v", terminal)
	}
}

func TestAgentRunRepositoryListsActiveAndAllRuns(t *testing.T) {
	t.Parallel()

	fixture := newRuntimeRepositoryFixture(t, "hacker-news-builder-brief")
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	completedAt := now.Add(2 * time.Minute)
	runs := []domain.AgentRun{
		{
			ID:            "run-queued",
			AgentName:     "hacker-news-builder-brief",
			AgentRevision: "rev-1",
			Trigger:       domain.RunTriggerSchedule,
			Status:        domain.AgentRunStatusQueued,
			DueAt:         &now,
			WorkDir:       "/tmp/run-queued",
			LogPath:       "/tmp/run-queued/run.log",
		},
		{
			ID:            "run-running",
			AgentName:     "hacker-news-builder-brief",
			AgentRevision: "rev-1",
			Trigger:       domain.RunTriggerManual,
			Status:        domain.AgentRunStatusRunning,
			StartedAt:     &now,
			WorkDir:       "/tmp/run-running",
			LogPath:       "/tmp/run-running/run.log",
		},
		{
			ID:            "run-completed",
			AgentName:     "hacker-news-builder-brief",
			AgentRevision: "rev-1",
			Trigger:       domain.RunTriggerSchedule,
			Status:        domain.AgentRunStatusCompleted,
			StartedAt:     &now,
			CompletedAt:   &completedAt,
			WorkDir:       "/tmp/run-completed",
			LogPath:       "/tmp/run-completed/run.log",
			Result:        "HN builder brief",
			ResultSummary: "HN brief",
		},
	}
	for _, run := range runs {
		if err := fixture.Runs.Create(context.Background(), run); err != nil {
			t.Fatalf("Create %s: %v", run.ID, err)
		}
	}

	active, err := fixture.Runs.ListActive(context.Background())
	if err != nil {
		t.Fatalf("ListActive: %v", err)
	}
	if got := runIDs(active); !sameStringSet(got, []string{"run-queued", "run-running"}) {
		t.Fatalf("active run ids: got %#v", got)
	}

	all, err := fixture.Runs.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if got := runIDs(all); !sameStringSet(got, []string{"run-queued", "run-running", "run-completed"}) {
		t.Fatalf("all run ids: got %#v", got)
	}
}

func TestAgentRunRepositoryLooksUpResultsByAgentAndRunID(t *testing.T) {
	t.Parallel()

	fixture := newRuntimeRepositoryFixture(t, "website-snapshot-analyst")
	now := time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC)
	completedAt := now.Add(time.Minute)
	active := domain.AgentRun{
		ID:            "active-run",
		AgentName:     "website-snapshot-analyst",
		AgentRevision: "rev-1",
		Trigger:       domain.RunTriggerManual,
		Status:        domain.AgentRunStatusRunning,
		StartedAt:     &now,
		WorkDir:       "/tmp/active-run",
		LogPath:       "/tmp/active-run/run.log",
	}
	terminal := domain.AgentRun{
		ID:            "terminal-run",
		AgentName:     "website-snapshot-analyst",
		AgentRevision: "rev-1",
		Trigger:       domain.RunTriggerManual,
		Status:        domain.AgentRunStatusCompleted,
		StartedAt:     &now,
		CompletedAt:   &completedAt,
		WorkDir:       "/tmp/terminal-run",
		LogPath:       "/tmp/terminal-run/run.log",
		Result:        "Full website analysis result",
		ResultSummary: "Website analysis summary",
	}
	for _, run := range []domain.AgentRun{active, terminal} {
		if err := fixture.Runs.Create(context.Background(), run); err != nil {
			t.Fatalf("Create %s: %v", run.ID, err)
		}
	}

	found, err := fixture.Runs.FindByID(context.Background(), terminal.ID)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if found.Result != terminal.Result || found.ResultSummary != terminal.ResultSummary {
		t.Fatalf("found result: %#v", found)
	}

	results, err := fixture.Runs.ListTerminal(context.Background())
	if err != nil {
		t.Fatalf("ListTerminal: %v", err)
	}
	if len(results) != 1 || results[0].ID != terminal.ID {
		t.Fatalf("terminal results: %#v", results)
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

func runIDs(runs []domain.AgentRun) []string {
	ids := make([]string, 0, len(runs))
	for _, run := range runs {
		ids = append(ids, run.ID)
	}

	return ids
}

func sameStringSet(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	counts := make(map[string]int, len(want))
	for _, value := range want {
		counts[value]++
	}
	for _, value := range got {
		counts[value]--
		if counts[value] < 0 {
			return false
		}
	}

	return true
}
