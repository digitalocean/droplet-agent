// SPDX-License-Identifier: Apache-2.0

//go:build !linux
// +build !linux

package sysutil

import "os"

// CopyFileAttribute copies a file's attribute to another
// Currently this is only required for Linux environment, therefore for non-linux environment it's a no-op
func (s *SysManager) CopyFileAttribute(from, to string) error {
	return nil
}

// ReadFileOfUser reads a file of a user.
// For non-linux environment, the user arg is ignored for now
// TODO: revisit this function when non-linux droplets need to be supported
func (s *SysManager) ReadFileOfUser(filename string, _ *User) ([]byte, error) {
	return os.ReadFile(filename)
}
