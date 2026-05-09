package runtime

import (
	"errors"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestParseEnvFileSupportsCommentsQuotesAndDuplicateKeys(t *testing.T) {
	t.Parallel()

	entries, err := ParseEnvFile("example.env", []byte(`
# comment
API_KEY=first
EMPTY=
DOUBLE_QUOTED="hello world"
SINGLE_QUOTED='single value'
export USER_AGENT=agentd-example/1.0
API_KEY=second
`))
	if err != nil {
		t.Fatalf("ParseEnvFile: %v", err)
	}
	values := envEntriesToMap(entries)

	if values["API_KEY"] != "second" {
		t.Fatalf("API_KEY: got %q want second", values["API_KEY"])
	}
	if values["EMPTY"] != "" {
		t.Fatalf("EMPTY: got %q want empty", values["EMPTY"])
	}
	if values["DOUBLE_QUOTED"] != "hello world" {
		t.Fatalf("DOUBLE_QUOTED: got %q", values["DOUBLE_QUOTED"])
	}
	if values["SINGLE_QUOTED"] != "single value" {
		t.Fatalf("SINGLE_QUOTED: got %q", values["SINGLE_QUOTED"])
	}
	if values["USER_AGENT"] != "agentd-example/1.0" {
		t.Fatalf("USER_AGENT: got %q", values["USER_AGENT"])
	}
}

func TestParseEnvFileRejectsInvalidLines(t *testing.T) {
	t.Parallel()

	_, err := ParseEnvFile("bad.env", []byte("NOT A VALID ENV LINE\n"))
	if !errors.Is(err, domain.ErrInvalidDefinition) {
		t.Fatalf("ParseEnvFile error: got %v want ErrInvalidDefinition", err)
	}
}

func TestMergeEnvironmentPrecedence(t *testing.T) {
	t.Parallel()

	merged := MergeEnvironment(EnvironmentMergeInput{
		FileEntries: []EnvFileEntry{
			{Key: "TOKEN", Value: "from-file"},
			{Key: "SHARED", Value: "from-file"},
		},
		Variables: []EnvFileEntry{
			{Key: "SHARED", Value: "from-literal"},
			{Key: "GLOBAL", Value: "from-literal"},
		},
		ToolEnv: []EnvFileEntry{
			{Key: "GLOBAL", Value: "from-tool"},
			{Key: "TOOL_ONLY", Value: "tool-value"},
		},
	})
	values := envEntriesToMap(merged)

	if values["TOKEN"] != "from-file" {
		t.Fatalf("TOKEN: got %q", values["TOKEN"])
	}
	if values["SHARED"] != "from-literal" {
		t.Fatalf("SHARED: got %q", values["SHARED"])
	}
	if values["GLOBAL"] != "from-tool" {
		t.Fatalf("GLOBAL: got %q", values["GLOBAL"])
	}
	if values["TOOL_ONLY"] != "tool-value" {
		t.Fatalf("TOOL_ONLY: got %q", values["TOOL_ONLY"])
	}
}

func TestMaskEnvironmentValue(t *testing.T) {
	t.Parallel()

	if got := MaskEnvironmentValue("secret-value"); got != "********" {
		t.Fatalf("masked value: got %q", got)
	}
	if got := MaskEnvironmentValue(""); got != "" {
		t.Fatalf("empty masked value: got %q", got)
	}
}

func envEntriesToMap(entries []EnvFileEntry) map[string]string {
	values := make(map[string]string, len(entries))
	for _, entry := range entries {
		values[entry.Key] = entry.Value
	}

	return values
}
