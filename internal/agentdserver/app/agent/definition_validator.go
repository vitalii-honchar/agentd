package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"

	"agentd/internal/agentdserver/domain"

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

func hashDefinition(markdown string) string {
	normalized := strings.TrimSpace(strings.ReplaceAll(markdown, "\r\n", "\n"))
	sum := sha256.Sum256([]byte(normalized))

	return hex.EncodeToString(sum[:])
}
