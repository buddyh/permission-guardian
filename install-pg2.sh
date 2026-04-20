#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$ROOT_DIR"

TARGET_NAME="${TARGET_NAME:-pg2}"
INSTALL_DIR="${INSTALL_DIR:-}"

if [[ -z "${INSTALL_DIR}" ]]; then
  if [[ -d "${HOME}/scripts" ]]; then
    INSTALL_DIR="${HOME}/scripts"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

mkdir -p "${INSTALL_DIR}"
"${ROOT_DIR}/build-pg2.sh"
install "${ROOT_DIR}/pg2" "${INSTALL_DIR}/${TARGET_NAME}"

echo "installed ${TARGET_NAME} to ${INSTALL_DIR}/${TARGET_NAME}"
