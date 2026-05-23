## common
> 本页说明整个网关的统一接入规则。后续左侧协议页会分别展开 OpenAI 原生、OpenAI 兼容、Anthropic / Claude、Gemini 原生、Grok、DeepSeek、Antigravity、Vertex / Batch，以及百度智能文档的细节。

### 概览

Sub2API 是一个多协议聚合网关。你面对的是一套统一站内 API Key，但可以按客户端需要选择不同的协议面来访问上游能力。

建议把文档理解成两层：

- 第一层是“入口协议”：你用什么客户端、发送什么格式、走哪条路径。
- 第二层是“运行时平台”：当前 Key 所绑定的分组最终调度到哪个平台，例如 OpenAI、Anthropic、Gemini、Grok、DeepSeek、Antigravity 或百度智能文档。

协议页固定分成以下 10 个子页：

| 协议页 ID | 页面名称 | 推荐使用者 | 重点内容 |
| --- | --- | --- | --- |
| `common` | 通用接入 | 所有调用方 | 认证、基础地址、错误、限流、模型目录 |
| `openai-native` | OpenAI 原生 | 新版 OpenAI SDK、Responses-first 客户端 | `responses`、子资源、长连接建议 |
| `openai` | OpenAI 兼容 | 旧版 OpenAI SDK、历史兼容客户端 | `chat/completions`、历史别名、兼容迁移 |
| `anthropic` | Anthropic / Claude | Claude SDK、Claude Code、Anthropic 风格客户端 | `messages`、`count_tokens`、保留头 |
| `gemini` | Gemini 原生 | Gemini SDK、AI Studio / Vertex 风格客户端 | `models`、`files`、`batches`、`live`、`openai compat` |
| `grok` | Grok | xAI / Grok 兼容接入 | 聊天、Responses、图像、视频 |
| `deepseek` | DeepSeek | DeepSeek 官方 API Key 调用方 | OpenAI / Anthropic 兼容入口、专用前缀、`chat/completions` 私有 `beta` 开关、公共 `/v1/completions`、不支持能力 |
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
| OpenAI 原生 / OpenAI 兼容 / Anthropic / Grok / DeepSeek / Antigravity | `Authorization: Bearer <API_KEY>` | `x-api-key`、`x-goog-api-key` | 适合大多数 SDK 和代理工具 |
| Gemini / Vertex / Batch 站内推荐入口 | `Authorization: Bearer <API_KEY>` | `x-goog-api-key`、`x-api-key`、部分路径支持 `?key=` | 新接入用户优先；`/v1/vertex/...` 与 `/vertex-batch/jobs...` 默认按这一套接入 |
| 原生 Gemini / Google SDK 兼容入口 | `x-goog-api-key: <API_KEY>` | `Authorization: Bearer`、`x-api-key`、部分路径支持 `?key=` | 当你直接复用 Gemini / Google 风格客户端时更省改造 |

- 虽然后端 Google / Gemini 风格中间件仍优先读取 `x-goog-api-key`，但文档默认建议新接入的 Vertex / Batch 简化入口统一使用 `Authorization: Bearer`。

查询参数的规则必须特别注意：

- `?api_key=...`：整个系统都视为废弃写法。
- `?key=...`：只在 Google / Gemini 风格白名单路径上保留兼容，不适用于 OpenAI 原生、OpenAI 兼容、Anthropic、Grok、DeepSeek、`/v1/vertex/...`、`/vertex-batch/jobs...`、严格 Vertex 路径或 archive 路径。
- 对于 `/v1/vertex/...`、`/vertex-batch/jobs...`、`/v1/projects/:project/locations/:location/...` 和 `/google/batch/archive/...`，请使用请求头，不要依赖 `?key=...`。

当前程序对认证头的优先级如下：

- 普通协议中间件：`Authorization: Bearer` -> `x-api-key` -> `x-goog-api-key` -> 允许时的 `?key=`
- Google / Gemini 风格中间件：`x-goog-api-key` -> `Authorization: Bearer` -> `x-api-key` -> 允许时的 `?key=`

### 请求体压缩（Content-Encoding）

网关支持在 **HTTP 请求体** 使用以下压缩编码（用于 JSON 等非流式请求）：

- `Content-Encoding: gzip`
- `Content-Encoding: deflate`（兼容 zlib-wrapped 与 raw DEFLATE）
- `Content-Encoding: zstd`

注意：

- 只支持单层编码（不支持 `gzip, br` 这种多层组合）；空值与 `identity` 视为未编码。
- 网关会在读取请求体时先解码，再进行协议解析与转发；解码成功后不会再继续保留 `Content-Encoding` 元数据。
- 出于安全考虑，**解码后的 body** 有硬上限（当前为 64MB）；超限会返回 `413 invalid_request_error`。

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
- 没有有效售价的模型不会出现在公共目录里，也不会出现在用户创建 / 编辑 Key 时的模型选择器里。这里的“有效售价”口径不是只看 `sale_form` 原始字段，而是按“sale 优先、缺失字段逐项回退 official”的生效展示价计算；因此只配置了官方价、但 sale 为空的模型，只要存在可展示价格，仍然可以进入公开目录。
- `model[1m]` 只是 Claude Cloud 百万上下文的请求时能力后缀，不会成为新的公开模型资产；`/v1/models`、`/v1beta/models`、公共模型目录、策略投影与模型选择器都不会枚举带 `[1m]` 的镜像项。
- 当站内 Key 开启“生图专用”后，`/v1/models`、`/v1beta/models`、模型详情和 Key 编辑器模型选择器都会在本地投影上继续收敛，只暴露 `image_generation` 原生生图模型；非生图模型即使在原分组可见，也不会成为这个 Key 的可调用 public ID。
- `/v1/models`、`/v1beta/models` 以及复用同一公共模型读路径的详情接口，都是“运行时可服务视图”，不是永久静态目录；如果当前所有可路由账号都因为同一类运行时额度冷却而暂时无法服务某个模型，该模型会临时从列表和详情里隐藏，额度恢复后会自动重新出现。
- 这类运行时隐藏不会反写账号白名单、`model_scope_v2`、probe snapshot 或 manual whitelist；它只是读路径上的临时过滤。
- 对 OpenAI Pro 来说，运行时额度侧是分开的：`gpt-5.3-codex-spark*` 只看 `Spark` 侧，其它 OpenAI 模型统一看 `普通` 侧，所以某一侧冷却时通常只会临时隐藏对应那一侧的模型。

`GET /api/v1/meta/model-catalog` 当前返回体额外包含：

- `etag`：当前有效目录快照的版本标识；命中已发布版本时来自最近一次发布，回退实时目录时来自当前实时快照
- `updated_at`：当前目录快照的更新时间；若命中已发布版本则表示最近一次发布时间，若回退实时则表示实时目录构建时间
- `page_size`：公开模型库前台默认每页数量；命中已发布版本时使用发布快照中的固定值，回退实时时使用当前默认页大小
- `catalog_source`：目录来源，固定为 `published` 或 `live_fallback`
- `items[].status`：面向前台展示的五态摘要，固定为 `ok` / `warning` / `maintenance` / `info` / `error`
- `items[].availability_state`：可服务性来源状态，固定为 `verified` / `unavailable` / `unknown`
- `items[].stale_state`：状态新鲜度，固定为 `fresh` / `stale` / `unverified`
- `items[].lifecycle_status`：生命周期标记，固定为 `stable` / `beta` / `deprecated`
- `items[].price_display.primary`：核心售价行，可能包含 `input_price`、`output_price`、`cache_price`、`batch_cache_price`
- `items[].price_display.secondary`：附加售价行，只保留 grounding / retrieval 等补充项
- `items`：公开模型数组，前台可优先展示 `display_name`；当它与 `model` 只是大小写或分隔符变体时，可按本地标题规则折叠重复 subtitle

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
      "status": "ok",
      "availability_state": "verified",
      "stale_state": "fresh",
      "lifecycle_status": "stable",
      "request_protocols": ["openai"],
      "mode": "chat",
      "currency": "USD",
      "price_display": {
        "primary": [
          { "id": "input_price", "unit": "input_token", "value": 0.0000012 },
          { "id": "output_price", "unit": "output_token", "value": 0.0000024 },
          { "id": "cache_price", "unit": "cache_create_token", "value": 0.0000003 },
          { "id": "batch_cache_price", "unit": "cache_create_token", "value": 0.00000015 }
        ],
        "secondary": [
          { "id": "file_search_retrieval", "unit": "file_search_retrieval_token", "value": 0.000001 }
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
    "provider_icon_key": "openai",
    "status": "ok",
    "availability_state": "verified",
    "stale_state": "fresh",
    "lifecycle_status": "stable",
    "request_protocols": ["openai"],
    "mode": "chat",
    "currency": "USD",
    "price_display": {
      "primary": [
        { "id": "input_price", "unit": "input_token", "value": 0.0000012 },
        { "id": "output_price", "unit": "output_token", "value": 0.0000024 },
        { "id": "cache_price", "unit": "cache_create_token", "value": 0.0000003 }
      ],
      "secondary": [
        { "id": "file_search_retrieval", "unit": "file_search_retrieval_token", "value": 0.000001 }
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
- 生图专用语义：生图专用 Key 下，空 `model_patterns` 表示该分组全部生图模型；提交了文本 / 非生图模型时会被后端归一化或拒绝，确保最终只保留生图模型。
- 回退语义：如果正式公开目录尚未发布，这个接口会和前台 `/models` 一样自动回退到实时目录，但仍会继续过滤掉没有有效售价的模型

如果前台已经处在明确的分组上下文里，还可以使用倍率后的展示价接口：

- 路径：`GET /api/v1/groups/model-catalog?group_id=<GROUP_ID>`
- 鉴权：必须登录
- 用途：返回“当前有效公开模型目录 + 指定分组倍率换算后的 `price_display`”；结构与 `/api/v1/meta/model-catalog` 保持一致，只替换价格字段，`status` / `availability_state` / `stale_state` / `lifecycle_status` 等状态字段保持原样
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
- `GET /api/v1/settings/public` 会额外返回 `maintenance_mode_enabled`、`available_channels_enabled`、`channel_monitor_enabled`、`affiliate_enabled`、`account_airy_white_surface_enabled`，前端可据此决定是否展示维护提示、菜单入口，以及账号管理页清透模式是否切换为纯白主表面。
- 登录 / 注册页还会读取 `login_agreement_enabled`、`login_agreement_mode`、`login_agreement_updated_at`、`login_agreement_documents`。
- 当前 `login_agreement_mode` 固定为 `checkbox`；只有当 `login_agreement_enabled=true` 且 `login_agreement_documents` 至少包含一篇已发布 markdown 页面时，前端才会启用勾选阻断。
- `login_agreement_documents[]` 只返回脱离正文后的文档引用：`{ id, title, page_slug }`；正文仍通过 `GET /api/v1/pages/:slug` 拉取。

### 混合币种计费字段

模型价格与用量统计采用“源币种计费 + USD 兼容字段”的响应契约：

- 公共目录、分组目录和后台价格页里的 `currency` 是源币种；`price_display.primary[].value` 按该源币种展示。价格展示统一使用“sale 优先、缺失字段逐项回退 official”的生效表单；sale 层倍率只作用于 sale 自身已填写字段，不会套到 official 回退字段上。CNY 官方价格保持人民币数值，不会在保存或展示时先折算成 USD。
- CNY 价格现在允许先保存为“待锁汇”状态：保存时可以暂时没有 `usd_to_cny_rate`、`fx_rate_date`、`fx_locked_at`。后台与审计会继续标记 warning；首次实际计费时，系统会尝试拉取当前 USD/CNY 并回写锁定汇率元数据，然后按该锁汇结果继续计费。
- 如果运行时遇到 CNY 价格但暂时无法获得可用汇率，接口会明确返回 `BILLING_FX_RATE_UNAVAILABLE`，而不是静默改扣或错误扣费。
- 管理端计费定价保存接口在参数非法时仍返回 `400`，并保持 `reason=BILLING_PRICE_INVALID`；如存在字段级校验错误，响应还会附带扁平 `metadata.field_errors.<field_id>`，便于前端直接回填对应输入框。
- `usage_logs` 和用量列表会返回 `billing_currency`、`total_cost_usd_equivalent`、`actual_cost_usd_equivalent`、`cost_by_currency`、`actual_cost_by_currency`。其中 `total_cost` / `actual_cost` 保留旧字段语义，对非 USD 记录按 USD 等值给旧客户端使用；分币种金额请读取 `*_by_currency`。
- `/api/v1/usage`、`/api/v1/admin/usage`、`/api/v1/usage/stats`、`/api/v1/admin/usage/stats` 支持 `platform` 查询参数；平台来源按分组平台、账号平台、`unknown` 的顺序归一，不改变既有扣费与配额扣减逻辑。
- `/api/v1/usage/stats`、`/api/v1/admin/usage/stats`、用户 Dashboard 和管理员 Dashboard 会返回 `cost_by_currency`、`actual_cost_by_currency`；Dashboard 还会返回 `today_cost_by_currency`、`today_actual_cost_by_currency`。
- `/api/v1/usage/stats`、`/api/v1/admin/usage/stats` 会额外返回 `platform_breakdown[]`，每项包含 `platform`、`requests`、`input_tokens`、`output_tokens`、`cache_tokens`、`total_tokens`、`cost`、`actual_cost`、`average_duration_ms`，用于前台和后台用量页的平台拆分展示。
- `/api/v1/usage/stats`、`/api/v1/admin/usage/stats` 现在还会额外返回 `today_requests`、`today_input_tokens`、`today_output_tokens`、`today_cache_tokens`、`today_tokens`、`today_cost`、`today_actual_cost`、`today_average_duration_ms`，用于前台和后台“今日统计”卡片。
- 这些 `today_*` 字段按请求里的 `timezone` 计算“今日”窗口：从调用方所在时区当天 `00:00` 到当前时间；如果 `timezone` 缺失或非法，则回退到服务端默认时区。
- `/v1/usage` 在钱包模式下会返回 `balances`，格式为 `{ "USD": 10, "CNY": 25 }`；旧 `balance` / `remaining` 仍只代表 USD 钱包影子余额。
- `/v1/usage` 的 API Key 配额块会返回 `quota.used_by_currency`，限流窗口会返回 `rate_limits[].used_by_currency`；订阅块会返回 `daily_usage_by_currency`、`weekly_usage_by_currency`、`monthly_usage_by_currency`。旧 `quota.used`、`daily_usage_usd`、`weekly_usage_usd`、`monthly_usage_usd` 均继续代表 USD。
- 自动换汇只用于 CNY 钱包不足时的运行时扣费：系统按价格保存时锁定的 `usd_to_cny_rate` 从 USD 钱包换入刚好覆盖缺口的 CNY，并写入 `fx_out`、`fx_in`、`usage_debit` 三类账本记录。

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

### 管理端：账号批量更新（Bulk Update Accounts）

管理员可以通过一个接口批量修改账号字段（例如分组、状态、并发、额外配置等）：

- 路径：`POST /api/v1/admin/accounts/bulk-update`
- 鉴权：管理员 JWT（`Authorization: Bearer <ADMIN_JWT>`）
- 目标选择：`account_ids` 与 `filters` 二选一（必须提供其一，且不能同时提供两者）
- 混合渠道风险：如果目标账号存在混合渠道风险，接口会返回 `409 mixed_channel_warning`；此时可在确认后重试并带上 `confirm_mixed_channel_risk=true` 继续执行

`filters` 的语义与管理端“账号列表”一致，当前至少支持：

- `platform`、`type`、`status`、`group`（数字或 `"ungrouped"`）、`search`
- `lifecycle`、`privacy_mode`
- `limited_view`、`limited_reason`、`runtime_view`

当 `filters` 解析出的目标为空时会返回 `400`。

#### REST
```bash
# 1) 按显式 account_ids 批量更新
curl -X POST "https://api.zyxai.de/api/v1/admin/accounts/bulk-update" \
  -H "Authorization: Bearer <ADMIN_JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "account_ids": [1, 2, 3],
    "status": "inactive",
    "group_ids": [10]
  }'

# 2) 按 filters（筛选结果）批量更新
curl -X POST "https://api.zyxai.de/api/v1/admin/accounts/bulk-update" \
  -H "Authorization: Bearer <ADMIN_JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "filters": {
      "platform": "openai",
      "status": "active",
      "group": "ungrouped",
      "search": "corp"
    },
    "schedulable": false
  }'
```

### 管理端：账号保存与手动探测约束

管理员创建或更新账号时，保存链路现在先执行本地出站安全校验，再决定是否落库：

- 保存时会校验 `credentials.base_url`，不接受空 host、非法端口、`file://`、`gopher://`、默认私网 / `localhost`，以及默认配置下的明文 `http://`
- 百度智能文档账号还会校验 `credentials.async_base_url` 与 `credentials.direct_api_urls`，默认必须命中 `security.url_allowlist.document_ai_hosts`
- 默认策略固定为 `security.url_allowlist.enabled=true`、`allow_private_hosts=false`、`allow_insecure_http=false`
- 普通账号 `base_url` 命中上述限制时，创建 / 更新接口直接返回 `400`，错误码 `ACCOUNT_INVALID_BASE_URL`
- 百度智能文档账号 URL / 凭证校验失败时，创建 / 更新 / 批量更新接口直接返回 `400`，错误码 `ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS`
- 保存成功后不会再自动触发后台模型 probe，也不会在保存动作里隐式刷新 `model_probe_snapshot`
- 如果需要验证连通性或探测模型，请在保存后手动使用现有 `Probe Models`、单账号 `Test` 或批量 `Batch Test` 入口
- 管理端测试、手动 probe 和运行态上游请求都禁止跟随上游 `3xx`；命中重定向时会返回受控错误 `502 UPSTREAM_REDIRECT_NOT_ALLOWED`

建议：

- 直接填写最终的 HTTPS 上游地址，不要依赖 `302/307` 重定向
- 审计 / 内网 mock 场景如需放宽策略，应通过显式配置项调整 allowlist，而不是依赖默认产品配置

典型错误响应：

```json
{
  "code": 400,
  "message": "base_url is invalid or not allowed by the current outbound policy",
  "reason": "ACCOUNT_INVALID_BASE_URL"
}
```

百度智能文档账号 URL / 凭证校验失败时：

```json
{
  "code": 400,
  "message": "Baidu Document AI account credentials are invalid or not allowed by the current outbound policy",
  "reason": "ACCOUNT_INVALID_DOCUMENT_AI_CREDENTIALS"
}
```

### 管理端：每日 07:00 自动触发 5H 设置

管理员现在可以为账号管理页配置一个全局的“每日 07:00 自动触发 5H”任务：

- 路径：`GET /api/v1/admin/accounts/daily-5h-trigger-settings`
- 路径：`PUT /api/v1/admin/accounts/daily-5h-trigger-settings`
- 鉴权：管理员 JWT（`Authorization: Bearer <ADMIN_JWT>`）
- 用途：每天早上 07:00（按服务器本地时区）对被选中的账号类型执行一次后台文本测试，用来主动触发 5H 时限
- 默认文本：所有文本测试统一使用 `Output exactly: OK`
- 运行约束：每天只执行一次；如果服务重启且当天尚未执行，系统会补跑一次
- 候选约束：默认只处理“当前未进入限流窗口”的账号；黑名单账号始终跳过；`include_paused_accounts=true` 时，停用或 `schedulable=false` 但未限流的账号也会纳入
- 固定模型约束：若固定模型已不在该账号自己的白名单可见集合里，则当天跳过该账号，不会降级到别的模型家族
- 可观测性：账号 `extra` 会记录 `daily_5h_trigger_last_local_date`、`daily_5h_trigger_last_status`、`daily_5h_trigger_last_model_id`、`daily_5h_trigger_last_summary`；跳过时会把最近一次 skip reason 摘要写进 `daily_5h_trigger_last_summary`
- 到期自检优先态：`expiry_probe_priority_until` 只影响运行时调度顺序，不会修改数据库中的持久 `priority`；到新 `expires_at` 到期或账号后续探测失败后自动失效

`GET` 返回：

- `settings`
- `candidates[]`
- `candidates[].account_type`
- `candidates[].count`
- `candidates[].models[]`
- `candidates[].models[].model_id`
- `candidates[].models[].display_name`
- `candidates[].models[].provider`
- `candidates[].models[].provider_label`
- `candidates[].models[].account_count`

`PUT` 请求体字段固定为：

- `enabled`
- `selected_account_types[]`
- `include_paused_accounts`
- `openai_model_mode.mode`
- `openai_model_mode.fixed_model_id`
- `anthropic_model_mode.mode`
- `anthropic_model_mode.fixed_model_id`
- `gemini_model_mode.mode`
- `gemini_model_mode.fixed_model_id`

账号类型与自动家族规则固定如下：

- `chatgpt_oauth`：自动选择该账号白名单里最新的 `mini` 家族模型
- `claude_code_oauth_setup_token`：自动选择该账号白名单里最新的 `haiku` 家族模型
- `google_oauth`：自动选择该账号白名单里最新的 `gemini` 家族模型

#### REST
```bash
# 1) 读取当前设置与候选统计
curl "https://api.zyxai.de/api/v1/admin/accounts/daily-5h-trigger-settings" \
  -H "Authorization: Bearer <ADMIN_JWT>"

# 2) 更新设置
curl -X PUT "https://api.zyxai.de/api/v1/admin/accounts/daily-5h-trigger-settings" \
  -H "Authorization: Bearer <ADMIN_JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "enabled": true,
    "selected_account_types": ["chatgpt_oauth", "google_oauth"],
    "include_paused_accounts": false,
    "openai_model_mode": {
      "mode": "auto",
      "fixed_model_id": ""
    },
    "anthropic_model_mode": {
      "mode": "fixed",
      "fixed_model_id": "claude-3.5-haiku"
    },
    "gemini_model_mode": {
      "mode": "auto",
      "fixed_model_id": ""
    }
  }'
```

### 管理端：按筛选结果批量更新用户并发

管理员现在可以直接基于“用户列表当前筛选结果”批量修改用户并发：

- 路径：`POST /api/v1/admin/users/batch-concurrency`
- 鉴权：管理员 JWT（`Authorization: Bearer <ADMIN_JWT>`）
- 幂等：必须携带 `Idempotency-Key`
- 用途：把当前搜索词、角色、状态、分组名和用户属性筛选条件对应到的所有用户，并发上限统一改成同一个值

请求体字段：

- `concurrency`：必填，目标并发值，最小为 `1`
- `search`：可选，按邮箱 / 用户名模糊筛选
- `role`：可选，`admin` 或 `user`
- `status`：可选，`active` 或 `disabled`
- `group_name`：可选，按允许分组名称模糊筛选
- `attributes`：可选，对应用户属性筛选，结构为 `{ "<attribute_id>": "<value>" }`

响应体会返回：

- `matched`：命中的用户数
- `success_count` / `failed_count`：成功 / 失败条数
- `concurrency`：本次写入的目标并发值
- `results[]`：逐个用户的执行结果，包含 `user_id`、`email`、`success` 与可选 `error`

#### REST
```bash
curl -X POST "https://api.zyxai.de/api/v1/admin/users/batch-concurrency" \
  -H "Authorization: Bearer <ADMIN_JWT>" \
  -H "Idempotency-Key: users-batch-concurrency-001" \
  -H "Content-Type: application/json" \
  -d '{
    "concurrency": 8,
    "search": "example.com",
    "status": "active",
    "attributes": {
      "12": "enterprise"
    }
  }'
```

### 自定义 Markdown 页面

站内自定义菜单现在支持 `markdown` 页面模式。前端会优先按 slug 读取 markdown 页面；如果菜单项仍然是旧 `iframe` URL 模式，则继续回退到原有嵌入行为。

- 路径：`GET /api/v1/pages/:slug`
- 鉴权：公开可访问
- 可见性：
  - `visibility=user` 的已发布页面可公开读取
  - `visibility=admin` 的已发布页面只有管理员登录态可读取；未登录或普通用户会收到 `404`
  - 未发布的 markdown 页面不会出现在公开设置返回的菜单里，也无法通过该接口读取

返回体字段：

- `id`：菜单项 ID
- `slug`：规范化后的页面 slug
- `label`：页面标题
- `visibility`：`user` 或 `admin`
- `page_mode`：固定为 `markdown`
- `content`：Markdown 正文

#### REST
```bash
curl "https://api.zyxai.de/api/v1/pages/getting-started"
```

登录 / 注册条款确认复用的也是同一套页面能力：

- 管理员在 `GET|PUT /api/v1/admin/settings` 中写入 `login_agreement_documents[]`，每一项都必须引用一个已发布的 markdown 页面 slug。
- 前端公开入口会把这些文档渲染成 `/legal/:slug` 链接；`/legal/:slug` 只是前端别名路由，最终内容源仍是 `GET /api/v1/pages/:slug`。
- 这意味着系统不会新增独立“法律文档表”或另一套正文存储；条款页与现有 custom markdown page 共用发布、可见性和正文渲染能力。

### GitHub / Google / 钉钉快捷登录

除 LinuxDo 之外，站点还支持 GitHub、Google 与钉钉的快捷登录和登录后绑定：

- `GET /api/v1/auth/oauth/:provider/start`
- `GET /api/v1/auth/oauth/:provider/callback`
- `POST /api/v1/auth/oauth/:provider/complete`

其中 `:provider` 只允许：

- `github`
- `google`
- `dingtalk`

`start` 支持以下查询参数：

- `mode`：可选，`login` 或 `bind`；默认 `login`
- `redirect`：可选，前端相对路径，登录或绑定成功后跳回该页面
- `aff_code`：可选，登录注册场景沿用现有邀请返利码

安全规则：

- `bind` 模式必须带当前用户登录态，且只会绑定到当前账号，不会切换到其他用户
- provider 邮箱已验证且唯一匹配现有用户时，会自动登录并补齐绑定
- provider 邮箱未验证时，不能接管现有邮箱账号
- 邀请码注册开启时，新用户会进入 pending token + `complete` 的补邀请码流程
- 钉钉 OAuth 默认关闭；只有管理员配置并启用后，公开设置才会返回 `dingtalk_oauth_enabled=true`，前端才展示入口。钉钉回调会优先使用 `unionId/openId` 作为第三方身份 ID，凭证和 token 不会出现在公开设置或用户资料接口中。

#### REST
```bash
# 1) 发起 GitHub 登录
curl -I "https://api.zyxai.de/api/v1/auth/oauth/github/start?mode=login&redirect=%2Fdashboard"

# 2) 发起 Google 绑定
curl -I "https://api.zyxai.de/api/v1/auth/oauth/google/start?mode=bind&redirect=%2Fprofile" \
  -H "Authorization: Bearer <USER_JWT>"

# 3) 发起钉钉登录
curl -I "https://api.zyxai.de/api/v1/auth/oauth/dingtalk/start?mode=login&redirect=%2Fdashboard"

# 4) 邀请码补全注册
curl -X POST "https://api.zyxai.de/api/v1/auth/oauth/github/complete" \
  -H "Content-Type: application/json" \
  -d '{
    "pending_oauth_token": "<PENDING_OAUTH_TOKEN>",
    "invitation_code": "INVITE123"
  }'
```

### 当前用户第三方身份

登录后，用户资料页可以查看和解绑已绑定的第三方身份：

- `GET /api/v1/user/auth-identities`
- `DELETE /api/v1/user/auth-identities/:provider`

鉴权要求：

- 两条接口都要求用户 JWT
- `DELETE` 只会移除当前用户自己的绑定记录

返回字段包含：

- `id`
- `provider`
- `provider_user_id`
- `email`
- `email_verified`
- `display_name`
- `avatar_url`
- `created_at`
- `updated_at`

#### REST
```bash
curl "https://api.zyxai.de/api/v1/user/auth-identities" \
  -H "Authorization: Bearer <USER_JWT>"

curl -X DELETE "https://api.zyxai.de/api/v1/user/auth-identities/github" \
  -H "Authorization: Bearer <USER_JWT>"
```

### 管理端：兑换码

管理员可以生成、查询、导出和作废兑换码。兑换码自身有效期与订阅兑换后的有效天数是两个独立概念：

- `expires_at`：兑换码本身的过期时间，可选；为空表示兑换码不过期。
- `validity_days`：仅订阅兑换码使用，表示兑换成功后订阅增加或扣减的天数。
- 旧兑换码没有 `expires_at` 时按未过期处理。
- 列表筛选 `status=expired` 会包含手动作废以及自然过期的未使用兑换码；`status=unused` 不包含自然过期码。

主要接口：

- `GET /api/v1/admin/redeem-codes`
- `POST /api/v1/admin/redeem-codes/generate`
- `POST /api/v1/admin/redeem-codes/create-and-redeem`
- `POST /api/v1/admin/redeem-codes/:id/expire`
- `GET /api/v1/admin/redeem-codes/export`

`expires_at` 接受 RFC3339 时间戳，或 `YYYY-MM-DDTHH:mm`、`YYYY-MM-DD HH:mm:ss`、`YYYY-MM-DD`；传入过去时间会返回统一错误码 `REDEEM_CODE_EXPIRES_AT_INVALID`。兑换已过期的兑换码会返回 `REDEEM_CODE_EXPIRED`。

#### REST
```bash
curl -X POST "https://api.zyxai.de/api/v1/admin/redeem-codes/generate" \
  -H "Authorization: Bearer <ADMIN_JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "count": 5,
    "type": "subscription",
    "value": 10,
    "group_id": 2,
    "validity_days": 30,
    "expires_at": "2026-06-01T08:30:00Z"
  }'
```

### 管理端：内容审核审计

内容审核 v1 默认是“审计而非阻断”能力。它只覆盖文本类请求入口，并且默认 `fail-open`：

- `POST /v1/responses`
- `POST /v1/chat/completions`
- `POST /v1/messages`
- `POST /messages`
- `POST /v1beta/models/{model}:generateContent`
- `POST /v1beta/models/{model}:streamGenerateContent`
- `POST /v1beta/openai/chat/completions`

后台查询接口：

- `GET /api/v1/admin/moderation/audits`
- `GET /api/v1/admin/moderation/audits/:id`

鉴权要求：

- 两条接口都要求管理员 JWT
- 审计记录只保存脱敏摘要、内容哈希、provider/model、命中状态、错误原因、耗时，以及 `request_id` / `client_request_id` / `user_id` / `api_key_id`
- 不会把原始明文内容写入数据库

列表筛选参数：

- `page` / `page_size`
- `request_id`
- `client_request_id`
- `provider`
- `model`
- `source_endpoint`
- `content_hash`
- `user_id`
- `hit=true|false`

#### REST
```bash
curl "https://api.zyxai.de/api/v1/admin/moderation/audits?page=1&page_size=20&provider=openai&hit=false" \
  -H "Authorization: Bearer <ADMIN_JWT>"

curl "https://api.zyxai.de/api/v1/admin/moderation/audits/42" \
  -H "Authorization: Bearer <ADMIN_JWT>"
```

### 管理端：内容审核设置字段

系统设置接口 `GET|PUT /api/v1/admin/settings` 现在包含以下审核字段：

- `content_moderation_enabled`
- `content_moderation_provider`
- `content_moderation_base_url`
- `content_moderation_api_key`（兼容旧单 key 输入；仅 `PUT` 可写）
- `content_moderation_api_keys`（可选多 key 追加/替换输入；仅 `PUT` 可写）
- `content_moderation_api_keys_mode`（`append` 或 `replace`；仅 `PUT` 可写）
- `delete_content_moderation_api_key_hashes`（按 hash 删除已保存 key；仅 `PUT` 可写）
- `content_moderation_api_key_configured`（`GET` 只读布尔值）
- `content_moderation_api_key_statuses[]`（`GET` 只读列表，包含 `hash`、`masked`、可选 `frozen_until`、`last_error`）
- `content_moderation_model`
- `content_moderation_timeout_ms`
- `content_moderation_dedupe_window_seconds`
- `content_moderation_fail_open`

兼容与安全规则：

- 旧客户端继续可以只传 `content_moderation_api_key`；后端会把它归一化进内部 key 列表。
- `append` 会在现有列表后追加并按 hash 去重；`replace` 会用本次提交的新 key 集合替换原列表。
- 删除操作只接受 `delete_content_moderation_api_key_hashes[]`，不会通过 `GET` 返回任何明文 key。
- `content_moderation_api_key_statuses[]` 只用于后台展示 masked key 与冻结状态；错误摘要会脱敏，不应出现 URL、Bearer token 或 key 明文。
- 这些字段全部挂在现有 settings 体系，不会新增独立 settings API。

### 错误响应与限流

错误体会按协议风格返回，而不是统一强行包成一种格式。

### 当前登录用户资料与使用记录展示偏好

除了协议入口外，站内前端还会调用一组登录后资料接口来读取当前用户信息和保存个人偏好：

- `GET /api/v1/auth/me`
- `GET /api/v1/user/profile`
- `PUT /api/v1/user`

这三条接口都要求用户 JWT：

```text
Authorization: Bearer <USER_JWT>
```

其中 `GET /api/v1/auth/me` 与 `GET /api/v1/user/profile` 当前都会返回统一的当前用户资料；前者通常用于登录态恢复，后者更多用于用户资料页。

当前用户资料返回体里新增了：

- `usage_model_display_mode`：使用记录里“模型列”的全局展示偏好
- `global_realtime_countdown_enabled`：非账号页实时倒计时显示偏好
- `account_realtime_countdown_enabled`：账号页实时倒计时显示偏好
- `visual_preset_preference`：当前用户的全局视觉预设偏好
- `account_visual_preset_override`：账号管理页局部视觉预设覆盖

允许值固定为：

- `model_only`：仅显示模型 ID
- `display_only`：仅显示展示名；如果本地目录解析不到展示名，则回退模型 ID
- `display_and_model`：第一行显示展示名，第二行显示模型 ID

视觉预设的允许值固定为：

- `classic`：经典样式
- `airy`：清透样式

两个视觉预设偏好字段的允许值为：

- `inherit`：跟随上一层
- `classic`：强制经典
- `airy`：强制清透

兼容规则如下：

- 新用户与旧用户默认都是 `model_only`
- 如果数据库里读到空值或脏值，后端会统一归一化回 `model_only`
- 这个偏好只影响前台 / 后台“使用记录类表格”的显示层，不影响模型路由、可见模型集合、计费、权限或限流
- 站点默认视觉预设固定为 `visual_preset_default`，默认值是 `classic`
- `account_airy_white_surface_enabled` 默认是 `false`，它只影响管理员账号管理页在 `airy` 视觉预设下的表格、卡片和分组容器底色，不影响 `classic` 或其他页面
- `visual_preset_preference` 与 `account_visual_preset_override` 对新用户默认都是 `inherit`
- 最终有效视觉预设按 `site default -> user preference -> account override` 解析
- 这组偏好只影响站内视觉呈现，不改变账号数据、调度、配额、限流或自动刷新逻辑本体

两个实时倒计时偏好的固定语义如下：

- `global_realtime_countdown_enabled`
  - 默认 `false`
  - 只控制非账号页的实时倒计时显示，例如 Ops 自动刷新剩余秒数
- `account_realtime_countdown_enabled`
  - 默认 `true`
  - 只控制账号页内部的实时倒计时显示，例如限流恢复、窗口重置、分段数字倒计时与账号页自动刷新剩余秒数

这两个偏好都是“当前登录账号自己的显示偏好”，互不覆盖，也不会改变真实自动刷新调度、限流判断、配额计算或接口返回数据。

`PUT /api/v1/user` 当前支持的请求体字段包括：

- `username`：可选，更新用户名
- `usage_model_display_mode`：可选，更新模型列展示偏好
- `global_realtime_countdown_enabled`：可选，更新非账号页实时倒计时显示偏好
- `account_realtime_countdown_enabled`：可选，更新账号页实时倒计时显示偏好
- `visual_preset_preference`：可选，更新当前用户全局视觉预设偏好
- `account_visual_preset_override`：可选，更新账号页局部视觉预设覆盖

更新接口遵循“只改传入字段”的局部更新语义；未传字段保持不变。`usage_model_display_mode`、`visual_preset_preference` 或 `account_visual_preset_override` 如果传入非法值会直接返回 `400`；两个实时倒计时字段如果传入，会按布尔值原样更新当前账号偏好。

同时，用户 / 管理员使用记录列表与管理员请求详情台账现在会返回：

- `request_context_length_tokens`：本次请求采用的上下文档位快照，单位为 token 整数，例如 `128000`、`200000`、`1000000`

这个字段的语义固定为：

- 它表示“本次请求按哪个上下文档位发起”，不是实际输入 token
- 它也不是 `max_tokens` / `max_output_tokens` 这类输出上限
- 如果请求显式表达 1M 上下文并被解析到（例如 Claude / DeepSeek 风格的 `[1m]`），这里会记录 `1000000`
- 否则会记录请求时最终模型对应的默认上下文档位快照
- 历史记录允许为空；为空时前端通常展示为 `-`

#### REST
```bash
# 读取当前登录用户资料
curl https://api.zyxai.de/api/v1/auth/me \
  -H "Authorization: Bearer <USER_JWT>"

# 或者读取用户资料页使用的同类接口
curl https://api.zyxai.de/api/v1/user/profile \
  -H "Authorization: Bearer <USER_JWT>"

# 更新使用记录里的模型展示模式
curl -X PUT https://api.zyxai.de/api/v1/user \
  -H "Authorization: Bearer <USER_JWT>" \
  -H "Content-Type: application/json" \
  -d '{
    "usage_model_display_mode": "display_and_model",
    "global_realtime_countdown_enabled": true,
    "account_realtime_countdown_enabled": false,
    "visual_preset_preference": "airy",
    "account_visual_preset_override": "classic"
  }'
```

典型成功响应示例：

```json
{
  "code": 0,
  "message": "success",
  "data": {
    "id": 1,
    "email": "alice@example.com",
    "username": "alice",
    "role": "user",
    "usage_model_display_mode": "display_and_model",
    "global_realtime_countdown_enabled": true,
    "account_realtime_countdown_enabled": false,
    "visual_preset_preference": "airy",
    "account_visual_preset_override": "classic",
    "balance": 12.5,
    "concurrency": 5,
    "status": "active",
    "allowed_groups": null,
    "created_at": "2025-01-02T03:04:05Z",
    "updated_at": "2025-01-02T03:04:05Z"
  }
}
```

典型非法枚举响应示例：

```json
{
  "code": 400,
  "message": "usage_model_display_mode must be one of model_only, display_only, display_and_model",
  "reason": "USER_USAGE_MODEL_DISPLAY_MODE_INVALID"
}
```

```json
{
  "code": 400,
  "message": "visual_preset_preference must be one of inherit, classic, airy",
  "reason": "VISUAL_PRESET_PREFERENCE_INVALID"
}
```

```json
{
  "code": 400,
  "message": "account_visual_preset_override must be one of inherit, classic, airy",
  "reason": "VISUAL_PRESET_PREFERENCE_INVALID"
}
```

OpenAI / Anthropic 风格常见于：

- `/v1/responses`
- `/v1/chat/completions`
- `/v1/messages`
- `/grok/v1/...`
- `/deepseek/v1/...`
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
| DeepSeek 文本 / Claude 风格消息 / FIM completion | `/deepseek/v1/...` + `/v1/completions` | `models`、`messages` 走 `/deepseek/v1/...`；`chat/completions` 走同一路由并支持私有 `beta:true|false`；FIM / Beta completion 只走公共 `/v1/completions` |
| 显式只走 Antigravity | `/antigravity/...` | 禁止混合调度 |
| Vertex / Batch / Archive | `/v1/vertex/...`、`/vertex-batch/jobs...` | 新接入优先；严格 `/v1/projects/...` 只留给 SDK 兼容，结果归档继续走 `/google/batch/archive/...` |
| 百度智能文档解析 | `/document-ai/v1/...` | 优先区分 `async` 与 `direct` |

跨协议兼容也存在，但不是无条件开放：

- `/v1/messages` 在 OpenAI 平台下可能被翻译到 Responses。
- `/v1/messages` 在 DeepSeek 平台下会走 DeepSeek 官方 Anthropic 兼容入口，但会预检并拒绝 DeepSeek 官方未支持的多模态、搜索、工具和容器类内容块。
- `/v1/messages/count_tokens` 只应当期望在 Anthropic 原生平台成功；DeepSeek 明确不支持。
- DeepSeek `/deepseek/v1/chat/completions` 额外支持顶层私有字段 `beta?: boolean`：`true` 强制走 beta，`false` 强制稳定面；未传时才按 `prefix` / `reasoning_content` 自动识别 beta。
- `/v1/completions` 是 DeepSeek 专用公共 FIM / Beta completion 入口；它不会提供 `/deepseek/v1/completions` 或 `/completions` 别名。
- `/v1/responses`、Images、Videos 不属于 DeepSeek 接入面；文本类请求请改用 `/deepseek/v1/chat/completions`、`/deepseek/v1/messages` 或公共 `/v1/completions`。
- `/user/balance` 不会对 DeepSeek 分组开放，避免把上游 Key 余额暴露给下游调用方。
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

- 网关会在转发上游前做并发安全的额度预占：本次请求期望生成 `n` 张图时，会按 `n × 分辨率计数权重` 预占；如果 `used + units > max` 直接返回 `429`，错误码 `IMAGE_ONLY_KEY_IMAGE_QUOTA_EXHAUSTED`。
- 每个 Key 可以设置 `image_count_weights`，默认 `{"1K":1,"2K":1,"4K":2}`；例如 `4K=2` 表示生成 1 张 4K 图片计 2 个单位。未识别、空值或 `auto` 尺寸会归入 `2K` 档。
- **只在成功生图时计数**；上游失败与上游 `429` 不计数，并会自动回滚预占额度。
- 成功后会按实际上游返回图片张数结算差额；如果实际张数少于预占张数，多预占的计数单位会回滚。
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

### 站内支付与订阅购买

站内支付接口用于用户余额充值与订阅购买。它与模型网关 API Key 是两套认证面：用户支付接口使用登录后的 JWT，管理员支付接口需要管理员权限，Airwallex webhook 不接受 JWT，只接受 webhook 签名校验。

功能默认关闭。公开设置响应中的 `payment_provider_airwallex_enabled` 是展示用有效态：只有管理员同时开启 `purchase_subscription_enabled`、`payment_provider_airwallex_enabled`，并满足 Airwallex Client ID/API Key configured、币种、默认币种和充值上下限校验后才会返回 `true`；旧的 `purchase_subscription_url` 仍保留为外部购买链接兜底。公开设置只返回这个布尔态与非密钥商品配置，不返回 Airwallex secret、client secret 或 webhook secret。管理员设置接口仍返回原始 `payment_provider_airwallex_enabled` 开关和 `payment_provider_airwallex_effective` 有效态，便于排障。

关键设置字段：

- `payment_provider_airwallex_enabled`：是否启用 Airwallex 站内支付。
- `airwallex_env`：`demo` 或 `prod`。
- `airwallex_client_id`、`airwallex_api_key`、`airwallex_webhook_secret`：仅管理员可写。公开设置只使用 Client ID/API Key 的 configured 状态计算支付展示开关，不返回这些字段，也不返回 webhook secret。
- `payment_allowed_currencies`：允许币种白名单，默认 `USD/CNY/HKD`。
- `payment_default_currency`：用户购买页默认币种。
- `payment_min_topup_amount`、`payment_max_topup_amount`：余额充值上下限。
- `payment_subscription_plans`：订阅计划数组，包含 `plan_id`、`name`、`group_id`、`validity_days`、`prices_by_currency`、`enabled`。
- `antigravity_user_agent_version`：可选 Antigravity User-Agent 版本，留空使用系统默认；显式值需符合 `major.minor.patch[-suffix]`。

用户接口：

- `POST /api/v1/payment/orders`：创建订单。需要 JWT，建议带 `Idempotency-Key`；同一用户同一请求体与同一 `Idempotency-Key` 重放会返回同一订单，且不会创建新的 Airwallex PaymentIntent。请求可传 `return_url`，支持 `__ORDER_NO__` 占位；未传时默认生成站内 `/payment/result/:orderNo` 结果页地址。
- `GET /api/v1/payment/orders/:order_no`：查询当前用户自己的订单。
- `GET /api/v1/payment/orders/:order_no/resume`：按订单号恢复当前用户自己的未完成订单，返回 Airwallex 前端初始化字段。结果页应优先使用该接口恢复 `created` / `pending` 订单，避免依赖一次性可见的恢复令牌。
- `GET /api/v1/payment/resume/:resume_token`：用恢复令牌恢复未完成订单，返回支付组件初始化所需的非密钥字段。仅 `created` / `pending` 状态应展示支付组件。
- `POST /api/v1/payment/orders/:order_no/cancel`：取消当前用户自己的未支付订单。

创建与恢复订单响应都会包含：

- `client_id`：Airwallex 前端 SDK 初始化使用的 Client ID。
- `provider_env`：`demo` 或 `prod`，对应 Airwallex 前端 SDK `init({ env })`。
- `intent_id`：Airwallex PaymentIntent ID。
- `client_secret`：仅返回给当前订单用户，用于 Airwallex `confirm({ client_secret, intent_id })`；禁止写入日志、公开设置或持久化明文字段。
- `order`：订单快照，包含 `order_no`、`status`、`amount_minor`、`currency`、`refunded_amount_minor`、`refundable_amount_minor` 等字段。

管理员接口：

- `GET /api/v1/admin/payment/orders`：分页查询订单，支持 `status`、`provider`、`product_type`、`user_id`。
- `POST /api/v1/admin/payment/orders/:order_no/refund`：发起全额或部分退款。请求体可传 `amount_minor` 与 `reason`，建议带 `Idempotency-Key`；重复提交会按本地幂等记录 replay 或返回冲突。
- `GET /api/v1/admin/ops/runtime/payment`：返回内存运行时支付指标快照，包含订单创建成功/失败、provider 耗时计数与总耗时、webhook 成功/失败、恢复成功/失败、退款成功/失败。

公开 webhook：

- `POST /api/v1/payment/webhooks/airwallex`：Airwallex 回调入口，不走 JWT。服务端会按 Airwallex webhook secret 验签，并按 provider event id 幂等处理。
- 验签输入固定为 `x-timestamp + raw JSON body`，使用 HMAC-SHA256，与 `x-signature` 比对；时间戳允许 5 分钟窗口，支持秒或毫秒时间戳。
- webhook payload 入库时只保存 hash 与脱敏 JSON；`secret`、`token`、`api_key`、`authorization`、`client_secret`、邮箱、电话、姓名、地址等字段会被替换为 `[REDACTED]`。
- 重复的 provider event id 会直接返回成功，避免 Airwallex 重试造成重复发放。

数据层说明：

- 支付表 `payment_orders`、`payment_events`、`payment_refunds` 当前由 raw SQL 仓储维护，不生成 ent entity。这是支付仓储的刻意边界：订单、webhook 事件与退款幂等写入集中在 `PaymentRepository`，权益发放继续复用既有钱包与订阅表。
- 前端 Airwallex 嵌入式组件依赖 `@airwallex/components-sdk@1.32.0`，已按 npm 包页面核验为 MIT；仓库根 `LICENSE` 保持 MIT。

退款说明：

- 服务端调用 Airwallex 官方退款接口 `POST /api/v1/pa/refunds/create`，请求体使用 `payment_intent_id`，必要时可携带 `payment_attempt_id`。
- `amount_minor` 为空或小于等于 0 时按全额退款处理；部分退款使用订单币种的最小货币单位。
- 订单退款状态按 `payment_refunds` 中 `accepted` / `settled` 累计成功退款金额判断；累计小于订单金额为 `partial_refunded`，达到订单金额为 `refunded`。服务端和管理员前端都会拒绝超过剩余可退金额的退款请求。
- provider 失败会映射为统一 `PAYMENT_PROVIDER_FAILED`，响应与日志都不会包含 Airwallex 原始错误体中的密钥、client secret 或个人信息。

错误响应示例：

```json
{
  "code": 400,
  "message": "unsupported payment currency",
  "reason": "PAYMENT_UNSUPPORTED_CURRENCY",
  "metadata": {
    "currency": "EUR"
  }
}
```

订单状态固定为：

```text
created | pending | paid | failed | cancelled | expired | partial_refunded | refunded
```

#### Python
```python focus=5-16,19-24
import requests

base_url = "https://api.zyxai.de/api/v1"
jwt = "用户登录后获得的 JWT"

resp = requests.post(
    f"{base_url}/payment/orders",
    headers={
        "Authorization": f"Bearer {jwt}",
        "Content-Type": "application/json",
        "Idempotency-Key": "topup-20260522-001",
    },
    json={
        "product_type": "balance_topup",
        "amount": 10,
        "currency": "USD",
        "country_code": "US",
        "return_url": "https://app.example.com/payment/result/__ORDER_NO__",
    },
    timeout=30,
)
order = resp.json()["data"]
print(order["order"]["order_no"], order["client_id"], order["provider_env"])
print("client_secret is returned only to the current order user")
```

#### JavaScript
```javascript focus=5-17
const baseUrl = "https://api.zyxai.de/api/v1";
const jwt = "用户登录后获得的 JWT";

const response = await fetch(`${baseUrl}/payment/orders`, {
  method: "POST",
  headers: {
    Authorization: `Bearer ${jwt}`,
    "Content-Type": "application/json",
    "Idempotency-Key": "subscription-20260522-001",
  },
  body: JSON.stringify({
    product_type: "subscription",
    plan_id: "pro-monthly",
    currency: "USD",
    country_code: "US",
    return_url: `${window.location.origin}/payment/result/__ORDER_NO__`,
  }),
});

const data = await response.json();
console.log(data.data.client_id, data.data.intent_id, data.data.provider_env);
```

#### REST
```bash focus=1-12
curl https://api.zyxai.de/api/v1/payment/orders \
  -H "Authorization: Bearer 用户登录后获得的JWT" \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: topup-20260522-001" \
  -d '{
    "product_type": "balance_topup",
    "amount": 10,
    "currency": "USD",
    "country_code": "US",
    "return_url": "https://app.example.com/payment/result/__ORDER_NO__"
  }'
```

### 接入最佳实践

- 先选协议，再选模型，再调优参数。
- 把 `Base URL` 固定为网关根地址，路径由 SDK 或你自己的请求代码拼接。
- 新项目优先使用 `openai-native`、`anthropic`、`gemini` 或 `vertex-batch` 页推荐的主入口，不要继续扩散历史别名路径。
- 如果你要显式绑定 DeepSeek，请使用 `deepseek` 页的 `/deepseek/v1/...` 前缀；如果需要 chat beta，可在 `chat/completions` 请求体顶层传 `beta:true|false`。只有 FIM / Beta completion 例外，它固定走公共 `/v1/completions`。默认模型使用 `deepseek-v4-flash` 或 `deepseek-v4-pro`。
- 调试 `404` 时先确认“当前平台是否支持这个动作”，再排查路径拼写。
- 调试 `429` 时先区分是站内订阅窗口、Key 自身额度，还是上游平台限流。
- 对 Vertex / Batch 简化入口，默认统一使用 `Authorization: Bearer`；只有确实复用 Gemini / Google 原生客户端时，才优先使用 `x-goog-api-key`。
- 如果你要接入百度智能文档，请直接切到 `document-ai` 协议页，优先区分 `async` 与 `direct` 两种模式再选模型。
