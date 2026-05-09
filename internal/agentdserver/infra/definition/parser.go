package definition

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"

	"gopkg.in/yaml.v3"
)

func ParseMarkdown(sourcePath string, markdown string) (domain.AgentDefinition, error) {
	frontMatter, prompt, err := splitFrontMatter(markdown)
	if err != nil {
		return domain.AgentDefinition{}, err
	}

	var raw definitionFrontMatter
	if err := yaml.Unmarshal([]byte(frontMatter), &raw); err != nil {
		return domain.AgentDefinition{}, fmt.Errorf("%w: parse front matter: %v", domain.ErrInvalidDefinition, err)
	}

	definition := raw.toDomain(sourcePath, markdown, prompt)
	if err := validateContractSchemaText(definition.Contract); err != nil {
		return domain.AgentDefinition{}, err
	}
	if err := definition.Validate(); err != nil {
		return domain.AgentDefinition{}, err
	}

	return definition, nil
}

type definitionFrontMatter struct {
	Name        string                 `yaml:"name"`
	Enabled     *bool                  `yaml:"enabled"`
	Schedule    scheduleFrontMatter    `yaml:"schedule"`
	Vendor      vendorFrontMatter      `yaml:"vendor"`
	Environment environmentFrontMatter `yaml:"environment"`
	Inputs      []inputFrontMatter     `yaml:"inputs"`
	Contract    contractFrontMatter    `yaml:"contract"`
	Tools       []toolFrontMatter      `yaml:"tools"`
	MCPServers  []toolFrontMatter      `yaml:"mcp_servers"`
	Access      accessFrontMatter      `yaml:"access"`
}

type scheduleFrontMatter struct {
	Type       string `yaml:"type"`
	Expression string `yaml:"expression"`
}

type vendorFrontMatter struct {
	Name  string `yaml:"name"`
	Model string `yaml:"model"`
}

type environmentFrontMatter struct {
	Variables map[string]string `yaml:"variables"`
	Files     []string          `yaml:"files"`
}

type inputFrontMatter struct {
	Name        string `yaml:"name"`
	Required    bool   `yaml:"required"`
	Description string `yaml:"description"`
}

type contractFrontMatter struct {
	Input  string `yaml:"input"`
	Output string `yaml:"output"`
}

type toolFrontMatter struct {
	Name         string             `yaml:"name"`
	Kind         string             `yaml:"kind"`
	Command      string             `yaml:"command"`
	Args         []string           `yaml:"args"`
	Env          []string           `yaml:"env"`
	Timeout      string             `yaml:"timeout"`
	ReadPaths    []string           `yaml:"read_paths"`
	WritePaths   []string           `yaml:"write_paths"`
	NetworkAllow []string           `yaml:"network_allow"`
	Network      networkFrontMatter `yaml:"network"`
}

type accessFrontMatter struct {
	Filesystem filesystemFrontMatter `yaml:"filesystem"`
	Network    networkFrontMatter    `yaml:"network"`
}

type filesystemFrontMatter struct {
	Read  []string `yaml:"read"`
	Write []string `yaml:"write"`
}

type networkFrontMatter struct {
	Allow []string `yaml:"allow"`
}

func (f definitionFrontMatter) toDomain(
	sourcePath string,
	rawMarkdown string,
	prompt string,
) domain.AgentDefinition {
	enabled := true
	if f.Enabled != nil {
		enabled = *f.Enabled
	}

	definition := domain.AgentDefinition{
		Name:    strings.TrimSpace(f.Name),
		Enabled: enabled,
		Schedule: domain.Schedule{
			Type:       domain.ScheduleType(strings.TrimSpace(f.Schedule.Type)),
			Expression: strings.TrimSpace(f.Schedule.Expression),
		},
		Vendor: domain.Vendor{
			Name:  strings.TrimSpace(f.Vendor.Name),
			Model: strings.TrimSpace(f.Vendor.Model),
		},
		Environment: domain.DefinitionEnvironment{
			Variables: copyStringMap(f.Environment.Variables),
			Files:     copyStrings(f.Environment.Files),
		},
		Inputs:     make([]domain.InputDefinition, 0, len(f.Inputs)),
		Tools:      make([]domain.ToolPermission, 0, len(f.Tools)),
		MCPServers: make([]domain.ToolPermission, 0, len(f.MCPServers)),
		Access: domain.AccessPolicy{
			Filesystem: domain.FilesystemAccess{
				Read:  copyStrings(f.Access.Filesystem.Read),
				Write: copyStrings(f.Access.Filesystem.Write),
			},
			Network: domain.NetworkAccess{
				Allow: copyStrings(f.Access.Network.Allow),
			},
		},
		Prompt:      strings.TrimSpace(prompt),
		SourcePath:  sourcePath,
		RawMarkdown: rawMarkdown,
	}
	if strings.TrimSpace(f.Contract.Input) != "" || strings.TrimSpace(f.Contract.Output) != "" {
		definition.Contract = &domain.AgentContract{
			InputSchemaRaw:  f.Contract.Input,
			OutputSchemaRaw: f.Contract.Output,
		}
	}
	for _, input := range f.Inputs {
		definition.Inputs = append(definition.Inputs, input.toDomain())
	}
	for _, tool := range f.Tools {
		definition.Tools = append(definition.Tools, tool.toDomain(definition.Name, normalizeToolKind(tool.Kind)))
	}
	for _, server := range f.MCPServers {
		definition.MCPServers = append(
			definition.MCPServers,
			server.toDomain(definition.Name, domain.ToolKindMCPServer),
		)
	}

	return definition
}

func validateContractSchemaText(contract *domain.AgentContract) error {
	if contract == nil {
		return nil
	}
	if strings.TrimSpace(contract.InputSchemaRaw) == "" {
		return fmt.Errorf("%w: contract.input is required when contract is specified", domain.ErrInvalidDefinition)
	}
	if strings.TrimSpace(contract.OutputSchemaRaw) == "" {
		return fmt.Errorf("%w: contract.output is required when contract is specified", domain.ErrInvalidDefinition)
	}
	if !json.Valid([]byte(contract.InputSchemaRaw)) {
		return fmt.Errorf("%w: contract.input must be valid JSON schema text", domain.ErrInvalidDefinition)
	}
	if !json.Valid([]byte(contract.OutputSchemaRaw)) {
		return fmt.Errorf("%w: contract.output must be valid JSON schema text", domain.ErrInvalidDefinition)
	}

	return nil
}

func normalizeToolKind(kind string) domain.ToolKind {
	normalized := domain.ToolKind(strings.TrimSpace(kind))
	if normalized == domain.ToolKindLocalTool {
		return domain.ToolKindCustomTool
	}

	return normalized
}

func (i inputFrontMatter) toDomain() domain.InputDefinition {
	return domain.InputDefinition{
		Name:        strings.TrimSpace(i.Name),
		Required:    i.Required,
		Description: strings.TrimSpace(i.Description),
	}
}

func (t toolFrontMatter) toDomain(agentName string, kind domain.ToolKind) domain.ToolPermission {
	networkAllow := copyStrings(t.NetworkAllow)
	if len(networkAllow) == 0 {
		networkAllow = copyStrings(t.Network.Allow)
	}

	return domain.ToolPermission{
		AgentName:    agentName,
		Kind:         kind,
		Name:         strings.TrimSpace(t.Name),
		Command:      strings.TrimSpace(t.Command),
		Args:         copyStrings(t.Args),
		Env:          copyStrings(t.Env),
		Timeout:      strings.TrimSpace(t.Timeout),
		ReadPaths:    copyStrings(t.ReadPaths),
		WritePaths:   copyStrings(t.WritePaths),
		NetworkAllow: networkAllow,
	}
}

func splitFrontMatter(markdown string) (string, string, error) {
	normalized := strings.ReplaceAll(markdown, "\r\n", "\n")
	lines := strings.SplitAfter(normalized, "\n")
	if len(lines) < 3 || strings.TrimSpace(lines[0]) != "---" {
		return "", "", fmt.Errorf("%w: missing front matter", domain.ErrInvalidDefinition)
	}

	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[1:i], ""), strings.Join(lines[i+1:], ""), nil
		}
	}

	return "", "", fmt.Errorf("%w: unclosed front matter", domain.ErrInvalidDefinition)
}

func copyStrings(values []string) []string {
	if values == nil {
		return nil
	}
	copied := make([]string, len(values))
	copy(copied, values)

	return copied
}

func copyStringMap(values map[string]string) map[string]string {
	if values == nil {
		return nil
	}
	copied := make(map[string]string, len(values))
	for key, value := range values {
		copied[strings.TrimSpace(key)] = strings.TrimSpace(value)
	}

	return copied
}
