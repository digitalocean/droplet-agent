// SPDX-License-Identifier: Apache-2.0

package sysutil

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
)

// Possible errors
var (
	// ErrGetUserFailed indicates the system call for fetching user entry from passwd has failed
	ErrGetUserFailed = fmt.Errorf("failed to get user")
	// ErrUserNotFound indicates the user does not exist in the system
	ErrUserNotFound = fmt.Errorf("user not found")
	// ErrMakeDirFailed indicates the system call for making a directory has failed
	ErrMakeDirFailed = fmt.Errorf("failed to make directory")
	// ErrCreateFileFailed indicates the error of failing to create a file
	ErrCreateFileFailed = fmt.Errorf("failed to create file")
	// ErrFileNotFound indicates a file not exist error
	ErrFileNotFound = fmt.Errorf("file not found")
	// ErrOpenFileFailed indicates the error of failing to open a file
	ErrOpenFileFailed = fmt.Errorf("failed to open file")
	// ErrRunCmdFailed is returned when a command is failed to run
	ErrRunCmdFailed = fmt.Errorf("failed to run command")
	// ErrInvalidFileType is returned when the file type is unexpected
	ErrInvalidFileType = fmt.Errorf("invalid file type")
	// ErrUnexpected indicates an unexpected error
	ErrUnexpected = fmt.Errorf("unexpected error")
	// ErrPermissionDenied indicates the given permission is not sufficient to perform the designated operation
	ErrPermissionDenied = fmt.Errorf("insufficient permission")
	// ErrReadAuthorizedKeysFileFailed indicates the error of failing to read the authorized_keys file
	ErrReadAuthorizedKeysFileFailed = errors.New("failed to read authorized_keys file")
	// ErrWriteAuthorizedKeysFileFailed indicates the error of failing to write the authorized_keys file
	ErrWriteAuthorizedKeysFileFailed = errors.New("failed to write authorized_keys file")
)

// User struct contains information of a user
type User struct {
	Name    string
	UID     int
	GID     int
	HomeDir string
}

// CmdResult struct contains the result of executing a command
type CmdResult struct {
	ExitCode int
	StdOut   string
	StdErr   string
}

// File contains common operations on *os.File
type File interface {
	Name() string
	Close() error
	Stat() (os.FileInfo, error)

	io.ReadWriteCloser
}

type osOperator interface {
	getpwnam(username string) (*User, error)
	mkdir(dir string, user *User, perm os.FileMode) error
	createTempFile(dir, pattern string, user *User) (File, error)
	openFile(name string, flag int, perm os.FileMode) (File, error)
	executable() (string, error)
	evalSymLinks(path string) (string, error)
	command(name string, args ...string) cmd
	dir(path string) string
	newBuffer() bytes.Buffer
	newStringReader(contents string) io.Reader
}

type cmd interface {
	Run() error
	SetStdout(io.Writer)
	SetStdin(io.Reader)
	SetStderr(io.Writer)
	SetDir(string)
	SetUser(user *User)
}

func newCmd(name string, args ...string) cmd {
	return &cmdImpl{
		exec.Command(name, args...),
	}
}

type cmdImpl struct {
	cmd *exec.Cmd
}

func (c *cmdImpl) Run() error {
	return c.cmd.Run()
}

func (c *cmdImpl) SetStdout(w io.Writer) {
	c.cmd.Stdout = w
}

func (c *cmdImpl) SetStdin(r io.Reader) {
	c.cmd.Stdin = r
}

func (c *cmdImpl) SetStderr(w io.Writer) {
	c.cmd.Stderr = w
}

func (c *cmdImpl) SetDir(dir string) {
	c.cmd.Dir = dir
}

func (c *cmdImpl) SetUser(user *User) {
	if user != nil {
		c.cmd.SysProcAttr = &syscall.SysProcAttr{
			Credential: &syscall.Credential{
				Uid: uint32(user.UID),
				Gid: uint32(user.GID),
			},
		}
	} else {
		c.cmd.SysProcAttr = nil
	}
}
