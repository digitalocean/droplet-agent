// SPDX-License-Identifier: Apache-2.0

package config

import "time"

const (
	AppFullName  = "DigitalOcean Droplet Agent (code name: DOTTY)"
	AppShortName = "Droplet Agent"
	AppDebugAddr = "127.0.0.1:304"

	UserAgent = "Droplet-Agent/" + Version

	backgroundJobInterval = 120 * time.Second
)

// Conf contains the configurations needed to run the agent
type Conf struct {
	UseSyslog bool
	DebugMode bool

	CustomSSHDPort              int
	CustomSSHDCfgFile           string
	AuthorizedKeysCheckInterval time.Duration
}

// Init initializes the agent's configuration
func Init() *Conf {
	cliArgs := parseCLIArgs()
	return &Conf{
		UseSyslog:                   cliArgs.useSyslog,
		DebugMode:                   cliArgs.debugMode,
		CustomSSHDPort:              cliArgs.sshdPort,
		CustomSSHDCfgFile:           cliArgs.sshdCfgFile,
		AuthorizedKeysCheckInterval: backgroundJobInterval,
	}
}
