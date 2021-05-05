// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"sync"
	"time"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/metadata/actioner"
	"github.com/digitalocean/droplet-agent/internal/netutil"
	"golang.org/x/time/rate"
)

const (
	sshPort  = 22
	doSeqNum = 68796879 // DODO -- 'D' = 68, 'O' = 79 in ASCII
	doAckNum = 848489   // TTY  -- 'T' = 84, 'Y' = 89
)

var tcpPacketPattern = &netutil.TCPPacketIdentifier{
	TargetPort: sshPort,
	SeqNum:     doSeqNum,
	AckNum:     doAckNum,
	TCPFlag:    netutil.TCPFlagSYN,
}

// NewSSHWatcher creates a new metadata watcher that is based on watching port knocking messages on port 22
func NewSSHWatcher() MetadataWatcher {
	ret := &sshWatcher{
		fetcher:             newMetadataFetcher(),
		sniffer:             netutil.NewTCPPacketSniffer(),
		limiter:             rate.NewLimiter(rate.Every(time.Second/maxFetchPerSecond), 1),
		registeredActioners: nil,
		done:                make(chan struct{}),
	}
	return ret
}

type sshWatcher struct {
	fetcher             metadataFetcher
	sniffer             netutil.TCPPacketSniffer
	limiter             *rate.Limiter
	registeredActioners []actioner.MetadataActioner

	done chan struct{}
}

// RegisterActioner registers a new actioner
// Note: this function is not thread-safe
func (w *sshWatcher) RegisterActioner(actioner actioner.MetadataActioner) {
	w.registeredActioners = append(w.registeredActioners, actioner)
}

// Run launches the watcher
func (w *sshWatcher) Run() error {
	log.Info("[SSH Watcher] Running")
	packetChan, err := w.sniffer.Capture(tcpPacketPattern)
	if err != nil {
		return err
	}
	defer w.sniffer.Stop()
	for {
		select {
		case packet := <-packetChan:
			log.Info("[SSH Watcher] Port knocking detected.")
			log.Debug("Packet Info: %+v", packet)
			if !w.limiter.Allow() {
				log.Error("[SSH Watcher] too many requests")
				continue
			}
			log.Debug("[SSH Watcher] Fetching metadata")
			md, e := w.fetcher.fetchMetadata()
			if e != nil {
				// TODO: maybe add a retry here?
				log.Error("failed to fetch rmetadata: %v", e)
				continue
			}
			log.Debug("[SSH Watcher] Metadata fetched. Calling Actioners")
			for _, actioner := range w.registeredActioners {
				go actioner.Do(md)
			}

		case <-w.done:
			log.Info("[SSH Watcher] Stopped")
			return nil
		}
	}
}

// Shutdown shutdowns the watcher and all of the registered actioners
func (w *sshWatcher) Shutdown() {
	log.Info("[SSH Watcher] Shutting down")
	close(w.done)

	log.Debug("[SSH Watcher] Shutting down all actioners")

	var wg sync.WaitGroup
	for _, a := range w.registeredActioners {
		wg.Add(1)

		go func(ma actioner.MetadataActioner) {
			ma.Shutdown()
			wg.Done()
		}(a)
	}
	wg.Wait()
	log.Info("[SSH Watcher] Bye-bye")
}
