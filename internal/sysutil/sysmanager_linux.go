// SPDX-License-Identifier: Apache-2.0

package sysutil

import "github.com/opencontainers/selinux/go-selinux"

func  (s *SysManager) CopyFileAttribute(from, to string) error {
	if !selinux.GetEnabled() {
		return nil
	}
	srcLabel, err := selinux.FileLabel(from)
	if err != nil {
		return err
	}
	return selinux.SetFileLabel(to, srcLabel)
}
