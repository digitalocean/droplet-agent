// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/digitalocean/droplet-agent/internal/log"

	"github.com/digitalocean/droplet-agent/internal/sysutil"
)

type authorizedKeysFileUpdater interface {
	updateAuthorizedKeysFile(osUsername string, dottyKeys []*SSHKey) error
}

type updaterImpl struct {
	sshMgr *SSHManager

	userLocks sync.Map
}

func (u *updaterImpl) updateAuthorizedKeysFile(osUsername string, dottyKeys []*SSHKey) error {

	userLockRaw, _ := u.userLocks.LoadOrStore(osUsername, &sync.Mutex{})
	userLock := userLockRaw.(*sync.Mutex)
	userLock.Lock()
	defer userLock.Unlock()

	osUser, err := u.sshMgr.sysMgr.GetUserByName(osUsername)
	if err != nil {
		return err
	}
	authorizedKeysFile := u.sshMgr.authorizedKeysFile(osUser)
	dir := filepath.Dir(authorizedKeysFile)
	if err = u.sshMgr.sysMgr.MkDirIfNonExist(dir, osUser, 0700); err != nil {
		return err
	}
	localKeysRaw, err := u.sshMgr.sysMgr.ReadFile(authorizedKeysFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("%w:%v", ErrReadAuthorizedKeysFileFailed, err)
	}
	localKeys := make([]string, 0)
	if localKeysRaw != nil {
		localKeys = strings.Split(strings.TrimRight(string(localKeysRaw), "\n"), "\n")
	}
	updatedKeys := u.sshMgr.prepareAuthorizedKeys(localKeys, dottyKeys)
	if err = u.do(authorizedKeysFile, osUser, updatedKeys); err != nil {
		return err
	}
	return nil
}

func (u *updaterImpl) do(authorizedKeysFile string, user *sysutil.User, lines []string) (retErr error) {
	log.Debug("updating [%s]", authorizedKeysFile)
	tmpFilePath := authorizedKeysFile + ".dotty"
	tmpFile, err := u.sshMgr.sysMgr.CreateFileForWrite(tmpFilePath, user, 0600)
	if err != nil {
		return fmt.Errorf("%w: failed to create tmp file: %v", ErrWriteAuthorizedKeysFileFailed, err)
	}
	defer func() {
		log.Debug("[%s] updated", authorizedKeysFile)
		_ = tmpFile.Close()
		if retErr != nil {
			_ = u.sshMgr.sysMgr.RemoveFile(tmpFilePath)
		}
	}()

	for _, l := range lines {
		_, _ = fmt.Fprintf(tmpFile, "%s\n", l)
	}

	_, _ = u.sshMgr.sysMgr.RunCmd("restorecon", tmpFilePath)

	if err := u.sshMgr.sysMgr.RenameFile(tmpFilePath, authorizedKeysFile); err != nil {
		return fmt.Errorf("%w:failed to rename:%v", ErrWriteAuthorizedKeysFileFailed, err)
	}
	return nil
}
