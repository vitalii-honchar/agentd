package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"

	"github.com/google/uuid"
)

type Manager struct {
	runtimeDBs app.RuntimeDBManager
	logs       app.RunLogFactory
	isolation  *IsolationBuilder
	providers  map[string]appruntime.Provider
	now        func() time.Time

	mu     sync.Mutex
	active map[string]*activeRun
}

type activeRun struct {
	run       domain.AgentRun
	cancel    context.CancelFunc
	completed chan struct{}
}

var _ appruntime.Manager = (*Manager)(nil)

func NewManager(
	runtimeDBs app.RuntimeDBManager,
	logs app.RunLogFactory,
	isolation *IsolationBuilder,
	providers []appruntime.Provider,
) (*Manager, error) {
	if runtimeDBs == nil {
		return nil, fmt.Errorf("runtime db manager is required")
	}
	if logs == nil {
		return nil, fmt.Errorf("run log factory is required")
	}
	if isolation == nil {
		return nil, fmt.Errorf("isolation builder is required")
	}

	providerMap := make(map[string]appruntime.Provider, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		providerMap[provider.Name()] = provider
	}

	return &Manager{
		runtimeDBs: runtimeDBs,
		logs:       logs,
		isolation:  isolation,
		providers:  providerMap,
		now:        func() time.Time { return time.Now().UTC() },
		active:     make(map[string]*activeRun),
	}, nil
}

func (m *Manager) Execute(
	ctx context.Context,
	request appruntime.ExecuteRequest,
) (domain.AgentRun, error) {
	provider, ok := m.providers[request.Agent.Vendor.Name]
	if !ok {
		return domain.AgentRun{}, fmt.Errorf("%w: %s", domain.ErrUnsupportedVendor, request.Agent.Vendor.Name)
	}
	if err := m.runtimeDBs.EnsureAgent(ctx, request.Agent.Name); err != nil {
		return domain.AgentRun{}, err
	}

	m.mu.Lock()
	for _, active := range m.active {
		if active.run.AgentName == request.Agent.Name {
			m.mu.Unlock()

			return domain.AgentRun{}, domain.ErrRunAlreadyActive
		}
	}
	m.mu.Unlock()

	runID := uuid.NewString()
	runCtx, cancel := context.WithCancel(context.Background())
	env, err := m.isolation.Build(request.Agent, runID)
	if err != nil {
		cancel()

		return domain.AgentRun{}, err
	}
	logWriter, err := m.logs.Create(ctx, request.Agent.Name, runID)
	if err != nil {
		cancel()

		return domain.AgentRun{}, err
	}

	startedAt := m.now()
	run := domain.AgentRun{
		ID:            runID,
		AgentName:     request.Agent.Name,
		AgentRevision: request.Agent.Revision,
		Trigger:       request.Trigger,
		Status:        domain.AgentRunStatusRunning,
		StartedAt:     &startedAt,
		DueAt:         request.DueAt,
		WorkDir:       env.WorkDir,
		LogPath:       logWriter.Path(),
	}
	repo := m.runtimeDBs.Runs(request.Agent.Name)
	if repo == nil {
		_ = logWriter.Close()
		cancel()

		return domain.AgentRun{}, fmt.Errorf("run repository is required for agent %s", request.Agent.Name)
	}
	if err := repo.Create(ctx, run); err != nil {
		_ = logWriter.Close()
		cancel()

		return domain.AgentRun{}, err
	}

	active := &activeRun{run: run, cancel: cancel, completed: make(chan struct{})}
	m.mu.Lock()
	m.active[run.ID] = active
	m.mu.Unlock()

	go m.runProvider(runCtx, provider, request.Agent, run, logWriter, active)

	return run, nil
}

func (m *Manager) Stop(ctx context.Context, request appruntime.StopRequest) (domain.AgentRun, error) {
	active, err := m.findActive(request.AgentName, request.RunID)
	if err != nil {
		return domain.AgentRun{}, err
	}

	stopAt := m.now()
	run := active.run
	run.Status = domain.AgentRunStatusStopping
	run.StopRequestedAt = &stopAt
	m.setActiveRun(run)
	if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
		if err := repo.Update(ctx, run); err != nil {
			return domain.AgentRun{}, err
		}
	}
	active.cancel()

	return run, nil
}

func (m *Manager) Recover(ctx context.Context) (appruntime.RecoveryResult, error) {
	now := m.now()
	activeRuns := m.ActiveRunsSnapshot()
	for _, run := range activeRuns {
		run.Status = domain.AgentRunStatusInterrupted
		run.CompletedAt = &now
		if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
			if err := repo.Update(ctx, run); err != nil {
				return appruntime.RecoveryResult{}, err
			}
		}
		m.removeActive(run.ID)
	}

	return appruntime.RecoveryResult{InterruptedRuns: activeRuns, RecoveredAt: now}, nil
}

func (m *Manager) ActiveRuns(context.Context) ([]domain.AgentRun, error) {
	return m.ActiveRunsSnapshot(), nil
}

func (m *Manager) ActiveRunsSnapshot() []domain.AgentRun {
	m.mu.Lock()
	defer m.mu.Unlock()

	runs := make([]domain.AgentRun, 0, len(m.active))
	for _, active := range m.active {
		runs = append(runs, active.run)
	}

	return runs
}

func (m *Manager) runProvider(
	ctx context.Context,
	provider appruntime.Provider,
	agent domain.Agent,
	run domain.AgentRun,
	logWriter app.RunLogWriter,
	active *activeRun,
) {
	defer close(active.completed)
	defer logWriter.Close()
	defer m.removeActive(run.ID)

	response, err := provider.Execute(ctx, appruntime.ProviderRequest{
		RunID:     run.ID,
		AgentName: agent.Name,
		Model:     agent.Vendor.Model,
		Prompt:    agent.Prompt,
	})
	completedAt := m.now()
	run.CompletedAt = &completedAt
	if err != nil {
		if errors.Is(err, context.Canceled) {
			run.Status = domain.AgentRunStatusStopped
		} else {
			run.Status = domain.AgentRunStatusFailed
			run.ErrorCode = "provider_error"
			run.ErrorMessage = err.Error()
		}
	} else {
		run.Status = domain.AgentRunStatusCompleted
		run.ProviderRequestID = response.RequestID
		if response.Output != "" {
			_, _ = io.WriteString(logWriter, response.Output)
		}
	}
	if repo := m.runtimeDBs.Runs(run.AgentName); repo != nil {
		_ = repo.Update(context.Background(), run)
	}
}

func (m *Manager) findActive(agentName, runID string) (*activeRun, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, active := range m.active {
		if runID != "" && active.run.ID != runID {
			continue
		}
		if agentName != "" && active.run.AgentName != agentName {
			continue
		}

		return active, nil
	}

	return nil, domain.ErrNotFound
}

func (m *Manager) setActiveRun(run domain.AgentRun) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if active := m.active[run.ID]; active != nil {
		active.run = run
	}
}

func (m *Manager) removeActive(runID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.active, runID)
}
