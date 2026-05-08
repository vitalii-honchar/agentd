package http

import (
	stdhttp "net/http"
	"strconv"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleListRuns(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	includeAll, err := parseBoolQuery(r, "all")
	if err != nil {
		writeError(w, stdhttp.StatusBadRequest, errorCodeInvalidQuery, err.Error(), nil)

		return
	}
	runs, err := s.runListUseCase.ListRuns(r.Context(), includeAll)
	if err != nil {
		writeQueryError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusOK, toRunListResponse(runs))
}

func parseBoolQuery(r *stdhttp.Request, key string) (bool, error) {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return false, nil
	}

	return strconv.ParseBool(raw)
}

func toRunListResponse(runs []domain.AgentRun) model.RunListResponse {
	response := model.RunListResponse{Runs: make([]model.RunSummary, 0, len(runs))}
	for _, run := range runs {
		response.Runs = append(response.Runs, toRunSummary(run))
	}

	return response
}

func toRunSummary(run domain.AgentRun) model.RunSummary {
	return model.RunSummary{
		RunID:       run.ID,
		AgentName:   run.AgentName,
		Status:      string(run.Status),
		Trigger:     string(run.Trigger),
		StartedAt:   run.StartedAt,
		CompletedAt: run.CompletedAt,
	}
}
