package command

import (
	"context"
	"fmt"
	"io"
	"os/exec"
)

// CommandExecutor defines the interface for executing commands.
// This allows for testing with mock executors.
type CommandExecutor interface {
	// Exec runs a command and returns its stdout reader
	Exec(ctx context.Context, name string, args []string) (io.ReadCloser, error)
}

// commandExecutor implements CommandExecutor using os/exec
type commandExecutor struct{}

// Exec runs a real system command
func (e *commandExecutor) Exec(ctx context.Context, name string, args []string) (io.ReadCloser, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start command: %w", err)
	}
	// Return a wrapper that waits for the command on close
	return &commandReader{
		ReadCloser: stdout,
		cmd:        cmd,
	}, nil
}

// commandReader wraps stdout and waits for command completion on close
type commandReader struct {
	io.ReadCloser
	cmd *exec.Cmd
}

func (r *commandReader) Close() error {
	// Close the stdout pipe first
	if err := r.ReadCloser.Close(); err != nil {
		return err
	}
	// Then wait for the command to finish
	return r.cmd.Wait()
}
