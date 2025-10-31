############
## Macros ##
############
print = @printf ":::::::::::::::: [$(shell date -u)] $@ ::::::::::::::::\n"

docker = @docker run --rm --platform linux/amd64 --pull=always

shellcheck = $(docker) \
	-v "$(CURDIR):$(CURDIR)" \
	-w "$(CURDIR)" \
	-u $(shell id -u) \
	koalaman/shellcheck:latest

linter = $(docker) \
	-v "$(CURDIR):$(CURDIR)" \
	-w "$(CURDIR)" \
	-e "GO111MODULE=on" \
	-e "GOFLAGS=-mod=vendor -buildvcs=false" \
	-e "XDG_CACHE_HOME=$(CURDIR)/target/.cache/go" \
	-u $(shell id -u) golangci/golangci-lint:latest \
	golangci-lint run -D errcheck -E revive -E gosec

go_docker = $(docker) \
	-e "GOCACHE=$(CURDIR)/target/.cache/go" \
	-e "CGO_ENABLED=1" \
	-v "$(CURDIR):$(CURDIR)" \
	-w "$(CURDIR)"

cgo = $(go_docker) golang:latest go

mockgen_container = @echo "FROM golang:latest\nRUN go install go.uber.org/mock/mockgen@latest" | docker build -t droplet-agent-mockgen -

mockgen = $(go_docker) --pull=never droplet-agent-mockgen mockgen

#############
## Targets ##
#############
.PHONY: test
test: lint
	$(print)
	$(cgo) test -cover -race ./...

.PHONY: shellcheck
shellcheck:
	$(print)
	$(shellcheck) --version
	$(shellcheck) $(shell find . -type f -iname '*.sh' ! -path './vendor/*' ! -path './.git/*')

.PHONY: lint
lint: shellcheck
	$(print)
	$(linter) ./...

.PHONY: mockgen
mockgen:
	$(print)
	$(mockgen_container)
	$(mockgen) -package=mock_os -destination=internal/sysutil/internal/mocks/os_mocks.go os FileInfo
	$(mockgen) -source=internal/sysutil/common.go -package=sysutil -destination=internal/sysutil/common_mocks.go
	$(mockgen) -source=internal/sysutil/os_operations_helper.go -package=sysutil -destination=internal/sysutil/os_operations_helper_mocks.go
	$(mockgen) -source=internal/sysaccess/common.go -package=mocks -destination=internal/sysaccess/internal/mocks/mocks.go
	$(mockgen) -source=internal/sysaccess/ssh_helper.go -package=sysaccess -destination=internal/sysaccess/ssh_helper_mocks.go
	$(mockgen) -source=internal/sysaccess/authorized_keys_file_updater.go -package=sysaccess -destination=internal/sysaccess/authorized_keys_file_updater_mocks.go
	$(mockgen) -source=internal/metadata/actioner/do_managed_keys_actioner.go -package=mocks -destination=internal/metadata/actioner/internal/mocks/mocks.go
	$(mockgen) -source=internal/netutil/tcp_sniffer_helper_linux.go -package=mocks -destination=internal/netutil/internal/mocks/dependent_functions_mock.go
	$(mockgen) -source=internal/metadata/updater/updater.go -package=updater -destination=internal/metadata/updater/updater_mocks.go
	$(mockgen) -destination=internal/metadata/updater/readcloser_mocks.go -package=updater -build_flags=--mod=mod io ReadCloser
	$(mockgen) -source=internal/sysutil/usermanager.go -package=sysutil -destination=internal/sysutil/usermanager_mocks.go
	$(mockgen) -source=internal/troubleshooting/file/file.go -package=mocks -destination=internal/troubleshooting/mocks/file_mocks.go
	$(mockgen) -source=internal/troubleshooting/command/command.go -package=mocks -destination=internal/troubleshooting/mocks/command_mocks.go
	$(mockgen) -source=internal/troubleshooting/command/exec.go -package=mocks -destination=internal/troubleshooting/mocks/exec_mocks.go
	$(mockgen) -source=internal/troubleshooting/parser/parser.go -package=mocks -destination=internal/troubleshooting/mocks/parser_mocks.go
	$(mockgen) -source=internal/troubleshooting/otlp/client.go -package=mocks -destination=internal/troubleshooting/mocks/otlp_client_mocks.go
