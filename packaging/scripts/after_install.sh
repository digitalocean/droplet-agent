#!/bin/sh
# vim: noexpandtab

set -ue

SVC_NAME=droplet-agent
INSTALL_DIR=/opt/digitalocean/${SVC_NAME}
CRON_SCHEDULE=/etc/cron.hourly
CRON=${CRON_SCHEDULE}/${SVC_NAME}
INIT_SVC_FILE="/etc/init/${SVC_NAME}.conf"
SYSTEMD_SVC_FILE="/etc/systemd/system/${SVC_NAME}.service"

main() {
  if command -v systemctl >/dev/null 2>&1; then
    # systemd is used, remove the upstart script
    rm -f "${INIT_SVC_FILE}"
    # systemctl enable --now is unsupported on older versions of debian/systemd
    echo "enable systemd service"
    systemctl daemon-reload
    systemctl enable -f ${SVC_NAME}
    systemctl restart ${SVC_NAME}
  elif command -v initctl >/dev/null 2>&1; then
    # upstart is used, remove the systemd script
    rm -f "${SYSTEMD_SVC_FILE}"
    echo "enable upstart service"
    initctl stop ${SVC_NAME} || true
    initctl reload-configuration
    initctl start ${SVC_NAME}
  else
    echo "Unknown init system. Exiting..." >/dev/stderr
    exit 1
  fi

  patch_updates
}

patch_updates() {
  # make sure we have the latest
  [ -f "${CRON}" ] && rm -f "${CRON}"
  script="${INSTALL_DIR}/scripts/update.sh"
  mkdir -p ${CRON_SCHEDULE}

  cat <<-EOF >"${CRON}"
	#!/bin/sh
	/bin/bash ${script} >/var/log/droplet-agent.update.log 2>&1
	EOF

  chmod +x "${CRON}"
}

main
