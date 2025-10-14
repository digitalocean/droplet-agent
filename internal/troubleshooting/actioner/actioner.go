// SPDX-License-Identifier: Apache-2.0

package actioner

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/droplet-agent/internal/config"
	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/metadata"
	metadataactioner "github.com/digitalocean/droplet-agent/internal/metadata/actioner"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/command"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/file"
	"github.com/digitalocean/droplet-agent/internal/troubleshooting/otlp"
)

const (
	// OTLPEndpoint is the default endpoint for submitting troubleshooting logs
	OTLPEndpoint = "http://169.254.169.254/v1/sre-agents/droplets/logs"
	// CompletedEndpoint is the endpoint to mark an investigation as completed
	CompletedEndpoint = "http://169.254.169.254/v1/sre-agents/droplets/logs/completed"

	// InvestigationArtifactSyslog is used to request /var/log/syslog file collection
	InvestigationArtifactSyslog = "file:/var/log/syslog"
	// InvestigationArtifactMessages is used to request /var/log/messages file collection
	InvestigationArtifactMessages = "file:/var/log/messages"
	// InvestigationArtifactTop is used to request the output of the top command
	InvestigationArtifactTop = "command:top"
	// InvestigationArtifactPs is used to request the output of the ps command
	InvestigationArtifactPs = "command:ps"
	// InvestigationArtifactJournalctl is used to request the output of the journalctl command
	InvestigationArtifactJournalctl = "command:journalctl"

	// defaultLastLines is the number of lines to emit from a log file if no
	// alert timestamp is provided or we are unable to parse the timestamp of the file.
	defaultLastLines = 100
)

var (
	validInvestigationArtifacts = map[string]struct{}{
		InvestigationArtifactSyslog:     {},
		InvestigationArtifactMessages:   {},
		InvestigationArtifactTop:        {},
		InvestigationArtifactPs:         {},
		InvestigationArtifactJournalctl: {},
	}
)

// TroubleshootingExporter implements the MetadataActioner interface to export logs
// based on metadata updates
type TroubleshootingExporter struct {
	// AgentConfig holds the main configuration for the droplet-agent
	AgentConfig AgentConfig
	// Endpoint is the OTLP endpoint to send logs to
	Endpoint string
	// CompletedEndpoint is the endpoint to mark investigation as completed
	CompletedEndpoint string
	// newOTLPClient creates an OTLP client.
	newOTLPClient func(ctx context.Context, config otlp.ClientConfig) (otlp.Emitter, error)
	// newCommandRunner creates a command runner (for testing).
	newCommandRunner func(config command.Config, emitter otlp.Emitter) (command.Command, error)
	// newFileTailer creates a file tailer (for testing).
	newFileTailer func(config file.Config, emitter otlp.Emitter) (file.File, error)
	// httpClient is used to make HTTP requests
	httpClient httpClient
	// mu protects the runningInvestigations field
	mu sync.Mutex
	// runningInvestigations holds the set of currently running investigation UUIDs
	runningInvestigations map[string]struct{}
	// investigationWg is used to wait for running investigations to complete during shutdown
	investigationWg sync.WaitGroup
	// shutdownCtx is a context that is canceled when Shutdown is called
	shutdownCtx context.Context
	// shutdownCancel is the cancel function for shutdownCtx
	shutdownCancel context.CancelFunc
}

// httpClient interface for making HTTP requests
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// AgentConfig holds basic configuration for the droplet-agent
type AgentConfig struct {
	Version      string
	UserAgent    string
	OTLPEndpoint string
}

// Ensure TroubleshootingExporter implements MetadataActioner interface
var _ metadataactioner.MetadataActioner = (*TroubleshootingExporter)(nil)

// NewTroubleshootingExporter creates a new TroubleshootingExporter instance
// with default (real) factories
func NewTroubleshootingExporter(agentConfig AgentConfig) *TroubleshootingExporter {
	otlpEndpoint := agentConfig.OTLPEndpoint
	if otlpEndpoint == "" {
		otlpEndpoint = OTLPEndpoint
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TroubleshootingExporter{
		AgentConfig:       agentConfig,
		Endpoint:          otlpEndpoint,
		CompletedEndpoint: CompletedEndpoint,
		newOTLPClient: func(ctx context.Context, config otlp.ClientConfig) (otlp.Emitter, error) {
			return otlp.NewClient(ctx, config)
		},
		newCommandRunner:      command.NewRunner,
		newFileTailer:         file.NewFileTailer,
		runningInvestigations: make(map[string]struct{}),
		httpClient:            &http.Client{Timeout: 10 * time.Second},
		shutdownCtx:           ctx,
		shutdownCancel:        cancel,
	}
}

// Do processes metadata updates and performs log export actions
func (te *TroubleshootingExporter) Do(md *metadata.Metadata) {
	// Return early if there's no active investigation
	if md.TroubleshootingAgent == nil || md.TroubleshootingAgent.InvestigationUUID == "" {
		return
	}

	// Try-lock: skip if this specific investigation is already running
	investigationUUID := md.TroubleshootingAgent.InvestigationUUID
	if !te.tryAcquire(investigationUUID) {
		log.Info("[Troubleshooting Actioner] Investigation %s already in progress, skipping duplicate request", investigationUUID)
		return
	}

	te.investigationWg.Add(1)
	defer te.investigationWg.Done()
	defer te.release(investigationUUID)

	log.Info("[Troubleshooting Actioner] Metadata contains investigation (%s) request", md.TroubleshootingAgent.InvestigationUUID)

	// Validate requested artifacts against allowed list
	var artifacts []string
	for _, artifact := range md.TroubleshootingAgent.Requesting {
		if _, valid := validInvestigationArtifacts[artifact]; valid {
			artifacts = append(artifacts, artifact)
		}
	}

	ctx, cancel := context.WithCancel(te.shutdownCtx)
	defer cancel()

	client, err := te.newOTLPClient(ctx, otlp.ClientConfig{
		Endpoint:       te.Endpoint,
		DropletID:      md.DropletID,
		Hostname:       md.Hostname,
		Region:         md.Region,
		ServiceName:    te.AgentConfig.UserAgent,
		ServiceVersion: te.AgentConfig.Version,
		Investigation:  md.TroubleshootingAgent.InvestigationUUID,
	})
	if err != nil {
		log.Error("[Troubleshooting Actioner] Failed to create OTLP client: %v", err)
		// Note: Can't emit error since client creation failed
		return
	}

	// Calculate time window: 15 minutes before and after the trigger time
	var timeWindow *file.TimeWindow
	triggered := md.TroubleshootingAgent.TriggeredAt
	if triggered != "" {
		parsedTime, err := time.Parse(time.RFC3339, triggered)
		if err == nil {
			timeWindow = &file.TimeWindow{
				Start: parsedTime.Add(-15 * time.Minute),
				End:   parsedTime.Add(15 * time.Minute),
			}
			log.Info("[Troubleshooting Actioner] Using time window: %s to %s (±15 min from trigger)",
				timeWindow.Start.Format(time.RFC3339), timeWindow.End.Format(time.RFC3339))
		}
	}

	for _, artifact := range artifacts {
		log.Info("[Troubleshooting Actioner] Collecting artifact: %s", artifact)
		switch {
		case strings.HasPrefix(artifact, command.CmdPrefix):
			cmdConfig := command.Config{
				Source:     artifact,
				TimeWindow: timeWindow,
			}

			cmd, err := te.newCommandRunner(cmdConfig, client)
			if err != nil {
				log.Error("[Troubleshooting Actioner] Failed to create command runner for artifact '%s': %v", artifact, err)
				client.EmitError(ctx, "command_runner", fmt.Sprintf("Failed to create command runner for artifact '%s': %v", artifact, err))
				continue
			}

			if err := cmd.Run(ctx); err != nil {
				log.Error("[Troubleshooting Actioner] Failed to collect command artifact '%s': %v", artifact, err)
				client.EmitError(ctx, "command_runner", fmt.Sprintf("Failed to collect command artifact '%s': %v", artifact, err))
			}
		case strings.HasPrefix(artifact, file.FilePrefix):
			config := file.Config{
				Source:     artifact,
				LastLines:  defaultLastLines,
				TimeWindow: timeWindow,
			}

			fileTailer, err := te.newFileTailer(config, client)
			if err != nil {
				log.Error("[Troubleshooting Actioner] Failed to create file tailer for '%s': %v", artifact, err)
				client.EmitError(ctx, "file_tailer", fmt.Sprintf("Failed to create file tailer for '%s': %v", artifact, err))
				continue
			}

			if err := fileTailer.Tail(ctx); err != nil {
				log.Error("[Troubleshooting Actioner] Failed to collect file artifact '%s': %v", artifact, err)
				client.EmitError(ctx, "file_tailer", fmt.Sprintf("Failed to collect file artifact '%s': %v", artifact, err))
			}
		}
	}

	err = client.Flush(ctx)
	if err != nil {
		log.Error("[Troubleshooting Actioner] Failed to flush OTLP client: %v", err)
		client.EmitError(ctx, "otlp_flush", fmt.Sprintf("Failed to flush OTLP client: %v", err))
	}

	err = te.markInvestigationReady(ctx, investigationUUID)
	if err != nil {
		log.Error("[Troubleshooting Actioner] Failed to mark investigation as completed: %v", err)
		client.EmitError(ctx, "investigation_completion", fmt.Sprintf("Failed to mark investigation as ready: %v", err))
	}
}

// Shutdown gracefully shuts down the TroubleshootingExporter
func (te *TroubleshootingExporter) Shutdown() {
	log.Info("[Troubleshooting Actioner] Shutting down...")
	te.shutdownCancel()

	// Wait for investigations to finish, with a timeout
	waitChan := make(chan struct{})
	go func() {
		te.investigationWg.Wait()
		close(waitChan)
	}()

	select {
	case <-waitChan:
		log.Info("[Troubleshooting Actioner] Shutdown successful.")
	case <-time.After(30 * time.Second):
		log.Info("[Troubleshooting Actioner] Shutdown timed out.")
	}
}

// tryAcquire attempts to acquire the lock for the given investigation UUID.
// Returns true if acquired, false if the investigation is already running.
func (te *TroubleshootingExporter) tryAcquire(investigationUUID string) bool {
	te.mu.Lock()
	defer te.mu.Unlock()

	// Don't start new investigations if shutdown has been initiated
	if te.shutdownCtx.Err() != nil {
		return false
	}

	if _, exists := te.runningInvestigations[investigationUUID]; exists {
		return false
	}
	te.runningInvestigations[investigationUUID] = struct{}{}
	return true
}

// release releases the lock for the given investigation.
func (te *TroubleshootingExporter) release(investigationUUID string) {
	te.mu.Lock()
	defer te.mu.Unlock()
	delete(te.runningInvestigations, investigationUUID)
}

// markInvestigationReady sends a POST request to notify that log collection is
// completed, and that the investigation is ready for analysis.
func (te *TroubleshootingExporter) markInvestigationReady(ctx context.Context, investigationUUID string) error {
	const (
		maxRetries     = 5
		initialBackoff = 1 * time.Second
		maxBackoff     = 10 * time.Second
	)

	var lastErr error
	backoff := initialBackoff

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			log.Info("[Troubleshooting Actioner] Retrying completed endpoint (attempt %d/%d) after %v", attempt+1, maxRetries, backoff)
			time.Sleep(backoff)
			// Exponential backoff with cap
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, te.CompletedEndpoint, nil)
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("User-Agent", config.UserAgent)
		req.Header.Set(otlp.InvestigationUUIDHeader, investigationUUID)

		resp, err := te.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("request failed: %w", err)
			continue
		}

		defer func() {
			if resp.Body != nil {
				_ = resp.Body.Close()
			}
		}()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			log.Info("[Troubleshooting Actioner] Successfully marked investigation %s as completed", investigationUUID)
			return nil
		}

		lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, lastErr)
}
