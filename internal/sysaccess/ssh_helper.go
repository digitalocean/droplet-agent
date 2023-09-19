// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/digitalocean/droplet-agent/internal/sysutil"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/crypto/ssh"
)

const (
	manageDropletKeysDisabled uint32 = iota
	manageDropletKeysEnabled
)

type sshHelper interface {
	sshdConfigFile() string
	authorizedKeysFile(user *sysutil.User) string
	prepareAuthorizedKeys(localKeys []string, managedKeys []*SSHKey) []string
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

// prepareAuthorizedKeys prepares the authorized keys that will be updated to filesystem
// NOTE: setting managedKeys to nil or empty slice will result in different behaviors
//   - managedKeys = nil: will result in all temporary keys (keys with a TTL) being removed,
//     but all permanent DO managed droplet keys will be preserved
//   - managedKeys = []*SSHKey{}: means the droplet no longer has any DO managed keys (neither Droplet Keys nor DoTTY Keys),
//     therefore, all DigitalOcean managed keys will be removed
func (s *sshHelperImpl) prepareAuthorizedKeys(localKeys []string, managedKeys []*SSHKey) []string {
	managedDropletKeysEnabled := atomic.LoadUint32(&s.mgr.manageDropletKeys) == manageDropletKeysEnabled
	managedKeysQuickCheck := make(map[string]bool)
	keepLocalDropletKeys := false
	if managedKeys == nil {
		keepLocalDropletKeys = true
	} else {
		for _, k := range managedKeys {
			managedKeysQuickCheck[k.fingerprint] = true
		}
	}

	ret := make([]string, 0, len(localKeys))

	// First, filter out all DO managed keys
	for _, line := range localKeys {
		lineDup := strings.Trim(line, " \t")
		if strings.EqualFold(lineDup, dottyPrevComment) || strings.EqualFold(lineDup, dottyComment) || strings.HasSuffix(lineDup, dottyKeyIndicator) {
			continue
		}
		if managedDropletKeysEnabled && !keepLocalDropletKeys {
			if strings.EqualFold(lineDup, dropletKeyComment) || strings.HasSuffix(lineDup, dropletKeyIndicator) {
				continue
			}
			if pubKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(lineDup)); err == nil {
				// if the line contains a key, check if it should be marked as DOManaged
				fpt := ssh.FingerprintSHA256(pubKey)
				if managedKeysQuickCheck[fpt] {
					continue
				}
			}
		}
		ret = append(ret, line)
	}
	log.Debug("file will contain: [%d] lines of local keys, and [%d] managed keys, manageDropletKeys is set to [%v]", len(ret), len(managedKeys), managedDropletKeysEnabled)

	// Then append all managed keys to the end
	for _, key := range managedKeys {
		if key.Type == SSHKeyTypeDOTTY {
			ret = append(ret, []string{dottyComment, dottyKeyFmt(key)}...)
		} else if managedDropletKeysEnabled {
			ret = append(ret, []string{dropletKeyComment, dropletKeyFmt(key)}...)
		}
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
			if k.Type == SSHKeyTypeDOTTY && timeNow.After(k.expireAt) {
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
	if k.Type == SSHKeyTypeDOTTY {
		if k.TTL <= 0 {
			return fmt.Errorf("%w: invalid ttl", ErrInvalidKey)
		}
		k.expireAt = s.timeNow().Add(time.Duration(k.TTL) * time.Second)
	}
	k.PublicKey = strings.Trim(k.PublicKey, " \t\r\n")
	pubKey, _, _, _, e := ssh.ParseAuthorizedKey([]byte(k.PublicKey))
	if e != nil {
		return fmt.Errorf("%w: invalid ssh key: %s-%v", ErrInvalidKey, k.PublicKey, e)
	}
	k.fingerprint = ssh.FingerprintSHA256(pubKey)
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
	if ev.Op&fsnotify.Write == fsnotify.Write {
		log.Debug("[WatchSSHDConfig] sshd_config modified")
		return true
	} else if ev.Op&(fsnotify.Rename|fsnotify.Remove) != 0 {
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
			s.mgr.sysMgr.Sleep(fileCheckInterval)
		}
		log.Debug("[WatchSSHDConfig] sshd_config ready")
		_ = w.Add(sshdCfgFile)
		return true
	}
	log.Debug("[WatchSSHDConfig] sshd_config not modified, event ignored")
	return false
}

func dottyKeyFmt(key *SSHKey) string {
	info := &sshKeyInfo{
		OSUser:     key.OSUser,
		ActorEmail: key.ActorEmail,
		ExpireAt:   key.expireAt.Format(time.RFC3339),
	}
	keyComment, _ := json.Marshal(info)
	return fmt.Sprintf("%s %s-%s", key.PublicKey, string(keyComment), dottyKeyIndicator)
}

func dropletKeyFmt(key *SSHKey) string {
	return fmt.Sprintf("%s -%s", key.PublicKey, dropletKeyIndicator)
}
