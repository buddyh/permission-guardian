#!/usr/bin/env bash
set -euo pipefail

REPO="buddyh/permission-guardian"
BINARY="pg"
INSTALL_DIR="${INSTALL_DIR:-}"
VERSION="${VERSION:-latest}"

need_cmd() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "missing required command: $1" >&2
    exit 1
  }
}

need_cmd curl
need_cmd tar
need_cmd uname

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"

case "$os" in
  darwin|linux) ;;
  *)
    echo "unsupported OS: $os" >&2
    exit 1
    ;;
esac

case "$arch" in
  x86_64) arch="amd64" ;;
  arm64|aarch64) arch="arm64" ;;
  *)
    echo "unsupported architecture: $arch" >&2
    exit 1
    ;;
esac

if [[ "$VERSION" == "latest" ]]; then
  VERSION="$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | awk -F'"' '/"tag_name":/ {print $4; exit}')"
  VERSION="${VERSION#v}"
fi

if [[ -z "$VERSION" ]]; then
  echo "failed to resolve release version" >&2
  exit 1
fi

archive="permission-guardian_${VERSION}_${os}_${arch}.tar.gz"
checksums="checksums.txt"
base_url="https://github.com/${REPO}/releases/download/v${VERSION}"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

curl -fsSL "${base_url}/${archive}" -o "${tmpdir}/${archive}"
curl -fsSL "${base_url}/${checksums}" -o "${tmpdir}/${checksums}"

if command -v sha256sum >/dev/null 2>&1; then
  (cd "$tmpdir" && sha256sum -c --ignore-missing "${checksums}")
elif command -v shasum >/dev/null 2>&1; then
  expected="$(awk -v f="$archive" '$2 == f {print $1}' "${tmpdir}/${checksums}")"
  actual="$(shasum -a 256 "${tmpdir}/${archive}" | awk '{print $1}')"
  [[ -n "$expected" && "$expected" == "$actual" ]] || {
    echo "checksum verification failed for ${archive}" >&2
    exit 1
  }
else
  echo "missing checksum tool: need sha256sum or shasum" >&2
  exit 1
fi

tar -xzf "${tmpdir}/${archive}" -C "${tmpdir}" "${BINARY}"

if [[ -z "${INSTALL_DIR}" ]]; then
  if [[ -w "/usr/local/bin" ]]; then
    INSTALL_DIR="/usr/local/bin"
  elif [[ -w "/opt/homebrew/bin" ]]; then
    INSTALL_DIR="/opt/homebrew/bin"
  else
    INSTALL_DIR="${HOME}/.local/bin"
  fi
fi

mkdir -p "${INSTALL_DIR}"

if [[ -w "${INSTALL_DIR}" ]]; then
  install "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  sudo install "${tmpdir}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "installed ${BINARY} ${VERSION} to ${INSTALL_DIR}/${BINARY}"
