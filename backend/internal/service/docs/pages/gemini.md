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

服务端如果使用 Gemini OAuth 账号作为上游运行身份，还要注意一条接入约束：

- `google_one` 账号只有在服务端配置了自定义 OAuth client 时，才会申请 `drive.readonly` 并启用 Drive quota-based tier detection。
- 如果仍使用内置 Gemini CLI OAuth client，Google One 会继续走兼容模式，不要求重授权，也不会依赖 Drive scope。

#### Python
```python focus=1-12
import requests

response = requests.get(
    "https://api.zyxai.de/v1/models",
    headers={"x-goog-api-key": "sk-你的站内Key"},
    timeout=30,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-10
const response = await fetch("https://api.zyxai.de/v1beta/models", {
  headers: {
    Authorization: "Bearer sk-你的站内Key",
  },
});

// 若账号映射了 friendly-flash -> gemini-2.0-flash，
// 返回体只会看到 models/friendly-flash。
console.log(await response.json());
```

#### REST
```bash focus=1-6
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
- `GET /v1/models`、`GET /v1beta/models`、`GET /v1/models/:model`、`GET /v1beta/models/:model` 都会先经过账号级白名单 / 模型映射投影，只暴露 display ID。
- 如果账号把 `gemini-2.0-flash` 映射成 `friendly-flash`，那么列表与详情只会返回 `models/friendly-flash`；真实模型名不会再出现在返回体里，也不能再当作公开模型 ID 查询详情。
- 这些 `models` 读路径只读取本地策略投影和本地 availability snapshot，不会在请求时同步触发 Vertex / Gemini 上游探测来扩充列表。

#### Python
```python focus=1-12
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
```javascript focus=1-10
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
```bash focus=1-6
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
```python focus=1-12
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
```javascript focus=1-10
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
```bash focus=1-6
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
```python focus=1-12
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
```javascript focus=1-10
const response = await fetch("https://api.zyxai.de/v1beta/batches/batch_123", {
  headers: {
    "x-goog-api-key": "sk-你的站内Key",
  },
});

console.log(await response.json());
```

#### REST
```bash focus=1-6
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
```python focus=1-12
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
```javascript focus=1-10
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
```bash focus=1-6
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

对图片能力要额外记住两条：

- 公共 `POST /v1/images/generations` 现在也可以命中 Gemini 图片模型；网关内部会把它转到 Gemini 的 `/v1beta/openai/images/generations` 兼容链路。
- `POST /v1/images/edits` 目前不会对 Gemini 开放；如果模型最终解析到 Gemini provider，会直接返回该动作不支持。

#### Python
```python focus=1-12
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

#### REST
```bash focus=1-7
curl https://api.zyxai.de/v1beta/openai/images/generations \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-flash-image",
    "prompt": "生成一张简洁的 SaaS 首页插图。"
  }'
```

#### JavaScript
```javascript focus=1-10
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
```bash focus=1-6
curl https://api.zyxai.de/v1beta/openai/models \
  -H "x-goog-api-key: sk-你的站内Key"
```

### 常见坑与接入建议

- 不要把 `?api_key=` 当成 Gemini 的旧兼容写法继续保留，它会直接失败。
- `/v1/projects/...` 与 `/google/batch/archive/...` 都属于 Google 风格返回体，但不是 `?key=` 白名单路径。
- `upload`、`download`、`openai compat`、`live` 都是独立入口，不要自己拼到 `/v1beta/models` 下面。
- 批任务和实时会话都应当先在单次请求上验证认证，再扩展到异步链路。
