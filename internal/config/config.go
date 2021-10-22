// SPDX-License-Identifier: Apache-2.0

package config

import "time"

const (
	AppFullName  = "DigitalOcean Droplet Agent (code name: DoTTY)"
	AppShortName = "Droplet Agent"
)

const (
	backgroundJobIntervalSeconds = 120
)

// Conf contains the configurations needed to run the agent
type Conf struct {
	Version                     string
	UseSyslog                   bool
	DebugMode                   bool
	AuthorizedKeysCheckInterval time.Duration
	CustomSSHDPort              int
	CustomSSHDCfgFile           string
}

// Init initializes the agent's configuration
func Init() *Conf {
	cliArgs := parseCLIArgs()
	return &Conf{
		Version:                     version,
		UseSyslog:                   cliArgs.useSyslog,
		DebugMode:                   cliArgs.debugMode,
		CustomSSHDPort:              cliArgs.sshdPort,
		CustomSSHDCfgFile:           cliArgs.sshdCfgFile,
		AuthorizedKeysCheckInterval: backgroundJobIntervalSeconds * time.Second,
	}
}
