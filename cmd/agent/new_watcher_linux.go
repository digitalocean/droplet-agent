// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/digitalocean/dotty-agent/internal/log"
	"github.com/digitalocean/dotty-agent/internal/metadata/watcher"
)

func newMetadataWatcher() watcher.MetadataWatcher {
	log.Info("Launching SSH Port Knocking Watcher")
	return watcher.NewSSHWatcher()
}
