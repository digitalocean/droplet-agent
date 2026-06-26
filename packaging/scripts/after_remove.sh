#!/bin/sh
# vim: noexpandtab

set -ue

SVC_NAME=droplet-agent
UPDATE_TIMER=${SVC_NAME}-update.timer
INSTALL_RETRY_TIMER=${SVC_NAME}-install-retry.timer
INSTALL_RETRY_SERVICE=${SVC_NAME}-install-retry.service
INSTALL_RETRY_DIR=/var/lib/digitalocean/droplet-agent-install
LEGACY_INIT_SVC_FILE="/etc/init/${SVC_NAME}.conf"
LEGACY_CRON=/etc/cron.hourly/${SVC_NAME}
LEGACY_RETRY_CRON=/etc/cron.hourly/droplet-agent-install

# fix an issue where this script runs on upgrades for rpm
# see https://github.com/jordansissel/fpm/issues/1175#issuecomment-240086016
arg="${1:-0}"

main() {
	if echo "${arg}" | grep -qP '^\d+$' && [ "${arg}" -gt 0 ]; then
		# rpm upgrade
		exit 0
	elif echo "${arg}" | grep -qP '^upgrade$'; then
		# deb upgrade
		exit 0
	fi

	if command -v systemctl >/dev/null 2>&1; then
		clean_systemd
	else
		echo "Unknown init system" > /dev/stderr
	fi

	remove_legacy_cron
	rm -f "${LEGACY_INIT_SVC_FILE}"
}

clean_systemd() {
	echo "Cleaning up systemd scripts"
	systemctl stop "${UPDATE_TIMER}" || true
	systemctl disable "${UPDATE_TIMER}" || true
	systemctl stop "${INSTALL_RETRY_TIMER}" >/dev/null 2>&1 || true
	systemctl disable "${INSTALL_RETRY_TIMER}" >/dev/null 2>&1 || true
	systemctl stop ${SVC_NAME} || true
	systemctl disable ${SVC_NAME}.service || true
	systemctl daemon-reload || true
	rm -f \
		"/etc/systemd/system/${INSTALL_RETRY_TIMER}" \
		"/etc/systemd/system/${INSTALL_RETRY_SERVICE}" \
		"${INSTALL_RETRY_DIR}/retry_install.sh"
}

remove_legacy_cron() {
	rm -f "${LEGACY_CRON}"
	rm -f "${LEGACY_RETRY_CRON}"
}

main
