package http

import (
	stdhttp "net/http"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleExecute(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	run, err := s.executeUseCase.Execute(r.Context(), r.PathValue("name"))
	if err != nil {
		writeExecuteError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusAccepted, toRunResponse(run))
}

func (s *Server) handleStop(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	run, err := s.stopUseCase.Stop(r.Context(), r.PathValue("name"), r.PathValue("run_id"))
	if err != nil {
		writeStopError(w, err, true)

		return
	}

	writeJSON(w, stdhttp.StatusAccepted, toRunResponse(run))
}

func (s *Server) handleStopActive(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	run, err := s.stopUseCase.Stop(r.Context(), r.PathValue("name"), "")
	if err != nil {
		writeStopError(w, err, false)

		return
	}

	writeJSON(w, stdhttp.StatusAccepted, toRunResponse(run))
}

func toRunResponse(run domain.AgentRun) model.RunResponse {
	return model.RunResponse{
		RunID:     run.ID,
		AgentName: run.AgentName,
		Status:    string(run.Status),
	}
}
