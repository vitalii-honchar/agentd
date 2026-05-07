package agentdserver

import (
	"context"
	"fmt"
	"log/slog"

	"agentd/internal/agentdserver/config"
	"agentd/internal/agentdserver/infra/db"
	"agentd/internal/agentdserver/infra/db/repository"
	daemonhttp "agentd/internal/agentdserver/infra/http"
	openaiadapter "agentd/internal/agentdserver/infra/llm/openai"
)

type AgentdServer struct {
	Config *config.Config

	components []component
	settings   *db.DB
	runtimeDBs *repository.RuntimeDBManager
}

type component struct {
	name  string
	start func(context.Context) error
	stop  func(context.Context) error
}

func New() (*AgentdServer, error) {
	cfg, err := config.FromEnv()
	if err != nil {
		return nil, fmt.Errorf("load config: %w", err)
	}
	config.ConfigureLogger(cfg)

	return NewWithConfig(cfg)
}

func NewWithConfig(cfg *config.Config) (*AgentdServer, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate config: %w", err)
	}

	settings, err := db.New("settings", db.Config{
		Path:         cfg.Storage.SettingsDBPath,
		MaxOpenConns: cfg.Storage.SQLiteMaxConns,
		Pragmas:      db.PragmasSettings,
	})
	if err != nil {
		return nil, fmt.Errorf("open settings db: %w", err)
	}
	if _, err := repository.NewAgentRepository(settings); err != nil {
		_ = settings.Stop(context.Background())

		return nil, fmt.Errorf("new agent repository: %w", err)
	}

	runtimeDBs, err := repository.NewRuntimeDBManager(
		cfg.Storage.RuntimeDBDir,
		cfg.Storage.SQLiteMaxConns,
	)
	if err != nil {
		_ = settings.Stop(context.Background())

		return nil, fmt.Errorf("new runtime db manager: %w", err)
	}
	if cfg.OpenAI.APIKey != "" {
		if _, err := openaiadapter.NewProvider(openaiadapter.Config{APIKey: cfg.OpenAI.APIKey}); err != nil {
			_ = settings.Stop(context.Background())
			_ = runtimeDBs.Close(context.Background())

			return nil, fmt.Errorf("new openai provider: %w", err)
		}
	}

	httpServer := daemonhttp.NewServer(daemonhttp.Config{
		Address:      cfg.Server.Address(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	})

	return &AgentdServer{
		Config:     cfg,
		settings:   settings,
		runtimeDBs: runtimeDBs,
		components: []component{
			{name: "settings", start: settings.Start, stop: settings.Stop},
			{name: "http", start: startHTTP(httpServer), stop: httpServer.Stop},
			{name: "runtime-dbs", stop: runtimeDBs.Close},
		},
	}, nil
}

func (s *AgentdServer) Start(ctx context.Context) error {
	slog.Info("Starting agentdserver")
	for _, component := range s.components {
		if component.start == nil {
			continue
		}
		if err := component.start(ctx); err != nil {
			slog.Warn("Failed to start component", "component", component.name, "error", err)

			return fmt.Errorf("start %s: %w", component.name, err)
		}
	}
	slog.Info("agentdserver started")

	return nil
}

func (s *AgentdServer) Stop(ctx context.Context) error {
	slog.Info("Stopping agentdserver")
	for i := len(s.components) - 1; i >= 0; i-- {
		component := s.components[i]
		if component.stop == nil {
			continue
		}
		if err := component.stop(ctx); err != nil {
			slog.Warn("Failed to stop component", "component", component.name, "error", err)
		}
	}
	slog.Info("agentdserver stopped")

	return nil
}

func startHTTP(server *daemonhttp.Server) func(context.Context) error {
	return func(context.Context) error {
		go func() {
			if err := server.Start(); err != nil {
				slog.Error("HTTP server stopped with error", "error", err)
			}
		}()

		return nil
	}
}
