package otlp

import (
	"context"
	"fmt"
	"time"

	"github.com/digitalocean/droplet-agent/internal/troubleshooting/parser"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlplog/otlploghttp"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/global"
	sdklog "go.opentelemetry.io/otel/sdk/log"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const (
	timeStampParsedAttribute   = "log.timestamp.parsed"
	logSourceAttribute         = "log.source"
	dropletIDAttribute         = "droplet.id"
	dropletRegionAttribute     = "droplet.region"
	investigationUUIDAttribute = "investigation.uuid"
	InvestigationUUIDHeader    = "X-Investigation-UUID"
)

// ClientConfig holds configuration for creating an OpenTelemetry client
type ClientConfig struct {
	Endpoint       string // OTLP HTTP endpoint
	DropletID      int    // ID of the Droplet
	Hostname       string // Hostname of the Droplet
	Region         string // Region of the Droplet
	ServiceName    string // Service name for OpenTelemetry resource
	ServiceVersion string // Service version for OpenTelemetry resource
	Investigation  string // UUID of the investigation
}

// Emitter defines the interface for emitting log entries
type Emitter interface {
	// EmitLog sends a log record with proper timestamp handling
	// sourceFile specifies the source of the log (e.g., "file:/var/log/syslog" or "command:top")
	EmitLog(ctx context.Context, sourceFile string, entry parser.LogEntry)
	// EmitError sends an error log record
	EmitError(ctx context.Context, sourceComponent string, msg string)
	// Flush exports log records that have not yet been exported.
	Flush(ctx context.Context) error
}

// Client wraps the OpenTelemetry logging functionality
type Client struct {
	logger            log.Logger
	provider          *sdklog.LoggerProvider
	investigationUUID string
}

// Ensure Client implements Emitter interface
var _ Emitter = (*Client)(nil)

// NewClient creates a new OpenTelemetry client configured for log export
func NewClient(ctx context.Context, config ClientConfig) (*Client, error) {
	// Create resource with service information
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.HostName(config.Hostname),
			attribute.KeyValue{
				Key:   dropletIDAttribute,
				Value: attribute.IntValue(config.DropletID)},
			attribute.KeyValue{
				Key:   dropletRegionAttribute,
				Value: attribute.StringValue(config.Region)},
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create OTLP HTTP exporter
	exporterOptions := []otlploghttp.Option{
		otlploghttp.WithEndpointURL(config.Endpoint),
		otlploghttp.WithCompression(otlploghttp.GzipCompression),
		otlploghttp.WithHeaders(map[string]string{
			InvestigationUUIDHeader: config.Investigation,
		}),
	}

	exporter, err := otlploghttp.New(ctx, exporterOptions...)
	if err != nil {
		return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
	}

	// Create log provider with batch processor
	provider := sdklog.NewLoggerProvider(
		sdklog.WithResource(res),
		sdklog.WithProcessor(sdklog.NewBatchProcessor(
			exporter,
			sdklog.WithExportMaxBatchSize(1000),             // (default: 512)
			sdklog.WithMaxQueueSize(8192),                   // (default: 2048)
			sdklog.WithExportInterval(500*time.Millisecond), // (default: 1s)
			sdklog.WithExportTimeout(10*time.Second),        // (default: 30s)
		)),
	)

	// Set global logger provider
	global.SetLoggerProvider(provider)

	// Create logger
	logger := provider.Logger(config.ServiceName)

	return &Client{
		logger:            logger,
		provider:          provider,
		investigationUUID: config.Investigation,
	}, nil
}

// EmitLog sends a log record via OpenTelemetry with proper timestamp handling
// sourceFile specifies the source of the log (e.g., "file:/var/log/syslog" or "command:top")
func (c *Client) EmitLog(ctx context.Context, sourceFile string, entry parser.LogEntry) {
	record := log.Record{}
	logAttributes := []log.KeyValue{
		log.String(logSourceAttribute, sourceFile),
		log.String(investigationUUIDAttribute, c.investigationUUID),
	}

	record.SetBody(log.StringValue(entry.Original))

	// Set timestamp: uses parsed timestamp if available, otherwise the observed
	// time is used. We add an attribute to indicate which was used.
	record.SetTimestamp(entry.Timestamp)
	if entry.TimestampParsed {
		logAttributes = append(logAttributes, log.String(timeStampParsedAttribute, "parsed"))
	} else {
		logAttributes = append(logAttributes, log.String(timeStampParsedAttribute, "observed"))
	}

	record.AddAttributes(logAttributes...)

	c.logger.Emit(ctx, record)
}

func (c *Client) EmitError(ctx context.Context, sourceComponent string, msg string) {
	record := log.Record{}
	logAttributes := []log.KeyValue{
		log.String(logSourceAttribute, fmt.Sprintf("error:%s", sourceComponent)),
		log.String(investigationUUIDAttribute, c.investigationUUID),
		log.String(timeStampParsedAttribute, "observed"),
	}

	record.SetBody(log.StringValue(msg))
	record.SetTimestamp(time.Now())
	record.AddAttributes(logAttributes...)

	c.logger.Emit(ctx, record)
}

// Flush exports log records that have not yet been exported.
func (c *Client) Flush(ctx context.Context) error {
	return c.provider.ForceFlush(ctx)
}
