# CPA Credit Manager

[English](README.en.md) | [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) | [项目仓库](https://github.com/yuluo688/credit-manager) | [下载最新版本](https://github.com/yuluo688/credit-manager/releases/latest)

`credit-manager` 是 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 的原生 Go 插件。它把插件 Key 鉴权、额度预占、实际用量结算和使用分析放在同一条请求链路中，适合为团队、用户或自动化任务分发独立、可计费的模型访问 Key。

| 插件 ID | 运行形态 |
|---|---|
| `credit-manager` | CGO `c-shared` 动态库 |

## 它解决什么问题

宿主的上游认证、前端 `api-keys` 与 usage 回调在高并发下未必能稳定关联到同一个调用方。此插件将身份、预占和结算绑定到同一个 `tk-...` Key：

- 为每个 Key 分别配置总、日、周、月消费额度、最大并发、模型访问范围和过期时间。
- 在请求转发前按保守估算严格预占；额度或并发不足时直接拒绝，不让超额请求进入上游。
- 根据上游实际 usage 结算文本 Token；纯出图按张结算。价格支持模型规则、上下文档位和 `service_tier` 档位。
- 提供管理控制台、管理 API、审计和使用统计；Key 持有人还可在独立页面查询自己的额度与明细。
- 汇总 Codex、Claude、Antigravity、Kimi、xAI OAuth 认证的上游额度窗口，并可按认证限制最大并发。

> 启用后，插件接管模型代理请求的前端鉴权。原有 CLIProxyAPI `api-keys` 不能再用于模型调用；客户端必须携带签发出的完整 `tk-...` Key。

```text
Authorization: Bearer tk-...
```

`GET /v1/models` 和 `GET /v1beta/models` 可匿名获取公共模型目录，以兼容宿主管理中心；它们不能调用模型。全局禁用的模型会从目录移除，但某个 Key 的模型白名单不会裁剪公共目录。

## 快速开始

### 1. 准备环境

| 组件 | 要求 |
|---|---|
| CLIProxyAPI | 建议 `v7.2.128+`，且上游认证已可正常调用模型 |
| Go | `1.26+` |
| C 工具链 | `gcc` 或 `clang`；Windows 也支持 Zig 的 `zig cc` |
| 数据目录 | 默认 `./data/credit-manager`，宿主进程必须可写 |

### 2. 构建并部署

**Windows PowerShell**

```powershell
# 编译 DLL
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1

# 编译并部署至默认目录 D:\CLIProxyAPI\plugins\windows\amd64
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1
```

也可以指定部署目录，或跳过已完成的编译：

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

将产物复制到 CLIProxyAPI 的对应插件目录，在插件管理中启用 `credit-manager`。Windows 通常锁定已加载的 DLL，因此直接替换 DLL 后必须重启宿主。

### 3. 启用并签发第一个 Key

宿主的最小配置如下，插件自身可以零配置启动：

```yaml
plugins:
  enabled: true
  dir: ./plugins
  items:
    credit-manager:
      enabled: true
```

首次启动会自动创建 `data_dir/key-peppers`、`default` 归属记录、名为 `bootstrap-all-models` 的全模型免费规则。**不会自动签发插件 Key**，生产环境也必须替换免费规则为真实价格。

在 CLIProxyAPI 侧栏打开 **CPA 额度管理**，输入宿主管理密钥，然后创建 Key、设置限额和价格。也可以调用管理 API：

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

响应中的 `plaintext` 是要交给客户端的完整 `tk-...` 字符串；`id` 仅用于后续管理操作，不能作为 Bearer Token。

### 4. 调用模型

```bash
curl -sS "http://127.0.0.1:8317/v1/chat/completions" \
  -H "Authorization: Bearer tk-..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role": "user", "content": "你好"}],
    "max_tokens": 256
  }'
```

## 管理界面

| 页面 | 地址 | 权限与用途 |
|---|---|---|
| 管理控制台 | `/v0/resource/plugins/credit-manager/console` | 输入宿主管理密钥后管理 Key、价格、使用统计和认证额度 |
| Key 自助查询 | `/v0/resource/plugins/credit-manager/lookup` | Key 持有人输入自己的 `tk-...` 后，只能查看自己的额度与使用情况 |

控制台的金额输入和显示单位是 USD，切换到 CNY 仅影响显示。管理 API 所有金额均为整数 **micro-USD**，即 `1 USD = 1_000_000 micro-USD`。例如控制台的 `$10` 在 API 中应传 `10000000`，不能传 `10`。

| 控制台标签 | 内容 |
|---|---|
| 概览 | 按时间、Key、上游账号、模型和来源汇总；“今日”使用浏览器本地日历日 |
| 密钥管理 | 签发、揭示、轮换、禁用、撤销、删除、重置额度；配置消费额度、并发、模型范围与 Token 上限 |
| 模型与价格 | 读取当前代理模型，设置 Token 或按张价格，启停模型与价格规则 |
| 使用统计 | 按 Key、模型和多种条件筛选的汇总及分页明细 |
| 认证额度 | OAuth 认证的上游额度窗口、本地使用预测和认证并发上限 |

自助查询页不会出现在管理侧栏，不需要宿主管理密钥。插件 Key 仅作为当前请求的 `Authorization` 头发送，不写入 URL 或浏览器存储；公开响应不包含 caller ID、认证账号、邮箱或认证文件路径。

## 额度与结算

### Key 消费额度

每个 Key 可独立设置以下字段。`0` 或省略表示不限制。

| 字段 | 说明 |
|---|---|
| `total_quota_micro_usd` | 总消费额度；`quota_micro_usd` 是兼容别名 |
| `daily_quota_micro_usd` | UTC 自然日消费额度 |
| `weekly_quota_micro_usd` | UTC 自然周消费额度，周一开始 |
| `monthly_quota_micro_usd` | UTC 自然月消费额度 |
| `max_concurrent_requests` | 同时处于预占状态的最大请求数 |
| `allowed_models` | 空数组代表全部模型；否则为 exact/glob 模式 |
| `expires_at` | 可选 RFC3339 过期时间 |
| `unmatched_models_mode` | 设为 `disabled` 时，未匹配模型 Token 规则的模型不可用 |

周期额度计算为已结算费用加在途预占。请求会先按保守 Token 上限或出图张数预占，随后按真实 usage 结算。实际结算可能超过预占，此时余额可为负；之后请求会 fail-closed，直到额度恢复。

日、周、月额度按 UTC 统计，而管理控制台中的“今日”按浏览器本地日历日。可以重置总、日、周、月已用额度，不会删除账本记录或修改限额；周期在下一 UTC 边界前从重置时刻开始重新累计。

### 模型 Token 上限

一个 Key 可按模型精确名或 glob 配置总、日、周、月 Token 硬上限：

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

匹配规则时精确模型名优先；多个 glob 命中时，较长模式优先。Token 限制在预占时检查，已结算 Token 与在途请求估算都会计入。总 Token 重置后从重置时刻重新累计。

### 定价规则

文本模型价格的单位是 **每 1M Token 的 micro-USD**。纯出图模型使用 `billing_mode: "per_image"`，`per_image` 表示每张图的 micro-USD。

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

`match_kind` 支持 `exact`、`glob` 和 `regexp`。优先级高的规则优先，同优先级按规则 ID 排序。禁用规则仍参与匹配：最高优先级命中的禁用规则会拒绝模型请求，并从公共目录中移除该模型。

`tiers` 可在默认价格上叠加：`context` 档位依据实际输入 Token 达到的 `threshold` 生效，`service` 档位依据响应的 `service_tier` 生效。预占只使用请求中可识别的 service 档位，不会用粗略 Token 估算套用上下文档位。

文本请求若始终没有可用 usage，默认不按 `max_tokens` 扣费，先记为 0，后续收到宿主的官方 usage 回调后再补记。纯出图在缺 usage 时按预占图数结算。设置 `settlement.missing_usage: release` 可直接释放预占而不记账。

## 管理 API

所有管理接口都使用**宿主管理密钥**，基础路径为：

```text
http://<CPA_HOST>:8317/v0/management/credit-manager
```

接口没有 `/keys/{id}` 形式的路径参数，管理记录 ID 通过 JSON 或查询参数传递。

| 方法 | 相对路径 | 用途 |
|---|---|---|
| GET | `/health` | 健康检查与插件版本 |
| GET | `/overview` | 控制台概览数据 |
| POST / GET | `/callers` | 创建或列出 Key 归属记录 |
| POST | `/callers/enabled` | 启用或禁用归属记录 |
| POST / GET | `/keys` | 签发或列出 Key |
| POST | `/keys/update` | 更新 Key 策略、限额、模型和过期时间 |
| POST | `/keys/rotate` | 轮换 Key；旧 Key 会立即撤销 |
| POST | `/keys/reveal` | 揭示可恢复保存的 Key 明文 |
| POST | `/keys/revoke` | 撤销 Key，不能重新启用 |
| POST | `/keys/delete` | 标记删除，保留账本历史 |
| POST | `/keys/reset-spend` | 重置总、日、周、月已用额度，不删除账本 |
| POST / GET | `/pricing` | 创建、更新或列出价格规则 |
| POST | `/pricing/enabled` | 启停价格规则及对应模型 |
| POST | `/pricing/delete` | 删除价格规则 |
| GET | `/balance?key_id=` | 查询 Key 余额与限额 |
| GET | `/usage` | 查询分页用量明细 |
| GET | `/usage/summary` | 按 Key 与模型汇总用量 |
| GET | `/audit` | 查询审计事件 |
| GET | `/auth-quotas` | 查询 OAuth 认证额度窗口 |
| POST | `/auth-quotas/refresh` | 刷新认证额度快照 |
| POST | `/auth-quotas/concurrency` | 设置单个认证最大并发 |
| POST | `/auth-quotas/concurrency/batch` | 批量设置认证最大并发 |

签发、轮换、揭示 Key，以及认证额度响应均带 `Cache-Control: no-store`。

## 配置与安全

完整示例见 [`config.example.yaml`](config.example.yaml)。可以在宿主插件配置中指定外部文件，也可以设置环境变量：

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

宿主内联字段覆盖外部文件；外部文件内的嵌套 `config_file` 会被拒绝。

| 配置项 | 默认值 | 作用 |
|---|---|---|
| `data_dir` | `./data/credit-manager` | SQLite、锁文件、pepper、价格目录缓存 |
| `database_file` | `credit-manager.db` | SQLite 文件名 |
| `keys.pepper_env` | `CREDIT_MANAGER_KEY_PEPPERS` | 优先读取的 pepper 环境变量 |
| `keys.pepper_file` | `key-peppers` | 相对 `data_dir` 的 pepper 文件 |
| `keys.active_pepper_id` | 第一个 pepper | 新 Key 使用的 pepper ID |
| `limits.max_token_estimate` | `1000000` | 单请求最大 Token 估算值 |
| `limits.default_output_reserve` | `4096` | 未提供 `max_tokens` 时的输出预占 |
| `pricing.unknown_policy` | `allow` | 未匹配价格规则时的策略：`allow`、`deny`、`default` |
| `settlement.missing_usage` | `settle_reserved` | 缺 usage 时的结算策略 |
| `settlement.host_usage_wait` | `4s` | 等待宿主 usage 回调的时长；示例配置覆盖为 `1500ms` |
| `stream.stale_reservation_timeout` | `2h` | 无心跳在途预占的自动释放阈值 |

### Pepper 保护

pepper 用于 Key 的 HMAC 校验和可恢复明文加密，绝不会写入 SQLite、日志或普通管理响应。解析顺序是非空 `CREDIT_MANAGER_KEY_PEPPERS`、`data_dir/key-peppers`、首次启动自动生成的 32 字节随机值。

必须备份整个 `data_dir`，尤其是 `key-peppers`。遗失 pepper 会使旧 Key 全部无法校验，并且无法再揭示其保存的密文。轮换 pepper 时，先追加新 pepper，设置 `active_pepper_id`，重启宿主后签发新 Key；确认不再需要旧 Key 后再删除旧 pepper。

### OAuth 认证额度

认证额度仅面向管理端，支持 Codex/ChatGPT、Claude、Antigravity、Kimi 和 xAI OAuth 认证；只含 API Key 的认证不会显示。每个认证最多每 15 分钟尝试刷新一次，失败时保留最后成功快照并标记为 `stale`。快照和响应不会保存 OAuth Token、上游账号 ID、认证文件路径、代理 URL 或原始上游响应。

## 运维与验证

- 同一 SQLite 数据库只允许一个写入进程。插件使用 `*.lock` 排他锁保护。
- 生产环境要配置真实价格；零配置中的全模型免费规则只适用于首次体验。
- 宿主仍负责上游 OAuth 或 API Key；本插件不处理上游登录。
- 将自助查询公开到互联网前，应限制网络暴露范围并通过 HTTPS 反向代理提供服务。

```bash
go test ./...
go test -race ./internal/store ./internal/service
go vet ./...
go run ./scripts/smoke_ledger.go
```

`smoke_ledger` 会在临时目录中验证 Key 签发、鉴权、预占、实际用量结算与撤销。

## 常见问题

| 问题 | 处理方式 |
|---|---|
| 构建提示 `gcc not found` | 安装 MinGW-w64、MSYS2 GCC 或 Zig；Windows 构建脚本会先找 `gcc`，再尝试 `zig cc`。 |
| 管理接口返回 401 | 管理路由必须使用宿主管理密钥，不是 `tk-...`。 |
| 模型调用返回 401 | 确认发送的是完整 `tk-...`，且签发 Key 的 pepper 未丢失或替换。 |
| 提示 `no pricing rule` | 添加价格规则，或将 `pricing.unknown_policy` 调整为 `allow` 或 `default`。 |
| Key 余额为负 | 实际 usage 超过预占额。提高额度后才可继续调用。 |
| 无 usage 是否按 `max_tokens` 扣费 | 文本请求默认不会；它先记 0，收到官方 usage 回调后再补记。 |
| 侧栏没有管理入口 | 确认插件已启用、部署的动态库包含 Resources，替换 DLL 后已重启宿主。 |
| 无法揭示旧 Key | 该 Key 可能早于可恢复存储功能；轮换它以获取可揭示的新 Key。 |
