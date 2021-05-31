package main

import (
	"os"
	"os/exec"

	"github.com/digitalocean/droplet-agent/internal/log"
)

const (
	switchScriptPath = "/opt/digitalocean/droplet-agent/scripts/switch.sh"
)

func main() {

	log.Info("Reinstalling Droplet Agent via APT/YUM repository")
	cmd := exec.Command("/bin/bash", switchScriptPath)
	_ = cmd.Start()
	log.Info("Current Droplet-Agent is exiting. Bye-bye!")
	// exits with code 0
	os.Exit(0)
}
