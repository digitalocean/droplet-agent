package file

import (
	"bufio"
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/digitalocean/droplet-agent/internal/troubleshooting/mocks"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/parser"

	"go.uber.org/mock/gomock"
)

func TestFileTailer_Tail(t *testing.T) {
	tests := []struct {
		name             string
		logLines         []string
		timeWindow       *TimeWindow
		lastLines        int
		expectedLines    []string
		verifyTimestamps bool
		verifyTimeWindow bool
	}{
		{
			name: "stream all lines - no filtering",
			logLines: []string{
				"2023-10-01T10:00:00Z First log entry",
				"2023-10-01T11:00:00Z Second log entry",
				"2023-10-01T12:00:00Z Third log entry",
			},
			timeWindow: nil,
			lastLines:  0,
			expectedLines: []string{
				"2023-10-01T10:00:00Z First log entry",
				"2023-10-01T11:00:00Z Second log entry",
				"2023-10-01T12:00:00Z Third log entry",
			},
			verifyTimestamps: true,
		},
		{
			name: "stream last N lines only",
			logLines: []string{
				"2023-10-01T10:00:00Z First log entry",
				"2023-10-01T11:00:00Z Second log entry",
				"2023-10-01T12:00:00Z Third log entry",
				"2023-10-01T13:00:00Z Fourth log entry",
				"2023-10-01T14:00:00Z Fifth log entry",
			},
			timeWindow: nil,
			lastLines:  2,
			expectedLines: []string{
				"2023-10-01T13:00:00Z Fourth log entry",
				"2023-10-01T14:00:00Z Fifth log entry",
			},
			verifyTimestamps: true,
		},
		{
			name: "time window filtering - includes within window",
			logLines: []string{
				"2023-10-01T10:00:00Z Before window",
				"2023-10-01T11:00:00Z Start of window",
				"2023-10-01T11:30:00Z Middle of window",
				"2023-10-01T12:00:00Z End of window",
				"2023-10-01T13:00:00Z After window",
			},
			timeWindow: &TimeWindow{
				Start: *timeMustParse("2023-10-01T11:00:00Z"),
				End:   *timeMustParse("2023-10-01T12:00:00Z"),
			},
			lastLines: 0,
			expectedLines: []string{
				"2023-10-01T11:00:00Z Start of window",
				"2023-10-01T11:30:00Z Middle of window",
				"2023-10-01T12:00:00Z End of window",
			},
			verifyTimestamps: true,
			verifyTimeWindow: true,
		},
		{
			name: "time filtering with timestamps - ignores lastLines setting",
			logLines: []string{
				"2023-10-01T10:00:00Z First log entry",
				"2023-10-01T11:00:00Z Second log entry",
				"2023-10-01T12:00:00Z Third log entry",
			},
			timeWindow: &TimeWindow{
				Start: *timeMustParse("2023-10-01T10:30:00Z"),
				End:   *timeMustParse("2023-10-01T12:30:00Z"),
			},
			lastLines: 1, // Should be ignored since timestamps are detected
			expectedLines: []string{
				"2023-10-01T11:00:00Z Second log entry",
				"2023-10-01T12:00:00Z Third log entry",
			},
			verifyTimestamps: true,
		},
		{
			name: "time filtering without timestamps - falls back to lastLines",
			logLines: []string{
				"No timestamp - First log entry",
				"No timestamp - Second log entry",
				"No timestamp - Third log entry",
				"No timestamp - Fourth log entry",
			},
			timeWindow: &TimeWindow{
				Start: *timeMustParse("2023-10-01T10:30:00Z"),
				End:   *timeMustParse("2023-10-01T12:30:00Z"),
			},
			lastLines: 2,
			expectedLines: []string{
				"No timestamp - Third log entry",
				"No timestamp - Fourth log entry",
			},
			verifyTimestamps: false,
		},
		{
			name: "time filtering without timestamps and no lastLines - emits nothing",
			logLines: []string{
				"No timestamp - First log entry",
				"No timestamp - Second log entry",
			},
			timeWindow: &TimeWindow{
				Start: *timeMustParse("2023-10-01T10:30:00Z"),
				End:   *timeMustParse("2023-10-01T12:30:00Z"),
			},
			lastLines: 0,
			// No expectedLines - nothing should be emitted when time filtering
			// is requested but no timestamps found and no lastLines fallback
			expectedLines:    []string{},
			verifyTimestamps: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockEmitter := mocks.NewMockEmitter(ctrl)

			// Create temporary file
			tmpFile, err := os.CreateTemp("", "test_log_*.log")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if len(tt.logLines) > 0 {
				content := strings.Join(tt.logLines, "\n")
				if _, err := tmpFile.WriteString(content); err != nil {
					t.Fatalf("Failed to write to temp file: %v", err)
				}
			}
			tmpFile.Close()

			// Set up expectations for each expected line
			for i, expectedLine := range tt.expectedLines {
				call := mockEmitter.EXPECT().
					EmitLog(gomock.Any(), FilePrefix+tmpFile.Name(), gomock.Any()).
					Times(1)

				if tt.verifyTimestamps || tt.verifyTimeWindow {
					lineIdx := i
					call.Do(func(ctx context.Context, sourceFile string, entry parser.LogEntry) {
						if entry.Original != expectedLine {
							t.Errorf("Line %d mismatch. Got %q, want %q", lineIdx, entry.Original, expectedLine)
						}
						if tt.verifyTimestamps && !entry.TimestampParsed {
							t.Errorf("Line %d: Expected timestamp to be parsed", lineIdx)
						}
						if tt.verifyTimeWindow && entry.TimestampParsed {
							if entry.Timestamp.Before(tt.timeWindow.Start) || entry.Timestamp.After(tt.timeWindow.End) {
								t.Errorf("Line %d: Timestamp %v is outside window [%v, %v]",
									lineIdx, entry.Timestamp, tt.timeWindow.Start, tt.timeWindow.End)
							}
						}
					})
				}
			}

			config := Config{
				Source:     FilePrefix + tmpFile.Name(),
				LastLines:  tt.lastLines,
				TimeWindow: tt.timeWindow,
			}

			tailer, err := NewFileTailer(config, mockEmitter)
			if err != nil {
				t.Fatalf("Failed to create tailer: %v", err)
			}

			if err := tailer.Tail(context.Background()); err != nil {
				t.Errorf("Tail() failed: %v", err)
			}
		})
	}
}

func TestFileTailer_Errors(t *testing.T) {
	tests := []struct {
		name           string
		source         string
		expectedErrMsg string
	}{
		{
			name:           "nonexistent file",
			source:         FilePrefix + "/nonexistent/file.log",
			expectedErrMsg: "does not exist",
		},
		{
			name:           "empty source path",
			source:         FilePrefix,
			expectedErrMsg: "cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockEmitter := mocks.NewMockEmitter(ctrl)

			config := Config{
				Source:     tt.source,
				LastLines:  0,
				TimeWindow: nil,
			}

			_, err := NewFileTailer(config, mockEmitter)
			if err == nil {
				t.Error("Expected error but got nil")
			} else if !strings.Contains(err.Error(), tt.expectedErrMsg) {
				t.Errorf("Expected error containing %q, got: %v", tt.expectedErrMsg, err)
			}
		})
	}
}

func TestFileTailer_ContextCancellation(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockEmitter := mocks.NewMockEmitter(ctrl)

	tmpFile, err := os.CreateTemp("", "test_log_*.log")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, "2023-10-01T10:00:00Z Log entry")
	}
	content := strings.Join(lines, "\n")
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after first emission
	callCount := 0
	mockEmitter.EXPECT().
		EmitLog(gomock.Any(), FilePrefix+tmpFile.Name(), gomock.Any()).
		Do(func(ctx context.Context, sourceFile string, entry parser.LogEntry) {
			callCount++
			if callCount == 1 {
				cancel()
			}
		}).
		AnyTimes()

	config := Config{
		Source:     FilePrefix + tmpFile.Name(),
		LastLines:  0,
		TimeWindow: nil,
	}

	tailer, err := NewFileTailer(config, mockEmitter)
	if err != nil {
		t.Fatalf("Failed to create tailer: %v", err)
	}

	err = tailer.Tail(ctx)
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
	if !strings.Contains(err.Error(), "context canceled") {
		t.Errorf("Expected 'context canceled' error, got: %v", err)
	}
}

func TestFileTailer_BufferOverflowReturnsError(t *testing.T) {
	tests := []struct {
		name            string
		fileContent     string
		lastLines       int
		timeWindow      *TimeWindow
		expectedErr     error
		expectEmitCount int
	}{
		{
			name: "line exceeds 1MB buffer limit - stream all",
			fileContent: "Normal line 1\n" +
				strings.Repeat("x", 1024*1024+100) + "\n" + // Exceeds 1MB limit
				"Normal line 3\n",
			lastLines:       0,
			expectedErr:     bufio.ErrTooLong,
			expectEmitCount: 1,
		},
		{
			name: "line exceeds 1MB buffer limit - last N lines",
			fileContent: "Normal line 1\n" +
				strings.Repeat("y", 1024*1024+50) + "\n" + // Exceeds 1MB limit
				"Normal line 3\n",
			lastLines:       2,
			expectedErr:     bufio.ErrTooLong,
			expectEmitCount: 1,
		},
		{
			name: "line exceeds 1MB buffer limit - time window",
			fileContent: "2023-10-01T11:30:00Z Normal line 1\n" +
				"2023-10-01T11:30:00Z " + strings.Repeat("y", 1024*1024+50) + "\n" + // Exceeds 1MB limit
				"2023-10-01T11:30:00Z  Normal line 3\n",
			timeWindow: &TimeWindow{
				Start: *timeMustParse("2023-10-01T11:00:00Z"),
				End:   *timeMustParse("2023-10-01T12:00:00Z"),
			},
			expectedErr:     bufio.ErrTooLong,
			expectEmitCount: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockEmitter := mocks.NewMockEmitter(ctrl)

			tmpFile, err := os.CreateTemp("", "test_overflow_*.log")
			if err != nil {
				t.Fatalf("Failed to create temp file: %v", err)
			}
			defer os.Remove(tmpFile.Name())

			if _, err := tmpFile.WriteString(tt.fileContent); err != nil {
				t.Fatalf("Failed to write to temp file: %v", err)
			}
			tmpFile.Close()

			mockEmitter.EXPECT().EmitLog(gomock.Any(),
				FilePrefix+tmpFile.Name(),
				gomock.Any(),
			).Times(tt.expectEmitCount)

			config := Config{
				Source:     FilePrefix + tmpFile.Name(),
				LastLines:  tt.lastLines,
				TimeWindow: tt.timeWindow,
			}

			tailer, err := NewFileTailer(config, mockEmitter)
			if err != nil {
				t.Fatalf("Failed to create tailer: %v", err)
			}

			ctx := context.Background()
			err = tailer.Tail(ctx)
			if err == nil {
				t.Fatal("Expected buffer overflow error, got nil")
			}

			if !errors.Is(err, tt.expectedErr) {
				t.Errorf("Expected error %v, got: %v", tt.expectedErr.Error(), err.Error())
			}
		})
	}
}

func timeMustParse(timeStr string) *time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		panic(err)
	}
	return &t
}
