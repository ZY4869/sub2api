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
```python focus=1-12
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
```javascript focus=1-10
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
```bash focus=1-6
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
```python focus=1-12
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
```javascript focus=1-10
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
```bash focus=1-6
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
