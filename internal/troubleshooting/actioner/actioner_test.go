package actioner

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/digitalocean/droplet-agent/internal/metadata"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/command"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/file"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/mocks"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/otlp"

	"go.uber.org/mock/gomock"
)

type expectedCommand struct {
	command string
	args    []string
}

func TestTroubleshootingExporter_Do(t *testing.T) {
	tests := []struct {
		name     string
		metadata *metadata.Metadata

		expectCommandCalls     int
		expectedCommands       []expectedCommand
		commandCreationErrors  map[int]error // keyed by command index
		commandExecutionErrors map[int]error // keyed by command index

		expectFileCalls    int
		expectedFiles      []file.Config
		fileErrors         map[string]error // keyed by source (e.g., InvestigationArtifactSyslog)
		fileCreationErrors map[int]error    // keyed by file index

		metadataUpdateError error

		expectEmitErrorCalls int // expected number of EmitError calls
	}{
		{
			name: "no investigation - nil TroubleshootingAgent",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
			},
			expectCommandCalls: 0,
			expectFileCalls:    0,
		},
		{
			name: "no investigation - empty UUID",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "",
				},
			},
			expectCommandCalls: 0,
			expectFileCalls:    0,
		},
		{
			name: "single command artifact",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "2023-10-15T14:30:00Z",
					Requesting:        []string{InvestigationArtifactPs},
				},
			},
			expectCommandCalls: 1,
			expectFileCalls:    0,
			expectedCommands:   []expectedCommand{{command: "ps", args: []string{"aux"}}},
		},
		{
			name: "single file artifact",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "2023-10-01T11:00:00Z",
					Requesting:        []string{InvestigationArtifactSyslog},
				},
			},
			expectCommandCalls: 0,
			expectFileCalls:    1,
			expectedFiles: []file.Config{
				{
					Source:    InvestigationArtifactSyslog,
					LastLines: 100, // Default fallback
					TimeWindow: &file.TimeWindow{ // 15 minutes before and after trigger_at
						Start: time.Date(2023, 10, 1, 10, 45, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 1, 11, 15, 0, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "multiple artifacts - command and file",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "2023-10-15T15:00:00Z",
					Requesting:        []string{InvestigationArtifactPs, InvestigationArtifactMessages},
				},
			},
			expectCommandCalls: 1,
			expectFileCalls:    1,
			expectedCommands:   []expectedCommand{{command: "ps", args: []string{"aux"}}},
			expectedFiles: []file.Config{
				{
					Source:    InvestigationArtifactMessages,
					LastLines: 100,
					TimeWindow: &file.TimeWindow{
						Start: time.Date(2023, 10, 15, 14, 45, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 15, 15, 15, 0, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "invalid artifacts are filtered out",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "2023-10-15T16:00:00Z",
					Requesting:        []string{"invalid:artifact", InvestigationArtifactPs, "another:bad"},
				},
			},
			expectCommandCalls: 1, // Only ps is valid
			expectFileCalls:    0,
			expectedCommands:   []expectedCommand{{command: "ps", args: []string{"aux"}}},
		},
		{
			name: "invalid time format - artifacts still processed without time window",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "not-a-valid-time",
					Requesting:        []string{InvestigationArtifactSyslog},
				},
			},
			expectCommandCalls: 0,
			expectFileCalls:    1,
			expectedFiles: []file.Config{
				{
					Source:     InvestigationArtifactSyslog,
					LastLines:  100, // Fallback to tailing
					TimeWindow: nil, // No time window with invalid timestamp
				},
			},
		},
		{
			name: "multiple commands - ps and top",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "2023-10-15T16:30:00Z",
					Requesting:        []string{InvestigationArtifactPs, InvestigationArtifactTop},
				},
			},
			expectCommandCalls: 2,
			expectFileCalls:    0,
			expectedCommands: []expectedCommand{
				{command: "ps", args: []string{"aux"}},
				{command: "top", args: []string{"-bn", "1"}},
			},
		},
		{
			name: "multiple files - syslog and messages",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "2023-10-15T17:00:00Z",
					Requesting:        []string{InvestigationArtifactSyslog, InvestigationArtifactMessages},
				},
			},
			expectCommandCalls: 0,
			expectFileCalls:    2,
			expectedFiles: []file.Config{
				{
					Source:    InvestigationArtifactSyslog,
					LastLines: 100,
					TimeWindow: &file.TimeWindow{
						Start: time.Date(2023, 10, 15, 16, 45, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 15, 17, 15, 0, 0, time.UTC),
					},
				},
				{
					Source:    InvestigationArtifactMessages,
					LastLines: 100,
					TimeWindow: &file.TimeWindow{
						Start: time.Date(2023, 10, 15, 16, 45, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 15, 17, 15, 0, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "multiple artifacts of mixed types",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "2023-10-15T18:00:00Z",
					Requesting: []string{
						InvestigationArtifactPs,
						InvestigationArtifactSyslog,
						InvestigationArtifactTop,
						InvestigationArtifactMessages,
					},
				},
			},
			expectCommandCalls: 2,
			expectFileCalls:    2,
			expectedCommands: []expectedCommand{
				{command: "ps", args: []string{"aux"}},
				{command: "top", args: []string{"-bn", "1"}},
			},
			expectedFiles: []file.Config{
				{
					Source:    InvestigationArtifactSyslog,
					LastLines: 100,
					TimeWindow: &file.TimeWindow{
						Start: time.Date(2023, 10, 15, 17, 45, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 15, 18, 15, 0, 0, time.UTC),
					},
				},
				{
					Source:    InvestigationArtifactMessages,
					LastLines: 100,
					TimeWindow: &file.TimeWindow{
						Start: time.Date(2023, 10, 15, 17, 45, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 15, 18, 15, 0, 0, time.UTC),
					},
				},
			},
		},
		{
			name: "journalctl with time window",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "2023-10-15T19:00:00Z",
					Requesting:        []string{InvestigationArtifactJournalctl},
				},
			},
			expectCommandCalls: 1,
			expectFileCalls:    0,
			expectedCommands: []expectedCommand{
				{
					command: "journalctl",
					args: []string{
						"--no-pager",
						"--since=2023-10-15T18:45:00Z",
						"--until=2023-10-15T19:15:00Z",
					},
				},
			},
		},
		{
			name: "journalctl without time window",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid",
					TriggeredAt:       "invalid-time",
					Requesting:        []string{InvestigationArtifactJournalctl},
				},
			},
			expectCommandCalls: 1,
			expectFileCalls:    0,
			expectedCommands: []expectedCommand{
				{
					command: "journalctl",
					args:    []string{"--no-pager"}, // No time flags when TimeWindow is nil
				},
			},
		},
		{
			name: "file artifact not exist is not fatal - other artifacts still collected",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid-file-not-exist",
					TriggeredAt:       "2023-10-16T14:30:00Z",
					Requesting: []string{
						InvestigationArtifactSyslog,   // will return os.ErrNotExist
						InvestigationArtifactMessages, // will succeed
						InvestigationArtifactPs,       // will succeed
					},
				},
			},
			expectCommandCalls: 1, // ps command should run
			expectFileCalls:    2, // both syslog and messages attempted
			expectedCommands:   []expectedCommand{{command: "ps", args: []string{"aux"}}},
			expectedFiles: []file.Config{
				{
					Source:    InvestigationArtifactSyslog,
					LastLines: 100,
					TimeWindow: &file.TimeWindow{
						Start: time.Date(2023, 10, 16, 14, 15, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 16, 14, 45, 0, 0, time.UTC),
					},
				},
				{
					Source:    InvestigationArtifactMessages,
					LastLines: 100,
					TimeWindow: &file.TimeWindow{
						Start: time.Date(2023, 10, 16, 14, 15, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 16, 14, 45, 0, 0, time.UTC),
					},
				},
			},
			fileErrors: map[string]error{
				InvestigationArtifactSyslog: os.ErrNotExist,
			},
			expectEmitErrorCalls: 1, // EmitError called for file not exist
		},
		{
			name: "command runner creation failure emits error log",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid-cmd-create-fail",
					TriggeredAt:       "2023-10-16T15:00:00Z",
					Requesting:        []string{InvestigationArtifactPs, InvestigationArtifactTop},
				},
			},
			expectCommandCalls: 2, // both commands attempted
			expectFileCalls:    0,
			expectedCommands: []expectedCommand{
				{command: "ps", args: []string{"aux"}},       // first succeeds
				{command: "top", args: []string{"-bn", "1"}}, // second fails at creation
			},
			commandCreationErrors: map[int]error{
				1: errors.New("failed to create command runner"), // second command fails
			},
			expectEmitErrorCalls: 1, // EmitError called for command creation failure
		},
		{
			name: "command execution failure emits error log",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid-cmd-exec-fail",
					TriggeredAt:       "2023-10-16T15:30:00Z",
					Requesting:        []string{InvestigationArtifactPs},
				},
			},
			expectCommandCalls: 1,
			expectFileCalls:    0,
			expectedCommands:   []expectedCommand{{command: "ps", args: []string{"aux"}}},
			commandExecutionErrors: map[int]error{
				0: errors.New("command execution failed"),
			},
			expectEmitErrorCalls: 1, // EmitError called for command execution failure
		},
		{
			name: "file tailer creation failure emits error log",
			metadata: &metadata.Metadata{
				DropletID: 12345,
				Hostname:  "test-droplet",
				Region:    "nyc3",
				TroubleshootingAgent: &metadata.TroubleshootingAgent{
					InvestigationUUID: "test-uuid-file-create-fail",
					TriggeredAt:       "2023-10-16T16:00:00Z",
					Requesting:        []string{InvestigationArtifactSyslog, InvestigationArtifactMessages},
				},
			},
			expectCommandCalls: 0,
			expectFileCalls:    2,
			expectedFiles: []file.Config{
				{
					Source:    InvestigationArtifactSyslog,
					LastLines: 100,
					TimeWindow: &file.TimeWindow{
						Start: time.Date(2023, 10, 16, 15, 45, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 16, 16, 15, 0, 0, time.UTC),
					},
				},
				{
					Source:    InvestigationArtifactMessages,
					LastLines: 100,
					TimeWindow: &file.TimeWindow{
						Start: time.Date(2023, 10, 16, 15, 45, 0, 0, time.UTC),
						End:   time.Date(2023, 10, 16, 16, 15, 0, 0, time.UTC),
					},
				},
			},
			fileCreationErrors: map[int]error{
				0: errors.New("failed to create file tailer"), // first file fails
			},
			expectEmitErrorCalls: 1, // EmitError called for file creation failure
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			commandCallCount := 0
			fileCallCount := 0
			emitErrorCallCount := 0

			config := AgentConfig{
				Version:   "1.0.0-test",
				UserAgent: "test-agent",
			}
			exporter := NewTroubleshootingExporter(config)

			// Mock HTTP client to avoid real calls to completed endpoint
			mockHTTPClient := &mockHTTPClient{}
			exporter.httpClient = mockHTTPClient

			// Setup mock emitter to track EmitLog and EmitError calls
			mockEmitter := mocks.NewMockEmitter(ctrl)
			mockEmitter.EXPECT().EmitLog(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			// Flush should be called once if there are any artifacts to process
			if tt.expectCommandCalls > 0 || tt.expectFileCalls > 0 {
				mockEmitter.EXPECT().Flush(gomock.Any()).Times(1)
			}

			if tt.expectEmitErrorCalls > 0 {
				mockEmitter.EXPECT().EmitError(gomock.Any(), gomock.Any(), gomock.Any()).Times(tt.expectEmitErrorCalls).Do(func(ctx context.Context, component, msg string) {
					emitErrorCallCount++
				})
			}
			exporter.newOTLPClient = func(ctx context.Context, config otlp.ClientConfig) (otlp.Emitter, error) {
				return mockEmitter, nil
			}

			exporter.newCommandRunner = func(cmdConfig command.Config, emitter otlp.Emitter) (command.Command, error) {
				if emitter == nil {
					t.Error("Expected non-nil emitter passed to newCommandRunner")
				}

				currentIndex := commandCallCount

				// Check for creation error
				if tt.commandCreationErrors != nil {
					if err, hasError := tt.commandCreationErrors[currentIndex]; hasError {
						commandCallCount++
						return nil, err
					}
				}

				// Validate command against expected commands
				if commandCallCount >= len(tt.expectedCommands) {
					t.Errorf("Unexpected command call #%d, only expected %d command(s)", commandCallCount+1, len(tt.expectedCommands))
					commandCallCount++
					return nil, errors.New("unexpected command call")
				}

				commandCallCount++

				mockCmd := mocks.NewMockCommand(ctrl)

				// Check for execution error
				var execErr error
				if tt.commandExecutionErrors != nil {
					if err, hasError := tt.commandExecutionErrors[currentIndex]; hasError {
						execErr = err
					}
				}

				mockCmd.EXPECT().Run(gomock.Any()).Return(execErr)

				return mockCmd, nil
			}

			exporter.newFileTailer = func(cfg file.Config, emitter otlp.Emitter) (file.File, error) {
				if emitter == nil {
					t.Error("Expected non-nil emitter passed to newFileTailer")
				}

				currentIndex := fileCallCount

				// Check for creation error
				if tt.fileCreationErrors != nil {
					if err, hasError := tt.fileCreationErrors[currentIndex]; hasError {
						fileCallCount++
						return nil, err
					}
				}

				// Validate file config against expected files
				if fileCallCount >= len(tt.expectedFiles) {
					t.Errorf("Unexpected file call #%d, only expected %d file(s)", fileCallCount+1, len(tt.expectedFiles))
					fileCallCount++
					return nil, errors.New("unexpected file call")
				}

				expectedFile := tt.expectedFiles[fileCallCount]
				fileCallCount++

				// Validate file config directly
				if expectedFile.Source != "" && cfg.Source != expectedFile.Source {
					t.Errorf("Expected source %s, got %s", expectedFile.Source, cfg.Source)
				}
				if expectedFile.LastLines > 0 && cfg.LastLines != expectedFile.LastLines {
					t.Errorf("Expected LastLines %d, got %d", expectedFile.LastLines, cfg.LastLines)
				}
				if expectedFile.TimeWindow != nil {
					if cfg.TimeWindow == nil {
						t.Error("Expected time window to be set")
					} else {
						if !cfg.TimeWindow.Start.Equal(expectedFile.TimeWindow.Start) {
							t.Errorf("Expected start %v, got %v", expectedFile.TimeWindow.Start, cfg.TimeWindow.Start)
						}
						if !cfg.TimeWindow.End.Equal(expectedFile.TimeWindow.End) {
							t.Errorf("Expected end %v, got %v", expectedFile.TimeWindow.End, cfg.TimeWindow.End)
						}
					}
				} else if cfg.TimeWindow != nil {
					t.Errorf("Expected no TimeWindow, but got %+v", cfg.TimeWindow)
				}

				mockFile := mocks.NewMockFile(ctrl)

				// Check if this file should return an error
				var tailErr error
				if tt.fileErrors != nil {
					if err, hasError := tt.fileErrors[cfg.Source]; hasError {
						tailErr = err
					}
				}

				mockFile.EXPECT().Tail(gomock.Any()).Return(tailErr)

				return mockFile, nil
			}

			exporter.Do(tt.metadata)

			if commandCallCount != tt.expectCommandCalls {
				t.Errorf("Expected %d command calls, got %d", tt.expectCommandCalls, commandCallCount)
			}
			if fileCallCount != tt.expectFileCalls {
				t.Errorf("Expected %d file calls, got %d", tt.expectFileCalls, fileCallCount)
			}
			if emitErrorCallCount != tt.expectEmitErrorCalls {
				t.Errorf("Expected %d EmitError calls, got %d", tt.expectEmitErrorCalls, emitErrorCallCount)
			}
		})
	}
}

func TestTroubleshootingExporter_Concurrency(t *testing.T) {
	workDuration := 100 * time.Millisecond
	tests := []struct {
		name           string
		investigations []string // UUIDs to launch
		expectedCount  int      // expected number of commands that run
		minConcurrent  int      // minimum concurrent executions (0 = no check)
	}{
		{
			name:           "duplicate investigations skipped when concurrent",
			investigations: []string{"uuid-1", "uuid-1", "uuid-1", "uuid-1", "uuid-1"},
			expectedCount:  1, // only first one runs
		},
		{
			name:           "different investigations run concurrently",
			investigations: []string{"uuid-1", "uuid-2", "uuid-3"},
			expectedCount:  3, // all 3 run
			minConcurrent:  2, // at least 2 should overlap
		},
		{
			name:           "duplicate blocked even with multiple concurrent investigations",
			investigations: []string{"uuid-1", "uuid-2", "uuid-1"},
			expectedCount:  2, // uuid-1, uuid-2 run; duplicate uuid-1 blocked
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			config := AgentConfig{
				Version:   "1.0.0-test",
				UserAgent: "test-agent",
			}
			exporter := NewTroubleshootingExporter(config)

			// Mock HTTP client to avoid real calls to completed endpoint
			mockHTTPClient := &mockHTTPClient{}
			exporter.httpClient = mockHTTPClient

			var mu sync.Mutex
			var commandCallCount int
			var maxConcurrent int
			var currentConcurrent int

			exporter.newCommandRunner = func(cmdConfig command.Config, emitter otlp.Emitter) (command.Command, error) {
				mu.Lock()
				commandCallCount++
				currentConcurrent++
				if currentConcurrent > maxConcurrent {
					maxConcurrent = currentConcurrent
				}
				mu.Unlock()

				time.Sleep(workDuration)

				mu.Lock()
				currentConcurrent--
				mu.Unlock()

				mockCmd := mocks.NewMockCommand(ctrl)
				mockCmd.EXPECT().Run(gomock.Any()).Return(nil).AnyTimes()
				return mockCmd, nil
			}

			exporter.newFileTailer = func(cfg file.Config, emitter otlp.Emitter) (file.File, error) {
				mockFile := mocks.NewMockFile(ctrl)
				mockFile.EXPECT().Tail(gomock.Any()).Return(nil).AnyTimes()
				return mockFile, nil
			}

			var wg sync.WaitGroup
			for _, uuid := range tt.investigations {
				wg.Add(1)
				go func(investigationUUID string) {
					defer wg.Done()
					md := &metadata.Metadata{
						DropletID: 12345,
						Hostname:  "test-droplet",
						Region:    "nyc3",
						TroubleshootingAgent: &metadata.TroubleshootingAgent{
							InvestigationUUID: investigationUUID,
							TriggeredAt:       "2023-10-15T14:30:00Z",
							Requesting:        []string{InvestigationArtifactPs},
						},
					}
					exporter.Do(md)
				}(uuid)
			}
			wg.Wait()

			if commandCallCount != tt.expectedCount {
				t.Errorf("Expected %d commands to run, but got %d", tt.expectedCount, commandCallCount)
			}

			if tt.minConcurrent > 0 && maxConcurrent < tt.minConcurrent {
				t.Errorf("Expected at least %d concurrent executions, but got %d", tt.minConcurrent, maxConcurrent)
			}
		})
	}
}

func TestTroubleshootingExporter_markInvestigationReady(t *testing.T) {
	tests := []struct {
		name              string
		investigationUUID string
		serverBehavior    func(attempts *int) http.HandlerFunc
		expectError       bool
		expectedAttempts  int
	}{
		{
			name:              "successful on first attempt",
			investigationUUID: "test-uuid-001",
			serverBehavior: func(attempts *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attempts++
					if r.Method != http.MethodPost {
						t.Errorf("Expected POST method, got %s", r.Method)
					}
					if uuid := r.Header.Get("X-Investigation-UUID"); uuid != "test-uuid-001" {
						t.Errorf("Expected X-Investigation-UUID header to be test-uuid-001, got %s", uuid)
					}
					w.WriteHeader(http.StatusAccepted)
				}
			},
			expectError:      false,
			expectedAttempts: 1,
		},
		{
			name:              "retry on server error then success",
			investigationUUID: "test-uuid-002",
			serverBehavior: func(attempts *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attempts++
					if *attempts < 3 {
						w.WriteHeader(http.StatusInternalServerError)
						return
					}
					w.WriteHeader(http.StatusAccepted)
				}
			},
			expectError:      false,
			expectedAttempts: 3,
		},
		{
			name:              "other 2xx status codes are success",
			investigationUUID: "test-uuid-004",
			serverBehavior: func(attempts *int) http.HandlerFunc {
				return func(w http.ResponseWriter, r *http.Request) {
					*attempts++
					w.WriteHeader(http.StatusOK)
				}
			},
			expectError:      false,
			expectedAttempts: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attempts := 0
			server := httptest.NewServer(tt.serverBehavior(&attempts))
			defer server.Close()

			exporter := &TroubleshootingExporter{
				CompletedEndpoint: server.URL,
				httpClient:        server.Client(),
			}

			ctx := context.Background()
			err := exporter.markInvestigationReady(ctx, tt.investigationUUID)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
			if attempts != tt.expectedAttempts {
				t.Errorf("Expected %d attempts, but got %d", tt.expectedAttempts, attempts)
			}
		})
	}
}

// mockHTTPClient is a simple mock that returns success for any HTTP request
type mockHTTPClient struct{}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       http.NoBody,
	}, nil
}
