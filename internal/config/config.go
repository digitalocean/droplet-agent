// SPDX-License-Identifier: Apache-2.0

package config

import "time"

const (
	backgroundJobIntervalSeconds = 120
)

// Conf contains the configurations needed to run the agent
type Conf struct {
	Version                     string
	UseSyslog                   bool
	DebugMode                   bool
	AuthorizedKeysCheckInterval time.Duration
}

// Init initializes the agent's configuration
func Init() *Conf {
	cliArgs := parseCLIArgs()
	return &Conf{
		Version:                     version,
		UseSyslog:                   cliArgs.useSyslog,
		DebugMode:                   cliArgs.debugMode,
		AuthorizedKeysCheckInterval: backgroundJobIntervalSeconds * time.Second,
	}
}
