// SPDX-License-Identifier: Apache-2.0

package sysutil

import (
	"fmt"
	"github.com/digitalocean/droplet-agent/internal/log"
	"github.com/opencontainers/selinux/go-selinux"
	"io"
	"os"
	"syscall"
)

// CopyFileAttribute copies a file's attribute to another
// In Linux, this is specifically designed to apply the selinux labels of a file to another
func (s *SysManager) CopyFileAttribute(from, to string) error {
	if !selinux.GetEnabled() {
		return nil
	}
	srcLabel, err := selinux.FileLabel(from)
	if err != nil {
		return err
	}
	err = selinux.SetFileLabel(to, srcLabel)
	if err == nil {
		log.Debug("SELinux context applied!")
	}
	return err
}

// ReadFileOfUser reads a file of the given user
// either the user is root (i.e. uid=0), or the file has to be owned by the user
func (s *SysManager) ReadFileOfUser(filename string, user *User) ([]byte, error) {
	file, err := s.openFile(filename, os.O_RDONLY, 0)
	if err != nil {
		return nil, fmt.Errorf("%w:failed to open file:%v", ErrOpenFileFailed, err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("%w: failed to stat:%v", ErrUnexpected, err)
	}
	if !info.Mode().IsRegular() {
		return nil, ErrInvalidFileType
	}
	if user.UID != 0 {
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			return nil, fmt.Errorf("%w: failed to check file stat", ErrUnexpected)
		}
		if stat.Uid != uint32(user.UID) {
			return nil, ErrPermissionDenied
		}
	}
	return io.ReadAll(file)
}
