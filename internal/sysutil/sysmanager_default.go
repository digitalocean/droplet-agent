// SPDX-License-Identifier: Apache-2.0

// +build !linux

package sysutil

func  (s *SysManager) CopyFileAttribute(from, to string) error {
	return nil
}
