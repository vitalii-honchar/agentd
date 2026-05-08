package runtime

import (
	"bytes"
	"context"
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
		StdoutSummary: appresult.Summarize(stdout.String(), appresult.DefaultSummaryLimit),
		StderrSummary: appresult.Summarize(stderr.String(), appresult.DefaultSummaryLimit),
	}
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
