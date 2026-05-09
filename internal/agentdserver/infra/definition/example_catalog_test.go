package definition

import (
	"os"
	"path/filepath"
	"testing"

	appagent "github.com/vitalii-honchar/agentd/internal/agentdserver/app/agent"
)

var requiredExampleNames = []string{
	"cybersecurity-reddit-watch",
	"hacker-news-builder-brief",
	"reddit-customer-pain-monitor",
	"product-hunt-launch-radar",
	"github-trending-engineering-radar",
	"developer-dependency-release-monitor",
	"ai-engineering-hiring-signal-monitor",
	"website-snapshot-analyst",
}

func TestRequiredExampleCatalogLayout(t *testing.T) {
	t.Parallel()

	examplesRoot := filepath.Clean("../../../../examples")
	for _, name := range requiredExampleNames {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			dir := filepath.Join(examplesRoot, name)
			assertRegularFile(t, filepath.Join(dir, name+".md"))
			assertRegularFile(t, filepath.Join(dir, "README.md"))
			assertDirectory(t, filepath.Join(dir, "tools"))
			if !pathExists(filepath.Join(dir, "sources")) && !pathExists(filepath.Join(dir, "fixtures")) {
				t.Fatalf("%s must include sources/ or fixtures/", dir)
			}
		})
	}
}

func TestRequiredExampleCatalogDefinesValidContracts(t *testing.T) {
	t.Parallel()

	examplesRoot := filepath.Clean("../../../../examples")
	for _, name := range requiredExampleNames {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			path := filepath.Join(examplesRoot, name, name+".md")
			body, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("ReadFile %s: %v", path, err)
			}
			parsed, err := ParseMarkdown(path, string(body))
			if err != nil {
				t.Fatalf("ParseMarkdown %s: %v", path, err)
			}
			if parsed.Contract == nil {
				t.Fatalf("%s must define contract.input and contract.output", path)
			}
			normalized, err := appagent.NormalizeDefinition(parsed)
			if err != nil {
				t.Fatalf("NormalizeDefinition %s: %v", path, err)
			}
			if normalized.Definition.Contract.InputSchemaDigest == "" ||
				normalized.Definition.Contract.OutputSchemaDigest == "" {
				t.Fatalf("contract digests were not computed: %#v", normalized.Definition.Contract)
			}
		})
	}
}

func assertRegularFile(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("%s is a directory, want regular file", path)
	}
}

func assertDirectory(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", path)
	}
}

func pathExists(path string) bool {
	_, err := os.Stat(path)

	return err == nil
}
