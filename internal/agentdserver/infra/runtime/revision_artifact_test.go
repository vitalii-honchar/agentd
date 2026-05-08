package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestRevisionArtifactServiceCopiesCustomToolAndDeclaredReadFiles(t *testing.T) {
	t.Parallel()

	sourceDir := t.TempDir()
	writeArtifactSource(t, sourceDir, "tools/fetch.py", []byte("#!/usr/bin/env python3\nprint('ok')\n"), 0o755)
	writeArtifactSource(t, sourceDir, "fixtures/input.json", []byte(`{"ok":true}`), 0o644)
	service, err := NewRevisionArtifactService(filepath.Join(t.TempDir(), "work"))
	if err != nil {
		t.Fatalf("NewRevisionArtifactService: %v", err)
	}
	definition := artifactTestDefinition(sourceDir)
	revisionID := "44444444-4444-4444-8444-444444444444"

	result, err := service.Create(context.Background(), RevisionArtifactRequest{
		Definition: definition,
		RevisionID: revisionID,
		CreatedAt:  time.Date(2026, 5, 8, 13, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	artifactPath := filepath.Join(service.workRoot, definition.Name, revisionID)
	if result.Revision.ArtifactPath != artifactPath {
		t.Fatalf("artifact path: got %q want %q", result.Revision.ArtifactPath, artifactPath)
	}
	assertArtifactFile(t, artifactPath, "tools/fetch.py", 0o755)
	assertArtifactFile(t, artifactPath, "fixtures/input.json", 0o644)
	if len(result.Revision.Tools) != 1 {
		t.Fatalf("revision tools: got %d want 1", len(result.Revision.Tools))
	}
	tool := result.Revision.Tools[0]
	if tool.Kind != domain.ToolKindCustomTool {
		t.Fatalf("tool kind: got %q", tool.Kind)
	}
	if tool.OriginalCommand != "tools/fetch.py" {
		t.Fatalf("original command: got %q", tool.OriginalCommand)
	}
	if tool.RewrittenCommand != filepath.Join(artifactPath, "tools/fetch.py") {
		t.Fatalf("rewritten command: got %q", tool.RewrittenCommand)
	}
	if len(tool.CopiedFiles) != 2 {
		t.Fatalf("copied files: %#v", tool.CopiedFiles)
	}
	assertArtifactManifestEntry(t, result.Revision.ArtifactFiles, "tools/fetch.py", sha256Hex([]byte("#!/usr/bin/env python3\nprint('ok')\n")))
	assertArtifactManifestEntry(t, result.Revision.ArtifactFiles, "fixtures/input.json", sha256Hex([]byte(`{"ok":true}`)))
}

func artifactTestDefinition(sourceDir string) domain.AgentDefinition {
	return domain.AgentDefinition{
		Name:    "artifact-agent",
		Enabled: true,
		Schedule: domain.Schedule{
			Type: domain.ScheduleTypeManual,
		},
		Vendor: domain.Vendor{Name: "openai", Model: "gpt-5"},
		Tools: []domain.ToolPermission{{
			Kind:      domain.ToolKindCustomTool,
			Name:      "fetch",
			Command:   "tools/fetch.py",
			ReadPaths: []string{"fixtures/input.json"},
		}},
		Prompt:      "Use the copied tool.",
		SourcePath:  filepath.Join(sourceDir, "artifact-agent.md"),
		RawMarkdown: "definition",
	}
}

func writeArtifactSource(t *testing.T, root, relative string, body []byte, mode os.FileMode) {
	t.Helper()

	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, body, mode); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
}

func assertArtifactFile(t *testing.T, artifactPath, relative string, wantMode os.FileMode) {
	t.Helper()

	info, err := os.Stat(filepath.Join(artifactPath, relative))
	if err != nil {
		t.Fatalf("stat artifact file %s: %v", relative, err)
	}
	if got := info.Mode().Perm(); got != wantMode {
		t.Fatalf("mode for %s: got %o want %o", relative, got, wantMode)
	}
}

func assertArtifactManifestEntry(t *testing.T, files []domain.RevisionArtifactFile, relative, wantSHA string) {
	t.Helper()

	for _, file := range files {
		if file.ArtifactRelativePath == relative {
			if file.SHA256 != wantSHA {
				t.Fatalf("sha for %s: got %q want %q", relative, file.SHA256, wantSHA)
			}

			return
		}
	}
	t.Fatalf("manifest missing %s in %#v", relative, files)
}

func sha256Hex(body []byte) string {
	sum := sha256.Sum256(body)

	return hex.EncodeToString(sum[:])
}
