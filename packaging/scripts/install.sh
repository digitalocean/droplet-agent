#!/bin/sh
#   curl -sSL https://repos-droplet.digitalocean.com/install.sh | sudo bash
#   wget -qO- https://repos-droplet.digitalocean.com/install.sh | sudo bash

set -ue

UNSTABLE=${UNSTABLE:-0}
BETA=${BETA:-0}

REPO_HOST="https://repos-droplet.digitalocean.com"
REPO_GPG_KEY=${REPO_HOST}/gpg.key
REPO_GPG_KEY_FPRS=${REPO_HOST}/gpg.key.fpr

repo="droplet-agent"
[ "${UNSTABLE}" != 0 ] && repo="droplet-agent-unstable"
[ "${BETA}" != 0 ] && repo="droplet-agent-beta"

RETRY_CRON_SCHEDULE=/etc/cron.hourly
RETRY_CRON=${RETRY_CRON_SCHEDULE}/droplet-agent-install
cron_install_log=/var/log/droplet_agent.install.cron.log

dist="unknown"
architecture="unknown"
pkg_url="unknown"
tmp_dir="unknown"
exit_status=0
no_retry="false"

main() {
  [ "$(id -u)" != "0" ] &&
    abort "This script must be executed as root."

  trap 'exit_status=$?; script_cleanup; exit $exit_status' EXIT

  check_do
  check_dist
  check_arch

  case "${dist}" in
  debian | ubuntu)
    install_deps "deb"
    find_latest_pkg ${repo} "deb"
    install_pkg "deb"
    ;;
  centos | fedora)
    install_deps "rpm"
    find_latest_pkg ${repo} "rpm"
    install_pkg "rpm"
    ;;
  *)
    not_supported
    ;;
  esac
}

patch_retry_install() {
  [ -f "${RETRY_CRON}" ] && rm -f "${RETRY_CRON}"
  mkdir -p ${RETRY_CRON_SCHEDULE}

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
  if [ ${tmp_dir} != "unknown" ]; then
    echo "Removing temporary files"
    rm -rf ${tmp_dir}
    echo "Done"
  fi
}

find_latest_pkg() {
  repo=${1:-}
  platform=${2:-}
  [ -z "${repo}" ] || [ -z "${platform}" ] && abort "Destination repository is required. Usage: find_latest_pkg <repo> <platform>"
  repo_tree=$(curl -sSL ${REPO_HOST} || wget -qO- ${REPO_HOST})
  files=$(echo "${repo_tree}" | grep -oP '(?<=Key>signed/'"${repo}"'/'"${platform}"'/'"${architecture}"'/)[^<]+' | grep -v '\b.sum$' | tr ' ' '\n')
  sorted_files=$(echo "${files}" | sort -V)
  latest_pkg=$(echo "${sorted_files}" | tail -1)
  echo "latest version: ${latest_pkg}"
  pkg_url="${REPO_HOST}/signed/${repo}/${platform}/${architecture}/${latest_pkg}"
}

install_deps() {
  platform=${1:-}
  [ -z "${platform}" ] && abort "Destination repository is required. Usage: install_deps <platform>"
  echo "Checking dependencies for installing droplet-agent"
  case "${platform}" in
  rpm)
    if ! command -v gpg &>/dev/null; then
      echo "Installing GNUPG"
      yum install -y gpgme
    fi
    ;;
  deb)
    if ! command -v gpg &>/dev/null; then
      echo "Installing GNUPG"
      apt-get -qq update || true
      apt-get install -y gnupg2
    fi
    ;;
  esac

}

ensure_valid_package() {
  file=${1:-}
  [ -z "${file}" ] && abort "signed file must be provided. Usage: ensure_valid_package <signed_file>"
  verifyOutput=$(mktemp gpg_verifyXXXXXX)
  gpg --status-fd 3 --verify "${file}" 3>"${verifyOutput}" || exit 1
  grep -E -q '^\[GNUPG:] TRUST_(ULTIMATE|FULLY)' "${verifyOutput}"
}


install_pkg() {
  platform=${1:-}
  [ -z "${platform}" ] && abort "Destination repository is required. Usage: install_pkg <platform>"

  echo "Importing GPG public key"
  gpg_key=$(wget -qO- "${REPO_GPG_KEY}" || curl -sL "${REPO_GPG_KEY}")
  echo "${gpg_key}" | gpg --import
  gpg_key_fprs=$(wget -qO- "${REPO_GPG_KEY_FPRS}" || curl -sL "${REPO_GPG_KEY_FPRS}")
  for fpr in ${gpg_key_fprs}; do
    echo -e "5\ny\n" | gpg --command-fd 0 --expert --edit-key "$fpr" trust
  done

  tmp_dir=$(mktemp -d -t droplet-agent-XXXXXXXXXX)
  cd "${tmp_dir}"
  echo "Temporary directory: $(pwd)"

  echo "Downloading ${pkg_url}"
  case "${platform}" in
  rpm)
    curl "${pkg_url}" --output ./droplet-agent.rpm.signed
    echo "Verifying package signature..."
    ensure_valid_package droplet-agent.rpm.signed
    echo "OK"
    echo "Extracting package"
    gpg --output droplet-agent.rpm --decrypt droplet-agent.rpm.signed
    rpm -i droplet-agent.rpm
    yum install -y cronie >${cron_install_log} 2>&1 &
    cron_ins_pid=$!
    ;;
  deb)
    wget -O ./droplet-agent.deb.signed "${pkg_url}"
    echo "Verifying package signature..."
    ensure_valid_package droplet-agent.deb.signed
    echo "OK"
    echo "Extracting package"
    gpg --output droplet-agent.deb --decrypt droplet-agent.deb.signed
    dpkg -i droplet-agent.deb
    apt-get install -y cron >${cron_install_log} 2>&1 &
    cron_ins_pid=$!
    ;;
  esac

  echo "Checking crond..."
  wait ${cron_ins_pid}
  stat=$?
  if [ $stat -eq 0 ]; then
    echo "Crond is ready"
  else
    echo "Crond is not ready, please check more detail at ${cron_install_log}"
  fi
}

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
  debian | ubuntu | centos | fedora)
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
  cat <<-EOF

	This script does not support the OS/Distribution on this machine.
	If you feel that this is an error contact support@digitalocean.com

	EOF
  exit 1
}

# abort with an error message
abort() {
  echo "ERROR: $1" >/dev/stderr
  exit 1
}

# leave this last to prevent any partial executions
main
