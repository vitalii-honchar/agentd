package http

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	stdhttp "net/http"
	"strings"
	"time"

	appagent "github.com/vitalii-honchar/agentd/internal/agentdserver/app/agent"
	applogs "github.com/vitalii-honchar/agentd/internal/agentdserver/app/logs"
	appresult "github.com/vitalii-honchar/agentd/internal/agentdserver/app/result"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
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
	runListUseCase RunListUseCase
	resultUseCase  ResultUseCase
	listUseCase    ListUseCase
	inspectUseCase InspectUseCase
	logsUseCase    LogsUseCase
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

type RunListUseCase interface {
	ListRuns(context.Context, bool) ([]domain.AgentRun, error)
}

type ResultUseCase interface {
	ResultsByAgent(context.Context, string) ([]appresult.RunResult, error)
	ResultByRunID(context.Context, string) (appresult.RunResult, error)
}

type ListUseCase interface {
	List(context.Context) ([]domain.Agent, error)
}

type InspectUseCase interface {
	Inspect(context.Context, string) (domain.Agent, error)
}

type LogsUseCase interface {
	Read(context.Context, applogs.Query) (applogs.Result, error)
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

func WithRunListUseCase(useCase RunListUseCase) Option {
	return func(s *Server) {
		s.runListUseCase = useCase
	}
}

func WithResultUseCase(useCase ResultUseCase) Option {
	return func(s *Server) {
		s.resultUseCase = useCase
	}
}

func WithListUseCase(useCase ListUseCase) Option {
	return func(s *Server) {
		s.listUseCase = useCase
	}
}

func WithInspectUseCase(useCase InspectUseCase) Option {
	return func(s *Server) {
		s.inspectUseCase = useCase
	}
}

func WithLogsUseCase(useCase LogsUseCase) Option {
	return func(s *Server) {
		s.logsUseCase = useCase
	}
}

func NewServer(cfg Config, opts ...Option) *Server {
	mux := stdhttp.NewServeMux()
	server := &Server{
		mux: mux,
		server: &stdhttp.Server{
			Addr:         cfg.Address,
			Handler:      sameHostMiddleware(mux),
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
	return s.server.Handler
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
	if s.listUseCase != nil {
		s.mux.HandleFunc("GET /v1/agents", s.handleList)
	}
	if s.applyUseCase != nil {
		s.mux.HandleFunc("POST /v1/agents/apply", s.handleApply)
	}
	if s.inspectUseCase != nil {
		s.mux.HandleFunc("GET /v1/agents/{name}", s.handleInspect)
	}
	if s.executeUseCase != nil {
		s.mux.HandleFunc("POST /v1/agents/{name}/runs", s.handleExecute)
	}
	if s.stopUseCase != nil {
		s.mux.HandleFunc("POST /v1/agents/{name}/runs/stop", s.handleStopActive)
		s.mux.HandleFunc("POST /v1/agents/{name}/runs/{run_id}/stop", s.handleStop)
	}
	if s.runListUseCase != nil {
		s.mux.HandleFunc("GET /v1/runs", s.handleListRuns)
	}
	if s.resultUseCase != nil {
		s.mux.HandleFunc("GET /v1/agents/{name}/results", s.handleAgentResults)
		s.mux.HandleFunc("GET /v1/runs/{run_id}/result", s.handleRunResult)
	}
	if s.logsUseCase != nil {
		s.mux.HandleFunc("GET /v1/agents/{name}/logs", s.handleLogs)
	}
}

func healthHandler(w stdhttp.ResponseWriter, _ *stdhttp.Request) {
	writeJSON(w, stdhttp.StatusOK, map[string]string{"status": "ok"})
}

func sameHostMiddleware(next stdhttp.Handler) stdhttp.Handler {
	return stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		if !isSameHostRemoteAddr(r.RemoteAddr) {
			writeError(
				w,
				stdhttp.StatusForbidden,
				errorCodeRemoteClientForbidden,
				"requests must originate from the same host",
				nil,
			)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func isSameHostRemoteAddr(remoteAddr string) bool {
	host := strings.TrimSpace(remoteAddr)
	if host == "" {
		return false
	}
	if splitHost, _, err := net.SplitHostPort(host); err == nil {
		host = splitHost
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	return ip.IsLoopback()
}

func writeJSON(w stdhttp.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		// The response is already committed; there is no useful recovery path here.
		return
	}
}
