// SPDX-License-Identifier: Apache-2.0

package actioner

import (
	"sync/atomic"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/metadata"
	"github.com/digitalocean/droplet-agent/internal/sysaccess"
)

// NewDOManagedKeysActioner returns a new DigitalOcean Managed keys actioner
func NewDOManagedKeysActioner(sshMgr *sysaccess.SSHManager) MetadataActioner {
	return &doManagedKeysActioner{
		sshMgr:    sshMgr,
		keyParser: metadata.NewSSHKeyParser(),
		allDone:   make(chan struct{}, 1),
	}
}

type sshManager interface {
	UpdateKeys(keys []*sysaccess.SSHKey) (retErr error)
	RemoveDoTTYKeys() error
}

type sshKeyParser interface {
	FromPublicKey(key string) (*sysaccess.SSHKey, error)
	FromDOTTYKey(key string) (*sysaccess.SSHKey, error)
}

type doManagedKeysActioner struct {
	sshMgr        sshManager
	keyParser     sshKeyParser
	activeActions int32
	closing       uint32
	allDone       chan struct{}
}

func (da *doManagedKeysActioner) do(metadata *metadata.Metadata) {
	log.Info("[DO-Managed Keys Actioner] Attempting to update %d ssh keys and %d dotty keys", len(metadata.PublicKeys), len(metadata.DOTTYKeys))
	sshKeys := make([]*sysaccess.SSHKey, 0, len(metadata.PublicKeys)+len(metadata.DOTTYKeys))
	// prepare ssh keys
	for _, kRaw := range metadata.PublicKeys {
		k, e := da.keyParser.FromPublicKey(kRaw)
		if e != nil {
			log.Error("[DO-Managed Keys Actioner] invalid public key object. %v", e)
			continue
		}
		sshKeys = append(sshKeys, k)
	}
	// prepare dotty keys
	for _, kRaw := range metadata.DOTTYKeys {
		k, e := da.keyParser.FromDOTTYKey(kRaw)
		if e != nil {
			log.Error("[DO-Managed Keys Actioner] invalid ssh key object. %v", e)
			continue
		}
		sshKeys = append(sshKeys, k)
	}
	log.Info("[DO-Managed Keys Actioner] Updating %d keys", len(sshKeys))
	if err := da.sshMgr.UpdateKeys(sshKeys); err != nil {
		log.Error("[DO-Managed Keys Actioner] failed to update keys: %v", err)
		return
	}
	log.Info("[DO-Managed Keys Actioner] Keys updated")
}

func (da *doManagedKeysActioner) Do(metadata *metadata.Metadata) {
	atomic.AddInt32(&da.activeActions, 1)
	defer func() {
		ret := atomic.AddInt32(&da.activeActions, -1)
		if ret == 0 && atomic.LoadUint32(&da.closing) == 1 {
			close(da.allDone)
		}
	}()
	da.do(metadata)
}
func (da *doManagedKeysActioner) Shutdown() {
	log.Info("[DO-Managed Keys Actioner] Shutting down")
	atomic.StoreUint32(&da.closing, 1)
	if atomic.LoadInt32(&da.activeActions) != 0 {
		// if there are still jobs in progress, wait for them to finish
		log.Debug("[DO-Managed Keys Actioner] Waiting for jobs in progress")
		<-da.allDone
	}
	log.Debug("[DO-Managed Keys Actioner] Clearing dotty keys from filesystem")
	// clear all agent managed keys
	// this is for resolving the bug that:
	// 1. agent started, and installed keys for user A and B
	// 2. agent restarted, but all keys for user B are already removed from metadata
	//    therefore, it will not appear in the response of metadata query,
	//    and agent will leave some garbage keys in user B's authorized_keys file.
	_ = da.sshMgr.RemoveDoTTYKeys()
	log.Info("[DO-Managed Keys Actioner] Bye-bye")
}
