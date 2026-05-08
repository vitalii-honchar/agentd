package agentdclient_test

import (
	"testing"

	"github.com/vitalii-honchar/agentd/pkg/agentdclient"
)

func TestPublicClientImportSmoke(t *testing.T) {
	t.Parallel()

	client, err := agentdclient.New(agentdclient.Config{ServerURL: "http://127.0.0.1:18080"})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if client == nil {
		t.Fatal("client is nil")
	}

	_ = agentdclient.RunSummary{}
	_ = agentdclient.RunResult{}
	_ = agentdclient.LogsQuery{}
}
