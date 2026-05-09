package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
)

type ExecuteUseCase struct {
	agents    app.AgentRepository
	revisions app.AgentRevisionRepository
	manager   Manager
}

func NewExecuteUseCase(agents app.AgentRepository, manager Manager) *ExecuteUseCase {
	revisions, _ := agents.(app.AgentRevisionRepository)

	return &ExecuteUseCase{agents: agents, revisions: revisions, manager: manager}
}

func (u *ExecuteUseCase) Execute(ctx context.Context, agentSelector string, inputs map[string]string) (domain.AgentRun, error) {
	return u.ExecuteWithRuntimeInput(ctx, agentSelector, domain.RuntimeInput{
		LegacyInputs: inputs,
		Source:       domain.RuntimeInputSourcePublicClient,
	})
}

func (u *ExecuteUseCase) ExecuteWithRuntimeInput(
	ctx context.Context,
	agentSelector string,
	input domain.RuntimeInput,
) (domain.AgentRun, error) {
	agentName, revisionID := splitAgentRevisionSelector(agentSelector)
	agent, err := u.agents.FindByName(ctx, agentName)
	if err != nil {
		return domain.AgentRun{}, err
	}
	if err := agent.CanExecute(); err != nil {
		return domain.AgentRun{}, err
	}
	revision, hasRevision, err := u.resolveRevision(ctx, agent.Name, revisionID)
	if err != nil {
		return domain.AgentRun{}, err
	}
	if hasRevision {
		agent = agentFromRevision(agent, revision)
	}
	if err := validateRuntimeInput(agent.Contract, input); err != nil {
		return domain.AgentRun{}, err
	}

	return u.manager.Execute(ctx, ExecuteRequest{
		Agent:        agent,
		Revision:     revision,
		Trigger:      domain.RunTriggerManual,
		Inputs:       input.LegacyInputs,
		RuntimeInput: input,
	})
}

func (u *ExecuteUseCase) resolveRevision(
	ctx context.Context,
	agentName string,
	revisionID string,
) (domain.AgentRevision, bool, error) {
	if u.revisions == nil {
		return domain.AgentRevision{}, false, nil
	}
	var (
		revision domain.AgentRevision
		err      error
	)
	if revisionID != "" {
		revision, err = u.revisions.FindRevisionByID(ctx, agentName, revisionID)
	} else {
		revision, err = u.revisions.FindLatestFinalizedRevision(ctx, agentName)
	}
	if err != nil {
		if revisionID == "" && errors.Is(err, domain.ErrNotFound) {
			return domain.AgentRevision{}, false, nil
		}

		return domain.AgentRevision{}, false, err
	}
	if revision.Status != domain.AgentRevisionStatusFinalized {
		return domain.AgentRevision{}, false, fmt.Errorf("%w: revision %s is %s", domain.ErrInvalidState, revision.RevisionID, revision.Status)
	}

	return revision, true, nil
}

func splitAgentRevisionSelector(selector string) (string, string) {
	agentName, revisionID, ok := strings.Cut(selector, ":")
	if !ok {
		return selector, ""
	}

	return agentName, revisionID
}

func agentFromRevision(agent domain.Agent, revision domain.AgentRevision) domain.Agent {
	agent.Revision = revision.RevisionID
	agent.Prompt = revision.Prompt
	agent.Vendor = revision.Vendor
	agent.Schedule = revision.Schedule
	agent.Contract = contractFromRevision(revision)
	agent.Tools = toolsFromRevision(revision.Tools)

	return agent
}

func validateRuntimeInput(contract *domain.AgentContract, input domain.RuntimeInput) error {
	if contract == nil {
		return nil
	}
	raw, err := runtimeInputJSON(input)
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrContractInputInvalid, err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	schemaDoc, err := jsonschema.UnmarshalJSON(strings.NewReader(contract.InputSchemaRaw))
	if err != nil {
		return fmt.Errorf("%w: contract.input: %v", domain.ErrInvalidContractSchema, err)
	}
	if err := compiler.AddResource("contract-input.json", schemaDoc); err != nil {
		return fmt.Errorf("%w: contract.input: %v", domain.ErrInvalidContractSchema, err)
	}
	schema, err := compiler.Compile("contract-input.json")
	if err != nil {
		return fmt.Errorf("%w: contract.input: %v", domain.ErrInvalidContractSchema, err)
	}
	value, err := jsonschema.UnmarshalJSON(strings.NewReader(string(raw)))
	if err != nil {
		return fmt.Errorf("%w: %v", domain.ErrContractInputInvalid, err)
	}
	if err := schema.Validate(value); err != nil {
		return fmt.Errorf("%w: %v", domain.ErrContractInputInvalid, err)
	}

	return nil
}

func runtimeInputJSON(input domain.RuntimeInput) (json.RawMessage, error) {
	if len(input.RawJSON) > 0 {
		if !json.Valid(input.RawJSON) {
			return nil, fmt.Errorf("input must be valid JSON")
		}

		return input.RawJSON, nil
	}
	if input.LegacyInputs != nil {
		body, err := json.Marshal(input.LegacyInputs)
		if err != nil {
			return nil, err
		}

		return body, nil
	}

	return json.RawMessage(`{}`), nil
}

func contractFromRevision(revision domain.AgentRevision) *domain.AgentContract {
	if revision.ContractInputSchemaRaw == "" &&
		revision.ContractOutputSchemaRaw == "" &&
		revision.ContractInputSchemaDigest == "" &&
		revision.ContractOutputSchemaDigest == "" {
		return nil
	}

	return &domain.AgentContract{
		InputSchemaRaw:      revision.ContractInputSchemaRaw,
		OutputSchemaRaw:     revision.ContractOutputSchemaRaw,
		InputSchemaDigest:   revision.ContractInputSchemaDigest,
		OutputSchemaDigest:  revision.ContractOutputSchemaDigest,
		CreatedFromRevision: revision.RevisionID,
	}
}

func toolsFromRevision(revisionTools []domain.RevisionTool) []domain.ToolPermission {
	tools := make([]domain.ToolPermission, 0, len(revisionTools))
	for _, revisionTool := range revisionTools {
		command := revisionTool.OriginalCommand
		if revisionTool.Kind == domain.ToolKindCustomTool && revisionTool.RewrittenCommand != "" {
			command = revisionTool.RewrittenCommand
		}
		if revisionTool.Kind == domain.ToolKindHostTool && revisionTool.HostCommand != "" {
			command = revisionTool.HostCommand
		}
		tools = append(tools, domain.ToolPermission{
			AgentName:    revisionTool.AgentName,
			Kind:         revisionTool.Kind,
			Name:         revisionTool.Name,
			Command:      command,
			Args:         append([]string(nil), revisionTool.Args...),
			Env:          append([]string(nil), revisionTool.Env...),
			Timeout:      revisionTool.Timeout,
			ReadPaths:    append([]string(nil), revisionTool.ReadPaths...),
			WritePaths:   append([]string(nil), revisionTool.WritePaths...),
			NetworkAllow: append([]string(nil), revisionTool.NetworkAllow...),
		})
	}

	return tools
}
