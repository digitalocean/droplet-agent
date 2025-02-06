// SPDX-License-Identifier: Apache-2.0

//go:build !windows
// +build !windows

package sysutil

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

func newOSOperator() osOperator {
	return &osOperatorImpl{
		osOpHelper: &osOpHelperImpl{},
	}
}

const (
	passwdIdxName    = 0
	passwdIdxUID     = 2
	passwdIdxGID     = 3
	passwdIdxHomeDir = 5
	passwdIdxShell   = 6
)

type osOperatorImpl struct {
	osOpHelper
}

func (o *osOperatorImpl) getpwnam(username string) (*User, error) {
	content, err := o.ReadFile("/etc/passwd")
	if err != nil {
		return nil, fmt.Errorf("%w: error getting user info for:%s. error: %v", ErrGetUserFailed, username, err)
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		entry, err := parseLine(line)
		if err != nil {
			continue
		}
		if entry.Name == username {
			return entry, nil
		}
	}
	return nil, fmt.Errorf("%w: user %s not found", ErrUserNotFound, username)
}

func (o *osOperatorImpl) mkdir(dir string, user *User, perm os.FileMode) error {
	if _, err := o.Stat(dir); err != nil {
		if os.IsNotExist(err) {
			if err = o.MkDir(dir, perm); err != nil {
				return fmt.Errorf("%w: mkdir failed: %v", ErrMakeDirFailed, err)
			}
			if err = o.Chown(dir, user.UID, user.GID); err != nil {
				return fmt.Errorf("%w: chown failed: %v", ErrMakeDirFailed, err)
			}
		} else {
			return fmt.Errorf("%w: os.Stat failed: %v", ErrMakeDirFailed, err)
		}
	}
	return nil
}

func (o *osOperatorImpl) createFileForWrite(file string, user *User, perm os.FileMode) (io.WriteCloser, error) {
	return nil, fmt.Errorf("deprecated")
}

func (o *osOperatorImpl) createTempFile(dir, pattern string, user *User) (File, error) {
	f, err := o.CreateTemp(dir, pattern)
	if err != nil {
		return nil, fmt.Errorf("%w: open file failed: %v", ErrCreateFileFailed, err)
	}
	if err := o.Chown(f.Name(), user.UID, user.GID); err != nil {
		_ = f.Close()
		_ = o.Remove(f.Name())
		return nil, fmt.Errorf("%w: failed to set owner: %v", ErrCreateFileFailed, err)
	}
	return f, nil
}

func parseLine(line string) (*User, error) {
	ret := &User{}
	items := strings.Split(line, ":")
	if len(items) != 7 {
		return nil, fmt.Errorf("invalid line: [%s] contains unexpected number of items. %d != 7", line, len(items))
	}
	ret.Name = strings.TrimSpace(items[passwdIdxName])

	uid, err := strconv.ParseInt(items[passwdIdxUID], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid line: %s is not a valid uid", items[passwdIdxUID])
	}
	ret.UID = int(uid)

	gid, err := strconv.ParseInt(items[passwdIdxGID], 10, 32)
	if err != nil {
		return nil, fmt.Errorf("invalid line: %s is not a valid gid", items[passwdIdxGID])
	}
	ret.GID = int(gid)

	ret.HomeDir = strings.TrimSpace(items[passwdIdxHomeDir])
	ret.Shell = strings.TrimSpace(items[passwdIdxShell])
	return ret, nil
}
