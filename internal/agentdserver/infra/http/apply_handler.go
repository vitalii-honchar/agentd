package http

import (
	"encoding/json"
	"errors"
	stdhttp "net/http"

	appagent "agentd/internal/agentdserver/app/agent"
	"agentd/internal/agentdserver/domain"
	"agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleApply(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	var request model.ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, stdhttp.StatusBadRequest, "invalid_json", "invalid JSON request body", nil)

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

func writeApplyError(w stdhttp.ResponseWriter, err error) {
	var validationErr *domain.ValidationError
	if errors.As(err, &validationErr) || errors.Is(err, domain.ErrInvalidDefinition) {
		var fields []model.FieldError
		if validationErr != nil {
			fields = make([]model.FieldError, 0, len(validationErr.Issues))
			for _, issue := range validationErr.Issues {
				fields = append(fields, model.FieldError{Path: issue.Field, Message: issue.Message})
			}
		}
		writeError(w, stdhttp.StatusBadRequest, "invalid_definition", err.Error(), fields)

		return
	}

	writeError(w, stdhttp.StatusInternalServerError, "internal_error", "internal server error", nil)
}

func writeError(
	w stdhttp.ResponseWriter,
	status int,
	code string,
	message string,
	fields []model.FieldError,
) {
	writeJSON(w, status, model.ErrorResponse{
		Error: model.ErrorBody{
			Code:    code,
			Message: message,
			Fields:  fields,
		},
	})
}

func toAgentDetail(agent domain.Agent) model.AgentDetail {
	return model.AgentDetail{
		AgentSummary: model.AgentSummary{
			Name:         agent.Name,
			Enabled:      agent.Enabled,
			Status:       string(agent.Status),
			ScheduleType: string(agent.Schedule.Type),
			NextRunAt:    agent.NextRunAt,
		},
		Revision:    agent.Revision,
		VendorName:  agent.Vendor.Name,
		VendorModel: agent.Vendor.Model,
		LastRunID:   agent.LastRunID,
		RecentError: agent.LastError,
	}
}
