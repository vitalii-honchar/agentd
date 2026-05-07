package http

import (
	stdhttp "net/http"

	"agentd/internal/agentdserver/domain"
	"agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleList(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	agents, err := s.listUseCase.List(r.Context())
	if err != nil {
		writeQueryError(w, err)

		return
	}

	summaries := make([]model.AgentSummary, 0, len(agents))
	for _, agent := range agents {
		summaries = append(summaries, toAgentSummary(agent))
	}

	writeJSON(w, stdhttp.StatusOK, model.ListResponse{Agents: summaries})
}

func toAgentSummary(agent domain.Agent) model.AgentSummary {
	return model.AgentSummary{
		Name:         agent.Name,
		Enabled:      agent.Enabled,
		Status:       string(agent.Status),
		ScheduleType: string(agent.Schedule.Type),
		NextRunAt:    agent.NextRunAt,
	}
}
