package http

import (
	"encoding/json"
	stdhttp "net/http"

	appagent "github.com/vitalii-honchar/agentd/internal/agentdserver/app/agent"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleApply(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var request model.ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, stdhttp.StatusBadRequest, errorCodeInvalidJSON, "invalid JSON request body", nil)

		return
	}

	result, err := s.applyUseCase.Apply(r.Context(), appagent.ApplyRequest{
		SourcePath: request.SourcePath,
		Markdown:   request.Markdown,
	})
	if err != nil {
		writeApplyError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusOK, model.ApplyResponse{
		Outcome: string(result.Outcome),
		Agent:   toAgentDetail(result.Agent),
	})
}
