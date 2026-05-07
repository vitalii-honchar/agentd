package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	stdhttp "net/http"
	"time"

	appagent "agentd/internal/agentdserver/app/agent"
	"agentd/internal/agentdserver/domain"
)

type Config struct {
	Address      string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type Server struct {
	server         *stdhttp.Server
	mux            *stdhttp.ServeMux
	applyUseCase   ApplyUseCase
	executeUseCase ExecuteUseCase
	stopUseCase    StopUseCase
}

type ApplyUseCase interface {
	Apply(context.Context, appagent.ApplyRequest) (appagent.ApplyResult, error)
}

type ExecuteUseCase interface {
	Execute(context.Context, string) (domain.AgentRun, error)
}

type StopUseCase interface {
	Stop(context.Context, string, string) (domain.AgentRun, error)
}

type Option func(*Server)

func WithApplyUseCase(useCase ApplyUseCase) Option {
	return func(s *Server) {
		s.applyUseCase = useCase
	}
}

func WithExecuteUseCase(useCase ExecuteUseCase) Option {
	return func(s *Server) {
		s.executeUseCase = useCase
	}
}

func WithStopUseCase(useCase StopUseCase) Option {
	return func(s *Server) {
		s.stopUseCase = useCase
	}
}

func NewServer(cfg Config, opts ...Option) *Server {
	mux := stdhttp.NewServeMux()
	server := &Server{
		mux: mux,
		server: &stdhttp.Server{
			Addr:         cfg.Address,
			Handler:      mux,
			ReadTimeout:  cfg.ReadTimeout,
			WriteTimeout: cfg.WriteTimeout,
		},
	}
	for _, opt := range opts {
		opt(server)
	}
	server.registerRoutes()

	return server
}

func (s *Server) Handler() stdhttp.Handler {
	return s.mux
}

func (s *Server) Start() error {
	if err := s.server.ListenAndServe(); err != nil && !errors.Is(err, stdhttp.ErrServerClosed) {
		return fmt.Errorf("listen and serve: %w", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	if err := s.server.Shutdown(ctx); err != nil {
		return fmt.Errorf("shutdown http server: %w", err)
	}

	return nil
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("GET /health", healthHandler)
	if s.applyUseCase != nil {
		s.mux.HandleFunc("POST /v1/agents/apply", s.handleApply)
	}
	if s.executeUseCase != nil {
		s.mux.HandleFunc("POST /v1/agents/{name}/runs", s.handleExecute)
	}
	if s.stopUseCase != nil {
		s.mux.HandleFunc("POST /v1/agents/{name}/runs/{run_id}/stop", s.handleStop)
	}
}

func healthHandler(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	writeJSON(w, stdhttp.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w stdhttp.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The response is already committed; there is no useful recovery path here.
		return
	}
}
