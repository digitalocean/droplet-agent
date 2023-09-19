// SPDX-License-Identifier: Apache-2.0

package sysutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// NewSysManager returns a new SysManager Object
func NewSysManager() *SysManager {
	return &SysManager{
		osOperator: newOSOperator(),
	}
}

// SysManager is the tool for interacting with the OS
type SysManager struct {
	osOperator
}

// ReadFile reads a file
func (s *SysManager) ReadFile(filename string) ([]byte, error) {
	return os.ReadFile(filename)
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

// CreateFileForWrite creates a file for write
func (s *SysManager) CreateFileForWrite(file string, user *User, perm os.FileMode) (io.WriteCloser, error) {
	return s.createFileForWrite(file, user, perm)
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

func (s *SysManager) Sleep(d time.Duration) {
	time.Sleep(d)
}

// RunCmd runs a command and return the result
func (s *SysManager) RunCmd(name string, arg ...string) (*CmdResult, error) {
	var stdOut, stdErr bytes.Buffer
	cmd := exec.Command(name, arg...)
	cmd.Stdout = &stdOut
	cmd.Stderr = &stdErr
	err := cmd.Run()
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			return &CmdResult{
				ExitCode: e.ExitCode(),
				StdErr:   stdErr.String(),
				StdOut:   stdOut.String(),
			}, nil
		}
		return nil, fmt.Errorf("%w:%v", ErrRunCmdFailed, err)
	}
	return &CmdResult{
		ExitCode: 0,
		StdOut:   stdOut.String(),
	}, nil
}
