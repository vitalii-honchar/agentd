package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/definition"
)

func TestApplyUseCaseCreatedUpdatedUnchanged(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	runtimeDBs := &memoryRuntimeDBManager{}
	useCase := newApplyUseCaseForTest(t, repo, runtimeDBs)

	created, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "release-notes-helper.md",
		Markdown:   manualDefinition("Summarize changes."),
	})
	if err != nil {
		t.Fatalf("Apply created: %v", err)
	}
	if created.Outcome != ApplyOutcomeCreated {
		t.Fatalf("created outcome: got %q want %q", created.Outcome, ApplyOutcomeCreated)
	}
	if !runtimeDBs.ensured["release-notes-helper"] {
		t.Fatal("runtime DB was not ensured for created Agent")
	}

	unchanged, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "release-notes-helper.md",
		Markdown:   manualDefinition("Summarize changes."),
	})
	if err != nil {
		t.Fatalf("Apply unchanged: %v", err)
	}
	if unchanged.Outcome != ApplyOutcomeUnchanged {
		t.Fatalf("unchanged outcome: got %q want %q", unchanged.Outcome, ApplyOutcomeUnchanged)
	}

	updated, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "release-notes-helper.md",
		Markdown:   manualDefinition("Summarize updated changes."),
	})
	if err != nil {
		t.Fatalf("Apply updated: %v", err)
	}
	if updated.Outcome != ApplyOutcomeUpdated {
		t.Fatalf("updated outcome: got %q want %q", updated.Outcome, ApplyOutcomeUpdated)
	}
	if updated.Agent.Revision == created.Agent.Revision {
		t.Fatalf("updated revision should differ from created revision %q", created.Agent.Revision)
	}
}

func TestApplyUseCaseCreatesRevisionAndReusesUnchangedRevision(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	runtimeDBs := &memoryRuntimeDBManager{}
	useCase := newApplyUseCaseForTest(t, repo, runtimeDBs)

	created, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "release-notes-helper.md",
		Markdown:   manualDefinition("Summarize changes."),
	})
	if err != nil {
		t.Fatalf("Apply created: %v", err)
	}
	if created.Outcome != ApplyOutcomeCreated {
		t.Fatalf("created outcome: got %q want %q", created.Outcome, ApplyOutcomeCreated)
	}
	if len(repo.revisions) != 1 {
		t.Fatalf("created revisions: got %d want 1", len(repo.revisions))
	}
	firstRevisionID := repo.revisions[0].RevisionID
	if firstRevisionID == "" {
		t.Fatal("created revision id is empty")
	}
	if repo.revisions[0].Status != domain.AgentRevisionStatusFinalized {
		t.Fatalf("created revision status: got %q", repo.revisions[0].Status)
	}

	unchanged, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "release-notes-helper.md",
		Markdown:   manualDefinition("Summarize changes."),
	})
	if err != nil {
		t.Fatalf("Apply unchanged: %v", err)
	}
	if unchanged.Outcome != ApplyOutcomeUnchanged {
		t.Fatalf("unchanged outcome: got %q want %q", unchanged.Outcome, ApplyOutcomeUnchanged)
	}
	if len(repo.revisions) != 1 {
		t.Fatalf("unchanged revisions: got %d want 1", len(repo.revisions))
	}
	if repo.revisions[0].RevisionID != firstRevisionID {
		t.Fatalf("unchanged revision id: got %q want %q", repo.revisions[0].RevisionID, firstRevisionID)
	}
}

func TestApplyUseCaseCreatesDistinctRevisionsForPromptAndToolMutation(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	runtimeDBs := &memoryRuntimeDBManager{}
	useCase := newApplyUseCaseForTest(t, repo, runtimeDBs)

	if _, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "examples/revisioned/revisioned.md",
		Markdown:   toolDefinition("Summarize public repos.", "tools/fetch.py"),
	}); err != nil {
		t.Fatalf("Apply initial: %v", err)
	}
	if _, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "examples/revisioned/revisioned.md",
		Markdown:   toolDefinition("Summarize public repos with more detail.", "tools/fetch.py"),
	}); err != nil {
		t.Fatalf("Apply prompt mutation: %v", err)
	}
	if _, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "examples/revisioned/revisioned.md",
		Markdown:   toolDefinition("Summarize public repos with more detail.", "tools/fetch_v2.py"),
	}); err != nil {
		t.Fatalf("Apply tool mutation: %v", err)
	}

	if len(repo.revisions) != 3 {
		t.Fatalf("revisions: got %d want 3", len(repo.revisions))
	}
	if repo.revisions[0].Prompt == repo.revisions[1].Prompt {
		t.Fatalf("prompt mutation was not captured: %#v", repo.revisions)
	}
	if len(repo.revisions[2].Tools) != 1 || repo.revisions[2].Tools[0].OriginalCommand != "tools/fetch_v2.py" {
		t.Fatalf("tool mutation was not captured: %#v", repo.revisions[2].Tools)
	}
}

func TestApplyUseCaseRevisionRetainsPromptAndToolsAfterSourceDeletion(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "revisioned.md")
	repo := newMemoryAgentRepository()
	runtimeDBs := &memoryRuntimeDBManager{}
	useCase := newApplyUseCaseForTest(t, repo, runtimeDBs)

	if _, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: sourcePath,
		Markdown:   toolDefinition("Keep this prompt in the revision.", "tools/fetch.py"),
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if err := os.RemoveAll(sourceDir); err != nil {
		t.Fatalf("RemoveAll source dir: %v", err)
	}

	if len(repo.revisions) != 1 {
		t.Fatalf("revisions: got %d want 1", len(repo.revisions))
	}
	revision := repo.revisions[0]
	if revision.Prompt != "Keep this prompt in the revision." {
		t.Fatalf("revision prompt: got %q", revision.Prompt)
	}
	if len(revision.Tools) != 1 || revision.Tools[0].OriginalCommand != "tools/fetch.py" {
		t.Fatalf("revision tools: %#v", revision.Tools)
	}
}

func TestApplyUseCaseCapturesEnvironmentVariablesAndFiles(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	sourcePath := filepath.Join(sourceDir, "env-agent.md")
	if err := os.WriteFile(filepath.Join(sourceDir, ".env"), []byte("TOKEN=from-file\nSHARED=from-file\n"), 0o600); err != nil {
		t.Fatalf("WriteFile .env: %v", err)
	}
	repo := newMemoryAgentRepository()
	runtimeDBs := &memoryRuntimeDBManager{}
	useCase := newApplyUseCaseForTest(t, repo, runtimeDBs)

	if _, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: sourcePath,
		Markdown:   environmentDefinition("Use captured environment."),
	}); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if len(repo.revisions) != 1 {
		t.Fatalf("revisions: got %d want 1", len(repo.revisions))
	}
	values := revisionEnvironmentToMap(repo.revisions[0].Environment)
	if values["TOKEN"] != "from-file" {
		t.Fatalf("TOKEN: got %q", values["TOKEN"])
	}
	if values["SHARED"] != "from-literal" {
		t.Fatalf("SHARED: got %q", values["SHARED"])
	}
	if values["REPORT_LIMIT"] != "10" {
		t.Fatalf("REPORT_LIMIT: got %q", values["REPORT_LIMIT"])
	}
	if len(repo.revisions[0].ArtifactFiles) != 1 || repo.revisions[0].ArtifactFiles[0].ArtifactRelativePath != ".env" {
		t.Fatalf("artifact env files: %#v", repo.revisions[0].ArtifactFiles)
	}
}

func TestApplyUseCaseRejectsInvalidDefinitionWithoutMutation(t *testing.T) {
	t.Parallel()

	repo := newMemoryAgentRepository()
	runtimeDBs := &memoryRuntimeDBManager{}
	useCase := newApplyUseCaseForTest(t, repo, runtimeDBs)

	_, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "bad.md",
		Markdown: `---
name: Bad Name
schedule:
  type: manual
vendor:
  name: openai
  model: ""
---
`,
	})
	if err == nil {
		t.Fatal("Apply returned nil error")
	}
	if !errors.Is(err, domain.ErrInvalidDefinition) {
		t.Fatalf("Apply error %v does not match ErrInvalidDefinition", err)
	}
	if len(repo.agents) != 0 {
		t.Fatalf("repo mutated after invalid apply: %#v", repo.agents)
	}
	if len(runtimeDBs.ensured) != 0 {
		t.Fatalf("runtime DBs mutated after invalid apply: %#v", runtimeDBs.ensured)
	}
}

func TestApplyUseCaseLogsAppliedAndRejectedOutcomes(t *testing.T) {
	repo := newMemoryAgentRepository()
	runtimeDBs := &memoryRuntimeDBManager{}
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&logBuffer, &slog.HandlerOptions{}))
	useCase := newApplyUseCaseForTest(t, repo, runtimeDBs, WithLogger(logger))

	_, err := useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "/tmp/release-notes-helper.md",
		Markdown:   manualDefinition("Do not leak this prompt into service logs."),
	})
	if err != nil {
		t.Fatalf("Apply created: %v", err)
	}
	_, err = useCase.Apply(context.Background(), ApplyRequest{
		SourcePath: "/tmp/bad.md",
		Markdown: `---
name: Bad Name
schedule:
  type: manual
vendor:
  name: openai
  model: ""
---
secret prompt text
`,
	})
	if err == nil {
		t.Fatal("Apply invalid definition returned nil error")
	}

	records := parseLogRecords(t, logBuffer.Bytes())
	if len(records) != 2 {
		t.Fatalf("log records: got %d want 2: %#v", len(records), records)
	}

	created := records[0]
	if created["msg"] != "agent.apply.created" {
		t.Fatalf("created msg: got %#v", created["msg"])
	}
	if created["event"] != "agent.apply.created" {
		t.Fatalf("created event: got %#v", created["event"])
	}
	if created["agent"] != "release-notes-helper" {
		t.Fatalf("created agent: got %#v", created["agent"])
	}
	if created["outcome"] != "created" {
		t.Fatalf("created outcome: got %#v", created["outcome"])
	}
	if created["revision"] == "" {
		t.Fatal("created revision was not logged")
	}
	if created["source_path"] != "/tmp/release-notes-helper.md" {
		t.Fatalf("created source_path: got %#v", created["source_path"])
	}
	if _, ok := created["prompt"]; ok {
		t.Fatalf("created log leaked prompt attribute: %#v", created)
	}
	if _, ok := created["markdown"]; ok {
		t.Fatalf("created log leaked markdown attribute: %#v", created)
	}

	rejected := records[1]
	if rejected["msg"] != "agent.apply.rejected" {
		t.Fatalf("rejected msg: got %#v", rejected["msg"])
	}
	if rejected["event"] != "agent.apply.rejected" {
		t.Fatalf("rejected event: got %#v", rejected["event"])
	}
	if rejected["outcome"] != "rejected" {
		t.Fatalf("rejected outcome: got %#v", rejected["outcome"])
	}
	if rejected["source_path"] != "/tmp/bad.md" {
		t.Fatalf("rejected source_path: got %#v", rejected["source_path"])
	}
	if rejected["error"] == "" {
		t.Fatalf("rejected error was not logged: %#v", rejected)
	}
	if _, ok := rejected["prompt"]; ok {
		t.Fatalf("rejected log leaked prompt attribute: %#v", rejected)
	}
	if _, ok := rejected["markdown"]; ok {
		t.Fatalf("rejected log leaked markdown attribute: %#v", rejected)
	}
}

func newApplyUseCaseForTest(
	t *testing.T,
	repo *memoryAgentRepository,
	runtimeDBs *memoryRuntimeDBManager,
	options ...ApplyOption,
) *ApplyUseCase {
	t.Helper()

	useCase, err := NewApplyUseCase(
		ParserFunc(definition.ParseMarkdown),
		repo,
		runtimeDBs,
		options...,
	)
	if err != nil {
		t.Fatalf("NewApplyUseCase: %v", err)
	}

	return useCase
}

func parseLogRecords(t *testing.T, logs []byte) []map[string]any {
	t.Helper()

	lines := bytes.Split(bytes.TrimSpace(logs), []byte("\n"))
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal(line, &record); err != nil {
			t.Fatalf("parse log record %q: %v", string(line), err)
		}
		records = append(records, record)
	}

	return records
}

type memoryAgentRepository struct {
	agents    map[string]domain.Agent
	revisions []domain.AgentRevision
}

func newMemoryAgentRepository() *memoryAgentRepository {
	return &memoryAgentRepository{agents: make(map[string]domain.Agent)}
}

func (r *memoryAgentRepository) Save(
	_ context.Context,
	agent domain.Agent,
	_ []domain.ToolPermission,
	_ []domain.ToolPermission,
) error {
	r.agents[agent.Name] = agent

	return nil
}

func (r *memoryAgentRepository) FindByName(_ context.Context, name string) (domain.Agent, error) {
	agent, ok := r.agents[name]
	if !ok {
		return domain.Agent{}, domain.ErrNotFound
	}

	return agent, nil
}

func (r *memoryAgentRepository) List(context.Context) ([]domain.Agent, error) {
	agents := make([]domain.Agent, 0, len(r.agents))
	for _, agent := range r.agents {
		agents = append(agents, agent)
	}

	return agents, nil
}

func (r *memoryAgentRepository) SaveRevision(_ context.Context, revision domain.AgentRevision) error {
	r.revisions = append(r.revisions, revision)

	return nil
}

func (r *memoryAgentRepository) ListRevisions(_ context.Context, agentName string) ([]domain.AgentRevision, error) {
	var revisions []domain.AgentRevision
	for _, revision := range r.revisions {
		if revision.AgentName == agentName {
			revisions = append(revisions, revision)
		}
	}

	return revisions, nil
}

func (r *memoryAgentRepository) FindRevisionByID(
	_ context.Context,
	agentName string,
	revisionID string,
) (domain.AgentRevision, error) {
	for _, revision := range r.revisions {
		if revision.AgentName == agentName && revision.RevisionID == revisionID {
			return revision, nil
		}
	}

	return domain.AgentRevision{}, domain.ErrNotFound
}

func (r *memoryAgentRepository) FindRevisionByDigest(
	_ context.Context,
	agentName string,
	contentDigest string,
) (domain.AgentRevision, error) {
	for _, revision := range r.revisions {
		if revision.AgentName == agentName && revision.ContentDigest == contentDigest {
			return revision, nil
		}
	}

	return domain.AgentRevision{}, domain.ErrNotFound
}

func (r *memoryAgentRepository) FindLatestFinalizedRevision(
	_ context.Context,
	agentName string,
) (domain.AgentRevision, error) {
	for i := len(r.revisions) - 1; i >= 0; i-- {
		revision := r.revisions[i]
		if revision.AgentName == agentName && revision.Status == domain.AgentRevisionStatusFinalized {
			return revision, nil
		}
	}

	return domain.AgentRevision{}, domain.ErrNotFound
}

func (r *memoryAgentRepository) MarkRevisionCorrupt(
	_ context.Context,
	agentName string,
	revisionID string,
	errorMessage string,
) error {
	for index, revision := range r.revisions {
		if revision.AgentName == agentName && revision.RevisionID == revisionID {
			r.revisions[index].Status = domain.AgentRevisionStatusCorrupt
			r.revisions[index].ErrorMessage = errorMessage

			return nil
		}
	}

	return domain.ErrNotFound
}

type memoryRuntimeDBManager struct {
	ensured map[string]bool
}

func (m *memoryRuntimeDBManager) EnsureAgent(_ context.Context, agentName string) error {
	if m.ensured == nil {
		m.ensured = make(map[string]bool)
	}
	m.ensured[agentName] = true

	return nil
}

func (m *memoryRuntimeDBManager) Runs(string) app.AgentRunRepository {
	return nil
}

func (m *memoryRuntimeDBManager) Events(string) app.RuntimeEventRepository {
	return nil
}

func (m *memoryRuntimeDBManager) Close(context.Context) error {
	return nil
}

func manualDefinition(prompt string) string {
	return `---
name: release-notes-helper
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: ["api.openai.com"]
---
` + prompt
}

func toolDefinition(prompt, command string) string {
	return `---
name: revisioned
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools:
  - name: fetch
    kind: custom_tool
    command: ` + command + `
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: ["api.github.com"]
---
` + prompt
}

func environmentDefinition(prompt string) string {
	return `---
name: env-agent
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
environment:
  variables:
    SHARED: from-literal
    REPORT_LIMIT: "10"
  files:
    - .env
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: []
---
` + prompt
}

func revisionEnvironmentToMap(environment []domain.RevisionEnvironment) map[string]string {
	values := make(map[string]string, len(environment))
	for _, entry := range environment {
		values[entry.Key] = entry.Value
	}

	return values
}
