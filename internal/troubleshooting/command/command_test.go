package command

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/droplet-agent/internal/troubleshooting/file"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/mocks"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/parser"

	"go.uber.org/mock/gomock"
)

func TestNewRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmitter := mocks.NewMockEmitter(ctrl)

	tests := []struct {
		name        string
		source      string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid command - top",
			source:      CmdPrefix + "top",
			expectError: false,
		},
		{
			name:        "valid command - ps",
			source:      CmdPrefix + "ps",
			expectError: false,
		},
		{
			name:        "empty command",
			source:      CmdPrefix,
			expectError: true,
			errorMsg:    "command cannot be empty",
		},
		{
			name:        "empty command with whitespace",
			source:      CmdPrefix + "   ",
			expectError: true,
			errorMsg:    "command cannot be empty",
		},
		{
			name:        "command not in allowlist",
			source:      CmdPrefix + "rm",
			expectError: true,
			errorMsg:    "is not in the allowlist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := NewRunner(Config{Source: tt.source}, mockEmitter)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorMsg, err)
				}
				if cmd != nil {
					t.Error("Expected nil command on error")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
				if cmd == nil {
					t.Error("Expected non-nil command")
				}
			}
		})
	}
}

func TestCommand_Run(t *testing.T) {
	tests := []struct {
		name              string
		command           string
		expectedPath      string
		commandOutput     string
		executorError     error
		expectError       bool
		errorContains     string
		verifyLineContent bool
		verifyReaderClose bool
	}{
		{
			name:         "ps command with output",
			command:      "ps",
			expectedPath: "/usr/bin/ps",
			commandOutput: `USER       PID %CPU %MEM    VSZ   RSS TTY      STAT START   TIME COMMAND
root         1  0.0  0.1 169540 13940 ?        Ss   Oct14   0:02 /sbin/init
root         2  0.0  0.0      0     0 ?        S    Oct14   0:00 [kthreadd]`,
			expectError:       false,
			verifyLineContent: true,
			verifyReaderClose: true,
		},
		{
			name:         "top command with output",
			command:      "top",
			expectedPath: "/usr/bin/top",
			commandOutput: `top - 10:00:00 up 1 day
Tasks: 123 total
%Cpu(s):  2.3 us,  1.2 sy
KiB Mem : 16384000 total`,
			expectError:       false,
			verifyLineContent: true,
		},
		{
			name:         "journalctl command with output",
			command:      "journalctl",
			expectedPath: "/usr/bin/journalctl",
			commandOutput: `-- Logs begin at Mon 2023-10-15 10:00:00 UTC, end at Mon 2023-10-15 11:00:00 UTC --
Oct 15 10:15:23 droplet systemd[1]: Started Session 1 of user root.
Oct 15 10:30:45 droplet sshd[1234]: Accepted publickey for root from 192.168.1.1
Oct 15 10:45:12 droplet kernel: [12345.678] Out of memory: Kill process 5678`,
			expectError:       false,
			verifyLineContent: true,
		},
		{
			name:          "executor error",
			command:       "ps",
			expectedPath:  "/usr/bin/ps",
			executorError: io.ErrClosedPipe,
			expectError:   true,
			errorContains: "failed to execute command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockEmitter := mocks.NewMockEmitter(ctrl)
			mockExecutor := mocks.NewMockCommandExecutor(ctrl)

			var mockRdr *mockReader
			if tt.executorError != nil {
				mockExecutor.EXPECT().
					Exec(gomock.Any(), tt.expectedPath, gomock.Any()).
					Return(nil, tt.executorError).Times(1)
				mockEmitter.EXPECT().
					EmitLog(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			} else {
				mockRdr = newMockReader(tt.commandOutput)
				mockExecutor.EXPECT().
					Exec(gomock.Any(), tt.expectedPath, gomock.Any()).Return(mockRdr, nil).Times(1)

				expectedLines := strings.Split(tt.commandOutput, "\n")
				for i, expectedLine := range expectedLines {
					call := mockEmitter.EXPECT().
						EmitLog(gomock.Any(), CmdPrefix+tt.command, gomock.Any()).Times(1)

					if tt.verifyLineContent {
						call.Do(func(ctx context.Context, sourceFile string, entry parser.LogEntry) {
							if entry.Original != expectedLine {
								t.Errorf("Line %d mismatch. Got %q, want %q", i, entry.Original, expectedLine)
							}
							if entry.Timestamp.IsZero() {
								t.Errorf("Line %d: Timestamp should not be zero", i)
							}
						})
					}
				}
			}

			cmd, err := NewRunnerWithExecutor(Config{Source: CmdPrefix + tt.command}, mockEmitter, mockExecutor)
			if err != nil {
				t.Fatalf("Failed to create command: %v", err)
			}

			err = cmd.Run(context.Background())

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got nil")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			if tt.verifyReaderClose && mockRdr != nil && !mockRdr.closeCalled {
				t.Error("Expected reader Close() to be called")
			}
		})
	}
}

func TestCommand_Run_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmitter := mocks.NewMockEmitter(ctrl)
	mockExecutor := mocks.NewMockCommandExecutor(ctrl)

	// Simulate a long-running command
	longOutput := strings.Repeat("line\n", 1000)
	mockReader := newMockReader(longOutput)

	mockExecutor.EXPECT().
		Exec(gomock.Any(), "/usr/bin/ps", []string{"aux"}).
		Return(mockReader, nil).
		Times(1)

	// May emit some lines before cancellation is detected
	mockEmitter.EXPECT().
		EmitLog(gomock.Any(), CmdPrefix+"ps", gomock.Any()).
		AnyTimes()

	cmd, err := NewRunnerWithExecutor(Config{Source: CmdPrefix + "ps"}, mockEmitter, mockExecutor)
	if err != nil {
		t.Fatalf("Failed to create command: %v", err)
	}

	// Create a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = cmd.Run(ctx)
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected 'context canceled' error, got: %v", err)
	}
}

func TestCommand_AllowedCommands(t *testing.T) {
	// Verify that the allowlist contains expected commands with proper args
	tests := []struct {
		command      string
		expectedName string
		expectedPath string
		expectedArgs []string
	}{
		{
			command:      "top",
			expectedName: "top",
			expectedPath: "/usr/bin/top",
			expectedArgs: []string{"-bn", "1"},
		},
		{
			command:      "ps",
			expectedName: "ps",
			expectedPath: "/usr/bin/ps",
			expectedArgs: []string{"aux"},
		},
		{
			command:      "journalctl",
			expectedName: "journalctl",
			expectedPath: "/usr/bin/journalctl",
			expectedArgs: []string{"--no-pager"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			spec, exists := allowedCommands[tt.command]
			if !exists {
				t.Errorf("Command %q not found in allowlist", tt.command)
				return
			}

			if spec.name != tt.expectedName {
				t.Errorf("Name mismatch for %q. Got %q, want %q", tt.command, spec.name, tt.expectedName)
			}

			if spec.path != tt.expectedPath {
				t.Errorf("Path mismatch for %q. Got %q, want %q", tt.command, spec.path, tt.expectedPath)
			}

			if len(spec.args) != len(tt.expectedArgs) {
				t.Errorf("Args length mismatch for %q. Got %d, want %d", tt.command, len(spec.args), len(tt.expectedArgs))
				return
			}

			for i, arg := range spec.args {
				if arg != tt.expectedArgs[i] {
					t.Errorf("Arg %d mismatch for %q. Got %q, want %q", i, tt.command, arg, tt.expectedArgs[i])
				}
			}
		})
	}
}

func TestNewRunnerWithConfig_TimeWindow(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmitter := mocks.NewMockEmitter(ctrl)
	mockExecutor := mocks.NewMockCommandExecutor(ctrl)
	mockTime := time.Date(2023, 10, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name         string
		config       Config
		expectedArgs []string
	}{
		{
			name: "journalctl with TimeWindow",
			config: Config{
				Source: "command:journalctl",
				TimeWindow: &file.TimeWindow{
					Start: time.Date(2023, 10, 15, 10, 0, 0, 0, time.UTC),
					End:   time.Date(2023, 10, 15, 11, 0, 0, 0, time.UTC),
				},
			},
			expectedArgs: []string{
				"--no-pager",
				"--since=2023-10-15T10:00:00Z",
				"--until=2023-10-15T11:00:00Z",
			},
		},
		{
			name: "journalctl without TimeWindow",
			config: Config{
				Source:     "command:journalctl",
				TimeWindow: nil,
				timeNow:    func() time.Time { return mockTime },
			},
			expectedArgs: []string{"--no-pager", "--since=2023-10-15T11:45:00Z"},
		},
		{
			name: "ps command ignores TimeWindow",
			config: Config{
				Source: "command:ps",
				TimeWindow: &file.TimeWindow{
					Start: time.Date(2023, 10, 15, 10, 0, 0, 0, time.UTC),
					End:   time.Date(2023, 10, 15, 11, 0, 0, 0, time.UTC),
				},
			},
			expectedArgs: []string{"aux"}, // TimeWindow should be ignored for ps
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := NewRunnerWithExecutor(tt.config, mockEmitter, mockExecutor)
			if err != nil {
				t.Fatalf("Failed to create command: %v", err)
			}

			// Access the internal commandImpl to verify args
			cmdImpl, ok := cmd.(*commandImpl)
			if !ok {
				t.Fatal("Expected *commandImpl type")
			}

			if len(cmdImpl.commandArgs) != len(tt.expectedArgs) {
				t.Errorf("Args length mismatch. Got %d, want %d\nGot: %v\nWant: %v",
					len(cmdImpl.commandArgs), len(tt.expectedArgs),
					cmdImpl.commandArgs, tt.expectedArgs)
				return
			}

			for i, arg := range cmdImpl.commandArgs {
				if arg != tt.expectedArgs[i] {
					t.Errorf("Arg %d mismatch. Got %q, want %q", i, arg, tt.expectedArgs[i])
				}
			}
		})
	}
}

// mockReader implements io.ReadCloser for testing
type mockReader struct {
	*strings.Reader
	closeCalled bool
}

func (m *mockReader) Close() error {
	m.closeCalled = true
	return nil
}

func newMockReader(content string) *mockReader {
	return &mockReader{
		Reader: strings.NewReader(content),
	}
}
