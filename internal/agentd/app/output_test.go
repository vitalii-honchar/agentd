package app

import (
	"bytes"
	"strings"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentd/config"
)

func TestOutputWritesIndentedJSON(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	err := NewOutput(config.OutputJSON, &out).Write(ListResponse{
		Agents: []AgentSummary{{
			Name:         "release-notes-helper",
			Enabled:      true,
			Status:       "active",
			ScheduleType: "manual",
		}},
	})
	if err != nil {
		t.Fatalf("Write: %v", err)
	}

	const want = `{
  "agents": [
    {
      "name": "release-notes-helper",
      "enabled": true,
      "status": "active",
      "schedule_type": "manual"
    }
  ]
}
`
	if out.String() != want {
		t.Fatalf("json output:\n got: %q\nwant: %q", out.String(), want)
	}
}

func TestInspectTextOutputSnapshot(t *testing.T) {
	t.Parallel()

	client := &fakeQueryClient{agent: AgentDetail{
		Name:         "release-notes-helper",
		Status:       "active",
		ScheduleType: "manual",
		Revision:     "rev-1",
		VendorName:   "openai",
		VendorModel:  "gpt-5",
		LastRunID:    "run-1",
	}}
	var out bytes.Buffer
	cmd := NewInspectCommand(client, NewOutput(config.OutputText, &out))
	cmd.SetArgs([]string{"release-notes-helper"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	wantLines := []string{
		"name: release-notes-helper",
		"status: active",
		"schedule: manual",
		"vendor: openai/gpt-5",
		"revision: rev-1",
		"last_run: run-1",
	}
	for _, line := range wantLines {
		if !strings.Contains(out.String(), line) {
			t.Fatalf("output missing %q: %q", line, out.String())
		}
	}
}
