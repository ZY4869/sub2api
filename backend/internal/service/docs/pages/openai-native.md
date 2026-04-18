## openai-native
> 本页面向 Responses-first 的 OpenAI 客户端。重点是 `responses`、Responses 子资源，以及长连接 / 流式相关建议。

### 协议定位与适用场景

当你满足下面任一条件时，优先阅读本页：

- 你在使用新版 OpenAI SDK
- 你准备以 `responses` 作为长期维护的文本主入口
- 你需要统一文本、工具调用和后续的响应子资源管理

当前程序对 OpenAI 原生入口的理解是：

- `responses` 是当前最推荐的 OpenAI 文本主入口，适合新项目。
- `responses` 子资源保留了查询、删除和后续状态管理能力。
- 长连接 / WebSocket 风格动作只应当对明确支持的平台做成功预期。

### 推荐路径与最短请求

主要路径如下：

| 动作 | 推荐路径 | 历史别名 | 说明 |
| --- | --- | --- | --- |
| 创建 Responses | `POST /v1/responses` | `POST /responses` | 新项目优先 |
| 查询 / 删除 Responses 子资源 | `GET/DELETE /v1/responses/*subpath` | `GET/DELETE /responses/*subpath` | 保留 OpenAI 风格子资源访问 |
| Responses 长连接 / WebSocket 风格入口 | `GET /v1/responses` | `GET /responses` | 仅对支持的平台有意义 |

如果你在配置官方 SDK，最简单的方式是把 `Base URL` 指到根地址或 `/v1` 前缀，而不是继续使用历史别名。

#### Python
```python focus=3-12,15
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
        "input": "请总结 OpenAI 原生入口的推荐用法。",
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-11,13
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
    input: "请总结 OpenAI 原生入口的推荐用法。",
  }),
});

console.log(await response.json());
```

#### REST
```bash focus=1-5
curl https://api.zyxai.de/v1/responses \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-5.4",
    "input": "请总结 OpenAI 原生入口的推荐用法。"
  }'
```

### Responses 子资源与流式建议

`responses` 的核心规则如下：

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
```python focus=2-11
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
                "content": [{"type": "input_text", "text": "列出 Responses 与旧兼容入口的差别。"}],
            }
        ],
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-10
const response = await fetch("https://api.zyxai.de/v1/responses/resp_123", {
  method: "DELETE",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
  },
});

console.log(response.status);
```

#### REST
```bash focus=1-2
curl https://api.zyxai.de/v1/responses/resp_123 \
  -H "Authorization: Bearer sk-你的站内Key"
```

### 迁移与边界

- 如果你的客户端本来就支持 `responses`，不要再退回 `chat/completions`。
- 如果你只是为了历史兼容而使用旧参数结构，请切到 `openai` 兼容页。
- 如果你要做图像或视频能力，不要在本页继续扩展，直接转到 `grok` 页查看媒体入口。
- 新项目围绕 `responses` 设计，比继续叠加 `chat/completions` 特性债务更稳妥。
