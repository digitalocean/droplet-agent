// SPDX-License-Identifier: Apache-2.0

package actioner

import (
	"sync/atomic"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/metadata"
	"github.com/digitalocean/droplet-agent/internal/reservedipv6"
)

const (
	rip6LogPrefix string = "[Reserved IPv6 Actioner]"
)

// NewReservedIPv6Actioner returns a new DigitalOcean Reserved IPv6 actioner
func NewReservedIPv6Actioner(mgr reservedipv6.Manager) MetadataActioner {
	return &reservedIPv6Actioner{
		mgr:     mgr,
		allDone: make(chan struct{}, 1),
	}
}

type reservedIPv6Actioner struct {
	mgr           reservedipv6.Manager
	activeActions int32
	closing       uint32
	allDone       chan struct{}
}

func (da *reservedIPv6Actioner) Do(md *metadata.Metadata) {
	atomic.AddInt32(&da.activeActions, 1)
	defer func() {
		ret := atomic.AddInt32(&da.activeActions, -1)
		if ret == 0 && atomic.LoadUint32(&da.closing) == 1 {
			close(da.allDone)
		}
	}()

	ipv6 := md.ReservedIP.IPv6

	if ipv6.Active {
		log.Info("%s Attempting to assign Reserved IPv6 address '%s'", rip6LogPrefix, ipv6.IPAddress)
		if err := da.mgr.Assign(ipv6.IPAddress); err != nil {
			log.Error("%s failed to assign Reserved IPv6 address '%s': %v", rip6LogPrefix, ipv6.IPAddress, err)
		}
		log.Info("%s Assigned Reserved IPv6 address '%s'", rip6LogPrefix, ipv6.IPAddress)
	} else {
		log.Info("%s Attempting to unassign all Reserved IPv6 addresses", rip6LogPrefix)
		if err := da.mgr.Unassign(); err != nil {
			log.Error("%s failed to unassign all Reserved IPv6 addresses: %v", rip6LogPrefix, err)
		}
		log.Info("%s Unassigned all Reserved IPv6 addresses", rip6LogPrefix)
	}
}

func (da *reservedIPv6Actioner) Shutdown() {
	log.Info("%s Shutting down", rip6LogPrefix)
	atomic.StoreUint32(&da.closing, 1)
	if atomic.LoadInt32(&da.activeActions) != 0 {
		// if there are still jobs in progress, wait for them to finish
		log.Debug("%s Waiting for jobs in progress", rip6LogPrefix)
		<-da.allDone
	}
	log.Info("%s Bye-bye", rip6LogPrefix)
}
