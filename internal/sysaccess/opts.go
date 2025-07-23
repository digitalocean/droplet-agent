// SPDX-License-Identifier: Apache-2.0

package sysaccess

type sshMgrOpts struct {
	customSSHDPort    int
	customSSHDCfgFile string
	manageDropletKeys bool
}

// SSHManagerOpt allows creating the SSHManager instance with designated options
type SSHManagerOpt func(opt *sshMgrOpts)

// WithCustomSSHDPort indicates the SSHD is running on a custom port which is specified via command line argument
func WithCustomSSHDPort(port int) SSHManagerOpt {
	return func(opt *sshMgrOpts) {
		opt.customSSHDPort = port
	}
}

// WithCustomSSHDCfg specifies the path the custom sshd_config file that the sshd instance uses
func WithCustomSSHDCfg(cfgFile string) SSHManagerOpt {
	return func(opt *sshMgrOpts) {
		opt.customSSHDCfgFile = cfgFile
	}
}

// WithoutManagingDropletKeys tells the agent to not attempt to manage the ssh keys
func WithoutManagingDropletKeys() SSHManagerOpt {
	return func(opt *sshMgrOpts) {
		opt.manageDropletKeys = false
	}
}

func defaultMgrOpts() *sshMgrOpts {
	return &sshMgrOpts{
		customSSHDPort:    0,
		customSSHDCfgFile: "",
		manageDropletKeys: true,
	}
}
