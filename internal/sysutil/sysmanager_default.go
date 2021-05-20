// SPDX-License-Identifier: Apache-2.0

// +build !linux

package sysutil


// CopyFileAttribute copies a file's attribute to another
// Currently this is only required for Linux environment, therefore for non-linux environment it's a no-op
func  (s *SysManager) CopyFileAttribute(from, to string) error {
	return nil
}
