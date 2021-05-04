package watcher

import "github.com/digitalocean/dotty-agent/internal/metadata/actioner"

// MetadataWatcher watches for metadata changes of the given droplet,
// It notifies every registered actioner when it detects any metadata changes.
type MetadataWatcher interface {
	RegisterActioner(actioner actioner.MetadataActioner)
	Run() error
	Shutdown()
}
