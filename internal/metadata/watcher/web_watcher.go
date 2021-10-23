// SPDX-License-Identifier: Apache-2.0

package watcher

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/digitalocean/droplet-agent/internal/log"

	"github.com/digitalocean/droplet-agent/internal/metadata/actioner"
	"golang.org/x/time/rate"
)

type webBasedWatcher struct {
	metadataFetcher

	server              *http.Server
	limiter             *rate.Limiter
	registeredActioners []actioner.MetadataActioner
}

// NewWebBasedWatcher creates a new metadata watcher that is based on a webserver
func NewWebBasedWatcher(_ *Conf) MetadataWatcher {
	ret := &webBasedWatcher{
		metadataFetcher: newMetadataFetcher(),
		limiter:         rate.NewLimiter(rate.Every(time.Second/maxFetchPerSecond), 1),
	}
	return ret
}

// RegisterActioner registers a new actioner
// Note: this function is not thread-safe
func (w *webBasedWatcher) RegisterActioner(actioner actioner.MetadataActioner) {
	w.registeredActioners = append(w.registeredActioners, actioner)
}

// Run launches the watcher
func (w *webBasedWatcher) Run() error {
	log.Info("[Web Based Watcher] Running")
	if len(w.registeredActioners) == 0 {
		return ErrNoRegisteredActioner
	}

	r := http.NewServeMux()
	r.HandleFunc("/new_metadata", func(rw http.ResponseWriter, r *http.Request) {
		if !w.limiter.Allow() {
			rw.WriteHeader(http.StatusTooManyRequests)
			return
		}
		log.Debug("Metadata changes notified")
		rw.WriteHeader(http.StatusAccepted)
		m, e := w.fetchMetadata()
		if e != nil {
			log.Error("failed to fetch rmetadata: %v", e)
			return
		}
		for _, actioner := range w.registeredActioners {
			go actioner.Do(m)
		}
	})

	w.server = &http.Server{
		Addr:    webAddr,
		Handler: r,
	}
	if err := w.server.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			log.Info("http server closed")
			return nil
		}
		log.Error("failed to run web based watcher:%v", err)
		return err
	}
	return nil
}

// Shutdown shutdowns the watcher and all of the registered actioners
func (w *webBasedWatcher) Shutdown() {
	log.Info("[Web Based Watcher] Shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), maxShutdownWaitTimeSeconds*time.Second)
	defer cancel()
	_ = w.server.Shutdown(ctx)

	log.Debug("[Web Based Watcher] HTTP server shutdown")
	log.Debug("[Web Based Watcher] Shutting down all actioners")

	var wg sync.WaitGroup
	for _, a := range w.registeredActioners {
		wg.Add(1)

		go func(ma actioner.MetadataActioner) {
			ma.Shutdown()
			wg.Done()
		}(a)
	}
	wg.Wait()
	log.Info("[Web Based Watcher] Bye-bye")
}
