// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"encoding/json"
	"fmt"
	"github.com/fsnotify/fsnotify"
	"os"
	"strings"
	"time"

	"github.com/digitalocean/droplet-agent/internal/log"

	"golang.org/x/crypto/ssh"

	"github.com/digitalocean/droplet-agent/internal/sysutil"
)

type sshHelper interface {
	sshdConfigFile() string
	authorizedKeysFile(user *sysutil.User) string
	prepareAuthorizedKeys(localKeys []string, dottyKeys []*SSHKey) []string
	removeExpiredKeys(originalKeys map[string][]*SSHKey) (filteredKeys map[string][]*SSHKey)
	areSameKeys(keys1, keys2 []*SSHKey) bool
	validateKey(k *SSHKey) error
	newFSWatcher() (fsWatcher, <-chan fsnotify.Event, <-chan error, error)
	sshdCfgModified(w fsWatcher, sshdCfgFile string, ev *fsnotify.Event) bool
}

type fsWatcher interface {
	Add(name string) error
	Remove(name string) error
	Close() error
}

type sshHelperImpl struct {
	mgr     *SSHManager
	timeNow func() time.Time

	customSSHDCfgFile string
}

func (s *sshHelperImpl) authorizedKeysFile(user *sysutil.User) string {
	filePath := s.mgr.authorizedKeysFilePattern
	filePath = strings.ReplaceAll(filePath, "%%", "%")
	filePath = strings.ReplaceAll(filePath, "%h", strings.TrimRight(user.HomeDir, string(os.PathSeparator)))
	filePath = strings.ReplaceAll(filePath, "%u", user.Name)
	return filePath
}

func (s *sshHelperImpl) prepareAuthorizedKeys(localKeys []string, dottyKeys []*SSHKey) []string {
	ret := make([]string, 0, len(localKeys))

	// First, filter out all dotty keys
	for _, line := range localKeys {
		lineDup := strings.Trim(line, " \t")
		if lineDup == dottyPrevComment || lineDup == dottyComment || strings.HasSuffix(lineDup, dottyKeyIndicator) {
			continue
		}
		ret = append(ret, line)
	}
	log.Debug("file will contain: [%d] lines of local keys, and [%d] dotty keys", len(ret), len(dottyKeys))

	// Then append all dotty keys to the end
	for _, key := range dottyKeys {
		ret = append(ret, []string{dottyComment, dottyKeyFmt(key, s.timeNow())}...)
	}
	return ret
}

func (s *sshHelperImpl) removeExpiredKeys(originalKeys map[string][]*SSHKey) (filteredKeys map[string][]*SSHKey) {
	if len(originalKeys) == 0 {
		return originalKeys
	}
	filteredKeys = make(map[string][]*SSHKey)
	timeNow := s.timeNow()
	for user, keys := range originalKeys {
		if len(keys) == 0 {
			continue
		}
		filteredKeys[user] = make([]*SSHKey, 0, len(keys))
		for _, k := range keys {
			if timeNow.After(k.expireAt) {
				// key already expired
				continue
			}
			filteredKeys[user] = append(filteredKeys[user], k)
		}
		if len(filteredKeys[user]) == 0 {
			delete(filteredKeys, user)
		}
	}
	return
}
func (s *sshHelperImpl) validateKey(k *SSHKey) (err error) {
	if k.OSUser == "" {
		k.OSUser = defaultOSUser
	}
	defer func() {
		if err == nil {
			k.expireAt = s.timeNow().Add(time.Duration(k.TTL) * time.Second)
		}
	}()
	if k.TTL <= 0 {
		return fmt.Errorf("%w: invalid ttl", ErrInvalidKey)
	}
	k.PublicKey = strings.Trim(k.PublicKey, " \t\r\n")
	if _, _, _, _, e := ssh.ParseAuthorizedKey([]byte(k.PublicKey)); e != nil {
		return fmt.Errorf("%w: invalid ssh key: %s-%v", ErrInvalidKey, k.PublicKey, e)
	}
	return nil
}

func (s *sshHelperImpl) areSameKeys(keys1, keys2 []*SSHKey) bool {
	if keys1 == nil || keys2 == nil {
		return keys1 == nil && keys2 == nil
	}
	if len(keys1) != len(keys2) {
		return false
	}
	keyIdx := func(k *SSHKey) string {
		return fmt.Sprintf("%s:%s", k.OSUser, k.PublicKey)
	}
	counts := make(map[string]int)
	for _, k := range keys1 {
		idx := keyIdx(k)
		counts[idx]++
	}
	for _, k := range keys2 {
		idx := keyIdx(k)
		counts[idx]--
	}
	for _, c := range counts {
		if c != 0 {
			return false
		}
	}
	return true
}

func (s *sshHelperImpl) newFSWatcher() (fsWatcher, <-chan fsnotify.Event, <-chan error, error) {
	w, e := fsnotify.NewWatcher()
	if e != nil {
		return nil, nil, nil, e
	}
	return w, w.Events, w.Errors, nil
}
func (s *sshHelperImpl) sshdCfgModified(w fsWatcher, sshdCfgFile string, ev *fsnotify.Event) bool {
	if ev.Name != sshdCfgFile {
		return false
	}
	log.Info("[WatchSSHDConfig] sshd_config events detected.")
	if ev.Op&(fsnotify.Rename|fsnotify.Remove) != 0 {
		// if sshd_config is being renamed or removed, wait until it appears again
		log.Debug("[WatchSSHDConfig] sshd_config was renamed or removed, waiting until it's back")
		if err := w.Remove(sshdCfgFile); err != nil {
			log.Error("[WatchSSHDConfig] failed to stop monitoring old sshd_config: %v", err)
		}
		// the reasons for having the wait loop here are:
		// - when the sshd_config is removed or not presented in the configured path,
		//   restarting the droplet-agent will result in failure, therefore, to prevent a
		//   restart burst to the systemd, we wait until the file is ready
		// - removing the sshd_config will not impact the sshd service until it is restarted,
		//   therefore, we don't necessarily need to restart the droplet-agent service unless
		//   a new sshd_config file is presented
		for {
			if exists, _ := s.mgr.sysMgr.FileExists(sshdCfgFile); exists {
				break
			}
			time.Sleep(fileCheckInterval)
		}
		log.Debug("[WatchSSHDConfig] sshd_config ready")
		_ = w.Add(sshdCfgFile)
		return true
	} else if ev.Op&fsnotify.Write == fsnotify.Write {
		log.Debug("[WatchSSHDConfig] sshd_config modified")
		return true
	}
	log.Debug("[WatchSSHDConfig] sshd_config not modified, event ignored")
	return false
}

func dottyKeyFmt(key *SSHKey, now time.Time) string {
	info := &sshKeyInfo{
		OSUser:     key.OSUser,
		ActorEmail: key.ActorEmail,
		ExpireAt:   now.Add(time.Second * time.Duration(key.TTL)).Format(time.RFC3339),
	}
	keyComment, _ := json.Marshal(info)
	return fmt.Sprintf("%s %s-%s", key.PublicKey, string(keyComment), dottyKeyIndicator)
}
