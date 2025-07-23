// SPDX-License-Identifier: Apache-2.0

package sysaccess

import (
	"fmt"
	"strings"
	"sync"

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

	res, err := u.sshMgr.sysMgr.UtilSubprocess(osUser, nil)
	if e := u.checkSubprocessError(res, err, ErrReadAuthorizedKeysFileFailed); e != nil {
		return e
	}

	var localKeys []string
	if res != nil && res.StdOut != "" {
		localKeys = strings.Split(strings.TrimSpace(res.StdOut), "\n")
	}
	updatedKeys := u.sshMgr.prepareAuthorizedKeys(localKeys, managedKeys)

	ret, err := u.sshMgr.sysMgr.UtilSubprocess(osUser, updatedKeys)
	if e := u.checkSubprocessError(ret, err, ErrWriteAuthorizedKeysFileFailed); e != nil {
		return e
	}

	return nil
}

func (u *updaterImpl) checkSubprocessError(res *sysutil.CmdResult, err error, parent error) error {
	if err != nil {
		return fmt.Errorf("%w: subprocess error: %v", parent, err)
	}

	if res != nil && res.ExitCode != 0 {
		return fmt.Errorf("%w: subprocess exit %d: %s", parent, res.ExitCode, res.StdErr)
	}

	return nil
}
