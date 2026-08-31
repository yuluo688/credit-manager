# CPA Credit Manager (`credit-manager`)

[中文文档](README.md)

A native Go plugin for [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI). It provides dedicated plugin-key authentication, Key-level credit controls, usage-based settlement, and a self-service usage dashboard.

Plugin ID: `credit-manager`  
Version: `1.7.0`  
Repository: https://github.com/yuluo688/credit-manager

## Features

1. Dedicated `tk-...` plugin-key authentication, independent of host `api-keys`.
2. Strict Key-level reservation before a request is forwarded.
3. Settlement from actual upstream usage: tokens for text models, per image for image-only models. Pricing rules can overlay context and `service_tier` cards.
4. Per-Key total, daily, weekly, monthly, and concurrency limits, plus optional model allowlists.
5. Management console and API for Keys, model enable/disable, pricing, usage, and audit history.
6. A public self-service dashboard where a Key holder can view only their own quota and usage.
7. A management-only OAuth auth-quota view for Codex, Claude, Antigravity, Kimi, and xAI credentials.

## Quick Start

1. Build and deploy the shared library. On Windows:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1
```

This copies the DLL to `D:\CLIProxyAPI\plugins\windows\amd64` by default.

2. Enable `credit-manager` in CLIProxyAPI Plugin Management.
3. Restart CLIProxyAPI after replacing the DLL. Windows cannot normally hot-replace a loaded DLL.
4. Open **CPA Credit Manager** in the CPA sidebar, enter the host management token, create a Key, and set its quota, concurrency, allowed models, and pricing rules.
5. Call models with the issued Key as an opaque string:

```text
Authorization: Bearer tk-...
```

> When enabled, this plugin owns frontend authentication for proxied model requests. Existing CPA `api-keys` no longer authenticate model calls. Create and use `tk-...` plugin Keys instead.

> For CPA Management Center compatibility, `GET /v1/models` and `GET /v1beta/models` work without a plugin Key. That only returns the model directory; it cannot execute models. Globally disabled models are omitted. A Key's allowlist does not hide models from this public catalog.

Zero-config first start:

| Item | Default |
|---|---|
| Pepper | Auto-created at `data_dir/key-peppers` |
| Caller | `default` for Key ownership and usage grouping; it does not hold quota |
| Pricing | Rule `bootstrap-all-models`: regexp `.*`, price 0; `unknown_policy=allow` |
| Plugin Key | **Not** auto-created; mint one from the console or management API |

## Prerequisites

| Component | Requirement |
|---|---|
| CLIProxyAPI | `v7.2.128+` recommended, with upstream credentials already working |
| Plugin binary | Platform shared library (Windows `.dll` / Linux `.so` / macOS `.dylib`) |
| C toolchain | `gcc`/`clang` for CGO `c-shared`. Windows can also use Zig (`zig cc`) |
| Go | `1.26+` |

### Build

**Windows (PowerShell)**

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1

# or:
$env:CGO_ENABLED = "1"
New-Item -ItemType Directory -Force -Path dist | Out-Null
go build -buildmode=c-shared -o dist\credit-manager.dll .
```

`scripts/build.ps1` prefers `gcc`, then falls back to `zig cc`.

**Linux / macOS**

```bash
chmod +x scripts/build.sh
./scripts/build.sh
# Linux: dist/credit-manager.so
# macOS: dist/credit-manager.dylib
```

### Pepper

HMAC verification and recoverable Key storage depend on pepper material. Peppers are **never stored in SQLite**.

Resolution order:

1. Non-empty `CREDIT_MANAGER_KEY_PEPPERS`
2. Else `data_dir/key-peppers`
3. If missing, first start writes a 32-byte random pepper (`0600`)

Back up the entire `data_dir`, including `key-peppers`. Losing the pepper invalidates existing Keys and makes stored ciphertext unreadable.

Format: `id:pepper`. Multiple entries can be comma-separated or one per line. `keys.active_pepper_id` selects the pepper used to mint new Keys.

### Host config

```yaml
plugins:
  enabled: true
  dir: ./plugins
  items:
    credit-manager:
      enabled: true
```

Optional overlay:

```yaml
items:
  credit-manager:
    enabled: true
    config: |
      config_file: /path/to/config.yaml
```

```bash
export CREDIT_MANAGER_CONFIG_FILE=/path/to/config.yaml
```

Host inline fields overlay the file. Nested `config_file` inside that file is ignored. See [`config.example.yaml`](config.example.yaml).

The plugin does not log into upstream providers. The host must still have OAuth files or API keys. Default `data_dir` is `./data/credit-manager` and must be writable.

## Key Limits

Every Key can have the following independent limits. `0` or an omitted field means unlimited.

| Field | Meaning |
|---|---|
| `total_quota_micro_usd` | Total spend limit (`quota_micro_usd` is an alias) |
| `daily_quota_micro_usd` | UTC calendar-day limit |
| `weekly_quota_micro_usd` | UTC calendar-week limit, starting Monday |
| `monthly_quota_micro_usd` | UTC calendar-month limit |
| `max_concurrent_requests` | Maximum held/in-flight requests |
| `allowed_models` | Empty = all models; otherwise exact/glob patterns |
| `expires_at` | Optional RFC3339 expiry |
| `key_material` | Optional existing `tk-...` plaintext to import |

Period limits include settled usage plus active reservations. Amounts are integer micro-USD: `1 USD = 1,000,000 micro-USD`.

The system reserves a conservative amount before forwarding, then settles from actual usage. Settlement can exceed the reservation; a Key balance may go negative, after which new requests fail closed.

Token accounting matches [cap-token-usage-tracker](https://github.com/AITNR/cap-token-usage-tracker): only Input, Output, Cache Read, and Cache Creation are billed. OpenAI-compatible `input`/`prompt_tokens` already include cache tokens, so those cache counters are subtracted before input is priced. Claude/Anthropic `input_tokens` exclude cache, so the four counters are billed independently. Reasoning and generic Cached counts are statistics only; missing `cache_read` falls back to `cached`. Official `total_tokens` is preferred for displayed totals.

Image-only models use `billing_mode: per_image`. They cannot reuse token prices. If usage is missing, image requests settle the reserved image count (default 1), not a token estimate.

Missing **token** usage does **not** bill `max_tokens` estimates. The ledger records zero cost. If official usage arrives later, the row is repriced. `settlement.missing_usage=release` drops the hold with no charge.

## Self-Service Key Dashboard

```text
http://<CPA_HOST>:8317/v0/resource/plugins/credit-manager/lookup
```

The page accepts the full `tk-...` Key and shows only that Key's data:

- Total/daily/weekly/monthly quota progress and concurrency. Period quotas use UTC.
- The "Today" filter is the **UTC** calendar day, unlike the admin console's local calendar day.
- Independent Token and cost trend grains (hour / day / month)
- Model usage share and efficiency ranking
- USD/CNY display conversion (display only; the ledger stays in micro-USD)

Security properties:

- Not listed in the CPA management sidebar; no host management token required
- The plugin Key is sent only in an `Authorization: Bearer` header, never in the URL or browser storage. Theme, language, and display-unit preferences are separate.
- Public responses exclude host accounts, caller IDs, plugin Key IDs, emails, and auth-file paths
- Expose the CPA port only on trusted networks; put HTTPS in front before making this page public

## Management Console

Sidebar entry **CPA Credit Manager**:

```text
/v0/resource/plugins/credit-manager/console
```

The page stores the host management token in `sessionStorage` and calls the management API.

| Tab | Contents |
|---|---|
| Overview | Filters by time, Key, upstream account, model, and source. "Today" is the browser's local calendar day, not the Key's UTC daily quota |
| Keys | Mint, reveal, rotate, enable/disable, revoke, delete; quotas, concurrency, allowlists |
| Models & pricing | Load proxied models, set token or per-image prices, context/`service_tier` cards, enable/disable models; optional models.dev public-price backfill |
| Usage | Same filters plus cost/Token ranges; per-Key and per-model summaries with pagination |
| Auth quotas | Upstream OAuth quota windows and local estimates |

The console enters and displays USD (CNY is display-only). Management API quota and price fields are **micro-USD** (`1 USD = 1,000,000`). Do not POST a console value like `10` as `10` micro-USD. Display FX is cached about 30 minutes and falls back to `7.2`.

## Management API

Management endpoints require the **host management token**, not a plugin Key. Paths are fixed; there is no `/keys/{id}`. The `id` returned at create is a management record ID. Model requests still use the full `tk-...` string.

| Method | Path | Purpose |
|---|---|---|
| GET | `/credit-manager/health` | Health check (includes plugin version) |
| GET | `/credit-manager/overview` | Console overview |
| POST / GET | `/credit-manager/callers` | Create or list ownership records |
| POST | `/credit-manager/callers/enabled` | Enable or disable a caller |
| POST / GET | `/credit-manager/keys` | Create or list Keys |
| POST | `/credit-manager/keys/update` | Update Key policy and limits |
| POST | `/credit-manager/keys/rotate` | Rotate a Key (new plaintext; old Key revoked and cannot be re-enabled) |
| POST | `/credit-manager/keys/reveal` | Reveal stored plaintext |
| POST | `/credit-manager/keys/revoke` | Revoke a Key (cannot be re-enabled) |
| POST | `/credit-manager/keys/delete` | Mark deleted (row kept; console shows Deleted) |
| POST / GET | `/credit-manager/pricing` | Create, update, or list pricing rules |
| POST | `/credit-manager/pricing/enabled` | Enable or disable a rule (and therefore a model) |
| POST | `/credit-manager/pricing/delete` | Delete a pricing rule |
| GET | `/credit-manager/balance?key_id=` | Get Key balance |
| GET | `/credit-manager/usage` | Query usage records (paginated) |
| GET | `/credit-manager/usage/summary` | Summarize usage by Key and model |
| GET | `/credit-manager/audit` | Query audit events |
| GET | `/credit-manager/auth-quotas` | Inspect OAuth quota windows |
| POST | `/credit-manager/auth-quotas/concurrency` | Set per-auth max concurrent requests |
| POST | `/credit-manager/auth-quotas/concurrency/batch` | Batch-set per-auth max concurrent requests |

Browser pages do not use host management auth. The console still asks for the management token; lookup still asks for a plugin Key. The last two URLs are for the pages themselves:

| Path | Purpose |
|---|---|
| `/v0/resource/plugins/credit-manager/console` | Admin console page |
| `/v0/resource/plugins/credit-manager/lookup` | Self-service page |
| `/v0/resource/plugins/credit-manager/lookup/data` | Self-service data (plugin Key required) |
| `/v0/resource/plugins/credit-manager/fx/usd-cny` | Display FX used by the UI |
| `/v0/resource/plugins/credit-manager/models-dev` | Public price catalog cache used by the UI |

Mint, rotate, reveal, and auth-quota responses set `Cache-Control: no-store`.

Callers organize Keys and usage. They do **not** hold quota.

### Create a Key

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "caller_id": "team-a",
    "label": "ci-bot",
    "total_quota_micro_usd": 10000000,
    "daily_quota_micro_usd": 1000000,
    "weekly_quota_micro_usd": 5000000,
    "monthly_quota_micro_usd": 10000000,
    "max_concurrent_requests": 3,
    "allowed_models": ["gpt-4o", "claude-*"]
  }'
```

`plaintext` is the client Key. `id` is the management record ID for update/reveal/revoke; it is not a Bearer token. List endpoints never return plaintext. Keys minted before recoverable storage must be rotated before reveal will work.

To change `allowed_models` on update, send `set_allowed_models: true`. An empty array means all models are allowed.

### Pricing

`match_kind` is `exact`, `glob`, or `regexp`. Higher `priority` wins; ties break on rule ID. **Disabled rules still participate in matching.** If the winning rule has `enabled: false`, the model is rejected and removed from `GET /v1/models` / `GET /v1beta/models`. The plugin also merges those IDs into host auth-file exclude lists, updating only the plugin-managed fields so the rest of the OAuth JSON is copied as-is.

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/pricing" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "gpt-4o",
    "match_kind": "exact",
    "pattern": "gpt-4o",
    "priority": 100,
    "enabled": true,
    "price": {
      "input": 2500000,
      "output": 10000000,
      "cache_read": 1250000,
      "accounting_mode": "input_includes_cache"
    }
  }'
```

Per-image example:

```json
{
  "id": "gpt-image-1",
  "match_kind": "exact",
  "pattern": "gpt-image-1",
  "priority": 100,
  "enabled": true,
  "price": { "billing_mode": "per_image", "per_image": 40000 }
}
```

Price fields: `input`, `output`, `reasoning`, `cached`, `cache_read`, `cache_creation`, optional `accounting_mode`, `billing_mode` (`token` or `per_image`), and `per_image`.

Optional `tiers` overlay the base card. Blank overlay rates inherit the default. Reservation only applies request `service_tier` cards, not crude token estimates against context thresholds.

- `kind: "context"`: used when actual input tokens reach `threshold`
- `kind: "service"`: used when the response `service_tier` matches `service` (for example `fast` or `priority`)

```json
"tiers": [
  {"kind":"context","label":"272K","threshold":272000,"price":{"input":400000,"output":1800000,"cache_read":40000}},
  {"kind":"service","service":"fast,priority","price":{"input":400000,"output":2400000}}
]
```

### Call a model

```bash
curl -sS "http://127.0.0.1:8317/v1/chat/completions" \
  -H "Authorization: Bearer tk-..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role":"user","content":"hello"}],
    "max_tokens": 256
  }'
```

Text/chat requests are reserved, forwarded, and settled from upstream usage or a later host usage callback. Image-only traffic (`/v1/images/*`, `gpt-image-*`) uses the host native image path: reserve first, settle when the request completes.

Quota failures are rejected before forwarding. A Key that lists allowed models rejects anything outside that list. Globally disabled models are also rejected and omitted from the public catalog.

## OAuth Auth Quotas

The **Auth Quotas** tab and `GET /v0/management/credit-manager/auth-quotas` are management-authenticated, return `Cache-Control: no-store`, and are not available from the public lookup page.

- Supported collectors: Codex/ChatGPT, Claude, Antigravity, Kimi, and xAI OAuth. API-key-only credentials are excluded.
- Sampling uses the CLIProxyAPI host HTTP callback and its outbound transport. Credential-file `proxy_url` is not applied to this management callback.
- Refreshes are limited to one attempt per credential every 15 minutes. A failed refresh keeps the last successful snapshot as `stale`; never-sampled credentials are `unavailable`.
- Quota windows keep their native upstream units (percent, requests, credits, USD). Local request estimates appear only where the ledger can safely attribute the window. Usage from other clients, websites, or proxy nodes can reduce remaining calls.
- Each card can set a max concurrent request cap for that credential; `0` or omitted means unlimited. The toolbar can apply the same cap to the current page or to the current filter. When a cap is set, the plugin scheduler skips busy accounts and rejects the request if every candidate is full. In-flight counts are unsettled or unreleased requests already attributed to that auth.
- Snapshots and responses exclude OAuth tokens, upstream account identifiers, auth-file paths, proxy URLs, and raw upstream bodies.

## Configuration

See [`config.example.yaml`](config.example.yaml).

| Field | Description |
|---|---|
| `data_dir` | Plugin data directory for SQLite, lock files, pepper, and models.dev cache |
| `database_file` | SQLite filename; default `credit-manager.db` |
| `busy_timeout` | SQLite busy timeout, for example `5s` |
| `keys.pepper_env` | Optional pepper environment variable name |
| `keys.pepper_file` | Pepper file, relative to `data_dir` or absolute |
| `keys.active_pepper_id` | Pepper ID used to issue new Keys |
| `limits.max_token_estimate` | Maximum request Token estimate |
| `limits.default_output_reserve` | Default output reservation when `max_tokens` is absent |
| `limits.require_estimate` | Reject requests that cannot be estimated when `true` |
| `pricing.unknown_policy` | `deny`, `allow`, or `default` for unmatched models |
| `pricing.default` | Required only when `unknown_policy` is `default` |
| `settlement.missing_usage` | `release` drops the hold. `settle_reserved` records 0 for text (and waits for a later callback) and bills reserved image count for image models. The name does not mean "charge the reservation" |
| `settlement.host_usage_wait` | How long to wait for a host usage callback before fallback; default `1500ms` |
| `stream.max_buffer_bytes` | Local stream settlement buffer limit |
| `stream.stale_reservation_timeout` | Release threshold for holds without a heartbeat; default `2h` |

Streaming and non-streaming requests refresh their reservation heartbeat while active. Stale reservations are reclaimed during startup, configuration reload, and periodically before new reservations.

## Security Notes

- Callers treat the issued `tk-...` value as an opaque string.
- SQLite does not store plaintext. Reveal uses pepper-derived ciphertext. Losing the pepper invalidates existing Keys.
- Pepper material lives in `data_dir/key-peppers` or an environment variable, never in SQLite or ordinary management queries.
- Back up the entire `data_dir`, including `key-peppers`.
- Overwriting a loaded Windows DLL still requires a CLIProxyAPI restart. Plugin-store upgrades write a versioned file and, since `1.4.0`, hand over the SQLite lock to the new instance. The first upgrade from an older build needs one uninstall/reinstall or host restart.
- Auth-file OAuth material, upstream quota responses, and auth-file paths are not written to quota snapshots or exposed through the public lookup endpoint.

Pepper rotation: append the new pepper, set `active_pepper_id`, restart, mint new Keys, and keep the old pepper until every Key that needs it has been retired.

## Request path

```text
Client
  |  Authorization: Bearer tk-...
  v
CLIProxyAPI
  |  frontend_auth.authenticate  -> plugin verifies the Key
  |  GET /v1/models              -> directory without a plugin Key; globally disabled models omitted
  |  model.route                 -> text/chat to the plugin executor
  |                              -> image-only stays on the host native path
  |  executor.execute / stream   -> reserve, forward, settle
  |  request interceptors        -> image reserve / completion settle
  |  host usage callback         -> backfill official usage
  v
Upstream model
```

The host skips this plugin as an upstream executor to avoid recursion. Supported formats include openai, chat-completions, claude, gemini, openai-response, responses, codex, openai-image, and openai-video.

## Operations

1. One writer per database. The plugin takes an exclusive `*.lock`.
2. Back up all of `data_dir`.
3. Direct DLL overwrite still needs a host restart. Store upgrades since `1.4.0` hand over the DB lock; the first jump from an older build needs uninstall/reinstall or restart.
4. Enabling the plugin disables host `api-keys` for proxied model auth.
5. Zero-config allows unknown models at $0. Set real prices in production; use `deny` if unmatched models must be blocked.
6. The sidebar entry is registered as a plugin Resource, not inferred from API routes.

## Verification

```bash
go test ./...
go test -race ./internal/store ./internal/service
go vet ./...
go run ./scripts/smoke_ledger.go
```

`smoke_ledger` creates a caller, mints a Key with quota, authenticates, reserves, settles, and revokes in a temporary directory.

## FAQ

**`gcc not found`:** install MinGW-w64, MSYS2 gcc, or Zig. `scripts/build.ps1` tries gcc, then `zig cc`.

**Management 401:** use the host management token, not a plugin Key.

**Proxy 401:** send the full issued `tk-...` Key and keep the same pepper that minted it.

**`no pricing rule`:** restore a pricing rule, or set `unknown_policy` to `allow` / `default`.

**Disabled model still listed:** disable it under Models & pricing, deploy the latest plugin, and restart. A Key allowlist does not hide models from the public catalog.

**Sidebar missing:** enable the plugin and deploy a binary that includes the console Resource.

**Management shows unregistered after a store upgrade:** `1.4.0+` hands over `*.db.lock`. The first upgrade from an older binary needs uninstall/reinstall or a host restart.

**Negative Key balance:** actual token usage exceeded the reservation. Later requests fail until quota is raised.

**Did missing usage bill `max_tokens`?** Not for text. The default records zero; a later official usage callback can reprice the row. Image requests settle per reserved image count. `settlement.missing_usage=release` drops the hold with no charge.

**Reveal failed:** the Key predates recoverable storage. Rotate it to mint a viewable replacement and revoke the old Key.

**Deleted Key still listed?** Delete does not erase the database row. It revokes the Key and the console labels it Deleted so usage history remains. Disable can be turned back on; revoke/delete cannot authenticate with the old plaintext.
