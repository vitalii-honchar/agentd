package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	appagent "github.com/vitalii-honchar/agentd/internal/agentdserver/app/agent"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/definition"
)

func TestExampleDefinitionsParseInSmokeHarness(t *testing.T) {
	t.Parallel()

	examples := []string{
		"cybersecurity-reddit-watch",
		"hacker-news-builder-brief",
		"reddit-customer-pain-monitor",
		"product-hunt-launch-radar",
		"github-trending-engineering-radar",
		"developer-dependency-release-monitor",
		"ai-engineering-hiring-signal-monitor",
		"website-snapshot-analyst",
	}

	examplesRoot := filepath.Clean("../../examples")
	for _, name := range examples {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(examplesRoot, name, name+".md")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			parsed, err := definition.ParseMarkdown(path, string(body))
			if err != nil {
				t.Fatalf("ParseMarkdown %s: %v", path, err)
			}
			if parsed.Name != name {
				t.Fatalf("name: got %q want %q", parsed.Name, name)
			}
		})
	}
}

func TestExampleDefinitionsExposeContractedOutputSchemas(t *testing.T) {
	t.Parallel()

	examples := []string{
		"cybersecurity-reddit-watch",
		"hacker-news-builder-brief",
		"reddit-customer-pain-monitor",
		"product-hunt-launch-radar",
		"github-trending-engineering-radar",
		"developer-dependency-release-monitor",
		"ai-engineering-hiring-signal-monitor",
		"website-snapshot-analyst",
	}

	examplesRoot := filepath.Clean("../../examples")
	for _, name := range examples {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(examplesRoot, name, name+".md")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			parsed, err := definition.ParseMarkdown(path, string(body))
			if err != nil {
				t.Fatalf("ParseMarkdown %s: %v", path, err)
			}
			normalized, err := appagent.NormalizeDefinition(parsed)
			if err != nil {
				t.Fatalf("NormalizeDefinition %s: %v", path, err)
			}
			if normalized.Definition.Contract == nil {
				t.Fatalf("%s must define a contract", path)
			}
			var outputSchema struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			}
			if err := json.Unmarshal([]byte(normalized.Definition.Contract.OutputSchemaRaw), &outputSchema); err != nil {
				t.Fatalf("Unmarshal output schema: %v", err)
			}
			if len(outputSchema.Properties) == 0 || len(outputSchema.Required) == 0 {
				t.Fatalf("output schema must declare required structured properties: %s", normalized.Definition.Contract.OutputSchemaRaw)
			}
		})
	}
}
