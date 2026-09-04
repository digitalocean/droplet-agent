module github.com/digitalocean/droplet-agent

go 1.25.0

require (
	github.com/fsnotify/fsnotify v1.10.1
	github.com/opencontainers/selinux v1.15.1
	github.com/peterbourgon/ff/v3 v3.4.0
	go.uber.org/mock v0.6.0
	golang.org/x/crypto v0.55.0
	golang.org/x/net v0.58.0
	golang.org/x/sync v0.22.0
	golang.org/x/sys v0.47.0
	golang.org/x/time v0.15.0
)

require (
	cyphar.com/go-pathrs v0.2.1 // indirect
	github.com/cyphar/filepath-securejoin v0.6.0 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	golang.org/x/mod v0.38.0 // indirect
	golang.org/x/tools v0.48.0 // indirect
)

tool go.uber.org/mock/mockgen
