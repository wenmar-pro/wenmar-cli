# Development

How to build, test, and point the CLI at a local server.

## Prerequisites

The CLI depends on the [wenmar-sdk](https://github.com/wenmar-pro/wenmar-sdk)
Go module. The `go.work` at the repo root wires it in as a local module, so
clone it as a sibling:

```bash
git clone https://github.com/wenmar-pro/wenmar-sdk ~/Projects/wenmar-sdk
```

The `go.work` expects the SDK at `../wenmar-sdk/go` relative to this repo.

## Build

```bash
go build -o ./wenmar ./cmd/wenmar
./wenmar --help
```

## Run the test suite

The tests use an in-process fake API (`httptest`) — no network or real server
needed.

```bash
go test ./...
```

Run a single package or test:

```bash
go test ./cmd/ -v
go test ./cmd/ -run TestCustomersList_WrongToken_DebugOutput -v
```

## Point the CLI at a local server

The CLI resolves its base URL from, in order: `--base-url` flag, the
`WENMAR_URL` env var, the config file, a per-repo `.wenmar.yml`, then the
production default. To hit a local Rails server on `localhost:3000`:

```bash
# Start your Rails server
bin/rails server

# Option 1: per-command flag
./wenmar customers list --base-url http://localhost:3000 --token your_dev_token

# Option 2: environment variables
export WENMAR_URL=http://localhost:3000
export WENMAR_TOKEN=your_dev_token
./wenmar customers list
```

The token is resolved from, in order: `--token` flag, `WENMAR_TOKEN` env var,
then the config file. For local development, the env var or flag is simplest.

## Debugging failed requests

When a request fails, the CLI prints a debug block to stderr showing the token
source (masked), the base URL, the HTTP method/path, the status code, and any
per-field validation errors:

```
ERROR: GET /customers -> unauthorized: Invalid or missing API token (HTTP 401)

  token:    abcd...wxyz  (WENMAR_TOKEN env)
  base URL: http://localhost:3000
  request:  GET /customers
  status:   401

  Hint: the token may be invalid, expired, or missing. Run `wenmar setup` to reconfigure.
```

To see the same context on successful commands, pass `--debug`:

```bash
./wenmar customers list --debug
```

## How the debug info is wired

- `internal/errors/debug.go` — `PrintError` renders the debug block; `MaskToken`
  masks tokens for display.
- `internal/auth/auth.go` — `ResolveTokenWithSource` reports where a token came
  from (flag / env / config).
- `cmd/root.go` — `Execute()` calls `PrintError` on failure; the `--debug` flag
  prints context on success.
- `cmd/*.go` — `newSDKClient()` populates `currentDebugInfo` (token source,
  base URL); `setRequest(method, path)` records the request for the current
  command.
- `wenmar-sdk/go/wenmar` — `APIError` carries `Method`/`Path`; the SDK's
  `parseError` captures them from the failed response.
