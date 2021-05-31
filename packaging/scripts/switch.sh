#!/bin/sh

SVC_NAME=droplet-agent
CRON_SCHEDULE=/etc/cron.hourly
CRON=${CRON_SCHEDULE}/${SVC_NAME}
RETRY_CRON=${CRON_SCHEDULE}/dotty-agent-switch
exit_status=0

clean_upstart() {
  echo "Cleaning up init scripts"
  initctl stop ${SVC_NAME} || true
  initctl reload-configuration || true
}

clean_systemd() {
  echo "Cleaning up systemd scripts"
  systemctl stop ${SVC_NAME} || true
  systemctl disable ${SVC_NAME}.service || true
  systemctl daemon-reload || true
}

manual_cleanup() {
  if command -v systemctl >/dev/null 2>&1; then
    clean_systemd
  elif command -v initctl >/dev/null 2>&1; then
    clean_upstart
  else
    echo "Unknown init system" >/dev/stderr
  fi
  rm -rf "/opt/digitalocean/dotty-agent"
  rm -fv "${CRON}"
}

patch_retry_switch() {
  [ -f "${RETRY_CRON}" ] && rm -f "${RETRY_CRON}"
  mkdir -p ${CRON_SCHEDULE}
  cat <<'EOF' >"${RETRY_CRON}"
#!/bin/sh
/bin/bash /opt/digitalocean/dotty-agent/scripts/switch.sh >/var/log/dotty.switch.log 2>&1
EOF

  chmod +x "${RETRY_CRON}"
}

remove_retry() {
  rm -f "${RETRY_CRON}"
}

script_cleanup() {
  if [ ${exit_status} -ne 0 ]; then
    patch_retry_switch
  else
    echo "DoTTY agent is successfully removed and Droplet agent is now installed"
    remove_retry || true
  fi
}

main() {

  trap 'exit_status=$?; script_cleanup; exit $exit_status' EXIT

  if command -v yum 2 &>/dev/null; then
    rpm -e --allmatches dotty-agent
  elif command -v apt-get 2 &>/dev/null; then
    dpkg --purge dotty-agent
  fi

  if [ $? -ne 0 ]; then
    manual_cleanup
  fi

  if command -v curl 2 &>/dev/null; then
    curl -sSL https://repos-droplet.digitalocean.com/install.sh | sudo bash
  elif command -v wget 2 &>/dev/null; then
    wget -qO- https://repos-droplet.digitalocean.com/install.sh | sudo bash
  else
    echo "fatal error!"
    exit 1
  fi
}

main