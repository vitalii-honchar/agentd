package definition

import (
	"fmt"

	"agentd/internal/agentdserver/domain"
)

func ParseMarkdown(_ string, _ string) (domain.AgentDefinition, error) {
	return domain.AgentDefinition{}, fmt.Errorf("%w: parser not implemented", domain.ErrInvalidDefinition)
}
