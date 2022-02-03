// SPDX-License-Identifier: Apache-2.0

package actioner

import (
	"encoding/json"
	"sync/atomic"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/metadata"
	"github.com/digitalocean/droplet-agent/internal/sysaccess"
)

// NewDOTTYKeysActioner returns a new DOTTY keys actioner
func NewDOTTYKeysActioner(sshMgr *sysaccess.SSHManager) MetadataActioner {
	return &dottyKeysActioner{
		sshMgr:  sshMgr,
		allDone: make(chan struct{}, 1),
	}
}

type sshManager interface {
	UpdateKeys(keys []*sysaccess.SSHKey) (retErr error)
	RemoveDoTTYKeys() error
}

type dottyKeysActioner struct {
	sshMgr        sshManager
	activeActions int32
	closing       uint32
	allDone       chan struct{}
}

func (da *dottyKeysActioner) do(metadata *metadata.Metadata) {
	log.Info("[DOTTY-Keys Actioner] Attempting to update %d keys", len(metadata.DOTTYKeys))
	sshKeys := make([]*sysaccess.SSHKey, 0, len(metadata.DOTTYKeys))
	for _, kRaw := range metadata.DOTTYKeys {
		key := &sysaccess.SSHKey{}
		if err := json.Unmarshal([]byte(kRaw), key); err != nil {
			log.Error("[DOTTY-Keys Actioner] invalid ssh key object. %v", err)
			continue
		}
		sshKeys = append(sshKeys, key)
	}
	if err := da.sshMgr.UpdateKeys(sshKeys); err != nil {
		log.Error("[DOTTY-Keys Actioner] failed to update keys: %v", err)
		return
	}
	log.Info("[DOTTY-Keys Actioner] Keys updated")
}

func (da *dottyKeysActioner) Do(metadata *metadata.Metadata) {
	atomic.AddInt32(&da.activeActions, 1)
	defer func() {
		ret := atomic.AddInt32(&da.activeActions, -1)
		if ret == 0 && atomic.LoadUint32(&da.closing) == 1 {
			close(da.allDone)
		}
	}()
	da.do(metadata)
}
func (da *dottyKeysActioner) Shutdown() {
	log.Info("[DOTTY Keys Actioner] Shutting down")
	atomic.StoreUint32(&da.closing, 1)
	if atomic.LoadInt32(&da.activeActions) != 0 {
		// if there are still jobs in progress, wait for them to finish
		log.Debug("[DOTTY Keys Actioner] Waiting for jobs in progress")
		<-da.allDone
	}
	log.Debug("[DOTTY Keys Actioner] Clearing dotty keys from filesystem")
	// clear all agent managed keys
	// this is for resolving the bug that:
	// 1. agent started, and installed keys for user A and B
	// 2. agent restarted, but all keys for user B are already removed from metadata
	//    therefore, it will not appear in the response of metadata query,
	//    and agent will leave some garbage keys in user B's authorized_keys file.
	_ = da.sshMgr.RemoveDoTTYKeys()
	log.Info("[DOTTY Keys Actioner] Bye-bye")
}
