package agentdserver

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	appagent "github.com/vitalii-honchar/agentd/internal/agentdserver/app/agent"
	applogs "github.com/vitalii-honchar/agentd/internal/agentdserver/app/logs"
	appresult "github.com/vitalii-honchar/agentd/internal/agentdserver/app/result"
	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
	appscheduling "github.com/vitalii-honchar/agentd/internal/agentdserver/app/scheduling"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/config"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db/repository"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/definition"
	daemonhttp "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http"
	openaiadapter "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/llm/openai"
	runlogs "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/logs"
	infraruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/runtime"
	infrascheduler "github.com/vitalii-honchar/agentd/internal/agentdserver/infra/scheduler"
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
	agentRepo, err := repository.NewAgentRepository(settings)
	if err != nil {
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
	var providers []appruntime.Provider
	if cfg.OpenAI.APIKey != "" {
		openAIProvider, err := openaiadapter.NewProvider(openaiadapter.Config{APIKey: cfg.OpenAI.APIKey})
		if err != nil {
			_ = settings.Stop(context.Background())
			_ = runtimeDBs.Close(context.Background())

			return nil, fmt.Errorf("new openai provider: %w", err)
		}
		providers = append(providers, openAIProvider)
	}
	logFactory, err := runlogs.NewRunLogFactory(cfg.Storage.RunLogDir)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new run log factory: %w", err)
	}
	logReader, err := runlogs.NewRunLogReader(cfg.Storage.RunLogDir)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new run log reader: %w", err)
	}
	isolation, err := infraruntime.NewIsolationBuilder(filepath.Join(cfg.Storage.DataDir, "work"))
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new isolation builder: %w", err)
	}
	revisionArtifacts, err := infraruntime.NewRevisionArtifactService(filepath.Join(cfg.Storage.DataDir, "work"))
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new revision artifact service: %w", err)
	}
	runtimeManager, err := infraruntime.NewManager(runtimeDBs, logFactory, isolation, providers)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new runtime manager: %w", err)
	}
	runtimeManager.SetToolExecutor(infraruntime.NewProcessToolExecutor(60 * time.Second))
	scheduler := infrascheduler.New()
	executeUC := appruntime.NewExecuteUseCase(agentRepo, runtimeManager)
	stopUC := appruntime.NewStopUseCase(runtimeManager)
	recoveryUC := appruntime.NewRecoveryUseCase(agentRepo, runtimeDBs)
	reconcileUC := appscheduling.NewReconcileUseCase(
		agentRepo,
		scheduler,
		func(ctx context.Context, trigger appscheduling.Trigger) error {
			agent, err := agentRepo.FindByName(ctx, trigger.AgentName)
			if err != nil {
				return err
			}
			_, err = runtimeManager.Execute(ctx, appruntime.ExecuteRequest{
				Agent:   agent,
				Trigger: domain.RunTriggerSchedule,
				DueAt:   &trigger.DueAt,
			})

			return err
		},
	)
	applyUC, err := appagent.NewApplyUseCase(
		appagent.ParserFunc(definition.ParseMarkdown),
		agentRepo,
		runtimeDBs,
		appagent.WithRevisionArtifactCreator(revisionArtifactCreator{service: revisionArtifacts}),
	)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new apply use case: %w", err)
	}
	listUC, err := appagent.NewListUseCase(agentRepo)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new list use case: %w", err)
	}
	inspectUC, err := appagent.NewInspectUseCase(agentRepo)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new inspect use case: %w", err)
	}
	revisionUC, err := appagent.NewRevisionUseCase(agentRepo)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new revision use case: %w", err)
	}
	logsUC, err := applogs.NewUseCase(agentRepo, runtimeDBs, logReader)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new logs use case: %w", err)
	}
	runListUC, err := appresult.NewListRunsUseCase(agentRepo, runtimeDBs)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new run list use case: %w", err)
	}
	resultUC, err := appresult.NewUseCase(agentRepo, runtimeDBs)
	if err != nil {
		_ = settings.Stop(context.Background())
		_ = runtimeDBs.Close(context.Background())

		return nil, fmt.Errorf("new result use case: %w", err)
	}

	httpServer := daemonhttp.NewServer(daemonhttp.Config{
		Address:      cfg.Server.Address(),
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	},
		daemonhttp.WithApplyUseCase(applyUC),
		daemonhttp.WithExecuteUseCase(executeUC),
		daemonhttp.WithStopUseCase(stopUC),
		daemonhttp.WithRunListUseCase(runListUC),
		daemonhttp.WithResultUseCase(resultUC),
		daemonhttp.WithListUseCase(listUC),
		daemonhttp.WithInspectUseCase(inspectUC),
		daemonhttp.WithRevisionUseCase(revisionUC),
		daemonhttp.WithLogsUseCase(logsUC),
	)
	workRoot := filepath.Join(cfg.Storage.DataDir, "work")

	return &AgentdServer{
		Config:     cfg,
		settings:   settings,
		runtimeDBs: runtimeDBs,
		components: []component{
			{name: "settings", start: settings.Start, stop: settings.Stop},
			{name: "revision-artifact-recovery", start: func(ctx context.Context) error {
				return recoverRevisionArtifacts(ctx, agentRepo, workRoot)
			}},
			{name: "execution-dir-cleanup", start: func(context.Context) error {
				return cleanupStaleExecutionDirs(workRoot)
			}},
			{name: "recovery", start: func(ctx context.Context) error {
				_, err := recoveryUC.Recover(ctx)

				return err
			}},
			{name: "scheduler", start: scheduler.Start, stop: scheduler.Stop},
			{name: "schedule-reconcile", start: func(ctx context.Context) error {
				return reconcileUC.Reconcile(ctx)
			}},
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

func recoverRevisionArtifacts(ctx context.Context, revisions revisionArtifactRepository, workRoot string) error {
	agents, err := revisions.List(ctx)
	if err != nil {
		return fmt.Errorf("list agents for revision recovery: %w", err)
	}
	for _, agent := range agents {
		agentRevisions, err := revisions.ListRevisions(ctx, agent.Name)
		if err != nil {
			return fmt.Errorf("list revisions for %s: %w", agent.Name, err)
		}
		for _, revision := range agentRevisions {
			switch revision.Status {
			case domain.AgentRevisionStatusPending:
				message := "revision creation was interrupted before finalization"
				if err := revisions.MarkRevisionCorrupt(ctx, revision.AgentName, revision.RevisionID, message); err != nil {
					return fmt.Errorf("mark pending revision corrupt %s:%s: %w", revision.AgentName, revision.RevisionID, err)
				}
				slog.Warn(
					"Marked pending revision corrupt during startup recovery",
					"agent", revision.AgentName,
					"revision", revision.RevisionID,
					"error", message,
				)
			case domain.AgentRevisionStatusFinalized:
				if message := revisionArtifactCorruption(revision, workRoot); message != "" {
					if err := revisions.MarkRevisionCorrupt(ctx, revision.AgentName, revision.RevisionID, message); err != nil {
						return fmt.Errorf("mark finalized revision corrupt %s:%s: %w", revision.AgentName, revision.RevisionID, err)
					}
					slog.Warn(
						"Marked revision artifact corrupt during startup recovery",
						"agent", revision.AgentName,
						"revision", revision.RevisionID,
						"error", message,
					)
				}
			}
		}
	}

	return nil
}

type revisionArtifactRepository interface {
	List(ctx context.Context) ([]domain.Agent, error)
	ListRevisions(ctx context.Context, agentName string) ([]domain.AgentRevision, error)
	MarkRevisionCorrupt(ctx context.Context, agentName, revisionID, errorMessage string) error
}

type revisionArtifactCreator struct {
	service *infraruntime.RevisionArtifactService
}

func (c revisionArtifactCreator) CreateRevisionArtifact(
	ctx context.Context,
	request appagent.RevisionArtifactRequest,
) (appagent.RevisionArtifactResult, error) {
	result, err := c.service.Create(ctx, infraruntime.RevisionArtifactRequest{
		Definition: request.Definition,
		RevisionID: request.RevisionID,
		CreatedAt:  request.CreatedAt,
	})
	if err != nil {
		return appagent.RevisionArtifactResult{}, err
	}

	return appagent.RevisionArtifactResult{Revision: result.Revision}, nil
}

func revisionArtifactCorruption(revision domain.AgentRevision, workRoot string) string {
	artifactPath := revision.ArtifactPath
	if artifactPath == "" {
		artifactPath = filepath.Join(workRoot, revision.AgentName, revision.RevisionID)
	}
	if info, err := os.Stat(artifactPath); err != nil {
		return fmt.Sprintf("revision artifact directory is missing: %s", artifactPath)
	} else if !info.IsDir() {
		return fmt.Sprintf("revision artifact path is not a directory: %s", artifactPath)
	}
	for _, file := range revision.ArtifactFiles {
		if message := revisionFileCorruption(artifactPath, file.ArtifactRelativePath); message != "" {
			return message
		}
	}
	for _, tool := range revision.Tools {
		if tool.Kind != domain.ToolKindCustomTool {
			continue
		}
		for _, copiedFile := range tool.CopiedFiles {
			if message := revisionFileCorruption(artifactPath, copiedFile); message != "" {
				return message
			}
		}
	}

	return ""
}

func revisionFileCorruption(artifactPath string, relativePath string) string {
	if relativePath == "" {
		return ""
	}
	path := filepath.Join(artifactPath, relativePath)
	if info, err := os.Stat(path); err != nil {
		return fmt.Sprintf("revision artifact file is missing: %s", path)
	} else if info.IsDir() {
		return fmt.Sprintf("revision artifact file is a directory: %s", path)
	}

	return ""
}

func cleanupStaleExecutionDirs(workRoot string) error {
	agentEntries, err := os.ReadDir(workRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read work root for execution cleanup: %w", err)
	}
	for _, agentEntry := range agentEntries {
		if !agentEntry.IsDir() {
			continue
		}
		executionsPath := filepath.Join(workRoot, agentEntry.Name(), "executions")
		executionEntries, err := os.ReadDir(executionsPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}

			return fmt.Errorf("read executions dir %s: %w", executionsPath, err)
		}
		for _, executionEntry := range executionEntries {
			if !executionEntry.IsDir() {
				continue
			}
			path := filepath.Join(executionsPath, executionEntry.Name())
			if err := os.RemoveAll(path); err != nil {
				return fmt.Errorf("remove stale execution dir %s: %w", path, err)
			}
			slog.Info("Removed stale execution directory", "path", path)
		}
	}

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
