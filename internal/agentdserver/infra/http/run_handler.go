package http

import (
	"errors"
	stdhttp "net/http"

	"agentd/internal/agentdserver/domain"
	"agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleExecute(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	run, err := s.executeUseCase.Execute(r.Context(), r.PathValue("name"))
	if err != nil {
		writeRunError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusAccepted, toRunResponse(run))
}

func (s *Server) handleStop(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	run, err := s.stopUseCase.Stop(r.Context(), r.PathValue("name"), r.PathValue("run_id"))
	if err != nil {
		writeRunError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusAccepted, toRunResponse(run))
}

func (s *Server) handleStopActive(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	run, err := s.stopUseCase.Stop(r.Context(), r.PathValue("name"), "")
	if err != nil {
		writeRunError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusAccepted, toRunResponse(run))
}

func writeRunError(w stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, stdhttp.StatusNotFound, "not_found", err.Error(), nil)
	case errors.Is(err, domain.ErrRunAlreadyActive),
		errors.Is(err, domain.ErrAgentDisabled),
		errors.Is(err, domain.ErrInvalidState):
		writeError(w, stdhttp.StatusConflict, "conflict", err.Error(), nil)
	default:
		writeError(w, stdhttp.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func toRunResponse(run domain.AgentRun) model.RunResponse {
	return model.RunResponse{
		RunID:     run.ID,
		AgentName: run.AgentName,
		Status:    string(run.Status),
	}
}
