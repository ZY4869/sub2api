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
```python focus=1-12
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
```javascript focus=1-10
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
```bash focus=1-6
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
```python focus=1-12
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
```javascript focus=1-10
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
```bash focus=1-6
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

### cache_control 与 TTL（管理员可选）

Anthropic 的 prompt cache 语义会通过 `cache_control` 体现（例如 `{"cache_control":{"type":"ephemeral"}}`）。在部分 OAuth / SetupToken 账号下，管理员可以开启一个“统一 TTL 注入”开关：

- 入口：`GET/PUT /api/v1/admin/settings` 字段 `enable_anthropic_cache_ttl_1h_injection`（默认关闭）
- 生效范围：仅对 Anthropic OAuth / SetupToken 账号生效
- 注入规则：当 `cache_control.type=="ephemeral"` 时，网关会在转发前写入或覆盖 `cache_control.ttl="1h"`（当前仅处理 `system[]` 与 `messages[].content[]` 里的结构化 block）

如果你需要自定义 TTL，请与管理员确认该开关是否开启；开启后，非 `"1h"` 的 `ttl` 可能会被归一为 `"1h"`。

### `count_tokens` 规则

`POST /v1/messages/count_tokens` 的结论要比 `messages` 更严格：

- 只应当对 Anthropic 原生平台抱有成功预期。
- OpenAI、Copilot、Grok、Antigravity 在当前能力矩阵中都不是成功面。
- 你即使看到了路由存在，也不应该把它当作“所有 Claude 风格入口都支持”的信号。

也就是说，`count_tokens` 是一条“能力收窄”的路径，而不是“语法兼容就一定成功”的路径。

#### Python
```python focus=1-12
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
```javascript focus=1-10
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
```bash focus=1-6
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
