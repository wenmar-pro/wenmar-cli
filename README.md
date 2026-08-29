# Wenmar CLI

A command-line interface for the [Wenmar Pro](https://wenmarpro.com) automotive shop management software.

Built on the [wenmar-sdk Go module](https://github.com/wenmar-pro/wenmar-sdk). Single static binary, no runtime dependencies.
- [API documentation](https://github.com/wenmar-pro/wenmar-sdk/tree/main/docs/api) — full API reference in the wenmar-sdk repo

## Install

### macOS / Linux / WSL2 (curl)

```bash
curl -fsSL https://raw.githubusercontent.com/wenmar-pro/wenmar-cli/main/install-cli | bash
```

The script detects your OS/arch, downloads the latest release, verifies
its SHA-256 checksum, and installs `wenmar` to `~/bin` or `~/.local/bin`
(whichever is on your PATH). It then prompts you to add the directory to
PATH if needed.

To install a specific version or into a custom directory:

```bash
WENMAR_VERSION=0.1.0 curl -fsSL https://raw.githubusercontent.com/wenmar-pro/wenmar-cli/main/install-cli | bash
WENMAR_BIN_DIR=~/custom/bin curl -fsSL https://raw.githubusercontent.com/wenmar-pro/wenmar-cli/main/install-cli | bash
```

### Binary download

Download the latest release from [GitHub Releases](https://github.com/wenmar-pro/wenmar-cli/releases) for your platform:

- `wenmar_<version>_darwin_arm64.tar.gz` (Apple Silicon)
- `wenmar_<version>_darwin_amd64.tar.gz` (Intel Mac)
- `wenmar_<version>_linux_amd64.tar.gz`
- `wenmar_<version>_linux_arm64.tar.gz`
- `wenmar_<version>_windows_amd64.zip`

### Package managers

```bash
# Homebrew (macOS / Linux)
brew install wenmar-pro/tap/wenmar

# Scoop (Windows)
scoop install wenmar

# AUR (Arch Linux)
yay -S wenmar-cli

# mise
mise use -g github:wenmar-pro/wenmar-cli

# deb / rpm / apk
# Available as release assets (wenmar_<version>_<os>_<arch>.deb / .rpm / .apk)
```

### Build from source

```bash
go install github.com/wenmar-pro/wenmar-cli/cmd/wenmar@latest
```

## Setup

```bash
export WENMAR_TOKEN="your-api-token"
```

Get a token from the Wenmar Pro settings page.

```bash
# Interactive setup (stores token in the system keyring)
wenmar setup

# Or store a token non-interactively
wenmar auth login --token <your-api-token>
```

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
| `--html` | HTML document |
| `--styled` | Force human tables even when piped |

When stdout is not a TTY (e.g. piped to another command) and no explicit
output mode is set, wenmar emits raw JSON so the output is machine-readable.
Use `--styled` to force human tables in a pipe.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Generic error |
| 2 | Auth failure |
| 3 | Not found |
| 4 | Validation error |
| 5 | Rate limited |
| 6 | Server error |
| 7 | Conflict (e.g. duplicate VIN) |
| 8 | Forbidden |
| 9 | Truncated response without `--allow-partial` |
| 10 | Network unreachable |

## Agent discovery

```bash
wenmar commands          # full command catalog as JSON
wenmar customers list --help --agent  # structured help for one command
```

See [`skills/wenmar/SKILL.md`](skills/wenmar/SKILL.md) for the full agent skill file.

## License

MIT
