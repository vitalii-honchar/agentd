package app

import (
	"bytes"
	"testing"
)

func TestRootCommandDefaultsToClientModeBehavior(t *testing.T) {
	t.Parallel()

	client := &fakeQueryClient{listResponse: ListResponse{Agents: []AgentSummary{{
		Name:         "release-notes-helper",
		Enabled:      true,
		Status:       "active",
		ScheduleType: "manual",
	}}}}
	var out bytes.Buffer
	var errOut bytes.Buffer
	cmd := NewRootCommand(RootOptions{
		Config:      testCLIConfig(),
		QueryClient: client,
		Out:         &out,
		Err:         &errOut,
	})

	if err := executeTestCommand(t, cmd, "list"); err != nil {
		t.Fatalf("Execute list: %v stderr=%s", err, errOut.String())
	}
	if !client.listCalled {
		t.Fatal("list client was not called")
	}
	requireOutputContains(t, &out, "release-notes-helper")
}
