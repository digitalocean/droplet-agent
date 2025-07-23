// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"os"
	"time"

	"github.com/peterbourgon/ff/v3"
)

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
	UtilMode  bool

	CustomSSHDPort              int
	CustomSSHDCfgFile           string
	AuthorizedKeysCheckInterval time.Duration
}

// Init initializes the agent's configuration
func Init() *Conf {
	cfg := Conf{
		AuthorizedKeysCheckInterval: backgroundJobInterval,
	}

	args := os.Args[1:]
	for i, arg := range args {
		// util mode is used for subprocessing file utility interactions
		// it is hidden from the cli to not cause unnecessary confusion
		if arg == "-util" {
			cfg.UtilMode = true
			args = append(args[:i], args[i+1:]...)
			break
		}
	}

	fs := flag.NewFlagSet("droplet-agent", flag.ExitOnError)

	fs.BoolVar(&cfg.UseSyslog, "syslog", false, "Use syslog service for logging")
	fs.BoolVar(&cfg.DebugMode, "debug", false, "Turn on debug mode")
	fs.IntVar(&cfg.CustomSSHDPort, "sshd_port", 0, "The port sshd is binding to")
	fs.StringVar(&cfg.CustomSSHDCfgFile, "sshd_config", "", "The location of sshd_config")

	err := ff.Parse(fs, args, ff.WithEnvVarPrefix("DROPLET_AGENT"))

	if err != nil {
		panic("failed to parse command line arguments: " + err.Error())
	}

	return &cfg
}
