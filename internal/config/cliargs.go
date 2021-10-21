// SPDX-License-Identifier: Apache-2.0

package config

import "flag"

type cliArgs struct {
	debugMode   bool
	useSyslog   bool
	sshdPort    int
	sshdCfgFile string
}

func parseCLIArgs() *cliArgs {
	ret := &cliArgs{}

	flag.BoolVar(&ret.debugMode, "debug", false, "Turn on debug mode")
	flag.BoolVar(&ret.useSyslog, "syslog", false, "Use syslog service for logging")
	flag.IntVar(&ret.sshdPort, "sshd_port", 0, "The port sshd is binding to")
	flag.StringVar(&ret.sshdCfgFile, "sshd_config", "", "The location of sshd_config")

	flag.Parse()

	return ret
}
