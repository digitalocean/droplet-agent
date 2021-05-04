#!/bin/bash
set -ueo pipefail

function main() {
  ver=${1:-}
  [ -z "${ver}" ] && abort "version number is required. Usage: ${FUNCNAME[0]} <ver>"
  git tag -d "${ver}"
  git push origin ":refs/tags/${ver}"
}

main "$@"
