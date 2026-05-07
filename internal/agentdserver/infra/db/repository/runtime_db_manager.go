package repository

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"agentd/internal/agentdserver/app"
	"agentd/internal/agentdserver/domain"
	"agentd/internal/agentdserver/infra/db"
)

var errRuntimeDBDirRequired = errors.New("runtime db directory is required")

type RuntimeDBManager struct {
	baseDir      string
	maxOpenConns int

	mu     sync.Mutex
	stores map[string]*runtimeStore
}

type runtimeStore struct {
	database *db.DB
	runs     app.AgentRunRepository
	events   app.RuntimeEventRepository
}

var _ app.RuntimeDBManager = (*RuntimeDBManager)(nil)

func NewRuntimeDBManager(baseDir string, maxOpenConns int) (*RuntimeDBManager, error) {
	if baseDir == "" {
		return nil, errRuntimeDBDirRequired
	}
	if maxOpenConns < 1 {
		return nil, fmt.Errorf("max open conns must be at least 1")
	}

	return &RuntimeDBManager{
		baseDir:      baseDir,
		maxOpenConns: maxOpenConns,
		stores:       make(map[string]*runtimeStore),
	}, nil
}

func (m *RuntimeDBManager) EnsureAgent(ctx context.Context, agentName string) error {
	if !domain.IsValidAgentName(agentName) {
		return fmt.Errorf("%w: invalid agent name %q", domain.ErrInvalidDefinition, agentName)
	}

	m.mu.Lock()
	if _, ok := m.stores[agentName]; ok {
		m.mu.Unlock()

		return nil
	}
	m.mu.Unlock()

	database, err := db.New("runtime", db.Config{
		Path:         filepath.Join(m.baseDir, agentName+".db"),
		MaxOpenConns: m.maxOpenConns,
		Pragmas:      db.PragmasRuntime,
	})
	if err != nil {
		return fmt.Errorf("open runtime db for agent %q: %w", agentName, err)
	}
	if err := database.Start(ctx); err != nil {
		_ = database.Stop(ctx)

		return fmt.Errorf("start runtime db for agent %q: %w", agentName, err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.stores[agentName]; ok {
		_ = database.Stop(ctx)
		if existing.database == nil {
			return fmt.Errorf("runtime db for agent %q is not open", agentName)
		}

		return nil
	}
	m.stores[agentName] = &runtimeStore{database: database}

	return nil
}

func (m *RuntimeDBManager) Runs(agentName string) app.AgentRunRepository {
	m.mu.Lock()
	defer m.mu.Unlock()

	store := m.stores[agentName]
	if store == nil {
		return nil
	}

	return store.runs
}

func (m *RuntimeDBManager) Events(agentName string) app.RuntimeEventRepository {
	m.mu.Lock()
	defer m.mu.Unlock()

	store := m.stores[agentName]
	if store == nil {
		return nil
	}

	return store.events
}

func (m *RuntimeDBManager) Close(ctx context.Context) error {
	m.mu.Lock()
	stores := m.stores
	m.stores = make(map[string]*runtimeStore)
	m.mu.Unlock()

	var closeErr error
	for agentName, store := range stores {
		if store == nil || store.database == nil {
			continue
		}
		if err := store.database.Stop(ctx); err != nil {
			closeErr = errors.Join(closeErr, fmt.Errorf("close runtime db %q: %w", agentName, err))
		}
	}

	return closeErr
}
