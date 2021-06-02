#!/bin/bash
# vim: noexpandtab

main() {
  # add some jitter to prevent overloading the remote repo server
  sleep $((RANDOM % 900))

  if command -v apt-get 2 &>/dev/null; then
    export DEBIAN_FRONTEND="noninteractive"
    apt-get -qq update -o Dir::Etc::SourceParts=/dev/null -o APT::Get::List-Cleanup=no -o Dir::Etc::SourceList="sources.list.d/droplet-agent.list"
    apt-get -o Dpkg::Options::="--force-confdef" -o Dpkg::Options::="--force-confold" -qq install -y --only-upgrade droplet-agent
  elif command -v yum 2 &>/dev/null; then
    yum -q -y --disablerepo="*" --enablerepo="droplet-agent" makecache
    yum -q -y update droplet-agent
  fi
}

main
