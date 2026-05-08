package app

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
)

func TestApplyCommandReadsFileAndCallsClient(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "agent.md")
	markdown := "---\nname: release-notes-helper\n---\nPrompt"
	if err := os.WriteFile(path, []byte(markdown), 0o600); err != nil {
		t.Fatalf("write definition: %v", err)
	}

	client := &fakeApplyClient{response: ApplyResponse{
		Outcome:        "created",
		Agent:          AgentDetail{Name: "release-notes-helper"},
		RevisionID:     "11111111-1111-4111-8111-111111111111",
		ArtifactPath:   "data/work/release-notes-helper/11111111-1111-4111-8111-111111111111",
		RevisionStatus: "finalized",
	}}
	var out bytes.Buffer
	cmd := NewApplyCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{path})

	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if client.request.SourcePath != path {
		t.Fatalf("source path: got %q want %q", client.request.SourcePath, path)
	}
	if client.request.Markdown != markdown {
		t.Fatalf("markdown: got %q", client.request.Markdown)
	}
	if !strings.Contains(out.String(), "APPLIED release-notes-helper") ||
		!strings.Contains(out.String(), "OUTCOME created") ||
		!strings.Contains(out.String(), "REVISION 11111111-1111-4111-8111-111111111111") ||
		!strings.Contains(out.String(), "ARTIFACT data/work/release-notes-helper/11111111-1111-4111-8111-111111111111") ||
		!strings.Contains(out.String(), "STATUS finalized") {
		t.Fatalf("output: got %q", out.String())
	}
}

func TestRootCommandWiresApplyCommand(t *testing.T) {
	t.Parallel()

	client := &fakeApplyClient{}
	cmd := NewRootCommand(RootOptions{
		Config: &config.Config{
			ServerURL:      config.DefaultServerURL,
			OutputFormat:   config.OutputText,
			RequestTimeout: config.DefaultRequestTimeout,
		},
		Client: client,
		Out:    &bytes.Buffer{},
		Err:    &bytes.Buffer{},
	})

	found := false
	for _, child := range cmd.Commands() {
		if child.Name() == "apply" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("apply command was not wired into root command")
	}
}

type fakeApplyClient struct {
	request  ApplyRequest
	response ApplyResponse
	err      error
}

func (f *fakeApplyClient) Apply(_ context.Context, request ApplyRequest) (ApplyResponse, error) {
	f.request = request
	if f.err != nil {
		return ApplyResponse{}, f.err
	}

	return f.response, nil
}
