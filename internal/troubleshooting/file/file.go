package file

// It supports reading the entire file or just the last N lines using a ring buffer.

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/otlp"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/parser"
)

const (
	// FilePrefix is the prefix for file sources.
	FilePrefix = "file:"
)

type File interface {
	Tail(context.Context) error
}

// Ensure fileImpl implements File interface
var _ File = (*fileImpl)(nil)

type fileImpl struct {
	client otlp.Emitter
	config Config
}

// Config holds configuration options for the Tailer
type Config struct {
	// Source is the source of the log file to be tailed
	Source string
	// LastLines specifies how many lines from the end of the file to read.
	// If 0 or negative, constraint is ignored.
	// When used with TimeWindow, serves as a fallback if timestamp parsing fails.
	LastLines int
	// TimeWindow defines the time range for filtering logs.
	// Only lines with timestamps within this window are included.
	// If timestamp parsing fails, falls back to LastLines behavior.
	// If nil, timestamp filtering is disabled.
	TimeWindow *TimeWindow
	// filePath is the actual path to the file, derived from Source
	filePath string
}

// TimeWindow represents a time range for log filtering
type TimeWindow struct {
	Start time.Time // Include logs from this time onward
	End   time.Time // Include logs up to this time
}

// NewFileTailer creates a new file tailer with the specified configuration
func NewFileTailer(conf Config, client otlp.Emitter) (File, error) {
	filePath := strings.TrimPrefix(conf.Source, FilePrefix)
	if filePath == "" {
		return nil, fmt.Errorf("file path cannot be empty")
	}
	conf.filePath = filePath

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file '%s' does not exist", filePath)
	}

	return &fileImpl{
		client: client,
		config: conf,
	}, nil
}

// Tail begins tailing the file.
func (f *fileImpl) Tail(ctx context.Context) error {
	file, err := os.Open(f.config.filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", f.config.filePath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Error("[Troubleshooting Actioner] failed to close file: %v", err)
		}
	}()

	scanner := f.setupScanner(file)

	// Route to appropriate processing method based on configuration
	if f.config.TimeWindow == nil && f.config.LastLines <= 0 {
		// No filtering - stream everything
		return f.streamAllLines(ctx, scanner)
	}

	if f.config.TimeWindow == nil && f.config.LastLines > 0 {
		// Only last N lines needed
		return f.streamLastLines(ctx, scanner)
	}

	// Time-based filtering with potential fallback
	return f.streamWithTimeFilter(ctx, scanner)
}

// streamAllLines reads and emits lines directly without storing in memory
func (f *fileImpl) streamAllLines(ctx context.Context, scanner *bufio.Scanner) error {
	logParser := parser.NewLogParser()

	for scanner.Scan() {
		if err := f.checkContext(ctx); err != nil {
			return err
		}
		line := scanner.Text()
		entry := logParser.ParseLine(line)
		f.client.EmitLog(ctx, f.config.Source, entry)
	}
	return f.checkScannerError(scanner)
}

// streamLastLines reads the last N lines and emits them
func (f *fileImpl) streamLastLines(ctx context.Context, scanner *bufio.Scanner) error {
	ringBuf := newRingBuffer(f.config.LastLines)
	logParser := parser.NewLogParser()

	// Collect last N lines using ring buffer
	for scanner.Scan() {
		if err := f.checkContext(ctx); err != nil {
			return err
		}
		ringBuf.add(scanner.Text())
	}

	// Emit the collected lines with parsed timestamps
	for _, line := range ringBuf.getLines() {
		if err := f.checkContext(ctx); err != nil {
			return err
		}
		entry := logParser.ParseLine(line)
		f.client.EmitLog(ctx, f.config.Source, entry)
	}

	return f.checkScannerError(scanner)
}

// streamWithTimeFilter handles time-based filtering with a fallback to last n
// lines if a timestamp is unable to be parsed.
func (f *fileImpl) streamWithTimeFilter(ctx context.Context, scanner *bufio.Scanner) error {
	var (
		fallbackBuffer    *ringBuffer
		timestampDetected = false
		emittedAny        = false
	)
	logParser := parser.NewLogParser()

	// If LastLines is configured, prepare a ring buffer for the fallback.
	if f.config.LastLines > 0 {
		fallbackBuffer = newRingBuffer(f.config.LastLines)
	}

	for scanner.Scan() {
		if err := f.checkContext(ctx); err != nil {
			return err
		}
		line := scanner.Text()

		entry := logParser.ParseLine(line)
		if entry.TimestampParsed {
			timestampDetected = true
			// Check if timestamp is within the time window
			if f.isInTimeWindow(entry.Timestamp) {
				// First time we've emitted a line, discard the fallback buffer.
				if !emittedAny {
					fallbackBuffer = nil
					emittedAny = true
				}
				f.client.EmitLog(ctx, f.config.Source, entry)
			}
		} else if fallbackBuffer != nil {
			// Only buffer lines when we haven't emitted anything yet.
			fallbackBuffer.add(line)
		}
	}

	// Fallback to LastLines if:
	// - No timestamps were successfully parsed in the entire file, OR
	// - Timestamps were found but none matched the Since filter
	if (!timestampDetected || !emittedAny) && fallbackBuffer != nil {
		for _, line := range fallbackBuffer.getLines() {
			if err := f.checkContext(ctx); err != nil {
				return err
			}
			entry := logParser.ParseLine(line)
			f.client.EmitLog(ctx, f.config.Source, entry)
		}
	}

	return f.checkScannerError(scanner)
}

// setupScanner creates and configures a scanner for the file
func (f *fileImpl) setupScanner(file *os.File) *bufio.Scanner {
	log.Debug("[Troubleshooting Actioner] Creating scanner for file: %s", f.config.filePath)

	scanner := bufio.NewScanner(file)

	// 16KB initial handles typical cases without growth
	// 1MB max still handles extreme cases (JSON logs, stack traces)
	buf := make([]byte, 0, 16*1024)
	scanner.Buffer(buf, 1024*1024)

	return scanner
}

// isInTimeWindow checks if a timestamp falls within the configured time window
func (f *fileImpl) isInTimeWindow(t time.Time) bool {
	if f.config.TimeWindow == nil {
		return true // No time filtering
	}
	return !t.Before(f.config.TimeWindow.Start) && !t.After(f.config.TimeWindow.End)
}

func (f *fileImpl) checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}

func (f *fileImpl) checkScannerError(scanner *bufio.Scanner) error {
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %s: %w", f.config.filePath, err)
	}
	return nil
}
