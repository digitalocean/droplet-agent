// SPDX-License-Identifier: Apache-2.0

// +build !windows

package sysaccess

// sshd_config can be a different file if sshd is started with "-f" option
// this can be fixed by parsing the command line arguments of the sshd process
// but that complicates the cross-OS support.
// if the running sshd is launched with a custom sshd_config, the path to the
// sshd_config file must be specified when launching the agent via the command
// line argument "-sshd_config"
func (s *sshHelperImpl) sshdConfigFile() string {
	if s.customSSHDCfgFile != "" {
		return s.customSSHDCfgFile
	}
	return "/etc/ssh/sshd_config"
}
