# Wenmar CLI

A command-line interface for the [Wenmar Pro](https://github.com/Wenmar-Pro/wenmar-pro) automotive shop management API.

Built on the [wenmar-sdk Go module](https://github.com/Wenmar-Pro/wenmar-sdk). Single static binary, no runtime dependencies.

## Install

### Homebrew (macOS)

```bash
brew tap Wenmar-Pro/homebrew-wenmar
brew install wenmar
```

### Binary download

Download the latest release from [GitHub Releases](https://github.com/Wenmar-Pro/wenmar-cli/releases) for your platform:

- `wenmar_darwin_arm64` (Apple Silicon)
- `wenmar_darwin_amd64` (Intel Mac)
- `wenmar_linux_amd64`
- `wenmar_linux_arm64`
- `wenmar_windows_amd64.exe`

### Build from source

```bash
go install github.com/Wenmar-Pro/wenmar-cli/cmd/wenmar@latest
```

## Setup

```bash
export WENMAR_TOKEN="your-api-token"
```

Get a token from the Wenmar Pro settings page.

## Usage

```bash
# List customers (human-readable table)
wenmar customers list

# List customers (JSON envelope for scripts)
wenmar customers list --json

# List customers (raw data for AI agents)
wenmar customers list --agent

# Filter with jq
wenmar customers list --jq '.[].full_name'

# Show a customer
wenmar customers show 42

# Create a customer
wenmar customers create --full-name "Jane Doe" --email "jane@test.com"

# Work orders
wenmar work_orders list
wenmar work_orders show 100

# Vehicles
wenmar vehicles show 5
```

## Output modes

| Flag | Description |
|------|-------------|
| (default) | GFM table — human-readable |
| `--md` / `-m` | GFM table (explicit) |
| `--json` | Full JSON envelope `{ok, data, summary, meta}` |
| `--agent` | Raw JSON data (no envelope) |
| `--jq 'filter'` | jq-filtered JSON |

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Generic error |
| 2 | Auth failure |
| 3 | Not found |
| 4 | Validation error |
| 6 | Server error |

## Agent discovery

```bash
wenmar commands          # full command catalog as JSON
wenmar customers list --help --agent  # structured help for one command
```

See [`skills/wenmar/SKILL.md`](skills/wenmar/SKILL.md) for the full agent skill file.

## License

MIT
