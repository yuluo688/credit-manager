# CPA Credit Manager (`credit-manager`)

[中文文档](README.md)

A native Go plugin for [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI). It provides dedicated plugin-key authentication, Key-level credit controls, usage-based settlement, and a self-service usage dashboard.

Plugin ID: `credit-manager`

## Features

1. Dedicated `tk-...` plugin-key authentication independent of host `api-keys`.
2. Strict Key-level reservation before a request is forwarded.
3. Settlement from actual upstream Token usage.
4. Per-Key total, daily, weekly, monthly, and concurrency limits.
5. Management console and API for Keys, models, pricing, usage, and audit history.
6. A public self-service dashboard where a Key holder can view only their own quota and usage.

## Quick Start

1. Build and deploy the shared library. On Windows, run:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1
```

This deploys to `D:\CLIProxyAPI\plugins\windows\amd64` by default.

2. Enable `credit-manager` in CLIProxyAPI Plugin Management.
3. Restart CLIProxyAPI after replacing the DLL. Windows cannot normally hot-replace a loaded DLL.
4. Open **CPA Credit Manager** in the CPA sidebar, enter the host management token, create a Key, and set its quota, concurrency, allowed models, and pricing rules.
5. Call models with the generated Key:

```text
Authorization: Bearer tk-...
```

> When enabled, this plugin owns frontend authentication for proxied model requests. Existing CPA `api-keys` no longer authenticate model requests. Create and use `tk-...` plugin Keys instead.

> For CPA Management Center compatibility, anonymous `GET /v1/models` is allowed. This narrow exception exposes only the model directory; it does not permit model execution or management API access.

## Key Limits

Every Key can have the following independent limits. `0` or an omitted field means unlimited.

| Field | Meaning |
|---|---|
| `total_quota_micro_usd` | Total spend limit |
| `daily_quota_micro_usd` | UTC calendar-day limit |
| `weekly_quota_micro_usd` | UTC calendar-week limit, starting Monday |
| `monthly_quota_micro_usd` | UTC calendar-month limit |
| `max_concurrent_requests` | Maximum held/in-flight requests |

Period limits include settled usage plus active reservations. The system reserves a conservative amount before forwarding a request, then settles using actual usage. Amounts are stored as integer micro-USD: `1 USD = 1,000,000 micro-USD`.

## Self-Service Key Dashboard

Key holders can view their own quota and usage without CPA management login:

```text
http://<CPA_HOST>:8317/v0/resource/plugins/credit-manager/lookup
```

The page accepts a `tk-...` Key and shows only that Key's data:

- Total/daily/weekly/monthly quota progress and concurrency.
- Filterable request, Token, and cost summaries.
- Token and cost trends.
- Model usage share, with request / Token / cost metrics.
- Paginated recent usage records.

Security properties:

- The page is not listed in the CPA management sidebar and does not require the host management token.
- The Key is sent only in an `Authorization: Bearer` request header. It is not placed in the URL or persisted in browser/page storage.
- The public response excludes host account details, caller IDs, plugin Key IDs, emails, and auth-file paths.
- Expose the CPA port only on trusted networks. Use an HTTPS reverse proxy before making this page externally reachable.

## Management API

Management endpoints require the **host management token**, not a plugin Key. Base path:

```text
/v0/management/credit-manager
```

| Method | Path | Purpose |
|---|---|---|
| GET | `/health` | Health check |
| GET | `/overview` | Console overview |
| POST / GET | `/callers` | Create or list ownership records |
| POST | `/callers/enabled` | Enable or disable a caller |
| POST / GET | `/keys` | Create or list Keys |
| POST | `/keys/update` | Update Key policy and limits |
| POST | `/keys/revoke` | Revoke a Key |
| POST / GET | `/pricing` | Create, update, or list pricing rules |
| POST | `/pricing/delete` | Delete a pricing rule |
| GET | `/balance?key_id=` | Get Key balance |
| GET | `/usage` | Query usage records |
| GET | `/usage/summary` | Summarize usage by Key and model |
| GET | `/audit` | Query audit events |

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
    "max_concurrent_requests": 3
  }'
```

The plaintext Key is returned only when the Key is created. Store it securely.

## Configuration

See [`config.example.yaml`](config.example.yaml) for the complete example.

| Field | Description |
|---|---|
| `data_dir` | Plugin data directory for SQLite, lock files, and pepper material |
| `database_file` | SQLite database filename; default `credit-manager.db` |
| `busy_timeout` | SQLite busy timeout, for example `5s` |
| `keys.pepper_env` | Optional pepper environment variable name |
| `keys.pepper_file` | Pepper file, relative to `data_dir` or absolute |
| `keys.active_pepper_id` | Pepper ID used to issue new Keys |
| `limits.max_token_estimate` | Maximum request Token estimate |
| `limits.default_output_reserve` | Default output reservation when `max_tokens` is absent |
| `pricing.unknown_policy` | `deny`, `allow`, or `default` for unmatched models |
| `settlement.missing_usage` | `settle_reserved` or `release` when upstream omits usage |
| `stream.max_buffer_bytes` | Local stream settlement buffer limit |
| `stream.stale_reservation_timeout` | Release threshold for holds without a heartbeat; default `2h` |

Streaming and non-streaming requests refresh their reservation heartbeat while active. Stale reservations are reclaimed during startup, configuration reload, and periodically before new reservations.

## Security Notes

- Plaintext format: `tk-<kid>-<secret>`.
- SQLite stores the Key ID, HMAC, pepper ID, fingerprint, principal, and caller scope, not plaintext Key material.
- Pepper material is stored in `data_dir/key-peppers` or an environment variable, never in SQLite or management responses.
- Back up the entire `data_dir`, including `key-peppers`. Losing the pepper invalidates existing Keys.
- Replacing a Windows DLL requires a CLIProxyAPI restart.

## Verification

```bash
go test ./...
go test -race ./internal/store ./internal/service
go vet ./...
go run ./scripts/smoke_ledger.go
```
