package testutil

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TempDir(t *testing.T) string {
	t.Helper()

	dir, err := os.MkdirTemp("", "agentd-test-*")
	if err != nil {
		t.Fatalf("create temp dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.RemoveAll(dir)
	})

	return dir
}

func TempPath(t *testing.T, name string) string {
	t.Helper()

	return filepath.Join(TempDir(t), name)
}

func RequireNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func MustParseJSON(t *testing.T, raw string) any {
	t.Helper()

	var value any
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		t.Fatalf("parse JSON: %v\n%s", err, raw)
	}
	return value
}

func RequireJSONEqual(t *testing.T, expected string, actual string) {
	t.Helper()

	expectedValue := MustParseJSON(t, expected)
	actualValue := MustParseJSON(t, actual)
	if !reflect.DeepEqual(expectedValue, actualValue) {
		t.Fatalf("JSON mismatch\nexpected: %s\nactual:   %s", expected, actual)
	}
}

func RequireJSONSchemaValid(t *testing.T, schemaRaw string, instanceRaw string) {
	t.Helper()

	schema := compileJSONSchema(t, schemaRaw)
	instance, err := jsonschema.UnmarshalJSON(strings.NewReader(instanceRaw))
	if err != nil {
		t.Fatalf("parse JSON instance: %v\n%s", err, instanceRaw)
	}
	if err := schema.Validate(instance); err != nil {
		t.Fatalf("expected JSON instance to satisfy schema: %v\nschema: %s\ninstance: %s", err, schemaRaw, instanceRaw)
	}
}

func RequireJSONSchemaInvalid(t *testing.T, schemaRaw string, instanceRaw string) {
	t.Helper()

	schema := compileJSONSchema(t, schemaRaw)
	instance, err := jsonschema.UnmarshalJSON(strings.NewReader(instanceRaw))
	if err != nil {
		t.Fatalf("parse JSON instance: %v\n%s", err, instanceRaw)
	}
	if err := schema.Validate(instance); err == nil {
		t.Fatalf("expected JSON instance to fail schema validation\nschema: %s\ninstance: %s", schemaRaw, instanceRaw)
	}
}

func compileJSONSchema(t *testing.T, schemaRaw string) *jsonschema.Schema {
	t.Helper()

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaRaw))
	if err != nil {
		t.Fatalf("parse JSON schema: %v\n%s", err, schemaRaw)
	}
	if err := compiler.AddResource("schema.json", doc); err != nil {
		t.Fatalf("add JSON schema resource: %v", err)
	}
	schema, err := compiler.Compile("schema.json")
	if err != nil {
		t.Fatalf("compile JSON schema: %v\n%s", err, schemaRaw)
	}
	return schema
}

func HostToolDefinitionMarkdown(agentName string) string {
	if agentName == "" {
		agentName = "host-tool-agent"
	}

	return fmt.Sprintf(`---
name: %s
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools:
  - name: github_api
    kind: host_tool
    command: gh
    args: ["api", "search/repositories"]
access:
  filesystem:
    read: []
    write: []
  network:
    allow: ["api.github.com"]
---
Use the host GitHub CLI to inspect public repositories.`, agentName)
}
