#!/usr/bin/env bash
# Sub2API local WSL Docker Compose helper.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=deploy/wsl-local-lib.sh
source "${SCRIPT_DIR}/wsl-local-lib.sh"
# shellcheck source=deploy/wsl-local-flow.sh
source "${SCRIPT_DIR}/wsl-local-flow.sh"

main "$@"
