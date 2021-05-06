#!/bin/bash

FALSE=1
TRUE=0

SPACES_ROOT_URL="https://nyc3.digitaloceanspaces.com/eng-droplet-packages"

# print a message to stdout
function announce() {
  msg=${1:-}
  [ -z "$msg" ] && abort "Usage: ${FUNCNAME[0]} <msg>"
  echo ":::::::::::::::::::::::::::::::::::::::::::::::::: $msg ::::::::::::::::::::::::::::::::::::::::::::::::::" >/dev/stderr
}

# send a request to the given url and return the http status
function http_status_for() {
  url=${1:-}
  [ -z "$url" ] && abort "Usage: ${FUNCNAME[0]} <url>"
  curl -LISsL "$url" | grep 'HTTP/' | awk '{ print $2 }'
}

# abort with an error message
function abort() {
  read -r line func file <<<"$(caller 0)"
  echo "ABORT ERROR in $file:$func:$line: $1" >/dev/stderr
  exit 1
}

function confirm() {
  msg=${1:-}
  [ -z "$msg" ] && abort "Usage: ${FUNCNAME[0]} <msg>"

  while true; do
    read -r -p "${msg} (Y/N)" yn
    case $yn in
    [Yy]*)
      return 0
      break
      ;;
    [Nn]*) return 1 ;;
    *) echo "Please answer yes or no." ;;
    esac
  done
}

function spaces_version_exist() {
  version=${1:-}
  [ -z "$version" ] && abort "version is required. Usage: ${FUNCNAME[0]} <version>"

  spaces_url=$(printf "${SPACES_ROOT_URL}/tar/droplet-agent/droplet-agent.%s.amd64.tar.gz" "${version}")
  status_code=$(http_status_for "${spaces_url}")
  case $status_code in
  404)
    return $FALSE
    ;;
  200)
    return $TRUE
    ;;
  *)
    abort "Failed to check if a remote package version already exists. Try again? Got status code '$status_code'"
    ;;
  esac
}

function github_release_exist() {
  version=${1:-}
  [ -z "$version" ] && abort "version is required. Usage: ${FUNCNAME[0]} <version>"

  github_release_url=$(printf "https://github.com/digitalocean/droplet-agent/releases/tag/%s" "${version}")
  status_code=$(http_status_for "${github_release_url}")
  case $status_code in
  404)
    return $FALSE
    ;;
  200)
    return $TRUE
    ;;
  *)
    abort "Failed to check if a github release version already exists. Try again? Got status code '$status_code'"
    ;;
  esac
}

function github_version_exist() {
  version=${1:-}
  [ -z "$version" ] && abort "version is required. Usage: ${FUNCNAME[0]} <version>"

  exist=$(git --no-pager tag | grep "${version}" || echo "no-exist")

  if [ "${exist}" == "no-exist" ]; then
    return $FALSE
  fi
  return $TRUE
}

function is_valid_semver() {
  version=${1:-}
  [ -z "$version" ] && abort "version is required. Usage: ${FUNCNAME[0]} <version>"
  ver_regex='^([0-9]+\.){0,2}(\*|[0-9]+)$'
  if [[ $version =~ $ver_regex ]]; then
    return $TRUE
  else
    return $FALSE
  fi
}

function is_main_branch() {
  branch=${1:-}
  [ -z "$branch" ] && abort "branch is required. Usage: ${FUNCNAME[0]} <branch>"
  if [ "$branch" = "main" ] || [ "$branch" == "master" ]; then
    return $TRUE
  else
    return $FALSE
  fi
}

function get_latest_version() {
  branch=${1:-}
  [ -z "$branch" ] && abort "branch is required. Usage: ${FUNCNAME[0]} <branch>"
  git_tag=$(git describe "${branch}" --tags --abbrev=0 | tr -d v)
  echo "$git_tag"
}

function git_main_branch() {
  if [[ -n "$(git branch --list main)" ]]; then
    echo main
  else
    echo master
  fi
}
