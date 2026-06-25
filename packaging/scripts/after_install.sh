#!/bin/sh
# vim: noexpandtab

set -ue

SVC_NAME=droplet-agent
LEGACY_INIT_SVC_FILE="/etc/init/${SVC_NAME}.conf"
LEGACY_CRON=/etc/cron.hourly/${SVC_NAME}
LEGACY_RETRY_CRON=/etc/cron.hourly/droplet-agent-install
UPDATE_TIMER=${SVC_NAME}-update.timer
INSTALL_RETRY_TIMER=${SVC_NAME}-install-retry.timer
INSTALL_RETRY_SERVICE=${SVC_NAME}-install-retry.service
INSTALL_RETRY_DIR=/var/lib/digitalocean/droplet-agent-install

main() {
  if ! command -v systemctl >/dev/null 2>&1; then
    echo "systemd is required. Exiting..." >/dev/stderr
    exit 1
  fi

  rm -f "${LEGACY_INIT_SVC_FILE}"

  # systemctl enable --now is unsupported on older versions of debian/systemd
  echo "enable systemd service"
  systemctl daemon-reload
  systemctl enable -f ${SVC_NAME}
  systemctl restart ${SVC_NAME}

  configure_updates
}

configure_updates() {
  remove_legacy_cron
  remove_install_retry_timer

  systemctl daemon-reload
  systemctl enable "${UPDATE_TIMER}"
  systemctl restart "${UPDATE_TIMER}"
}

remove_install_retry_timer() {
  systemctl stop "${INSTALL_RETRY_TIMER}" || true
  systemctl disable "${INSTALL_RETRY_TIMER}" || true
  rm -f \
    "/etc/systemd/system/${INSTALL_RETRY_TIMER}" \
    "/etc/systemd/system/${INSTALL_RETRY_SERVICE}" \
    "${INSTALL_RETRY_DIR}/retry_install.sh"
}

remove_legacy_cron() {
  [ -f "${LEGACY_CRON}" ] && rm -f "${LEGACY_CRON}"
  [ -f "${LEGACY_RETRY_CRON}" ] && rm -f "${LEGACY_RETRY_CRON}"
}

main
