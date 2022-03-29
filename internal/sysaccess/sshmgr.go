// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/digitalocean/droplet-agent/internal/config"
	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/sysutil"

	"golang.org/x/sync/errgroup"
)

const (
	defaultAuthorizedKeysFile = "%h/.ssh/authorized_keys"
	dottyPrevComment          = "# Added and Managed by DigitalOcean TTY service (DOTTY)" // for backward compatibility
	dottyComment              = "# Added and Managed by " + config.AppFullName
	dropletKeyComment         = "# Managed through DigitalOcean"
	dropletKeyIndicator       = "do_managed_key"
	dottyKeyIndicator         = "dotty_ssh"
	defaultOSUser             = "root"
	defaultSSHDPort           = 22
	fileCheckInterval         = 5 * time.Second
)

// SSHManager provides functions for managing SSH access
type SSHManager struct {
	sshHelper
	authorizedKeysFileUpdater

	authorizedKeysFilePattern string // same as the AuthorizedKeysFile in sshd_config, default to %h/.ssh/authorized_keys
	sshdPort                  int

	sysMgr            sysManager
	fsWatcher         fsWatcher
	fsWatcherQuitHook func()

	cachedKeys       map[string][]*SSHKey
	cachedKeysOpLock sync.Mutex
}

// NewSSHManager constructs a new SSHManager object
func NewSSHManager(opts ...SSHManagerOpt) (*SSHManager, error) {
	defaultOpts := &sshMgrOpts{
		customSSHDPort:    0,
		customSSHDCfgFile: "",
	}
	for _, opt := range opts {
		opt(defaultOpts)
	}
	ret := &SSHManager{
		sysMgr:     sysutil.NewSysManager(),
		cachedKeys: make(map[string][]*SSHKey),
		sshdPort:   defaultOpts.customSSHDPort,
	}
	ret.sshHelper = &sshHelperImpl{
		mgr:               ret,
		timeNow:           time.Now,
		customSSHDCfgFile: defaultOpts.customSSHDCfgFile,
	}
	ret.authorizedKeysFileUpdater = &updaterImpl{sshMgr: ret}

	err := ret.parseSSHDConfig()
	if err != nil {
		return nil, err
	}
	if !validPort(ret.sshdPort) {
		return nil, fmt.Errorf("%w:[%d]", ErrInvalidPortNumber, ret.sshdPort)
	}
	log.Info("SSH Manager Initialized. sshd_config:[%s], sshd_port:[%d]", ret.sshdConfigFile(), ret.sshdPort)
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
	eg, _ := errgroup.WithContext(context.Background())
	for user, keys := range s.cachedKeys {
		u := user
		if s.areSameKeys(keys, cleanKeys[u]) {
			// keys all still valid for this user, no need to update
			continue
		}
		hasExpired = true
		eg.Go(func() error {
			log.Debug("removing expired keys for %s", u)
			if e := s.updateAuthorizedKeysFile(u, cleanKeys[u]); e != nil {
				log.Error("failed to remove expired keys for %s: %v", u, e)
				return e
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

// UpdateKeys updates the given ssh keys to corresponding authorized_keys files.
func (s *SSHManager) UpdateKeys(keys []*SSHKey) (retErr error) {
	s.cachedKeysOpLock.Lock() // this lock may be too aggressive and can be possibly refined
	defer s.cachedKeysOpLock.Unlock()
	if keys == nil {
		return ErrInvalidArgs
	}
	keyGroups := make(map[string][]*SSHKey) // group the keys by os user
	updatedKeys := make(map[string][]*SSHKey)
	for _, key := range keys {
		if err := s.validateKey(key); err != nil {
			//invalid key, skip
			log.Error("invalid key, %s", err.Error())
			continue
		}
		if _, ok := keyGroups[key.OSUser]; !ok {
			keyGroups[key.OSUser] = make([]*SSHKey, 0, 1)
		}
		keyGroups[key.OSUser] = append(keyGroups[key.OSUser], key)
	}
	defer func() {
		if retErr == nil {
			s.cachedKeys = updatedKeys
		}
	}()

	cleanKeys := s.removeExpiredKeys(s.cachedKeys)
	for username, keys := range keyGroups {
		if s.areSameKeys(keys, cleanKeys[username]) {
			//key not changed for the current user, skip
			log.Debug("keys not changed for %s, skipped", username)
			updatedKeys[username] = cleanKeys[username]
			continue
		}
		log.Debug("updating %d keys for %s", len(keys), username)
		if err := s.updateAuthorizedKeysFile(username, keys); err != nil {
			log.Error("failed to update keys for %s:%v", username, err)
			continue
		}
		updatedKeys[username] = keys
	}

	for user := range s.cachedKeys {
		// update the authorized_keys file for users that no longer have valid keys
		if _, ok := keyGroups[user]; !ok {
			// if keys of a user is deleted
			log.Debug("removing keys for %s", user)
			if err := s.updateAuthorizedKeysFile(user, []*SSHKey{}); err != nil {
				if errors.Is(err, sysutil.ErrUserNotFound) {
					log.Info("os user [%s] no longer exists", user)
					continue
				}
				log.Error("failed to remove keys for user %s:%v", user, err)
				// if failed to remove ssh keys for a user,
				// preserve them so that the removal can be retried next time
				updatedKeys[user] = s.cachedKeys[user]
			}
		}
	}
	return nil
}

// RemoveDOTTYKeys removes all dotty keys from the droplet
// When the agent exit, all temporary keys managed through DigitalOcean must be cleaned up
// to avoid leaving stale expired keys in the system
func (s *SSHManager) RemoveDOTTYKeys() error {
	s.cachedKeysOpLock.Lock()
	defer s.cachedKeysOpLock.Unlock()
	eg, _ := errgroup.WithContext(context.Background())
	for user := range s.cachedKeys {
		u := user
		eg.Go(func() error {
			if err := s.updateAuthorizedKeysFile(u, nil); err != nil {
				if errors.Is(err, sysutil.ErrUserNotFound) {
					log.Info("os user [%s] no longer exists", u)
					return nil
				}
				return fmt.Errorf("%w: failed to remove keys for user %s", err, user)
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	return nil
}

// SSHDPort returns the port sshd is binding to
func (s *SSHManager) SSHDPort() int {
	return s.sshdPort
}

// WatchSSHDConfig watches if sshd_config is modified,
// if yes, it will close the returned channel so that all subscribers to that
// channel will be notified
func (s *SSHManager) WatchSSHDConfig() (<-chan bool, error) {
	sshdCfgFile := s.sshdConfigFile()
	log.Info("[WatchSSHDConfig] watching file: %s", sshdCfgFile)
	w, evChan, errChan, e := s.newFSWatcher()
	if e != nil {
		log.Error("[WatchSSHDConfig] failed to launch watcher: %v", e)
		return nil, e
	}
	ret := make(chan bool, 1)
	go func() {
		if s.fsWatcherQuitHook != nil {
			defer s.fsWatcherQuitHook()
		}
		defer close(ret)
		for {
			select {
			case ev, ok := <-evChan:
				log.Debug("[WatchSSHDConfig] Event received. [%s]", ev.String())
				if !ok {
					// watcher closed
					log.Info("[WatchSSHDConfig] Events channel closed. Watcher quit")
					return
				}
				if s.sshdCfgModified(w, sshdCfgFile, &ev) {
					ret <- true
				}
			case fsErr, ok := <-errChan:
				if !ok {
					// watcher closed
					log.Info("[WatchSSHDConfig] Errors channel closed. Watcher quit")
					return
				}
				log.Error("received fs watcher error: %v", fsErr)
			}
		}
	}()
	e = w.Add(sshdCfgFile)
	if e != nil {
		_ = w.Close()
		return nil, e
	}
	s.fsWatcher = w
	return ret, nil
}

// Close properly shutdowns the SSH manager
func (s *SSHManager) Close() error {
	if s.fsWatcher != nil {
		return s.fsWatcher.Close()
	}
	return nil
}

// parseSSHDConfig parses the sshd_config file and retrieves configurations needed by the agent, which are:
//  - AuthorizedKeysFile : to know how to locate the authorized_keys file
//  - Port | ListenAddress : to know which port sshd is currently binding to
// NOTES:
//  - the port specified in the command line arguments (--sshd_port) when launching the agent has the highest priority,
//    if given, parseSSHDConfig will skip parsing port numbers specified in the sshd_config
//  - only 1 port is currently supported, if there are multiple ports presented, for example, multiple "Port" entries
//    or more ports are found from `ListenAddress` entry/entries, the agent will only take the first one found, and this
//    *MAY NOT* be the right one. If this happens to be the case, please explicit specify which port the agent should
//    watch via the command line argument "--sshd_port"
func (s *SSHManager) parseSSHDConfig() error {
	defer func() {
		if s.authorizedKeysFilePattern == "" {
			log.Info("Did not find AuthorizedKeysFile pattern from sshd_config, using default pattern:%s", defaultAuthorizedKeysFile)
			s.authorizedKeysFilePattern = defaultAuthorizedKeysFile
		}
		if s.sshdPort == 0 {
			log.Info("Did not find sshd port from sshd_config, using default port:%d", defaultSSHDPort)
			s.sshdPort = defaultSSHDPort
		}
	}()

	sshdConfigBytes, err := s.sysMgr.ReadFile(s.sshdConfigFile())
	if err != nil {
		return fmt.Errorf("%w:%s", ErrSSHDConfigParseFailed, err.Error())
	}
	sshdConfigs := strings.Split(string(sshdConfigBytes), "\n")
	jobDoneCnt := 0
	var errsEncountered []error
	for _, line := range sshdConfigs {
		line = strings.ReplaceAll(line, "#", " #")
		line = strings.ReplaceAll(line, "\t", " ")
		line = strings.TrimLeft(line, " ")
		var e error
		if strings.HasPrefix(line, "AuthorizedKeysFile ") {
			e = s.parseAuthorizedKeysFile(line)
		} else if s.sshdPort == 0 && (strings.HasPrefix(line, "Port") || strings.HasPrefix(line, "ListenAddress")) {
			e = s.parseSSHDPort(line)
		} else {
			continue
		}
		if e == nil {
			jobDoneCnt++
		} else {
			errsEncountered = append(errsEncountered, e)
		}
		if jobDoneCnt == 2 {
			break
		}
	}
	if len(errsEncountered) != 0 {
		log.Error("errors encountered while parsing sshd_config: %v", errsEncountered)
	}
	return nil
}

func (s *SSHManager) parseAuthorizedKeysFile(line string) error {
	keyFiles := strings.Split(line, " ")
	if len(keyFiles) < 2 {
		return fmt.Errorf("%w: invalid format of AuthorizedKeysFile", ErrSSHDConfigParseFailed)
	}
	for i := 1; i != len(keyFiles); i++ {
		keyFile := keyFiles[i]
		if keyFile == "" {
			continue
		}
		if keyFile == "#" {
			break
		}
		if keyFile[0] != '/' {
			keyFile = "%h/" + keyFile
		}
		s.authorizedKeysFilePattern = keyFile
		return nil
	}
	return fmt.Errorf("%w: failed to parse AuthorizedKeysFile", ErrSSHDConfigParseFailed)
}

func (s *SSHManager) parseSSHDPort(line string) error {
	items := strings.Split(line, " ")
	if len(items) < 2 {
		return fmt.Errorf("%w: invalid configuration when parsing sshd port", ErrSSHDConfigParseFailed)
	}
	cfg := ""
	for i := 1; i != len(items); i++ {
		if items[i] == "#" {
			break
		}
		if items[i] != "" {
			cfg = items[i]
			break
		}
	}
	if cfg == "" {
		return fmt.Errorf("%w: failed to find configuration for %v", ErrSSHDConfigParseFailed, items[0])
	}
	switch items[0] {
	case "Port":
		portTmp, err := strconv.Atoi(cfg)
		if err != nil {
			return fmt.Errorf("%w: invalid Port:%v", ErrSSHDConfigParseFailed, err)
		}
		s.sshdPort = portTmp
	case "ListenAddress":
		_, port, err := net.SplitHostPort(cfg)
		if err != nil {
			// failed to fetch the port from the config due to either missing port number or an invalid config,
			// but either case, we skip parsing this line
			break
		}
		portTmp, err := strconv.Atoi(port)
		if err != nil {
			return fmt.Errorf("%w: invalid Port in address:%v", ErrSSHDConfigParseFailed, err)
		}
		s.sshdPort = portTmp
	}
	return nil
}

func validPort(port int) bool {
	return port > 0 && port <= 65535
}
