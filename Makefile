#################
## Env Configs ##
#################
GOOS   ?= linux
GOARCH ?= amd64
CTHULHU_DIR ?= null
SKIP_VERSION_CHECK ?= 0
VERSION     ?= $(shell ./scripts/get_version.sh)

ifeq ($(GOARCH),386)
PKG_ARCH = i386
else
PKG_ARCH = amd64
endif

############
## Macros ##
############
mkdir        = @mkdir -p $(dir $@)
touch        = @touch $@
cp           = @cp $< $@
print        = @printf "\n:::::::::::::::: [$(shell date -u)] $@ ::::::::::::::::\n"
now          = $(shell date -u)
fpm          = @docker run --platform linux/amd64 --rm -i -v "$(CURDIR):$(CURDIR)" -w "$(CURDIR)" -u $(shell id -u) digitalocean/fpm:latest
shellcheck   = @docker run --platform linux/amd64 --rm -i -v "$(CURDIR):$(CURDIR)" -w "$(CURDIR)" -u $(shell id -u) koalaman/shellcheck:v0.6.0
version_check = @./scripts/check_version.sh
linter = docker run --platform linux/amd64 --rm -i -v "$(CURDIR):$(CURDIR)" -w "$(CURDIR)" -e "GO111MODULE=on" -e "GOFLAGS=-mod=vendor" -e "XDG_CACHE_HOME=$(CURDIR)/target/.cache/go" \
	-u $(shell id -u) golangci/golangci-lint:v1.39 \
	golangci-lint run --skip-files=.*_test.go -D errcheck -E golint -E gosec -E gofmt

go_docker_linux = golang:1.18.2
ifeq ($(GOOS), linux)
go = docker run --platform linux/amd64 --rm -i \
	-e "GOOS=$(GOOS)" \
	-e "GOARCH=$(GOARCH)" \
	-e "GOCACHE=$(CURDIR)/target/.cache/go" \
	-v "$(CURDIR):$(CURDIR)" \
	-w "$(CURDIR)" \
	$(go_docker_linux) \
	go
else
go = GOOS=$(GOOS) \
     GOARCH=$(GOARCH) \
     GO111MODULE=on \
     GOFLAGS=-mod=vendor \
     GOCACHE=$(CURDIR)/target/.cache/go \
     $(shell which go)
endif

ldflags = '\
	-s -w \
	-X "main.version=$(VERSION)" \
	-X "main.buildDate=$(now)" \
'

SYSINIT_CONF="packaging/syscfg/init/droplet-agent.conf"
SYSTEMD_CONF="packaging/syscfg/systemd/droplet-agent.service"
UPDATE_SCRIPT="packaging/scripts/update.sh"

###########
## Paths ##
###########
out                := target
package_dir        := $(out)/pkg
cache              := $(out)/.cache
project            := droplet-agent
pkg_project        := $(subst _,-,$(project))# make sure package does not have underscores in the name
gofiles            := $(shell find . -type f -iname '*.go' ! -path './vendor/*')
shellscripts       := $(shell find . -type f -iname '*.sh' ! -path './vendor/*' ! -path './.git/*')
binary             := $(out)/$(project)-$(GOOS)-$(GOARCH) # the name of the binary built with local resources
base_linux_package := $(package_dir)/$(pkg_project).$(VERSION).$(PKG_ARCH).BASE.deb
# note: to be compliant with repository's naming requirement:
# deb files should end with _version_arch.deb
# rpm files should end with -version-release.arch.rpm
deb_package        := $(package_dir)/$(pkg_project)_$(VERSION)_$(PKG_ARCH).deb
rpm_package        := $(package_dir)/$(pkg_project).$(VERSION).$(PKG_ARCH).rpm
tar_package        := $(package_dir)/$(pkg_project).$(VERSION).$(PKG_ARCH).tar.gz

#############
## Targets ##
#############
.PHONY: test
test:
ifndef pkg
	$(linter) ./...
	$(go) test -cover -race ./...
else
	$(linter) $(pkg)
	$(go) test -cover -race -v ./$(pkg)
endif

.PHONY: internal/config/version.go
internal/config/version.go:
	$(print)
	@printf "version = v%s" $(VERSION)
	@printf "// SPDX-License-Identifier: Apache-2.0\n\npackage config\n\nconst version = \"v%s\"\n" $(VERSION) > $@

.PHONY: target/VERSION internal/config/version.go
target/VERSION:
	$(print)
	$(mkdir)
ifneq ($(SKIP_VERSION_CHECK), 1)
	$(version_check)
else
	@echo "========Skipping Version Check======"
endif
	@echo $(VERSION) > $@

.INTERMEDIATE: $(binary)
build: $(binary)
$(binary): $(gofiles)
	$(print)
	$(mkdir)
	$(go) build -ldflags $(ldflags) -trimpath -o "$@" ./cmd/agent/

shellcheck: $(cache)/shellcheck
$(cache)/shellcheck: $(shellscripts)
	$(print)
	$(mkdir)
	@$(shellcheck) --version
	@$(shellcheck) $^
	$(touch)

.PHONY: clean
clean:
	$(print)
	@rm -rf $(out)

release: target/VERSION
	$(print)
	@GOOS=linux GOARCH=386 $(MAKE) build deb rpm tar
	@GOOS=linux GOARCH=amd64 $(MAKE) build deb rpm tar

lint: $(cache)/lint $(cache)/shellcheck
$(cache)/lint: $(gofiles)
	$(print)
	$(mkdir)
	@$(linter) ./...
	$(touch)

.INTERMEDIATE: $(base_linux_package)
$(base_linux_package): $(binary)
	$(print)
	$(mkdir)
	@$(fpm) --output-type deb \
		--verbose \
		--input-type dir \
		--force \
		--architecture $(PKG_ARCH) \
		--package $@ \
		--no-depends \
		--name $(pkg_project) \
		--maintainer "DigitalOcean Droplet Engineering" \
		--version $(VERSION) \
		--url https://github.com/digitalocean/droplet-agent \
		--description "DigitalOcean Droplet Agent" \
		--license apache-2.0 \
		--vendor DigitalOcean \
		--log info \
		--after-install packaging/scripts/after_install.sh \
		--after-remove packaging/scripts/after_remove.sh \
		--config-files /etc/init/droplet-agent.conf \
		--config-files /etc/systemd/system/droplet-agent.service \
		$<=/opt/digitalocean/bin/droplet-agent \
		$(UPDATE_SCRIPT)=/opt/digitalocean/droplet-agent/scripts/update.sh \
		$(SYSINIT_CONF)=/etc/init/droplet-agent.conf \
		$(SYSTEMD_CONF)=/etc/systemd/system/droplet-agent.service

deb: $(deb_package)
$(deb_package): $(base_linux_package)
	$(print)
	$(mkdir)
	@$(fpm) --output-type deb \
		--verbose \
		--input-type deb \
		--force \
		--depends cron \
		-p $@ \
		$<
	# print information about the compiled deb package
	@docker run --platform linux/amd64 --rm -i -v "$(CURDIR):$(CURDIR)" -w "$(CURDIR)" ubuntu:xenial /bin/bash -c 'dpkg --info $@ && dpkg -c $@'

rpm: $(rpm_package)
$(rpm_package): $(base_linux_package)
	$(print)
	$(mkdir)
	@$(fpm) \
		--verbose \
		--output-type rpm \
		--input-type deb \
		--depends cronie \
		--rpm-posttrans packaging/scripts/after_install.sh \
		--force \
		-p $@ \
		$<
	# print information about the compiled rpm package
	@docker run --platform linux/amd64 --rm -i -v "$(CURDIR):$(CURDIR)" -w "$(CURDIR)" centos:7 rpm -qilp $@

tar: $(tar_package)
$(tar_package): $(base_linux_package)
	$(print)
	$(mkdir)
	@$(fpm) \
		--verbose \
		--output-type tar \
		--input-type deb \
		--force \
		-p $@ \
		$<
	# print all files within the archive
	@docker run --platform linux/amd64 --rm -i -v "$(CURDIR):$(CURDIR)" -w "$(CURDIR)" ubuntu:xenial tar -ztvf $@

## mockgen: generates the mocks for the droplet agent service
mockgen:
	@echo "Generating mocks"
	mockgen -source=internal/sysaccess/common.go -package=mocks -destination=internal/sysaccess/internal/mocks/mocks.go
	mockgen -source=internal/sysaccess/ssh_helper.go -package=sysaccess -destination=internal/sysaccess/ssh_helper_mocks.go
	mockgen -source=internal/sysaccess/authorized_keys_file_updater.go -package=sysaccess -destination=internal/sysaccess/authorized_keys_file_updater_mocks.go
	mockgen -source=internal/metadata/actioner/do_managed_keys_actioner.go -package=mocks -destination=internal/metadata/actioner/internal/mocks/mocks.go
	GOOS=linux mockgen -source=internal/netutil/tcp_sniffer_helper_linux.go -package=mocks -destination=internal/netutil/internal/mocks/dependent_functions_mock.go
	mockgen -source=internal/metadata/updater/updater.go -package=updater -destination=internal/metadata/updater/updater_mocks.go
