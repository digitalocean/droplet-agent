// SPDX-License-Identifier: Apache-2.0

package actioner

import "github.com/digitalocean/droplet-agent/internal/metadata"

// MetadataActioner performs action on a metadata update
type MetadataActioner interface {
	Do(metadata *metadata.Metadata)
	Shutdown()
}
