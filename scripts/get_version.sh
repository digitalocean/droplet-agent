#!/bin/bash
set -ueo pipefail

DIR="${BASH_SOURCE%/*}"
if [[ ! -d "${DIR}" ]]; then DIR="${PWD}"; fi
. "${DIR}/util.sh"

current_branch=$(git rev-parse --abbrev-ref HEAD)

function main() {
  ver_in_branch=$(get_latest_version "${current_branch}")
  echo "${ver_in_branch}"
}

main "$@"
