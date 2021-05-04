#!/bin/bash
set -ueo pipefail

DIR="${BASH_SOURCE%/*}"
if [[ ! -d "${DIR}" ]]; then DIR="${PWD}"; fi
. "${DIR}/util.sh"

remote_latest_ver="0.0.0"
function newer_than_spaces_versions() {
  ver=${1:-}
  [ -z "${ver}" ] && abort "version number is required. Usage: ${FUNCNAME[0]} <ver>"
  repo_tree=$(curl -sSL "${SPACES_ROOT_URL}" || wget -qO- "${SPACES_ROOT_URL}")
  files=$(echo "${repo_tree}" | grep -oE '(Key>signed/dotty-agent/deb/x86_64/)[^<]+' | grep -oE '\d.\d.\d' | tr ' ' '\n')
  files=$(printf "%s\n%s" "${files}" "${ver}")
  latest_ver=$(echo "${files}" | sort -V | tail -1)

  if [ "${latest_ver}" != "${ver}" ]; then
    remote_latest_ver=${latest_ver}
    return "$FALSE"
  fi
  return "$TRUE"
}

git_latest_ver="0.0.0"
function newer_than_git_releases() {
  ver=${1:-}
  [ -z "${ver}" ] && abort "Destination repository is required. Usage: ${FUNCNAME[0]} <ver>"

  git_vers=$(git --no-pager tag)
  git_vers=$(printf "%s\n%s" "${git_vers}" "${ver}")
  latest_ver=$(echo "${git_vers}" | sort -V | tail -1)

  if [ "${latest_ver}" != "${ver}" ]; then
    git_latest_ver=${latest_ver}
    return "$FALSE"
  fi
  return "$TRUE"
}

function main() {
  ver=""
  desc=""
  commit=""
  skip_spaces_check="$FALSE"
  while test $# -gt 0; do
    case "$1" in
    -h | --help)
      echo "Usage: ${FUNCNAME[0]} -v <ver> -d \"<desc>\" -m commit(optional) --skip_spaces_check(optional)"
      exit 0
      ;;
    -v)
      shift
      if test $# -gt 0; then
        ver=${1:-}
      else
        echo "no version specified"
        exit 1
      fi
      shift
      ;;
    -d)
      shift
      if test $# -gt 0; then
        desc=${1:-}
      else
        echo "no description specified"
        exit 1
      fi
      shift
      ;;
    -m)
      shift
      if test $# -gt 0; then
        commit=${1:-}
      else
        echo "no commit sha specified"
        exit 1
      fi
      shift
      ;;
    --skip_spaces_check)
      skip_spaces_check="$TRUE"
      shift
      ;;
    esac
  done

  if [ -z "${ver}" ] || [ -z "${desc}" ]; then
    abort "missing version or description. Usage: ${FUNCNAME[0]} -v <ver> -d \"<desc>\" -m commit(optional) --skip_spaces_check(optional)"
  fi

  echo -n "Validating version ${ver}..."
  if ! is_valid_semver "${ver}"; then
    abort "${ver} is not a valid semantic version number"
  fi
  echo "OK"

  echo -n "Checking if ${ver} has been deployed to spaces..."
  if [ "${skip_spaces_check}" == "$TRUE" ]; then
    echo "Skipped"
  else
    if spaces_version_exist "${ver}"; then
      abort "${ver} already deployed to spaces"
    fi
    echo "OK"
  fi
  echo -n "Checking if ${ver} has been released to github..."
  if github_release_exist "${ver}"; then
    abort "${ver} already released to github"
  fi
  echo "OK"

  echo -n "Checking if ${ver} already exist..."
  if github_version_exist "${ver}"; then
    echo "Failed"
    echo "${ver} already exist, here's the info:"
    echo "--------------------- Version: ${ver} ------------------------"
    git --no-pager show "${ver}"
    echo "--------------------------------------------------------------"
    if ! confirm "Do you want to replace the existing tag:${ver}"; then
      echo "Operation Cancelled"
      exit 1
    fi
    git tag -d "${ver}"
    git push --delete origin "${ver}" || true
  else
    echo "OK"
  fi

  echo -n "Checking if ${ver} can be deployed to spaces..."
  if [ "${skip_spaces_check}" == "$TRUE" ]; then
    echo "Skipped"
  else
    if ! newer_than_spaces_versions "${ver}"; then
      abort "${ver} should be greater than the latest version on spaces: ${remote_latest_ver}"
    fi
    echo "OK"
  fi
  echo -n "Checking if ${ver} can be released to github..."
  if ! newer_than_git_releases "${ver}"; then
    abort "${ver} should be greater than the latest version on github: ${git_latest_ver}"
  fi
  echo "OK"

  if [ -z "${commit}" ]; then
    echo "Tagging version ${ver} to the latest commit with message: ${desc}"
    git tag -a "${ver}" -m "${desc}"
  else
    echo "Tagging version ${ver} to commit ${commit} with message: ${desc}"
    git tag -a "${ver}" -m "${desc}" "${commit}"
  fi
  git push origin "${ver}"
}

main "$@"
