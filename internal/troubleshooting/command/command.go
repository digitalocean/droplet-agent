package command

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/digitalocean/droplet-agent/internal/troubleshooting/file"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/otlp"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/parser"
)

const (
	// CmdPrefix is the prefix for command sources.
	CmdPrefix = "command:"
)

// commandSpec defines a command with its executable path and required arguments
type commandSpec struct {
	name string
	path string
	args []string
}

var (
	// allowedCommands defines the allowlist of commands that can be executed.
	// The map keys are the command names (e.g., "top"), and the values contain
	// the executable path and required arguments for safety and predictability.
	allowedCommands = map[string]commandSpec{
		"top": {
			name: "top",
			path: "/usr/bin/top",
			args: []string{"-bn", "1"},
		},
		"ps": {
			name: "ps",
			path: "/usr/bin/ps",
			args: []string{"aux"},
		},
		"journalctl": {
			name: "journalctl",
			path: "/usr/bin/journalctl",
			args: []string{"--no-pager", "--output=short-iso"},
		},
	}
)

// Config holds configuration for creating a command runner.
type Config struct {
	// Source is the command source (e.g., "command:journalctl")
	Source string
	// TimeWindow optionally specifies time bounds for commands that support it (e.g., journalctl)
	TimeWindow *file.TimeWindow
	// timeNow allows for mocking time in tests
	timeNow func() time.Time
}

// Command represents a command that can be run and emit logs.
type Command interface {
	Run(context.Context) error
}

// Ensure commandImpl implements Command interface
var _ Command = (*commandImpl)(nil)

type commandImpl struct {
	client      otlp.Emitter
	source      string
	spec        commandSpec
	commandArgs []string
	timeWindow  *file.TimeWindow
	executor    CommandExecutor
}

// NewRunner creates a new Command instance with the default executor.
func NewRunner(config Config, client otlp.Emitter) (Command, error) {
	return NewRunnerWithExecutor(config, client, &commandExecutor{})
}

// NewRunnerWithExecutor creates a new Command instance with configuration and a custom executor.
// This is useful for testing with mock executors.
func NewRunnerWithExecutor(config Config, client otlp.Emitter, executor CommandExecutor) (Command, error) {
	// Set default time function if not provided
	if config.timeNow == nil {
		config.timeNow = time.Now
	}

	command := strings.TrimPrefix(config.Source, CmdPrefix)
	commandName := strings.TrimSpace(command)
	if commandName == "" {
		return nil, errors.New("command cannot be empty")
	}

	// Validate against allowlist and get command spec
	spec, exists := allowedCommands[commandName]
	if !exists {
		return nil, fmt.Errorf("command '%s' is not in the allowlist", commandName)
	}

	// Make a copy of the base args to avoid modifying the allowlist
	commandArgs := append([]string{}, spec.args...)

	// Add time-specific arguments for journalctl if TimeWindow is provided
	if commandName == "journalctl" {
		if config.TimeWindow != nil {
			// Format times using RFC3339 for journalctl compatibility
			sinceTime := config.TimeWindow.Start.Format(time.RFC3339)
			untilTime := config.TimeWindow.End.Format(time.RFC3339)
			commandArgs = append(commandArgs, "--since="+sinceTime, "--until="+untilTime)
		} else {
			// Default to last 15 minutes if no time window is provided
			lastFifteen := config.timeNow().Add(-15 * time.Minute).Format(time.RFC3339)
			commandArgs = append(commandArgs, "--since="+lastFifteen)
		}
	}

	return &commandImpl{
		client:      client,
		source:      config.Source,
		spec:        spec,
		commandArgs: commandArgs,
		timeWindow:  config.TimeWindow,
		executor:    executor,
	}, nil
}

// Run a command and emits its output as log lines.
// The command must be in the allowlist, and the arguments are predefined.
func (c *commandImpl) Run(ctx context.Context) error {
	stdout, err := c.executor.Exec(ctx, c.spec.path, c.commandArgs)
	if err != nil {
		return fmt.Errorf("failed to execute command '%s %s': %w", c.spec.name, strings.Join(c.commandArgs, " "), err)
	}
	defer func() { _ = stdout.Close() }()

	// Read and send lines
	scanner := bufio.NewScanner(stdout)
	logParser := parser.NewLogParser()
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			line := scanner.Text()
			parsed := logParser.ParseLine(line)
			c.client.EmitLog(ctx, c.source, parsed)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading output from command '%s': %w", c.spec.name, err)
	}

	// Close will trigger cmd.Wait() in the commandReader
	if err := stdout.Close(); err != nil {
		var exitError *exec.ExitError
		if errors.As(err, &exitError) {
			return fmt.Errorf("command '%s %s' exited with code %d: %w",
				c.spec.name, strings.Join(c.commandArgs, " "), exitError.ExitCode(), err)
		}
		return fmt.Errorf("command '%s %s' failed: %w", c.spec.name, strings.Join(c.commandArgs, " "), err)
	}

	return nil
}
