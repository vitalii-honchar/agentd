package http

import (
	stdhttp "net/http"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleInspect(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	agent, err := s.inspectUseCase.Inspect(r.Context(), r.PathValue("name"))
	if err != nil {
		writeQueryError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusOK, toAgentDetail(agent))
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
