// SPDX-License-Identifier: Apache-2.0

package sysutil

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// NewSysManager returns a new SysManager Object
func NewSysManager() *SysManager {
	return &SysManager{
		osOperator:   newOSOperator(),
		userOperator: newUserOperator(),
	}
}

// SysManager is the tool for interacting with the OS
type SysManager struct {
	osOperator
	userOperator
}

// ReadFile reads a file
func (s *SysManager) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename) //nolint:gosec
}

// RenameFile renames a file
func (s *SysManager) RenameFile(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

// GetUserByName gets an OS user info
func (s *SysManager) GetUserByName(username string) (*User, error) {
	return s.getpwnam(username)
}

// RemoveFile removes a file
func (s *SysManager) RemoveFile(name string) error {
	return os.Remove(name)
}

// MkDirIfNonExist creates a directory if it does not exist
func (s *SysManager) MkDirIfNonExist(dir string, user *User, perm os.FileMode) error {
	return s.mkdir(dir, user, perm)
}

// CreateTempFile creates a temporary file for the designated user to read and write
func (s *SysManager) CreateTempFile(dir, pattern string, user *User) (File, error) {
	return s.createTempFile(dir, pattern, user)
}

// FileExists checks whether a file exists or not
func (s *SysManager) FileExists(name string) (bool, error) {
	_, err := os.Stat(name)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Sleep pauses execution
func (s *SysManager) Sleep(d time.Duration) {
	time.Sleep(d)
}

// UtilSubprocess calls this binary in util mode in a subprocess
func (s *SysManager) UtilSubprocess(user *User, stdin []string) (*CmdResult, error) {
	exStart, err := s.executable()
	if err != nil {
		return nil, err
	}

	ex, err := s.evalSymLinks(exStart)
	if err != nil {
		return nil, err
	}

	cmd := s.command(ex, "-util")

	stdOut := s.newBuffer()
	stdErr := s.newBuffer()
	cmd.SetStdout(&stdOut)
	cmd.SetStderr(&stdErr)
	cmd.SetDir(s.dir(ex))

	if len(stdin) > 0 {
		cmd.SetStdin(s.newStringReader(strings.Join(stdin, "\n")))
	}

	cmd.SetUser(user)

	if err := cmd.Run(); err != nil {
		var e *exec.ExitError
		if errors.As(err, &e) {
			return &CmdResult{
				ExitCode: e.ExitCode(),
				StdErr:   stdErr.String(),
				StdOut:   stdOut.String(),
			}, nil
		}
		return nil, fmt.Errorf("%w: %v", ErrRunCmdFailed, err)
	}

	return &CmdResult{
		ExitCode: 0,
		StdOut:   stdOut.String(),
		StdErr:   stdErr.String(),
	}, nil
}

// GetCurrentUser retrieves the current user information
func (s *SysManager) GetCurrentUser() (*User, error) {
	usr, err := s.Current()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrGetUserFailed, err)
	}

	uid, err := strconv.ParseUint(usr.Uid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid UID %s: %v", ErrGetUserFailed, usr.Uid, err)
	}

	gid, err := strconv.ParseUint(usr.Gid, 10, 32)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid GID %s: %v", ErrGetUserFailed, usr.Gid, err)
	}

	return &User{
		Name:    usr.Username,
		UID:     uint32(uid),
		GID:     uint32(gid),
		HomeDir: usr.HomeDir,
	}, nil
}

// IsSymLink checks if the given path is a symbolic link
func (s *SysManager) IsSymLink(path string) (bool, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return false, err
	}
	return info.Mode()&os.ModeSymlink != 0, nil
}
