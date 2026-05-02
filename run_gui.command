#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$SCRIPT_DIR"

if [[ -x "./honda-go-gui" ]]; then
  ./honda-go-gui
else
  go run ./cmd/honda-gui
fi
