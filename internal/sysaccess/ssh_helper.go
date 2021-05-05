// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"encoding/json"
	"fmt"
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
}

type sshHelperImpl struct {
	mgr     *SSHManager
	timeNow func() time.Time
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

func dottyKeyFmt(key *SSHKey, now time.Time) string {
	info := &sshKeyInfo{
		OSUser:     key.OSUser,
		ActorEmail: key.ActorEmail,
		ExpireAt:   now.Add(time.Second * time.Duration(key.TTL)).Format(time.RFC3339),
	}
	keyComment, _ := json.Marshal(info)
	return fmt.Sprintf("%s %s-%s", key.PublicKey, string(keyComment), dottyKeyIndicator)
}
