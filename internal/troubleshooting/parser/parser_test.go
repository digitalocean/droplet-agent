package parser

import (
	"testing"
	"time"
)

func TestLogParser_ParseLine(t *testing.T) {
	tests := []struct {
		name               string
		logLines           []string         // Multiple lines to test format detection caching
		expectedTimestamps []time.Time      // Expected timestamps (zero time if not parseable)
		expectedLayout     string           // Expected detected layout
		expectParsed       []bool           // Whether timestamp was successfully parsed for each line
		nowFunc            func() time.Time // Optional: override time source for this test
	}{
		{
			name: "RFC3339 format",
			logLines: []string{
				"2023-10-01T10:00:00Z First log entry",
				"2023-10-01T11:00:00Z Second log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2023, time.October, 1, 10, 0, 0, 0, time.UTC),
				time.Date(2023, time.October, 1, 11, 0, 0, 0, time.UTC),
			},
			expectedLayout: LayoutRFC3339,
			expectParsed:   []bool{true, true},
		},
		{
			name: "RFC3339 with timezone offset",
			logLines: []string{
				"2023-10-01T10:00:00+02:00 First log entry",
				"2023-10-01T11:00:00+02:00 Second log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2023, time.October, 1, 10, 0, 0, 0, time.FixedZone("+0200", 2*60*60)),
				time.Date(2023, time.October, 1, 11, 0, 0, 0, time.FixedZone("+0200", 2*60*60)),
			},
			expectedLayout: LayoutRFC3339,
			expectParsed:   []bool{true, true},
		},
		{
			name: "RFC3339Nano format",
			logLines: []string{
				"2023-10-01T10:00:00.123456789Z First log entry",
				"2023-10-01T11:00:00.987654321Z Second log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2023, time.October, 1, 10, 0, 0, 123456789, time.UTC),
				time.Date(2023, time.October, 1, 11, 0, 0, 987654321, time.UTC),
			},
			expectedLayout: LayoutRFC3339Nano,
			expectParsed:   []bool{true, true},
		},
		{
			name: "RFC3339Nano with timezone offset",
			logLines: []string{
				"2023-10-01T10:00:00.123456789+02:00 First log entry",
				"2023-10-01T11:00:00.987654321+02:00 Second log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2023, time.October, 1, 10, 0, 0, 123456789, time.FixedZone("+0200", 2*60*60)),
				time.Date(2023, time.October, 1, 11, 0, 0, 987654321, time.FixedZone("+0200", 2*60*60)),
			},
			expectedLayout: LayoutRFC3339Nano,
			expectParsed:   []bool{true, true},
		},
		{
			name: "Syslog format",
			logLines: []string{
				"Sep 10 15:04:05 kernel: First log entry",
				"Sep 10 15:05:10 kernel: Second log entry",
				"Sep 10 15:06:20 kernel: Third log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2025, time.September, 10, 15, 4, 5, 0, time.UTC),
				time.Date(2025, time.September, 10, 15, 5, 10, 0, time.UTC),
				time.Date(2025, time.September, 10, 15, 6, 20, 0, time.UTC),
			},
			expectedLayout: LayoutSyslog,
			expectParsed:   []bool{true, true, true},
			nowFunc:        func() time.Time { return time.Date(2025, time.September, 10, 20, 0, 0, 0, time.UTC) },
		},
		{
			name: "Syslog edge case: Dec log in Jan",
			logLines: []string{
				"Dec 31 23:59:59 kernel: End of year log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2024, time.December, 31, 23, 59, 59, 0, time.UTC),
			},
			expectedLayout: LayoutSyslog,
			expectParsed:   []bool{true},
			nowFunc:        func() time.Time { return time.Date(2025, time.January, 1, 0, 0, 0, 0, time.UTC) },
		},
		{
			name: "DateTime format",
			logLines: []string{
				"2023-09-10 15:04:05 INFO: First log entry",
				"2023-09-10 15:05:10 INFO: Second log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2023, time.September, 10, 15, 4, 5, 0, time.UTC),
				time.Date(2023, time.September, 10, 15, 5, 10, 0, time.UTC),
			},
			expectedLayout: LayoutISO8601DateTime,
			expectParsed:   []bool{true, true},
		},
		{
			name: "DateTime with milliseconds",
			logLines: []string{
				"2023-09-10 15:04:05.123 INFO: First log entry",
				"2023-09-10 15:05:10.456 INFO: Second log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2023, time.September, 10, 15, 4, 5, 123000000, time.UTC),
				time.Date(2023, time.September, 10, 15, 5, 10, 456000000, time.UTC),
			},
			expectedLayout: LayoutISO8601DateTimeMillis,
			expectParsed:   []bool{true, true},
		},
		{
			name: "Apache/Nginx format",
			logLines: []string{
				`[10/Sep/2023:15:04:05 +0000] "GET /index.html HTTP/1.1" 200`,
				`[10/Sep/2023:15:05:10 -0500] "POST /api HTTP/1.1" 201`,
			},
			expectedTimestamps: []time.Time{
				time.Date(2023, time.September, 10, 15, 4, 5, 0, time.UTC),
				time.Date(2023, time.September, 10, 15, 5, 10, 0, time.FixedZone("-0500", -5*60*60)),
			},
			expectedLayout: LayoutApache,
			expectParsed:   []bool{true, true},
		},
		{
			name: "US DateTime format",
			logLines: []string{
				"09/10/2023 15:04:05 INFO: First log entry",
				"09/10/2023 15:05:10 INFO: Second log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2023, time.September, 10, 15, 4, 5, 0, time.UTC),
				time.Date(2023, time.September, 10, 15, 5, 10, 0, time.UTC),
			},
			expectedLayout: LayoutUSDateTime,
			expectParsed:   []bool{true, true},
		},
		{
			name: "ISO with comma format - matches DateTime pattern first",
			logLines: []string{
				"2023-09-10 15:04:05,123 INFO: First log entry",
				"2023-09-10 15:05:10,456 INFO: Second log entry",
			},
			expectedTimestamps: []time.Time{
				time.Date(2023, time.September, 10, 15, 4, 5, 123000000, time.UTC),
				time.Date(2023, time.September, 10, 15, 5, 10, 456000000, time.UTC),
			},
			expectedLayout: LayoutISO8601Comma,
			expectParsed:   []bool{true, true},
		},
		{
			name: "No timestamp - falls back to observed time",
			logLines: []string{
				"No timestamp here - First log entry",
				"No timestamp here - Second log entry",
				"No timestamp here - Third log entry",
			},
			expectedTimestamps: []time.Time{time.Time{}, time.Time{}, time.Time{}},
			expectedLayout:     "none",
			expectParsed:       []bool{false, false, false},
		},
		{
			name: "Ignores lines without timestamps",
			logLines: []string{
				"# header without timestamp",
				"2023-10-01T10:00:00Z First with RFC3339",
				"2023-10-01T11:00:00Z Second with RFC3339",
			},
			expectedTimestamps: []time.Time{
				time.Time{},
				time.Date(2023, time.October, 1, 10, 0, 0, 0, time.UTC),
				time.Date(2023, time.October, 1, 11, 0, 0, 0, time.UTC),
			},
			expectedLayout: LayoutRFC3339,
			expectParsed:   []bool{false, true, true},
		},
	}
	// Use a fixed "now" for deterministic tests
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parser := NewLogParser()
			if tt.nowFunc != nil {
				parser.nowFunc = tt.nowFunc
			}

			for i, line := range tt.logLines {
				entry := parser.ParseLine(line)

				// Verify original line is preserved
				if entry.Original != line {
					t.Errorf("Line %d: Original mismatch. Got %q, want %q", i, entry.Original, line)
				}

				// Verify parsed status
				if entry.TimestampParsed != tt.expectParsed[i] {
					t.Errorf("Line %d: TimestampParsed mismatch. Got %v, want %v", i, entry.TimestampParsed, tt.expectParsed[i])
				}

				// Verify timestamp if it should be parsed
				if tt.expectParsed[i] && !tt.expectedTimestamps[i].IsZero() {
					expectedTime := tt.expectedTimestamps[i]
					if !entry.Timestamp.Equal(expectedTime) {
						t.Errorf("Line %d: Timestamp mismatch. Got %v, want %v", i, entry.Timestamp, expectedTime)
					}
				}

				// Verify timestamp is not zero even when not parsed (should use observed time)
				if !tt.expectParsed[i] && entry.Timestamp.IsZero() {
					t.Errorf("Line %d: Timestamp is zero but should have observed time", i)
				}
			}

			// Verify detected format
			detectedLayout, detected := parser.DetectedFormat()
			if detectedLayout != tt.expectedLayout {
				t.Errorf("Detected layout mismatch. Got %q, want %q", detectedLayout, tt.expectedLayout)
			}

			// After sample size lines are processed, format is always marked as detected
			// (even if no pattern was found, in which case the layout is "none")
			if !detected && len(tt.logLines) >= 3 {
				t.Errorf("Detected status should be true after %d lines (sample size)", len(tt.logLines))
			}
		})
	}
}

func TestLogParser_Reset(t *testing.T) {
	parser := NewLogParser()

	// Parse some lines to detect format
	line1 := "2023-10-01T10:00:00Z First log entry"
	entry1 := parser.ParseLine(line1)
	if !entry1.TimestampParsed {
		t.Error("Expected first line timestamp to be parsed")
	}

	// Verify format is detected
	detectedLayout, detected := parser.DetectedFormat()
	if !detected || detectedLayout != LayoutRFC3339 {
		t.Errorf("Expected RFC3339 format to be detected, got %q (detected: %v)", detectedLayout, detected)
	}

	// Reset the parser
	parser.Reset()

	// Verify format is no longer detected
	detectedLayout, detected = parser.DetectedFormat()
	if detected {
		t.Errorf("Expected format not to be detected after reset, got %q", detectedLayout)
	}
	if detectedLayout != "none" {
		t.Errorf("Expected layout 'none' after reset, got %q", detectedLayout)
	}

	// Parse a line with a different format
	line2 := "2023-09-10 15:04:05 INFO: New format"
	entry2 := parser.ParseLine(line2)
	if !entry2.TimestampParsed {
		t.Error("Expected timestamp to be parsed with new format")
	}

	// Verify new format is detected
	detectedLayout, detected = parser.DetectedFormat()
	if !detected || detectedLayout != LayoutISO8601DateTime {
		t.Errorf("Expected DateTime format to be detected, got %q (detected: %v)", detectedLayout, detected)
	}
}

func TestLogParser_FormatDetectionCaching(t *testing.T) {
	parser := NewLogParser()

	// First line with RFC3339 format
	line1 := "2023-10-01T10:00:00Z First log entry"
	entry1 := parser.ParseLine(line1)
	if !entry1.TimestampParsed {
		t.Error("Expected first line timestamp to be parsed")
	}

	// Second line with different format - should not be parsed because RFC3339 is cached
	line2 := "2023-09-10 15:04:05 Second with different format"
	entry2 := parser.ParseLine(line2)
	if entry2.TimestampParsed {
		t.Error("Expected second line timestamp NOT to be parsed (different format, but RFC3339 cached)")
	}

	// Third line with RFC3339 format - should be parsed using cached format
	line3 := "2023-10-01T11:00:00Z Third with original format"
	entry3 := parser.ParseLine(line3)
	if !entry3.TimestampParsed {
		t.Error("Expected third line timestamp to be parsed using cached format")
	}

	// Verify the cached format
	detectedLayout, detected := parser.DetectedFormat()
	if !detected || detectedLayout != LayoutRFC3339 {
		t.Errorf("Expected RFC3339 format to remain cached, got %q (detected: %v)", detectedLayout, detected)
	}
}
