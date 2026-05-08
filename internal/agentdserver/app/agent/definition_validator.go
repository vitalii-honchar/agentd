package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"

	"github.com/robfig/cron/v3"
)

type NormalizedDefinition struct {
	Definition domain.AgentDefinition
	Revision   string
}

func NormalizeDefinition(definition domain.AgentDefinition) (NormalizedDefinition, error) {
	if err := definition.Validate(); err != nil {
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
	if err := validateExampleLocalTools(definition); err != nil {
		return NormalizedDefinition{}, err
	}

	return NormalizedDefinition{
		Definition: definition,
		Revision:   hashDefinition(definition.RawMarkdown),
	}, nil
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

func validateExampleLocalTools(definition domain.AgentDefinition) error {
	sourcePath := filepath.ToSlash(strings.TrimSpace(definition.SourcePath))
	if !strings.HasPrefix(sourcePath, "examples/") {
		return nil
	}
	for _, tool := range definition.Tools {
		if tool.Kind != domain.ToolKindLocalTool {
			continue
		}
		command := filepath.ToSlash(strings.TrimSpace(tool.Command))
		if command == "" {
			return fmt.Errorf("%w: tools.command is required", domain.ErrInvalidDefinition)
		}
		if filepath.IsAbs(command) || strings.HasPrefix(command, "../") || strings.Contains(command, "/../") {
			return fmt.Errorf("%w: tools.command must stay inside the example folder", domain.ErrInvalidDefinition)
		}
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
	sum := sha256.Sum256([]byte(normalized))

	return hex.EncodeToString(sum[:])
}
