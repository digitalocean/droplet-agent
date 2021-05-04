package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/digitalocean/dotty-agent/internal/config"
	"github.com/digitalocean/dotty-agent/internal/log"
	"github.com/digitalocean/dotty-agent/internal/metadata"
	"github.com/digitalocean/dotty-agent/internal/metadata/actioner"
	"github.com/digitalocean/dotty-agent/internal/metadata/status"
	"github.com/digitalocean/dotty-agent/internal/metadata/watcher"
	"github.com/digitalocean/dotty-agent/internal/sysaccess"
)

func main() {
	cfg := config.Init()

	log.Info("Config Loaded. Agent Starting (version:%s)", cfg.Version)

	if cfg.DebugMode {
		log.EnableDebug()
		log.Info("Debug mode enabled")
	}

	sshMgr, err := sysaccess.NewSSHManager()
	if err != nil {
		log.Fatal("failed to initialize SSHManager: %v", err)
	}

	dottyKeysActioner := actioner.NewDOTTYKeysActioner(sshMgr)
	metadataWatcher := newMetadataWatcher()
	metadataWatcher.RegisterActioner(dottyKeysActioner)
	updater := status.NewStatusUpdater()

	// Launch background jobs
	bgJobsCtx, bgJobsCancel := context.WithCancel(context.Background())
	go bgJobsRemoveExpiredDOTTYKeys(bgJobsCtx, sshMgr, cfg.AuthorizedKeysCheckInterval)

	// handle shutdown
	go handleShutdown(bgJobsCancel, metadataWatcher, updater)

	// set status to running
	go setStatus(updater, metadata.RunningStatus, true)

	// launch the watcher
	if err := metadataWatcher.Run(); err != nil {
		log.Fatal("Failed to run watcher... %v", err)
	} else {
		log.Info("Watcher finished")
	}
}

func handleShutdown(bgJobsCancel context.CancelFunc, metadataWatcher watcher.MetadataWatcher, updater status.Updater) {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGTSTP,
		syscall.SIGQUIT,
	)

	c := <-signalChan
	setStatus(updater, metadata.StoppedStatus, false)
	switch c {
	case syscall.SIGINT, syscall.SIGTERM:
		log.Info("[DOTTY Agent] Shutting down")
		bgJobsCancel()
		metadataWatcher.Shutdown()
	case syscall.SIGTSTP, syscall.SIGQUIT:
		log.Info("[DOTTY Agent] Forced to quit! You may lose jobs in progress")
	default:
		log.Error("unsupported signal, %d", c)
		os.Exit(1)
	}
}

func setStatus(updater status.Updater, agentStatus metadata.AgentStatus, retry bool) {
	fn := func() error { return updater.Update(agentStatus) }
	sleepTime := time.Second * 5

	if !retry {
		err := fn()
		if err != nil {
			log.Error("error setting status: %s", err)
		}
		return
	}

	for {
		log.Debug("setting status")
		err := fn()
		if err == nil {
			return
		}

		time.Sleep(sleepTime)
		log.Error("error setting status: %s, retrying", err)
	}
}
