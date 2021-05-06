// +build !linux

package main

import (
	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/metadata/watcher"
)

func newMetadataWatcher() watcher.MetadataWatcher {
	log.Info("Launching Web-based Watcher")
	return watcher.NewWebBasedWatcher()
}
