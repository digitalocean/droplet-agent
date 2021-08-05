#!/bin/bash
set -ueo pipefail

DIR="${BASH_SOURCE%/*}"
if [[ ! -d "${DIR}" ]]; then DIR="${PWD}"; fi
. "${DIR}/util.sh"

current_branch=$(git rev-parse --abbrev-ref HEAD)
main_branch=$(git_main_branch)

function main() {
  announce "Checking Version"
  latest_ver=$(get_latest_version "${main_branch}")
  [ -z "${latest_ver}" ] && abort "Missing version tag on branch: ${main_branch}"

  if ! is_main_branch "${current_branch}"; then
    ver_in_branch=$(get_latest_version "${current_branch}")
    if [ "${ver_in_branch}" == "${latest_ver}" ]; then
      abort "no new version found on current branch"
    fi
    latest_ver=${ver_in_branch}
  fi
  echo "Version:${latest_ver}"

  if ! is_valid_semver "${latest_ver}"; then
    abort "${latest_ver} not valid semantic version number"
  fi
  if spaces_version_exist "${latest_ver}"; then
    abort "${latest_ver} already deployed"
  fi
  echo "Okay to deployed"
}

main "$@"
