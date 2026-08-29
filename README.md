# CPA Credit Manager（`credit-manager`）

面向 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 的原生 Go 插件，为模型代理请求提供独立鉴权、额度控制和基于实际用量的结算。

[English README](README.en.md)

插件 ID：`credit-manager`  
当前版本：`1.6.0`  
仓库：https://github.com/yuluo688/credit-manager

## 核心能力

1. **独立插件 Key 鉴权**：代理请求使用插件签发的 `tk-...` Key，不依赖宿主 `api-keys`。
2. **按 Key 严格预占**：转发前按保守 Token 上限或出图张数预留额度，余额或并发不足时拒绝请求。
3. **按实际用量结算**：解析上游 usage；文本按 Token 计价，纯出图按张计价。
4. **可视化与 API 管理**：统一管理 Key、模型启停、价格规则、额度、用量和审计。
5. **认证额度视图**：管理端可查看 Codex、Claude、Antigravity、Kimi、xAI OAuth 认证的上游额度窗口，以及可安全映射的本地用量预测。
6. **Key 自助查询**：Key 持有人无需 CPA 管理密钥即可查看自己的额度和用量。

## 设计目标

在高并发场景中，宿主 `api-keys` 与 usage 回调难以稳定关联调用身份和实际用量。本插件将鉴权、预占、透明转发与结算纳入同一执行链路，确保每笔用量归属到明确的插件 Key。

请求只带签发出来的整串 Key，不要拆、也不要另传 ID：

```text
Authorization: Bearer tk-...
```

## 快速开始

1. 编译并部署动态库。Windows 可使用 `scripts\deploy.ps1`，默认复制到 `D:\CLIProxyAPI\plugins\windows\amd64`。
2. 在 CLIProxyAPI 管理中心启用 `credit-manager`。宿主配置可为空；切换开关或重载配置通常无需重启。
3. 仅在替换动态库文件后重启 CLIProxyAPI。Windows 已加载的 DLL 通常无法热替换。
4. 在侧栏打开 **CPA 额度管理**，输入宿主管理密钥后，签发插件 Key，并设置总/日/周/月额度、最大并发、可用模型和计价规则。
5. 客户端携带 `Authorization: Bearer tk-...` 调用模型接口。
6. 可将 Key 自助查询链接提供给 Key 持有人，无需提供 CPA 管理密钥。

> **重要：** 启用后，插件独占代理请求的前端鉴权。原先配置在 CPA 中的 `api-keys` 或其他宿主 Key 不能再用于模型请求。请签发 `tk-...` 插件 Key，并让客户端改用该 Key。

> 为兼容 CPA 管理中心，`GET /v1/models` 与 `GET /v1beta/models` 可不带插件 Key。这只返回模型目录，不能据此调用模型。全局禁用的模型会从目录中去掉；某个 Key 的可用模型名单不会裁剪这份公共目录。

首次启动时，插件会自动完成：

| 项 | 默认 |
|----|------|
| pepper | `data_dir/key-peppers` 自动生成 |
| 归属记录 | `default`，仅用于 Key 归属与使用统计，不承载额度 |
| 价格 | 规则 `bootstrap-all-models`：regexp `.*`，费用 0；`unknown_policy=allow` |
| 插件 Key | **不自动创建**；请通过侧栏或管理 API 签发，并按需设置额度 |

> 生产环境应配置实际价格，并为每个插件 Key 设置合理额度。

### 前置要求

| 组件 | 要求 |
|------|------|
| CLIProxyAPI | 建议 `v7.2.128+`，并确保上游模型代理已正常运行 |
| 本插件 | 编译成对应平台动态库（Windows `.dll` / Linux `.so` / macOS `.dylib`） |
| C 工具链 | 需要 `gcc`/`clang`（CGO + `c-shared`）。Windows 也可用 Zig（`zig cc`） |
| Go | `1.26+` |

> Windows 替换 DLL 后必须重启 CLIProxyAPI。商店升级写入带版本号的新 DLL 时，见下文「Windows 插件升级」。

### 1. 编译插件

**Windows（PowerShell）**

```powershell
# 需要 PATH 里有 gcc，或已安装 Zig
# 若脚本执行受限，请选择以下任一方式：

# 方式 A：仅本次绕过策略（推荐）
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1

# 编译并复制至本机 CLIProxyAPI 插件目录
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1

# 方式 B：不使用脚本，直接编译
$env:CGO_ENABLED = "1"
New-Item -ItemType Directory -Force -Path dist | Out-Null
go build -buildmode=c-shared -o dist\credit-manager.dll .

# 方式 C：为当前用户调整执行策略（可选）
# Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
```

输出：`dist\credit-manager.dll` 及同名 `.h` 头文件。

**Linux / macOS**

```bash
chmod +x scripts/build.sh
./scripts/build.sh
# Linux: dist/credit-manager.so
# macOS: dist/credit-manager.dylib
```

### Pepper 管理

插件 Key 的 HMAC 校验与可恢复存储依赖 pepper。pepper **不得写入 SQLite**。

默认顺序：

1. 非空环境变量 `CREDIT_MANAGER_KEY_PEPPERS`
2. 否则读取 `data_dir/key-peppers`
3. 文件不存在时，首次启动生成 32 字节随机 pepper 并写入该文件（权限 `0600`）

```text
./data/credit-manager/key-peppers
# 内容形如：
# active:a1b2c3...（64 位 hex）
```

**备份 `data_dir` 时必须一并备份 `key-peppers`。** 删除 pepper 后重启会生成新 pepper，现有 Key 全部失效，已加密保存的明文也无法再揭示。

可选覆盖：

```powershell
$env:CREDIT_MANAGER_KEY_PEPPERS = "active:0123456789abcdef0123456789abcdef"
```

```bash
export CREDIT_MANAGER_KEY_PEPPERS='active:0123456789abcdef0123456789abcdef'
```

格式为 `id:pepper`；多个 pepper 可写为 `id1:p1,id2:p2`，或在文件中每行一个。`active_pepper_id` 指定签发新 Key 时使用的 pepper ID。

### 部署到 CLIProxyAPI

1. 将动态库复制到宿主插件目录。Windows 示例：`D:\CLIProxyAPI\plugins\windows\amd64\credit-manager.dll`；也可使用 `scripts\deploy.ps1`。
2. 在 CLIProxyAPI 管理中心的 **插件管理** 中启用 `credit-manager`，也可在宿主 YAML 中设置 `enabled: true`。
3. 替换动态库后，重启 CLIProxyAPI。

**推荐的宿主配置（零配置）**：

```yaml
plugins:
  enabled: true
  dir: ./plugins
  items:
    credit-manager:
      enabled: true
      # config 可完全省略；需要覆盖时再写
```

可选外部文件：

```yaml
items:
  credit-manager:
    enabled: true
    config: |
      config_file: E:/path/to/config.yaml
```

```bash
export CREDIT_MANAGER_CONFIG_FILE=/path/to/config.yaml
```

宿主会将 `plugins.items.*.config` 作为 `config_yaml` 注入插件；插件随后解析 `config_file` 或环境变量。完整字段见 `config.example.yaml`。宿主内联字段覆盖文件；文件内嵌套的 `config_file` 会被忽略。

同时确保：

- 宿主已配置**上游模型凭据**（OAuth、API Key 等）。本插件只做鉴权与额度，不处理上游登录。
- **独占前端鉴权**：代理请求只接受插件 `tk-...` Key。
- 默认 `data_dir`（`./data/credit-manager`）必须可写，其中保存 `key-peppers`、SQLite，以及可选的 `models-dev-api.json` 缓存。

### 签发插件 Key

1. 确认宿主已启动且插件已启用。
2. 在侧栏打开 **CPA 额度管理**（资源页 `/console`），签发 Key 并设置额度。
3. 也可调用 `POST /v0/management/credit-manager/keys`；可设置总/日/周/月额度、最大并发、可用模型、过期时间，或导入已有明文。
4. 健康检查使用**宿主管理密钥**，不是插件 Key：

```bash
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/health" \
  -H "Authorization: Bearer <宿主管理密钥>"
```

端口与管理鉴权头以宿主配置为准。若侧栏没有菜单，请确认插件已启用、动态库包含 Resources，并在替换文件后重启宿主。

### Key 自助查询页面

Key 持有人可通过独立页面查看**自己的**额度和使用统计，无需登录 CPA 管理面板：

```text
http://<CPA_HOST>:8317/v0/resource/plugins/credit-manager/lookup
```

页面要求输入整串 `tk-...`，并只返回该 Key 的数据：

- 总/日/周/月额度、在途请求和最大并发。日/周/月额度按 UTC 计算。
- 筛选里的「今日」按 **UTC 自然日**，与管理控制台的本地自然日不同。
- Token/费用趋势（时/日/月相互独立）、模型调用占比和最近调用明细。
- 显示货币可在 USD / CNY 间切换（汇率仅用于展示，不写入账本）。

安全边界：

- 页面不出现在 CPA 管理侧栏，也不需要宿主管理密钥。
- 插件 Key 只作为当前请求的 `Authorization: Bearer` 头发送，不写入 URL 或浏览器存储。主题、语言、显示单位等偏好除外。
- 公开响应不包含宿主账号、caller ID、插件 Key ID、认证文件路径或邮箱。
- 建议仅在受信任网络暴露 CPA 端口，并通过 HTTPS 反向代理对外提供访问。

### 管理控制台

启用后，管理中心侧栏显示 **CPA 额度管理**：

```text
/v0/resource/plugins/credit-manager/console
```

页面在浏览器 `sessionStorage` 中保存宿主管理密钥，并调用管理 API。标签页：

| 标签 | 内容 |
|------|------|
| 概览 | 按时间、Key、上游账号、模型、来源筛选。「今日」按浏览器本地自然日，与 Key 的 UTC 日额度不是同一口径 |
| 密钥管理 | 签发、查看、揭示、轮换、启停、撤销、删除；设置额度、并发和可用模型 |
| 模型与价格 | 加载当前代理模型，按模型设 Token 价或按张价，启用/禁用模型；可从 models.dev 目录回填公开价格 |
| 使用统计 | 与概览类似的筛选，外加费用/Token 区间；按 Key、按模型汇总和分页明细 |
| 认证额度 | OAuth 认证的上游额度窗口与本地预测 |

控制台里填写和展示的是美元（可切人民币显示）。管理 API 的额度/价格字段是 **micro-USD**（`1 USD = 1_000_000`）。不要把控制台里的 `10` 原样发到 API。汇率来自公开报价，缓存约 30 分钟，失败时回退 `7.2`，只用于显示。

---

## 管理 API：生产配置

零配置会创建 `default` 归属记录和全模型免费规则，**不会自动创建插件 Key**。示例假定：

- 宿主地址：`http://127.0.0.1:8317`
- 管理密钥：`MGMT_TOKEN`

所有管理路由使用宿主管理密钥，路径是固定的，没有 `/keys/{id}` 这种路径参数。记录 ID 放在 JSON 或查询参数里。创建 Key 返回的 `id` 只给管理接口用，请求模型时仍只带整串 `tk-...`。

#### 创建归属记录（caller）

caller 用于组织插件 Key 并聚合用量，**不承载额度**。可直接使用 `default`，也可按团队创建。

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/callers" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "team-a",
    "display_name": "A 组",
    "enabled": true
  }'
```

金额单位为 **micro-USD**：`1 USD = 1_000_000 micro-USD`。

#### 配置模型价格

文本模型价格是 **每 100 万 Token 的 micro-USD**。纯出图模型使用 `billing_mode: per_image`，`per_image` 为每张的 micro-USD。

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
      "cache_creation": 0,
      "accounting_mode": "input_includes_cache"
    }
  }'
```

出图示例：

```json
{
  "id": "gpt-image-1",
  "match_kind": "exact",
  "pattern": "gpt-image-1",
  "priority": 100,
  "enabled": true,
  "price": {
    "billing_mode": "per_image",
    "per_image": 40000
  }
}
```

也可使用通配：

```json
{
  "id": "default-all",
  "match_kind": "glob",
  "pattern": "*",
  "priority": 1,
  "enabled": true,
  "price": { "input": 1000000, "output": 3000000 }
}
```

`match_kind`：`exact` | `glob` | `regexp`  
匹配时 **priority 更高者优先**；同分再按规则 ID 升序。**禁用规则仍参与匹配。** 最高优先级命中的规则若 `enabled: false`，该模型会被拒绝，并从 `GET /v1/models` / `GET /v1beta/models` 中剔除；插件还会把这些模型合并进宿主认证文件的排除列表（只改插件管理的字段，其它 JSON 原样拷贝）。

`pricing.unknown_policy: deny` 时，未命中任何规则的模型请求会被拒绝。

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/pricing/enabled" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"gpt-4o","enabled":false}'
```

#### 签发插件 Key（明文仅在创建/轮换/揭示时返回）

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

响应包含：

```json
{
  "id": "...",
  "fingerprint": "...",
  "plaintext": "tk-..."
}
```

`plaintext` 才是客户端要带的 Key。`id` 是管理记录 ID，给 update/reveal/revoke 用，不能拿去当 Bearer。列表接口不返回明文。较新签发的 Key 可用 `POST .../keys/reveal` 再看一次；更早、尚未做可恢复存储的 Key 需要先轮换。

可选字段：

| 字段 | 含义 |
|------|------|
| `key_material` | 导入已有 `tk-...` 明文，而不是随机签发 |
| `expires_at` | RFC3339 过期时间 |
| `allowed_models` | 空表示全部模型；否则仅允许列出的 exact/glob 模式 |
| `quota_micro_usd` | `total_quota_micro_usd` 的别名 |

#### 查询额度、用量与审计

```bash
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/balance?key_id=<plugin_key_id>" \
  -H "Authorization: Bearer MGMT_TOKEN"

curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/usage?caller_id=team-a&limit=50" \
  -H "Authorization: Bearer MGMT_TOKEN"

curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/audit?caller_id=team-a&limit=50" \
  -H "Authorization: Bearer MGMT_TOKEN"
```

用量与概览还支持 `plugin_key_id`、`model`、`source`、`auth_id`、`auth_provider`、`from`、`to`、费用和 Token 区间等筛选。

### 调用模型接口

**请勿使用宿主 `api-keys`。**

```bash
curl -sS "http://127.0.0.1:8317/v1/chat/completions" \
  -H "Authorization: Bearer tk-..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role":"user","content":"你好"}],
    "max_tokens": 256
  }'
```

文本/对话请求：

1. 校验 Bearer Key。
2. 按模型价格和保守 Token 上限**严格预占**。
3. 通过宿主透明转发至上游。
4. 从上游 usage 或宿主后续用量回调结算。若始终没有 usage：文本默认记 `0` 费用（不会把预估 `max_tokens` 当实扣）；出图按预占张数结算。配置 `settlement.missing_usage=release` 时直接释放预占、不记账。

纯出图（如 `/v1/images/*`、`gpt-image-*`）走宿主原生出图接口：先预占，请求完成后再结算。

额度不足时，请求在预占阶段被拒绝。额度按插件 Key 独立管理。Key 若限制了可用模型，不在名单里的调用会被拒绝；全局禁用的模型同样会被拒绝，并且不会出现在公共模型目录里。

---

## 日常运维

### 调整插件 Key 限制

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/update" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "id":"<plugin_key_id>",
    "total_quota_micro_usd":20000000,
    "daily_quota_micro_usd":2000000,
    "weekly_quota_micro_usd":10000000,
    "monthly_quota_micro_usd":20000000,
    "max_concurrent_requests":5,
    "set_allowed_models": true,
    "allowed_models": ["gpt-4o"]
  }'
```

更新可用模型时必须带 `set_allowed_models: true`，否则不会改该字段。传入空数组表示允许全部模型。

### 禁用、轮换、揭示、撤销、删除

```bash
# 禁用
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/update" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>","enabled":false}'

# 揭示明文（需要可恢复存储）
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/reveal" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>"}'

# 轮换：签发新明文，并撤销旧 Key（旧明文立刻失效，不能再启用；历史用量留在旧记录上）
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/rotate" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>"}'

# 撤销：立刻失效且不能再启用
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/revoke" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>"}'

# 删除：效果等同撤销，数据库行仍在，控制台显示「已删除」，用量历史可查
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/delete" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>"}'
```

### 列出插件 Key（不含明文）

```bash
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/keys?caller_id=team-a" \
  -H "Authorization: Bearer MGMT_TOKEN"
```

---

## 管理 API

| 方法 | 路径 | 作用 |
|------|------|------|
| GET | `/v0/management/credit-manager/health` | 健康检查（含插件版本） |
| GET | `/v0/management/credit-manager/overview` | 控制台总览 |
| POST | `/v0/management/credit-manager/callers` | 创建归属记录 |
| GET | `/v0/management/credit-manager/callers` | 列出归属记录 |
| POST | `/v0/management/credit-manager/callers/enabled` | 启停归属记录 |
| POST | `/v0/management/credit-manager/keys` | 签发 Key |
| GET | `/v0/management/credit-manager/keys` | 列出 Key |
| POST | `/v0/management/credit-manager/keys/update` | 更新标签、启停、额度、并发、可用模型、过期时间 |
| POST | `/v0/management/credit-manager/keys/rotate` | 轮换 Key |
| POST | `/v0/management/credit-manager/keys/reveal` | 揭示明文 |
| POST | `/v0/management/credit-manager/keys/revoke` | 撤销 Key（不可再启用） |
| POST | `/v0/management/credit-manager/keys/delete` | 标记删除（行仍保留，控制台显示已删除） |
| POST | `/v0/management/credit-manager/pricing` | 新增/更新价格规则 |
| GET | `/v0/management/credit-manager/pricing` | 列出价格规则 |
| POST | `/v0/management/credit-manager/pricing/enabled` | 启用/禁用规则（从而启停模型） |
| POST | `/v0/management/credit-manager/pricing/delete` | 删除价格规则 |
| GET | `/v0/management/credit-manager/balance?key_id=` | 查询 Key 余额 |
| GET | `/v0/management/credit-manager/usage` | 用量流水（分页） |
| GET | `/v0/management/credit-manager/usage/summary` | 按 Key 和模型汇总 |
| GET | `/v0/management/credit-manager/audit` | 审计事件 |
| GET | `/v0/management/credit-manager/auth-quotas` | OAuth 认证额度窗口 |
| POST | `/v0/management/credit-manager/auth-quotas/concurrency` | 设置认证最大并发 |
| POST | `/v0/management/credit-manager/auth-quotas/concurrency/batch` | 批量设置认证最大并发 |

浏览器页面不走宿主管理鉴权。控制台打开后仍要输入管理密钥；自助查询打开后仍要输入插件 Key。下面两个辅助地址给页面自己用，一般不必手调：

| 路径 | 作用 |
|------|------|
| `/v0/resource/plugins/credit-manager/console` | 管理控制台页面 |
| `/v0/resource/plugins/credit-manager/lookup` | Key 自助查询页面 |
| `/v0/resource/plugins/credit-manager/lookup/data` | 自助查询数据（要带插件 Key） |
| `/v0/resource/plugins/credit-manager/fx/usd-cny` | 控制台用的 USD/CNY 展示汇率 |
| `/v0/resource/plugins/credit-manager/models-dev` | 控制台用的公开价格目录缓存 |

签发、轮换、揭示和认证额度响应固定 `Cache-Control: no-store`。

### 认证额度视图

侧栏 **认证额度** 和 `GET /v0/management/credit-manager/auth-quotas` 仅允许宿主管理鉴权；公开自助查询页不展示认证文件或认证额度。

- 支持 Codex/ChatGPT、Claude、Antigravity、Kimi、xAI 的 OAuth 认证；仅 API Key 的认证不会进入结果。
- 采样走 CLIProxyAPI 宿主 HTTP callback 和全局出站策略。管理 callback 不会套用认证文件里的 `proxy_url`。
- 单个认证最多每 15 分钟尝试刷新一次。失败会保留最后成功快照并标为 `stale`；从未成功采样则为 `unavailable`。
- 百分比、请求数、credits、USD 等上游单位分别展示。只有能安全映射到账本周期和模型池的窗口才显示本地调用预测；网页、其它 CLI 或其它代理节点的使用会降低真实剩余次数。
- 卡片可设置该认证的最大并发；`0` 或不填表示不限制。工具栏可按本页或当前筛选批量写入。有限制时插件会在宿主选凭证前跳过已满的账号，全部满员则拒绝请求。在途数按尚未结算或释放、且已归属到该认证的请求计算。
- SQLite 快照与管理响应不包含 OAuth Token、上游账户 ID、认证文件路径、proxy URL 或原始上游响应。私有上游接口可能随时调整。

---

## 插件配置说明

完整示例见 [`config.example.yaml`](config.example.yaml)。

| 字段 | 含义 |
|------|------|
| `data_dir` | 插件数据目录：SQLite、锁文件、pepper、models.dev 缓存 |
| `database_file` | 数据库文件名，默认 `credit-manager.db` |
| `busy_timeout` | SQLite 忙等待，例如 `5s` |
| `keys.pepper_env` | pepper 环境变量名；有有效值时优先于文件 |
| `keys.pepper_file` | pepper 文件，相对 `data_dir` 或绝对路径；默认 `key-peppers` |
| `keys.active_pepper_id` | 签发新 Key 使用的 pepper ID |
| `limits.max_token_estimate` | 单请求 Token 预估上限，超过则拒绝 |
| `limits.default_output_reserve` | 请求体未提供 `max_tokens` 时的默认输出预占 |
| `limits.require_estimate` | `true` 时拒绝无法估计 Token 的请求 |
| `pricing.unknown_policy` | 未匹配规则：`deny`、`allow` 或 `default`；零配置默认 `allow` |
| `pricing.default` | 仅 `unknown_policy=default` 时需要的默认单价 |
| `settlement.missing_usage` | 上游无 usage：`release` 直接放预占；`settle_reserved` 对文本记 0 并等回调，对出图按张结算。名字不代表按预占额实扣 |
| `settlement.host_usage_wait` | 结算前等待宿主用量回调的时长，默认 `1500ms`；`0` 关闭 |
| `stream.max_buffer_bytes` | 流式结算本地缓冲上限 |
| `stream.stale_reservation_timeout` | 无心跳在途预约自动释放阈值，默认 `2h` |

正常流式和非流式请求会刷新预约心跳。过期预约在启动、配置重载以及新预占前定期回收。

---

## 金额与计费

- 账本只存整数 **micro-USD**，不用浮点金额。
- 文本价格单位为 **每 1M Token 的 micro-USD**。
- 各 Token 类别费用分别**向上取整**后相加。
- 计费口径与 [cap-token-usage-tracker](https://github.com/AITNR/cap-token-usage-tracker) 对齐：只对 Input、Output、Cache Read、Cache Creation 四项计价。
- OpenAI 兼容 usage 的 `input`/`prompt_tokens` 已包含缓存，结算时先扣除 Cache Read/Creation，避免重复计费。
- Claude/Anthropic 的 `input_tokens` 不含缓存，四项独立计价。
- Reasoning 与通用 Cached 只作统计，不再单独加价；未提供 `cache_read` 时回退使用 `cached`。
- 官方 `total_tokens` 优先作为展示用总 Token；否则为 input+output+reasoning。
- 纯出图按张计费，不能套用 Token 价。缺 usage 时按请求里的张数（默认 1）结算，而不是按 Token 预估实扣。
- **严格预占**：单个 Key 可用额度不足时直接拒绝；Key 之间不共享额度。
- Key 可分别设置总、日、周、月额度和最大并发；`0` 或省略表示不限制。日/周/月按 UTC 自然周期，周从周一开始。
- 周期额度 = 已结算费用 + 当前周期在途预占；并发按未结算或未释放的请求数计算。全部检查在同一个预占事务内完成。
- **实际结算可能超过预占额**：文本按真实 usage 计价，可能大于预占。此时 Key 余额可能为负，后续请求继续 fail-closed，直到额度恢复。
- 文本请求若始终没有 usage，默认**不会**把预估 Token 当成实扣；账本先记 0。若后来宿主补到官方用量，再按实际回填。

### 价格字段

| 字段 | 含义 |
|------|------|
| `input` | 输入 Token |
| `output` | 输出 Token |
| `reasoning` | 推理 Token（统计用，默认不加价） |
| `cached` | 通用缓存 Token；`cache_read` 为空时回退 |
| `cache_read` | 缓存读取 Token |
| `cache_creation` | 缓存创建 Token |
| `accounting_mode` | `input_includes_cache`（OpenAI）或 `input_excludes_cache`（Claude）；空则按模型名判断 |
| `billing_mode` | `token`（默认）或 `per_image` |
| `per_image` | 每张图的 micro-USD，仅 `per_image` 模式使用 |

---

## 密钥安全

- 客户端把签发的 `tk-...` 当不透明字符串使用即可。
- 数据库不存明文；管理端揭示依赖 pepper 派生的密文。pepper 丢失后现有 Key 全部失效。
- pepper 只在环境变量或 `data_dir/key-peppers` 中，不进入 SQLite、日志或普通管理查询。
- 签发 / 轮换 / 揭示响应带 `Cache-Control: no-store`。
- 认证额度快照不保存 OAuth 凭据、认证文件路径或原始上游响应；公开 `/lookup` 不含认证身份或认证额度字段。

### Pepper 轮换建议

1. 在 pepper 文件或环境变量中**追加**新 pepper，例如 `newid:....,active:....`。
2. 设置 `active_pepper_id: newid`。
3. 重启宿主，之后新 Key 使用新 pepper。
4. 保留旧 pepper，直到依赖它的旧 Key 全部退役；删除旧 pepper 后，旧 Key 无法校验，其密文也无法揭示。

---

## 请求架构

```text
客户端
  |  Authorization: Bearer tk-...
  v
CLIProxyAPI
  |  frontend_auth.authenticate  -> 本插件校验 Key，返回 principal
  |  GET /v1/models              -> 允许匿名读目录；全局禁用的模型不会出现
  |  model.route                 -> 文本/对话路由到本插件 executor
  |                              -> 纯出图交给宿主原生路径
  |  executor.execute / stream   -> 预占、透明转发、结算
  |  request interceptors        -> 出图预占 / 完成结算
  |  宿主用量回调                 -> 补记官方 usage
  v
上游模型
```

宿主会跳过当前插件自身作为上游执行器，避免递归。支持的协议包括 openai、chat-completions、claude、gemini、openai-response、responses、codex，以及 openai-image / openai-video。

---

## 运维注意事项

1. **单写者限制**：同一数据库不应由多个进程同时写入。插件使用 `*.lock` 排他锁保护写操作。
2. **数据备份**：备份整个 `data_dir`，包括 SQLite 和 `key-peppers`。建议停止写入后备份，或使用 SQLite 在线备份。
3. **Windows 插件升级**：商店升级会写入带版本号的新 DLL，并在旧实例仍占用库锁时注册新实例。`1.4.0+` 会通过 handover 把库锁交给新实例。从更旧版本第一次升到 `1.4.0` 时，旧实例还不认识该协议，需要卸载再安装一次（或重启宿主）；之后的商店升级无需卸载。若直接覆盖正在加载的同名 DLL，仍须重启宿主。
4. **独占鉴权**：启用后宿主原有 `api-keys` 不再用于代理鉴权。
5. **未知模型**：零配置默认 `unknown_policy: allow`，并预置全模型免费规则。生产应配置实际价格，必要时改为 `deny`。
6. **侧栏菜单**：**CPA 额度管理** 由插件 Resources 注册，不是由 API 路由自动生成。插件未启用时不会显示。
7. **模型目录**：全局禁用会同步到宿主认证排除列表，但只改插件管理的 `credit_manager_excluded_models` 相关字段，避免整份认证 JSON round-trip 弄坏 OAuth 文件。

---

## 开发验证

```bash
go test ./...
go test -race ./internal/store ./internal/service
go vet ./...
go run ./scripts/smoke_ledger.go
```

`smoke_ledger` 会在临时目录中验证：创建归属记录、签发带额度的插件 Key、鉴权、按 Key 预占、usage 结算及撤销。

---

## 常见问题

**Q：编译时出现 `gcc not found`？**

A：安装 MinGW-w64、MSYS2 gcc，或 Zig，并加入 `PATH`。`scripts/build.ps1` 会优先用 gcc，否则尝试 `zig cc`。

**Q：管理接口返回 401？**

A：管理路由使用宿主管理鉴权，不是插件 Key。

**Q：代理请求返回 401？**

A：确认请求带的是签发出来的整串 `tk-...`，且当前 pepper 与签发时一致。检查环境变量或 `data_dir/key-peppers` 未被替换。

**Q：请求因 `no pricing rule` 被拒绝？**

A：零配置默认包含 `.*` 免费规则。若已删除全部规则并把 `unknown_policy` 设为 `deny`，请重新添加价格规则，或改为 `allow` / `default`。

**Q：模型被禁用后，客户端目录里还看得到？**

A：在「模型与价格」里禁用该模型后，调用会被拒绝，公共模型目录里也不应再出现。若还在，请确认已部署最新插件并重启宿主。某个 Key 的可用模型名单不会裁剪公共目录。

**Q：侧栏未显示「CPA 额度管理」？**

A：确认插件已启用，并已部署包含 console Resource 的动态库。仅在替换文件后需要重启宿主。

**Q：管理中心显示「未注册」，重新启用后仍未恢复？**

A：商店升级会先加载新的版本化 DLL 并调用 `plugin.register`，此时旧实例通常仍占着 `*.db.lock`。`1.4.0+` 会让旧实例交锁。若当前仍是更旧版本，请先卸载再安装一次（或重启宿主）。之后商店升级应能直接注册。

**Q：插件 Key 额度为何变为负数？**

A：当实际 usage 超过预占额时，系统按实际用量结算，避免漏计费。该 Key 后续请求会因额度不足被拒绝，直到上调额度。

**Q：上游没返回 usage，会不会按 `max_tokens` 扣费？**

A：文本不会。默认先记 0；如果后来拿到官方用量再按实际补记。出图在缺 usage 时按张数结算。若配置 `settlement.missing_usage=release`，则释放预占、不记账。

**Q：客户端仍需使用原来的 `api-keys` 吗？**

A：不需要。插件启用独占前端鉴权后，代理请求只接受插件 Key。

**Q：Key 持有人如何查看自己的额度和用量？**

A：访问 `http://<CPA_HOST>:8317/v0/resource/plugins/credit-manager/lookup`，输入自己的 `tk-...` Key。该页面无需 CPA 管理密钥。

**Q：揭示 Key 失败，提示需要轮换？**

A：该 Key 在可恢复存储之前签发，数据库没有加密明文。轮换会生成可揭示的新 Key，并撤销旧 Key。

**Q：删除 Key 后列表里还能看到？**

A：删除不会抹掉数据库行，只是撤销后标成「已删除」，用量历史还在。禁用可以再打开；撤销/删除不能再拿旧明文鉴权。
