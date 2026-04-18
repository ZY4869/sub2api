## antigravity
> 本页用于显式只走 Antigravity 平台的接入。`/antigravity` 前缀不是装饰，它意味着路由已经强制绑定到 Antigravity，不再进行普通混合调度。

### 协议定位与强制前缀

Antigravity 相关路由分成三组：

- `GET /antigravity/models`
- `/antigravity/v1/...`：Anthropic 风格入口
- `/antigravity/v1beta/...`：Gemini 风格入口

与普通公共路径相比，`/antigravity/...` 的核心区别是：

- 路由中间件会强制平台为 Antigravity。
- 这适合你明知要落到 Antigravity、并且不希望被其他平台接管的场景。
- 也正因为它是强制平台，所以某些“在公共混合路径里可兼容翻译”的动作，在这里可能会被明确拒绝。

### Anthropic 风格入口

Anthropic 风格路径包括：

- `POST /antigravity/v1/messages`
- `POST /antigravity/v1/messages/count_tokens`
- `GET /antigravity/v1/models`
- `GET /antigravity/v1/usage`

需要准确理解两点：

- `messages` 是 Antigravity 的主文本入口之一，可以按 Anthropic 风格发送。
- `messages/count_tokens` 虽然注册了路由，但当前能力矩阵里对 Antigravity 平台并不视为成功面，不能当稳定能力依赖。

#### Python
```python focus=1-12
import requests

response = requests.post(
    "https://api.zyxai.de/antigravity/v1/messages",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
        "anthropic-version": "2023-06-01",
    },
    json={
        "model": "claude-sonnet-4-20250514",
        "max_tokens": 256,
        "messages": [{"role": "user", "content": "请说明 Antigravity 前缀的作用。"}],
    },
    timeout=60,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-10
const response = await fetch("https://api.zyxai.de/antigravity/v1/messages", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
    "anthropic-version": "2023-06-01",
  },
  body: JSON.stringify({
    model: "claude-sonnet-4-20250514",
    max_tokens: 256,
    messages: [{ role: "user", content: "请说明 Antigravity 前缀的作用。" }],
  }),
});

console.log(await response.json());
```

#### REST
```bash focus=1-6
curl https://api.zyxai.de/antigravity/v1/usage \
  -H "Authorization: Bearer sk-你的站内Key"
```

### Gemini 风格入口

Gemini 风格路径包括：

- `GET /antigravity/v1beta/models`
- `GET /antigravity/v1beta/models/:model`
- `POST /antigravity/v1beta/models/*modelAction`

当前行为可以概括为：

- `generateContent`、`streamGenerateContent`、`countTokens` 可以走兼容翻译。
- `batchGenerateContent` 当前能力矩阵明确拒绝。
- 这条前缀非常适合“我要让 Gemini 风格客户端只打到 Antigravity”的场景。

#### Python
```python focus=1-12
import requests

response = requests.post(
    "https://api.zyxai.de/antigravity/v1beta/models/gemini-2.5-pro:generateContent",
    headers={
        "x-goog-api-key": "sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "contents": [
            {
                "role": "user",
                "parts": [{"text": "请解释 Antigravity 的 Gemini 风格入口。"}],
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
  "https://api.zyxai.de/antigravity/v1beta/models/gemini-2.5-flash:countTokens",
  {
    method: "POST",
    headers: {
      "x-goog-api-key": "sk-你的站内Key",
      "Content-Type": "application/json",
    },
    body: JSON.stringify({
      contents: [{ role: "user", parts: [{ text: "请解释 Antigravity 的 Gemini 风格入口。" }] }],
    }),
  }
);

console.log(await response.json());
```

#### REST
```bash focus=1-6
curl https://api.zyxai.de/antigravity/v1beta/models \
  -H "x-goog-api-key: sk-你的站内Key"
```

### 已注册但不建议依赖的动作

这一节非常重要，因为它决定了你能否安全长期维护：

- `/antigravity/v1/messages/count_tokens`：当前不应视为稳定可用能力。
- `/antigravity/v1beta/models/{model}:batchGenerateContent`：当前能力矩阵明确拒绝。

如果你的调用链确实需要这些动作，建议回到公共协议页重新评估是否应该改用 Anthropic 原生或 Gemini 原生分组，而不是在 Antigravity 强制前缀下硬做兼容。

#### Python
```python focus=1-12
# 当前不建议依赖 antigravity /v1/messages/count_tokens。
# 如果你的业务必须使用 token 预估，请改为 Anthropic 原生分组测试，
# 不要把这个前缀当成与 Anthropic 完全等价的稳定能力面。
```

#### JavaScript
```javascript focus=1-10
// 当前不建议依赖 /antigravity/v1beta/models/{model}:batchGenerateContent。
// 如果你需要稳定批任务，请优先回到 Gemini 原生分组。
```

#### REST
```bash focus=1-6
# 当前协议页下，这些动作属于“路由存在，但能力矩阵不建议依赖”的范围。
# 排障时请优先检查平台能力，而不是继续重试。
```

### 接入建议

- 只有在你明确要“固定走 Antigravity”时才使用该前缀。
- 不要把公共混合入口与 `/antigravity/...` 混在同一个客户端实例里。
- 对 token 预估、批任务这类能力收窄动作，要单独验证，不要因为路径能访问就假设功能完整。
