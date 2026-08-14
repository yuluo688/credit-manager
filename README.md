# CPA Credit Manager（`credit-manager`）

面向 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 的原生 Go 插件，为模型代理请求提供独立鉴权、额度控制和基于实际 Token 用量的结算能力。

插件 ID：`credit-manager`

## 核心能力

1. **独立插件 Key 鉴权**：代理请求使用插件签发的 Key，不依赖宿主 `api-keys`。
2. **按 Key 严格预占额度**：请求转发前按保守 Token 上限预留额度，余额不足时拒绝请求。
3. **按实际 Token 用量结算**：解析上游响应中的 usage，在同一请求链路内完成结算。
4. **可视化与 API 管理**：支持 Key、可用模型、价格规则、额度、用量和审计记录的统一管理。

## 设计目标

在高并发场景中，宿主 `api-keys` 与 usage 回调难以稳定关联调用身份和实际用量。本插件将鉴权、预占、透明转发与结算纳入同一执行链路，从而确保每笔用量均归属到明确的插件 Key。

请求使用以下鉴权头：

```text
Authorization: Bearer tk-...
```

## 快速开始

1. 编译并部署动态库。Windows 可使用 `scripts\deploy.ps1` 部署至 `D:\CLIProxyAPI\plugins\windows\amd64`。
2. 在 CLIProxyAPI 管理中心启用 `credit-manager`。宿主配置可为空；切换开关或重载配置通常无需重启。
3. 仅在替换动态库文件后重启 CLIProxyAPI。Windows 已加载的 DLL 通常无法热替换。
4. 在侧栏打开 **CPA 额度管理**，输入宿主管理密钥后，签发插件 Key 并设置独立额度、可用模型和模型计价规则；随后可查询按 Key 和模型聚合的用量统计。
5. 客户端携带 `Authorization: Bearer tk-...` 调用模型接口。

> **重要：** 启用 `credit-manager` 后，插件会接管代理请求的前端鉴权。原先配置在 CPA 中的 `api-keys` 或其他宿主 Key 将无法用于模型请求；请在 **CPA 额度管理** 中签发 `tk-...` 格式的插件 Key，并将客户端改用该 Key。

首次启动时，插件会自动完成以下初始化：

| 项 | 默认 |
|----|------|
| pepper | `data_dir/key-peppers` 自动生成 |
| 归属记录 | `default`，仅用于 Key 归属与使用统计 |
| 价格 | 全模型规则 `.*`，费用 0；`unknown_policy=allow` |
| 插件 Key | **不自动创建**；请通过侧栏或管理 API 手动签发，并提供 `quota_micro_usd` |

> 生产环境应配置实际价格，并为每个插件 Key 设置合理额度。以下章节说明编译、部署及可选高级配置。

### 前置要求

| 组件 | 要求 |
|------|------|
| CLIProxyAPI | 建议使用 `v7.2.128+`，并确保模型代理已正常运行 |
| 本插件 | 编译成对应平台动态库（Windows `.dll` / Linux `.so` / macOS `.dylib`） |
| C 工具链 | 编译插件需要 `gcc`/`clang`（CGO + `c-shared`） |

> Windows 环境需要安装 MinGW-w64 或其他可用的 gcc 工具链。替换 DLL 后，必须重启 CLIProxyAPI 以加载新的二进制文件。

### 1. 编译插件

**Windows（PowerShell）**

```powershell
# 需要 PATH 里有 gcc
# 若 PowerShell 提示脚本执行受到限制，请选择以下任一方式：

# 方式 A：仅本次执行时绕过策略（推荐）
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

输出文件：`dist\credit-manager.dll` 及其同名 `.h` 头文件。

**Linux / macOS**

```bash
chmod +x scripts/build.sh
./scripts/build.sh
# Linux: dist/credit-manager.so
# macOS: dist/credit-manager.dylib
```

### Pepper 管理

插件 Key 的 HMAC 校验依赖 pepper。pepper **不得存储于 SQLite 数据库中**。

默认行为：

1. 若设置了非空环境变量 `CREDIT_MANAGER_KEY_PEPPERS`，则使用该环境变量。
2. 否则读取 `data_dir/key-peppers`。
3. 若文件不存在，则在**首次启动时生成 32 字节随机 pepper 并写入该文件**，后续持续复用。

```text
# 自动生成后的文件示例（权限 0600）
./data/credit-manager/key-peppers
# 内容形如：
# active:a1b2c3...（64 位 hex）
```

**备份 `data_dir` 时必须一并备份 `key-peppers`。** 删除 pepper 后重新启动会生成新的 pepper，现有插件 Key 将全部失效。

可选：通过环境变量覆盖文件配置，适用于运维托管或多节点共享密钥的场景：

```powershell
$env:CREDIT_MANAGER_KEY_PEPPERS = "active:0123456789abcdef0123456789abcdef"
```

```bash
export CREDIT_MANAGER_KEY_PEPPERS='active:0123456789abcdef0123456789abcdef'
```

格式为 `id:pepper`；多个 pepper 可写为 `id1:p1,id2:p2`，或在文件中每行写入一个。`active_pepper_id` 用于指定签发新 Key 时使用的 pepper ID。

### 部署到 CLIProxyAPI

1. 将动态库复制到宿主插件目录。Windows 示例：`D:\CLIProxyAPI\plugins\windows\amd64\credit-manager.dll`；也可使用 `scripts\deploy.ps1`。
2. 在 CPAMC 或管理中心的 **插件管理** 中启用 `credit-manager`，也可在宿主 YAML 中设置 `enabled: true`。
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

可选：通过外部文件覆盖默认值：

```yaml
items:
  credit-manager:
    enabled: true
    config: |
      config_file: E:/path/to/config.yaml
```

```bash
# Linux/macOS：通过环境变量指定外部配置文件
export CREDIT_MANAGER_CONFIG_FILE=/path/to/config.yaml
```

宿主会将 `plugins.items.*.config` 作为 `config_yaml` 注入插件；插件随后解析 `config_file` 或环境变量。完整字段定义见 `config.example.yaml`。

同时确保：

- 宿主已配置**上游模型凭据**（OAuth、API Key 等）。本插件仅负责鉴权与额度管理，不处理上游登录。
- **独占前端鉴权**：启用后，代理请求仅接受插件 `tk-...` Key；原先的宿主 `api-keys` 和其他 CPA Key 均不再可用于模型请求。
- 默认 `data_dir`（`./data/credit-manager`）必须可写，其中保存 `key-peppers` 和 SQLite 数据库。

### 签发插件 Key

1. 确认宿主已启动且插件处于启用状态。
2. 在侧栏打开 **CPA 额度管理**（资源页 `/console`），签发插件 Key 并设置额度。
3. 也可调用 `POST /v0/management/credit-manager/keys`；`quota_micro_usd` 为必填字段。
4. 可选：通过以下接口进行健康检查。此处使用宿主管理密钥，而非插件 Key：

```bash
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/health" \
  -H "Authorization: Bearer <宿主管理密钥>"
```

端口与管理鉴权头以宿主配置为准。若侧栏未显示菜单，请确认插件已启用、DLL 为包含 Resources 的最新构建，并在替换 DLL 后重启宿主。

### 管理 API：生产配置

零配置会自动创建 `default` 归属记录与覆盖全部模型的免费价格规则，**不会自动创建插件 Key**。以下接口用于生产环境中管理额度、价格规则和多个插件 Key。示例假定：

- 宿主地址：`http://127.0.0.1:8317`
- 管理密钥：`MGMT_TOKEN`（替换成你的）

#### 创建归属记录（caller）

caller 用于组织插件 Key 并聚合用量记录，**不承载额度**。可直接使用默认 `default`，也可按团队或业务单元创建。

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

金额单位为 **micro-USD**，用于插件 Key 的 `quota_micro_usd`：

- `1 USD = 1_000_000 micro-USD`

#### 配置模型价格

价格使用 **每 100 万 Token 的 micro-USD 单价**。

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
      "reasoning": 0,
      "cached": 0,
      "cache_read": 0,
      "cache_creation": 0
    }
  }'
```

也可使用通配规则：

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
规则匹配时，**priority 值更高的规则优先**。

配置 `pricing.unknown_policy: deny` 后，**未命中价格规则的模型请求将被直接拒绝**。

#### 签发插件 Key（明文仅返回一次）

```bash
curl -sS -D - -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "caller_id": "team-a",
    "label": "ci-bot",
    "quota_micro_usd": 10000000
  }'
```

响应里会有：

```json
{
  "id": "...",
  "kid": "...",
  "fingerprint": "...",
  "plaintext": "tk-xxxx_yyyy",
  ...
}
```

**请立即安全保存 `plaintext`。后续任何接口均不会返回完整密钥。**

#### 查询额度、用量与审计记录

```bash
# Key 额度
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/balance?key_id=<plugin_key_id>" \
  -H "Authorization: Bearer MGMT_TOKEN"

# 用量流水
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/usage?caller_id=team-a&limit=50" \
  -H "Authorization: Bearer MGMT_TOKEN"

# 审计
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/audit?caller_id=team-a&limit=50" \
  -H "Authorization: Bearer MGMT_TOKEN"
```

### 调用模型接口

**请勿使用宿主 `api-keys`。** 请通过侧栏或管理 API 手动签发设置了额度的插件 Key：

```bash
curl -sS "http://127.0.0.1:8317/v1/chat/completions" \
  -H "Authorization: Bearer tk-xxxx_yyyy" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role":"user","content":"你好"}],
    "max_tokens": 256
  }'
```

请求处理过程如下：

1. 校验 Bearer Key。
2. 根据模型价格和保守 Token 上限执行**严格预占**。
3. 通过宿主透明转发至实际的上游模型。
4. 从响应 usage 结算；若未返回 usage，则按 `settlement.missing_usage` 策略处理，默认按预占额结算。

额度不足时，请求将在预占阶段被直接拒绝（fail-closed）。额度按插件 Key 独立管理，互不共享。

---

## 日常运维

### 调整插件 Key 额度

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/update" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>","quota_micro_usd":20000000}'
```

### 禁用或启用插件 Key

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/update" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>","enabled":false}'
```

### 撤销插件 Key

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/revoke" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>"}'
```

### 列出插件 Key（不包含明文）

```bash
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/keys?caller_id=team-a" \
  -H "Authorization: Bearer MGMT_TOKEN"
```

---

## 管理 API

所有接口均使用宿主 `/v0/management` 鉴权。路由采用**精确匹配**；ID 应放在请求体或查询参数中，而非路径参数中。

启用插件后，管理中心侧栏将显示 **「CPA 额度管理」**（资源页：`/v0/resource/plugins/credit-manager/console`）。在页面中输入宿主管理密钥后，可视化管理插件 Key、可用模型、额度及按 Key 统计数据。若菜单未显示，请确认插件已启用，并在替换最新 DLL 构建后重启宿主。

| 方法 | 路径 | 作用 |
|------|------|------|
| GET | `/v0/management/credit-manager/health` | 健康检查 |
| GET | `/v0/management/credit-manager/overview` | 控制台总览（归属记录/Key/价格/统计） |
| POST | `/v0/management/credit-manager/callers` | 创建插件 Key 归属记录 |
| GET | `/v0/management/credit-manager/callers` | 列出插件 Key 归属记录 |
| POST | `/v0/management/credit-manager/callers/enabled` | 启停归属记录 |
| POST | `/v0/management/credit-manager/keys` | 签发插件 Key（`quota_micro_usd` 必填，可指定 `allowed_models`） |
| GET | `/v0/management/credit-manager/keys` | 列出插件 Key（`caller_id` 可选） |
| POST | `/v0/management/credit-manager/keys/update` | 更新 Key 标签、启用状态、额度或可用模型 |
| POST | `/v0/management/credit-manager/keys/revoke` | 撤销插件 Key |
| POST | `/v0/management/credit-manager/pricing` | 新增/更新价格规则 |
| GET | `/v0/management/credit-manager/pricing` | 列价格规则 |
| POST | `/v0/management/credit-manager/pricing/delete` | 删除价格规则 |
| GET | `/v0/management/credit-manager/balance?key_id=` | 查询插件 Key 剩余额度 |
| GET | `/v0/management/credit-manager/usage` | 查询用量流水（可按 `plugin_key_id` 或 `model` 过滤） |
| GET | `/v0/management/credit-manager/usage/summary` | 按插件 Key 和模型汇总 |
| GET | `/v0/management/credit-manager/audit?caller_id=` | 审计事件 |

---

## 插件配置说明

完整配置示例见 `config.example.yaml`。

| 字段 | 含义 |
|------|------|
| `data_dir` | 插件数据目录，用于存放 SQLite、锁文件和 pepper 文件 |
| `database_file` | 数据库文件名，默认值为 `credit-manager.db` |
| `busy_timeout` | SQLite 忙等待时长，例如 `5s` |
| `keys.pepper_env` | 可选的 pepper 环境变量名；存在有效值时优先于文件 |
| `keys.pepper_file` | pepper 文件路径，可相对 `data_dir` 或使用绝对路径；默认 `key-peppers`，不存在时首次启动自动生成 |
| `keys.active_pepper_id` | 签发新插件 Key 时使用的 pepper ID |
| `limits.max_token_estimate` | 单个请求允许的 Token 预估上限，超过时直接拒绝 |
| `limits.default_output_reserve` | 请求体未提供 `max_tokens` 时的默认输出 Token 预占量 |
| `pricing.unknown_policy` | 未匹配价格规则时的策略：`deny`、`allow` 或 `default`；零配置默认 `allow` |
| `settlement.missing_usage` | 上游未返回 usage 时的处理策略：`settle_reserved` 或 `release` |
| `stream.max_buffer_bytes` | 流式结算的本地缓冲区上限 |

---

## 金额与计费

- 账本仅存储整数 **micro-USD**，不使用浮点金额。
- 价格单位为 **每 1M Token 的 micro-USD**。
- 各 Token 类别的费用分别**向上取整**后相加。
- **严格预占**：单个插件 Key 的可用额度不足时，请求直接拒绝；Key 之间不共享额度。
- **实际结算可能超过预占额**：为避免遗漏真实用量，结算额可大于预占额。此时 Key 余额可能为负，后续请求将继续遵循 fail-closed 策略，直至额度恢复。

### 价格字段

| 字段 | 含义 |
|------|------|
| `input` | 输入 Token |
| `output` | 输出 Token |
| `reasoning` | 推理 Token |
| `cached` | 通用缓存 Token |
| `cache_read` | 缓存读取 Token |
| `cache_creation` | 缓存创建 Token |

---

## 密钥安全

- 明文格式：`tk-<kid>-<secret>`。
- 数据库仅存储：`kid`、HMAC、pepper ID、指纹、principal 和 `caller_scope`。
- pepper 仅保存在环境变量或 `data_dir/key-peppers` 中，不写入 SQLite、日志或管理查询结果。
- 签发 Key 的响应包含 `Cache-Control: no-store`。

### Pepper 轮换建议

1. 在 pepper 文件或环境变量中**追加**新的 pepper，例如 `newid:....,active:....`，也可使用多行格式。
2. 设置 `active_pepper_id: newid`。
3. 重启宿主，并使用新的 pepper 签发后续插件 Key。
4. 保留旧 pepper，直至依赖它的全部旧 Key 退役后再删除。

---

## 请求架构

```text
客户端
  |  Authorization: Bearer tk-...
  v
CLIProxyAPI
  |  frontend_auth.authenticate  -> 本插件校验 Key，返回 principal
  |  model.route                 -> 路由到本插件 executor
  |  executor.execute / stream
  |     1) 查询 Key，计算保守费用，并按 Key 原子预占
  |     2) 使用 host.model.execute(_stream) 透明转发至上游
  |     3) 解析 usage 并结算，或按策略兜底处理
  v
上游模型
```

宿主会跳过当前插件自身作为上游执行器，以避免递归调用。

---

## 运维注意事项

1. **单写者限制**：同一数据库不应由多个进程同时写入。插件使用 `*.lock` 排他锁保护写操作。
2. **数据备份**：备份整个 `data_dir`，包括 SQLite 数据库和 `key-peppers`。建议在停止写入后备份，或使用 SQLite 在线备份机制。
3. **Windows 插件升级**：替换 DLL 后必须重启宿主；仅切换插件开关或重载配置通常无需重启。
4. **独占鉴权影响**：启用本插件后，宿主原有 `api-keys` 不再用于代理鉴权，应改用插件 Key。
5. **未知模型处理**：零配置默认 `unknown_policy: allow`，并预置了全模型免费规则。生产环境应配置实际价格，必要时将策略调整为 `deny`。
6. **侧栏菜单来源**：**CPA 额度管理** 由插件 Resources 注册，并非由 API 路由自动生成。插件未启用时不会显示该菜单。

---

## 开发验证

```bash
go build ./internal/...
go vet ./internal/...
go run ./scripts/smoke_ledger.go
```

`smoke_ledger` 会在临时目录中验证以下流程：创建归属记录、签发带额度的插件 Key、鉴权、按 Key 预占、usage 结算及撤销。

---

## 常见问题

**Q：编译时出现 `gcc not found`，应如何处理？**

A：安装 MinGW-w64 或 MSYS2 提供的 gcc，并将其加入 `PATH`。

**Q：管理接口返回 401，原因是什么？**

A：管理路由使用宿主管理鉴权，而非插件 Key。请使用宿主的 management token。

**Q：代理请求返回 401，原因是什么？**

A：确认请求使用了 `tk-...`，并确认当前 pepper 与 Key 签发时一致。请检查环境变量或 `data_dir/key-peppers` 未被替换或删除。

**Q：请求因 `no pricing rule` 被拒绝，如何处理？**

A：零配置默认包含 `.*` 免费规则。若已删除所有规则并将 `unknown_policy` 设为 `deny`，请重新添加价格规则，或将策略改为 `allow` 或 `default`。

**Q：侧栏未显示「CPA 额度管理」，如何处理？**

A：确认插件开关已启用，并已部署包含 console Resource 的 DLL。仅在替换 DLL 后需要重启宿主；CPAMC 仅在插件启用后显示该菜单。

**Q：管理中心显示「未注册」，重新启用后仍未恢复，如何处理？**

A：旧版本在 reconfigure 时可能因重复申请 `*.db.lock` 而失败。请部署包含该修复的 DLL，并重启一次宿主以加载新二进制；之后正常切换开关或重载配置应可恢复注册。

**Q：插件 Key 额度为何变为负数？**

A：当实际 usage 超过预占额时，系统会按实际用量结算，以避免漏计费。这是预期行为；该 Key 后续请求将因额度不足被拒绝，直至上调其额度。

**Q：客户端仍需使用原来的 `api-keys` 吗？**

A：不需要。插件启用独占前端鉴权后，代理请求仅接受插件 Key。
