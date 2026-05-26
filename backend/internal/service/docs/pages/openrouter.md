## openrouter
> 本页面向需要显式走 OpenRouter 官方平台的 OpenAI-compatible 客户端。OpenRouter 账号和分组是独立一级平台，不会与普通 OpenAI 账号或协议网关账号混合。

### 支持范围

- 默认上游 Base URL：`https://openrouter.ai/api/v1`
- 用户侧公开入口：
  - `GET /openrouter/v1/models`
  - `POST /openrouter/v1/chat/completions`
- OpenAI-compatible 公共入口：
  - `GET /v1/models`
  - `POST /v1/chat/completions`
- 账号凭据字段：
  - `credentials.api_key`：OpenRouter API Key
  - `credentials.http_referer`：可选，转发为 `HTTP-Referer`
  - `credentials.openrouter_title`：可选，转发为 `X-OpenRouter-Title`

当前只承诺 OpenAI-compatible 文本聊天与模型列表。`responses`、图片、视频、Embeddings、Anthropic 原生和 Gemini 原生入口不属于本页能力范围。

### 分组与调度

- `openrouter` 分组只匹配 `openrouter` 账号。
- 官方 `openai` 账号不会进入 `openrouter` 分组，`openrouter` 账号也不会进入 `openai` 分组。
- `/openrouter/v1/...` 会强制选择 OpenRouter 平台；公共 `/v1/chat/completions` 仍按 API Key 绑定分组和模型策略调度。
- 模型列表与模型投影只返回公开模型 ID，不暴露内部 `target_model_id` 或等价路由元数据。

### Chat Completions

#### Python

```python
from openai import OpenAI

client = OpenAI(
    base_url="https://api.zyxai.de/openrouter/v1",
    api_key="sk-...",
)

response = client.chat.completions.create(
    model="openrouter/auto",
    messages=[{"role": "user", "content": "Say hello from OpenRouter"}],
)
print(response.choices[0].message.content)
```

#### JavaScript

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "https://api.zyxai.de/openrouter/v1",
  apiKey: "sk-...",
});

const response = await client.chat.completions.create({
  model: "openrouter/auto",
  messages: [{ role: "user", content: "Say hello from OpenRouter" }],
});

console.log(response.choices[0].message.content);
```

#### REST

```bash
curl https://api.zyxai.de/openrouter/v1/chat/completions \
  -H "Authorization: Bearer sk-..." \
  -H "Content-Type: application/json" \
  -d '{
    "model": "openrouter/auto",
    "messages": [
      { "role": "user", "content": "Say hello from OpenRouter" }
    ]
  }'
```

### Models

#### Python

```python
from openai import OpenAI

client = OpenAI(
    base_url="https://api.zyxai.de/openrouter/v1",
    api_key="sk-...",
)

for model in client.models.list().data:
    print(model.id)
```

#### JavaScript

```javascript
import OpenAI from "openai";

const client = new OpenAI({
  baseURL: "https://api.zyxai.de/openrouter/v1",
  apiKey: "sk-...",
});

const models = await client.models.list();
for (const model of models.data) {
  console.log(model.id);
}
```

#### REST

```bash
curl https://api.zyxai.de/openrouter/v1/models \
  -H "Authorization: Bearer sk-..."
```

### 注意事项

- OpenRouter 官方可选的 `HTTP-Referer` 与 `X-OpenRouter-Title` 应配置在后台账号凭据里，客户端请求不需要也不建议自行携带这些归因头。
- OpenRouter 后台模型导入走 `GET https://openrouter.ai/api/v1/models`，并会带上同一组可选归因头。
- 如需原生 OpenAI Responses、OpenAI 图片或 WebSocket，请使用 `openai-native` 页的能力范围；如需 OpenAI-compatible 历史聊天入口，请参考 `openai` 页。
