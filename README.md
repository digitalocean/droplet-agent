# droplet-agent
Droplet Agent is the daemon that runs on customer droplets to enable some features such as web console access.

* [Installing][#installing]
* [Contributing][#contributing]

## Installing

To build the agent, do the following:
1. `cd ./cmd/agent`
2. `GOOS=<target OS> go build -o droplet-agent`
This will generate the `droplet-agent` binary. Upload that binary to your droplet and run:
   `./droplet-agent -debug`

The Droplet Agent should now be running on your droplet.

## Contributing

Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on our code of conduct, and the process for submitting pull requests.

"Droplet Agent" is copyright (c) 2021 DigitalOcean. All rights reserved.