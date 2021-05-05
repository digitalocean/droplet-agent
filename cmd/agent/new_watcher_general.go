// SPDX-License-Identifier: Apache-2.0

// +build !linux

package main

import (
	"github.com/digitalocean/dotty-agent/internal/log"
	"github.com/digitalocean/dotty-agent/internal/metadata/watcher"
)

func newMetadataWatcher() watcher.MetadataWatcher {
	log.Info("Launching Web-based Watcher")
	return watcher.NewWebBasedWatcher()
}
