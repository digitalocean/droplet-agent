// SPDX-License-Identifier: Apache-2.0

package metadata

const (
	// BaseURL address of the droplet's metadata service
	BaseURL = "http://169.254.169.254/metadata"
)

// AgentStatus is a string type used to identify the current status of the agent
type AgentStatus string

const (
	// InstalledStatus indicates that the agent has been installed but is not yet running
	InstalledStatus AgentStatus = "installed"
	// RunningStatus indicates that the agent is running on the droplet and should be functioning properly
	RunningStatus AgentStatus = "running"
	// StoppedStatus indicates the agent is stopping or has stopped. The agent will be stopped for shutdown and restarts
	// as well as agent updates
	StoppedStatus AgentStatus = "stopped"
)

// Metadata is part of the object returned by the metadata/v1.json.
type Metadata struct {
	// DOTTYKeys contains ssh keys managed through DigitalOcean
	DOTTYKeys []string `json:"dotty_keys,omitempty"`
	// DOTTYStatus represents the state of the dotty agent valid states are "installed", "running", or "stopped"
	DOTTYStatus AgentStatus `json:"dotty_status,omitempty"`
}
