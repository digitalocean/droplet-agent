// SPDX-License-Identifier: Apache-2.0

package sysutil

import (
	"fmt"
	"io"
	"os"
)

// Possible errors
var (
	// ErrGetUserFailed indicates the system call for fetching user entry from passwd has failed
	ErrGetUserFailed = fmt.Errorf("failed to get user")
	// ErrMakeDirFailed indicates the system call for making a directory has failed
	ErrMakeDirFailed = fmt.Errorf("failed to make directory")
	// ErrCreateFileFailed indicates the error of failing to create a file
	ErrCreateFileFailed = fmt.Errorf("failed to create file")
	// ErrRunCmdFailed is returned when a command is failed to run
	ErrRunCmdFailed = fmt.Errorf("failed to run command")
)

// User struct contains information of a user
type User struct {
	Name    string
	UID     int
	GID     int
	HomeDir string
	Shell   string
}

// CmdResult struct contains the result of executing a command
type CmdResult struct {
	ExitCode int
	StdOut   string
	StdErr   string
}

type osOperator interface {
	getpwnam(username string) (*User, error)
	mkdir(dir string, user *User, perm os.FileMode) error
	createFileForWrite(file string, user *User, perm os.FileMode) (io.WriteCloser, error)
}
