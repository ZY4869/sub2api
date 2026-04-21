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
- 用途：返回前台 `/models` 页面使用的“已发布公开模型快照”，包含供应商、请求协议族、发布时冻结的基础出售价格、倍率摘要，以及发布配置中的 `page_size`
- 发布语义：这个接口不再按 TTL 自动重建；只有管理员在计费中心“对外模型展示”页执行“推送更新”后，列表内容、排序、分页大小与详情示例才会一起更新
- 未发布语义：如果当前还没有任何已发布快照，`GET /api/v1/meta/model-catalog` 会显式返回空快照（`items=[]`，默认 `page_size=10`），`GET /api/v1/meta/model-catalog/:model` 会返回 `404`
- 详情语义：`GET /api/v1/meta/model-catalog/:model` 会返回发布时固化的单模型价格块与调用示例元数据，不会在请求时实时重新拼装示例
- 缓存：响应会返回 `ETag`，客户端可通过 `If-None-Match` 复用 `304 Not Modified`

这个接口只暴露展示所需的公共目录数据，不替代具体协议页中的 `/v1/models`、`/v1beta/models` 等运行时模型枚举接口。

公共目录与运行时模型枚举接口共用同一套投影规则，必须记住下面四条：

- 公共目录优先使用账号添加时明确配置的可用模型 ID；只有账号未设置显式限制时，才会回退到保存的探测快照或实时探测结果。
- 账号级白名单 / 取模勾选会直接影响 downstream `/v1/models`、`/v1beta/models` 的返回结果；未被该账号允许的模型不会出现在列表里。
- 如果某个账号把真实模型配置成了自定义映射名，那么 downstream `models list` 与 `models detail` 只返回映射名；`id` / `name` / `displayName` 不再暴露真实模型名，真实模型只保留在内部转发链路。
- 没有有效售价的模型不会出现在公共目录里，也不会出现在用户创建 / 编辑 Key 时的模型选择器里。

`GET /api/v1/meta/model-catalog` 当前返回体额外包含：

- `etag`：本次已发布快照的版本标识
- `updated_at`：最近一次发布完成时间
- `page_size`：公开模型库前台默认每页数量；如果管理员修改草稿但尚未推送，这个值不会变化
- `items`：已发布模型数组，卡片标题应优先展示 `display_name`

典型响应示例：

```json
{
  "etag": "W/\"4b0c0d...\"",
  "updated_at": "2026-04-21T10:05:00Z",
  "page_size": 10,
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

如果前台已经处在明确的分组上下文里，还可以使用倍率后的展示价接口：

- 路径：`GET /api/v1/groups/model-catalog?group_id=<GROUP_ID>`
- 鉴权：必须登录
- 用途：返回“已发布公开模型快照 + 指定分组倍率换算后的 `price_display`”；结构与 `/api/v1/meta/model-catalog` 保持一致，只替换价格字段
- 约束：这里只用于用户自己的分组上下文页面；匿名公共模型库仍然固定展示已发布基础售价

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
- `GET /api/v1/settings/public` 会额外返回 `maintenance_mode_enabled`，前端可据此决定是否展示维护提示。

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
- 维护模式限制：管理员后台、管理员 JWT、管理员用户名下 API Key 调用继续放行；普通用户接口、自助认证流、普通 API Key / 百度智能文档 Key 调用统一返回 `503`。
- 维护模式文案固定为：`维护模式开启中，恢复时间请关注官网公告或官方频道`。
- 普通 JSON 接口会继续使用现有统一错误结构，并附带错误码 `MAINTENANCE_MODE_ACTIVE`；Google / Gemini 风格接口保持 Google 风格错误体，`status` 为 `UNAVAILABLE`。

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
- `/antigravity/v1beta/models/{model}:batchGenerateContent` 已注册，但当前能力矩阵明确拒绝。

### 接入最佳实践

- 先选协议，再选模型，再调优参数。
- 把 `Base URL` 固定为网关根地址，路径由 SDK 或你自己的请求代码拼接。
- 新项目优先使用 `openai-native`、`anthropic`、`gemini` 或 `vertex-batch` 页推荐的主入口，不要继续扩散历史别名路径。
- 调试 `404` 时先确认“当前平台是否支持这个动作”，再排查路径拼写。
- 调试 `429` 时先区分是站内订阅窗口、Key 自身额度，还是上游平台限流。
- 对 Vertex / Batch 简化入口，默认统一使用 `Authorization: Bearer`；只有确实复用 Gemini / Google 原生客户端时，才优先使用 `x-goog-api-key`。
- 如果你要接入百度智能文档，请直接切到 `document-ai` 协议页，优先区分 `async` 与 `direct` 两种模式再选模型。
