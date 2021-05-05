// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/dotty-agent/internal/log"

	"github.com/digitalocean/dotty-agent/internal/sysutil"
)

const (
	defaultAuthorizedKeysFile = "%h/.ssh/authorized_keys"
	dottyComment              = "# Added and Managed by DigitalOcean TTY service (DOTTY)"
	dottyKeyIndicator         = "dotty_ssh"
	defaultOSUser             = "root"
)

// SSHManager provides functions for managing SSH access
type SSHManager struct {
	sshHelper
	authorizedKeysFileUpdater

	authorizedKeysFilePattern string // same as the AuthorizedKeysFile in sshd_config, default to %h/.ssh/authorized_keys

	sysMgr sysManager

	cachedKeys       map[string][]*SSHKey
	cachedKeysOpLock sync.Mutex
}

// NewSSHManager constructs a new SSHManager object
func NewSSHManager() (*SSHManager, error) {
	ret := &SSHManager{
		sysMgr:     sysutil.NewSysManager(),
		cachedKeys: make(map[string][]*SSHKey),
	}
	ret.sshHelper = &sshHelperImpl{
		mgr:     ret,
		timeNow: time.Now,
	}
	ret.authorizedKeysFileUpdater = &updaterImpl{sshMgr: ret}

	err := ret.parseSSHDConfig()
	if err != nil {
		return nil, err
	}

	return ret, nil
}

// RemoveExpiredKeys removes expired keys from the authorized_keys file
func (s *SSHManager) RemoveExpiredKeys() (err error) {
	log.Debug("removing expired keys")
	s.cachedKeysOpLock.Lock()
	defer s.cachedKeysOpLock.Unlock()

	if len(s.cachedKeys) == 0 {
		log.Debug("empty cached keys, skip removing")
		return nil
	}
	cleanKeys := s.removeExpiredKeys(s.cachedKeys)
	hasExpired := false
	defer func() {
		if hasExpired && err == nil {
			log.Debug("expired keys removed")
			s.cachedKeys = cleanKeys
		} else {
			log.Debug("has expired keys: %v, update file error: %v", hasExpired, err)
		}
	}()
	for user, keys := range s.cachedKeys {
		if s.areSameKeys(keys, cleanKeys[user]) {
			// keys all still valid for this user, no need to update
			continue
		}
		hasExpired = true
		log.Debug("removing expired keys for %s", user)
		if e := s.updateAuthorizedKeysFile(user, cleanKeys[user]); e != nil {
			return e
		}
	}
	return nil
}

// UpdateKeys updates the given ssh keys to corresponding authorized_keys files.
func (s *SSHManager) UpdateKeys(keys []*SSHKey) (retErr error) {
	s.cachedKeysOpLock.Lock() // this lock may be too aggressive and can be possibly refined
	defer s.cachedKeysOpLock.Unlock()

	keyGroups := make(map[string][]*SSHKey) // group the keys by os user
	for _, key := range keys {
		if err := s.validateKey(key); err != nil {
			return err
		}
		if _, ok := keyGroups[key.OSUser]; !ok {
			keyGroups[key.OSUser] = make([]*SSHKey, 0, 1)
		}
		keyGroups[key.OSUser] = append(keyGroups[key.OSUser], key)
	}
	defer func() {
		if retErr == nil {
			s.cachedKeys = keyGroups
		}
	}()

	for username, keys := range keyGroups {
		if s.areSameKeys(keys, s.cachedKeys[username]) {
			//key not changed for the current user, skip
			log.Debug("keys not changed for %s, skipped", username)
			continue
		}
		log.Debug("updating %d keys for %s", len(keys), username)
		if err := s.updateAuthorizedKeysFile(username, keys); err != nil {
			return err
		}
	}

	for user := range s.cachedKeys {
		// update the authorized_keys file for users that no longer have valid keys
		if _, ok := keyGroups[user]; !ok {
			// if keys of a user is deleted
			log.Debug("removing keys for %s", user)
			if err := s.updateAuthorizedKeysFile(user, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SSHManager) parseSSHDConfig() error {
	s.authorizedKeysFilePattern = defaultAuthorizedKeysFile
	sshdConfigBytes, err := s.sysMgr.ReadFile(s.sshdConfigFile())
	if err != nil {
		return fmt.Errorf("%w:%s", ErrSSHDConfigParseFailed, err.Error())
	}
	sshdConfigs := strings.Split(string(sshdConfigBytes), "\n")
sshdParsing:
	for _, line := range sshdConfigs {
		line = strings.TrimLeft(line, "\t ")
		if !strings.HasPrefix(line, "AuthorizedKeysFile ") {
			continue
		}
		keyFiles := strings.Split(line, " ")
		if len(keyFiles) < 2 {
			return fmt.Errorf("%w: invalid format of AuthorizedKeysFile", ErrSSHDConfigParseFailed)
		}
		for i := 1; i != len(keyFiles); i++ {
			keyFile := strings.Trim(keyFiles[i], "\t")
			if keyFile == "" {
				continue
			}
			if keyFile[0] != '/' {
				keyFile = "%h/" + keyFile
			}
			s.authorizedKeysFilePattern = keyFile
			break sshdParsing
		}
	}
	return nil
}
