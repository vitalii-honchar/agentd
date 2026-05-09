package http

import (
	stdhttp "net/http"

	appresult "github.com/vitalii-honchar/agentd/internal/agentdserver/app/result"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleAgentResults(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	agentName := r.PathValue("name")
	results, err := s.resultUseCase.ResultsByAgent(r.Context(), agentName)
	if err != nil {
		writeResultError(w, err, errorCodeAgentNotFound)

		return
	}

	writeJSON(w, stdhttp.StatusOK, model.AgentResultsResponse{
		AgentName: agentName,
		Results:   toResultModels(results),
	})
}

func (s *Server) handleRunResult(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	result, err := s.resultUseCase.ResultByRunID(r.Context(), r.PathValue("run_id"))
	if err != nil {
		writeResultError(w, err, errorCodeRunNotFound)

		return
	}

	writeJSON(w, stdhttp.StatusOK, toResultModel(result))
}

func toResultModels(results []appresult.RunResult) []model.RunResult {
	mapped := make([]model.RunResult, 0, len(results))
	for _, result := range results {
		mapped = append(mapped, toResultModel(result))
	}

	return mapped
}

func toResultModel(result appresult.RunResult) model.RunResult {
	mapped := model.RunResult{
		RunSummary: model.RunSummary{
			RunID:       result.RunID,
			AgentName:   result.AgentName,
			Status:      string(result.Status),
			Trigger:     string(result.Trigger),
			StartedAt:   result.StartedAt,
			CompletedAt: result.CompletedAt,
		},
		ResultFormat:  string(result.ResultFormat),
		Result:        result.Result,
		ResultJSON:    result.ResultJSON,
		ResultSummary: result.ResultSummary,
	}
	if result.Failure != nil {
		mapped.Failure = &model.Failure{
			Code:    result.Failure.Code,
			Message: result.Failure.Message,
		}
	}

	return mapped
}
