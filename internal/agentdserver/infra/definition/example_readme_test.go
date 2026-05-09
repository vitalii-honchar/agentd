package definition

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRequiredExampleREADMEsDocumentZeroSetupUsage(t *testing.T) {
	t.Parallel()

	requiredPhrases := []string{
		"Install",
		"agentd apply",
		"agentd result <agent-name>",
		"agentd result <run-id>",
		"agentd logs",
		"API keys",
		"zero configuration",
	}

	examplesRoot := filepath.Clean("../../../../examples")
	for _, name := range requiredExampleNames {
		name := name
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			body, err := os.ReadFile(filepath.Join(examplesRoot, name, "README.md"))
			if err != nil {
				t.Fatalf("ReadFile README: %v", err)
			}
			text := string(body)
			for _, phrase := range requiredPhrases {
				if !strings.Contains(text, phrase) {
					t.Fatalf("README missing %q", phrase)
				}
			}
		})
	}
}
