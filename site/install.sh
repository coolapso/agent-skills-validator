#!/usr/bin/env bash
# Install agent-skills-validator from GitHub Releases.
#
#   curl -fsSL https://coolapso.github.io/agent-skills-validator/install.sh | bash
#   curl -fsSL https://coolapso.github.io/agent-skills-validator/install.sh | VERSION=v1.2.0 bash
#
# Also used by action.yaml, which sets INSTALL_DIR to a writable temp directory.
#
# Environment:
#   VERSION      release tag to install (default: latest)
#   INSTALL_DIR  destination directory (default: /usr/local/bin)
#   GH_TOKEN     optional token used when resolving the latest release via the GitHub API
#   BASE_URL     override the release download URL (testing only)
set -euo pipefail

REPO="coolapso/agent-skills-validator"
BINARY_NAME="agent-skills-validator"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

case "$OS" in
  linux | darwin) EXT="tar.gz" ;;
  mingw* | msys* | cygwin* | windows*)
    OS="windows"
    EXT="zip"
    BINARY_NAME="${BINARY_NAME}.exe"
    ;;
  *)
    echo "Unsupported OS: $OS" >&2
    exit 1
    ;;
esac

if [[ -n "${VERSION:-}" && "$VERSION" != "latest" ]]; then
  TAG="$VERSION"
else
  AUTH=()
  if [[ -n "${GH_TOKEN:-}" ]]; then
    AUTH=(-H "Authorization: Bearer ${GH_TOKEN}")
  fi
  TAG="$(curl -fsSL "${AUTH[@]}" -H "Accept: application/vnd.github+json" \
    "https://api.github.com/repos/${REPO}/releases/latest" \
    | sed -n 's/^[[:space:]]*"tag_name":[[:space:]]*"\([^"]*\)".*/\1/p' | head -n1)"
fi

if [[ -z "$TAG" ]]; then
  echo "Failed to determine the release version." >&2
  exit 1
fi
TAG="v${TAG#v}"
VERSION_NO_V="${TAG#v}"

FILENAME="agent-skills-validator_${VERSION_NO_V}_${OS}_${ARCH}.${EXT}"
BASE_URL="${BASE_URL:-https://github.com/${REPO}/releases/download/${TAG}}"

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading agent-skills-validator ${TAG} for ${OS}/${ARCH}..."
curl -fsSL --retry 3 -o "${TMP_DIR}/${FILENAME}" "${BASE_URL}/${FILENAME}"
curl -fsSL --retry 3 -o "${TMP_DIR}/checksums.txt" "${BASE_URL}/checksums.txt"

EXPECTED="$(grep -E "[[:space:]]\*?${FILENAME}\$" "${TMP_DIR}/checksums.txt" | awk '{print $1}')"
if [[ -z "$EXPECTED" ]]; then
  echo "checksums.txt does not list ${FILENAME}" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  ACTUAL="$(sha256sum "${TMP_DIR}/${FILENAME}" | awk '{print $1}')"
else
  ACTUAL="$(shasum -a 256 "${TMP_DIR}/${FILENAME}" | awk '{print $1}')"
fi
if [[ "$EXPECTED" != "$ACTUAL" ]]; then
  echo "Checksum mismatch for ${FILENAME}" >&2
  exit 1
fi
echo "Checksum verified."

if [[ "$EXT" == "tar.gz" ]]; then
  tar -xzf "${TMP_DIR}/${FILENAME}" -C "$TMP_DIR"
elif command -v unzip >/dev/null 2>&1; then
  unzip -q "${TMP_DIR}/${FILENAME}" -d "$TMP_DIR"
elif command -v 7z >/dev/null 2>&1; then
  7z x -y -o"$TMP_DIR" "${TMP_DIR}/${FILENAME}" >/dev/null
else
  powershell -NoProfile -Command "Expand-Archive -Force -LiteralPath '${TMP_DIR}/${FILENAME}' -DestinationPath '${TMP_DIR}'"
fi

# Create the directory without privileges when possible; fall back to sudo
# only when the destination is not writable by the current user.
mkdir -p "$INSTALL_DIR" 2>/dev/null || true
SUDO=""
if [[ ! -d "$INSTALL_DIR" || ! -w "$INSTALL_DIR" ]]; then
  if command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
    echo "Elevating with sudo to write to ${INSTALL_DIR}"
  else
    echo "${INSTALL_DIR} is not writable; set INSTALL_DIR to a directory you own." >&2
    exit 1
  fi
fi

echo "Installing to ${INSTALL_DIR}/${BINARY_NAME}..."
$SUDO mkdir -p "$INSTALL_DIR"
$SUDO cp "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
$SUDO chmod 755 "${INSTALL_DIR}/${BINARY_NAME}"
"${INSTALL_DIR}/${BINARY_NAME}" version

echo ""
echo "✅ agent-skills-validator ${TAG} installed successfully!"
echo "Run 'agent-skills-validator validate ./my-skill' to get started."
