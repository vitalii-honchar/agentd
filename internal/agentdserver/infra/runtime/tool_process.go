package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"syscall"
	"time"

	appresult "github.com/vitalii-honchar/agentd/internal/agentdserver/app/result"
	appruntime "github.com/vitalii-honchar/agentd/internal/agentdserver/app/runtime"
)

type ProcessToolExecutor struct {
	defaultTimeout time.Duration
}

const toolOutputSummaryLimit = 8000

var _ appruntime.ToolExecutor = (*ProcessToolExecutor)(nil)

func NewProcessToolExecutor(defaultTimeout time.Duration) *ProcessToolExecutor {
	if defaultTimeout <= 0 {
		defaultTimeout = 60 * time.Second
	}

	return &ProcessToolExecutor{defaultTimeout: defaultTimeout}
}

func (e *ProcessToolExecutor) Execute(
	ctx context.Context,
	request appruntime.ToolRequest,
) (appruntime.ToolResult, error) {
	timeout := e.defaultTimeout
	if request.Tool.Timeout != "" {
		parsed, err := time.ParseDuration(request.Tool.Timeout)
		if err != nil {
			return appruntime.ToolResult{}, fmt.Errorf("parse tool timeout: %w", err)
		}
		timeout = parsed
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, request.Tool.Command, request.Tool.Args...)
	cmd.Dir = request.WorkDir
	cmd.Env = append([]string{}, request.Tool.Env...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error {
		if cmd.Process == nil {
			return nil
		}

		return syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	result := appruntime.ToolResult{
		StdoutSummary: appresult.Summarize(stdout.String(), toolOutputSummaryLimit),
		StderrSummary: appresult.Summarize(stderr.String(), toolOutputSummaryLimit),
	}
	result.ResultSummary = result.StdoutSummary
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if ctx.Err() == context.DeadlineExceeded {
		result.TimedOut = true
		return result, fmt.Errorf("tool %s timed out", request.Tool.Name)
	}
	if err != nil {
		return result, fmt.Errorf("tool %s exited with code %d: %w", request.Tool.Name, result.ExitCode, err)
	}

	return result, nil
}

func ToolResultObservationJSON(result appruntime.ToolResult) ([]byte, error) {
	return json.Marshal(map[string]any{
		"stdout":     result.StdoutSummary,
		"stderr":     result.StderrSummary,
		"result":     result.ResultSummary,
		"exit_code":  result.ExitCode,
		"timed_out":  result.TimedOut,
		"successful": result.ExitCode == 0 && !result.TimedOut,
	})
}
