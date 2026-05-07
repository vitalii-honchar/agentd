package repository

import (
	"context"
	"testing"
	"time"

	"agentd/internal/agentdserver/domain"
)

func TestAgentRepositorySaveFindAndList(t *testing.T) {
	t.Parallel()

	fixture := newSettingsRepositoryFixture(t)
	now := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	nextRun := now.Add(time.Hour)

	agent := domain.Agent{
		Name:               "daily-pr-review",
		Revision:           "rev-1",
		DefinitionSource:   "daily-pr-review.md",
		DefinitionMarkdown: "---\nname: daily-pr-review\n---\nReview PRs",
		Prompt:             "Review PRs",
		Enabled:            true,
		Vendor:             domain.Vendor{Name: "openai", Model: "gpt-5"},
		Schedule: domain.Schedule{
			Type:       domain.ScheduleTypeCron,
			Expression: "0 9 * * MON-FRI",
		},
		NextRunAt: &nextRun,
		Status:    domain.AgentStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
		AppliedAt: now,
	}
	tools := []domain.ToolPermission{{
		Kind:       domain.ToolKindLocalTool,
		Name:       "git",
		Command:    "git",
		Args:       []string{"status", "--short"},
		ReadPaths:  []string{"."},
		WritePaths: []string{},
	}}
	mcpServers := []domain.ToolPermission{{
		Name:    "github",
		Command: "github-mcp-server",
		Env:     []string{"GITHUB_TOKEN"},
	}}

	if err := fixture.Agents.Save(context.Background(), agent, tools, mcpServers); err != nil {
		t.Fatalf("Save: %v", err)
	}

	found, err := fixture.Agents.FindByName(context.Background(), agent.Name)
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if found.Name != agent.Name || found.Revision != agent.Revision {
		t.Fatalf("found agent: %#v", found)
	}
	if found.Schedule.Expression != agent.Schedule.Expression {
		t.Fatalf("schedule expression: got %q", found.Schedule.Expression)
	}
	if found.NextRunAt == nil || !found.NextRunAt.Equal(nextRun) {
		t.Fatalf("next run: got %v want %v", found.NextRunAt, nextRun)
	}

	agents, err := fixture.Agents.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(agents) != 1 || agents[0].Name != agent.Name {
		t.Fatalf("agents: %#v", agents)
	}
	assertRowCount(t, fixture, "agent_tools", 1)
	assertRowCount(t, fixture, "agent_mcp_servers", 1)
}

func TestAgentRepositorySaveReplacesPolicies(t *testing.T) {
	t.Parallel()

	fixture := newSettingsRepositoryFixture(t)
	agent := domain.Agent{
		Name:               "release-notes-helper",
		Revision:           "rev-1",
		DefinitionSource:   "release-notes-helper.md",
		DefinitionMarkdown: "---\nname: release-notes-helper\n---\nSummarize",
		Prompt:             "Summarize",
		Enabled:            true,
		Vendor:             domain.Vendor{Name: "openai", Model: "gpt-5"},
		Schedule:           domain.Schedule{Type: domain.ScheduleTypeManual},
		Status:             domain.AgentStatusActive,
	}

	if err := fixture.Agents.Save(context.Background(), agent, []domain.ToolPermission{{
		Kind:    domain.ToolKindLocalTool,
		Name:    "git",
		Command: "git",
	}}, nil); err != nil {
		t.Fatalf("Save first: %v", err)
	}
	agent.Revision = "rev-2"
	if err := fixture.Agents.Save(context.Background(), agent, []domain.ToolPermission{{
		Kind:    domain.ToolKindLocalTool,
		Name:    "gh",
		Command: "gh",
	}}, []domain.ToolPermission{{
		Name:    "github",
		Command: "github-mcp-server",
	}}); err != nil {
		t.Fatalf("Save second: %v", err)
	}

	found, err := fixture.Agents.FindByName(context.Background(), agent.Name)
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if found.Revision != "rev-2" {
		t.Fatalf("revision: got %q want rev-2", found.Revision)
	}
	assertRowCount(t, fixture, "agent_tools", 1)
	assertRowCount(t, fixture, "agent_mcp_servers", 1)
}

func TestAgentRepositoryFindUnknownAgent(t *testing.T) {
	t.Parallel()

	fixture := newSettingsRepositoryFixture(t)
	_, err := fixture.Agents.FindByName(context.Background(), "missing")
	if err != domain.ErrNotFound {
		t.Fatalf("FindByName error: got %v want %v", err, domain.ErrNotFound)
	}
}

func assertRowCount(t *testing.T, fixture settingsRepositoryFixture, table string, want int) {
	t.Helper()

	var count int
	if err := fixture.DB.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM "+table,
	).Scan(&count); err != nil {
		t.Fatalf("count %s: %v", table, err)
	}
	if count != want {
		t.Fatalf("%s count: got %d want %d", table, count, want)
	}
}
