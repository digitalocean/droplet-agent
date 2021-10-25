package updater

import "github.com/digitalocean/droplet-agent/internal/metadata"

// AgentInfoUpdater updates the droplet agent related fields in the droplet's metadata
type AgentInfoUpdater interface {
	Update(md *metadata.Metadata) error
}
