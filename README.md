# Honda Go Engine

Motor de reserva em Go para uso definitivo, separado do sistema Python legado.

## Requisitos (Windows)

1. Go 1.22+
2. `config.ini` na raiz do projeto
3. `honda.sqlite` na raiz do projeto

## Bootstrap (baixar tudo automaticamente)

Windows:

```powershell
powershell -ExecutionPolicy Bypass -File .\setup.ps1
```

Windows (ja buildando os .exe):

```powershell
powershell -ExecutionPolicy Bypass -File .\setup.ps1 -Build
```

macOS / Linux:

```bash
chmod +x ./setup.sh
./setup.sh
```

macOS / Linux (ja buildando binarios):

```bash
./setup.sh --build
```

## Execucao

Da raiz `Honda Go`:

```powershell
go run ./cmd/honda-engine -config .\config.ini
```

GUI (Windows):

```powershell
go run ./cmd/honda-gui
```

A GUI abre no navegador automaticamente em `http://localhost:8787`. Por padrão, o sistema agora escuta em `0.0.0.0:8787`, o que permite acesso de outros computadores na mesma rede usando o IP da máquina (ex: `http://SEU_IP:8787`).

GUI (macOS):

```bash
go run ./cmd/honda-gui
```

Ou, apos build, execute o binario sem extensao:

```bash
./honda-go-gui
```

Forcar modo na linha de comando:

```powershell
go run ./cmd/honda-engine -config .\config.ini -mode b4
```

Build executavel (Windows):

```powershell
go build -o honda-go-engine.exe ./cmd/honda-engine
go build -o honda-go-gui.exe ./cmd/honda-gui
```

Build executavel local (macOS/Linux):

```bash
go build -o honda-go-engine ./cmd/honda-engine
go build -o honda-go-gui ./cmd/honda-gui
```

Verificacao de encoding UTF-8 (recomendado antes do build):

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\check_encoding.ps1
```

Build cross-platform completo (da maquina atual):

```powershell
# Linux amd64
$env:GOOS='linux';  $env:GOARCH='amd64'
go build -o dist/linux/honda-go-engine ./cmd/honda-engine
go build -o dist/linux/honda-go-gui    ./cmd/honda-gui

# macOS arm64
$env:GOOS='darwin'; $env:GOARCH='arm64'
go build -o dist/macos/honda-go-engine-arm64 ./cmd/honda-engine
go build -o dist/macos/honda-go-gui-arm64    ./cmd/honda-gui

# macOS amd64
$env:GOOS='darwin'; $env:GOARCH='amd64'
go build -o dist/macos/honda-go-engine-amd64 ./cmd/honda-engine
go build -o dist/macos/honda-go-gui-amd64    ./cmd/honda-gui

Remove-Item Env:GOOS
Remove-Item Env:GOARCH
```

Build para macOS (atalho):

```bash
./build_macos.sh
```

Atalho para abrir GUI no macOS (Finder):

```bash
chmod +x run_gui.command
open run_gui.command
```

## Logs

- Gera arquivo versionado em `log\`.
- Nome: `honda_go_YYYYMMDD_HHMMSS_±ZZZZ.log`.
- Tambem escreve no console.

## Configuracao

Le a secao `[BOOKING]` do `config.ini`:

- `reservar_modo` (`b1` ou `b4`)
- `modelo`
- `produto`
- `grupo`
- `cpf`
- `cod_empre`
- `vencimento`
- `id_cota`
- `loteria_federal`
- `acrescimo_decrescimo`
- `tipo_grupo`
- `limit`
- `dry_run`

Chaves adicionais:

- `cooldown_user_ms` (default `200`)
- `worker_count_go` (default `24`)
- `request_timeout_ms` (default `7000`)

Secao opcional `[SYSTEM]`:

- `database_path` (default `honda.sqlite`)
- `api_base_url` (default endpoint oficial)

Secao opcional `[CRAWLER]`:

- `api_url` (default endpoint oficial de valorlance)

Variaveis de ambiente (sobrescrevem `config.ini` se definidas):

- `HONDAGO_TOKEN_PRINCIPAL` -> `BOOKING.token_principal`
- `HONDAGO_API_BASE_URL` -> `SYSTEM.api_base_url`
- `HONDAGO_API_URL` -> `CRAWLER.api_url`
- `HONDAGO_DATABASE_URL` -> ativa conexao Postgres (quando vazio, usa `SYSTEM.database_path` / SQLite)

## Auth migration

No startup, o motor garante os campos abaixo na tabela `auth`:

- `cooldown_until`
- `in_flight`
- `error_401_count`
- `error_429_count`
- `blocked_until`
- `priority_score`
