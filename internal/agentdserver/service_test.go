package agentdserver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/config"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
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

func TestRecoverRevisionArtifactsMarksPendingAndMissingArtifactsCorrupt(t *testing.T) {
	t.Parallel()

	workRoot := t.TempDir()
	artifactPath := filepath.Join(workRoot, "release-notes-helper", "missing-file")
	if err := os.MkdirAll(artifactPath, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	repo := &memoryRevisionArtifactRepository{
		agents: []domain.Agent{{Name: "release-notes-helper"}},
		revisions: map[string][]domain.AgentRevision{
			"release-notes-helper": {
				{
					AgentName:  "release-notes-helper",
					RevisionID: "pending-revision",
					Status:     domain.AgentRevisionStatusPending,
				},
				{
					AgentName:    "release-notes-helper",
					RevisionID:   "missing-file",
					Status:       domain.AgentRevisionStatusFinalized,
					ArtifactPath: artifactPath,
					ArtifactFiles: []domain.RevisionArtifactFile{{
						ArtifactRelativePath: "tools/fetch.py",
					}},
				},
			},
		},
	}

	if err := recoverRevisionArtifacts(context.Background(), repo, workRoot); err != nil {
		t.Fatalf("recoverRevisionArtifacts: %v", err)
	}
	if len(repo.corrupt) != 2 {
		t.Fatalf("corrupt revisions: got %#v", repo.corrupt)
	}
	if repo.corrupt[0].revisionID != "pending-revision" ||
		!strings.Contains(repo.corrupt[0].message, "interrupted") {
		t.Fatalf("pending corruption: %#v", repo.corrupt[0])
	}
	if repo.corrupt[1].revisionID != "missing-file" ||
		!strings.Contains(repo.corrupt[1].message, "tools/fetch.py") {
		t.Fatalf("missing artifact corruption: %#v", repo.corrupt[1])
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

type memoryRevisionArtifactRepository struct {
	agents    []domain.Agent
	revisions map[string][]domain.AgentRevision
	corrupt   []revisionCorruption
}

type revisionCorruption struct {
	agentName  string
	revisionID string
	message    string
}

func (r *memoryRevisionArtifactRepository) List(context.Context) ([]domain.Agent, error) {
	return append([]domain.Agent(nil), r.agents...), nil
}

func (r *memoryRevisionArtifactRepository) ListRevisions(_ context.Context, agentName string) ([]domain.AgentRevision, error) {
	return append([]domain.AgentRevision(nil), r.revisions[agentName]...), nil
}

func (r *memoryRevisionArtifactRepository) MarkRevisionCorrupt(
	_ context.Context,
	agentName string,
	revisionID string,
	errorMessage string,
) error {
	r.corrupt = append(r.corrupt, revisionCorruption{
		agentName:  agentName,
		revisionID: revisionID,
		message:    errorMessage,
	})

	return nil
}
