// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"time"

	"github.com/digitalocean/dotty-agent/internal/log"

	"github.com/digitalocean/dotty-agent/internal/sysaccess"
)

func bgJobsRemoveExpiredDOTTYKeys(ctx context.Context, sshMgr *sysaccess.SSHManager, interval time.Duration) {
	log.Info("[authorized_keys files updater] launched")
	ticker := time.NewTicker(interval)
loop:
	for {
		select {
		case <-ctx.Done():
			log.Debug("[authorized_keys files updater] agent closing")
			break loop
		case <-ticker.C:
			log.Debug("[authorized_keys files updater] attempting to remove expired keys")
			if err := sshMgr.RemoveExpiredKeys(); err != nil {
				log.Error("[authorized_keys files updater] failed to remove expired keys:%v", err)
			}

		}
	}
	ticker.Stop()
	log.Info("[authorized_keys files updater] stopped")
}
