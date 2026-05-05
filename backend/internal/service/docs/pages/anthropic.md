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

Thinking 强度补充说明：

- 顶层 `effortLevel` 是 Claude 定向补位字段，可接受：`low`、`medium`、`high`、`xhigh`、`max`。
- 优先级固定为：`output_config.effort` 优先，顶层 `effortLevel` 只在原生字段缺失时补位，不会报错也不会覆盖原值。
- Claude 原生 `messages` 请求支持通过 `output_config.effort` 传递思考强度。
- 当前网关按原样识别并透传 5 档：`low`、`medium`、`high`、`xhigh`、`max`。
- 对 Anthropic 原生上游，`xhigh` 与 `max` 都会原样发送，不会再被静默折叠。
- 如果你希望 Claude Code 内还能继续动态切换，优先使用顶层 `effortLevel` 作为默认值。
- 如果你需要强制锁定某一档位，可使用 `CLAUDE_CODE_EFFORT_LEVEL`；该环境变量会覆盖客户端内后续切换。
- 对第三方 Claude 风格模型，建议同时声明模型能力（例如 `SUPPORTED_CAPABILITIES`），让 Claude Code 能正确显示 `xhigh` / `max` 等档位。

观测与请求详情也会同时保留两种口径：

- `reasoning_effort_raw`：用户原始意图，例如 `max`
- `reasoning_effort_effective`：实际上游发送值；Anthropic 原生路径下通常与 `raw` 相同

Claude Cloud 1M 上下文补充说明：

- 你可以在请求时把模型写成 `claude-sonnet-4.5[1m]`，表示“希望启用 Claude Cloud 1M context”。
- `[1m]` 只接受模型名尾部标准后缀；网关会先剥离后缀，再继续做模型归一化、策略匹配和上游路由。
- `[1m]` 不会变成新的公开模型 ID：`/v1/models`、模型策略和前端模型选择器里都不会出现 `claude-sonnet-4.5[1m]` 这种枚举项。
- 当前 `[1m]` 的最终实现方式是注入 `anthropic-beta: context-1m-2025-08-07`；如果请求本身已带 `anthropic-beta`，网关会合并并去重。
- 官方 Claude 模型以及 `deepseek-v4-flash`、`deepseek-v4-pro` 当前纳入 `[1m]` 支持名单；其它模型即使带了 `[1m]` 也只会静默忽略，不会报错。
- 请求详情与使用记录会额外保留 `requested_model_raw`、`requested_model_normalized`、`million_context_requested`、`million_context_effective`、`million_context_source`、`million_context_beta_token`，用于区分“用户请求了 1M”与“实际上游是否启用 1M”。
- 与 `openai` / `gemini` 页不同，Anthropic 家族入口当前就是 `[1m]` 真正会落到 Claude beta 注入的主生效面；其它入口目前只保证接受、剥离和观测，不承诺在当前运行时矩阵下直接转成同样的上游效果。

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

#### Python
```python focus=1-16
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
        "model": "claude-sonnet-4.5[1m]",
        "max_tokens": 512,
        "effortLevel": "max",
        "messages": [{"role": "user", "content": "请给我一份 Claude 1M 接入说明。"}],
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-14
const response = await fetch("https://api.zyxai.de/v1/messages", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
    "anthropic-version": "2023-06-01",
  },
  body: JSON.stringify({
    model: "deepseek-v4-pro[1m]",
    effortLevel: "max",
    output_config: { effort: "high" },
    max_tokens: 512,
    messages: [{ role: "user", content: "验证协议字段优先于顶层 effortLevel。" }],
  }),
});

console.log(await response.json());
```

#### REST
```bash focus=1-8
curl https://api.zyxai.de/v1/messages \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-sonnet-4.5[1m]",
    "effortLevel": "max",
    "output_config": { "effort": "high" },
    "max_tokens": 256,
    "messages": [{ "role": "user", "content": "这里最终会保持 high，而不是被 max 覆盖。" }]
  }'
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
