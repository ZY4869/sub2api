# API 文档中心

## common
> 本页说明整个网关的统一接入规则。后续左侧协议页会分别展开 OpenAI 兼容、Anthropic / Claude、Gemini 原生、Grok、Antigravity、Vertex / Batch 的细节。

### 概览

Sub2API 是一个多协议聚合网关。你面对的是一套统一站内 API Key，但可以按客户端需要选择不同的协议面来访问上游能力。

建议把文档理解成两层：

- 第一层是“入口协议”：你用什么客户端、发送什么格式、走哪条路径。
- 第二层是“运行时平台”：当前 Key 所绑定的分组最终调度到哪个平台，例如 OpenAI、Anthropic、Gemini、Grok、Antigravity。

协议页固定分成以下 7 个子页：

| 协议页 ID | 页面名称 | 推荐使用者 | 重点内容 |
| --- | --- | --- | --- |
| `common` | 通用接入 | 所有调用方 | 认证、基础地址、错误、限流、同步规则 |
| `openai` | OpenAI 兼容 | OpenAI SDK、Lobe、各种 OpenAI 兼容客户端 | `responses`、`chat/completions`、图像与视频 |
| `anthropic` | Anthropic / Claude | Claude SDK、Claude Code、Anthropic 风格客户端 | `messages`、`count_tokens`、保留头 |
| `gemini` | Gemini 原生 | Gemini SDK、AI Studio / Vertex 风格客户端 | `models`、`files`、`batches`、`live`、`openai compat` |
| `grok` | Grok | xAI / Grok 兼容接入 | 聊天、Responses、图像、视频 |
| `antigravity` | Antigravity | 需要显式绑定 Antigravity 平台的接入方 | Anthropic 风格入口 + Gemini 风格入口 |
| `vertex-batch` | Vertex / Batch | Vertex Batch Prediction、Google Batch Archive | 批任务、归档查询、文件下载 |

### 快速接入

接入时建议按下面顺序做，不要一开始就混用多种协议路径：

1. 在站内创建一个可用的 API Key。
2. 先确定你要模拟的协议，而不是先决定模型名。
3. 把客户端 `Base URL` 指向api.zyxai.de。
4. 选择对应协议推荐的认证头。
5. 先跑通一个最短请求，再扩展流式、上传、批任务和工具调用。

下面给出一个最短可联通的 smoke test，统一走 `OpenAI Responses`，因为它是当前最稳妥的公共文本入口之一。

#### Python
```python
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
```javascript
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
```bash
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
| OpenAI / Anthropic / Grok / Antigravity | `Authorization: Bearer <API_KEY>` | `x-api-key`、`x-goog-api-key` | 适合大多数 SDK 和代理工具 |
| Gemini / Google 风格入口 | `x-goog-api-key: <API_KEY>` | `Authorization: Bearer`、`x-api-key`、部分路径支持 `?key=` | 原生 Gemini 客户端优先使用 Google 头 |

查询参数的规则必须特别注意：

- `?api_key=...`：整个系统都视为废弃写法。
- `?key=...`：只在 Google / Gemini 风格白名单路径上保留兼容，不适用于 OpenAI、Anthropic、Grok、Vertex Batch。
- 对于 `/v1/projects/:project/locations/:location/...` 和 `/google/batch/archive/...`，请使用请求头，不要依赖 `?key=...`。

当前程序对认证头的优先级如下：

- 普通协议中间件：`Authorization: Bearer` -> `x-api-key` -> `x-goog-api-key` -> 允许时的 `?key=`
- Google / Gemini 风格中间件：`x-goog-api-key` -> `Authorization: Bearer` -> `x-api-key` -> 允许时的 `?key=`

下面的例子分别展示三种常用认证写法。

#### Python
```python
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"

requests.get(
    f"{base_url}/v1/models",
    headers={"x-goog-api-key": api_key},
    timeout=30,
)
```

#### JavaScript
```javascript
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
```bash
curl "https://api.zyxai.de/v1beta/models?key=sk-你的站内Key" \
  -H "Content-Type: application/json"
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
| `500` | 内部错误 | 网关内部异常或上游转发失败 |

需要特别关注的两类限制：

- API Key 自身限制：已过期、额度耗尽、IP 白名单 / 黑名单、用户状态异常。
- 分组 / 订阅限制：按日、按周、按月窗口限制，或订阅不存在、余额不足。

下面的例子展示如何在三种环境中读取错误体，而不是只看 HTTP 状态码。

#### Python
```python
import requests

response = requests.get("https://api.zyxai.de/v1beta/corpora")
payload = response.json()

if response.status_code >= 400:
    print("status:", response.status_code)
    print("error:", payload.get("error"))
```

#### JavaScript
```javascript
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
```bash
curl https://api.zyxai.de/v1beta/test?api_key=legacy
```

### 模型与路径兼容差异

不要把“模型名可用”误解成“所有协议入口都可用”。协议兼容是按“入口路径 + 动作 + 当前运行平台”共同决定的。

下面这张表是最实用的判断方式：

| 你要做的事 | 优先入口 | 说明 |
| --- | --- | --- |
| 通用文本生成 | `/v1/responses` | 新项目优先 |
| 兼容旧 OpenAI 客户端 | `/v1/chat/completions` | 旧生态广泛支持 |
| Claude 风格接入 | `/v1/messages` | 原生 Anthropic / Claude 最稳 |
| Gemini 原生生成 | `/v1beta/models/{model}:generateContent` | 原生 Google 风格 |
| Gemini 文件 / Batch / Live | `/v1beta/files`、`/v1beta/batches`、`/v1beta/live` | 走 Gemini 原生页 |
| Grok 图像 / 视频 | `/grok/v1/...` | 更明确，排错更容易 |
| 显式只走 Antigravity | `/antigravity/...` | 禁止混合调度 |
| Vertex Batch / Archive | `/v1/projects/...`、`/google/batch/archive/...` | Google 风格，但不是普通 `v1beta` 资源族 |

跨协议兼容也存在，但不是无条件开放：

- `/v1/messages` 在 OpenAI / Copilot 平台下可能被翻译到 Responses。
- `/v1/messages/count_tokens` 只应当期望在 Anthropic 原生平台成功。
- `/v1/responses` 在 Grok 平台可以工作，但 Responses 的 WebSocket / 长连接模式不应对 Grok 做乐观假设。
- `/antigravity/v1beta/models/{model}:batchGenerateContent` 已注册，但当前能力矩阵明确拒绝。

### 接入最佳实践

- 先选协议，再选模型，再调优参数。
- 把 `Base URL` 固定为网关根地址，路径由 SDK 或你自己的请求代码拼接。
- 新项目优先使用 `/v1/responses`、`/v1/messages` 或 Gemini 原生入口，不要继续扩散历史别名路径。
- 调试 `404` 时先确认“当前平台是否支持这个动作”，再排查路径拼写。
- 调试 `429` 时先区分是站内订阅窗口、Key 自身额度，还是上游平台限流。
- 只有确实使用 Gemini / Google 原生客户端时，才优先使用 `x-goog-api-key`。
- 运行时覆盖文档可以临时修正说明，但仓库基线才是需要随代码一起维护的版本事实。

### 文档同步说明

仓库内唯一基线文档位于：

```text
backend/internal/service/docs/api_reference.md
```

同步规则如下：

- 请求路径、别名路径、动作能力变化时，必须同步修改对应章节。
- 认证方式、认证优先级、查询参数兼容面变化时，必须同步修改本页和 Gemini / Vertex 相关页。
- 错误结构、状态码、限流语义、示例请求变化时，必须同步更新文档示例。
- 管理员页面保存的运行时覆盖不会回写 Git 文件，不能替代仓库基线维护。

## openai
> 本页面向 OpenAI 兼容客户端。重点是 `responses`、`chat/completions` 以及仅在 Grok 运行平台生效的图像 / 视频入口。

### 协议定位与适用客户端

当你使用以下客户端时，优先阅读本页：

- OpenAI Python / JavaScript SDK
- 各类 OpenAI 兼容代理、插件、聊天前端
- 仍然依赖 `chat/completions` 的历史工程

当前程序对 OpenAI 兼容入口的理解是：

- `responses` 是主入口，适合新项目。
- `chat/completions` 是兼容入口，适合旧项目。
- 图像和视频路径是 OpenAI 风格外观，但只有在运行平台为 Grok 时才真正可用。

### 推荐入口与别名路径

主要路径如下：

| 动作 | 推荐路径 | 兼容别名 | 说明 |
| --- | --- | --- | --- |
| 创建 Responses | `POST /v1/responses` | `POST /responses` | 推荐新项目使用 |
| 查询 / 删除 Responses 子资源 | `GET/DELETE /v1/responses/*subpath` | `GET/DELETE /responses/*subpath` | 保留 OpenAI 风格子资源访问 |
| Responses 长连接 / WebSocket 风格入口 | `GET /v1/responses` | `GET /responses` | 仅对支持的平台有意义 |
| Chat Completions | `POST /v1/chat/completions` | `POST /chat/completions` | 兼容旧生态 |
| 图像生成 / 编辑 | `POST /v1/images/generations`、`POST /v1/images/edits` | `/images/...` | 运行平台必须是 Grok |
| 视频创建 / 状态查询 | `POST /v1/videos`、`GET /v1/videos/:request_id` | `/videos...` | 运行平台必须是 Grok |

如果你在配置官方 SDK，最简单的方式是把 `Base URL` 指到 `/v1` 前缀，而不是使用历史别名。

#### Python
```python
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
        "input": "请总结 OpenAI 兼容入口的推荐用法。",
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
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
    input: "请总结 OpenAI 兼容入口的推荐用法。",
  }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1/responses \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "input": "请总结 OpenAI 兼容入口的推荐用法。"
  }'
```

### Responses API 详细规则

`responses` 是当前最推荐的 OpenAI 文本主入口，规则要点如下：

- `POST /v1/responses` 创建响应。
- `POST /v1/responses/*subpath` 用于保留 Responses 子资源动作。
- `GET /v1/responses/*subpath` 与 `DELETE /v1/responses/*subpath` 用于查询或删除子资源。
- `GET /v1/responses` 由专门的 Responses WebSocket / 长连接处理链路接管。
- 当运行平台为 OpenAI 或 Copilot 时，Responses 能力是原生直通。
- 当运行平台为 Grok 时，普通 `POST /grok/v1/responses` 可用，但 WebSocket 动作在能力矩阵中被拒绝。

排错建议：

- 普通文本生成失败时，先确认你用的是 `POST` 而不是 `GET`。
- 看到 `404` 时，不要只怀疑路径拼写，还要看当前分组平台是否真的支持该动作。
- 如果你依赖持续连接或多轮状态链路，请优先在 OpenAI / Copilot 平台验证。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/v1/responses",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "model": "gpt-5.4",
        "input": [
            {
                "role": "user",
                "content": [{"type": "input_text", "text": "列出 Responses 与 Chat Completions 的区别。"}],
            }
        ],
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/v1/responses", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "gpt-5.4",
    input: "列出 Responses 与 Chat Completions 的区别。",
  }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1/responses/resp_123 \
  -H "Authorization: Bearer sk-你的站内Key"
```

### Chat Completions 兼容规则

`chat/completions` 的定位是“兼容旧生态，而不是主推新设计”：

- `POST /v1/chat/completions` 对 OpenAI / Copilot / Grok 平台可直通。
- 对于仍然使用 `messages` 数组的旧应用，这是最省心的入口。
- 如果是全新项目，仍建议优先改用 `responses`。

选择 `chat/completions` 的典型场景：

- 现有代码或第三方工具不支持 `responses`
- 你明确依赖旧版 OpenAI SDK 或旧参数结构
- 你在做快速兼容，而不是长期演进

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/v1/chat/completions",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "model": "gpt-4.1",
        "messages": [
            {"role": "system", "content": "你是一个简洁的接口说明助手。"},
            {"role": "user", "content": "解释什么时候还应该使用 chat/completions。"},
        ],
        "stream": False,
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/v1/chat/completions", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "gpt-4.1",
    messages: [{ role: "user", content: "解释什么时候还应该使用 chat/completions。" }],
  }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1/chat/completions \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4.1",
    "messages": [
      { "role": "user", "content": "解释什么时候还应该使用 chat/completions。" }
    ]
  }'
```

### 图像与视频入口

本项目保留了 OpenAI 风格的媒体路径，但要牢记一个核心事实：

- 这些路径只在运行平台是 Grok 时才可用。
- 路径注册成功不等于所有分组都能调用成功。
- 如果当前分组并非 Grok，网关会返回兼容错误，而不是静默帮你切换平台。

媒体相关路径：

| 动作 | 路径 |
| --- | --- |
| 图像生成 | `POST /v1/images/generations` |
| 图像编辑 | `POST /v1/images/edits` |
| 视频创建 | `POST /v1/videos` |
| 视频创建别名 | `POST /v1/videos/generations` |
| 视频状态查询 | `GET /v1/videos/:request_id` |

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/v1/images/generations",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "model": "grok-2-image",
        "prompt": "一张用于 API 文档首页的极简科技插画",
        "size": "1024x1024",
    },
    timeout=120,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/v1/videos", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "grok-video",
    prompt: "一段 5 秒钟的云端数据流动画",
  }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1/videos/req_123 \
  -H "Authorization: Bearer sk-你的站内Key"
```

### 流式与常见坑

- `responses` 相关的长连接 / WebSocket 模式不要直接套用到 Grok 平台。
- 新项目不要再从 `/responses`、`/chat/completions` 这些无 `/v1` 别名开始写。
- 如果你需要同时支持文本和媒体，请先确认分组平台是否真的是 Grok，而不是只看模型名里像不像。
- 要做长期维护的系统，应该优先围绕 `responses` 设计，而不是继续叠加 `chat/completions` 特性债务。

## anthropic
> 本页面向 Claude / Anthropic 风格客户端。重点是 `messages`、`count_tokens`、请求头透传，以及在非 Anthropic 平台下的兼容与拒绝规则。

### 协议定位与适用客户端

本页适用于：

- Anthropic 官方 SDK
- Claude Code 或任何依赖 `messages` 协议的工具
- 希望保留 `anthropic-version`、`anthropic-beta` 头的调用方

当前程序对 `messages` 的处理不是“死板只认 Anthropic 平台”：

- 运行平台是 Anthropic 时，`/v1/messages` 原生直通。
- 运行平台是 OpenAI / Copilot 时，`/v1/messages` 可能被翻译到 Responses。
- 运行平台是 Antigravity 时，`/antigravity/v1/messages` 可以走原生 Antigravity 路径。
- 运行平台是 Grok 时，`messages` 在能力矩阵中被明确拒绝。

### 认证方式与保留请求头

推荐认证方式：

- 首选：`Authorization: Bearer <API_KEY>`
- 兼容：`x-api-key: <API_KEY>`

建议保留的头：

- `anthropic-version`
- `anthropic-beta`

这些头不会被要求改成自定义字段，网关会尽量按原语义透传。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/v1/messages",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
        "anthropic-version": "2023-06-01",
        "anthropic-beta": "output-128k-2025-02-19",
    },
    json={
        "model": "claude-sonnet-4-20250514",
        "max_tokens": 256,
        "messages": [{"role": "user", "content": "解释 Claude 风格入口的保留头规则。"}],
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/v1/messages", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
    "anthropic-version": "2023-06-01",
  },
  body: JSON.stringify({
    model: "claude-sonnet-4-20250514",
    max_tokens: 256,
    messages: [{ role: "user", content: "解释 Claude 风格入口的保留头规则。" }],
  }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1/messages \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 256,
    "messages": [
      { "role": "user", "content": "解释 Claude 风格入口的保留头规则。" }
    ]
  }'
```

### `messages` 详细规则

`POST /v1/messages` 是 Claude 风格接入的核心路径，使用时请注意：

- 这是当前最稳定的 Claude 入口。
- 当分组平台是 Anthropic 时按原生协议透传。
- 当分组平台是 OpenAI / Copilot 时，网关可能把请求翻译成 Responses。
- Grok 平台不支持 `messages`，会返回协议能力错误。

实际接入建议：

- Claude SDK、Claude Code 优先用这一条路径。
- 如果你的项目未来要兼容多个平台，请让调用方固定协议，不要同一个客户端实例一会儿用 `messages`，一会儿改成 `responses`。

#### Python
```python
import requests

payload = {
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 512,
    "messages": [
        {"role": "user", "content": "用 Claude 风格给我写一个接入检查清单。"}
    ],
}

response = requests.post(
    "https://api.zyxai.de/v1/messages",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
        "anthropic-version": "2023-06-01",
    },
    json=payload,
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const payload = {
  model: "claude-sonnet-4-20250514",
  max_tokens: 512,
  messages: [{ role: "user", content: "用 Claude 风格给我写一个接入检查清单。" }],
};

const response = await fetch("https://api.zyxai.de/v1/messages", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
    "anthropic-version": "2023-06-01",
  },
  body: JSON.stringify(payload),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1/messages \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "max_tokens": 512,
    "messages": [
      { "role": "user", "content": "用 Claude 风格给我写一个接入检查清单。" }
    ]
  }'
```

### `count_tokens` 规则

`POST /v1/messages/count_tokens` 的结论要比 `messages` 更严格：

- 只应当对 Anthropic 原生平台抱有成功预期。
- OpenAI、Copilot、Grok、Antigravity 在当前能力矩阵中都不是成功面。
- 你即使看到了路由存在，也不应该把它当作“所有 Claude 风格入口都支持”的信号。

也就是说，`count_tokens` 是一条“能力收窄”的路径，而不是“语法兼容就一定成功”的路径。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/v1/messages/count_tokens",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
        "anthropic-version": "2023-06-01",
    },
    json={
        "model": "claude-sonnet-4-20250514",
        "messages": [{"role": "user", "content": "请估算这段文本的 token 数。"}],
    },
    timeout=60,
)

print(response.status_code, response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/v1/messages/count_tokens", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
    "anthropic-version": "2023-06-01",
  },
  body: JSON.stringify({
    model: "claude-sonnet-4-20250514",
    messages: [{ role: "user", content: "请估算这段文本的 token 数。" }],
  }),
});

console.log(response.status, await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1/messages/count_tokens \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4-20250514",
    "messages": [
      { "role": "user", "content": "请估算这段文本的 token 数。" }
    ]
  }'
```

### 平台差异与常见坑

- `messages` 可兼容翻译，不代表 `count_tokens` 也可兼容翻译。
- 不要去掉 `anthropic-version`，除非你非常确定当前客户端和上游都不依赖它。
- 如果你在 Antigravity 前缀下调用 `messages/count_tokens`，当前实现里应该预期失败，而不是预期与 Anthropic 原生等价。
- 遇到 `404` 时优先判断平台不支持，而不是先怀疑 SDK 库版本。

## gemini
> 本页面向 Gemini / Google 风格客户端。重点覆盖 `models`、`files`、上传下载、检索资源、`batches`、`live`、`authTokens`、`openai compat` 与当前认证优先级。

### 协议定位与资源族总览

Gemini 原生入口是整个项目里资源族最丰富的一组。你可以把它分成 6 类：

1. 模型与生成：`/v1/models`、`/v1beta/models`
2. 文件与检索：`/v1beta/files`、`/upload/v1beta/files`、`/download/v1beta/files`、`/v1beta/fileSearchStores`
3. 长任务与批处理：`/v1beta/batches`、`/v1beta/operations`
4. 专用资源族：`cachedContents`、`documents`、`embeddings`、`interactions`、`corpora`、`dynamic`、`generatedFiles`、`tunedModels`
5. 实时会话：`/v1beta/live`、`/v1alpha/authTokens`
6. Gemini 的 OpenAI 兼容层：`/v1beta/openai/...`

其中最容易被忽略的点有两个：

- `/v1/models` 与 `/v1beta/models` 都存在，前者更像 Google 官方样式总入口，后者是 Gemini 资源族主干。
- `upload` / `download` 是独立根路径，不挂在 `/v1beta` 组里面。

### 认证优先级与查询参数规则

Gemini / Google 风格中间件的优先级是：

1. `x-goog-api-key`
2. `Authorization: Bearer <API_KEY>`
3. `x-api-key`
4. 允许时的 `?key=<API_KEY>`

关于查询参数，必须记住以下结论：

- `?api_key=...`：直接返回 `400`
- `?key=...`：只在这些路径前缀上保留兼容
  - `/v1beta/...`
  - `/antigravity/v1beta/...`
  - `/v1/models...`
  - `/v1alpha/authTokens`
  - `/upload/v1beta/files`
  - `/upload/v1beta/fileSearchStores...`
- `/v1/projects/:project/locations/:location/...` 和 `/google/batch/archive/...` 不在白名单里，请用请求头

#### Python
```python
import requests

response = requests.get(
    "https://api.zyxai.de/v1/models",
    headers={"x-goog-api-key": "sk-你的站内Key"},
    timeout=30,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/v1beta/models", {
  headers: {
    Authorization: "Bearer sk-你的站内Key",
  },
});

console.log(await response.json());
```

#### REST
```bash
curl "https://api.zyxai.de/v1beta/models?key=sk-你的站内Key"
```

### `models` 与文本生成

模型相关的关键路径包括：

| 动作 | 路径 |
| --- | --- |
| 列表 / 详情 | `GET /v1/models`、`GET /v1/models/:model`、`GET /v1beta/models`、`GET /v1beta/models/:model` |
| 文本生成 | `POST /v1beta/models/{model}:generateContent` |
| 流式生成 | `POST /v1beta/models/{model}:streamGenerateContent` |
| 统计 token | `POST /v1beta/models/{model}:countTokens` |
| 生成答案 | `POST /v1beta/models/{model}:generateAnswer` |
| 向量与批向量 | `POST /v1/models/{model}:embedContent`、`POST /v1beta/models/{model}:batchEmbedContents`、`POST /v1beta/models/{model}:asyncBatchEmbedContent` |

兼容差异：

- 在 Gemini 平台下这些动作按原生协议直通。
- 在 Antigravity 平台下，`generateContent`、`streamGenerateContent`、`countTokens` 可以兼容翻译。
- `batchGenerateContent` 在 Antigravity 平台下当前明确拒绝。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/v1beta/models/gemini-2.5-pro:generateContent",
    headers={
        "x-goog-api-key": "sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "contents": [
            {
                "role": "user",
                "parts": [{"text": "请总结 Gemini 原生入口的核心资源族。"}],
            }
        ]
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch(
  "https://api.zyxai.de/v1beta/models/gemini-2.5-flash:countTokens",
  {
    method: "POST",
    headers: {
      "x-goog-api-key": "sk-你的站内Key",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      contents: [{ role: "user", parts: [{ text: "请估算 token。" }] }],
    }),
  }
);

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1beta/models/gemini-2.5-pro:streamGenerateContent \
  -H "x-goog-api-key: sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [{ "text": "请给我一份 Gemini 资源族速查表。" }]
      }
    ]
  }'
```

### `files`、上传下载与检索资源

文件相关能力拆成三层：

1. 基础文件资源：`/v1beta/files`
2. 独立上传 / 下载：`/upload/v1beta/files`、`/download/v1beta/files/*subpath`
3. 检索扩展：`/v1beta/fileSearchStores`、`/v1beta/documents`、`/v1beta/operations`

常见用法：

- 先上传文件，再在 `fileSearchStores` 中导入或上传到指定检索库
- 通过 `documents` 与 `operations` 查看异步处理状态
- 对下载动作使用 `download` 根路径，而不是自己猜子资源 URL

#### Python
```python
import requests

files = {
    "file": ("guide.txt", b"Sub2API Gemini file upload example", "text/plain"),
}

response = requests.post(
    "https://api.zyxai.de/upload/v1beta/files",
    headers={"x-goog-api-key": "sk-你的站内Key"},
    files=files,
    timeout=120,
)

print(response.json())
```

#### JavaScript
```javascript
const formData = new FormData();
formData.append("file", new Blob(["Sub2API Gemini file upload example"]), "guide.txt");

const response = await fetch("https://api.zyxai.de/upload/v1beta/files", {
  method: "POST",
  headers: {
    "x-goog-api-key": "sk-你的站内Key",
  },
  body: formData,
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1beta/fileSearchStores/default-store:importFile \
  -H "x-goog-api-key: sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "fileUri": "files/abc123"
  }'
```

### `batches`、`operations` 与长任务

Batch 相关路径包括：

- `GET /v1beta/batches`
- `GET /v1beta/batches/*subpath`
- `POST /v1beta/batches/*subpath`
- `PATCH /v1beta/batches/*subpath`
- `DELETE /v1beta/batches/*subpath`
- `POST /v1beta/models/{model}:batchGenerateContent`

这里最重要的规则是：

- Gemini 平台下 `batchGenerateContent` 是原生支持的。
- Antigravity 平台下同名动作当前明确拒绝，不要把 `/antigravity/v1beta/models/...:batchGenerateContent` 当成稳定能力。
- Batch 任务通常会和 `operations`、归档下载、文件资源联动，请把整个流程看成异步系统，而不是单次同步请求。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/v1beta/models/gemini-2.5-pro:batchGenerateContent",
    headers={
        "x-goog-api-key": "sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "requests": [
            {
                "request": {
                    "contents": [
                        {
                            "role": "user",
                            "parts": [{"text": "为批任务返回一句问候语。"}],
                        }
                    ]
                }
            }
        ]
    },
    timeout=120,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/v1beta/batches/batch_123", {
  headers: {
    "x-goog-api-key": "sk-你的站内Key",
  },
});

console.log(await response.json());
```

#### REST
```bash
curl -X PATCH https://api.zyxai.de/v1beta/batches/batch_123:updateGenerateContentBatch \
  -H "x-goog-api-key: sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{}'
```

### `live`、`authTokens` 与实时会话

实时会话相关的入口有三组：

| 入口 | 说明 |
| --- | --- |
| `GET/POST /v1beta/live` | Gemini Live 主入口 |
| `GET/POST /v1beta/live/*subpath` | Live 子路径 |
| `POST /v1alpha/authTokens` | 官方推荐的 Live 授权 token 入口 |

兼容别名仍然存在，但不建议作为新文档主路径：

- `POST /v1beta/live/auth-token`
- `POST /v1beta/live/auth-tokens`

调用建议：

- 如果你是普通 HTTP 客户端，请先调用 `authTokens` 获取会话票据。
- 如果你的运行账号是 API Key 账号，Gemini Live 上游 WebSocket URL 里会使用 `?key=` 形式。
- 如果你的运行账号是 OAuth 账号，会改为 Bearer access token。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/v1alpha/authTokens",
    headers={
        "x-goog-api-key": "sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={"ttl": 60},
    timeout=30,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/v1beta/live/auth-token", {
  method: "POST",
  headers: {
    "x-goog-api-key": "sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({ ttl: 60 }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1alpha/authTokens \
  -H "x-goog-api-key: sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{"ttl":60}'
```

### OpenAI 兼容层

Gemini 还暴露了一套专用 OpenAI compat 路径：

- `/v1beta/openai/models`
- `/v1beta/openai/files`
- `/v1beta/openai/batches`
- `/v1beta/openai/chat/completions`
- `/v1beta/openai/embeddings`
- `/v1beta/openai/images/generations`
- `/v1beta/openai/videos`

这套入口适合“客户端只会说 OpenAI，但你又明确要把它落到 Gemini 资源面”的场景。它不是普通 `/v1/chat/completions` 的等价替代，而是 Gemini 自己的兼容层。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/v1beta/openai/chat/completions",
    headers={
        "x-goog-api-key": "sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "model": "gemini-2.5-flash",
        "messages": [{"role": "user", "content": "请说明 Gemini OpenAI compat 的定位。"}],
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/v1beta/openai/embeddings", {
  method: "POST",
  headers: {
    "x-goog-api-key": "sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "text-embedding-004",
    input: "Gemini OpenAI compat embedding sample",
  }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1beta/openai/models \
  -H "x-goog-api-key: sk-你的站内Key"
```

### 常见坑与接入建议

- 不要把 `?api_key=` 当成 Gemini 的旧兼容写法继续保留，它会直接失败。
- `/v1/projects/...` 与 `/google/batch/archive/...` 都属于 Google 风格返回体，但不是 `?key=` 白名单路径。
- `upload`、`download`、`openai compat`、`live` 都是独立入口，不要自己拼到 `/v1beta/models` 下面。
- 批任务和实时会话都应当先在单次请求上验证认证，再扩展到异步链路。

## grok
> 本页面向 xAI / Grok 平台。重点是 Grok 专用前缀、聊天与 Responses、图像和视频能力，以及与 OpenAI 公共路径的区别。

### 协议定位与适用客户端

Grok 是一个“看起来很像 OpenAI，但并不是所有 OpenAI 公共动作都完全等价”的平台。

建议优先使用专用前缀：

- `/grok/v1/chat/completions`
- `/grok/v1/responses`
- `/grok/v1/images/generations`
- `/grok/v1/images/edits`
- `/grok/v1/videos`
- `/grok/v1/videos/:request_id`

这样做的好处是：

- 你一眼就知道当前调用目标是 Grok。
- 当公共 `/v1/...` 路径失败时，不会误判成全局协议问题。
- 媒体能力的权限边界更清楚。

### 聊天与 Responses

Grok 当前支持的核心文本入口是：

- `POST /grok/v1/chat/completions`
- `POST /grok/v1/responses`
- `POST /grok/v1/responses/*subpath`
- `GET /grok/v1/responses/*subpath`
- `DELETE /grok/v1/responses/*subpath`

需要特别注意：

- Grok 不支持 Anthropic 的 `messages` 入口。
- Grok 的 Responses WebSocket 动作在当前能力矩阵中被拒绝。
- 如果你只是做普通文本生成，`chat/completions` 和 `responses` 都可以；如果做新接入，优先统一到 `responses`。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/grok/v1/responses",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "model": "grok-3",
        "input": "请概括 Grok 专用路径的优点。",
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/grok/v1/chat/completions", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "grok-3",
    messages: [{ role: "user", content: "请概括 Grok 专用路径的优点。" }],
  }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/grok/v1/responses/resp_123 \
  -H "Authorization: Bearer sk-你的站内Key"
```

### 图像与视频能力

Grok 是当前媒体能力最明确的一组运行平台。可用路径包括：

| 动作 | 路径 |
| --- | --- |
| 图像生成 | `POST /grok/v1/images/generations` |
| 图像编辑 | `POST /grok/v1/images/edits` |
| 视频创建 | `POST /grok/v1/videos` |
| 视频创建别名 | `POST /grok/v1/videos/generations` |
| 视频状态查询 | `GET /grok/v1/videos/:request_id` |

如果你走的是公共 `/v1/images/...`、`/v1/videos...` 路径，也必须保证当前分组平台最终落到 Grok；否则会拿到能力不支持错误。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/grok/v1/images/generations",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "model": "grok-2-image",
        "prompt": "一张体现三栏文档站的界面概念图",
        "size": "1024x1024",
    },
    timeout=120,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/grok/v1/videos", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "grok-video",
    prompt: "一段展示协议导航滚动高亮效果的短视频",
  }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/grok/v1/videos/req_123 \
  -H "Authorization: Bearer sk-你的站内Key"
```

### 返回行为与限制

- Grok 没有 `messages` 能力；看到 `404` 或兼容错误时不要再尝试 Anthropic 风格入口。
- `GET /grok/v1/responses` 根路径不是通用查询入口，常见做法是访问具体子资源。
- 如果你需要图像、视频和文本都走同一个平台，Grok 是清晰的选择；如果你需要 Claude 工具链，就不要强行复用 Grok 分组。
- 调试时优先看你是否用了专用前缀，而不是先猜是鉴权失败。

### 接入建议

- 新接入优先使用 `/grok/v1/...`，不要依赖公共 `/v1/...` 去“猜中”当前平台。
- 统一把 Grok 作为 OpenAI 风格变体对待，而不是 Anthropic / Gemini 变体。
- 图像与视频接口耗时通常更长，调用侧要设置更高超时并允许轮询状态。

## antigravity
> 本页用于显式只走 Antigravity 平台的接入。`/antigravity` 前缀不是装饰，它意味着路由已经强制绑定到 Antigravity，不再进行普通混合调度。

### 协议定位与强制前缀

Antigravity 相关路由分成三组：

- `GET /antigravity/models`
- `/antigravity/v1/...`：Anthropic 风格入口
- `/antigravity/v1beta/...`：Gemini 风格入口

与普通公共路径相比，`/antigravity/...` 的核心区别是：

- 路由中间件会强制平台为 Antigravity。
- 这适合你明知要落到 Antigravity、并且不希望被其他平台接管的场景。
- 也正因为它是强制平台，所以某些“在公共混合路径里可兼容翻译”的动作，在这里可能会被明确拒绝。

### Anthropic 风格入口

Anthropic 风格路径包括：

- `POST /antigravity/v1/messages`
- `POST /antigravity/v1/messages/count_tokens`
- `GET /antigravity/v1/models`
- `GET /antigravity/v1/usage`

需要准确理解两点：

- `messages` 是 Antigravity 的主文本入口之一，可以按 Anthropic 风格发送。
- `messages/count_tokens` 虽然注册了路由，但当前能力矩阵里对 Antigravity 平台并不视为成功面，不能当稳定能力依赖。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/antigravity/v1/messages",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
        "anthropic-version": "2023-06-01",
    },
    json={
        "model": "claude-sonnet-4-20250514",
        "max_tokens": 256,
        "messages": [{"role": "user", "content": "请说明 Antigravity 前缀的作用。"}],
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch("https://api.zyxai.de/antigravity/v1/messages", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
    "anthropic-version": "2023-06-01",
  },
  body: JSON.stringify({
    model: "claude-sonnet-4-20250514",
    max_tokens: 256,
    messages: [{ role: "user", content: "请说明 Antigravity 前缀的作用。" }],
  }),
});

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/antigravity/v1/usage \
  -H "Authorization: Bearer sk-你的站内Key"
```

### Gemini 风格入口

Gemini 风格路径包括：

- `GET /antigravity/v1beta/models`
- `GET /antigravity/v1beta/models/:model`
- `POST /antigravity/v1beta/models/*modelAction`

当前行为可以概括为：

- `generateContent`、`streamGenerateContent`、`countTokens` 可以走兼容翻译。
- `batchGenerateContent` 当前能力矩阵明确拒绝。
- 这条前缀非常适合“我要让 Gemini 风格客户端只打到 Antigravity”的场景。

#### Python
```python
import requests

response = requests.post(
    "https://api.zyxai.de/antigravity/v1beta/models/gemini-2.5-pro:generateContent",
    headers={
        "x-goog-api-key": "sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "contents": [
            {
                "role": "user",
                "parts": [{"text": "请解释 Antigravity 的 Gemini 风格入口。"}],
            }
        ]
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch(
  "https://api.zyxai.de/antigravity/v1beta/models/gemini-2.5-flash:countTokens",
  {
    method: "POST",
    headers: {
      "x-goog-api-key": "sk-你的站内Key",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      contents: [{ role: "user", parts: [{ text: "请解释 Antigravity 的 Gemini 风格入口。" }] }],
    }),
  }
);

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/antigravity/v1beta/models \
  -H "x-goog-api-key: sk-你的站内Key"
```

### 已注册但不建议依赖的动作

这一节非常重要，因为它决定了你能否安全长期维护：

- `/antigravity/v1/messages/count_tokens`：当前不应视为稳定可用能力。
- `/antigravity/v1beta/models/{model}:batchGenerateContent`：当前能力矩阵明确拒绝。

如果你的调用链确实需要这些动作，建议回到公共协议页重新评估是否应该改用 Anthropic 原生或 Gemini 原生分组，而不是在 Antigravity 强制前缀下硬做兼容。

#### Python
```python
# 当前不建议依赖 antigravity /v1/messages/count_tokens。
# 如果你的业务必须使用 token 预估，请改为 Anthropic 原生分组测试，
# 不要把这个前缀当成与 Anthropic 完全等价的稳定能力面。
```

#### JavaScript
```javascript
// 当前不建议依赖 /antigravity/v1beta/models/{model}:batchGenerateContent。
// 如果你需要稳定批任务，请优先回到 Gemini 原生分组。
```

#### REST
```bash
# 当前协议页下，这些动作属于“路由存在，但能力矩阵不建议依赖”的范围。
# 排障时请优先检查平台能力，而不是继续重试。
```

### 接入建议

- 只有在你明确要“固定走 Antigravity”时才使用该前缀。
- 不要把公共混合入口与 `/antigravity/...` 混在同一个客户端实例里。
- 对 token 预估、批任务这类能力收窄动作，要单独验证，不要因为路径能访问就假设功能完整。

## vertex-batch
> 本页说明 Google 风格的特殊入口：Vertex 模型动作、Vertex Batch Prediction Jobs，以及 Google Batch Archive 的查询和下载。

### 协议定位与适用场景

这一页和 Gemini 原生页的区别在于：

- 它讨论的是 Vertex / Batch 专用路径，而不是通用 `v1beta` 资源族。
- 返回体仍然是 Google 风格错误。
- 认证依然使用站内 API Key，而不是让你直接把 GCP 凭据暴露给客户端。

适用场景包括：

- 你要驱动 Vertex 的模型动作路径
- 你要创建、轮询、取消 `batchPredictionJobs`
- 你要读取 Google Batch Archive 中的批任务归档和归档文件

### Vertex 模型动作与 `batchPredictionJobs`

Vertex 相关路径：

- `POST /v1/projects/:project/locations/:location/publishers/google/models/*modelAction`
- `GET /v1/projects/:project/locations/:location/batchPredictionJobs`
- `POST /v1/projects/:project/locations/:location/batchPredictionJobs`
- `GET /v1/projects/:project/locations/:location/batchPredictionJobs/*subpath`
- `POST /v1/projects/:project/locations/:location/batchPredictionJobs/*subpath`
- `DELETE /v1/projects/:project/locations/:location/batchPredictionJobs/*subpath`

这里有两个最容易踩的坑：

- 这组路径不在 `?key=` 兼容白名单内，推荐使用请求头。
- 它虽然是 Google 风格返回体，但并不是普通 Gemini `v1beta` 路径的别名。

#### Python
```python
import requests

project = "demo-project"
location = "us-central1"

response = requests.post(
    f"https://api.zyxai.de/v1/projects/{project}/locations/{location}/batchPredictionJobs",
    headers={
        "x-goog-api-key": "sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "displayName": "sub2api-batch-job",
        "model": "publishers/google/models/gemini-2.5-pro",
        "inputConfig": {"instancesFormat": "jsonl"},
        "outputConfig": {"predictionsFormat": "jsonl"},
    },
    timeout=120,
)

print(response.json())
```

#### JavaScript
```javascript
const project = "demo-project";
const location = "us-central1";

const response = await fetch(
  `https://api.zyxai.de/v1/projects/${project}/locations/${location}/batchPredictionJobs`,
  {
    method: "GET",
    headers: {
      Authorization: "Bearer sk-你的站内Key",
    },
  }
);

console.log(await response.json());
```

#### REST
```bash
curl https://api.zyxai.de/v1/projects/demo-project/locations/us-central1/publishers/google/models/gemini-2.5-pro:generateContent \
  -H "x-goog-api-key: sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [{ "text": "请概括 Vertex 模型动作路径的用途。" }]
      }
    ]
  }'
```

### Google Batch Archive

Google Batch Archive 当前开放两个只读路径：

- `GET /google/batch/archive/v1beta/batches/*subpath`
- `GET /google/batch/archive/v1beta/files/*subpath`

它们的作用很明确：

- 查询归档后的批任务元信息
- 下载归档后的输出文件

这组接口尤其适合“批任务已经结束，但还要回查结果”的运维和审计场景。

#### Python
```python
import requests

response = requests.get(
    "https://api.zyxai.de/google/batch/archive/v1beta/batches/batch_123",
    headers={"x-goog-api-key": "sk-你的站内Key"},
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript
const response = await fetch(
  "https://api.zyxai.de/google/batch/archive/v1beta/files/file_123:download",
  {
    headers: {
      Authorization: "Bearer sk-你的站内Key",
    },
  }
);

console.log(response.status);
```

#### REST
```bash
curl https://api.zyxai.de/google/batch/archive/v1beta/batches/batch_123 \
  -H "x-goog-api-key: sk-你的站内Key"
```

### 认证、权限与常见坑

这一页最容易犯错的是把它和普通 Gemini `v1beta` 路径混为一谈。请记住：

- 它仍然使用 Google 风格中间件，因此支持 `x-goog-api-key`、`Authorization: Bearer`、`x-api-key`。
- 它不在 `?key=` 白名单里，所以不要把示例写成查询参数认证。
- 因为这是特殊能力面，排错时要先确认分组平台确实是 Google / Gemini 能力面，而不是只看路径像不像 Google。

建议流程：

1. 先在普通 Gemini 原生页确认你的 Key 与分组能通过 Google 风格鉴权。
2. 再测试 `publishers/google/models` 或 `batchPredictionJobs`。
3. 批任务完成后，最后再接归档查询和文件下载。

