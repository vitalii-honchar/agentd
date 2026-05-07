package app

import (
	"context"
	"io"
	"time"

	"agentd/internal/agentdserver/domain"
)

type AgentRepository interface {
	Save(ctx context.Context, agent domain.Agent, tools []domain.ToolPermission, mcpServers []domain.ToolPermission) error
	FindByName(ctx context.Context, name string) (domain.Agent, error)
	List(ctx context.Context) ([]domain.Agent, error)
}

type RuntimeDBManager interface {
	EnsureAgent(ctx context.Context, agentName string) error
	Runs(agentName string) AgentRunRepository
	Events(agentName string) RuntimeEventRepository
	Close(ctx context.Context) error
}

type AgentRunRepository interface {
	Create(ctx context.Context, run domain.AgentRun) error
	Update(ctx context.Context, run domain.AgentRun) error
	FindByID(ctx context.Context, runID string) (domain.AgentRun, error)
	FindLatest(ctx context.Context) (domain.AgentRun, error)
	FindActive(ctx context.Context) (domain.AgentRun, error)
	ListActive(ctx context.Context) ([]domain.AgentRun, error)
}

type RuntimeEventRepository interface {
	Append(ctx context.Context, event domain.RuntimeEvent) error
	ListByRun(ctx context.Context, runID string, limit int) ([]domain.RuntimeEvent, error)
	ListRecent(ctx context.Context, limit int) ([]domain.RuntimeEvent, error)
}

type RunLogFactory interface {
	Create(ctx context.Context, agentName, runID string) (RunLogWriter, error)
}

type RunLogWriter interface {
	io.WriteCloser
	Path() string
}

type RunLogReader interface {
	Read(ctx context.Context, query LogQuery) ([]LogEntry, error)
}

type LogQuery struct {
	AgentName string
	RunID     string
	LogPath   string
	Tail      int
}

type LogEntry struct {
	Timestamp time.Time
	RunID     string
	Line      string
}
