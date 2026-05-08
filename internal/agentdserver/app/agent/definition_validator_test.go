package agent

import (
	"errors"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestNormalizeDefinitionRejectsExampleToolOutsideToolsDirectory(t *testing.T) {
	t.Parallel()

	definition := validExampleDefinition()
	definition.Tools[0].Command = "../fetch.py"

	_, err := NormalizeDefinition(definition)
	if !errors.Is(err, domain.ErrInvalidDefinition) {
		t.Fatalf("NormalizeDefinition error: got %v want ErrInvalidDefinition", err)
	}
}

func TestNormalizeDefinitionRejectsExampleToolRequiredSecrets(t *testing.T) {
	t.Parallel()

	definition := validExampleDefinition()
	definition.Tools[0].Env = []string{"REDDIT_CLIENT_SECRET"}

	_, err := NormalizeDefinition(definition)
	if !errors.Is(err, domain.ErrInvalidDefinition) {
		t.Fatalf("NormalizeDefinition error: got %v want ErrInvalidDefinition", err)
	}
}

func TestNormalizeDefinitionAcceptsExampleLocalTool(t *testing.T) {
	t.Parallel()

	if _, err := NormalizeDefinition(validExampleDefinition()); err != nil {
		t.Fatalf("NormalizeDefinition: %v", err)
	}
}

func validExampleDefinition() domain.AgentDefinition {
	return domain.AgentDefinition{
		Name:    "cybersecurity-reddit-watch",
		Enabled: true,
		Schedule: domain.Schedule{
			Type:       domain.ScheduleTypeCron,
			Expression: "0 8 * * *",
		},
		Vendor: domain.Vendor{Name: "openai", Model: "gpt-5.4-mini"},
		Tools: []domain.ToolPermission{{
			Kind:    domain.ToolKindLocalTool,
			Name:    "fetch_reddit",
			Command: "tools/fetch_reddit.py",
		}},
		Prompt:      "Watch public security posts.",
		SourcePath:  "examples/cybersecurity-reddit-watch/cybersecurity-reddit-watch.md",
		RawMarkdown: "definition",
	}
}
