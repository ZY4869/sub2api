## common
> 本页说明整个网关的统一接入规则。后续左侧协议页会分别展开 OpenAI 原生、OpenAI 兼容、Anthropic / Claude、Gemini 原生、Grok、Antigravity、Vertex / Batch，以及百度智能文档的细节。

### 概览

Sub2API 是一个多协议聚合网关。你面对的是一套统一站内 API Key，但可以按客户端需要选择不同的协议面来访问上游能力。

建议把文档理解成两层：

- 第一层是“入口协议”：你用什么客户端、发送什么格式、走哪条路径。
- 第二层是“运行时平台”：当前 Key 所绑定的分组最终调度到哪个平台，例如 OpenAI、Anthropic、Gemini、Grok、Antigravity 或百度智能文档。

协议页固定分成以下 9 个子页：

| 协议页 ID | 页面名称 | 推荐使用者 | 重点内容 |
| --- | --- | --- | --- |
| `common` | 通用接入 | 所有调用方 | 认证、基础地址、错误、限流、模型目录 |
| `openai-native` | OpenAI 原生 | 新版 OpenAI SDK、Responses-first 客户端 | `responses`、子资源、长连接建议 |
| `openai` | OpenAI 兼容 | 旧版 OpenAI SDK、历史兼容客户端 | `chat/completions`、历史别名、兼容迁移 |
| `anthropic` | Anthropic / Claude | Claude SDK、Claude Code、Anthropic 风格客户端 | `messages`、`count_tokens`、保留头 |
| `gemini` | Gemini 原生 | Gemini SDK、AI Studio / Vertex 风格客户端 | `models`、`files`、`batches`、`live`、`openai compat` |
| `grok` | Grok | xAI / Grok 兼容接入 | 聊天、Responses、图像、视频 |
| `antigravity` | Antigravity | 需要显式绑定 Antigravity 平台的接入方 | Anthropic 风格入口 + Gemini 风格入口 |
| `vertex-batch` | Vertex / Batch | 使用站内 Vertex / Batch 简化入口或严格兼容入口的调用方 | `/v1/vertex/...`、`/vertex-batch/jobs...`、严格 `/v1/projects/...`、统一 archive 回查 |
| `document-ai` | 百度智能文档 | 百度智能文档 / OCR 调用方 | 直连解析、异步任务、模型模式差异 |

### 快速接入

接入时建议按下面顺序做，不要一开始就混用多种协议路径：

1. 在站内创建一个可用的 API Key。
2. 先确定你要模拟的协议，而不是先决定模型名。
3. 把客户端 `Base URL` 指向 `https://api.zyxai.de`。
4. 选择对应协议推荐的认证头。
5. 先跑通一个最短请求，再扩展流式、上传、批任务和工具调用。

下面给出一个最短可联通的 smoke test，统一走 `OpenAI Responses`，因为它是当前最稳妥的公共文本入口之一。

#### Python
```python focus=3-12,15-16
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"

response = requests.post(
    f"{base_url}/v1/responses",
    headers={
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    },
    json={
        "model": "gpt-5.4",
        "input": "请用一句话确认网关已经联通。",
    },
    timeout=60,
)

print(response.status_code)
print(response.json())
```

#### JavaScript
```javascript focus=1-10,12-13
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";

const response = await fetch(`${baseUrl}/v1/responses`, {
  method: "POST",
  headers: {
    Authorization: `Bearer ${apiKey}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "gpt-5.4",
    input: "请用一句话确认网关已经联通。",
  }),
});

console.log(response.status, await response.json());
```

#### REST
```bash focus=1-5
curl https://api.zyxai.de/v1/responses \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "input": "请用一句话确认网关已经联通。"
  }'
```

### 基础地址与认证

基础地址统一填写网关根地址，不建议把固定协议路径写死到 `Base URL` 里：

```text
https://api.zyxai.de
```

认证分成两套：

| 适用范围 | 推荐认证方式 | 兼容方式 | 说明 |
| --- | --- | --- | --- |
| OpenAI 原生 / OpenAI 兼容 / Anthropic / Grok / Antigravity | `Authorization: Bearer <API_KEY>` | `x-api-key`、`x-goog-api-key` | 适合大多数 SDK 和代理工具 |
| Gemini / Vertex / Batch 站内推荐入口 | `Authorization: Bearer <API_KEY>` | `x-goog-api-key`、`x-api-key`、部分路径支持 `?key=` | 新接入用户优先；`/v1/vertex/...` 与 `/vertex-batch/jobs...` 默认按这一套接入 |
| 原生 Gemini / Google SDK 兼容入口 | `x-goog-api-key: <API_KEY>` | `Authorization: Bearer`、`x-api-key`、部分路径支持 `?key=` | 当你直接复用 Gemini / Google 风格客户端时更省改造 |

- 虽然后端 Google / Gemini 风格中间件仍优先读取 `x-goog-api-key`，但文档默认建议新接入的 Vertex / Batch 简化入口统一使用 `Authorization: Bearer`。

查询参数的规则必须特别注意：

- `?api_key=...`：整个系统都视为废弃写法。
- `?key=...`：只在 Google / Gemini 风格白名单路径上保留兼容，不适用于 OpenAI 原生、OpenAI 兼容、Anthropic、Grok、`/v1/vertex/...`、`/vertex-batch/jobs...`、严格 Vertex 路径或 archive 路径。
- 对于 `/v1/vertex/...`、`/vertex-batch/jobs...`、`/v1/projects/:project/locations/:location/...` 和 `/google/batch/archive/...`，请使用请求头，不要依赖 `?key=...`。

当前程序对认证头的优先级如下：

- 普通协议中间件：`Authorization: Bearer` -> `x-api-key` -> `x-goog-api-key` -> 允许时的 `?key=`
- Google / Gemini 风格中间件：`x-goog-api-key` -> `Authorization: Bearer` -> `x-api-key` -> 允许时的 `?key=`

### 公共模型库

系统另外提供一个只读公共目录接口：

- 路径：`GET /api/v1/meta/model-catalog`
- 详情：`GET /api/v1/meta/model-catalog/:model`
- 鉴权：无需登录，游客与已登录用户都可访问
- 用途：返回前台 `/models` 页面使用的公开模型目录，包含供应商、请求协议族、基础出售价格、倍率摘要，以及目录配置中的 `page_size`
- 发布语义：有已发布快照时，接口固定返回最近一次“推送更新”冻结下来的正式目录，列表内容、排序、分页大小与详情示例都会以该版本为准
- 未发布语义：如果当前还没有任何已发布快照，接口会自动回退到实时目录构建结果，而不是返回空目录；详情接口也会同步回退到实时详情
- 详情语义：`GET /api/v1/meta/model-catalog/:model` 在 `catalog_source=published` 时返回发布时固化的单模型价格块与调用示例元数据；在 `catalog_source=live_fallback` 时返回当前实时目录对应的详情结果
- 缓存：响应会返回 `ETag`，客户端可通过 `If-None-Match` 复用 `304 Not Modified`

这个接口只暴露展示所需的公共目录数据，不替代具体协议页中的 `/v1/models`、`/v1beta/models` 等运行时模型枚举接口。

公共目录与运行时模型枚举接口共用同一套账号模型投影规则，必须记住下面五条：

- 账号模型集合只来自两层：账号显式白名单 / 映射，或默认模型库。探测结果、已知模型快照和 saved snapshot 只能补充状态，不会扩展列表。
- 账号级白名单 / 取模勾选会直接影响 downstream `/v1/models`、`/v1beta/models` 的返回结果；未被该账号允许的模型不会出现在列表或详情里。
- 如果某个账号把真实模型配置成了自定义映射名，那么 downstream `models list` 与 `models detail` 只返回映射名这个 display ID；`target model` 只保留在内部转发链路和后台诊断里。
- 模型列表读路径只读取本地策略投影和本地 availability snapshot。即使 snapshot 缺失或过期，也只会返回现有投影并标记状态，不会在读请求里同步触发实时探测。
- 没有有效售价的模型不会出现在公共目录里，也不会出现在用户创建 / 编辑 Key 时的模型选择器里。
- `/v1/models`、`/v1beta/models` 以及复用同一公共模型读路径的详情接口，都是“运行时可服务视图”，不是永久静态目录；如果当前所有可路由账号都因为同一类运行时额度冷却而暂时无法服务某个模型，该模型会临时从列表和详情里隐藏，额度恢复后会自动重新出现。
- 这类运行时隐藏不会反写账号白名单、`model_scope_v2`、probe snapshot 或 manual whitelist；它只是读路径上的临时过滤。
- 对 OpenAI Pro 来说，运行时额度侧是分开的：`gpt-5.3-codex-spark*` 只看 `Spark` 侧，其它 OpenAI 模型统一看 `普通` 侧，所以某一侧冷却时通常只会临时隐藏对应那一侧的模型。

`GET /api/v1/meta/model-catalog` 当前返回体额外包含：

- `etag`：本次已发布快照的版本标识
- `etag`：当前有效目录快照的版本标识
- `updated_at`：当前目录快照的更新时间；若命中已发布版本则表示最近一次发布时间，若回退实时则表示实时目录构建时间
- `page_size`：公开模型库前台默认每页数量；命中已发布版本时使用发布快照中的固定值，回退实时时使用当前默认页大小
- `catalog_source`：目录来源，固定为 `published` 或 `live_fallback`
- `items`：公开模型数组，卡片标题应优先展示 `display_name`

典型响应示例：

```json
{
  "etag": "W/\"4b0c0d...\"",
  "updated_at": "2026-04-21T10:05:00Z",
  "page_size": 10,
  "catalog_source": "published",
  "items": [
    {
      "model": "gpt-5.4",
      "display_name": "GPT-5.4",
      "provider": "openai",
      "provider_icon_key": "openai",
      "request_protocols": ["openai"],
      "mode": "chat",
      "currency": "USD",
      "price_display": {
        "primary": [
          { "id": "input_price", "unit": "input_token", "value": 0.0000012 },
          { "id": "output_price", "unit": "output_token", "value": 0.0000024 }
        ]
      },
      "multiplier_summary": {
        "enabled": false,
        "kind": "disabled"
      }
    }
  ]
}
```

`GET /api/v1/meta/model-catalog/:model` 返回的 `example_*` 字段来自发布时冻结的详情快照，典型结构如下：

```json
{
  "item": {
    "model": "gpt-5.4",
    "display_name": "GPT-5.4",
    "provider": "openai",
    "currency": "USD",
    "price_display": {
      "primary": [
        { "id": "input_price", "unit": "input_token", "value": 0.0000012 },
        { "id": "output_price", "unit": "output_token", "value": 0.0000024 }
      ]
    },
    "multiplier_summary": {
      "enabled": false,
      "kind": "disabled"
    }
  },
  "catalog_source": "published",
  "example_source": "docs_section",
  "example_protocol": "openai",
  "example_page_id": "common",
  "example_markdown": "```bash\\ncurl https://api.zyxai.de/v1/responses ...\\n```"
}
```

已登录用户另外还有一个仅用于 Key 编辑器的辅助接口：

- 路径：`GET /api/v1/groups/model-options`
- 鉴权：必须登录
- 用途：返回当前用户可绑定分组下、且当前具备有效价格的公开模型列表；普通用户保存 Key 时会把结构化勾选结果写回 `groups[].model_patterns`
- 语义：如果某个分组绑定没有提交 `model_patterns`，表示这个 Key 在该分组下可以调用全部公开模型；多个分组绑定的最终可调用模型集合取并集
- 回退语义：如果正式公开目录尚未发布，这个接口会和前台 `/models` 一样自动回退到实时目录，但仍会继续过滤掉没有有效售价的模型

如果前台已经处在明确的分组上下文里，还可以使用倍率后的展示价接口：

- 路径：`GET /api/v1/groups/model-catalog?group_id=<GROUP_ID>`
- 鉴权：必须登录
- 用途：返回“当前有效公开模型目录 + 指定分组倍率换算后的 `price_display`”；结构与 `/api/v1/meta/model-catalog` 保持一致，只替换价格字段
- 约束：这里只用于用户自己的分组上下文页面；如果存在已发布快照则优先使用已发布基础售价，没有已发布快照时才回退实时目录

下面的例子分别展示三种常用认证写法。

#### Python
```python focus=4-8,11-12
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"

requests.get(
    f"{base_url}/v1/models",
    headers={"x-goog-api-key": api_key},
    timeout=30,
)

# 如果账号把 gemini-2.0-flash 映射成 friendly-flash，
# 这里的列表只会返回 friendly-flash，而不会返回真实模型名。
```

#### JavaScript
```javascript focus=1-9
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";

await fetch(`${baseUrl}/v1/messages`, {
  method: "POST",
  headers: {
    Authorization: `Bearer ${apiKey}`,
    "Content-Type": "application/json",
    "anthropic-version": "2023-06-01",
  },
  body: JSON.stringify({
    model: "claude-sonnet-4-20250514",
    max_tokens: 128,
    messages: [{ role: "user", content: "你好" }],
  }),
});
```

#### REST
```bash focus=1-2
curl "https://api.zyxai.de/v1beta/models?key=sk-你的站内Key" \
  -H "Content-Type: application/json"
# 如果账号配置了自定义映射名，返回体里只会出现映射名。
```

补充说明：

- `public_model_catalog_enabled` 默认开启。
- 开启时，游客与已登录用户都可以访问 `GET /api/v1/meta/model-catalog`。
- 关闭时，游客访问该接口会返回 `401`，前端 `/models` 页面会跳转到登录页。
- 已登录用户不受这个开关影响，仍然可以继续访问模型库与对应接口。
- `GET /api/v1/settings/public` 会额外返回 `maintenance_mode_enabled`、`available_channels_enabled`、`channel_monitor_enabled`、`affiliate_enabled`，前端可据此决定是否展示维护提示与菜单入口。

### 可用渠道（Available Channels）

系统提供一个用户视角的“可用渠道”只读接口，用于前端快速展示当前 Key / 分组可用的渠道聚合视图：

- 路径：`GET /api/v1/channels/available`
- 鉴权：必须登录
- 开关：`available_channels_enabled`（默认 `false`，关闭时该接口返回空数组）
- 返回：只包含展示必要字段，不包含管理侧敏感字段（例如渠道内部配置、secret、状态细节等）

#### Python
```python
import requests

base_url = "https://api.zyxai.de"
jwt = "你的JWT"

resp = requests.get(
    f"{base_url}/api/v1/channels/available",
    headers={"Authorization": f"Bearer {jwt}"},
    timeout=30,
)

print(resp.status_code)
print(resp.json())
```

#### JavaScript
```javascript
const baseUrl = "https://api.zyxai.de";
const jwt = "你的JWT";

const resp = await fetch(`${baseUrl}/api/v1/channels/available`, {
  headers: { Authorization: `Bearer ${jwt}` },
});

console.log(resp.status, await resp.json());
```

#### REST
```bash
curl "https://api.zyxai.de/api/v1/channels/available" \
  -H "Authorization: Bearer <JWT>"
```

### 渠道监控（Channel Monitor）

系统提供一套“渠道监控”能力，用户侧可以读取监控状态页，管理员侧可以配置监控与模板并触发运行。

用户侧只读接口：

- 列表：`GET /api/v1/channel-monitors`
- 详情：`GET /api/v1/channel-monitors/:id/status`
- 鉴权：必须登录
- 开关：`channel_monitor_enabled`（默认 `false`）
- 语义：
  - 关闭时：列表接口返回空数组；详情接口返回 `404`
  - 开启时：返回监控概览与每个监控的最近状态、可用率摘要与简化时间线

#### Python
```python
import requests

base_url = "https://api.zyxai.de"
jwt = "你的JWT"

items = requests.get(
    f"{base_url}/api/v1/channel-monitors",
    headers={"Authorization": f"Bearer {jwt}"},
    timeout=30,
).json()

print(items)
```

#### JavaScript
```javascript
const baseUrl = "https://api.zyxai.de";
const jwt = "你的JWT";

const resp = await fetch(`${baseUrl}/api/v1/channel-monitors`, {
  headers: { Authorization: `Bearer ${jwt}` },
});

console.log(resp.status, await resp.json());
```

#### REST
```bash
curl "https://api.zyxai.de/api/v1/channel-monitors" \
  -H "Authorization: Bearer <JWT>"
```

### 邀请返利（Affiliate / Invite Rebate）

邀请返利是站内可选能力，默认关闭。开启后，系统会为每个用户生成一个 `aff_code`（返利码），新用户注册时可携带 `aff_code` 绑定邀请关系。被邀请用户产生“消耗计费”或“正向入金”时，系统会按设置的比例给邀请者累计返利。

核心开关与规则均在 `GET /api/v1/admin/settings` 中配置：

- `affiliate_enabled`：总开关（默认 `false`）。关闭时不绑定、不累计；前端入口也应隐藏。
- `affiliate_transfer_enabled`：转入余额开关（默认 `true`）。仅控制用户 `transfer` API 是否可用。
- `affiliate_rebate_on_usage_enabled` / `affiliate_rebate_on_topup_enabled`：分别控制“按消耗返利 / 按入金返利”。
- `affiliate_rebate_rate`：全局默认返利比例（0–100）。
- `affiliate_rebate_freeze_hours`：冻结期（小时），大于 0 时返利先进入冻结余额。
- `affiliate_rebate_duration_days`：有效期（天），0 表示永久；用于限制 invitee 产生返利的窗口。
- `affiliate_rebate_per_invitee_cap`：单人上限，0 表示不限；限制单 invitee 对 inviter 的累计返利。
- `affiliate_aff_code_length`：返利码长度（用于新生成返利码）。

Public settings 只额外暴露：

- `GET /api/v1/settings/public` 返回 `affiliate_enabled`，前端可据此在注册页与侧边栏做严格 gating（`=== true` 才显示）。

#### 注册：可选 aff_code

注册请求新增可选字段 `aff_code?: string`（不替代已有 `invitation_code`）：

- 路径：`POST /api/v1/auth/register`
- 说明：当 `affiliate_enabled=true` 且填写了 `aff_code` 时，系统会尝试绑定邀请关系；绑定失败不会阻断注册。

#### Python
```python
import requests

base_url = "https://api.zyxai.de"

resp = requests.post(
    f"{base_url}/api/v1/auth/register",
    json={
        "email": "new-user@example.com",
        "password": "your-password",
        "aff_code": "ABC123XYZ9",
    },
    timeout=30,
)

print(resp.status_code)
print(resp.json())
```

#### JavaScript
```javascript
const baseUrl = "https://api.zyxai.de";

const resp = await fetch(`${baseUrl}/api/v1/auth/register`, {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({
    email: "new-user@example.com",
    password: "your-password",
    aff_code: "ABC123XYZ9",
  }),
});

console.log(resp.status, await resp.json());
```

#### REST
```bash
curl "https://api.zyxai.de/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{
    "email": "new-user@example.com",
    "password": "your-password",
    "aff_code": "ABC123XYZ9"
  }'
```

#### 用户端：查看返利信息

- 路径：`GET /api/v1/user/aff`
- 鉴权：必须登录（JWT）
- 返回：是否开启、是否允许转入、我的返利码、邀请人数、可转余额/冻结余额、累计返利、实际生效比例、以及当前规则开关。

#### Python
```python
import requests

base_url = "https://api.zyxai.de"
jwt = "你的JWT"

resp = requests.get(
    f"{base_url}/api/v1/user/aff",
    headers={"Authorization": f"Bearer {jwt}"},
    timeout=30,
)

print(resp.status_code)
print(resp.json())
```

#### JavaScript
```javascript
const baseUrl = "https://api.zyxai.de";
const jwt = "你的JWT";

const resp = await fetch(`${baseUrl}/api/v1/user/aff`, {
  headers: { Authorization: `Bearer ${jwt}` },
});

console.log(resp.status, await resp.json());
```

#### REST
```bash
curl "https://api.zyxai.de/api/v1/user/aff" \
  -H "Authorization: Bearer <JWT>"
```

#### 用户端：转入余额（transfer）

- 路径：`POST /api/v1/user/aff/transfer`
- 鉴权：必须登录（JWT）
- 语义：将可转入的返利余额原子转入用户主余额；重复点击不会重复转入（`transferred_amount=0`）。
- 开关：受 `affiliate_transfer_enabled` 控制（独立于 `affiliate_enabled`）。

#### Python
```python
import requests

base_url = "https://api.zyxai.de"
jwt = "你的JWT"

resp = requests.post(
    f"{base_url}/api/v1/user/aff/transfer",
    headers={"Authorization": f"Bearer {jwt}"},
    timeout=30,
)

print(resp.status_code)
print(resp.json())
```

#### JavaScript
```javascript
const baseUrl = "https://api.zyxai.de";
const jwt = "你的JWT";

const resp = await fetch(`${baseUrl}/api/v1/user/aff/transfer`, {
  method: "POST",
  headers: { Authorization: `Bearer ${jwt}` },
});

console.log(resp.status, await resp.json());
```

#### REST
```bash
curl -X POST "https://api.zyxai.de/api/v1/user/aff/transfer" \
  -H "Authorization: Bearer <JWT>"
```

#### 管理端：返利用户管理

管理员可通过以下接口查询与运营返利用户（需要管理员 JWT）：

- 列表：`GET /api/v1/admin/affiliates/users?page=1&page_size=20&has_custom_code=1&has_custom_rate=1&has_inviter=1`
- 查询：`GET /api/v1/admin/affiliates/users/lookup?q=<user_id|email|aff_code>`
- 设置专属：`PUT /api/v1/admin/affiliates/users/:user_id`
- 撤销专属：`DELETE /api/v1/admin/affiliates/users/:user_id`
- 批量设置比例：`POST /api/v1/admin/affiliates/users/batch-rate`

`PUT` 更新说明：

- `custom_rebate_rate_percent`：可传 `null` 表示清空（回退到全局默认）。
- `aff_code`：传空字符串 `\"\"` 表示清空专属返利码（回退到默认随机码）。

#### Python
```python
import requests

base_url = "https://api.zyxai.de"
admin_jwt = "你的管理员JWT"

# lookup
items = requests.get(
    f"{base_url}/api/v1/admin/affiliates/users/lookup?q=alice",
    headers={"Authorization": f"Bearer {admin_jwt}"},
    timeout=30,
).json()

print(items)

# batch-rate
resp = requests.post(
    f"{base_url}/api/v1/admin/affiliates/users/batch-rate",
    headers={"Authorization": f"Bearer {admin_jwt}"},
    json={"user_ids": [101, 102, 103], "custom_rebate_rate_percent": 25.0},
    timeout=30,
)
print(resp.status_code, resp.json())
```

#### JavaScript
```javascript
const baseUrl = "https://api.zyxai.de";
const adminJwt = "你的管理员JWT";

const lookup = await fetch(`${baseUrl}/api/v1/admin/affiliates/users/lookup?q=alice`, {
  headers: { Authorization: `Bearer ${adminJwt}` },
});
console.log(await lookup.json());

const resp = await fetch(`${baseUrl}/api/v1/admin/affiliates/users/batch-rate`, {
  method: "POST",
  headers: { Authorization: `Bearer ${adminJwt}`, "Content-Type": "application/json" },
  body: JSON.stringify({ user_ids: [101, 102, 103], custom_rebate_rate_percent: 25.0 }),
});
console.log(resp.status, await resp.json());
```

#### REST
```bash
# 1) lookup
curl "https://api.zyxai.de/api/v1/admin/affiliates/users/lookup?q=alice" \
  -H "Authorization: Bearer <ADMIN_JWT>"

# 2) update user custom settings
curl -X PUT "https://api.zyxai.de/api/v1/admin/affiliates/users/123" \
  -H "Authorization: Bearer <ADMIN_JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "aff_code": "CUSTOMCODE1",
    "custom_rebate_rate_percent": 30.0
  }'

# 3) reset user custom settings
curl -X DELETE "https://api.zyxai.de/api/v1/admin/affiliates/users/123" \
  -H "Authorization: Bearer <ADMIN_JWT>"

# 4) batch rate
curl -X POST "https://api.zyxai.de/api/v1/admin/affiliates/users/batch-rate" \
  -H "Authorization: Bearer <ADMIN_JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_ids": [101, 102, 103],
    "custom_rebate_rate_percent": 25.0
  }'
```

### 错误响应与限流

错误体会按协议风格返回，而不是统一强行包成一种格式。

OpenAI / Anthropic 风格常见于：

- `/v1/responses`
- `/v1/chat/completions`
- `/v1/messages`
- `/grok/v1/...`
- `/antigravity/v1/...`

Gemini / Google 风格常见于：

- `/v1/models`
- `/v1beta/...`
- `/v1alpha/authTokens`
- `/upload/v1beta/...`
- `/download/v1beta/...`
- `/google/batch/archive/...`
- `/v1/projects/:project/locations/:location/...`

常见状态码与含义：

| 状态码 | 含义 | 典型原因 |
| --- | --- | --- |
| `400` | 请求参数错误 | 使用了废弃的 `api_key` 查询参数、JSON 非法、动作与路径不匹配 |
| `401` | 鉴权失败 | Key 缺失、Key 无效、用户被禁用、用户不活跃 |
| `403` | 权限或余额不足 | 没有有效订阅、余额不足、Key 过期 |
| `404` | 当前平台不支持该动作 | 路径存在，但当前运行平台不支持这个协议动作 |
| `429` | 窗口限流或额度耗尽 | Key 额度耗尽、订阅窗口触发、上游平台限流 |
| `503` | 维护模式或服务暂不可用 | 系统维护开启时，非管理员请求统一返回维护提示 |
| `500` | 内部错误 | 网关内部异常或上游转发失败 |

需要特别关注的两类限制：

- API Key 自身限制：已过期、额度耗尽、IP 白名单 / 黑名单、用户状态异常。
- 分组 / 订阅限制：按日、按周、按月窗口限制，或订阅不存在、余额不足。
- OpenAI Pro 运行时额度侧限制：如果某个 OpenAI 模型在当前所有可路由账号上都只因为对应的 `Spark` / `普通` 额度侧冷却而不可服务，`/v1/responses`、`/v1/chat/completions`、`/v1/messages` 会返回 `429 rate_limit_error`，而不是把它视为永久不存在。
- 维护模式限制：管理员后台、管理员 JWT、管理员用户名下 API Key 调用继续放行；普通用户接口、自助认证流、普通 API Key / 百度智能文档 Key 调用统一返回 `503`。
- 维护模式文案固定为：`维护模式开启中，恢复时间请关注官网公告或官方频道`。
- 普通 JSON 接口会继续使用现有统一错误结构，并附带错误码 `MAINTENANCE_MODE_ACTIVE`；Google / Gemini 风格接口保持 Google 风格错误体，`status` 为 `UNAVAILABLE`。
- 对于上面这种 OpenAI Pro 运行时冷却，模型枚举读路径会临时隐藏对应模型，但真实调用失败仍然返回 `429`，不会返回 `404`。

下面的例子展示如何在三种环境中读取错误体，而不是只看 HTTP 状态码。

#### Python
```python focus=3-7
import requests

response = requests.get("https://api.zyxai.de/v1beta/corpora")
payload = response.json()

if response.status_code >= 400:
    print("status:", response.status_code)
    print("error:", payload.get("error"))
```

#### JavaScript
```javascript focus=1-9
const response = await fetch("https://api.zyxai.de/v1/messages", {
  method: "POST",
  headers: { "Content-Type": "application/json" },
  body: JSON.stringify({}),
});

const payload = await response.json();
if (!response.ok) {
  console.log("status", response.status);
  console.log("error", payload.error ?? payload);
}
```

#### REST
```bash focus=1
curl https://api.zyxai.de/v1beta/test?api_key=legacy
```

### 模型与路径兼容差异

不要把“模型名可用”误解成“所有协议入口都可用”。协议兼容是按“入口路径 + 动作 + 当前运行平台”共同决定的。

下面这张表是最实用的判断方式：

| 你要做的事 | 优先入口 | 说明 |
| --- | --- | --- |
| 通用文本生成 | `/v1/responses` | 新项目优先 |
| OpenAI / Grok / Gemini 原生图片生成 | `/v1/images/generations` | 公共智能入口，按模型策略与 provider 元数据分流 |
| OpenAI / Codex tool 生图 | `/v1/responses` + `tools:[{type:"image_generation"}]` | `gpt-5.4`、`gpt-5.4-mini` 这类主模型优先用这个模式 |
| 旧 OpenAI 客户端兼容 | `/v1/chat/completions` | 旧生态广泛支持 |
| Claude 风格接入 | `/v1/messages` | 原生 Anthropic / Claude 最稳 |
| Gemini 原生生成 | `/v1beta/models/{model}:generateContent` | 原生 Google 风格 |
| Gemini 文件 / Batch / Live | `/v1beta/files`、`/v1beta/batches`、`/v1beta/live` | 走 Gemini 原生页 |
| Grok 图像 / 视频 | `/grok/v1/...` | 更明确，排错更容易 |
| 显式只走 Antigravity | `/antigravity/...` | 禁止混合调度 |
| Vertex / Batch / Archive | `/v1/vertex/...`、`/vertex-batch/jobs...` | 新接入优先；严格 `/v1/projects/...` 只留给 SDK 兼容，结果归档继续走 `/google/batch/archive/...` |
| 百度智能文档解析 | `/document-ai/v1/...` | 优先区分 `async` 与 `direct` |

跨协议兼容也存在，但不是无条件开放：

- `/v1/messages` 在 OpenAI / Copilot 平台下可能被翻译到 Responses。
- `/v1/messages/count_tokens` 只应当期望在 Anthropic 原生平台成功。
- `/v1/responses` 在 Grok 平台可以工作，但 Responses 的 WebSocket / 长连接模式不应对 Grok 做乐观假设。
- `/v1/responses` 额外支持一组“网关兼容扩展”的生图写法：`$imagegen ...` 简写、`image_generation` / `reference_images` / `mask` 扩展字段、`multipart/form-data` 直传 `reference_image`，以及 **JSON 下的 `model=gpt-image-2` 生图简写**；标准官方 Responses JSON 仍然原样可用。
- `/v1/images/generations`、`/v1/images/edits` 现在是公共智能图片入口：会先按当前 Key 的本地模型策略与 provider 元数据判断要落到 OpenAI、Grok 还是 Gemini。
- 对 OpenAI 的 `/v1/images/*` 来说，图片协议不靠 `model=gpt-image-2` 猜测，而是由账号 / Protocol Gateway / 分组三层配置共同决定；优先级是“分组强制模式 > 账号模式 > 默认策略”。
- OpenAI OAuth `free` 计划默认只开原生图片链路，不开放 compat 图片链路；如果分组强制 compat 但账号没有 compat 权限，接口会直接返回 `403 forbidden_error`，错误码 `image_compat_not_allowed`。
- 当 OpenAI 图片模式是 `compat` 时，`/v1/images/generations`、`/v1/images/edits` 会桥接到兼容执行链；`/v1/responses` 的 `image_generation` tool 会统一按 `gpt-image-2` 目标图片能力计费与审计，顶层文本模型（例如 `gpt-5.4-mini`）仍默认保持原样；但当你使用 `model=gpt-image-2` 的 JSON 简写时，网关会把上游顶层 `model` 内部路由为 `gpt-5.4-mini`（对外响应仍显示 `gpt-image-2`）。
- `/v1/responses` 的 `image_generation` tool 不会把显式非 OpenAI 图片模型静默转成 Codex / `gpt-image-2`；如果 tool `model` 解析不到 OpenAI GPT image profile，会返回 `400 image_tool_model_provider_unsupported`，请使用 Grok 或 Gemini 的专用图片端点。
- OpenAI 图片请求在进上游前还会统一套用能力矩阵：当前实现会先按 `image_protocol_mode + 目标图片模型` 判定 `generate/edit`、`stream`、`partial_images`、多图、`mask`、背景、输出格式 / 压缩率，再校验尺寸，并在 trace 里记录 `image_capability_profile`。
- 对 GPT image profile（版本化 `gpt-image-*`，例如 `gpt-image-1.5`、`gpt-image-2`，以及 `chatgpt-image-latest`）来说，native / compat 两条链都会放开 `stream`、多图、`mask`、`background=transparent` 与 `3840px` 以内自定义尺寸；如果账号把上游 target model 映射成自定义名字，网关也会按请求里的 display model 归并到这个 profile。未知或旧模型保持保守拒绝路径。
- `/grok/v1/images/*` 与 `/v1beta/openai/images/generations` 仍然保留为显式专用路径；它们更利于排障，也不会被公共智能路由覆盖。
- `image_generation` tool 型主模型不应该直接拿去打 `/v1/images/*`；这类请求应继续走 `/v1/responses`。
- `/antigravity/v1beta/models/{model}:batchGenerateContent` 已注册，但当前能力矩阵明确拒绝。

### 生图专用 Key（Image-only）与图片数量额度

站内创建 / 编辑 API Key 时可以开启 **生图专用**（image-only）。开启后：

- 这个 Key 只能 **拉取** 与 **调用** 图片模型：`GET /v1/models` 会仅返回图片模型；非图片模型请求会被拒绝。
- 允许的主要写入入口：
  - `POST /v1/images/generations`
  - `POST /v1/images/edits`
  - `POST /v1/responses`（仅当顶层 `model` 为原生图片模型，例如 `gpt-image-2` / `chatgpt-image-latest`）
- 典型拒绝：
  - 非图片端点（例如 `/v1/chat/completions`、`/v1/messages`）→ `403`，错误码 `IMAGE_ONLY_KEY_ENDPOINT_NOT_ALLOWED`
  - `/v1/responses` 顶层 `model` 不是原生图片模型 → `403`，错误码 `IMAGE_ONLY_KEY_MODEL_NOT_ALLOWED`

如果同时开启 **按数量计费** 并设置 **最大生图数量**（`> 0`）：

- 网关会在转发上游前做并发安全的额度预占：本次请求期望生成 `n` 张图时，如果 `used + n > max` 直接返回 `429`，错误码 `IMAGE_ONLY_KEY_IMAGE_QUOTA_EXHAUSTED`。
- **只在成功生图时计数**；上游失败与上游 `429` 不计数，并会自动回滚预占额度。
- 失败仍会写入 failed usage log 供排查（不计数但可查询）。

`n` 的解析规则：

- `/v1/images/*`：读取请求体里的 `n`（缺省为 1）
- `/v1/responses`：默认按 1 计；若使用 `tools:[{type:\"image_generation\", n: ...}]` 则使用该 `n`

典型错误体（OpenAI 风格）示例：

```json
{
  "error": {
    "type": "forbidden_error",
    "message": "生图专用 Key 仅允许调用图片模型",
    "code": "IMAGE_ONLY_KEY_MODEL_NOT_ALLOWED"
  }
}
```

```json
{
  "error": {
    "type": "rate_limit_error",
    "message": "图片数量额度已用完",
    "code": "IMAGE_ONLY_KEY_IMAGE_QUOTA_EXHAUSTED"
  }
}
```

### 接入最佳实践

- 先选协议，再选模型，再调优参数。
- 把 `Base URL` 固定为网关根地址，路径由 SDK 或你自己的请求代码拼接。
- 新项目优先使用 `openai-native`、`anthropic`、`gemini` 或 `vertex-batch` 页推荐的主入口，不要继续扩散历史别名路径。
- 调试 `404` 时先确认“当前平台是否支持这个动作”，再排查路径拼写。
- 调试 `429` 时先区分是站内订阅窗口、Key 自身额度，还是上游平台限流。
- 对 Vertex / Batch 简化入口，默认统一使用 `Authorization: Bearer`；只有确实复用 Gemini / Google 原生客户端时，才优先使用 `x-goog-api-key`。
- 如果你要接入百度智能文档，请直接切到 `document-ai` 协议页，优先区分 `async` 与 `direct` 两种模式再选模型。
