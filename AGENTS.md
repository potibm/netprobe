# Agent Notes for netprobe

## Project
- Go network-monitoring tool for the Evoke demoparty.
- Module: `github.com/potibm/netprobe` (Go 1.25+; `mise` pins 1.26).

## Entry Points
- **App**: `src/cmd/netprobe.go` (not `cmd/` at root).
- **Packages**: all code lives under `src/internal/{config,net,probes}`. CLI parsing is handled by cobra/viper in `src/cmd`.

## Task Runner
- Prefer **`mise`** over Make; tasks are defined in `mise.toml`.
- Useful commands:
  - `mise run build` — builds to `artifacts/build/netprobe`.
  - `mise run run` — runs with `--interface en0 --config config/example.yaml --ip-family 6`.
  - `mise run test` — runs `go test` with coverage, writes `artifacts/coverage/`.
  - `mise run lint` — runs Go formatting (`be:lint`) and repo formatting (`repo:lint`).
  - `mise run deps:install` — `go mod download`.
  - `mise run deps:update` — bumps mise tools, Go modules, and GitHub Actions.

## Build / Release
- Build and release inject version via `-ldflags "-X main.version=..."`.
- `.goreleaser.yaml` cross-builds `linux/windows/darwin` with `CGO_ENABLED=0`.
- `go generate ./...` is in the GoReleaser `before` hooks; there are no `go:generate` directives in the source today.

## Linting
- **Go**: only `gofmt -w ./src/` is used (`mise run be:lint`). `golangci-lint` is installed by mise but not invoked in tasks.
- **Repo files**: `npx prettier --check/write` via `mise/tasks/repo/lint`.

## Testing
- Run single package: `go test -v ./src/internal/config`
- Full suite: `mise run test` (uses `-coverpkg=./...`, outputs HTML report to `artifacts/coverage/coverage.html`).

## CLI
- Uses **cobra** for commands/flags and **viper** for env binding.
- Required flags: `--interface` (`-i`), `--config` (`-c`).
- Optional flags: `--log-level` (default `info`), `--ip-family` (default `4`).
- Env vars work via the `NETPROBE_` prefix (e.g., `NETPROBE_INTERFACE=eth0`).

## Running Locally
- The default `mise run run` uses interface `en0`, which is macOS-specific. On Linux/Windows replace `en0` with your interface name (e.g., `eth0`, `wlan0`).

## Dependencies
- Minimal external deps: `github.com/goccy/go-yaml` and `github.com/stretchr/testify`.
- `go-licenses` is used by `mise run licenses` to regenerate `THIRD-PARTY-NOTICES.md`.

## Gotchas
- `artifacts/` and `dist/` are gitignored.
- No GitHub Actions workflows exist yet, despite `mise` referencing `actions:deps:update`.
