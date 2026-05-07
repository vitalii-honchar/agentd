package http

import (
	"errors"
	stdhttp "net/http"

	"agentd/internal/agentdserver/domain"
	"agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleInspect(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	agent, err := s.inspectUseCase.Inspect(r.Context(), r.PathValue("name"))
	if err != nil {
		writeQueryError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusOK, toAgentDetail(agent))
}

func writeQueryError(w stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, stdhttp.StatusNotFound, "not_found", err.Error(), nil)
	default:
		writeError(w, stdhttp.StatusInternalServerError, "internal_error", "internal server error", nil)
	}
}

func toAgentDetail(agent domain.Agent) model.AgentDetail {
	return model.AgentDetail{
		AgentSummary: toAgentSummary(agent),
		Revision:     agent.Revision,
		VendorName:   agent.Vendor.Name,
		VendorModel:  agent.Vendor.Model,
		LastRunID:    agent.LastRunID,
		RecentError:  agent.LastError,
	}
}
