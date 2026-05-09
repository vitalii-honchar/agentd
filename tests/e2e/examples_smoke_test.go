package e2e

import (
	"os"
	"path/filepath"
	"testing"

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
