package parser

import (
	"regexp"
	"time"
)

const (
	LayoutRFC3339               = time.RFC3339
	LayoutRFC3339Nano           = time.RFC3339Nano
	LayoutSyslog                = "Jan _2 15:04:05"
	LayoutISO8601DateTime       = "2006-01-02 15:04:05"
	LayoutISO8601DateTimeMillis = "2006-01-02 15:04:05.000"
	LayoutISO8601Comma          = "2006-01-02 15:04:05,000"
	LayoutApache                = "02/Jan/2006:15:04:05 -0700"
	LayoutUSDateTime            = "01/02/2006 15:04:05"
)

// LogEntry represents a parsed log line
type LogEntry struct {
	// Timestamp is the timestamp for the log line. If one is not successfully
	// parsed, the observed time is used instead.
	Timestamp time.Time
	// Original is the original raw log line text
	Original string
	// TimestampParsed indicates if the timestamp was successfully parsed.
	// true if timestamp was successfully parsed, false if using observed time
	TimestampParsed bool
}

// Parser defines the interface for parsing log lines and extracting timestamps
type Parser interface {
	// ParseLine parses a log line and extracts timestamp information
	ParseLine(line string) LogEntry
	// DetectedFormat returns information about the detected timestamp format
	DetectedFormat() (timestampPattern string, detected bool)
	// Reset clears the cached format detection (useful for new files)
	Reset()
}

type timestampPattern struct {
	regex  *regexp.Regexp
	layout string
}

// Package-level timestamp patterns (shared across all parser instances)
var timestampPatterns = []timestampPattern{
	// RFC3339 with nanoseconds: 2023-09-10T15:04:05.123456789Z or 2023-09-10T15:04:05.123456789+02:00
	// Must be checked BEFORE basic RFC3339 since it's more specific
	{
		regex:  regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{1,9}(?:Z|[+-]\d{2}:\d{2}))`),
		layout: LayoutRFC3339Nano,
	},
	// RFC3339 format: 2023-09-10T15:04:05Z or 2023-09-10T15:04:05+02:00
	// This pattern should NOT match fractional seconds (handled by nano pattern above)
	{
		regex:  regexp.MustCompile(`(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}(?:Z|[+-]\d{2}:\d{2}))`),
		layout: LayoutRFC3339,
	},
	// Syslog (RFC 3164) format: Sep 10 15:04:05
	{
		regex:  regexp.MustCompile(`([A-Za-z]{3} +\d{1,2} +\d{2}:\d{2}:\d{2})`),
		layout: LayoutSyslog,
	},
	// ISO format: 2023-09-10 15:04:05,123
	{
		regex:  regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2},\d{3})`),
		layout: LayoutISO8601Comma,
	},
	// Common format with milliseconds: 2023-09-10 15:04:05.123
	{
		regex:  regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})`),
		layout: LayoutISO8601DateTimeMillis,
	},
	// Common format: 2023-09-10 15:04:05
	{
		regex:  regexp.MustCompile(`(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})`),
		layout: LayoutISO8601DateTime,
	},
	// Apache/Nginx format: 10/Sep/2023:15:04:05 +0000 or [10/Sep/2023:15:04:05 +0000]
	{
		regex:  regexp.MustCompile(`\[?(\d{2}/[A-Za-z]{3}/\d{4}:\d{2}:\d{2}:\d{2} [+-]\d{4})\]?`),
		layout: LayoutApache,
	},
	// US format: 09/10/2023 15:04:05
	{
		regex:  regexp.MustCompile(`(\d{2}/\d{2}/\d{4} \d{2}:\d{2}:\d{2})`),
		layout: LayoutUSDateTime,
	},
}

// LogParser implements format detection with caching for better performance
type LogParser struct {
	// Cached format information
	detectedTimestampPattern *timestampPattern
	formatDetected           bool
	sampleSize               int
	processedLines           int
	nowFunc                  func() time.Time
}

// Ensure LogParser implements Parser interface
var _ Parser = (*LogParser)(nil)

// NewLogParser creates a new cached log parser
func NewLogParser() *LogParser {
	return &LogParser{
		sampleSize: 3, // Detect format using first 3 lines
		nowFunc:    time.Now,
	}
}

// ParseLine parses a log line using cached format detection
func (p *LogParser) ParseLine(line string) LogEntry {
	entry := LogEntry{
		Timestamp:       p.nowFunc(), // fallback to current time
		Original:        line,
		TimestampParsed: false, // will be set to true if we successfully parse timestamp
	}

	// If format not yet detected, try to detect it
	if !p.formatDetected {
		p.detectFormats(line)
		p.processedLines++

		// Mark format as detected after sample size OR if we found timestamp pattern
		if p.processedLines >= p.sampleSize || p.detectedTimestampPattern != nil {
			p.formatDetected = true
		}
	}

	// Use cached format for timestamp parsing (if we have one)
	if p.detectedTimestampPattern != nil {
		timestamp := p.extractTimestamp(line, p.detectedTimestampPattern)
		if !timestamp.IsZero() {
			// Special handling for Syslog format: set year if missing
			if p.detectedTimestampPattern.layout == LayoutSyslog && timestamp.Year() <= 1 {
				now := p.nowFunc()
				year := now.Year()
				// If log month is greater than current month, it's from previous year
				if timestamp.Month() > now.Month() {
					year = now.Year() - 1
				}
				timestamp = time.Date(
					year,
					timestamp.Month(),
					timestamp.Day(),
					timestamp.Hour(),
					timestamp.Minute(),
					timestamp.Second(),
					timestamp.Nanosecond(),
					timestamp.Location(),
				)
			}

			entry.Timestamp = timestamp
			entry.TimestampParsed = true
		}
	}

	return entry
}

// detectFormats analyzes sample lines to detect timestamp format
func (p *LogParser) detectFormats(line string) {
	// Try to detect timestamp format
	if p.detectedTimestampPattern == nil {
		for i, pattern := range timestampPatterns {
			if pattern.regex.MatchString(line) {
				// Test if we can actually parse it
				if timestamp := p.extractTimestamp(line, &pattern); !timestamp.IsZero() {
					p.detectedTimestampPattern = &timestampPatterns[i]
					break
				}
			}
		}
	}
}

// extractTimestamp uses a specific pattern for faster parsing
func (p *LogParser) extractTimestamp(line string, pattern *timestampPattern) time.Time {
	matches := pattern.regex.FindStringSubmatch(line)
	if len(matches) > 1 {
		timestampStr := matches[1]
		layout := pattern.layout

		parsed, err := time.Parse(layout, timestampStr)
		if err == nil {
			return parsed
		}
	}
	return time.Time{}
}

// DetectedFormat returns information about the detected timestamp format
func (p *LogParser) DetectedFormat() (timestampPattern string, detected bool) {
	timestampPattern = "none"

	if p.detectedTimestampPattern != nil {
		timestampPattern = p.detectedTimestampPattern.layout
	}

	return timestampPattern, p.formatDetected
}

// Reset clears the cached format detection (useful for new files)
func (p *LogParser) Reset() {
	p.detectedTimestampPattern = nil
	p.formatDetected = false
	p.processedLines = 0
}
