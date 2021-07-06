#!/bin/sh
#   curl -sSL https://repos-droplet.digitalocean.com/install.sh | sudo bash
#   wget -qO- https://repos-droplet.digitalocean.com/install.sh | sudo bash
#
# To use the BETA branch of droplet-agent pass the BETA=1 flag to the script
#   curl -sSL https://repos-droplet.digitalocean.com/install.sh | sudo BETA=1 bash
#
# To use the UNSTABLE branch of droplet-agent pass the UNSTABLE=1 flag to the script
#   curl -sSL https://repos-droplet.digitalocean.com/install.sh | sudo UNSTABLE=1 bash

set -u

UNSTABLE=${UNSTABLE:-0}
BETA=${BETA:-0}

REPO_DOMAIN="repos-droplet.digitalocean.com"
REPO_HOST="https://${REPO_DOMAIN}"
REPO_GPG_KEY=${REPO_HOST}/gpg.key

branch="droplet-agent"
[ "${UNSTABLE}" != 0 ] && branch="droplet-agent-unstable"
[ "${BETA}" != 0 ] && branch="droplet-agent-beta"

RETRY_CRON_SCHEDULE=/etc/cron.hourly
RETRY_CRON=${RETRY_CRON_SCHEDULE}/droplet-agent-install

dist="unknown"
architecture="unknown"
exit_status=0
no_retry="false"
repo_name=droplet-agent
deb_list=/etc/apt/sources.list.d/${repo_name}.list
deb_pref=/etc/apt/preferences.d/${repo_name}.pref
deb_keyfile=/usr/share/keyrings/${repo_name}-keyring.gpg
rpm_repo=/etc/yum.repos.d/${repo_name}.repo

main() {
  [ "$(id -u)" != "0" ] &&
    abort "This script must be executed as root."

  trap 'script_cleanup; exit $exit_status' EXIT

  check_do
  check_dist
  check_arch

  case "${dist}" in
  debian | ubuntu)
    i=1
    until [ "$i" -ge 6 ]; do
      echo "Installing Droplet Agent, ${i} attempt"
      install_apt
      exit_status=$?
      if [ ${exit_status} -eq 0 ]; then
        break
      fi
      i=$((i+1))
      sleep 60
    done
    ;;
  centos | fedora | rocky)
    i=1
    until [ "$i" -ge 6 ]; do
      echo "Installing Droplet Agent, ${i} attempt"
      install_yum
      exit_status=$?
      if [ ${exit_status} -eq 0 ]; then
        break
      fi
      i=$((i+1))
      sleep 60
    done
    ;;
  *)
    not_supported
    ;;
  esac
}

patch_retry_install() {
  [ -f "${RETRY_CRON}" ] && rm -f "${RETRY_CRON}"
  mkdir -p ${RETRY_CRON_SCHEDULE}
  if ! command -v crontab >/dev/null 2>&1; then
    echo "cron not installed, installing"
    if command -v apt-get >/dev/null 2>&1; then
      apt-get -qq install -y cron
    elif command -v yum >/dev/null 2>&1; then
      yum install -y cronie
    else
      echo "not supported os"
      return 1
    fi
  fi

  cat <<'EOF' >"${RETRY_CRON}"
#!/bin/sh
tmp_file=$(mktemp -t droplet_agent.install.XXXXXX)
trap "rm -f ${tmp_file}" EXIT
url="https://repos-droplet.digitalocean.com/install.sh"
install_script=$(curl -sSL "${url}" || wget -qO- "${url}")
echo "${install_script}" > ${tmp_file}
now=$(date +"%T")
echo "Retry at: ${now}" > /var/log/droplet_agent.install.log
/bin/bash ${tmp_file} >> /var/log/droplet_agent.install.log 2>&1
EOF

  chmod +x "${RETRY_CRON}"
}

remove_retry_install() {
  rm -f "${RETRY_CRON}"
}

script_cleanup() {
  if [ ${exit_status} -ne 0 ]; then
    if [ ${no_retry} = "false" ]; then
      echo "Install failed, will retry again later"
      patch_retry_install
    else
      echo "Install failed, and will not be retried"
    fi
  else
    echo "DigitalOcean Droplet Agent is successfully installed"
    remove_retry_install || true
  fi
}

install_deps() {
  platform=${1:-}
  [ -z "${platform}" ] && abort "Destination repository is required. Usage: install_deps <platform>"
  echo "Checking dependencies for installing droplet-agent"
  case "${platform}" in
  rpm)
    yum install -y gpgme ca-certificates
    ;;
  deb)
    if ! command -v gpg >/dev/null 2>&1; then
      echo "Installing GNUPG"
      apt-get -qq update || true
      apt-get install -y gnupg2
    fi
    if ! apt-get -qq install -y ca-certificates apt-utils apt-transport-https; then
      apt-get -qq update
      apt-get -qq install -y ca-certificates apt-utils apt-transport-https
    fi
    ;;
  esac

}

install_apt() (
  set -e
  export DEBIAN_FRONTEND=noninteractive
  # forcefully remove any existing installations
  apt-get purge -y droplet-agent >/dev/null 2>&1 || :

  echo "Setting up droplet-agent apt repository..."
  install_deps "deb"
  echo "Importing GPG public key"
  wget -qO- "${REPO_GPG_KEY}" | gpg --dearmor >"${deb_keyfile}"
  echo "deb [signed-by=${deb_keyfile}] ${REPO_HOST}/apt/${branch} main main" >"${deb_list}"
  cat <<-EOF >${deb_pref}
	Package: *
	Pin: origin ${REPO_DOMAIN}
	Pin-Priority: 100
	EOF

  echo "Installing droplet-agent"
  apt-get -qq update -o Dir::Etc::SourceParts=/dev/null -o APT::Get::List-Cleanup=no -o Dir::Etc::SourceList="sources.list.d/droplet-agent.list"
  apt-get -qq install -y droplet-agent droplet-agent-keyring
)

install_yum() (
  set -e
  # forcefully remove any existing installations
  yum remove -y droplet-agent || :

  echo "Setting up droplet-agent yum repository..."
  install_deps "rpm"
  cat <<-EOF >${rpm_repo}
	[${repo_name}]
	name=DigitalOcean Droplet Agent
	baseurl=${REPO_HOST}/yum/${branch}/\$basearch
	repo_gpgcheck=0
	gpgcheck=1
	enabled=1
	gpgkey=${REPO_GPG_KEY}
	sslverify=0
	sslcacert=/etc/pki/tls/certs/ca-bundle.crt
	metadata_expire=300
	EOF

  yum --disablerepo="*" --enablerepo="${repo_name}" makecache
  yum install -y droplet-agent
  # to ensure crond service is started
  systemctl start crond.service || true
)

check_dist() {
  echo "Verifying compatibility with script..."
  if [ -f /etc/os-release ]; then
    dist=$(awk -F= '$1 == "ID" {gsub("\"", ""); print$2}' /etc/os-release)
  elif [ -f /etc/redhat-release ]; then
    dist=$(awk '{print tolower($1)}' /etc/redhat-release)
  else
    not_supported
  fi

  dist=$(echo "${dist}" | tr '[:upper:]' '[:lower:]')

  case "${dist}" in
  debian | ubuntu | centos | fedora | rocky)
    echo "OK"
    ;;
  *)
    not_supported
    ;;
  esac
}

check_arch() {
  echo "Checking architecture support..."
  case $(uname -m) in
  i386 | i686)
    architecture="i386"
    ;;
  x86_64)
    architecture="x86_64"
    ;;
  *) not_supported ;;
  esac
  echo "OK"
}

check_do() {
  echo "Verifying machine compatibility..."
  # DigitalOcean embedded platform information in the DMI data.
  dmi_bios_file="/sys/devices/virtual/dmi/id/bios_vendor"
  if [ -f "${dmi_bios_file}" ]; then
    read -r sys_vendor <${dmi_bios_file}
  else
    sys_vendor=$(dmidecode -s bios-vendor)
  fi
  if ! [ "$sys_vendor" = "DigitalOcean" ]; then
    cat <<-EOF

		The DigitalOcean Droplet Agent is only supported on DigitalOcean machines.

		If you are seeing this message on an older droplet, you may need to power-off
		and then power-on at http://cloud.digitalocean.com. After power-cycling,
		please re-run this script.

		EOF
    exit 1
  fi
  echo "OK"
}

not_supported() {
  no_retry="true"
  exit_status=1
  cat <<-EOF

	This script does not support the OS/Distribution on this machine.
	If you feel that this is an error contact support@digitalocean.com

	EOF
  exit ${exit_status}
}

# abort with an error message
abort() {
  echo "ERROR: $1" >/dev/stderr
  exit 1
}

# leave this last to prevent any partial executions
main
