package sysaccess

type sshMgrOpts struct {
	customSSHDPort    int
	customSSHDCfgFile string
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
