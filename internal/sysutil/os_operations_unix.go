// SPDX-License-Identifier: Apache-2.0

// +build !windows

package sysutil

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

func newOSOperator() osOperator {
	return &osOperatorImpl{
		readFileFn: ioutil.ReadFile,
		osStatFn:   os.Stat,
		osMkDir:    os.MkdirAll,
		osChown:    os.Chown,
		osOpenFile: func(name string, flag int, perm os.FileMode) (io.WriteCloser, error) {
			return os.OpenFile(name, flag, perm)
		},
		osRemove: os.Remove,
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
	readFileFn func(filename string) ([]byte, error)
	osStatFn   func(name string) (os.FileInfo, error)
	osMkDir    func(path string, perm os.FileMode) error
	osChown    func(name string, uid, gid int) error
	osOpenFile func(name string, flag int, perm os.FileMode) (io.WriteCloser, error)
	osRemove   func(name string) error
}

func (o *osOperatorImpl) getpwnam(username string) (*User, error) {
	content, err := o.readFileFn("/etc/passwd")
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
	if _, err := o.osStatFn(dir); err != nil {
		if os.IsNotExist(err) {
			if err = o.osMkDir(dir, perm); err != nil {
				return fmt.Errorf("%w: mkdir failed: %v", ErrMakeDirFailed, err)
			}
			if err = o.osChown(dir, user.UID, user.GID); err != nil {
				return fmt.Errorf("%w: chown failed: %v", ErrMakeDirFailed, err)
			}
		} else {
			return fmt.Errorf("%w: os.Stat failed: %v", ErrMakeDirFailed, err)
		}
	}
	return nil
}

func (o *osOperatorImpl) createFileForWrite(file string, user *User, perm os.FileMode) (io.WriteCloser, error) {
	f, err := o.osOpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return nil, fmt.Errorf("%w: open file failed: %v", ErrCreateFileFailed, err)
	}

	if err := o.osChown(file, user.UID, user.GID); err != nil {
		_ = f.Close()
		_ = o.osRemove(file)
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
