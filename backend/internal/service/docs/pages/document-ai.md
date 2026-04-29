## document-ai
> 本页聚焦百度智能文档代理接口。这里的重点不是聊天协议兼容，而是分组绑定、直连解析与异步任务两条独立工作流。

### 协议定位与权限边界

网关内置了一组百度智能文档代理接口，统一挂在 `/document-ai/v1/...` 下。

它和普通聊天 / 补全接口最大的区别是“分组绑定”要求更严格：

- 只有绑定到 `baidu_document_ai` 分组的站内 API Key 才能访问。
- 若当前 API Key 没有绑定任何百度智能文档分组，会返回 `403 document_ai_forbidden`。
- 对外仍然使用站内 API Key 认证；百度侧 `async_bearer_token` / `direct_token` 只保留在账号配置里，不会暴露给调用方。

当前公开路径如下：

| 路径 | 方法 | 用途 | 模式 |
| --- | --- | --- | --- |
| `/document-ai/v1/models` | `GET` | 列出当前网关支持的文档模型与能力 | 公共能力枚举 |
| `/document-ai/v1/models/{model}:parse` | `POST` | 直接同步解析文件 | `direct` |
| `/document-ai/v1/jobs` | `POST` | 创建异步解析任务 | `async` |
| `/document-ai/v1/jobs/:job_id` | `GET` | 查询异步任务状态 | `async` |
| `/document-ai/v1/jobs/:job_id/result` | `GET` | 拉取异步任务结果 | `async` |

所以百度智能文档并不只有异步解析。
如果你需要同步直出结果，请使用 `POST /document-ai/v1/models/{model}:parse`，它就是 `direct` 同步解析入口。

### 账号级模型限制与映射

百度智能文档账号现在也接入了统一的账号模型策略链路，和其它一级平台保持一致：

- 后台创建 / 编辑账号时，模型限制以 `extra.model_scope_v2.entries[]` 为唯一事实源。
- 如果账号配置了白名单或别名映射，`GET /document-ai/v1/models` 只会返回这些被允许的 display model ID。
- 如果某个 display model 被映射到不同的真实 target model，对外依然只暴露 display model ID；真实 target 只用于内部路由。
- 空选择或空的 `model_scope_v2.entries` 不表示“禁用全部模型”，而是表示“不限制”，网关会继续回退到默认百度文档模型库。
- 当同一个 `baidu_document_ai` 分组下配置了多个可调度账号时，`GET /document-ai/v1/models` 返回这些账号允许的 display model ID **并集**。
- 如果请求的 display model 不在该分组任何可用账号的 `model_scope_v2.entries[]` 允许范围内，接口会返回 `403 document_ai_model_forbidden`。

### 直连解析与模型模式差异

`direct` 模式适合低延迟、前台直出结果的场景。它支持 `multipart file` 与 JSON `file_base64`，并且必须携带 `file_type=image|pdf`。

模型与模式约束如下：

- `pp-ocrv5-server`、`paddleocr-vl-1.5` 同时支持 `async` 与 `direct`。
- `pp-structurev3`、`paddleocr-vl` 目前只支持 `async`。
- 如果你需要同步拿到结果，优先选择同时支持 `direct` 的模型，并在请求前就区分 `image` / `pdf` 文件类型。

#### Python
```python focus=6-12,15-19
import base64
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"
file_base64 = base64.b64encode(b"fake-image-bytes").decode("utf-8")

response = requests.post(
    f"{base_url}/document-ai/v1/models/pp-ocrv5-server:parse",
    headers={
        "Authorization": f"Bearer {api_key}",
        "Content-Type": "application/json",
    },
    json={
        "file_type": "image",
        "file_base64": file_base64,
        "options": {
            "useDocUnwarping": True,
            "useFormulaRecognition": True,
        },
    },
    timeout=60,
)
print(response.json())
```

#### JavaScript
```javascript focus=4-10,13-18
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";

const response = await fetch(`${baseUrl}/document-ai/v1/models/pp-ocrv5-server:parse`, {
  method: "POST",
  headers: {
    Authorization: `Bearer ${apiKey}`,
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    file_type: "image",
    file_base64: Buffer.from("fake-image-bytes").toString("base64"),
    options: {
      useDocUnwarping: true,
      useFormulaRecognition: true,
    },
  }),
});
console.log(await response.json());
```

#### REST
```bash focus=2-8
curl https://api.zyxai.de/document-ai/v1/models/pp-ocrv5-server:parse \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "file_type": "image",
    "file_base64": "ZmFrZS1pbWFnZS1ieXRlcw==",
    "options": {
      "useDocUnwarping": true,
      "useFormulaRecognition": true
    }
  }'
```

### 异步任务工作流

如果你只需要先做健康检查，建议先调 `GET /document-ai/v1/models` 确认当前 Key 已经绑定到正确分组，再创建异步任务。

`async` 模式适合大文件、长耗时 PDF、批量 OCR 或需要轮询状态的场景。它支持 `multipart file` 与 JSON `file_url` 两种输入方式。

#### Python
```python focus=8,12-18,21-26
import requests

base_url = "https://api.zyxai.de"
api_key = "sk-你的站内Key"

headers = {
    "Authorization": f"Bearer {api_key}",
    "Content-Type": "application/json",
}

models = requests.get(f"{base_url}/document-ai/v1/models", headers=headers, timeout=30)
print(models.json())

job = requests.post(
    f"{base_url}/document-ai/v1/jobs",
    headers=headers,
    json={
        "model": "pp-ocrv5-server",
        "file_url": "https://example.com/sample.pdf",
        "options": {
            "useChartRecognition": True,
            "useTableRecognition": True,
        },
    },
    timeout=60,
)
job_id = job.json()["data"]["job_id"]

result = requests.get(
    f"{base_url}/document-ai/v1/jobs/{job_id}/result",
    headers=headers,
    timeout=60,
)
print(result.json())
```

#### JavaScript
```javascript focus=8,12-19,22-26
const baseUrl = "https://api.zyxai.de";
const apiKey = "sk-你的站内Key";

const headers = {
  Authorization: `Bearer ${apiKey}`,
  "Content-Type": "application/json",
};

const models = await fetch(`${baseUrl}/document-ai/v1/models`, {
  headers,
});
console.log(await models.json());

const createJob = await fetch(`${baseUrl}/document-ai/v1/jobs`, {
  method: "POST",
  headers,
  body: JSON.stringify({
    model: "pp-structurev3",
    file_url: "https://example.com/sample.pdf",
    options: {
      useLayoutDetection: true,
      useSealRecognition: true,
    },
  }),
});
const createJobPayload = await createJob.json();
const jobId = createJobPayload.data.job_id;

const jobResult = await fetch(`${baseUrl}/document-ai/v1/jobs/${jobId}/result`, {
  headers,
});
console.log(await jobResult.json());
```

#### REST
```bash focus=2,6-12,15-16
# 1) 查看可用模型
curl https://api.zyxai.de/document-ai/v1/models \
  -H "Authorization: Bearer sk-你的站内Key"

# 2) 创建异步任务
curl https://api.zyxai.de/document-ai/v1/jobs \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "pp-ocrv5-server",
    "file_url": "https://example.com/sample.pdf",
    "options": {
      "useChartRecognition": true,
      "useTableRecognition": true
    }
  }'

# 3) 拉取异步结果
curl https://api.zyxai.de/document-ai/v1/jobs/job_xxx/result \
  -H "Authorization: Bearer sk-你的站内Key"
```

### 接入建议

- 先确认分组是否真的是 `baidu_document_ai`，再调接口；否则会一直卡在 `403`。
- 先定模式再定模型：需要一次出结果就选 `direct`，需要轮询就选 `async`。
- 先跑 `GET /document-ai/v1/models`，再填业务参数；这样最容易判断是权限问题还是模型 / 模式不匹配。
- 如果你在后台同时配置了 `async_bearer_token` 和 `direct_token`，调用方仍然只需要站内 API Key，不需要知道百度侧凭证细节。
