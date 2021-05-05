// SPDX-License-Identifier: Apache-2.0

// +build !windows

package sysaccess

// sshd_config can be a different file if sshd is started with "-f" option
// this can be fixed by parsing the command line arguments of the sshd process
// but that complicates the cross-OS support.
// As launching sshd on port 22 with a custom sshd_config file is a fair rare case,
// it's not supported for now to avoid unnecessary over-engineering.
func (s *sshHelperImpl) sshdConfigFile() string {
	return "/etc/ssh/sshd_config"
}
