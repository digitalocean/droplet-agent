// SPDX-License-Identifier: Apache-2.0

package config

import (
	"flag"
	"fmt"
	"math"
	"os"
	"time"

	"github.com/peterbourgon/ff/v3"
)

const (
	// AppFullName is the full name of this application
	AppFullName = "DigitalOcean Droplet Agent (code name: DOTTY)"
	// AppShortName is the short name of this application
	AppShortName = "Droplet Agent"
	// AppDebugAddr is the address where the agent listens for debug connections
	AppDebugAddr          = "127.0.0.1:304"
	backgroundJobInterval = 120 * time.Second
)

var (
	// UserAgent is the user agent string used by the agent
	UserAgent string

	// Version is injected at build-time
	Version string

	// BuildDate is injected at build-time
	BuildDate string
)

func init() {
	if Version == "" {
		Version = "local-dev"
	}

	if BuildDate == "" {
		BuildDate = time.Now().UTC().Format(time.RFC3339)
	}

	UserAgent = "Droplet-Agent/" + Version
}

// Conf contains the configurations needed to run the agent
type Conf struct {
	UseSyslog bool
	DebugMode bool
	UtilMode  bool

	CustomSSHDPort              uint16
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

	showVer := false

	fs := flag.NewFlagSet("droplet-agent", flag.ExitOnError)

	fs.BoolVar(&cfg.UseSyslog, "syslog", false, "Use syslog service for logging")
	fs.BoolVar(&cfg.DebugMode, "debug", false, "Turn on debug mode")
	var port uint
	fs.UintVar(&port, "sshd_port", 0, "The port sshd is binding to")
	fs.StringVar(&cfg.CustomSSHDCfgFile, "sshd_config", "", "The location of sshd_config")

	fs.BoolVar(&showVer, "version", false, "Show version information")

	err := ff.Parse(fs, args, ff.WithEnvVarPrefix("DROPLET_AGENT"))

	if err != nil {
		panic("failed to parse command line arguments: " + err.Error())
	}

	if port <= math.MaxUint16 {
		cfg.CustomSSHDPort = uint16(port)
	} else {
		panic("sshd_port value is out of range")
	}

	if showVer {
		fmt.Printf("Droplet-agent %s (Built: %s)\n", Version, BuildDate)
		os.Exit(0)
	}

	return &cfg
}
