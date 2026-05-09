package repository

import (
	"context"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
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

func TestAgentRepositorySaveFindListAndLatestRevisions(t *testing.T) {
	t.Parallel()

	fixture := newSettingsRepositoryFixture(t)
	agent := testRepositoryAgent("revisioned-agent")
	if err := fixture.Agents.Save(context.Background(), agent, nil, nil); err != nil {
		t.Fatalf("Save agent: %v", err)
	}
	first := testRepositoryRevision(agent.Name, "11111111-1111-4111-8111-111111111111", "sha256:first", time.Date(2026, 5, 8, 10, 0, 0, 0, time.UTC))
	first.Tools = []domain.RevisionTool{{
		Name:             "collect",
		Kind:             domain.ToolKindCustomTool,
		OriginalCommand:  "./tools/collect.py",
		RewrittenCommand: "tools/collect.py",
		Args:             []string{"--limit", "5"},
		ReadPaths:        []string{"fixtures/input.json"},
		CopiedFiles:      []string{"tools/collect.py"},
		CreatedAt:        first.CreatedAt,
	}}
	second := testRepositoryRevision(agent.Name, "22222222-2222-4222-8222-222222222222", "sha256:second", time.Date(2026, 5, 8, 11, 0, 0, 0, time.UTC))

	if err := fixture.Agents.SaveRevision(context.Background(), first); err != nil {
		t.Fatalf("SaveRevision first: %v", err)
	}
	if err := fixture.Agents.SaveRevision(context.Background(), second); err != nil {
		t.Fatalf("SaveRevision second: %v", err)
	}

	found, err := fixture.Agents.FindRevisionByID(context.Background(), agent.Name, first.RevisionID)
	if err != nil {
		t.Fatalf("FindRevisionByID: %v", err)
	}
	if found.RevisionID != first.RevisionID || found.ContentDigest != first.ContentDigest {
		t.Fatalf("found revision: %#v", found)
	}
	if len(found.Tools) != 1 || found.Tools[0].RewrittenCommand != "tools/collect.py" {
		t.Fatalf("found tools: %#v", found.Tools)
	}

	byDigest, err := fixture.Agents.FindRevisionByDigest(context.Background(), agent.Name, first.ContentDigest)
	if err != nil {
		t.Fatalf("FindRevisionByDigest: %v", err)
	}
	if byDigest.RevisionID != first.RevisionID {
		t.Fatalf("digest revision: got %q want %q", byDigest.RevisionID, first.RevisionID)
	}

	revisions, err := fixture.Agents.ListRevisions(context.Background(), agent.Name)
	if err != nil {
		t.Fatalf("ListRevisions: %v", err)
	}
	if len(revisions) != 2 {
		t.Fatalf("revision count: got %d want 2", len(revisions))
	}
	if revisions[0].RevisionID != second.RevisionID || !revisions[0].IsLatestFinalized {
		t.Fatalf("first listed revision: %#v", revisions[0])
	}
	if revisions[1].RevisionID != first.RevisionID || revisions[1].IsLatestFinalized {
		t.Fatalf("second listed revision: %#v", revisions[1])
	}

	latest, err := fixture.Agents.FindLatestFinalizedRevision(context.Background(), agent.Name)
	if err != nil {
		t.Fatalf("FindLatestFinalizedRevision: %v", err)
	}
	if latest.RevisionID != second.RevisionID || !latest.IsLatestFinalized {
		t.Fatalf("latest revision: %#v", latest)
	}
	foundAgent, err := fixture.Agents.FindByName(context.Background(), agent.Name)
	if err != nil {
		t.Fatalf("FindByName after revisions: %v", err)
	}
	if foundAgent.Revision != second.RevisionID {
		t.Fatalf("agent latest revision: got %q want %q", foundAgent.Revision, second.RevisionID)
	}
}

func TestAgentRepositoryPersistsAgentAndRevisionContracts(t *testing.T) {
	t.Parallel()

	fixture := newSettingsRepositoryFixture(t)
	agent := testRepositoryAgent("contract-agent")
	agent.Contract = &domain.AgentContract{
		InputSchemaRaw:     `{"type":"object","required":["topic"]}`,
		OutputSchemaRaw:    `{"type":"object","required":["summary"]}`,
		InputSchemaDigest:  "sha256:input",
		OutputSchemaDigest: "sha256:output",
	}
	if err := fixture.Agents.Save(context.Background(), agent, nil, nil); err != nil {
		t.Fatalf("Save agent: %v", err)
	}

	foundAgent, err := fixture.Agents.FindByName(context.Background(), agent.Name)
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if foundAgent.Contract == nil {
		t.Fatal("found agent contract is nil")
	}
	if foundAgent.Contract.InputSchemaRaw != agent.Contract.InputSchemaRaw ||
		foundAgent.Contract.OutputSchemaRaw != agent.Contract.OutputSchemaRaw ||
		foundAgent.Contract.InputSchemaDigest != agent.Contract.InputSchemaDigest ||
		foundAgent.Contract.OutputSchemaDigest != agent.Contract.OutputSchemaDigest {
		t.Fatalf("found agent contract: %#v", foundAgent.Contract)
	}

	revision := testRepositoryRevision(agent.Name, "44444444-4444-4444-8444-444444444444", "sha256:contracted", time.Date(2026, 5, 8, 13, 0, 0, 0, time.UTC))
	revision.ContractInputSchemaRaw = agent.Contract.InputSchemaRaw
	revision.ContractOutputSchemaRaw = agent.Contract.OutputSchemaRaw
	revision.ContractInputSchemaDigest = agent.Contract.InputSchemaDigest
	revision.ContractOutputSchemaDigest = agent.Contract.OutputSchemaDigest
	revision.ContractDigest = "sha256:contract"
	if err := fixture.Agents.SaveRevision(context.Background(), revision); err != nil {
		t.Fatalf("SaveRevision: %v", err)
	}

	foundRevision, err := fixture.Agents.FindRevisionByID(context.Background(), agent.Name, revision.RevisionID)
	if err != nil {
		t.Fatalf("FindRevisionByID: %v", err)
	}
	if foundRevision.ContractInputSchemaRaw != revision.ContractInputSchemaRaw ||
		foundRevision.ContractOutputSchemaRaw != revision.ContractOutputSchemaRaw ||
		foundRevision.ContractInputSchemaDigest != revision.ContractInputSchemaDigest ||
		foundRevision.ContractOutputSchemaDigest != revision.ContractOutputSchemaDigest ||
		foundRevision.ContractDigest != revision.ContractDigest {
		t.Fatalf("found revision contract: %#v", foundRevision)
	}
}

func TestAgentRepositoryPersistsRevisionEnvironmentArtifactFilesAndCorruption(t *testing.T) {
	t.Parallel()

	fixture := newSettingsRepositoryFixture(t)
	agent := testRepositoryAgent("artifact-agent")
	if err := fixture.Agents.Save(context.Background(), agent, nil, nil); err != nil {
		t.Fatalf("Save agent: %v", err)
	}
	createdAt := time.Date(2026, 5, 8, 12, 0, 0, 0, time.UTC)
	revision := testRepositoryRevision(agent.Name, "33333333-3333-4333-8333-333333333333", "sha256:artifact", createdAt)
	revision.Environment = []domain.RevisionEnvironment{
		{
			Key:                  "GITHUB_TOKEN",
			Value:                "from-file",
			Source:               domain.RevisionEnvironmentSourceEnvFile,
			SourcePath:           "/tmp/artifact-agent/.env",
			ArtifactRelativePath: "env/.env",
			Masked:               true,
			CreatedAt:            createdAt,
		},
		{
			Key:       "REPORT_LIMIT",
			Value:     "10",
			Source:    domain.RevisionEnvironmentSourceLiteral,
			Masked:    false,
			CreatedAt: createdAt,
		},
	}
	revision.ArtifactFiles = []domain.RevisionArtifactFile{
		{
			ArtifactRelativePath: "tools/collect.py",
			SourcePath:           "/tmp/artifact-agent/tools/collect.py",
			SHA256:               "abc123",
			Mode:                 0o755,
			SizeBytes:            1024,
			CopiedAt:             createdAt,
		},
	}

	if err := fixture.Agents.SaveRevision(context.Background(), revision); err != nil {
		t.Fatalf("SaveRevision: %v", err)
	}

	found, err := fixture.Agents.FindRevisionByID(context.Background(), agent.Name, revision.RevisionID)
	if err != nil {
		t.Fatalf("FindRevisionByID: %v", err)
	}
	if len(found.Environment) != 2 {
		t.Fatalf("environment count: got %d want 2", len(found.Environment))
	}
	if found.Environment[0].Key != "GITHUB_TOKEN" || found.Environment[0].ArtifactRelativePath != "env/.env" || !found.Environment[0].Masked {
		t.Fatalf("first environment entry: %#v", found.Environment[0])
	}
	if len(found.ArtifactFiles) != 1 {
		t.Fatalf("artifact file count: got %d want 1", len(found.ArtifactFiles))
	}
	if found.ArtifactFiles[0].ArtifactRelativePath != "tools/collect.py" || found.ArtifactFiles[0].Mode != 0o755 {
		t.Fatalf("artifact file: %#v", found.ArtifactFiles[0])
	}

	if err := fixture.Agents.MarkRevisionCorrupt(context.Background(), agent.Name, revision.RevisionID, "missing tools/collect.py"); err != nil {
		t.Fatalf("MarkRevisionCorrupt: %v", err)
	}
	corrupt, err := fixture.Agents.FindRevisionByID(context.Background(), agent.Name, revision.RevisionID)
	if err != nil {
		t.Fatalf("FindRevisionByID corrupt: %v", err)
	}
	if corrupt.Status != domain.AgentRevisionStatusCorrupt || corrupt.ErrorMessage != "missing tools/collect.py" {
		t.Fatalf("corrupt revision: %#v", corrupt)
	}
}

func testRepositoryAgent(name string) domain.Agent {
	now := time.Date(2026, 5, 8, 9, 0, 0, 0, time.UTC)

	return domain.Agent{
		Name:               name,
		Revision:           "source-revision",
		DefinitionSource:   name + ".md",
		DefinitionMarkdown: "---\nname: " + name + "\n---\nPrompt",
		Prompt:             "Prompt",
		Enabled:            true,
		Vendor:             domain.Vendor{Name: "openai", Model: "gpt-5"},
		Schedule:           domain.Schedule{Type: domain.ScheduleTypeManual},
		Status:             domain.AgentStatusActive,
		CreatedAt:          now,
		UpdatedAt:          now,
		AppliedAt:          now,
	}
}

func testRepositoryRevision(agentName, revisionID, digest string, createdAt time.Time) domain.AgentRevision {
	finalizedAt := createdAt.Add(time.Minute)

	return domain.AgentRevision{
		AgentName:       agentName,
		RevisionID:      revisionID,
		ContentDigest:   digest,
		SourcePath:      "/tmp/" + agentName + ".md",
		ArtifactPath:    "/tmp/data/work/" + agentName + "/" + revisionID,
		EnvironmentJSON: "[]",
		Prompt:          "Prompt " + revisionID,
		Vendor:          domain.Vendor{Name: "openai", Model: "gpt-5"},
		Schedule:        domain.Schedule{Type: domain.ScheduleTypeManual},
		Status:          domain.AgentRevisionStatusFinalized,
		CreatedAt:       createdAt,
		FinalizedAt:     &finalizedAt,
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
