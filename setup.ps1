Param(
    [switch]$Build
)

$ErrorActionPreference = "Stop"

Write-Host "[setup] Honda Go bootstrap (Windows)" -ForegroundColor Cyan

if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    throw "Go nao encontrado no PATH. Instale Go 1.22+ e tente novamente."
}

$goVersionRaw = (& go version)
Write-Host "[setup] $goVersionRaw"

if (-not (Test-Path ".\go.mod")) {
    throw "Execute este script na raiz do projeto (onde existe go.mod)."
}

Write-Host "[setup] Baixando modulos..."
& go mod download

Write-Host "[setup] Verificando modulos..."
& go mod verify

if (-not (Test-Path ".\log")) {
    New-Item -ItemType Directory -Path ".\log" | Out-Null
    Write-Host "[setup] Pasta log criada."
}

if ((-not (Test-Path ".\config.ini")) -and (Test-Path ".\config.ini.example")) {
    Copy-Item ".\config.ini.example" ".\config.ini"
    Write-Host "[setup] config.ini criado a partir de config.ini.example."
}

if ($Build) {
    Write-Host "[setup] Buildando executaveis..."
    & go build -o .\honda-go-engine.exe .\cmd\honda-engine
    & go build -o .\honda-go-gui.exe .\cmd\honda-gui
    Write-Host "[setup] Build concluido."
}

Write-Host "[setup] OK. Dependencias prontas." -ForegroundColor Green
Write-Host "[setup] GUI: go run ./cmd/honda-gui"
