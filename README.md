# CPA Credit Manager（`credit-manager`）

面向 [CLIProxyAPI](https://github.com/router-for-me/CLIProxyAPI) 的原生 Go 插件，用来给代理请求做：

1. **独立插件 Key 鉴权**（不再用宿主 `api-keys` 认代理请求）
2. **按 Key 额度预占**
3. **按真实 Token 结算**

插件 ID：`credit-manager`

---

## 它解决什么问题

CLIProxyAPI 自带的 `api-keys` + usage 回调，很难在高并发下把“是谁花的钱”和“真实 usage”可靠对上。

本插件改成：

- 客户端只带：`Authorization: Bearer tk-v1-...`
- 插件自己签发、校验 Key
- 请求进入插件执行器后：先预占额度 → 再透明转发给宿主上游 → 从响应里解析 usage 并结算

因此：**鉴权、预占、上游调用、结算都在同一次执行链路里完成。**

---

## 你怎么用（零配置最短路径）

1. 编译并部署 DLL（Windows 可用 `scripts\deploy.ps1` 拷到 `D:\CLIProxyAPI\plugins\windows\amd64`）
2. 在管理中心 **启用** `credit-manager`（宿主 config 可留空；开关/重载配置一般**不必**重启）
3. 仅当**替换了 DLL 文件**时才需要重启 CLIProxyAPI
4. 打开侧栏 **CPA 额度管理**，填入宿主**管理密钥**后：
   - 手动签发 Key，并设置每个 Key 的额度
   - 设置 Key 可用模型
   - 配置模型价格
   - 查看按 Key / 模型的使用统计
5. 客户端：`Authorization: Bearer tk-v1-...` 直接调模型

首次启动自动完成：

| 项 | 默认 |
|----|------|
| pepper | `data_dir/key-peppers` 自动生成 |
| 归属记录 | `default`，仅用于 Key 归属与使用统计 |
| 价格 | 全模型规则 `.*`，费用 0；`unknown_policy=allow` |
| API Key | **不自动创建**；通过侧栏或管理 API 手动签发，并提供 `quota_micro_usd` |

> 生产请配置真实价格，并为每个 Key 设定合适额度。下面是编译与可选进阶配置。

### 0. 前提

| 组件 | 要求 |
|------|------|
| CLIProxyAPI | 建议 `v7.2.128+`，已能正常跑模型代理 |
| 本插件 | 编译成对应平台动态库（Windows `.dll` / Linux `.so` / macOS `.dylib`） |
| C 工具链 | 编译插件需要 `gcc`/`clang`（CGO + `c-shared`） |

> 当前这台 Windows 若没装 MinGW-w64，先装 gcc 再编译。  
> 动态库编好后，**升级 DLL 需要重启 CLIProxyAPI**（Windows 通常不能热卸载）。

### 1. 编译插件

**Windows（PowerShell）**

```powershell
# 需要 PATH 里有 gcc
# 若提示“禁止运行脚本”，用下面任一方式：

# 方式 A：仅本次绕过执行策略（推荐）
powershell -ExecutionPolicy Bypass -File .\scripts\build.ps1

# 一键编译并复制到本机 CLIProxyAPI
powershell -ExecutionPolicy Bypass -File .\scripts\deploy.ps1

# 方式 B：不跑脚本，直接编译
$env:CGO_ENABLED = "1"
New-Item -ItemType Directory -Force -Path dist | Out-Null
go build -buildmode=c-shared -o dist\credit-manager.dll .

# 方式 C：当前用户永久放宽（可选）
# Set-ExecutionPolicy -Scope CurrentUser RemoteSigned
```

产出：`dist\credit-manager.dll` 以及同名 `.h`。

**Linux / macOS**

```bash
chmod +x scripts/build.sh
./scripts/build.sh
# Linux: dist/credit-manager.so
# macOS: dist/credit-manager.dylib
```

### 2. pepper（默认自动，一般不用管）

插件 Key 的 HMAC 依赖 pepper，**绝不进 SQLite**。

默认行为：

1. 若设置了环境变量 `CREDIT_MANAGER_KEY_PEPPERS`（且非空）→ 用环境变量  
2. 否则读 `data_dir/key-peppers`  
3. 文件不存在 → **首次启动自动生成 32 字节随机 pepper 并写入该文件**，之后一直复用  

```text
# 自动生成后的文件示例（权限 0600）
./data/credit-manager/key-peppers
# 内容形如：
# active:a1b2c3...（64 位 hex）
```

**备份 `data_dir` 时请一并备份 `key-peppers` 与 `bootstrap-api-key.txt`**。删掉 pepper 再启动会生成新 pepper，旧 Key 全部失效。

可选：用环境变量覆盖（运维/多机统一密钥时）：

```powershell
$env:CREDIT_MANAGER_KEY_PEPPERS = "active:0123456789abcdef0123456789abcdef"
```

```bash
export CREDIT_MANAGER_KEY_PEPPERS='active:0123456789abcdef0123456789abcdef'
```

格式：`id:pepper` 或多个 `id1:p1,id2:p2`（文件里也可一行一个）。  
`active_pepper_id` 指定签发新 Key 用哪个 id。

### 3. 放到 CLIProxyAPI 插件目录（宿主 config 可留空）

1. 复制动态库到宿主插件目录（Windows 示例：`D:\CLIProxyAPI\plugins\windows\amd64\credit-manager.dll`，或用 `scripts\deploy.ps1`）
2. 在 CPAMC / 管理中心 **插件管理** 打开 `credit-manager` 开关（或宿主 YAML `enabled: true`）
3. **重启** CLIProxyAPI（Windows 加载的 DLL 通常不能热替换）

**推荐宿主片段（零配置）**：

```yaml
plugins:
  enabled: true
  dir: ./plugins
  items:
    credit-manager:
      enabled: true
      # config 可完全省略；需要覆盖时再写
```

可选：用文件覆盖默认值（不要整段粘贴长 YAML）：

```yaml
items:
  credit-manager:
    enabled: true
    config: |
      config_file: E:/path/to/config.yaml
```

```bash
# 等价：环境变量指向外部配置
set CREDIT_MANAGER_CONFIG_FILE=E:\path\to\config.yaml
```

宿主把 `plugins.items.*.config` 作为 `config_yaml` 注入；本插件再解析 `config_file` / 环境变量。字段细节见 `config.example.yaml`。

同时确保：

- 宿主已配置**上游模型凭据**（OAuth / API Key 等）；插件只做鉴权与额度，不登录上游
- **独占前端鉴权**：启用后代理请求只认插件 `tk-v1-...`，宿主 `api-keys` 不再用于这些请求
- 默认 `data_dir`（`./data/credit-manager`）可写；含 `key-peppers` 与 SQLite

### 4. 启动后签发 Key

1. 重启宿主并确认插件已启用
2. 侧栏打开 **CPA 额度管理**（资源页 `/console`），手动签发 Key 并设置额度
3. 也可调用 `POST /v0/management/credit-manager/keys`；`quota_micro_usd` 为必填项
4. 可选探活（管理密钥，不是插件 Key）：

```bash
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/health" \
  -H "Authorization: Bearer <宿主管理密钥>"
```

端口与管理鉴权头以宿主配置为准。若侧栏没有菜单：确认开关已开、DLL 已是含 Resources 的新构建，并已重启。

### 5. 进阶：管理 API 改账本（生产）

零配置已自动创建 `default` 账户、免费全模型规则与一把 bootstrap Key。  
下面接口用于生产改配额 / 价格 / 多 Key。假设：

- 宿主地址：`http://127.0.0.1:8317`
- 管理密钥：`MGMT_TOKEN`（替换成你的）

#### 5.1 创建归属记录（caller）

一个 caller 用于组织 Key 与聚合使用记录，**不再承载额度**。可直接使用默认的 `default`，也可按团队创建。

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

金额单位为 **micro-USD**，仅用于 Key 的 `quota_micro_usd`：

- `1 USD = 1_000_000 micro-USD`

#### 5.2 配置模型价格

价格是 **每 100 万 token 的 micro-USD 单价**。

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

也可以用通配：

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
匹配时 **priority 大的优先**。

若配置 `pricing.unknown_policy: deny`，**没有匹配到价格规则的模型会直接拒请求**。

#### 5.3 签发插件 Key（只显示一次明文）

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
  "plaintext": "tk-v1-xxxx_yyyy",
  ...
}
```

**立刻保存 `plaintext`。之后任何接口都不会再返回完整密钥。**

#### 5.4 查 Key 额度 / 用量 / 审计

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

### 6. 客户端真正打模型

**不要用宿主 api-keys**；请通过侧栏或管理 API 手动签发带额度的 Key：

```bash
curl -sS "http://127.0.0.1:8317/v1/chat/completions" \
  -H "Authorization: Bearer tk-v1-xxxx_yyyy" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4o",
    "messages": [{"role":"user","content":"你好"}],
    "max_tokens": 256
  }'
```

请求过程中插件会：

1. 校验 Bearer Key  
2. 按模型价格 + 保守 token 上界做**严格预占**  
3. 调用宿主去打真实上游  
4. 从响应 usage 结算；若没有 usage，按 `settlement.missing_usage` 处理（默认按预占额结算）

额度不够 → 请求直接失败（预占阶段 fail-closed）。额度由每个 Key 独立管理，互不共享。

---

## 日常运维命令

### 调整 Key 额度

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/update" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>","quota_micro_usd":20000000}'
```

### 禁用 / 启用 Key

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/update" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>","enabled":false}'
```

### 撤销 Key

```bash
curl -sS -X POST "http://127.0.0.1:8317/v0/management/credit-manager/keys/revoke" \
  -H "Authorization: Bearer MGMT_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"id":"<plugin_key_id>"}'
```

### 列 Key（不含明文）

```bash
curl -sS "http://127.0.0.1:8317/v0/management/credit-manager/keys?caller_id=team-a" \
  -H "Authorization: Bearer MGMT_TOKEN"
```

---

## 管理 API 一览

所有接口都走宿主 `/v0/management` 鉴权。路径是**精确匹配**，ID 放 body 或 query，不放路径参数。

启用插件后，管理中心侧栏会出现 **「CPA 额度管理」**（资源页 `/v0/resource/plugins/credit-manager/console`）。页面内需填写宿主管理密钥（sessionStorage），即可可视化管理 Key、可用模型、限额与按 Key 统计。若没有菜单：确认已启用、DLL 已替换为最新构建，并重启宿主。

| 方法 | 路径 | 作用 |
|------|------|------|
| GET | `/v0/management/credit-manager/health` | 健康检查 |
| GET | `/v0/management/credit-manager/overview` | 控制台总览（归属记录/Key/价格/统计） |
| POST | `/v0/management/credit-manager/callers` | 创建 Key 归属记录 |
| GET | `/v0/management/credit-manager/callers` | 列出 Key 归属记录 |
| POST | `/v0/management/credit-manager/callers/enabled` | 启停归属记录 |
| POST | `/v0/management/credit-manager/keys` | 签发 Key（`quota_micro_usd` 必填，可带 `allowed_models`） |
| GET | `/v0/management/credit-manager/keys` | 列 Key（`caller_id` 可选） |
| POST | `/v0/management/credit-manager/keys/update` | 更新 Key 标签/启用/限额/可用模型 |
| POST | `/v0/management/credit-manager/keys/revoke` | 撤销 Key |
| POST | `/v0/management/credit-manager/pricing` | 新增/更新价格规则 |
| GET | `/v0/management/credit-manager/pricing` | 列价格规则 |
| POST | `/v0/management/credit-manager/pricing/delete` | 删除价格规则 |
| GET | `/v0/management/credit-manager/balance?key_id=` | 查 Key 剩余额度 |
| GET | `/v0/management/credit-manager/usage` | 用量流水（可按 `plugin_key_id` / `model` 过滤） |
| GET | `/v0/management/credit-manager/usage/summary` | 按 Key / 模型汇总 |
| GET | `/v0/management/credit-manager/audit?caller_id=` | 审计事件 |

---

## 插件配置说明

完整示例见 `config.example.yaml`。

| 字段 | 含义 |
|------|------|
| `data_dir` | 插件自己的数据目录（SQLite、锁文件、pepper 文件） |
| `database_file` | 库文件名，默认 `credit-manager.db` |
| `busy_timeout` | SQLite 忙等待，如 `5s` |
| `keys.pepper_env` | 可选：pepper 环境变量名；有值时优先于文件 |
| `keys.pepper_file` | pepper 文件（相对 `data_dir` 或绝对路径），默认 `key-peppers`；不存在则首次自动生成 |
| `keys.active_pepper_id` | 签发新 Key 使用的 pepper id |
| `limits.max_token_estimate` | 单请求 token 预估上限（超了直接拒） |
| `limits.default_output_reserve` | body 没写 max_tokens 时默认输出预占 |
| `pricing.unknown_policy` | 无价格规则时：`deny` / `allow` / `default`（零配置默认 `allow`） |
| `settlement.missing_usage` | 无 usage 时：`settle_reserved` 或 `release` |
| `stream.max_buffer_bytes` | 流式结算时本地缓冲上限 |

---

## 金额与计费规则

- 账本只存整数 **micro-USD**，不存浮点金额  
- 价格是 **每 1M tokens 的 micro-USD**  
- 费用按 token 类别分别 **向上取整** 再相加  
- **预占严格**：每个 Key 剩余额度不足时直接拒绝，Key 之间互不共享
- **结算可超过预占额**：真实 usage 绝不能丢；对应 Key 余额可能为负，之后该 Key 的请求继续 fail-closed

### 价格字段

| 字段 | 含义 |
|------|------|
| `input` | 输入 token |
| `output` | 输出 token |
| `reasoning` | 推理 token |
| `cached` | 通用缓存 token |
| `cache_read` | 缓存读取 |
| `cache_creation` | 缓存创建 |

---

## 密钥安全

- 明文形态：`tk-v1-<kid>-<secret>`
- 数据库只存：`kid`、HMAC、pepper id、指纹、principal、caller_scope
- pepper 只在环境变量或 `data_dir/key-peppers`，不进 SQLite / 日志 / 管理查询
- 创建 Key 的响应带 `Cache-Control: no-store`

### pepper 轮换建议

1. 在 pepper 文件或环境变量里**追加**新 pepper：`newid:....,active:....`（或换行）
2. 配置 `active_pepper_id: newid`
3. 重启宿主，给用户签发新 Key
4. 旧 pepper 保留到所有旧 Key 退役后再删

---

## 架构（请求怎么走）

```text
客户端
  |  Authorization: Bearer tk-v1-...
  v
CLIProxyAPI
  |  frontend_auth.authenticate  → 本插件校验 Key，返回 principal
  |  model.route                 → 路由到本插件 executor
  |  executor.execute / stream
  |     1) 查 Key，算保守费用，按 Key 原子预占
  |     2) host.model.execute(_stream) 透明转发上游
  |     3) 解析 usage，结算或按策略兜底
  v
上游模型
```

宿主会跳过“当前插件自己”作为上游执行器，避免递归。

---

## 运维注意

1. **单写者**：同一数据库不要多进程同时写；会有 `*.lock` 排他锁  
2. **备份**：备份整个 `data_dir`（SQLite、`key-peppers`）；最好停写或用 SQLite 在线备份  
3. **Windows 升级插件**：替换 DLL 后必须重启宿主；仅开关/重载配置一般无需重启  
4. **独占鉴权副作用**：开了本插件后，旧的宿主 `api-keys` 不再用于代理鉴权；请改用插件 Key  
5. **未知模型**：零配置默认 `unknown_policy: allow` 且已 seed 全模型 0 价规则；生产请改真实价格，必要时改 `deny`  
6. **侧栏菜单**：由插件 `Resources` 注册（「CPA 额度管理」），不是 API 路由自动生成；未启用插件时不会出现  

---

## 开发自检（不产出动态库）

```bash
go build ./internal/...
go vet ./internal/...
go run ./scripts/smoke_ledger.go
```

`smoke_ledger` 会在临时目录验证：创建归属记录、签发带额度的 Key、鉴权、按 Key 预占、usage 结算、撤销。

---

## 常见问题

**Q: 编译报 `gcc not found`？**  
A: 安装 MinGW-w64 / MSYS2 的 gcc，并加入 PATH。

**Q: 管理接口 401？**  
A: 管理路由走宿主管理鉴权，不是插件 Key。用宿主 management token。

**Q: 代理请求 401？**  
A: 检查是否用了 `tk-v1-...`，以及 pepper 是否与签发时一致（环境变量或 `data_dir/key-peppers` 未被替换/删除）。

**Q: 请求被拒 “no pricing rule”？**  
A: 零配置应已有 `.*` 免费规则。若你删光了规则且把 `unknown_policy` 设为 `deny`，需重新加价格规则或改回 `allow`/`default`。

**Q: 侧栏没有「CPA 额度管理」？**  
A: 插件开关必须为 ON；确认已部署含 console Resource 的 DLL。仅替换 DLL 后才需要重启宿主。CPAMC 只在启用后展示插件菜单。

**Q: 管理中心显示「未注册」，开关后仍不恢复？**  
A: 旧版本 reconfigure 会二次抢 `*.db.lock` 失败。请部署含本修复的 DLL（替换后需重启一次加载新二进制）；之后正常开关/重载应可恢复注册。

**Q: Key 额度突然变负？**  
A: 真实 usage 超过预占时会如实结算，这是故意的，避免漏计费；该 Key 之后会因额度不足被拒，直到上调该 Key 的额度。

**Q: 客户端还要用原来的 api-keys 吗？**  
A: 不用。本插件独占前端鉴权后，代理请求只认插件 Key。