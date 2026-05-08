package runtime

import (
	"context"
	"errors"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestStopUseCaseRequestsCancellation(t *testing.T) {
	t.Parallel()

	manager := &fakeManager{run: domain.AgentRun{
		ID:        "run-1",
		AgentName: "release-notes-helper",
		Status:    domain.AgentRunStatusStopping,
	}}
	useCase := NewStopUseCase(manager)

	run, err := useCase.Stop(context.Background(), "release-notes-helper", "run-1")
	if err != nil {
		t.Fatalf("Stop: %v", err)
	}
	if run.Status != domain.AgentRunStatusStopping {
		t.Fatalf("status: got %q", run.Status)
	}
	if manager.stopRequest.AgentName != "release-notes-helper" || manager.stopRequest.RunID != "run-1" {
		t.Fatalf("stop request: %#v", manager.stopRequest)
	}
}

func TestStopUseCasePropagatesNotFound(t *testing.T) {
	t.Parallel()

	useCase := NewStopUseCase(&fakeManager{err: domain.ErrNotFound})
	_, err := useCase.Stop(context.Background(), "missing", "")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("Stop error: got %v want %v", err, domain.ErrNotFound)
	}
}
