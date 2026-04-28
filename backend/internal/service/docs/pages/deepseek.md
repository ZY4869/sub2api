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

公共 `/v1/chat/completions`、`/v1/models`、`/v1/messages` 也可以按 DeepSeek 分组路由，但显式 `/deepseek/v1/...` 更适合排障和避免多平台歧义。

### OpenAI 兼容入口

DeepSeek OpenAI 兼容入口只走官方稳定面：`chat/completions` 和 `models`。不要把 `/v1/responses`、图片、视频或 FIM / Beta completion 写法用于 DeepSeek 分组。

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
    "messages": [{ "role": "user", "content": "用一句话说明 DeepSeek 专用前缀的作用。" }]
  }'
```

### Anthropic 兼容入口

DeepSeek 的 Anthropic 兼容入口通过 `/deepseek/v1/messages` 暴露。网关会转发到 DeepSeek 官方 `/anthropic/v1/messages`，并用 API Key 鉴权。

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

兼容保留模型仍可调用，但已标记为官方 2026-07-24 弃用；新示例和新配置请使用 v4 模型 ID。

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

DeepSeek 当前只接入官方稳定兼容面。以下能力会返回现有统一 capability error，不会尝试向上游转发：

- `/deepseek/v1/messages/count_tokens`
- `/v1/responses` 或 `/deepseek/v1/responses`
- Images / Videos
- FIM / Beta completion
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
