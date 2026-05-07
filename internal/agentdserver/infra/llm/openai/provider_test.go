package openai

import (
	"context"
	"strings"
	"testing"

	appruntime "agentd/internal/agentdserver/app/runtime"

	openaisdk "github.com/openai/openai-go/v3"
)

func TestNewProviderRequiresAPIKey(t *testing.T) {
	t.Parallel()

	_, err := NewProvider(Config{})
	if err == nil {
		t.Fatal("NewProvider returned nil error")
	}
	if strings.Contains(err.Error(), "sk-") {
		t.Fatalf("error should not include secret-like value: %v", err)
	}
}

func TestNewProviderAcceptsAPIKeyWithoutNetworkCall(t *testing.T) {
	t.Parallel()

	provider, err := NewProvider(Config{APIKey: "test-api-key"})
	if err != nil {
		t.Fatalf("NewProvider: %v", err)
	}
	if provider.Name() != ProviderName {
		t.Fatalf("Name: got %q want %q", provider.Name(), ProviderName)
	}
}

func TestExecuteValidatesRequestBeforeNetworkCall(t *testing.T) {
	t.Parallel()

	provider := NewProviderWithClient(openaisdk.Client{})

	tests := map[string]appruntime.ProviderRequest{
		"missing model": {
			Prompt: "hello",
		},
		"missing prompt": {
			Model: "gpt-5",
		},
	}
	for name, request := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := provider.Execute(context.Background(), request)
			if err == nil {
				t.Fatal("Execute returned nil error")
			}
		})
	}
}
