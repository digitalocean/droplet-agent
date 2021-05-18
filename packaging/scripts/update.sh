#!/bin/bash
# vim: noexpandtab
set -ueo pipefail

UNSTABLE=${UNSTABLE:-0}
BETA=${BETA:-0}

REPO_HOST="https://repos-droplet.digitalocean.com"
REPO_GPG_KEY=${REPO_HOST}/gpg.key
REPO_GPG_OWNERTRUST=${REPO_HOST}/gpg.ownertrust
repo="droplet-agent" # update.sh will always run off the stable branch by default
architecture="unknown"
pkg_url="unknown"
tmp_dir="unknown"
remote_ver="unknown"
exit_status=0

[ "${UNSTABLE}" != 0 ] && repo="droplet-agent-unstable"
[ "${BETA}" != 0 ] && repo="droplet-agent-beta"

find_latest_pkg() {
  repo=${1:-}
  platform=${2:-}
  [ -z "${repo}" ] || [ -z "${platform}" ] && abort "Destination repository is required. Usage: find_latest_pkg <repo> <platform>"
  repo_tree=$(curl -sSL ${REPO_HOST} || wget -qO- ${REPO_HOST})
  files=$(echo "${repo_tree}" | grep -oP '(?<=Key>signed/'"${repo}"'/'"${platform}"'/'"${architecture}"'/)[^<]+' | grep -v '\b.sum$' | tr ' ' '\n')
  sorted_files=$(echo "${files}" | sort -V)
  latest_pkg=$(echo "${sorted_files}" | tail -1)
  remote_ver=$(echo "${latest_pkg}" | grep -oP '(?<=droplet-agent.)\d.\d.\d')
  echo "latest version: ${remote_ver}"
  pkg_url="${REPO_HOST}/signed/${repo}/${platform}/${architecture}/${latest_pkg}"
}

check_arch() {
  echo -n "Checking architecture support..."
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

update_rpm() {
  # update rpm
  find_latest_pkg "${repo}" "rpm"
  #get installed version
  local_ver=$(rpm -q droplet-agent --qf '%{VERSION}')
  printf "local version:%s\nremote version:%s\n" "${local_ver}" "${remote_ver}"
  if [ "${local_ver}" = "${remote_ver}" ]; then
    echo "No need to update"
    exit 0
  fi

  sorted_vers=$(printf "%s\n%s" "${local_ver}" "${remote_ver}" | sort -V)
  newer_ver=$(echo "${sorted_vers}" | tail -1)

  if [ "${newer_ver}" = "${remote_ver}" ]; then
    echo "Upgrading droplet-agent to ver:${remote_ver}"

    if ! command -v gpg &>/dev/null; then
      echo "Installing GNUPG"
      yum install -y gpgme
    fi

    echo "Ensuring gpg key is ready..."
    curl -sL "${REPO_GPG_KEY}" | gpg --import
    echo "Ensuring gpg key is trusted..."
    gpg_key_ownertrust=$(curl -sL "${REPO_GPG_OWNERTRUST}")
    for item in ${gpg_key_ownertrust}; do
      fpr=$(echo "$item" | cut -d ':' -f 1)
      echo "trusting ${fpr}"
      echo -e "5\ny\n" | gpg --command-fd 0 --expert --edit-key "$fpr" trust
    done

    tmp_dir=$(mktemp -d -t droplet-agent-XXXXXXXXXX)
    cd "${tmp_dir}"
    echo "Temporary directory: $(pwd)"
    echo "Downloading ${pkg_url}"
    curl "${pkg_url}" --output ./droplet-agent.rpm.signed
    echo -n "Verifying package signature..."
    gpg --verify droplet-agent.rpm.signed >/dev/null 2>&1
    echo "OK"
    echo "Extracting package"
    gpg --output droplet-agent.rpm --decrypt droplet-agent.rpm.signed
    rpm -i droplet-agent.rpm --force
  fi

  echo "Finished upgrading droplet-agent"
}

update_deb() {
  # update deb
  find_latest_pkg "${repo}" "deb"
  #get installed version
  local_ver=$(dpkg -s droplet-agent | grep Version | cut -f 2 -d: | tr -d '[:space:]')
  printf "local version:%s\nremote version:%s\n" "${local_ver}" "${remote_ver}"
  if [ "${local_ver}" = "${remote_ver}" ]; then
    echo "No need to update"
    exit 0
  fi

  sorted_vers=$(printf "%s\n%s" "${local_ver}" "${remote_ver}" | sort -V)
  newer_ver=$(echo "${sorted_vers}" | tail -1)

  if [ "${newer_ver}" = "${remote_ver}" ]; then
    echo "Upgrading droplet-agent to ver:${remote_ver}"

    if ! command -v gpg &>/dev/null; then
      echo "Installing GNUPG"
      apt-get -qq update || true
      apt-get install -y gnupg2
    fi

    echo "Ensuring gpg key is ready..."
    wget -qO- "${REPO_GPG_KEY}" | gpg --import
    echo "Ensuring gpg key is trusted..."
    gpg_key_ownertrust=$(wget -qO- "${REPO_GPG_OWNERTRUST}")
    for item in ${gpg_key_ownertrust}; do
      fpr=$(echo "$item" | cut -d ':' -f 1)
      echo "trusting ${fpr}"
      echo -e "5\ny\n" | gpg --command-fd 0 --expert --edit-key "$fpr" trust
    done

    tmp_dir=$(mktemp -d -t droplet-agent-XXXXXXXXXX)
    cd "${tmp_dir}"
    echo "Temporary directory: $(pwd)"
    echo "Downloading ${pkg_url}"
    wget -O ./droplet-agent.deb.signed "${pkg_url}"
    echo -n "Verifying package signature..."
    gpg --verify droplet-agent.deb.signed >/dev/null 2>&1
    echo "OK"
    echo "Extracting package"
    gpg --output droplet-agent.deb --decrypt droplet-agent.deb.signed
    dpkg -i droplet-agent.deb
  fi

  echo "Finished upgrading droplet-agent"
}

script_cleanup() {
  if [ "${exit_status}" -ne 0 ]; then
    echo "droplet-agent update failed"
  fi
  if [ "${tmp_dir}" != "unknown" ]; then
    echo "Removing temporary files"
    rm -rf "${tmp_dir}"
    echo "Done"
  fi
}

main() {
  trap 'exit_status=$?; script_cleanup; exit $exit_status' EXIT

  check_arch
  if command -v apt-get 2 &>/dev/null; then
    update_deb
  elif command -v yum 2 &>/dev/null; then
    update_rpm
  fi
}

# abort with an error message
abort() {
  echo "ERROR: $1" >/dev/stderr
  exit 1
}

main
