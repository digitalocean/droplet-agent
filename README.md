<pre>
    ____                   __     __     ___                    __ 
   / __ \_________  ____  / /__  / /_   /   | ____ ____  ____  / /_
  / / / / ___/ __ \/ __ \/ / _ \/ __/  / /| |/ __ `/ _ \/ __ \/ __/
 / /_/ / /  / /_/ / /_/ / /  __/ /_   / ___ / /_/ /  __/ / / / /_  
/_____/_/   \____/ .___/_/\___/\__/  /_/  |_\__, /\___/_/ /_/\__/  
                /_/                        /____/                  
</pre>

Droplet Agent is the daemon that runs on DigitalOcean's customer droplets to enable some features such as web console access.

* [Building](#building)
  * [Building From Source Code](#building-from-source-code) 
  * [Packaging](#building-from-source-code) 
* [Running the Agent](#running-the-agent)
* [Running Tests](#running-tests)
* [Contributing](#contributing)

## Building 

### Building From Source Code
Clone this repository:

```bash
> git clone git@github.com:digitalocean/droplet-agent.git
> cd droplet-agent
```

To build the agent, do the following:

1. `cd ./cmd/agent`
2. `GOOS=<target OS> go build -o droplet-agent`

This will generate the `droplet-agent` binary. 

Upload that binary to your droplet and run:

`./droplet-agent -debug`

The Droplet Agent should now be running on your droplet.

### Packaging
We now support building `deb` and `rpm` packages. You are welcome to submit
PRs for supporting other package management systems.

To build a package, assumed the repo is already cloned, go to the repo directory and run:
`GOOS=<target OS> GOARCH=<go arch> make build <target package>`

NOTES:
1. As of now, the only supported <target OS> is Linux
2. Supported GOARCH are `amd64` and `386`
3. Supported <target package> are `deb`, `rpm` and/or `tar`
4. Multiple packages can be built at the same time by specifying the <target package> list in space separated format.  
For example, `GOOS=linux GOARCH=amd64 make build deb rpm tar` will generate `deb`, `rpm`, and `tar` packages
5. `systemd` is the preferred way for managing the droplet-agent service. Although `initctl` is also supported, it may 
not support all features provided by the droplet-agent, and should only be used on older system that does not have 
`systemd` support.
6. `systemd` configuration of the agent service is saved at `etc/systemd/system/droplet-agent.service`, once updated, 
please remember to apply the changes by running `systemctl daemon-reload`
7. Configuration for `initctl` is saved at `/etc/init/droplet-agent.conf`. If updated, please run 
`initctl reload-configuration` to apply the updated configuration.


## Running the Agent
The agent binary takes several command line arguments:
- `-debug` (boolean), if provided, the agent will run in debug mode with verbose logging. This is useful when debugging.
- `-syslog` (boolean), specify how the log is handled. By default, all logs will be sent to `stdout` and `stderr`, if 
`syslog` option is provided, logs will be sent to `syslogd`. When logging to `syslog`, the agent will use `DropletAgent`
as the identifier. To retrieve the logs, simply run `journalctl -t DropletAgent` command.
- `-sshd_port <port>`(integer), explicitly indicates which port sshd binds itself to, so that the agent can properly 
monitor the port knocking messages, as well as enabling the web console proxy to connect to the sshd instance. Without
specifying this option, the agent will try parse `sshd_config` to see if custom port is specified by checking the `Port`
and `ListenAddress` entries, if not, it falls to use the default port (22).
- `-sshd_config <path to sshd_config>` (string), explicitly specify the path to the `sshd_config` file. In the cases   
that the sshd is started with a custom `sshd_config` file other than the default one (/etc/ssh/sshd_config), this 
parameter must be supplied to let the agent function properly

NOTES:
- Be aware that `sshd_port` number has higher priority. The agent will skip attempting to parse the port from 
`sshd_config` if `sshd_port` is supplied. 
- When parsing the `sshd_config`, the agent will take the first occurrence of port number from either `Port` or 
`ListenAddress` entries. If the sshd is configured to bind to multiple interfaces and/or multiple ports, please sepcify
the port number that is exposed externally via `sshd_port` option.

## Running Tests

First, ensure that [Docker](https://www.docker.com) is installed and running.

Then, inside the droplet-agent project directory:

```bash
> go mod vendor
> make test
```

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests.

"Droplet Agent" is copyright (c) 2021 DigitalOcean. All rights reserved.
