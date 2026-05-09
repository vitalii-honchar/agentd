package http

import (
	"errors"
	stdhttp "net/http"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/http/model"
)

const (
	errorCodeAgentDisabled         = "agent_disabled"
	errorCodeAgentNotFound         = "agent_not_found"
	errorCodeContractInputInvalid  = "contract_input_invalid"
	errorCodeContractOutputInvalid = "contract_output_invalid"
	errorCodeDaemonUnavailable     = "daemon_unavailable"
	errorCodeInternal              = "internal_error"
	errorCodeInvalidJSON           = "invalid_json"
	errorCodeInvalidQuery          = "invalid_query"
	errorCodeInvalidState          = "invalid_state"
	errorCodeRemoteClientForbidden = "remote_client_forbidden"
	errorCodeProviderError         = "provider_error"
	errorCodeReActFailed           = "react_failed"
	errorCodeRunAlreadyActive      = "run_already_active"
	errorCodeRunNotFound           = "run_not_found"
	errorCodeRunNotTerminal        = "run_not_terminal"
	errorCodeValidationFailed      = "validation_failed"
)

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
		writeError(w, stdhttp.StatusBadRequest, errorCodeValidationFailed, err.Error(), fields)

		return
	}

	writeInternalError(w)
}

func writeQueryError(w stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, stdhttp.StatusNotFound, errorCodeAgentNotFound, err.Error(), nil)
	default:
		writeInternalError(w)
	}
}

func writeExecuteError(w stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrContractInputInvalid):
		writeError(w, stdhttp.StatusBadRequest, errorCodeContractInputInvalid, err.Error(), nil)
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, stdhttp.StatusNotFound, errorCodeAgentNotFound, err.Error(), nil)
	default:
		writeRunStateError(w, err)
	}
}

func writeStopError(w stdhttp.ResponseWriter, err error, hasRunID bool) {
	switch {
	case errors.Is(err, domain.ErrNotFound) && hasRunID:
		writeError(w, stdhttp.StatusNotFound, errorCodeRunNotFound, err.Error(), nil)
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, stdhttp.StatusNotFound, errorCodeAgentNotFound, err.Error(), nil)
	default:
		writeRunStateError(w, err)
	}
}

func writeResultError(w stdhttp.ResponseWriter, err error, missingCode string) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(w, stdhttp.StatusNotFound, missingCode, err.Error(), nil)
	case errors.Is(err, domain.ErrRunNotTerminal):
		writeError(w, stdhttp.StatusConflict, errorCodeRunNotTerminal, err.Error(), nil)
	default:
		writeInternalError(w)
	}
}

func writeRunStateError(w stdhttp.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrRunAlreadyActive):
		writeError(w, stdhttp.StatusConflict, errorCodeRunAlreadyActive, err.Error(), nil)
	case errors.Is(err, domain.ErrAgentDisabled):
		writeError(w, stdhttp.StatusConflict, errorCodeAgentDisabled, err.Error(), nil)
	case errors.Is(err, domain.ErrInvalidState):
		writeError(w, stdhttp.StatusConflict, errorCodeInvalidState, err.Error(), nil)
	default:
		writeInternalError(w)
	}
}

func writeInternalError(w stdhttp.ResponseWriter) {
	writeError(w, stdhttp.StatusInternalServerError, errorCodeInternal, "internal server error", nil)
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
