#!/usr/bin/env bash
# Remove agent-skills-validator installed by install.sh.
#   curl -fsSL https://coolapso.github.io/agent-skills-validator/uninstall.sh | bash
set -euo pipefail

INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TARGET="${INSTALL_DIR}/agent-skills-validator"

if [[ ! -e "$TARGET" ]]; then
  echo "agent-skills-validator is not installed in ${INSTALL_DIR}"
  exit 0
fi

SUDO=""
if [[ ! -w "$INSTALL_DIR" ]] && command -v sudo >/dev/null 2>&1; then
  SUDO="sudo"
fi

$SUDO rm -f "$TARGET"
echo "agent-skills-validator uninstalled successfully. Thank you for using it!"
