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
	// DropletID is the unique identifier for the Droplet
	DropletID int `json:"droplet_id,omitempty"`
	// Hostname is the hostname of the Droplet
	Hostname string `json:"hostname,omitempty"`
	// Region is the region where the Droplet is located
	Region string `json:"region,omitempty"`
	// PublicKeys contains all SSH Keys configured for this droplet
	PublicKeys []string `json:"public_keys,omitempty"`
	// DOTTYKeys contains temporary ssh keys used in cases such as web console access
	DOTTYKeys []string `json:"dotty_keys,omitempty"`
	// DOTTYStatus represents the state of the dotty agent valid states are "installed", "running", or "stopped"
	DOTTYStatus        AgentStatus `json:"dotty_status,omitempty"`
	SSHInfo            *SSHInfo    `json:"ssh_info,omitempty"`
	ManagedKeysEnabled *bool       `json:"managed_keys_enabled,omitempty"`
	// TroubleshootingAgent contains the configuration for the troubleshooting agent on a Droplet
	TroubleshootingAgent *TroubleshootingAgent `json:"troubleshooting_agent,omitempty"`
}

// SSHInfo contains the information of the sshd service running on the droplet
type SSHInfo struct {
	// Port is the port that the sshd is listening to
	Port uint16 `json:"port,omitempty"`
	// HostKeys is the public ssh keys of the droplet, needed for identifying the droplet
	HostKeys []string `json:"host_keys,omitempty"`
}

// TroubleshootingAgent represents the configuration for the troubleshooting agent on a Droplet.
type TroubleshootingAgent struct {
	// InvestigationUUID is the UUID of the investigation currently being
	// performed by the agent.
	InvestigationUUID string `json:"investigation_uuid,omitempty"`
	// TriggeredAt is an ISO 8601 timestamp indicating when the alert associated
	// with the investigation was triggered.
	TriggeredAt string `json:"triggered_at,omitempty"`
	// Requesting is a list of artifacts that the agent is being requested to
	// collect, e.g. "file:/var/log/syslog"
	Requesting []string `json:"requesting,omitempty"`
}
