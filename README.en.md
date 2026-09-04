# CPA Credit Manager

[中文文档](README.md) | [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) | [Repository](https://github.com/yuluo688/credit-manager) | [Download latest release](https://github.com/yuluo688/credit-manager/releases/latest)

`credit-manager` is a native Go plugin for [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI). It keeps plugin-key authentication, credit reservation, usage settlement, and analytics in one request path, so teams, users, and automated workloads can receive independent, billable model access Keys.

| Plugin ID | Runtime |
|---|---|
| `credit-manager` | CGO `c-shared` library |

## What It Solves

At high concurrency, a host's upstream credentials, frontend `api-keys`, and usage callbacks may not reliably identify the same caller. This plugin ties identity, reservation, and settlement to the same `tk-...` Key:

- Configure independent total, daily, weekly, monthly, concurrency, model-access, and expiry policies for every Key.
- Strictly reserve a conservative amount before forwarding. Requests that exceed spend or concurrency limits never reach the upstream provider.
- Settle text traffic from actual upstream token usage and image-only traffic per image. Pricing supports model rules, context tiers, and `service_tier` tiers.
- Operate Keys, prices, usage analytics, audits, and auth quotas through the management console and API. Key holders have a separate self-service page.
- Inspect upstream quota windows for Codex, Claude, Antigravity, Kimi, and xAI OAuth credentials, and set per-auth concurrency limits.

> When enabled, the plugin owns frontend authentication for proxied model requests. Existing CLIProxyAPI `api-keys` no longer authenticate model calls; clients must use the full issued `tk-...` Key.

```text
Authorization: Bearer tk-...
```

`GET /v1/models` and `GET /v1beta/models` remain anonymous for CLIProxyAPI Management Center compatibility. They only return the public model catalog and cannot execute models. Globally disabled models are omitted; a Key allowlist does not trim this public catalog.

## Quick Start

### 1. Prepare the environment

| Component | Requirement |
|---|---|
| CLIProxyAPI | `v7.2.128+` recommended, with working upstream credentials |
| Go | `1.26+` |
| C toolchain | `gcc` or `clang`; Windows also supports Zig's `zig cc` |
| Data directory | `./data/credit-manager` by default; writable by the host process |

### 2. Build and deploy

**Windows PowerShell**

```powershell
# Build the DLL
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1

# Build and deploy to D:\CLIProxyAPI\plugins\windows\amd64
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1
```

You can choose another deployment directory or reuse an existing build:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1 `
  -DestDir "E:\CLIProxyAPI\plugins\windows\amd64"

powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1 -SkipBuild
```

**Linux / macOS**

```bash
chmod +x scripts/build.sh
./scripts/build.sh
# Linux: dist/credit-manager.so
# macOS: dist/credit-manager.dylib
```

Copy the library to the appropriate CLIProxyAPI plugins directory and enable `credit-manager` in Plugin Management. Windows normally locks a loaded DLL, so restart the host after directly replacing it.

### 3. Enable the plugin and mint the first Key

The host configuration can be as small as this; the plugin itself supports zero-config startup:

```yaml
plugins:
  enabled: true
  dir: ./plugins
  items:
    credit-manager:
      enabled: true
```

On first start, the plugin creates `data_dir/key-peppers`, the `default` ownership record, and the `bootstrap-all-models` free pricing rule. It **does not** create a plugin Key. Replace the free rule with real production pricing.

Open **CPA Credit Manager** in the CLIProxyAPI sidebar, enter the host management token, and create a Key with limits and prices. The same operation is available through the API:

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "caller_id": "default",
    "label": "first-key",
    "total_quota_micro_usd": 10000000,
    "daily_quota_micro_usd": 1000000,
    "max_concurrent_requests": 3,
    "allowed_models": ["gpt-4o", "claude-*"]
  }'
```

The response field `plaintext` is the full `tk-...` string for the client. Its `id` is a management record ID only and cannot be used as a Bearer token.

### 4. Call a model

```bash
curl -sS "http://127.0.0.1:8317/v1/chat/completions" \
  -H "Authorization: Bearer tk-..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "hello"}],
    "max_tokens": 256
  }'
```

## Management UI

| Page | URL | Access and purpose |
|---|---|---|
| Management console | `/v0/resource/plugins/credit-manager/console` | Enter the host management token to manage Keys, pricing, usage, and auth quotas |
| Key self-service | `/v0/resource/plugins/credit-manager/lookup` | A Key holder enters their own `tk-...` Key and can view only that Key's quota and usage |

The console accepts and displays USD. Switching to CNY affects display only. Every management API amount is an integer in **micro-USD**: `1 USD = 1_000_000 micro-USD`. For example, a console value of `$10` must be submitted to the API as `10000000`, not `10`.

| Console tab | Contents |
|---|---|
| Overview | Aggregates and filters by time, Key, upstream account, model, and source. “Today” uses the browser's local calendar day. |
| Keys | Mint, reveal, rotate, disable, revoke, delete, and reset spend; configure credit limits, concurrency, model access, and Token caps. |
| Models & pricing | Load current proxy models, set token or per-image prices, and enable or disable models and rules. |
| Usage | Paginated detail and summaries with Key, model, and range filters. |
| Auth quotas | OAuth upstream quota windows, local usage estimates, and auth concurrency caps. |

The self-service page is not listed in the management sidebar and needs no host management token. The plugin Key is sent only in the current request's `Authorization` header, never in the URL or browser storage. Public responses exclude caller IDs, auth accounts, emails, and auth-file paths.

## Limits and Settlement

### Key spend limits

Every Key supports the following independent fields. `0` or an omitted value means unlimited.

| Field | Meaning |
|---|---|
| `total_quota_micro_usd` | Total spend limit; `quota_micro_usd` is a compatibility alias |
| `daily_quota_micro_usd` | UTC calendar-day spend limit |
| `weekly_quota_micro_usd` | UTC calendar-week spend limit, beginning Monday |
| `monthly_quota_micro_usd` | UTC calendar-month spend limit |
| `max_concurrent_requests` | Maximum number of concurrently held requests |
| `allowed_models` | Empty array permits every model; otherwise exact/glob patterns |
| `expires_at` | Optional RFC3339 expiration time |
| `unmatched_models_mode` | With `disabled`, models without a matching Token rule are unavailable |

Period limits include settled spend and active reservations. A request first reserves a conservative Token or image amount, then settles from actual usage. Actual settlement can exceed the reservation, yielding a negative balance; later requests fail closed until the limit is restored.

Day, week, and month limits use UTC, while the management console's “Today” uses the browser's local calendar day. Resetting total, daily, weekly, or monthly used spend retains the ledger and limit values; it restarts accumulation from the reset point until the next UTC boundary.

### Per-model Token caps

A Key can apply hard total, daily, weekly, and monthly Token caps to exact model names or glob patterns:

```json
{
  "set_model_token_limits": true,
  "model_token_limits": [
    {
      "model": "gpt-4o*",
      "total": {"tokens": 5000000, "mode": "available"},
      "daily": {"tokens": 300000, "mode": "available"},
      "weekly": {"tokens": 0, "mode": "unlimited"},
      "monthly": {"tokens": 0, "mode": "unlimited"}
    }
  ]
}
```

An exact model name wins. If several globs match, the longest pattern wins. Token caps are checked during reservation, using both settled tokens and in-flight estimates. Resetting total Token spend starts a new total count from the reset time.

### Pricing rules

Text prices are integer **micro-USD per 1M Tokens**. Image-only models use `billing_mode: "per_image"`; `per_image` is the micro-USD price per image.

```json
{
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
}
```

`match_kind` accepts `exact`, `glob`, and `regexp`. Higher priority wins; ties are ordered by rule ID. Disabled rules still participate in matching: if the highest-priority matching rule is disabled, the model request is rejected and removed from the public catalog.

`tiers` overlay the base price. A `context` tier activates when actual input tokens reach `threshold`; a `service` tier activates from response `service_tier`. Reservation uses only a request-visible service tier and never applies a context tier from a rough Token estimate.

When a text request never yields usable usage, the default does not charge its `max_tokens` estimate: it records zero and can later be repriced from an official host usage callback. Image-only traffic without usage settles from the reserved image count. Set `settlement.missing_usage: release` to release the hold without a charge.

## Management API

Every management endpoint requires the **host management token**. The base path is:

```text
http://<CPA_HOST>:8317/v0/management/credit-manager
```

Endpoints do not use `/keys/{id}` path parameters. Pass management record IDs through JSON or query parameters instead.

| Method | Relative path | Purpose |
|---|---|---|
| GET | `/health` | Health check and plugin version |
| GET | `/overview` | Console overview data |
| POST / GET | `/callers` | Create or list Key ownership records |
| POST | `/callers/enabled` | Enable or disable an ownership record |
| POST / GET | `/keys` | Mint or list Keys |
| POST | `/keys/update` | Update Key policy, limits, models, and expiry |
| POST | `/keys/rotate` | Rotate a Key; the old Key is immediately revoked |
| POST | `/keys/reveal` | Reveal recoverably stored Key plaintext |
| POST | `/keys/revoke` | Revoke a Key permanently |
| POST | `/keys/delete` | Mark deleted while retaining ledger history |
| POST | `/keys/reset-spend` | Reset total, daily, weekly, or monthly used spend without deleting the ledger |
| POST / GET | `/pricing` | Create, update, or list pricing rules |
| POST | `/pricing/enabled` | Enable or disable a price rule and its model |
| POST | `/pricing/delete` | Delete a pricing rule |
| GET | `/balance?key_id=` | Get Key balance and limits |
| GET | `/usage` | Query paginated usage records |
| GET | `/usage/summary` | Summarize usage by Key and model |
| GET | `/audit` | Query audit events |
| GET | `/auth-quotas` | Inspect OAuth auth quota windows |
| POST | `/auth-quotas/refresh` | Refresh auth-quota snapshots |
| POST | `/auth-quotas/concurrency` | Set one auth's max concurrency |
| POST | `/auth-quotas/concurrency/batch` | Batch-set auth max concurrency |

Key mint, rotation, and reveal responses, plus auth-quota responses, include `Cache-Control: no-store`.

## Configuration and Security

See [`config.example.yaml`](config.example.yaml) for the full example. Point the host configuration to an external file, or set an environment variable:

```yaml
plugins:
  items:
    credit-manager:
      enabled: true
      config: |
        config_file: E:/path/to/credit-manager.yaml
```

```bash
export CREDIT_MANAGER_CONFIG_FILE=/path/to/credit-manager.yaml
```

Host inline fields override the external file. A nested `config_file` in that external file is rejected.

| Setting | Default | Purpose |
|---|---|---|
| `data_dir` | `./data/credit-manager` | SQLite, locks, pepper, and public-price cache |
| `database_file` | `credit-manager.db` | SQLite file name |
| `keys.pepper_env` | `CREDIT_MANAGER_KEY_PEPPERS` | Preferred pepper environment variable |
| `keys.pepper_file` | `key-peppers` | Pepper file relative to `data_dir` |
| `keys.active_pepper_id` | First pepper | Pepper ID for new Keys |
| `limits.max_token_estimate` | `1000000` | Maximum request Token estimate |
| `limits.default_output_reserve` | `4096` | Output reservation when `max_tokens` is absent |
| `pricing.unknown_policy` | `allow` | Behavior for unmatched rules: `allow`, `deny`, or `default` |
| `settlement.missing_usage` | `settle_reserved` | Settlement behavior when usage is missing |
| `settlement.host_usage_wait` | `4s` | Time to await a host usage callback; the sample config overrides it to `1500ms` |
| `stream.stale_reservation_timeout` | `2h` | Release threshold for in-flight reservations without a heartbeat |

### Protect pepper material

Pepper material verifies Key HMACs and encrypts recoverable plaintext. It is never stored in SQLite, logs, or ordinary management responses. The resolution order is a non-empty `CREDIT_MANAGER_KEY_PEPPERS`, then `data_dir/key-peppers`, then a new 32-byte random value on first start.

Back up the entire `data_dir`, especially `key-peppers`. Losing pepper material invalidates old Keys and prevents later plaintext reveal. To rotate pepper, add the new material, set `active_pepper_id`, restart the host, mint new Keys, then remove the old pepper only after dependent Keys are retired.

### OAuth auth quotas

Auth quotas are management-only. Supported OAuth collectors are Codex/ChatGPT, Claude, Antigravity, Kimi, and xAI; API-key-only credentials are excluded. Each credential is refreshed at most once per 15 minutes. A failed refresh retains the last successful snapshot and marks it `stale`. Snapshots and responses exclude OAuth tokens, upstream account IDs, auth-file paths, proxy URLs, and raw upstream payloads.

## Operations and Verification

- A SQLite database permits one writer process. The plugin protects it with an exclusive `*.lock`.
- Configure real prices in production. The all-models free rule exists only for first-run convenience.
- CLIProxyAPI continues to own upstream OAuth and API-key login; this plugin does not perform provider login.
- Before exposing self-service lookup on the internet, limit network exposure and place it behind an HTTPS reverse proxy.

```bash
go test ./...
go test -race ./internal/store ./internal/service
go vet ./...
go run ./scripts/smoke_ledger.go
```

`smoke_ledger` verifies Key minting, authentication, reservation, actual-usage settlement, and revocation in a temporary directory.

## Troubleshooting

| Issue | Resolution |
|---|---|
| `gcc not found` while building | Install MinGW-w64, MSYS2 GCC, or Zig. The Windows build script tries `gcc` first, then `zig cc`. |
| Management API returns 401 | Use the host management token, not a `tk-...` Key. |
| Model call returns 401 | Send the complete issued `tk-...` and ensure its original pepper material was not lost or replaced. |
| `no pricing rule` | Add a pricing rule, or set `pricing.unknown_policy` to `allow` or `default`. |
| Key has a negative balance | Actual usage exceeded the reservation. Raise its quota before further calls. |
| Does missing usage charge `max_tokens`? | Not for text by default. It records zero and can later be repriced by an official usage callback. |
| The sidebar entry is missing | Confirm the plugin is enabled, the deployed library contains Resources, and the host was restarted after a DLL replacement. |
| An old Key cannot be revealed | It may predate recoverable storage. Rotate it to mint a revealable replacement. |
