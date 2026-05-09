package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"

	"github.com/robfig/cron/v3"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

type NormalizedDefinition struct {
	Definition domain.AgentDefinition
	Revision   string
}

func NormalizeDefinition(definition domain.AgentDefinition) (NormalizedDefinition, error) {
	if err := definition.Validate(); err != nil {
		return NormalizedDefinition{}, err
	}
	if err := validateContractSchemas(definition.Contract); err != nil {
		return NormalizedDefinition{}, err
	}
	if err := validateSchedule(definition.Schedule); err != nil {
		return NormalizedDefinition{}, err
	}
	if err := validatePermissionNames(definition.Tools, "tools"); err != nil {
		return NormalizedDefinition{}, err
	}
	if err := validatePermissionNames(definition.MCPServers, "mcp_servers"); err != nil {
		return NormalizedDefinition{}, err
	}
	if err := validateInputNames(definition.Inputs); err != nil {
		return NormalizedDefinition{}, err
	}
	if err := validateToolDefinitions(definition.Tools); err != nil {
		return NormalizedDefinition{}, err
	}
	if err := validateExampleToolPolicies(definition); err != nil {
		return NormalizedDefinition{}, err
	}
	definition.Contract = copyContract(definition.Contract)
	if definition.Contract != nil {
		definition.Contract.InputSchemaDigest = digestStrings(definition.Contract.InputSchemaRaw)
		definition.Contract.OutputSchemaDigest = digestStrings(definition.Contract.OutputSchemaRaw)
	}

	return NormalizedDefinition{
		Definition: definition,
		Revision:   hashDefinition(definition.RawMarkdown),
	}, nil
}

func validateContractSchemas(contract *domain.AgentContract) error {
	if contract == nil {
		return nil
	}
	if err := compileContractSchema(contract.InputSchemaRaw); err != nil {
		return fmt.Errorf("%w: contract.input: %v", domain.ErrInvalidContractSchema, err)
	}
	if err := compileContractSchema(contract.OutputSchemaRaw); err != nil {
		return fmt.Errorf("%w: contract.output: %v", domain.ErrInvalidContractSchema, err)
	}

	return nil
}

func compileContractSchema(schemaRaw string) error {
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	doc, err := jsonschema.UnmarshalJSON(strings.NewReader(schemaRaw))
	if err != nil {
		return err
	}
	if err := compiler.AddResource("schema.json", doc); err != nil {
		return err
	}
	_, err = compiler.Compile("schema.json")

	return err
}

func validateSchedule(schedule domain.Schedule) error {
	if schedule.Type != domain.ScheduleTypeCron {
		return nil
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow | cron.Descriptor)
	if _, err := parser.Parse(schedule.Expression); err != nil {
		return fmt.Errorf("%w: schedule.expression: %v", domain.ErrInvalidDefinition, err)
	}

	return nil
}

func validatePermissionNames(permissions []domain.ToolPermission, field string) error {
	seen := make(map[string]struct{}, len(permissions))
	for _, permission := range permissions {
		name := strings.TrimSpace(permission.Name)
		if name == "" {
			return fmt.Errorf("%w: %s.name is required", domain.ErrInvalidDefinition, field)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("%w: %s.name %q is duplicated", domain.ErrInvalidDefinition, field, name)
		}
		seen[name] = struct{}{}
	}

	return nil
}

func validateInputNames(inputs []domain.InputDefinition) error {
	seen := make(map[string]struct{}, len(inputs))
	for _, input := range inputs {
		name := strings.TrimSpace(input.Name)
		if name == "" {
			return fmt.Errorf("%w: inputs.name is required", domain.ErrInvalidDefinition)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("%w: inputs.name %q is duplicated", domain.ErrInvalidDefinition, name)
		}
		seen[name] = struct{}{}
	}

	return nil
}

func validateToolDefinitions(tools []domain.ToolPermission) error {
	for _, tool := range tools {
		command := filepath.ToSlash(strings.TrimSpace(tool.Command))
		switch tool.Kind {
		case domain.ToolKindCustomTool, domain.ToolKindLocalTool:
			if command == "" {
				return fmt.Errorf("%w: tools.command is required", domain.ErrInvalidDefinition)
			}
			if filepath.IsAbs(command) || strings.HasPrefix(command, "../") || strings.Contains(command, "/../") {
				return fmt.Errorf("%w: custom_tool command must stay inside the definition folder", domain.ErrInvalidDefinition)
			}
		case domain.ToolKindHostTool:
			if command == "" {
				return fmt.Errorf("%w: tools.command is required", domain.ErrInvalidDefinition)
			}
			if !filepath.IsAbs(command) && strings.Contains(command, "/") {
				return fmt.Errorf("%w: host_tool command must be a host executable name or absolute path", domain.ErrInvalidDefinition)
			}
		default:
			return fmt.Errorf("%w: tools.kind %q is not supported", domain.ErrInvalidDefinition, tool.Kind)
		}
	}

	return nil
}

func validateExampleToolPolicies(definition domain.AgentDefinition) error {
	sourcePath := filepath.ToSlash(strings.TrimSpace(definition.SourcePath))
	if !strings.HasPrefix(sourcePath, "examples/") {
		return nil
	}
	for _, tool := range definition.Tools {
		if tool.Kind != domain.ToolKindCustomTool && tool.Kind != domain.ToolKindLocalTool {
			continue
		}
		command := filepath.ToSlash(strings.TrimSpace(tool.Command))
		if !strings.HasPrefix(command, "tools/") {
			return fmt.Errorf("%w: tools.command must reference tools/", domain.ErrInvalidDefinition)
		}
		if len(tool.Env) > 0 {
			return fmt.Errorf("%w: example tools must not require environment secrets", domain.ErrInvalidDefinition)
		}
	}

	return nil
}

func hashDefinition(markdown string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(markdown, "\r\n", "\n"))
	return digestStrings(normalized)
}

func digestStrings(values ...string) string {
	normalized := strings.Join(values, "\x00")
	sum := sha256.Sum256([]byte(normalized))

	return hex.EncodeToString(sum[:])
}

func copyContract(contract *domain.AgentContract) *domain.AgentContract {
	if contract == nil {
		return nil
	}

	copied := *contract
	return &copied
}
