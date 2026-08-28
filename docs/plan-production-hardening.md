# Wenmar CLI Production Hardening Plan

> **Status:** Planning complete, ready for implementation.
> **Created:** 2026-08-27
> **Scope:** 6-phase overhaul to take wenmar-cli from v0.1 to production-grade.
> **Model:** Patterns derived from hey-sdk / hey-cli, adapted for automotive shop management.

---

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Auth layer location | `wenmar-sdk/go/pkg/auth/` | SDK gets CredentialStore, AuthManager, ForLocation — reusable by other Go consumers |
| OAuth stub | Prompt for token, store in keyring | `auth login` works now; OAuth browser flow added later without interface changes |
| Keyring library | `zalando/go-keyring` | Cross-platform (macOS Keychain, Windows Credential Manager, Linux Secret Service). File fallback for headless/CI |
| TUI/Watch through SDK | Yes | Eliminates duplicate HTTP+auth code; gets retry, caching, auth refresh for free |
| ForLocation scoping | `X-Wenmar-Location` header | Clean, doesn't pollute URLs. `LocationClient` injects header on every request |
| Auto-switch when piped | Yes | Breaking change: piped output → raw JSON. `--styled` forces human tables. Matches hey-cli conventions |
| Plan document | `docs/plan-production-hardening.md` | This file |

---

## Current State

### SDK (`wenmar-sdk/go`)

- Flat `Client{BaseURL, Token}` with `NewClient(baseURL, token)`.
- Already has: retry transport (method-aware), ETag caching, pagination (Link header), cross-origin redirect auth stripping, HTTPS enforcement.
- No auth abstraction — token is baked in at construction via a request editor.
- `ShowLocation` takes a **string** ID; all other methods take **int** IDs.
- Module: `github.com/wenmar-pro/wenmar-sdk/go`, Go 1.27.0.

### CLI (`wenmar-cli`)

- 47 Go files, cobra-based. Module `github.com/wenmar-pro/wenmar-cli`, Go 1.27.0.
- Static token auth only: `--token` flag → `WENMAR_TOKEN` env → config file. No keyring, no OAuth, no refresh.
- `auth login` aliases to `setup`. `auth logout` deletes config. `auth status` tests connection.
- TUI (`internal/tui/board.go`) and watch poller (`internal/watch/poller.go`) **bypass the SDK** — raw `net/http` with manual `Authorization: Bearer` headers.
- Breadcrumbs are just `os.Args` echo — not actionable navigation hints.
- Config struct: `{Token, BaseURL}` only.
- No Makefile. No `make check` drift validation.
- No `upgrade` command. No help topics. No skill auto-install. No Sigstore.

### Domain Terms (locked in)

| Term | Command | Alias | ID type |
|------|---------|-------|---------|
| `locations` | `wenmar locations show <id>` | — | **string** |
| `work_orders` | `wenmar work_orders ...` | `wo` | int |
| `customers` | `wenmar customers ...` | — | int |
| `vehicles` | `wenmar vehicles ...` | — | int |
| `account` | `wenmar account show` | — | singleton |

Never use "shop" (use **location**), never use "ro" (use **wo** or **work_order**).

### Command Tree (current)

```
wenmar
├── account show
├── auth
│   ├── login       (aliases to setup)
│   ├── logout
│   └── status
├── commands
├── completion [shell]
├── config
│   ├── get <key>
│   ├── set <key> <value>
│   ├── list
│   ├── path
│   └── trust
├── customers
│   ├── list (alias ls)
│   ├── show <id>
│   ├── create
│   └── update <id>
├── doctor
├── locations show <id>
├── setup
├── tui
├── url parse <url>
├── vehicles
│   ├── show <id>
│   ├── list (alias ls)
│   ├── create
│   ├── update <id>
│   ├── delete <id>
│   ├── decode-vin <vin>
│   └── duplicates <vin>
├── version
├── watch
└── work_orders (alias wo)
    ├── list (alias ls)
    ├── show <id>
    ├── create
    ├── update <id>
    └── delete <id>
```

### Global Flags (current)

`--token`, `--base-url`, `--md`/`-m`/`--markdown`, `--json`, `--agent`, `--quiet`, `--jq`, `--ids-only`, `--count`, `--config-path`, `--debug`

### Output Envelope (current)

```json
{"ok": true, "data": [...], "summary": "5 customers", "meta": {"has_next": true}, "breadcrumbs": [{"cmd": "wenmar customers list"}]}
```

Built as `map[string]any` — no typed `Envelope` struct. Breadcrumbs are `os.Args` joined.

---

## Phase 1: Auth & Identity Foundation

> **Goal:** Replace static-token-only auth with CredentialStore + AuthManager in the SDK, keyring-backed storage, `ForLocation` scoping, and a full `wenmar auth` command tree. OAuth is stubbed — `auth login` prompts for a token and stores it in the keyring.

### 1.1 SDK: Auth Package

**New files in `wenmar-sdk/go/pkg/auth/`:**

#### `credential_store.go`

```go
package auth

type CredentialStore interface {
    GetToken(ctx context.Context) (*Token, error)
    SaveToken(ctx context.Context, token *Token) error
    DeleteToken(ctx context.Context) error
}

// KeyringStore uses zalando/go-keyring (service: "wenmar-cli", item: "token")
type KeyringStore struct{}

// FileStore falls back to ~/.config/wenmar/credentials.json (mode 0600)
type FileStore struct {
    Path string
}

// NewCredentialStore tries keyring first, falls back to file.
// On headless Linux without D-Bus, keyring.ErrUnavailable triggers fallback.
func NewCredentialStore() CredentialStore
```

#### `token.go`

```go
package auth

type Token struct {
    AccessToken  string     `json:"access_token"`
    RefreshToken string     `json:"refresh_token,omitempty"`
    ExpiresAt    *time.Time `json:"expires_at,omitempty"`
    TokenType    string     `json:"token_type,omitempty"` // default "Bearer"
}

func (t *Token) IsExpired() bool
func (t *Token) WillExpireWithin(d time.Duration) bool
```

#### `token_provider.go`

```go
package auth

type TokenProvider interface {
    Token(ctx context.Context) (string, error)
}

// StaticTokenProvider wraps a fixed string. Used for --token flag and WENMAR_TOKEN env.
type StaticTokenProvider struct{ Token string }

// CredentialStoreProvider reads from a CredentialStore, auto-refreshing
// via the AuthManager when the token is expired or near-expiry (5 min window).
type CredentialStoreProvider struct {
    Store  CredentialStore
    Manager *AuthManager
}
```

#### `auth_manager.go`

```go
package auth

type AuthManager struct {
    Store          CredentialStore
    Provider       TokenProvider
    refreshFn      func(ctx context.Context, refreshToken string) (*Token, error)
}

func (m *AuthManager) Token(ctx context.Context) (string, error)
func (m *AuthManager) Refresh(ctx context.Context) error
func (m *AuthManager) Logout(ctx context.Context) error
```

**OAuth stub:** `refreshFn` defaults to a function that returns `ErrOAuthNotImplemented`:
```go
var ErrOAuthNotImplemented = errors.New("OAuth token refresh is not yet implemented. Re-run `wenmar auth login` to get a new token")
```

When OAuth lands, only `refreshFn` changes — no caller code breaks.

#### `auth_strategy.go`

```go
package auth

type AuthStrategy interface {
    Authenticate(req *http.Request) error
}

type BearerAuth struct {
    Provider TokenProvider
}

func (b *BearerAuth) Authenticate(req *http.Request) error
```

This replaces the hardcoded request editor in `client.go`.

#### `config.go`

```go
package auth

type Config struct {
    BaseURL    string `json:"base_url" yaml:"base_url"`
    Token      string `json:"token,omitempty" yaml:"token,omitempty"` // static token (legacy/file fallback)
    AuthMethod string `json:"auth_method" yaml:"auth_method"`         // "static" | "oauth"
    LocationID string `json:"location_id,omitempty" yaml:"location_id,omitempty"`
}

func DefaultConfig() Config
func LoadConfigFromEnv() Config // reads WENMAR_TOKEN, WENMAR_URL, WENMAR_LOCATION_ID
```

### 1.2 SDK: Client Changes

**Modify `wenmar-sdk/go/wenmar/client.go`:**

Change `NewClient` to functional options:

```go
type ClientOption func(*Client) error

func WithTokenProvider(tp auth.TokenProvider) ClientOption
func WithAuthStrategy(s auth.AuthStrategy) ClientOption
func WithHTTPClient(h *http.Client) ClientOption
func WithUserAgent(ua string) ClientOption

func NewClient(baseURL string, opts ...ClientOption) (*Client, error)

// Backward compat convenience wrapper
func NewClientWithToken(baseURL, token string) (*Client, error)
```

The request editor now calls `authStrategy.Authenticate(req)` instead of hardcoding `Bearer "+token`.

`Client.Token` field becomes `Client.Auth *auth.AuthManager` (accessible for refresh).

**New file `wenmar-sdk/go/wenmar/location.go`:**

```go
// LocationClient scopes all requests to a specific location via the
// X-Wenmar-Location header.
type LocationClient struct {
    *Client
    locationID string
}

// ForLocation verifies access to the given location and returns a scoped client.
func (c *Client) ForLocation(ctx context.Context, locationID string) (*LocationClient, error)
```

`LocationClient` injects `X-Wenmar-Location: <locationID>` header on every request by wrapping the `AuthStrategy` or adding a second request editor.

All `Client` methods are available on `LocationClient` via embedding.

**SDK `go.mod`:** Add `github.com/zalando/go-keyring`.

### 1.3 CLI: `wenmar auth` Command Tree

**Rewrite `cmd/auth.go`** — expand from 3 to 6 subcommands:

| Command | Flags | Behavior (Phase 1) | OAuth Future |
|---------|-------|-------------------|--------------|
| `auth login` | `--token <t>` | Prompt for token (or use `--token`), store in keyring via `CredentialStore`. Print: "OAuth browser flow will be added in a future release. For now, enter your API token." | Replace prompt with OAuth device/browser flow |
| `auth status` | — | Show: logged in / not logged in, token source (keyring / file / env / flag), token mask, expiry (if available), base URL, connection test via `ListAccount` | Add OAuth token expiry display |
| `auth token` | — | Print bearer token to stdout (for scripts). Refuses if auth method is cookie-based (future) | Same |
| `auth refresh` | — | Calls `AuthManager.Refresh(ctx)`. Returns `ErrOAuthNotImplemented` with guidance | Performs OAuth refresh, stores new token |
| `auth logout` | — | Clears keyring + file credentials + config file token | Same |
| `auth login` | `--token <t>` | Non-interactive token store. For scripts/CI | Token from OAuth exchange |

### 1.4 CLI: Config & Auth Resolution

**`internal/config/config.go`:**

```go
type Config struct {
    Token      string `yaml:"token"`       // legacy / file fallback
    BaseURL    string `yaml:"base_url"`
    AuthMethod string `yaml:"auth_method"` // "static" (default) | "oauth"
    LocationID string `yaml:"location_id"`
}
```

Config file stores `base_url`, `auth_method`, `location_id`. Token goes to keyring (config file token is legacy fallback).

**`internal/auth/auth.go`:**

```go
// ResolveAuthManager builds the full auth stack: keyring → file → env → flag
func ResolveAuthManager(flagToken, configPath string) (*auth.AuthManager, error)
```

Token resolution order:
1. `--token` flag → `StaticTokenProvider`
2. `WENMAR_TOKEN` env → `StaticTokenProvider`
3. Keyring (via `CredentialStore`) → `CredentialStoreProvider` with auto-refresh
4. Config file token → `StaticTokenProvider` (legacy)
5. None → error: "API token required. Run `wenmar auth login`."

### 1.5 CLI: `ForLocation` Scoping

**`cmd/root.go`:** Add `--location` global flag:
```go
rootCmd.PersistentFlags().StringVar(&locationIDFlag, "location", "", "Location ID to scope requests (or set WENMAR_LOCATION_ID)")
```

Resolution: `--location` flag → `WENMAR_LOCATION_ID` env → config `location_id` → `.wenmar.yml` `location_id` (if trusted).

When set, `newClient()` returns `client.ForLocation(ctx, locationID)` instead of the bare client.

### 1.6 CLI: Update `setup` and `newClient`

**`cmd/setup.go`:**
- `runSetup` uses `CredentialStore.SaveToken()` instead of writing `config.Config{Token: ...}`.
- Config file stores `base_url` and `auth_method` only (token goes to keyring).
- Falls back to file credentials if keyring unavailable.
- `auth login` is no longer a simple alias — it's its own command with `--token` support.

**`cmd/customers.go` (where `newSDKClient` lives):**
- Rename `newSDKClient()` → `newClient()`.
- Replace `wenmar.NewClient(baseURL, token)` with `wenmar.NewClient(baseURL, wenmar.WithTokenProvider(authManager))`.
- If `--location` is set, return `client.ForLocation(ctx, locationID)`.
- All `cmd/*.go` command files call `newClient()`.

### 1.7 Tests

- `wenmar-sdk/go/pkg/auth/credential_store_test.go` — mock keyring (go-keyring's `NewArrayKeyring`), file store roundtrip.
- `wenmar-sdk/go/pkg/auth/auth_manager_test.go` — token resolution, refresh stub, logout.
- `wenmar-sdk/go/wenmar/client_test.go` — functional options, `WithTokenProvider`, backward compat `NewClientWithToken`.
- `wenmar-sdk/go/wenmar/location_test.go` — `ForLocation` header injection.
- `cmd/auth_test.go` — `auth login --token`, `auth status`, `auth logout`, `auth token` output.
- `internal/auth/auth_test.go` — `ResolveAuthManager` precedence (flag/env/keyring/config).

### 1.8 Files Touched in Phase 1

| File | Action |
|------|--------|
| `wenmar-sdk/go/pkg/auth/credential_store.go` | **New** |
| `wenmar-sdk/go/pkg/auth/token.go` | **New** |
| `wenmar-sdk/go/pkg/auth/token_provider.go` | **New** |
| `wenmar-sdk/go/pkg/auth/auth_manager.go` | **New** |
| `wenmar-sdk/go/pkg/auth/auth_strategy.go` | **New** |
| `wenmar-sdk/go/pkg/auth/config.go` | **New** |
| `wenmar-sdk/go/pkg/auth/credential_store_test.go` | **New** |
| `wenmar-sdk/go/pkg/auth/token_test.go` | **New** |
| `wenmar-sdk/go/pkg/auth/auth_manager_test.go` | **New** |
| `wenmar-sdk/go/wenmar/client.go` | **Modify**: functional options, `WithTokenProvider`, `WithAuthStrategy` |
| `wenmar-sdk/go/wenmar/client_test.go` | **Modify**: test new options |
| `wenmar-sdk/go/wenmar/location.go` | **New**: `ForLocation`, `LocationClient` |
| `wenmar-sdk/go/wenmar/location_test.go` | **New** |
| `wenmar-sdk/go/go.mod` | **Modify**: add `zalando/go-keyring` |
| `cmd/auth.go` | **Rewrite**: 6 subcommands |
| `cmd/auth_test.go` | **New/Modify** |
| `cmd/setup.go` | **Modify**: use `CredentialStore` |
| `cmd/root.go` | **Modify**: add `--location` flag |
| `internal/config/config.go` | **Modify**: add `AuthMethod`, `LocationID` |
| `internal/config/config_test.go` | **Modify** |
| `internal/auth/auth.go` | **Modify**: `ResolveAuthManager`, keyring precedence |
| `internal/auth/auth_test.go` | **Modify** |
| `cmd/customers.go` | **Modify**: `newSDKClient` → `newClient` with options |
| All `cmd/*.go` command files | **Modify**: call `newClient()` |

---

## Phase 2: TUI Hardening + Live Watch Integration

> **Goal:** Route TUI + watch through the SDK. Add live polling to the TUI. Add TUI detail view and actions. Expand `wenmar watch` with `--location` and event filtering.

### 2.1 Route TUI + Watch Through the SDK

**`internal/tui/board.go`:**
- Remove `fetchWorkOrders(baseURL, token)` — the raw `net/http` call.
- `NewBoard` signature: `NewBoard(client *wenmar.Client, locationID string, interval time.Duration)`.
- `fetchWorkOrders` becomes a `tea.Cmd` that calls `client.ListWorkOrders(ctx)` (or `client.ForLocation(ctx, locationID).ListWorkOrders(ctx)`), extracts data via `extractData` pattern.
- Gets retry, caching, auth refresh for free.

**`internal/watch/poller.go`:**
- `Poller` struct: replace `URL` + `Token` with `Client *wenmar.Client` + `Resource string` + `LocationID string`.
- `fetch()` calls the SDK method instead of raw HTTP.
- Auth, retry, caching all inherited.

**`cmd/watch.go`:**
- Builds client via `newClient()` (same as all other commands).
- Passes client to `watch.Poller`.

### 2.2 TUI Live Updates

**`internal/tui/board.go`:**
- Add `tickMsg` and `tick()` command that fires every `interval` (default 10s, configurable via `wenmar tui --interval`).
- `Update` handles `tickMsg` → triggers `fetchWorkOrders`.
- Footer shows: last-refresh timestamp, connection status indicator (`● online` green / `● offline` red).
- `Ctrl+R` for manual refresh (in addition to `r`).

**`cmd/tui.go`:**
- Add flags: `--location <id>`, `--interval <dur>` (default 10s), `--work-order <id>` (jump to detail), `--remote` (desktop integration view spec).
- `--work-order <id>` opens detail view directly instead of the list.

### 2.3 TUI Detail View + Actions

**New file `internal/tui/detail.go`:**

`DetailModel` — shows a single work order:
- Customer info (name, phone, email)
- Vehicle info (year, make, model, VIN, plate)
- Status badge
- Line items (if present in API response)
- Notes
- Timestamps (created, updated)

Navigation:
- `Enter` on a list item → switch to `DetailModel`
- `Esc` / `Backspace` → return to list
- `q` → quit

Actions from detail view (require API endpoints):
- `s` → start time tracking (`POST /work_orders/{id}/time_entries` — **API endpoint needed**)
- `c` → mark complete (`PATCH /work_orders/{id}` with `status: "completed"`)
- `r` → refresh this work order

**`internal/tui/board.go`:** Add `view` state enum (`viewList` / `viewDetail`), dispatch `View()` and `Update()` based on current view.

**`internal/tui/keys.go`:** Add `Enter`, `s`, `c`, `Esc`, `Backspace`, `Ctrl+R`, `?` (help toggle) bindings.

### 2.4 Expand `wenmar watch`

**`cmd/watch.go`:**
- Add `--location <id>` flag (scopes the polled endpoint).
- Add `--resource <name>` flag (currently hardcoded to `work_orders`; expand to `customers`, `vehicles`).
- Event output gains richer fields: location name, vehicle make/model for work_order events.
- Fix `--run-sync`: currently uses `fmt.Sprintf("%v", e)` — change to `json.Encode`.
- Add `--run-async` flag: run script without blocking the poll loop.

### 2.5 Files Touched in Phase 2

| File | Action |
|------|--------|
| `internal/tui/board.go` | **Rewrite**: SDK client, live polling, view state |
| `internal/tui/detail.go` | **New**: work order detail view + actions |
| `internal/tui/detail_test.go` | **New** |
| `internal/tui/keys.go` | **Modify**: new bindings |
| `internal/tui/styles.go` | **Modify**: detail view styles, status indicator |
| `cmd/tui.go` | **Modify**: new flags, view routing |
| `internal/watch/poller.go` | **Rewrite**: SDK-based fetching |
| `internal/watch/poller_test.go` | **Modify**: mock client |
| `cmd/watch.go` | **Modify**: `--location`, `--resource`, fix `--run-sync` JSON |

---

## Phase 3: Output & Agent Experience

> **Goal:** Actionable breadcrumbs, `--html` output, `--styled` flag, help topics, `config show` with provenance.

### 3.1 Actionable Breadcrumbs

**`internal/output/output.go`:**

Replace `Breadcrumb{Cmd string}` with:
```go
type Breadcrumb struct {
    Action string `json:"action"` // "show", "create", "update", "delete"
    Cmd    string `json:"cmd"`    // "wenmar work_orders show <id>"
}
```

Replace `CaptureBreadcrumbs()` (which echoes `os.Args`) with per-command breadcrumbs:

- `customers list` → `[{Action: "show", Cmd: "wenmar customers show <id>"}, {Action: "create", Cmd: "wenmar customers create --full-name \"\""}]`
- `customers show 42` → `[{Action: "update", Cmd: "wenmar customers update 42"}, {Action: "delete", Cmd: "wenmar customers delete 42 --dry-run"}]`
- `work_orders list` → `[{Action: "show", Cmd: "wenmar work_orders show <id>"}, {Action: "create", Cmd: "wenmar work_orders create --customer-id <id> --vehicle-id <id>"}]`

Each command's `RunE` builds breadcrumbs based on what it returned and passes them to `output.Render` via `Options`.

### 3.2 `--html` Output Mode

**`internal/output/html.go` (new):**

`renderHTML(w, data, title)` — renders data as a simple HTML document:
- For work orders: customer info, vehicle info, line items table, status badge.
- For lists: HTML table with headers.
- For single entities: definition list.

Add `ModeHTML` to `Mode` enum and `ResolveMode`. Add `--html` persistent flag to root.

### 3.3 `--styled` Flag + Auto-Switch

**`cmd/root.go`:**
- Add `--styled` persistent flag.
- When stdout is **not a TTY** and no explicit output mode is set (`--json`, `--md`, `--agent`, `--quiet`, `--jq`, `--ids-only`, `--count`, `--html`), auto-switch to `ModeQuiet` (raw JSON).
- `--styled` forces `ModeDefault` (human tables) even when piped.
- Detection: `term.IsTerminal(int(os.Stdout.Fd()))` or equivalent.

This is a **breaking change**: `wenmar customers list | cat` now produces JSON. Users who want tables in pipes use `--styled`.

### 3.4 Help Topics

**New file `cmd/help_topics.go`:**

Register `wenmar help <topic>` for:

| Topic | Contents |
|-------|----------|
| `output` | Output formats (`--json`, `--md`, `--agent`, `--quiet`, `--jq`, `--ids-only`, `--count`, `--html`, `--styled`), auto-switch behavior, envelope structure |
| `exit-codes` | Stable process exit statuses (0-10), when each fires |
| `environment` | `WENMAR_TOKEN`, `WENMAR_URL`, `WENMAR_LOCATION_ID` env vars |
| `auth` | Auth methods (`static`, `oauth`), token sources, keyring vs file, refresh |
| `location` | Location scoping (`--location`, `X-Wenmar-Location` header, `ForLocation`) |
| `watch` | Watch command, event types, `--run-sync`, polling intervals |

Each topic is a cobra command that prints formatted help text. `wenmar help` (no args) lists available topics. Topics also emit `--agent` JSON.

**`internal/agent/discovery.go`:** Add help topics to catalog with `type: "topic"` field.

### 3.5 `config show` with Provenance

**`cmd/config.go`:** Add `wenmar config show`:
```
token:       sk-****1234  (from: keyring)
base_url:    https://app.wenmarpro.com  (from: default)
location_id: loc_abc123  (from: env WENMAR_LOCATION_ID)
auth_method: static  (from: config file ~/.config/wenmar/config)
```

**`internal/config/config.go`:** Add `ResolveWithProvenance()` that returns each value with its source.

### 3.6 Files Touched in Phase 3

| File | Action |
|------|--------|
| `internal/output/output.go` | **Modify**: new `Breadcrumb` struct, `ModeHTML`, auto-switch |
| `internal/output/html.go` | **New**: HTML renderer |
| `internal/output/html_test.go` | **New** |
| `cmd/root.go` | **Modify**: `--html`, `--styled` flags, auto-switch logic |
| `cmd/help_topics.go` | **New**: help topic commands |
| `internal/agent/discovery.go` | **Modify**: topic support in catalog |
| `cmd/config.go` | **Modify**: `config show` with provenance |
| `internal/config/config.go` | **Modify**: `ResolveWithProvenance` |
| All command files | **Modify**: build per-command breadcrumbs |

---

## Phase 4: Operations & Reliability

> **Goal:** `wenmar upgrade`, hardened `doctor`, pagination notices on stderr, `--allow-partial`, richer error taxonomy.

### 4.1 `wenmar upgrade`

**New file `cmd/upgrade.go`:**

Detects install method and delegates:
- `~/.local/bin` or `~/bin` (installer script): download latest release, verify SHA-256, swap binary with backup.
- `mise`: delegate to `mise upgrade wenmar`.
- Homebrew: delegate to `brew upgrade`.
- `go install`: refuse with guidance.

Flags:
- `wenmar upgrade` — latest version
- `wenmar upgrade 0.2.0` — pin to specific version
- `wenmar upgrade --check` — print latest available without upgrading
- `wenmar upgrade --force` — skip confirmation

Phase 6 adds Sigstore verification to this command.

### 4.2 Harden `wenmar doctor`

**`cmd/doctor.go`:** Add checks:

| Check | Description |
|-------|-------------|
| `config` | Config file exists and is readable |
| `token` | Token present (keyring or file) |
| `auth_method` | static vs oauth |
| `keyring` | Keyring accessible (or file fallback active) |
| `connectivity` | API reachable, token valid (`ListAccount`) |
| `completion` | Shell completion installed |
| `skill` | `~/.agents/skills/wenmar/SKILL.md` exists, managed by wenmar-cli (check `.managed-by-wenmar-cli` marker) |
| `plugin` | Claude Code plugin installed (if Claude detected) |

Fix: `doctor` currently has its own local `--json` flag shadowing the global one. Unify to use global `--json`.

### 4.3 Pagination Notices on stderr

**`internal/output/output.go`:**
- When `Meta.HasNext` is true and mode is `ModeIDsOnly` or `ModeCount`, print pagination notice to **stderr**: `"Page 1 of 3. Use --page 2 for next."`
- stdout remains pure IDs/count — pipeable.
- Ensure `--page <int>` flag exists on all list commands.

### 4.4 `--allow-partial` for Large Reads

**`cmd/work_orders.go` (and other show commands):**
- Add `--allow-partial` flag.
- If API response is truncated (detected via `X-Wenmar-Truncated: true` header or `truncated: true` in response body), fail by default: "Response was truncated. Use --allow-partial to accept partial data."
- With `--allow-partial`, add `notice` field to JSON envelope: `{"ok": true, "data": ..., "notice": "Response was truncated. 87 of 120 line items returned."}`

### 4.5 Richer Error Taxonomy

**`internal/errors/exit.go`:** Expand from 6 to 10 exit codes:

| Code | Constant | Trigger |
|------|----------|---------|
| 0 | `ExitSuccess` | Success |
| 1 | `ExitGeneric` | Unclassified error |
| 2 | `ExitAuth` | Unauthorized, token expired |
| 3 | `ExitNotFound` | Resource not found |
| 4 | `ExitValidation` | Validation failed |
| 5 | `ExitRateLimit` | Rate limited |
| 6 | `ExitServer` | 5xx server error |
| 7 | `ExitConflict` | 409 conflict (e.g., duplicate VIN) |
| 8 | `ExitForbidden` | 403 forbidden |
| 9 | `ExitPartial` | Truncated response without `--allow-partial` |
| 10 | `ExitOffline` | Network unreachable |

Update `ExitCode()` to map new API error codes. Document in `wenmar help exit-codes`.

### 4.6 Files Touched in Phase 4

| File | Action |
|------|--------|
| `cmd/upgrade.go` | **New**: self-upgrade command |
| `cmd/upgrade_test.go` | **New** |
| `cmd/doctor.go` | **Modify**: new checks, unify `--json` |
| `cmd/doctor_test.go` | **Modify** |
| `internal/output/output.go` | **Modify**: stderr pagination notices |
| `cmd/work_orders.go` | **Modify**: `--allow-partial` |
| `cmd/customers.go` | **Modify**: `--allow-partial` |
| `cmd/vehicles.go` | **Modify**: `--allow-partial` |
| `internal/errors/exit.go` | **Modify**: 10 exit codes |
| `internal/errors/exit_test.go` | **Modify** |

---

## Phase 5: Agent Integration

> **Goal:** Auto-install skill with `.managed-by` markers, `setup claude` / `setup codex`, `commands --json` with `compatibility_for`.

### 5.1 Skill Auto-Install with `.managed-by` Markers

**New file `cmd/skill.go`:**

| Command | Behavior |
|---------|----------|
| `wenmar skill install` | Copy `skills/wenmar/SKILL.md` to `~/.agents/skills/wenmar/SKILL.md`. Write `.managed-by-wenmar-cli` marker. Refuse if dir exists without marker (use `--force`). |
| `wenmar skill uninstall` | Remove dir only if `.managed-by-wenmar-cli` marker present. |
| `wenmar skill update` | Re-copy skill file (only if marker present). |

**New file `internal/agent/install.go`:**
```go
func InstallSkill(targetDir string, force bool) error
func UninstallSkill(targetDir string) error
func SkillInstalled() (path string, managed bool, err error)
```

**Critical rule:** Only write to directories with `.managed-by-wenmar-cli` marker. Never overwrite hand-authored skills without `--force`.

### 5.2 `setup claude` / `setup codex`

**`cmd/setup.go`:**
- `wenmar setup claude` — detect Claude Code (`~/.claude/` exists), install skill to `~/.agents/skills/wenmar/`, install Claude Code plugin/marketplace entry.
- `wenmar setup codex` — detect Codex, install skill to appropriate Codex skill directory.
- `wenmar setup` (no args) — full wizard: auth + agent auto-detection + completions.
- `--skip-agents` — auth only.
- `--silent-success` — minimal output on success (for harnesses).

### 5.3 `commands --json` with `compatibility_for`

**`internal/agent/discovery.go`:**
```go
type CommandInfo struct {
    Path            string     `json:"path"`
    Description     string     `json:"description"`
    Aliases         []string   `json:"aliases,omitempty"`
    Args            []ArgInfo  `json:"args,omitempty"`
    Flags           []FlagInfo `json:"flags,omitempty"`
    Canonical       bool       `json:"canonical"`
    CompatibilityFor string    `json:"compatibility_for,omitempty"`
    Type            string     `json:"type,omitempty"` // "command" | "topic"
}
```

When a command has aliases (`wo` for `work_orders`, `ls` for `list`), the alias entry gets `compatibility_for: "work_orders"` so agents know the canonical name.

### 5.4 Files Touched in Phase 5

| File | Action |
|------|--------|
| `cmd/skill.go` | **New**: skill install/uninstall/update |
| `cmd/skill_test.go` | **New** |
| `internal/agent/install.go` | **New**: skill installation logic |
| `internal/agent/install_test.go` | **New** |
| `cmd/setup.go` | **Modify**: `setup claude`, `setup codex`, agent detection |
| `cmd/setup_test.go` | **Modify** |
| `internal/agent/discovery.go` | **Modify**: `compatibility_for`, `canonical`, `type` |
| `internal/agent/discovery_test.go` | **Modify** |
| `cmd/commands.go` | **Modify**: emit enhanced catalog |

---

## Phase 6: Install & Distribution

> **Goal:** Sigstore verification, package manager support, Makefile with drift validation.

### 6.1 Sigstore Signature Verification

**`install-cli` script:**
After SHA-256 verification, add cosign verification:
```bash
cosign verify-blob --bundle checksums.txt.bundle \
  --certificate-identity "https://github.com/wenmar-pro/wenmar-cli/.github/workflows/release.yml@refs/tags/v${VERSION}" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  checksums.txt
```
If cosign is not installed, warn but continue (SHA-256 is the baseline). `wenmar upgrade` also verifies via cosign.

**`.github/workflows/release.yml`:** Add cosign signing step — sign `checksums.txt` with cosign, upload `.bundle` as release asset.

### 6.2 Package Managers

| Manager | Command | Setup |
|---------|---------|-------|
| Homebrew | `brew install --cask wenmar-pro/tap/wenmar` | Add tap repo with cask formula |
| Scoop | `scoop install wenmar` | Add to scoop bucket |
| AUR | `yay -S wenmar-cli` | PKGBUILD |
| mise | `mise use -g github:wenmar-pro/wenmar-cli` | Uses GitHub releases |
| deb/rpm/apk | GitHub releases | goreleaser nfpm section |

### 6.3 Makefile with Drift Validation

**New file `Makefile`:**
```makefile
.PHONY: build test check clean

build:
	go build -o ./wenmar ./cmd/wenmar

test:
	go test ./... -v

check: build test
	@echo "Checking SDK drift..."
	@(cd ../wenmar-sdk/go && go test ./...)
	./wenmar --help > /dev/null
	./wenmar commands --json | jq . > /dev/null

clean:
	rm -f wenmar
```

### 6.4 Files Touched in Phase 6

| File | Action |
|------|--------|
| `install-cli` | **Modify**: cosign verification |
| `.github/workflows/release.yml` | **Modify**: cosign signing step |
| `.goreleaser.yml` | **Modify**: nfpm (deb/rpm), Homebrew tap |
| `Makefile` | **New**: build/test/check/clean |
| `README.md` | **Modify**: package manager install instructions |

---

## Automotive-Specific Features

Domain features that make the CLI irreplaceable on a shop floor. Each depends on API endpoints — marked with what API support is needed:

| Feature | Command | API Needed | Phase |
|---------|---------|------------|-------|
| VIN decode | `wenmar vehicles decode-vin <vin>` | **Already exists** | Done |
| Plate lookup | `wenmar vehicles list --plate ABC123 --state ON` | Query param support | Phase 2+ |
| Time tracking | `wenmar time start <wo_id>` / `wenmar time stop` | `POST /work_orders/{id}/time_entries` | Phase 2 (TUI), later as CLI command |
| Bay assignment | `wenmar work_orders assign <id> --bay 3` | `PATCH /work_orders/{id}` with `bay_id` | Phase 2+ |
| Status pipeline | `wenmar work_orders move <id> --to in_progress` | `PATCH /work_orders/{id}` with `status` | Phase 2+ |
| Parts lookup | `wenmar parts search --q "brake pad" --vehicle 42` | `GET /parts` endpoint | Future |
| Customer check-in | `wenmar customers check-in 42 --vehicle 5` | `POST /customers/{id}/check_in` | Future |
| End-of-day batch | `wenmar work_orders bulk-close --completed-before 2026-08-27 --dry-run` | Bulk close endpoint | Future |
| Odometer conversion | `wenmar vehicles show 42 --units miles` | Client-side conversion | Future |

---

## Timeline

| Week | Phase | Key Deliverables |
|------|-------|------------------|
| **1** | Phase 1 | SDK auth package (6 new files). `NewClient` functional options. `ForLocation`. CLI `auth` command tree (6 commands). Keyring + file fallback. Update `setup`. `--location` flag. All commands routed through new client. |
| **2** | Phase 2 | TUI + watch through SDK. TUI live polling (10s). TUI detail view + actions (start/stop, mark complete). Expand `watch` with `--location`, `--resource`, fixed `--run-sync`. |
| **3** | Phase 3+5 | Actionable breadcrumbs. `--html`. `--styled` + auto-switch. Help topics. `config show` with provenance. Skill auto-install with `.managed-by`. `setup claude` / `setup codex`. `commands --json` with `compatibility_for`. |
| **4** | Phase 4+6 | `wenmar upgrade`. Hardened `doctor` (8 checks). Pagination stderr notices. `--allow-partial`. Error taxonomy (10 codes). Sigstore. Makefile. Package managers. |

---

## OAuth Note

OAuth browser/device flow is **not implemented in this plan**. The auth interfaces (`CredentialStore`, `TokenProvider`, `AuthManager`, `AuthStrategy`) are designed so that adding OAuth later requires only:

1. Implement `refreshFn` in `AuthManager` (the OAuth token exchange).
2. Replace `auth login` prompt with browser/device flow.
3. Store `RefreshToken` + `ExpiresAt` in `CredentialStore`.

No caller code changes. The `auth refresh` command will work automatically once `refreshFn` is implemented.

---

## Verification

After each phase:

```bash
make check          # build + test + SDK drift + CLI smoke
go test ./... -v    # full test suite
./wenmar doctor     # health check
./wenmar commands --json | jq .  # agent catalog valid
```

Phase 1 acceptance criteria:
- [ ] `wenmar auth login --token <t>` stores token in keyring
- [ ] `wenmar auth status` shows token source and connection test
- [ ] `wenmar auth token` prints bearer token
- [ ] `wenmar auth logout` clears keyring + config
- [ ] `wenmar auth refresh` returns `ErrOAuthNotImplemented` with guidance
- [ ] `wenmar --location <id> customers list` sends `X-Wenmar-Location` header
- [ ] `wenmar setup` stores token in keyring, not config file
- [ ] All existing tests pass
- [ ] New auth tests pass