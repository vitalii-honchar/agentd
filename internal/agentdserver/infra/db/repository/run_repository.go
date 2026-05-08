package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db"
)

var (
	errRunRepositoryNilDB   = errors.New("run repository requires a non-nil db")
	errEventRepositoryNilDB = errors.New("event repository requires a non-nil db")
)

type AgentRunRepository struct {
	db *sql.DB
}

var _ app.AgentRunRepository = (*AgentRunRepository)(nil)

func NewAgentRunRepository(database *db.DB) (*AgentRunRepository, error) {
	if database == nil || database.DB == nil {
		return nil, errRunRepositoryNilDB
	}

	return &AgentRunRepository{db: database.DB}, nil
}

func (r *AgentRunRepository) Create(ctx context.Context, run domain.AgentRun) error {
	if run.ID == "" {
		return fmt.Errorf("run id is required")
	}
	now := time.Now().UTC()

	const query = `INSERT INTO agent_runs (
	           id, agent_name, agent_revision, trigger, status, due_at,
	           started_at, completed_at, work_dir, log_path, provider_request_id,
	           error_code, error_message, stop_requested_at, created_at, updated_at
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, query,
		run.ID,
		run.AgentName,
		run.AgentRevision,
		run.Trigger,
		run.Status,
		nullTime(run.DueAt),
		nullTime(run.StartedAt),
		nullTime(run.CompletedAt),
		run.WorkDir,
		run.LogPath,
		nullString(run.ProviderRequestID),
		nullString(run.ErrorCode),
		nullString(run.ErrorMessage),
		nullTime(run.StopRequestedAt),
		formatTime(now),
		formatTime(now),
	); err != nil {
		return fmt.Errorf("insert agent run: %w", err)
	}

	return nil
}

func (r *AgentRunRepository) Update(ctx context.Context, run domain.AgentRun) error {
	const query = `UPDATE agent_runs SET
	           agent_revision = ?,
	           trigger = ?,
	           status = ?,
	           due_at = ?,
	           started_at = ?,
	           completed_at = ?,
	           work_dir = ?,
	           log_path = ?,
	           provider_request_id = ?,
	           error_code = ?,
	           error_message = ?,
	           stop_requested_at = ?,
	           updated_at = ?
	       WHERE id = ?`
	result, err := r.db.ExecContext(ctx, query,
		run.AgentRevision,
		run.Trigger,
		run.Status,
		nullTime(run.DueAt),
		nullTime(run.StartedAt),
		nullTime(run.CompletedAt),
		run.WorkDir,
		run.LogPath,
		nullString(run.ProviderRequestID),
		nullString(run.ErrorCode),
		nullString(run.ErrorMessage),
		nullTime(run.StopRequestedAt),
		formatTime(time.Now().UTC()),
		run.ID,
	)
	if err != nil {
		return fmt.Errorf("update agent run: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("agent run rows affected: %w", err)
	}
	if affected == 0 {
		return domain.ErrNotFound
	}

	return nil
}

func (r *AgentRunRepository) FindByID(ctx context.Context, runID string) (domain.AgentRun, error) {
	const query = `SELECT id, agent_name, agent_revision, trigger, status, due_at,
	       started_at, completed_at, work_dir, log_path, provider_request_id,
	       error_code, error_message, stop_requested_at
	       FROM agent_runs WHERE id = ?`

	return scanRun(r.db.QueryRowContext(ctx, query, runID))
}

func (r *AgentRunRepository) FindLatest(ctx context.Context) (domain.AgentRun, error) {
	const query = `SELECT id, agent_name, agent_revision, trigger, status, due_at,
	       started_at, completed_at, work_dir, log_path, provider_request_id,
	       error_code, error_message, stop_requested_at
	       FROM agent_runs ORDER BY created_at DESC LIMIT 1`

	return scanRun(r.db.QueryRowContext(ctx, query))
}

func (r *AgentRunRepository) FindActive(ctx context.Context) (domain.AgentRun, error) {
	const query = `SELECT id, agent_name, agent_revision, trigger, status, due_at,
	       started_at, completed_at, work_dir, log_path, provider_request_id,
	       error_code, error_message, stop_requested_at
	       FROM agent_runs
	       WHERE status IN ('queued', 'running', 'stopping')
	       ORDER BY created_at ASC LIMIT 1`

	return scanRun(r.db.QueryRowContext(ctx, query))
}

func (r *AgentRunRepository) ListActive(ctx context.Context) ([]domain.AgentRun, error) {
	const query = `SELECT id, agent_name, agent_revision, trigger, status, due_at,
	       started_at, completed_at, work_dir, log_path, provider_request_id,
	       error_code, error_message, stop_requested_at
	       FROM agent_runs
	       WHERE status IN ('queued', 'running', 'stopping')
	       ORDER BY created_at ASC`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query active agent runs: %w", err)
	}
	defer rows.Close()

	return scanRuns(rows)
}

type RuntimeEventRepository struct {
	db *sql.DB
}

var _ app.RuntimeEventRepository = (*RuntimeEventRepository)(nil)

func NewRuntimeEventRepository(database *db.DB) (*RuntimeEventRepository, error) {
	if database == nil || database.DB == nil {
		return nil, errEventRepositoryNilDB
	}

	return &RuntimeEventRepository{db: database.DB}, nil
}

func (r *RuntimeEventRepository) Append(ctx context.Context, event domain.RuntimeEvent) error {
	if event.ID == "" {
		return fmt.Errorf("runtime event id is required")
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now().UTC()
	}

	const query = `INSERT INTO runtime_events (
	           id, agent_name, run_id, event_type, level, message,
	           attributes_json, created_at
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := r.db.ExecContext(ctx, query,
		event.ID,
		nullString(event.AgentName),
		nullString(event.RunID),
		event.EventType,
		event.Level,
		event.Message,
		event.AttributesJSON,
		formatTime(event.CreatedAt),
	); err != nil {
		return fmt.Errorf("insert runtime event: %w", err)
	}

	return nil
}

func (r *RuntimeEventRepository) ListByRun(
	ctx context.Context,
	runID string,
	limit int,
) ([]domain.RuntimeEvent, error) {
	const query = `SELECT id, agent_name, run_id, event_type, level, message,
	       attributes_json, created_at
	       FROM runtime_events
	       WHERE run_id = ?
	       ORDER BY created_at DESC
	       LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, runID, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("query runtime events by run: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

func (r *RuntimeEventRepository) ListRecent(
	ctx context.Context,
	limit int,
) ([]domain.RuntimeEvent, error) {
	const query = `SELECT id, agent_name, run_id, event_type, level, message,
	       attributes_json, created_at
	       FROM runtime_events
	       ORDER BY created_at DESC
	       LIMIT ?`

	rows, err := r.db.QueryContext(ctx, query, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("query recent runtime events: %w", err)
	}
	defer rows.Close()

	return scanEvents(rows)
}

type runScanner interface {
	Scan(dest ...any) error
}

func scanRun(scanner runScanner) (domain.AgentRun, error) {
	var (
		run                                        domain.AgentRun
		dueAt, startedAt, completedAt, stoppedAt   sql.NullString
		providerRequestID, errorCode, errorMessage sql.NullString
	)

	err := scanner.Scan(
		&run.ID,
		&run.AgentName,
		&run.AgentRevision,
		&run.Trigger,
		&run.Status,
		&dueAt,
		&startedAt,
		&completedAt,
		&run.WorkDir,
		&run.LogPath,
		&providerRequestID,
		&errorCode,
		&errorMessage,
		&stoppedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.AgentRun{}, domain.ErrNotFound
		}

		return domain.AgentRun{}, fmt.Errorf("scan agent run: %w", err)
	}

	var errParse error
	if run.DueAt, errParse = parseNullableTime(dueAt); errParse != nil {
		return domain.AgentRun{}, errParse
	}
	if run.StartedAt, errParse = parseNullableTime(startedAt); errParse != nil {
		return domain.AgentRun{}, errParse
	}
	if run.CompletedAt, errParse = parseNullableTime(completedAt); errParse != nil {
		return domain.AgentRun{}, errParse
	}
	if run.StopRequestedAt, errParse = parseNullableTime(stoppedAt); errParse != nil {
		return domain.AgentRun{}, errParse
	}
	run.ProviderRequestID = providerRequestID.String
	run.ErrorCode = errorCode.String
	run.ErrorMessage = errorMessage.String

	return run, nil
}

func scanRuns(rows *sql.Rows) ([]domain.AgentRun, error) {
	var runs []domain.AgentRun
	for rows.Next() {
		run, err := scanRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agent runs: %w", err)
	}

	return runs, nil
}

func scanEvents(rows *sql.Rows) ([]domain.RuntimeEvent, error) {
	var events []domain.RuntimeEvent
	for rows.Next() {
		event, err := scanEvent(rows)
		if err != nil {
			return nil, err
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runtime events: %w", err)
	}

	return events, nil
}

func scanEvent(scanner runScanner) (domain.RuntimeEvent, error) {
	var (
		event     domain.RuntimeEvent
		agentName sql.NullString
		runID     sql.NullString
		createdAt string
	)

	err := scanner.Scan(
		&event.ID,
		&agentName,
		&runID,
		&event.EventType,
		&event.Level,
		&event.Message,
		&event.AttributesJSON,
		&createdAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.RuntimeEvent{}, domain.ErrNotFound
		}

		return domain.RuntimeEvent{}, fmt.Errorf("scan runtime event: %w", err)
	}

	parsed, err := parseTime(createdAt)
	if err != nil {
		return domain.RuntimeEvent{}, err
	}
	event.AgentName = agentName.String
	event.RunID = runID.String
	event.CreatedAt = parsed

	return event, nil
}

func parseNullableTime(value sql.NullString) (*time.Time, error) {
	if !value.Valid || value.String == "" {
		return nil, nil
	}
	parsed, err := parseTime(value.String)
	if err != nil {
		return nil, err
	}

	return &parsed, nil
}

func normalizeLimit(limit int) int {
	if limit < 1 {
		return 100
	}

	return limit
}
