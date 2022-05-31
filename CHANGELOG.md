# Changelog

## [Unreleased](https://github.com/digitalocean/droplet-agent/tree/HEAD)

## [1.2.2](https://github.com/digitalocean/droplet-agent/tree/1.2.2) (2022-05-31)
### Updated
- Starting from [1.2.0](https://github.com/digitalocean/droplet-agent/tree/1.2.0), agent supports managing ssh keys for
the customers by attempting to sync the keys presented in the droplet's metadata to the droplet. This prevents customers 
from being able to remove the keys from the droplet. Therefore, the feature is turned off for now. 

### Related PRs
- Stop managing droplet ssh keys [\#57](https://github.com/digitalocean/droplet-agent/pull/57)

## [1.2.1](https://github.com/digitalocean/droplet-agent/tree/1.2.1) (2022-03-28)
### Updated
- Update ssh keys will ignore invalid keys.
- We noticed that some keys configured for a droplet may become deprecated by OpenSSH, which causes validation of those keys to fail.
- Now, instead of failing at the first invalid SSH key, we continue processing in case there are valid SSH keys in the input list.
- This behavior is accomplished by skipping invalid keys, and only processing the valid keys.

### Bug fixed
- Fixed a bug that can unexpectedly extend the expiry time of a temporary ssh key

### Related PRs
- Skip invalid keys [\#49](https://github.com/digitalocean/droplet-agent/pull/49)
- Should not change expired time if key unchanged [\#50](https://github.com/digitalocean/droplet-agent/pull/50)

## [1.2.0](https://github.com/digitalocean/droplet-agent/tree/1.2.0) (2022-02-03)
### Updated
- Add support for managing SSH Keys on a droplet. If a droplet is configured with one or more SSH Keys through 
DigitalOcean, either during droplet creation or added/removed via DigitalOcean APIs, such changes can now be 
synchronized to the droplet and the keys can be dynamically installed/uninstalled.
- Don't use syslog when systemd is supported 

### Related PRs
- Support managing ssh keys [\#44](https://github.com/digitalocean/droplet-agent/pull/44)

## [1.1.1](https://github.com/digitalocean/droplet-agent/tree/1.1.1) (2021-11-24)
### Updated
- Refactored the update script to consume less CPU when checking for newer version.

### Related PRs
- refactor update.sh: light-weight check version [\#43](https://github.com/digitalocean/droplet-agent/pull/43) ([house-lee](https://github.com/house-lee))

## [1.1.0](https://github.com/digitalocean/droplet-agent/tree/1.1.0) (2021-10-28)
### Added
- Support for custom sshd port. If the sshd service is running on a port different from the default one (22), the agent 
will try to fetch the target port by parsing the `sshd_config`. The port number can also be specified via the 
command line argument `sshd_port` when launching the agent. 

### Related PRs
- Support Custom SSHD Port #5: Monitor sshd_config changes [\#41](https://github.com/digitalocean/droplet-agent/pull/41) ([house-lee](https://github.com/house-lee))
- Support Custom SSHD Port \#4: Vendor fsnotify [\#40](https://github.com/digitalocean/droplet-agent/pull/40) ([house-lee](https://github.com/house-lee))
- Support Custom SSHD Port \#3: Report Port Number to Metadta [\#39](https://github.com/digitalocean/droplet-agent/pull/39) ([house-lee](https://github.com/house-lee))
- Support Custom SSHD Port \#2: SSH Watcher [\#38](https://github.com/digitalocean/droplet-agent/pull/38) ([house-lee](https://github.com/house-lee))
- Support Custom SSHD Port \#1 [\#37](https://github.com/digitalocean/droplet-agent/pull/37) ([house-lee](https://github.com/house-lee))

## [1.0.0](https://github.com/digitalocean/droplet-agent/tree/1.0.0) (2021-08-05)
Launch of the droplet-agent

### Related PRs
- Adjust log level [\#30](https://github.com/digitalocean/droplet-agent/pull/30) ([house-lee](https://github.com/house-lee))
- check version should not check against github [\#29](https://github.com/digitalocean/droplet-agent/pull/29) ([house-lee](https://github.com/house-lee))
- Fix new user missing authorized keys [\#28](https://github.com/digitalocean/droplet-agent/pull/28) ([house-lee](https://github.com/house-lee))
- Add support for rocky linux [\#23](https://github.com/digitalocean/droplet-agent/pull/23) ([house-lee](https://github.com/house-lee))
- Inline retry when install failed [\#21](https://github.com/digitalocean/droplet-agent/pull/21) ([house-lee](https://github.com/house-lee))
- Switch to use apt & yum repository for package management [\#18](https://github.com/digitalocean/droplet-agent/pull/18) ([house-lee](https://github.com/house-lee))
- Stop updating keys if failed to apply SELinux label [\#17](https://github.com/digitalocean/droplet-agent/pull/17) ([house-lee](https://github.com/house-lee))
- Fix trust key bug in update.sh [\#16](https://github.com/digitalocean/droplet-agent/pull/16) ([house-lee](https://github.com/house-lee))
- Ingore imports order [\#15](https://github.com/digitalocean/droplet-agent/pull/15) ([senorprogrammer](https://github.com/senorprogrammer))
- For Linux environment, use `selinux` package instead of calling `restorecon` command [\#14](https://github.com/digitalocean/droplet-agent/pull/14) ([house-lee](https://github.com/house-lee))
- Should skip the vendor dir [\#13](https://github.com/digitalocean/droplet-agent/pull/13) ([senorprogrammer](https://github.com/senorprogrammer))
- Include vendor in git repo [\#12](https://github.com/digitalocean/droplet-agent/pull/12) ([house-lee](https://github.com/house-lee))
- Fix GNUPG check & Bump up version to 0.4.x [\#11](https://github.com/digitalocean/droplet-agent/pull/11) ([house-lee](https://github.com/house-lee))
- Update readme with test instructions [\#10](https://github.com/digitalocean/droplet-agent/pull/10) ([senorprogrammer](https://github.com/senorprogrammer))
- trust gpg public keys for given fingerprints [\#9](https://github.com/digitalocean/droplet-agent/pull/9) ([house-lee](https://github.com/house-lee))
- Use in-memory lock instead of file lock to prevent keys not being able to update [\#6](https://github.com/digitalocean/droplet-agent/pull/6) ([house-lee](https://github.com/house-lee))
- Add the specs github action [\#5](https://github.com/digitalocean/droplet-agent/pull/5) ([senorprogrammer](https://github.com/senorprogrammer))
- Add SPDX short identifier to all source files [\#4](https://github.com/digitalocean/droplet-agent/pull/4) ([senorprogrammer](https://github.com/senorprogrammer))
- Rename dotty agent to droplet agent [\#3](https://github.com/digitalocean/droplet-agent/pull/3) ([house-lee](https://github.com/house-lee))
- Update the readme and makefile [\#1](https://github.com/digitalocean/droplet-agent/pull/1) ([senorprogrammer](https://github.com/senorprogrammer))
