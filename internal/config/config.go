// SPDX-License-Identifier: Apache-2.0

package config

import "time"

const (
	AppFullName  = "DigitalOcean Droplet Agent (code name: DOTTY)"
	AppShortName = "Droplet Agent"
	AppDebugAddr = "127.0.0.1:304"

	UserAgent = "Droplet-Agent/" + version

	backgroundJobInterval = 120 * time.Second
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
		AuthorizedKeysCheckInterval: backgroundJobInterval,
	}
}
