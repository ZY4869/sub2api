## deepseek
> 本页面向 DeepSeek 一级平台。重点是官方 OpenAI / Anthropic 兼容入口、站内强制平台前缀，以及不支持能力边界。

### 协议定位与适用客户端

DeepSeek 是独立运行平台，不归入 OpenAI 或 Anthropic 账号族。账号类型固定为 API Key，后台默认上游根地址为：

- OpenAI 兼容：`https://api.deepseek.com`
- Anthropic 兼容：`https://api.deepseek.com/anthropic`

站内推荐显式前缀：

- `GET /deepseek/v1/models`
- `POST /deepseek/v1/chat/completions`
- `POST /deepseek/v1/messages`
- `POST /v1/completions`（DeepSeek FIM / Beta completion 专用公共入口）

公共 `/v1/chat/completions`、`/v1/models`、`/v1/messages` 也可以按 DeepSeek 分组路由，但显式 `/deepseek/v1/...` 更适合排障和避免多平台歧义。只有 FIM / Beta completion 例外：本站只开放公共 `POST /v1/completions`，不提供 `/deepseek/v1/completions` 或根级 `/completions` 别名。

### OpenAI 兼容入口

DeepSeek OpenAI 兼容面分成两块：

- `chat/completions`：默认稳定面，但可按参数或显式开关切到 beta
- `completions`：公共 `POST /v1/completions`，固定走 beta

对于 `chat/completions`，网关只定点改写 `model` 与流式 `stream_options.include_usage`；其余官方字段会尽量原样透传，包括：

- `thinking`
- `response_format`
- `logprobs`
- `top_logprobs`
- assistant message 上的 `prefix`
- assistant message 上的 `reasoning_content`

> 为满足 DeepSeek 官方“限速与隔离”建议，DeepSeek 上游请求中的 `user_id` 由网关统一注入内部派生值。即使下游请求传入了顶层 `user_id`，转发到 DeepSeek OpenAI 兼容入口时也会被覆盖，避免把用户邮箱、Token、原始 API Key 或其它隐私标识透传给上游。

`chat/completions` 额外支持一个网关私有请求字段：

- 顶层 `beta?: boolean`

这个字段只用于本站路由决策，不会透传到 DeepSeek 上游。优先级规则如下：

- `beta: true`：显式强制走 `https://api.deepseek.com/beta/chat/completions`
- `beta: false`：显式强制走稳定面 `https://api.deepseek.com/chat/completions`
- 未传 `beta`：继续按官方 beta-only 参数自动识别

自动识别范围保持最小，仅包含：

- assistant message 上的 `prefix`
- assistant message 上的 `reasoning_content`

当 `chat/completions` 命中 beta 路径时，当前允许模型为：

- `deepseek-v4-flash`
- `deepseek-v4-pro`

如果 `beta: true` 但模型不在这两个 v4 模型内，网关会直接返回显式 `400 invalid_request_error`。如果未传 `beta`，但请求带了 `prefix` / `reasoning_content` 且模型不在允许名单，网关会自动剥离这两个 beta-only 字段并继续走稳定面。若传了 `beta: false`，即使请求里带有这两个 beta-only 字段，也会按同样的安全降级策略剥离后走稳定面。

#### Python
```python focus=1-14
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"

response = requests.post(
    f"{base_url}/deepseek/v1/chat/completions",
    headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
    json={
        "model": "deepseek-v4-flash",
        "beta": True,
        "messages": [{"role": "user", "content": "用一句话说明 DeepSeek 专用前缀的作用。"}],
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-12
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";

const response = await fetch(`${baseUrl}/deepseek/v1/chat/completions`, {
  method: "POST",
  headers: { Authorization: `Bearer ${apiKey}`, "Content-Type": "application/json" },
  body: JSON.stringify({
    model: "deepseek-v4-flash",
    beta: true,
    messages: [{ role: "user", content: "用一句话说明 DeepSeek 专用前缀的作用。" }],
  }),
});

console.log(await response.json());
```

#### REST
```bash focus=1-7
curl https://api.zyxai.de/deepseek/v1/chat/completions \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-flash",
    "beta": true,
    "messages": [{ "role": "user", "content": "用一句话说明 DeepSeek 专用前缀的作用。" }]
  }'
```

### FIM / Beta Completion

DeepSeek FIM / Beta completion 在本站只开放公共 `POST /v1/completions`。这条路由仅对 DeepSeek 运行时平台可用，并固定转发到官方 `https://api.deepseek.com/beta/completions`。

当前网关内的 beta 允许模型为：

- `deepseek-v4-flash`
- `deepseek-v4-pro`

如果请求模型不在这两个 v4 模型内，网关会直接返回显式 `400 invalid_request_error`，不会把请求继续转发到上游。`/deepseek/v1/completions`、`/completions` 和 `/user/balance` 都不会开放给下游调用方。

FIM / Beta completion 也会按 DeepSeek OpenAI 兼容语义注入顶层 `user_id`。客户端传入的 `user_id` 会被覆盖为站内不可逆派生值。

#### Python
```python focus=1-14
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"

response = requests.post(
    f"{base_url}/v1/completions",
    headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
    json={
        "model": "deepseek-v4-pro",
        "prompt": "def fib(n):",
        "suffix": "\nprint(fib(8))",
        "max_tokens": 128,
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-12
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";

const response = await fetch(`${baseUrl}/v1/completions`, {
  method: "POST",
  headers: { Authorization: `Bearer ${apiKey}`, "Content-Type": "application/json" },
  body: JSON.stringify({
    model: "deepseek-v4-pro",
    prompt: "def fib(n):",
    suffix: "\nprint(fib(8))",
    max_tokens: 128,
  }),
});

console.log(await response.json());
```

#### REST
```bash focus=1-8
curl https://api.zyxai.de/v1/completions \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "prompt": "def fib(n):",
    "suffix": "\nprint(fib(8))",
    "max_tokens": 128
  }'
```

### Anthropic 兼容入口

DeepSeek 的 Anthropic 兼容入口通过 `/deepseek/v1/messages` 暴露。网关会转发到 DeepSeek 官方 `/anthropic/v1/messages`，并用 API Key 鉴权。

转发到 DeepSeek Anthropic 兼容入口时，网关会写入 `metadata.user_id`。如果客户端已经传入 `metadata.user_id`，该值会被覆盖为站内不可逆派生值；日志和审计不会记录派生 ID 原文。

当前 DeepSeek 官方 Anthropic 兼容能力不支持以下内容块：

- `image`
- `document`
- `search_result`
- `redacted_thinking`
- `server_tool_use`
- `web_search_tool_result`
- `code_execution_tool_result`
- `mcp_tool_use`
- `mcp_tool_result`
- `container_upload`

请求中出现这些内容块时，网关会在转发前返回能力错误，避免把不支持写法发送给上游。

#### Python
```python focus=1-15
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"

response = requests.post(
    f"{base_url}/deepseek/v1/messages",
    headers={"Authorization": f"Bearer {api_key}", "Content-Type": "application/json"},
    json={
        "model": "deepseek-v4-pro",
        "max_tokens": 256,
        "messages": [{"role": "user", "content": "请列出 DeepSeek Anthropic 兼容入口的限制。"}],
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-13
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";

const response = await fetch(`${baseUrl}/deepseek/v1/messages`, {
  method: "POST",
  headers: { Authorization: `Bearer ${apiKey}`, "Content-Type": "application/json" },
  body: JSON.stringify({
    model: "deepseek-v4-pro",
    max_tokens: 256,
    messages: [{ role: "user", content: "请列出 DeepSeek Anthropic 兼容入口的限制。" }],
  }),
});

console.log(await response.json());
```

#### REST
```bash focus=1-8
curl https://api.zyxai.de/deepseek/v1/messages \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-pro",
    "max_tokens": 256,
    "messages": [{ "role": "user", "content": "请列出 DeepSeek Anthropic 兼容入口的限制。" }]
  }'
```

### 模型枚举与模型 ID

`GET /deepseek/v1/models` 返回当前 DeepSeek 分组下的本地模型策略投影。模型可见性只来自账号白名单 / 映射或默认模型库；上游探测结果只用于状态标注，不会自动扩展可调用模型集合。

默认推荐模型：

- `deepseek-v4-flash`
- `deepseek-v4-pro`

兼容保留模型：

- `deepseek-chat`
- `deepseek-reasoner`

兼容保留模型已标记为官方 2026-07-24 弃用；DeepSeek beta、FIM 和本站新增的 V4 变体归一化链路只接受 `deepseek-v4-flash` / `deepseek-v4-pro` 及其可验证变体，新示例和新配置请使用 v4 模型 ID。

#### V4 变体归一化

DeepSeek V4 的运行时转发统一使用 canonical model ID：

- `deepseek-v4-flash`
- `deepseek-v4-pro`

后台账号模型策略仍遵守通用规则：公共模型列表只暴露管理员配置的 `display_model_id`，内部转发使用 `target_model_id`。当管理员在白名单 / 映射中配置了常见变体，例如：

- `Deepseek/deepseek V4 Flash:free`
- `deepseek_v4_flash_free`
- `DEEPSEEK V4 PRO`

保存和运行时会把 `target_model_id` 规范为对应 canonical ID，但保留原始 `display_model_id` 作为对外可见 ID。未知变体不会静默放行；无法归一化到受支持 V4 模型的请求会返回 `400 invalid_request_error`。

### user_id 隔离与并发限制

DeepSeek 官方文档《限速与隔离》说明：V4 账号并发默认值为 `deepseek-v4-pro=500`、`deepseek-v4-flash=2500`，并发限制以账号粒度计；`user_id` 需要匹配 `[a-zA-Z0-9\-_]+` 且最大长度为 512，不能包含用户隐私信息。

本站不透传客户端传入的 DeepSeek `user_id`。网关会使用站内用户 ID、API Key ID、账号 ID 与服务端密钥派生稳定不可逆的 `sub2api_<hash>`，并覆盖：

- OpenAI Chat Completions：顶层 `user_id`
- FIM / Beta Completion：顶层 `user_id`
- Anthropic Messages：`metadata.user_id`

DeepSeek API Key 账号可在后台配置模型级并发上限，保存于账号 `extra.deepseek_model_concurrency_limits`：

```json
{
  "deepseek_model_concurrency_limits": {
    "deepseek-v4-pro": 500,
    "deepseek-v4-flash": 2500
  }
}
```

运行时并发计算规则：

- 账号并发和模型并发都配置时，取两者较小值。
- 只配置账号并发时，使用账号并发。
- 只配置模型并发时，使用模型并发。
- 缺省、非正数或未知模型配置会被忽略。

非 DeepSeek 账号保存时会清理该字段，避免跨平台误配置。

#### Python
```python focus=1-10
import requests

models = requests.get(
    "https://api.zyxai.de/deepseek/v1/models",
    headers={"Authorization": "Bearer sk-你的站内Key"},
    timeout=30,
)

print(models.status_code)
print(models.json())
```

#### JavaScript
```javascript focus=1-7
const response = await fetch("https://api.zyxai.de/deepseek/v1/models", {
  headers: { Authorization: "Bearer sk-你的站内Key" },
});

console.log(response.status);
console.log(await response.json());
```

#### REST
```bash focus=1-2
curl https://api.zyxai.de/deepseek/v1/models \
  -H "Authorization: Bearer sk-你的站内Key"
```

### 不支持能力

DeepSeek 当前仍有一部分能力边界。以下内容会返回现有统一 capability error，或在网关侧保持关闭，不会继续向上游转发：

- `/deepseek/v1/messages/count_tokens`
- `/v1/responses` 或 `/deepseek/v1/responses`
- Images / Videos
- `/user/balance`（本站是中转场景，不向下游暴露上游 Key 余额）
- Anthropic `image`、`document`、`search_result` 以及工具/容器类 unsupported 内容块

#### Python
```python focus=1-12
import requests

response = requests.post(
    "https://api.zyxai.de/deepseek/v1/messages/count_tokens",
    headers={"Authorization": "Bearer sk-你的站内Key", "Content-Type": "application/json"},
    json={
        "model": "deepseek-v4-flash",
        "messages": [{"role": "user", "content": "count tokens"}],
    },
    timeout=30,
)

print(response.status_code, response.json())
```

#### JavaScript
```javascript focus=1-10
const response = await fetch("https://api.zyxai.de/deepseek/v1/messages/count_tokens", {
  method: "POST",
  headers: { Authorization: "Bearer sk-你的站内Key", "Content-Type": "application/json" },
  body: JSON.stringify({
    model: "deepseek-v4-flash",
    messages: [{ role: "user", content: "count tokens" }],
  }),
});

console.log(response.status, await response.json());
```

#### REST
```bash focus=1-7
curl https://api.zyxai.de/deepseek/v1/messages/count_tokens \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "deepseek-v4-flash",
    "messages": [{ "role": "user", "content": "count tokens" }]
  }'
```
