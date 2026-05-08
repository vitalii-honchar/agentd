package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/vitalii-honchar/agentd/internal/agentdserver/app"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/domain"
	"github.com/vitalii-honchar/agentd/internal/agentdserver/infra/db"
)

var errAgentRepositoryNilDB = errors.New("agent repository requires a non-nil db")

type AgentRepository struct {
	db *sql.DB
}

var _ app.AgentRepository = (*AgentRepository)(nil)

func NewAgentRepository(database *db.DB) (*AgentRepository, error) {
	if database == nil || database.DB == nil {
		return nil, errAgentRepositoryNilDB
	}

	return &AgentRepository{db: database.DB}, nil
}

func (r *AgentRepository) Save(
	ctx context.Context,
	agent domain.Agent,
	tools []domain.ToolPermission,
	mcpServers []domain.ToolPermission,
) error {
	now := time.Now().UTC()
	if agent.CreatedAt.IsZero() {
		agent.CreatedAt = now
	}
	if agent.UpdatedAt.IsZero() {
		agent.UpdatedAt = now
	}
	if agent.AppliedAt.IsZero() {
		agent.AppliedAt = now
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin save agent tx: %w", err)
	}
	if err := r.upsertAgent(ctx, tx, agent); err != nil {
		_ = tx.Rollback()

		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_tools WHERE agent_name = ?`, agent.Name); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("delete agent tools: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM agent_mcp_servers WHERE agent_name = ?`, agent.Name); err != nil {
		_ = tx.Rollback()

		return fmt.Errorf("delete agent mcp servers: %w", err)
	}
	for _, tool := range tools {
		tool.AgentName = agent.Name
		if err := insertTool(ctx, tx, tool, now); err != nil {
			_ = tx.Rollback()

			return err
		}
	}
	for _, server := range mcpServers {
		server.AgentName = agent.Name
		if err := insertMCPServer(ctx, tx, server, now); err != nil {
			_ = tx.Rollback()

			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit save agent tx: %w", err)
	}

	return nil
}

func (r *AgentRepository) FindByName(ctx context.Context, name string) (domain.Agent, error) {
	const query = `SELECT name, revision, definition_source_path, definition_markdown,
	       prompt, enabled, vendor_name, vendor_model, schedule_type,
	       schedule_expression, next_run_at, status, last_run_id, last_error,
	       created_at, updated_at, applied_at
	       FROM agents WHERE name = ?`

	agent, err := scanAgent(r.db.QueryRowContext(ctx, query, name))
	if err != nil {
		return domain.Agent{}, err
	}

	return agent, nil
}

func (r *AgentRepository) List(ctx context.Context) ([]domain.Agent, error) {
	const query = `SELECT name, revision, definition_source_path, definition_markdown,
	       prompt, enabled, vendor_name, vendor_model, schedule_type,
	       schedule_expression, next_run_at, status, last_run_id, last_error,
	       created_at, updated_at, applied_at
	       FROM agents ORDER BY name`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query agents: %w", err)
	}
	defer rows.Close()

	var agents []domain.Agent
	for rows.Next() {
		agent, err := scanAgent(rows)
		if err != nil {
			return nil, err
		}
		agents = append(agents, agent)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate agents: %w", err)
	}

	return agents, nil
}

func (r *AgentRepository) upsertAgent(
	ctx context.Context,
	tx *sql.Tx,
	agent domain.Agent,
) error {
	const query = `INSERT INTO agents (
	           name, revision, definition_source_path, definition_markdown,
	           prompt, enabled, vendor_name, vendor_model, schedule_type,
	           schedule_expression, next_run_at, status, last_run_id, last_error,
	           created_at, updated_at, applied_at
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	       ON CONFLICT(name) DO UPDATE SET
	           revision = excluded.revision,
	           definition_source_path = excluded.definition_source_path,
	           definition_markdown = excluded.definition_markdown,
	           prompt = excluded.prompt,
	           enabled = excluded.enabled,
	           vendor_name = excluded.vendor_name,
	           vendor_model = excluded.vendor_model,
	           schedule_type = excluded.schedule_type,
	           schedule_expression = excluded.schedule_expression,
	           next_run_at = excluded.next_run_at,
	           status = excluded.status,
	           last_run_id = excluded.last_run_id,
	           last_error = excluded.last_error,
	           updated_at = excluded.updated_at,
	           applied_at = excluded.applied_at`

	if _, err := tx.ExecContext(ctx, query,
		agent.Name,
		agent.Revision,
		agent.DefinitionSource,
		agent.DefinitionMarkdown,
		agent.Prompt,
		boolToInt(agent.Enabled),
		agent.Vendor.Name,
		agent.Vendor.Model,
		agent.Schedule.Type,
		nullString(agent.Schedule.Expression),
		nullTime(agent.NextRunAt),
		agent.Status,
		nullString(agent.LastRunID),
		nullString(agent.LastError),
		formatTime(agent.CreatedAt),
		formatTime(agent.UpdatedAt),
		formatTime(agent.AppliedAt),
	); err != nil {
		return fmt.Errorf("upsert agent: %w", err)
	}

	return nil
}

func insertTool(ctx context.Context, tx *sql.Tx, tool domain.ToolPermission, now time.Time) error {
	args, err := marshalList(tool.Args)
	if err != nil {
		return err
	}
	env, err := marshalList(tool.Env)
	if err != nil {
		return err
	}
	readPaths, err := marshalList(tool.ReadPaths)
	if err != nil {
		return err
	}
	writePaths, err := marshalList(tool.WritePaths)
	if err != nil {
		return err
	}
	networkAllow, err := marshalList(tool.NetworkAllow)
	if err != nil {
		return err
	}

	const query = `INSERT INTO agent_tools (
	           agent_name, name, kind, command, args_json, env_json,
	           read_paths_json, write_paths_json, network_allow_json, created_at
	       ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, query,
		tool.AgentName,
		tool.Name,
		tool.Kind,
		nullString(tool.Command),
		args,
		env,
		readPaths,
		writePaths,
		networkAllow,
		formatTime(now),
	); err != nil {
		return fmt.Errorf("insert agent tool %q: %w", tool.Name, err)
	}

	return nil
}

func insertMCPServer(
	ctx context.Context,
	tx *sql.Tx,
	server domain.ToolPermission,
	now time.Time,
) error {
	args, err := marshalList(server.Args)
	if err != nil {
		return err
	}
	env, err := marshalList(server.Env)
	if err != nil {
		return err
	}

	const query = `INSERT INTO agent_mcp_servers (
	           agent_name, name, command, args_json, env_json, created_at
	       ) VALUES (?, ?, ?, ?, ?, ?)`
	if _, err := tx.ExecContext(ctx, query,
		server.AgentName,
		server.Name,
		server.Command,
		args,
		env,
		formatTime(now),
	); err != nil {
		return fmt.Errorf("insert agent mcp server %q: %w", server.Name, err)
	}

	return nil
}

type agentScanner interface {
	Scan(dest ...any) error
}

func scanAgent(scanner agentScanner) (domain.Agent, error) {
	var (
		agent                                               domain.Agent
		enabled                                             int
		scheduleExpression, nextRunAt, lastRunID, lastError sql.NullString
		createdAt, updatedAt, appliedAt                     string
	)

	err := scanner.Scan(
		&agent.Name,
		&agent.Revision,
		&agent.DefinitionSource,
		&agent.DefinitionMarkdown,
		&agent.Prompt,
		&enabled,
		&agent.Vendor.Name,
		&agent.Vendor.Model,
		&agent.Schedule.Type,
		&scheduleExpression,
		&nextRunAt,
		&agent.Status,
		&lastRunID,
		&lastError,
		&createdAt,
		&updatedAt,
		&appliedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Agent{}, domain.ErrNotFound
		}

		return domain.Agent{}, fmt.Errorf("scan agent: %w", err)
	}

	agent.Enabled = enabled == 1
	agent.Schedule.Expression = scheduleExpression.String
	agent.LastRunID = lastRunID.String
	agent.LastError = lastError.String
	if nextRunAt.Valid {
		parsed, err := parseTime(nextRunAt.String)
		if err != nil {
			return domain.Agent{}, err
		}
		agent.NextRunAt = &parsed
	}
	if agent.CreatedAt, err = parseTime(createdAt); err != nil {
		return domain.Agent{}, err
	}
	if agent.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return domain.Agent{}, err
	}
	if agent.AppliedAt, err = parseTime(appliedAt); err != nil {
		return domain.Agent{}, err
	}

	return agent, nil
}

func marshalList(values []string) (string, error) {
	if values == nil {
		values = []string{}
	}
	body, err := json.Marshal(values)
	if err != nil {
		return "", fmt.Errorf("marshal string list: %w", err)
	}

	return string(body), nil
}

func nullString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	return value
}

func nullTime(value *time.Time) any {
	if value == nil {
		return nil
	}

	return formatTime(*value)
}

func formatTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func parseTime(value string) (time.Time, error) {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time %q: %w", value, err)
	}

	return parsed, nil
}

func boolToInt(value bool) int {
	if value {
		return 1
	}

	return 0
}
