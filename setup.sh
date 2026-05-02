#!/bin/bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "$0")" && pwd)"
cd "$ROOT_DIR"

echo "[setup] Honda Go bootstrap (macOS/Linux)"

if ! command -v go >/dev/null 2>&1; then
  echo "[setup] Go nao encontrado no PATH. Instale Go 1.22+." >&2
  exit 1
fi

echo "[setup] $(go version)"

if [[ ! -f "go.mod" ]]; then
  echo "[setup] Execute este script na raiz do projeto (onde existe go.mod)." >&2
  exit 1
fi

echo "[setup] Baixando modulos..."
go mod download

echo "[setup] Verificando modulos..."
go mod verify

mkdir -p log

if [[ ! -f "config.ini" && -f "config.ini.example" ]]; then
  cp config.ini.example config.ini
  echo "[setup] config.ini criado a partir de config.ini.example."
fi

if [[ "${1:-}" == "--build" ]]; then
  echo "[setup] Buildando binarios locais..."
  go build -o ./honda-go-engine ./cmd/honda-engine
  go build -o ./honda-go-gui ./cmd/honda-gui
  echo "[setup] Build concluido."
fi

echo "[setup] OK. Dependencias prontas."
echo "[setup] GUI: go run ./cmd/honda-gui"
