## vertex-batch
> 本页说明 Sub2API 对外暴露的 Vertex / Batch 网关入口。默认推荐“简化 Vertex 入口”，让下游只需要站内 API Key 和网关地址；严格 `/v1/projects/...` 兼容入口继续保留给老 SDK 或必须完全对齐官方路径语义的场景。

### 协议定位与推荐策略

先记住这一页的产品策略：

- 新接入用户，优先使用简化入口：
  - 模型动作：`POST /v1/vertex/models/{model}:generateContent`
  - Batch：`GET|POST|DELETE /v1/vertex/batchPredictionJobs...`
  - 短别名：`GET|POST|DELETE /vertex-batch/jobs...`
- 老 SDK / 严格兼容场景，继续保留：
  - `POST /v1/projects/{project}/locations/{location}/publishers/google/models/{model}:...`
  - `GET|POST|DELETE /v1/projects/{project}/locations/{location}/batchPredictionJobs...`

简化入口和严格入口的核心区别是：

- 简化入口不要求下游显式填写真实 `project/location`；后台会从当前分组里自动选择可用 Vertex 账号并补全真实作用域。
- 下游面对的仍然是 Vertex 语义，不是 OpenAI 风格；只是路径和 body 形态更适合站内接入。
- Batch 简化入口对外返回的任务名会隐藏真实上游作用域，例如 `batchPredictionJobs/job-1`，不会把 `projects/.../locations/...` 直接暴露给调用方。
- 无论走简化入口还是严格入口，请求都必须打到你的网关地址，例如 `https://api.zyxai.de`，而不是直接访问 Google 官方域名。

### 简化 Vertex 模型入口

简化模型入口当前支持这些动作：

- `POST /v1/vertex/models/{model}:generateContent`
- `POST /v1/vertex/models/{model}:streamGenerateContent`
- `POST /v1/vertex/models/{model}:countTokens`

请求体支持两种写法：

- 站内简化 body：`system`、`messages`、`temperature`、`top_p`、`top_k`、`max_tokens`、`stop`、`tools`、`tool_choice`、`response_mime_type`、`response_schema`、`thinking`、`safety_settings`、`labels`、`metadata`
- 原生 Vertex / Gemini body：`contents`、`systemInstruction`、`generationConfig`、`toolConfig`、`cachedContent`、`tools`、`safetySettings`

几条重要规则一定要记住：

- 路径中的 `model` 是主入口；如果 body 里也写了 `model`，两者必须归一化后一致，否则会返回 `400 VERTEX_SIMPLIFIED_MODEL_CONFLICT`。
- 不要把“简化字段”和“原生字段族”混在同一个请求体里，否则会返回 `400 VERTEX_SIMPLIFIED_BODY_MIXED`。
- `safety_settings` 会作为简化别名映射到原生 `safetySettings`；`labels` / `metadata` 会原样透传；`response_schema` 会自动补成原生 structured output 字段。
- 当你传的是简化 body 时，后台会先转换成 Vertex 原生 `contents` / `systemInstruction` / `generationConfig` 再继续转发。

#### Python
```python focus=4-12,15-27
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"

response = requests.post(
    f"{base_url}/v1/vertex/models/gemini-2.5-pro:generateContent",
    headers={
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    },
    json={
        "system": "你是一个严谨的票据抽取助手。",
        "messages": [
            {
                "role": "user",
                "content": "请把下面 OCR 文本整理成三条结构化摘要。",
            }
        ],
        "temperature": 0.2,
        "response_mime_type": "application/json",
        "response_schema": {
            "type": "object",
            "properties": {
                "summary": {"type": "array"},
            },
        },
    },
    timeout=60,
)

print(response.status_code)
print(response.json())
```

#### JavaScript
```javascript focus=3-10,13-25
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";

const response = await fetch(`${baseUrl}/v1/vertex/models/gemini-2.5-pro:generateContent`, {
  method: "POST",
  headers: {
    Authorization: `Bearer ${apiKey}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    system: "你是一个严谨的票据抽取助手。",
    messages: [
      {
        role: "user",
        content: "请把下面 OCR 文本整理成三条结构化摘要。",
      },
    ],
    temperature: 0.2,
    response_mime_type: "application/json",
    response_schema: {
      type: "object",
      properties: {
        summary: { type: "array" },
      },
    },
  }),
});

console.log(response.status, await response.json());
```

#### REST
```bash focus=1-6,9-19
curl https://api.zyxai.de/v1/vertex/models/gemini-2.5-pro:generateContent \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "system": "你是一个严谨的票据抽取助手。",
    "messages": [
      {
        "role": "user",
        "content": "请把下面 OCR 文本整理成三条结构化摘要。"
      }
    ],
    "temperature": 0.2,
    "response_mime_type": "application/json",
    "response_schema": {
      "type": "object",
      "properties": {
        "summary": { "type": "array" }
      }
    }
  }'
```

### 简化 Batch 入口

推荐优先使用下面这两组简化路径：

- `POST /v1/vertex/batchPredictionJobs`
- `GET /v1/vertex/batchPredictionJobs`
- `GET /v1/vertex/batchPredictionJobs/{job}`
- `POST /v1/vertex/batchPredictionJobs/{job}:cancel`
- `DELETE /v1/vertex/batchPredictionJobs/{job}`
- `POST /vertex-batch/jobs`
- `GET /vertex-batch/jobs`
- `GET /vertex-batch/jobs/{job}`
- `POST /vertex-batch/jobs/{job}:cancel`
- `DELETE /vertex-batch/jobs/{job}`

创建批任务时支持两种输入模式，二选一：

- `requests[]`：推荐模式，直接内联一组站内请求；每个元素形如 `{ "key": "...", "request": { ... } }`
- `input_uri`：高级模式，你已经自己准备好了 Vertex 需要的 GCS JSONL 输入文件

当前行为如下：

- `requests[].request` 同时接受站内简化 body 和原生 Vertex / Gemini `generateContent` body。
- `model` 只需要写模型名，例如 `gemini-2.5-pro`；后台会自动补成 `publishers/google/models/...`。
- `input_uri` 模式会直接复用你准备好的 GCS JSONL；`requests[]` 模式则会先把每条请求统一归一化成原生 `generateContent` 再编译成 Vertex JSONL。
- 如果没有填写 `output_uri_prefix`，后台会自动使用程序托管的 GCS 暂存，并继续接现有 archive 结果回查链路。
- 如果本次请求需要托管 GCS，但当前系统没有可用 GCS Profile，会返回 `503 VERTEX_SIMPLIFIED_GCS_PROFILE_UNAVAILABLE`。
- 简化入口返回的 `name` 是公开名，例如 `batchPredictionJobs/job-1`。如果你走 `/vertex-batch/jobs/{job}` 短别名，后续路径里使用的是裸 job ID，例如 `job-1`。

#### Python
```python focus=4-10,13-29,32-36
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"
headers = {
    "Authorization": f"Bearer {api_key}",
    "Content-Type": "application/json",
}

create_job = requests.post(
    f"{base_url}/vertex-batch/jobs",
    headers=headers,
    json={
        "model": "gemini-2.5-pro",
        "display_name": "invoice-review",
        "requests": [
            {
                "key": "invoice-001",
                "request": {
                    "system": "你是一个票据审阅助手。",
                    "messages": [
                        {"role": "user", "content": "请提取金额、日期和供应商。"}
                    ],
                },
            }
        ],
    },
    timeout=120,
)

create_payload = create_job.json()
job_name = create_payload["name"].split("/", 1)[1]

job_status = requests.get(
    f"{base_url}/vertex-batch/jobs/{job_name}",
    headers=headers,
    timeout=60,
)

print(create_payload)
print(job_status.json())
```

#### JavaScript
```javascript focus=3-8,11-28,31-35
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";
const headers = {
  Authorization: `Bearer ${apiKey}`,
  "Content-Type": "application/json",
};

const createJob = await fetch(`${baseUrl}/vertex-batch/jobs`, {
  method: "POST",
  headers,
  body: JSON.stringify({
    model: "gemini-2.5-pro",
    display_name: "invoice-review",
    requests: [
      {
        key: "invoice-001",
        request: {
          system: "你是一个票据审阅助手。",
          messages: [
            { role: "user", content: "请提取金额、日期和供应商。" },
          ],
        },
      },
    ],
  }),
});

const createPayload = await createJob.json();
const jobName = createPayload.name.split("/", 2)[1];

const jobStatus = await fetch(`${baseUrl}/vertex-batch/jobs/${jobName}`, {
  headers,
});

console.log(createPayload);
console.log(await jobStatus.json());
```

#### REST
```bash focus=1-6,9-24
# 创建简化批任务
curl https://api.zyxai.de/vertex-batch/jobs \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gemini-2.5-pro",
    "display_name": "invoice-review",
    "requests": [
      {
        "key": "invoice-001",
        "request": {
          "system": "你是一个票据审阅助手。",
          "messages": [
            { "role": "user", "content": "请提取金额、日期和供应商。" }
          ]
        }
      }
    ]
  }'

# 返回 name 类似 batchPredictionJobs/job-1
# 短别名 follow-up 要使用裸 job ID：
curl https://api.zyxai.de/vertex-batch/jobs/job-1 \
  -H "Authorization: Bearer sk-你的站内Key"
```

如果你已经自己准备好了 GCS JSONL 输入文件，可以直接改成高级模式：

- `POST /v1/vertex/batchPredictionJobs`
- body 最少包含：`model` + `input_uri`
- 可选再传：`output_uri_prefix`、`display_name`、`labels`、`metadata`

### 严格 Vertex 兼容入口

下面这些路径仍然保留，适合“必须对齐官方 Vertex 路径”的旧客户端：

- `POST /v1/projects/{project}/locations/{location}/publishers/google/models/{model}:generateContent`
- `POST /v1/projects/{project}/locations/{location}/publishers/google/models/{model}:streamGenerateContent`
- `POST /v1/projects/{project}/locations/{location}/publishers/google/models/{model}:countTokens`
- `GET|POST|DELETE /v1/projects/{project}/locations/{location}/batchPredictionJobs...`

但要明确：

- 这里的 `project/location` 不是让下游随便填写任意 GCP 项目，而是必须与后台已经绑定好的 Vertex 账号作用域一致。
- 如果你不是在兼容旧 SDK，请直接使用前面的简化入口，不要把真实作用域暴露给下游调用方。
- 即使走严格兼容入口，认证仍然使用站内 API Key，调用地址仍然是你的网关地址。

#### Python
```python focus=4-7,10-18
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"
bound_project = "your-bound-project"
bound_location = "your-bound-location"

response = requests.post(
    f"{base_url}/v1/projects/{bound_project}/locations/{bound_location}/publishers/google/models/gemini-2.5-pro:countTokens",
    headers={
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    },
    json={
        "contents": [
            {"role": "user", "parts": [{"text": "请估算这段文本大约需要多少 token。"}]}
        ]
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript focus=3-6,9-18
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";
const boundProject = "your-bound-project";
const boundLocation = "your-bound-location";

const response = await fetch(
  `${baseUrl}/v1/projects/${boundProject}/locations/${boundLocation}/publishers/google/models/gemini-2.5-pro:countTokens`,
  {
    method: "POST",
    headers: {
      "Authorization": `Bearer ${apiKey}`,
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      contents: [
        { role: "user", parts: [{ text: "请估算这段文本大约需要多少 token。" }] },
      ],
    }),
  }
);

console.log(await response.json());
```

#### REST
```bash focus=1-4,7-12
curl https://api.zyxai.de/v1/projects/your-bound-project/locations/your-bound-location/publishers/google/models/gemini-2.5-pro:countTokens \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [{ "text": "请估算这段文本大约需要多少 token。" }]
      }
    ]
  }'
```

### Google Batch Archive

如果你创建批任务时走了“托管输出”链路，结果归档依然统一走现有只读入口：

- `GET /google/batch/archive/v1beta/batches/{batch_id}`
- `GET /google/batch/archive/v1beta/files/{file_id}:download`

它们的作用是：

- 查询归档后的批任务元信息
- 读取或下载归档后的输出文件
- 用公共名称回查结果，而不是让下游自己去暴露或维护真实上游存储位置

#### Python
```python focus=4-8,11
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"

response = requests.get(
    f"{base_url}/google/batch/archive/v1beta/batches/job-1",
    headers={"Authorization": f"Bearer {api_key}"},
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-8
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";

const response = await fetch(
  `${baseUrl}/google/batch/archive/v1beta/files/job-1-results:download`,
  {
    headers: {
      Authorization: `Bearer ${apiKey}`,
    },
  }
);

console.log(response.status);
```

#### REST
```bash focus=1-3
curl https://api.zyxai.de/google/batch/archive/v1beta/batches/job-1 \
  -H "Authorization: Bearer sk-你的站内Key"
```

### 认证、自动补全与常见错误

这页最关键的接入原则如下：

- 新接入默认使用 `Authorization: Bearer <站内API_KEY>`，这样最省心；Google 风格客户端也可以继续用 `x-goog-api-key`。
- 无论是 `/v1/vertex/...`、`/vertex-batch/jobs...`，还是严格 `/v1/projects/...` 与 `/google/batch/archive/...`，都不要依赖 `?key=...`。
- 简化入口会自动选择当前分组里可用的 Vertex 账号，并补全真实 `project/location` 再请求上游。
- 严格入口只保留给兼容需求；如果路径里的 `project/location` 与后台绑定作用域不一致，就找不到可用上游账号。
- 如果你在简化模型入口里混用了简化字段和原生字段，或路径模型与 body 模型冲突，会直接返回 `400`。
- 如果简化 Batch 需要托管 GCS 输入 / 输出，但系统当前没有可用 GCS Profile，会返回 `503 VERTEX_SIMPLIFIED_GCS_PROFILE_UNAVAILABLE`。

建议调试顺序：

1. 先用 `/v1/vertex/models/{model}:generateContent` 做一次最短文本请求。
2. 再用 `/vertex-batch/jobs` 或 `/v1/vertex/batchPredictionJobs` 创建简化批任务。
3. 批任务完成后，再走 `/google/batch/archive/...` 做结果回查与下载。
4. 只有在旧 SDK 必须要求官方路径形态时，才切到严格 `/v1/projects/...`。
