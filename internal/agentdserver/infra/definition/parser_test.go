package definition

import (
	"errors"
	"os"
	"testing"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

func TestParseMarkdownManualDefinition(t *testing.T) {
	t.Parallel()

	definition, err := ParseMarkdown("examples/release-notes-helper.md", `---
name: release-notes-helper
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5
tools: []
mcp_servers: []
access:
  filesystem:
    read: []
    write: []
  network:
    allow: ["api.openai.com"]
---
You summarize recent project changes into concise release notes.`)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}

	if definition.Name != "release-notes-helper" {
		t.Fatalf("name: got %q", definition.Name)
	}
	if !definition.Enabled {
		t.Fatal("enabled: got false want true")
	}
	if definition.Schedule.Type != domain.ScheduleTypeManual {
		t.Fatalf("schedule type: got %q want manual", definition.Schedule.Type)
	}
	if definition.Schedule.Expression != "" {
		t.Fatalf("manual expression: got %q want empty", definition.Schedule.Expression)
	}
	if definition.Vendor.Name != "openai" || definition.Vendor.Model != "gpt-5" {
		t.Fatalf("vendor: got %q/%q", definition.Vendor.Name, definition.Vendor.Model)
	}
	if definition.SourcePath != "examples/release-notes-helper.md" {
		t.Fatalf("source path: got %q", definition.SourcePath)
	}
	if definition.Prompt != "You summarize recent project changes into concise release notes." {
		t.Fatalf("prompt: got %q", definition.Prompt)
	}
	if got := definition.Access.Network.Allow; len(got) != 1 || got[0] != "api.openai.com" {
		t.Fatalf("network allow: got %#v", got)
	}
}

func TestParseMarkdownCronDefinitionWithToolsAndMCPServers(t *testing.T) {
	t.Parallel()

	definition, err := ParseMarkdown("daily-pr-review.md", `---
name: daily-pr-review
enabled: true
schedule:
  type: cron
  expression: "0 9 * * MON-FRI"
vendor:
  name: openai
  model: gpt-5
tools:
  - name: git
    kind: local_tool
    command: git
    args: ["status", "--short"]
    env: []
    read_paths: ["."]
    write_paths: []
mcp_servers:
  - name: github
    command: github-mcp-server
    args: []
    env: ["GITHUB_TOKEN"]
access:
  filesystem:
    read: ["."]
    write: []
  network:
    allow: ["api.openai.com", "api.github.com"]
---
Review open pull requests and identify issues that need attention.`)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}

	if definition.Schedule.Type != domain.ScheduleTypeCron {
		t.Fatalf("schedule type: got %q want cron", definition.Schedule.Type)
	}
	if definition.Schedule.Expression != "0 9 * * MON-FRI" {
		t.Fatalf("schedule expression: got %q", definition.Schedule.Expression)
	}
	if len(definition.Tools) != 1 {
		t.Fatalf("tools: got %d want 1", len(definition.Tools))
	}
	tool := definition.Tools[0]
	if tool.Name != "git" || tool.Kind != domain.ToolKindLocalTool || tool.Command != "git" {
		t.Fatalf("tool: %#v", tool)
	}
	if len(tool.Args) != 2 || tool.Args[0] != "status" || tool.Args[1] != "--short" {
		t.Fatalf("tool args: %#v", tool.Args)
	}
	if len(definition.MCPServers) != 1 {
		t.Fatalf("mcp servers: got %d want 1", len(definition.MCPServers))
	}
	server := definition.MCPServers[0]
	if server.Name != "github" || server.Command != "github-mcp-server" {
		t.Fatalf("mcp server: %#v", server)
	}
	if len(server.Env) != 1 || server.Env[0] != "GITHUB_TOKEN" {
		t.Fatalf("mcp env: %#v", server.Env)
	}
}

func TestParseMarkdownDefinitionWithInputsAndNestedToolNetwork(t *testing.T) {
	t.Parallel()

	definition, err := ParseMarkdown("examples/website-snapshot-analyst/website-snapshot-analyst.md", `---
name: website-snapshot-analyst
enabled: true
schedule:
  type: manual
vendor:
  name: openai
  model: gpt-5.4-mini
inputs:
  - name: url
    required: true
    description: Website URL to capture
tools:
  - name: capture_website
    kind: local_tool
    command: tools/capture_website.js
    timeout: 60s
    network:
      allow:
        - https://example.com
access:
  filesystem:
    read: ["fixtures/"]
    write: [".agentd-work/"]
  network:
    allow: ["https://example.com"]
---
Capture a screenshot and summarize the supplied website.`)
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}

	if len(definition.Inputs) != 1 {
		t.Fatalf("inputs: got %d want 1", len(definition.Inputs))
	}
	if input := definition.Inputs[0]; input.Name != "url" || !input.Required {
		t.Fatalf("input: %#v", input)
	}
	if len(definition.Tools) != 1 {
		t.Fatalf("tools: got %d want 1", len(definition.Tools))
	}
	tool := definition.Tools[0]
	if tool.Timeout != "60s" {
		t.Fatalf("tool timeout: got %q", tool.Timeout)
	}
	if len(tool.NetworkAllow) != 1 || tool.NetworkAllow[0] != "https://example.com" {
		t.Fatalf("tool network allow: %#v", tool.NetworkAllow)
	}
}

func TestParseMarkdownRejectsInvalidDefinition(t *testing.T) {
	t.Parallel()

	_, err := ParseMarkdown("bad.md", `---
name: Bad Name
schedule:
  type: manual
vendor:
  name: openai
  model: ""
---
`)
	if err == nil {
		t.Fatal("ParseMarkdown returned nil error")
	}
	if !errors.Is(err, domain.ErrInvalidDefinition) {
		t.Fatalf("ParseMarkdown error %v does not match ErrInvalidDefinition", err)
	}
}

func TestParseMarkdownRejectsMissingFrontMatter(t *testing.T) {
	t.Parallel()

	_, err := ParseMarkdown("bad.md", "Prompt without front matter")
	if err == nil {
		t.Fatal("ParseMarkdown returned nil error")
	}
}

func TestParseAIProductResearchExample(t *testing.T) {
	t.Parallel()

	body, err := os.ReadFile("../../../../examples/ai-product-research.md")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	definition, err := ParseMarkdown("examples/ai-product-research.md", string(body))
	if err != nil {
		t.Fatalf("ParseMarkdown: %v", err)
	}
	if definition.Name != "ai-product-research" {
		t.Fatalf("name: got %q", definition.Name)
	}
	if len(definition.Tools) != 2 {
		t.Fatalf("tools length: got %d want 2", len(definition.Tools))
	}
	if definition.Tools[0].Command != "uv" {
		t.Fatalf("tool command: got %q", definition.Tools[0].Command)
	}
	if len(definition.Tools[0].Env) != 5 {
		t.Fatalf("tool env allow-list: %#v", definition.Tools[0].Env)
	}
}
