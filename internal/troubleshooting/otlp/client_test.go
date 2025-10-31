package otlp

import (
	"context"
	"testing"
	"time"

	"github.com/digitalocean/droplet-agent/internal/troubleshooting/parser"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/embedded"
)

func TestClient_EmitLog(t *testing.T) {
	tests := []struct {
		name                    string
		sourceFile              string
		logEntry                parser.LogEntry
		expectedBody            string
		expectedTimestampParsed string
		verifyTimestamp         bool
		expectedTimestamp       time.Time
	}{
		{
			name:       "log with parsed timestamp",
			sourceFile: "file:/var/log/syslog",
			logEntry: parser.LogEntry{
				Original:        "2023-10-01T10:00:00Z Application started",
				Timestamp:       timeMustParse("2023-10-01T10:00:00Z"),
				TimestampParsed: true,
			},
			expectedBody:            "2023-10-01T10:00:00Z Application started",
			expectedTimestampParsed: "parsed",
			verifyTimestamp:         true,
			expectedTimestamp:       timeMustParse("2023-10-01T10:00:00Z"),
		},
		{
			name:       "log with observed timestamp",
			sourceFile: "file:/var/log/app.log",
			logEntry: parser.LogEntry{
				Original:        "No timestamp - Application log entry",
				Timestamp:       timeMustParse("2023-10-01T12:00:00Z"),
				TimestampParsed: false,
			},
			expectedBody:            "No timestamp - Application log entry",
			expectedTimestampParsed: "observed",
			verifyTimestamp:         true,
			expectedTimestamp:       timeMustParse("2023-10-01T12:00:00Z"),
		},
		{
			name:       "command source with observed timestamp",
			sourceFile: "command:top",
			logEntry: parser.LogEntry{
				Original:        "Tasks: 123 total",
				Timestamp:       timeMustParse("2023-10-01T14:00:00Z"),
				TimestampParsed: false,
			},
			expectedBody:            "Tasks: 123 total",
			expectedTimestampParsed: "observed",
			verifyTimestamp:         true,
			expectedTimestamp:       timeMustParse("2023-10-01T14:00:00Z"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := &mockLogger{}
			client := &Client{
				logger:            mockLog,
				investigationUUID: "test-uuid",
			}

			ctx := context.Background()
			client.EmitLog(ctx, tt.sourceFile, tt.logEntry)

			// Verify one record was emitted
			if len(mockLog.emittedRecords) != 1 {
				t.Fatalf("Expected 1 emitted record, got %d", len(mockLog.emittedRecords))
			}

			record := mockLog.emittedRecords[0]

			// Verify body
			if record.body != tt.expectedBody {
				t.Errorf("Body mismatch. Got %q, want %q", record.body, tt.expectedBody)
			}

			// Verify timestamp
			if tt.verifyTimestamp {
				if !record.timestamp.Equal(tt.expectedTimestamp) {
					t.Errorf("Timestamp mismatch. Got %v, want %v", record.timestamp, tt.expectedTimestamp)
				}
			}

			// Verify log.source attribute
			sourceAttr, exists := record.attributes[logSourceAttribute]
			if !exists {
				t.Errorf("Missing %q attribute", logSourceAttribute)
			} else if sourceAttr != tt.sourceFile {
				t.Errorf("%q attribute mismatch. Got %q, want %q", logSourceAttribute, sourceAttr, tt.sourceFile)
			}

			// Verify log.timestamp.parsed attribute
			parsedAttr, exists := record.attributes[timeStampParsedAttribute]
			if !exists {
				t.Errorf("Missing %q attribute", timeStampParsedAttribute)
			} else if parsedAttr != tt.expectedTimestampParsed {
				t.Errorf("%q attribute mismatch. Got %q, want %q", timeStampParsedAttribute, parsedAttr, tt.expectedTimestampParsed)
			}

			// Verify investigation.uuid attribute
			invAttr, exists := record.attributes[investigationUUIDAttribute]
			if !exists {
				t.Errorf("Missing %q attribute", investigationUUIDAttribute)
			} else if invAttr != "test-uuid" {
				t.Errorf("%q attribute mismatch. Got %q, want %q", investigationUUIDAttribute, invAttr, "test-uuid")
			}

			// Verify context
			if record.ctx != ctx {
				t.Error("Context mismatch")
			}
		})
	}
}

func TestClient_EmitLog_MultipleRecords(t *testing.T) {
	mockLog := &mockLogger{}
	client := &Client{
		logger: mockLog,
	}

	ctx := context.Background()

	// Emit multiple log entries
	entries := []parser.LogEntry{
		{
			Original:        "2023-10-01T10:00:00Z First entry",
			Timestamp:       timeMustParse("2023-10-01T10:00:00Z"),
			TimestampParsed: true,
		},
		{
			Original:        "2023-10-01T11:00:00Z Second entry",
			Timestamp:       timeMustParse("2023-10-01T11:00:00Z"),
			TimestampParsed: true,
		},
		{
			Original:        "No timestamp - Third entry",
			Timestamp:       timeMustParse("2023-10-01T12:00:00Z"),
			TimestampParsed: false,
		},
	}

	for _, entry := range entries {
		client.EmitLog(ctx, "file:/var/log/test.log", entry)
	}

	// Verify all records were emitted
	if len(mockLog.emittedRecords) != len(entries) {
		t.Fatalf("Expected %d emitted records, got %d", len(entries), len(mockLog.emittedRecords))
	}

	// Verify each record
	for i, entry := range entries {
		record := mockLog.emittedRecords[i]
		if record.body != entry.Original {
			t.Errorf("Record %d body mismatch. Got %q, want %q", i, record.body, entry.Original)
		}
		if !record.timestamp.Equal(entry.Timestamp) {
			t.Errorf("Record %d timestamp mismatch. Got %v, want %v", i, record.timestamp, entry.Timestamp)
		}

		expectedParsed := "observed"
		if entry.TimestampParsed {
			expectedParsed = "parsed"
		}
		parsedAttr := record.attributes[timeStampParsedAttribute]
		if parsedAttr != expectedParsed {
			t.Errorf("Record %d parsed attribute mismatch. Got %q, want %q", i, parsedAttr, expectedParsed)
		}
	}
}

func TestClient_EmitError(t *testing.T) {
	tests := []struct {
		name              string
		sourceComponent   string
		errorMsg          string
		investigationUUID string
	}{
		{
			name:              "file tailer error",
			sourceComponent:   "file_tailer",
			errorMsg:          "failed to open /var/log/syslog: permission denied",
			investigationUUID: "test-uuid-001",
		},
		{
			name:              "command runner error",
			sourceComponent:   "command_runner",
			errorMsg:          "command 'top' exited with status 1",
			investigationUUID: "test-uuid-002",
		},
		{
			name:              "network error",
			sourceComponent:   "http_client",
			errorMsg:          "failed to POST to endpoint: connection timeout",
			investigationUUID: "test-uuid-003",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLog := &mockLogger{}
			client := &Client{
				logger:            mockLog,
				investigationUUID: tt.investigationUUID,
			}

			ctx := context.Background()
			client.EmitError(ctx, tt.sourceComponent, tt.errorMsg)

			// Verify one record was emitted
			if len(mockLog.emittedRecords) != 1 {
				t.Fatalf("Expected 1 emitted record, got %d", len(mockLog.emittedRecords))
			}

			record := mockLog.emittedRecords[0]

			// Verify body contains the error message
			if record.body != tt.errorMsg {
				t.Errorf("Body mismatch. Got %q, want %q", record.body, tt.errorMsg)
			}

			// Verify log.source attribute includes error prefix
			expectedSource := "error:" + tt.sourceComponent
			sourceAttr, exists := record.attributes[logSourceAttribute]
			if !exists {
				t.Errorf("Missing %q attribute", logSourceAttribute)
			} else if sourceAttr != expectedSource {
				t.Errorf("%q attribute mismatch. Got %q, want %q", logSourceAttribute, sourceAttr, expectedSource)
			}

			// Verify log.timestamp.parsed attribute is "observed"
			parsedAttr, exists := record.attributes[timeStampParsedAttribute]
			if !exists {
				t.Errorf("Missing %q attribute", timeStampParsedAttribute)
			} else if parsedAttr != "observed" {
				t.Errorf("%q attribute mismatch. Got %q, want %q", timeStampParsedAttribute, parsedAttr, "observed")
			}

			// Verify investigation.uuid attribute
			invAttr, exists := record.attributes[investigationUUIDAttribute]
			if !exists {
				t.Errorf("Missing %q attribute", investigationUUIDAttribute)
			} else if invAttr != tt.investigationUUID {
				t.Errorf("%q attribute mismatch. Got %q, want %q", investigationUUIDAttribute, invAttr, tt.investigationUUID)
			}
		})
	}
}

func timeMustParse(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		panic(err)
	}
	return t
}

// mockLogger implements log.Logger for testing
type mockLogger struct {
	embedded.Logger
	emittedRecords []capturedRecord
}

// capturedRecord holds the emitted record data for verification
type capturedRecord struct {
	ctx        context.Context
	body       string
	timestamp  time.Time
	attributes map[string]interface{}
}

func (m *mockLogger) Emit(ctx context.Context, record log.Record) {
	captured := capturedRecord{
		ctx:        ctx,
		body:       record.Body().AsString(),
		timestamp:  record.Timestamp(),
		attributes: make(map[string]interface{}),
	}

	record.WalkAttributes(func(kv log.KeyValue) bool {
		// Since we're only dealing with string attributes in our tests,
		// we can safely use AsString()
		captured.attributes[string(kv.Key)] = kv.Value.AsString()
		return true
	})

	m.emittedRecords = append(m.emittedRecords, captured)
}

func (m *mockLogger) Enabled(ctx context.Context, param log.EnabledParameters) bool {
	return true
}
