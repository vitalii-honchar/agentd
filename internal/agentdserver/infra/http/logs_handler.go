package http

import (
	stdhttp "net/http"
	"strconv"

	"agentd/internal/agentdserver/app"
	applogs "agentd/internal/agentdserver/app/logs"
	"agentd/internal/agentdserver/infra/http/model"
)

func (s *Server) handleLogs(w stdhttp.ResponseWriter, r *stdhttp.Request) {
	tail, err := parseTail(r.URL.Query().Get("tail"))
	if err != nil {
		writeError(w, stdhttp.StatusBadRequest, "invalid_query", err.Error(), nil)

		return
	}

	result, err := s.logsUseCase.Read(r.Context(), applogs.Query{
		AgentName: r.PathValue("name"),
		RunID:     r.URL.Query().Get("run_id"),
		Tail:      tail,
	})
	if err != nil {
		writeQueryError(w, err)

		return
	}

	writeJSON(w, stdhttp.StatusOK, toLogsResponse(result))
}

func parseTail(value string) (int, error) {
	if value == "" {
		return 0, nil
	}
	tail, err := strconv.Atoi(value)
	if err != nil {
		return 0, err
	}
	if tail < 1 {
		return 0, strconv.ErrSyntax
	}

	return tail, nil
}

func toLogsResponse(result applogs.Result) model.LogsResponse {
	entries := make([]model.LogEntry, 0, len(result.Entries))
	for _, entry := range result.Entries {
		entries = append(entries, toLogEntry(entry))
	}

	return model.LogsResponse{
		AgentName: result.Agent.Name,
		RunID:     result.Run.ID,
		Entries:   entries,
	}
}

func toLogEntry(entry app.LogEntry) model.LogEntry {
	return model.LogEntry{
		Timestamp: entry.Timestamp,
		RunID:     entry.RunID,
		Line:      entry.Line,
	}
}
