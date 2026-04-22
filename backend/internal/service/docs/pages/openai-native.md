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
if response.status_code == 429:
    print("当前模型可能因为对应的 OpenAI Pro 额度侧冷却而暂时不可用。")
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
if (response.status === 429) {
  console.log("当前模型可能因为对应的 OpenAI Pro 额度侧冷却而暂时不可用。");
}
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
# 如果命中的模型在所有可路由 OpenAI Pro 账号上都只因对应额度侧冷却而不可服务，
# 这里会返回 429 rate_limit_error，而不是 404。
```

### Responses 子资源与流式建议

`responses` 的核心规则如下：

- `POST /v1/responses` 创建响应。
- `POST /v1/responses/*subpath` 用于保留 Responses 子资源动作。
- `GET /v1/responses/*subpath` 与 `DELETE /v1/responses/*subpath` 用于查询或删除子资源。
- `GET /v1/responses` 由专门的 Responses WebSocket / 长连接处理链路接管。
- 当运行平台为 OpenAI 或 Copilot 时，Responses 能力是原生直通。
- 当运行平台为 Grok 时，普通 `POST /grok/v1/responses` 可用，但 WebSocket 动作在能力矩阵中被拒绝。
- 如果你在 `POST /v1/responses` 里使用 `image_generation` tool，而上游返回的是 compact SSE 且终态 `response.completed.response.output` 为空，网关会根据 `response.image_generation_call.partial_image` 自动回填 `output[].content[].type = "output_image"`，`image_url` 为 data URI，方便非流式客户端直接消费。
- 如果上游账号是 OpenAI Pro，运行时额度会拆成两侧：`gpt-5.3-codex-spark*` 只占用 `Spark` 侧，其它 OpenAI 模型统一占用 `普通` 侧；一侧冷却不会连带阻断另一侧。
- 当 `POST /v1/responses` 的目标模型在所有可路由账号上都只因为对应额度侧冷却而不可服务时，接口会返回 `429 rate_limit_error`；如果是账号忙、上游故障或其它选路失败，仍然保持原来的 `503` / `502` 语义。
- `/v1/models`、`/v1beta/models` 和对应 detail 读路径会按当前运行时可服务性临时隐藏受限侧模型；这不代表模型被永久删除，也不会把这类临时隐藏改成 `404`。
- 管理后台单账号测试 `POST /api/v1/admin/accounts/:id/test` 会在发起上游请求前先做本地预检；命中受限侧时直接返回 `400`，提示 `Spark 冷却中`、`普通额度冷却中` 或整号冷却。

排错建议：

- 普通文本生成失败时，先确认你用的是 `POST` 而不是 `GET`。
- 看到 `404` 时，不要只怀疑路径拼写，还要看当前分组平台是否真的支持该动作。
- 如果你依赖持续连接或多轮状态链路，请优先在 OpenAI / Copilot 平台验证。
- 如果是 OpenAI Pro，`429 rate_limit_error` 现在还可能表示“相关额度侧正在冷却”，这时另一侧模型通常仍可继续调用。

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

### OpenAI / Codex 生图

OpenAI 侧现在建议明确区分两类能力：

- `gpt-image-2` 这类原生图片模型，走 `/v1/images/generations` 或 `/v1/images/edits`。
- `gpt-5.4`、`gpt-5.4-mini`、`gpt-5.4-pro` 这类主模型，如果要生图，优先走 `/v1/responses` + `tools:[{type:"image_generation"}]`。
- 网关不会解析 `$imagegen ...` 这类文本前缀本身；如果你的客户端或 Codex 最终发出来的是 Responses tool 请求，网关会按标准 `image_generation` tool 语义处理。

#### Python
```python focus=3-16
import requests

response = requests.post(
    "https://api.zyxai.de/v1/responses",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "model": "gpt-5.4-mini",
        "input": "生成一张适合产品页首屏的海报图。",
        "tools": [{"type": "image_generation", "model": "gpt-image-2"}],
    },
    timeout=120,
)

print(response.json())
```

#### REST
```bash focus=1-8
curl https://api.zyxai.de/v1/images/generations \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "生成一张适合产品页首屏的海报图。"
  }'
```

### 迁移与边界

- 如果你的客户端本来就支持 `responses`，不要再退回 `chat/completions`。
- 如果你只是为了历史兼容而使用旧参数结构，请切到 `openai` 兼容页。
- 如果你要走 `/v1/images/*`，请先确认目标模型到底是原生图片模型还是 tool 生图主模型；原生图片模型走 `/v1/images/*`，tool 生图主模型继续走 `/v1/responses`。
- 如果你要看 Grok 显式媒体入口，请转到 `grok` 页；如果你是在 `/v1/responses` 内通过 `image_generation` tool 生图，仍以本页规则为准。
- 新项目围绕 `responses` 设计，比继续叠加 `chat/completions` 特性债务更稳妥。
