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
	updateAuthorizedKeysFile(osUsername string, managedKeys []*SSHKey) error
}

type updaterImpl struct {
	sshMgr *SSHManager

	keysFileLocks sync.Map
}

func (u *updaterImpl) updateAuthorizedKeysFile(osUsername string, managedKeys []*SSHKey) error {
	osUser, err := u.sshMgr.sysMgr.GetUserByName(osUsername)
	if err != nil {
		return err
	}
	authorizedKeysFile := u.sshMgr.authorizedKeysFile(osUser)

	// We must make sure we are exclusively accessing the authorized_keys file
	keysFileLockRaw, _ := u.keysFileLocks.LoadOrStore(authorizedKeysFile, &sync.Mutex{})
	keysFileLock := keysFileLockRaw.(*sync.Mutex)
	keysFileLock.Lock()
	defer keysFileLock.Unlock()

	dir := filepath.Dir(authorizedKeysFile)
	log.Debug("ensuring dir [%s] exists for user [%s]", dir, osUser.Name)
	if err = u.sshMgr.sysMgr.MkDirIfNonExist(dir, osUser, 0700); err != nil {
		return err
	}
	fileExist := true
	localKeysRaw, err := u.sshMgr.sysMgr.ReadFile(authorizedKeysFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("%w:%v", ErrReadAuthorizedKeysFileFailed, err)
		}
		fileExist = false
	}
	localKeys := make([]string, 0)
	if localKeysRaw != nil {
		localKeys = strings.Split(strings.TrimRight(string(localKeysRaw), "\n"), "\n")
	}
	updatedKeys := u.sshMgr.prepareAuthorizedKeys(localKeys, managedKeys)
	return u.do(authorizedKeysFile, osUser, updatedKeys, fileExist)
}

func (u *updaterImpl) do(authorizedKeysFile string, user *sysutil.User, lines []string, srcFileExist bool) (retErr error) {
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

	if srcFileExist {
		log.Debug("copying file attribute from [%s] to [%s]", authorizedKeysFile, tmpFilePath)
		err := u.sshMgr.sysMgr.CopyFileAttribute(authorizedKeysFile, tmpFilePath)
		if err != nil {
			return fmt.Errorf("%w:failed to apply file attribute :%v", ErrWriteAuthorizedKeysFileFailed, err)
		}
	}

	if err := u.sshMgr.sysMgr.RenameFile(tmpFilePath, authorizedKeysFile); err != nil {
		return fmt.Errorf("%w:failed to rename:%v", ErrWriteAuthorizedKeysFileFailed, err)
	}
	return nil
}
