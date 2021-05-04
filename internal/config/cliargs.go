package config

import "flag"

type cliArgs struct {
	debugMode bool
	useSyslog bool
}

func parseCLIArgs() *cliArgs {
	ret := &cliArgs{}

	flag.BoolVar(&ret.debugMode, "debug", false, "Turn on debug mode")
	flag.BoolVar(&ret.useSyslog, "syslog", false, "Use syslog service for logging")

	flag.Parse()

	return ret
}
