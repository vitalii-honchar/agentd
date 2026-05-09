package model

import (
	"encoding/json"
	"time"
)

type ExecuteRequest struct {
	Input        json.RawMessage   `json:"input,omitempty"`
	LegacyInputs map[string]string `json:"legacy_inputs,omitempty"`
	Inputs       map[string]string `json:"inputs,omitempty"`
}

type RunResponse struct {
	RunID         string `json:"run_id"`
	AgentName     string `json:"agent_name"`
	AgentRevision string `json:"agent_revision,omitempty"`
	Status        string `json:"status"`
}

type RunListResponse struct {
	Runs []RunSummary `json:"runs"`
}

type RunResult struct {
	RunSummary
	ResultFormat  string          `json:"result_format,omitempty"`
	Result        string          `json:"result,omitempty"`
	ResultJSON    json.RawMessage `json:"result_json,omitempty"`
	ResultSummary string          `json:"result_summary,omitempty"`
	Failure       *Failure        `json:"failure,omitempty"`
}

type Failure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type RunSummary struct {
	RunID       string     `json:"run_id"`
	AgentName   string     `json:"agent_name"`
	Status      string     `json:"status"`
	Trigger     string     `json:"trigger"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}
