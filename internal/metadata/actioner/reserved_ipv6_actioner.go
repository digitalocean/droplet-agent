// SPDX-License-Identifier: Apache-2.0

package actioner

import (
	"sync/atomic"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/metadata"
	"github.com/digitalocean/droplet-agent/internal/reservedipv6"
)

const (
	logPrefix string = "[Reserved IPv6 Actioner]"
)

// NewReservedIPv6Actioner returns a new DigitalOcean Reserved IPv6 actioner
func NewReservedIPv6Actioner(mgr reservedipv6.Manager) MetadataActioner {
	return &reservedIPv6Actioner{
		mgr:           mgr,
		activeActions: &atomic.Uint32{},
		closing:       &atomic.Bool{},
		allDone:       make(chan struct{}, 1),
	}
}

type reservedIPv6Actioner struct {
	mgr           reservedipv6.Manager
	activeActions *atomic.Uint32
	closing       *atomic.Bool
	allDone       chan struct{}
}

func (da *reservedIPv6Actioner) Do(md *metadata.Metadata) {
	da.activeActions.Add(1)
	defer func() {
		// decrement active counter, then check shutdown state
		ret := da.activeActions.Add(^uint32(0))
		if ret == 0 && da.closing.Load() {
			close(da.allDone)
		}
	}()

	ipv6 := md.ReservedIP.IPv6

	if ipv6.Active {
		logDebug("Attempting to assign Reserved IPv6 address '%s'", ipv6.IPAddress)
		if err := da.mgr.Assign(ipv6.IPAddress); err != nil {
			logError("failed to assign Reserved IPv6 address '%s': %v", ipv6.IPAddress, err)
			return
		}
		logInfo("Assigned Reserved IPv6 address '%s'", ipv6.IPAddress)
	} else {
		logDebug("Attempting to unassign all Reserved IPv6 addresses")
		if err := da.mgr.Unassign(); err != nil {
			logError("failed to unassign all Reserved IPv6 addresses: %v", err)
			return
		}
		logInfo("Unassigned all Reserved IPv6 addresses")
	}
}

func (da *reservedIPv6Actioner) Shutdown() {
	logInfo("Shutting down")
	da.closing.Store(true)

	// if there are still jobs in progress, wait for them to finish
	if da.activeActions.Load() > 0 {
		logDebug("Waiting for jobs in progress")
		<-da.allDone
	}
	logInfo("Bye-bye")
}

// logInfo wraps log.Info with rip6LogPrefix
func logInfo(format string, params ...any) {
	msg := logPrefix + " " + format
	log.Info(msg, params)
}

// logDebug wraps log.Debug with rip6LogPrefix
func logDebug(format string, params ...any) {
	msg := logPrefix + " " + format
	log.Debug(msg, params)
}

// logError wraps log.Error with rip6LogPrefix
func logError(format string, params ...any) {
	msg := logPrefix + " " + format
	log.Error(msg, params)
}
