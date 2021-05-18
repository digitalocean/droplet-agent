<pre>
    ____                   __     __     ___                    __ 
   / __ \_________  ____  / /__  / /_   /   | ____ ____  ____  / /_
  / / / / ___/ __ \/ __ \/ / _ \/ __/  / /| |/ __ `/ _ \/ __ \/ __/
 / /_/ / /  / /_/ / /_/ / /  __/ /_   / ___ / /_/ /  __/ / / / /_  
/_____/_/   \____/ .___/_/\___/\__/  /_/  |_\__, /\___/_/ /_/\__/  
                /_/                        /____/                  
</pre>

Droplet Agent is the daemon that runs on DigitalOcean's customer droplets to enable some features such as web console access.

* [Installing](#installing)
* [Running Tests](#running-tests)
* [Contributing](#contributing)

## Installing

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
