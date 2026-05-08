package httpclient

import (
	"context"
	"fmt"

	"github.com/vitalii-honchar/agentd/internal/agentd/app"
	"github.com/vitalii-honchar/agentd/internal/agentd/config"
	"github.com/vitalii-honchar/agentd/pkg/agentdclient"
)

type Client struct {
	client *agentdclient.Client
}

func New(cfg *config.Config) (*Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	client, err := agentdclient.New(agentdclient.Config{
		ServerURL: cfg.ServerURL,
		Timeout:   cfg.RequestTimeout,
	})
	if err != nil {
		return nil, err
	}

	return &Client{client: client}, nil
}

func (c *Client) Health(ctx context.Context) error {
	return c.client.Health(ctx)
}

func toAppAgentDetail(agent agentdclient.AgentDetail) app.AgentDetail {
	return app.AgentDetail{
		Name:         agent.Name,
		Enabled:      agent.Enabled,
		Status:       agent.Status,
		ScheduleType: agent.ScheduleType,
		Revision:     agent.Revision,
		VendorName:   agent.VendorName,
		VendorModel:  agent.VendorModel,
		LastRunID:    agent.LastRunID,
		RecentError:  agent.RecentError,
	}
}

func toAppAgentSummary(agent agentdclient.AgentSummary) app.AgentSummary {
	return app.AgentSummary{
		Name:          agent.Name,
		Enabled:       agent.Enabled,
		Status:        agent.Status,
		ScheduleType:  agent.ScheduleType,
		LastRunStatus: agent.LastRunStatus,
	}
}

func toAppRevisionSummary(revision agentdclient.RevisionSummary) app.RevisionSummary {
	return app.RevisionSummary{
		RevisionID:   revision.RevisionID,
		Status:       revision.Status,
		CreatedAt:    revision.CreatedAt,
		Latest:       revision.Latest,
		SourcePath:   revision.SourcePath,
		ArtifactPath: revision.ArtifactPath,
		FinalizedAt:  revision.FinalizedAt,
		ErrorMessage: revision.ErrorMessage,
	}
}

func toAppRunResponse(run agentdclient.RunSummary) app.RunResponse {
	return app.RunResponse{
		RunID:         run.RunID,
		AgentName:     run.AgentName,
		AgentRevision: run.AgentRevision,
		Status:        run.Status,
	}
}

func toAppRunSummary(run agentdclient.RunSummary) app.RunSummary {
	return app.RunSummary{
		RunID:         run.RunID,
		AgentName:     run.AgentName,
		AgentRevision: run.AgentRevision,
		Status:        run.Status,
		Trigger:       run.Trigger,
		StartedAt:     run.StartedAt,
		CompletedAt:   run.CompletedAt,
	}
}

func toAppRunResult(result agentdclient.RunResult) app.RunResult {
	mapped := app.RunResult{
		RunSummary:    toAppRunSummary(result.RunSummary),
		Result:        result.Result,
		ResultSummary: result.ResultSummary,
	}
	if result.Failure != nil {
		mapped.Failure = &app.Failure{
			Code:    result.Failure.Code,
			Message: result.Failure.Message,
		}
	}

	return mapped
}
