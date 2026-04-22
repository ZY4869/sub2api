## openai
> 本页面向旧版 OpenAI SDK 和历史兼容客户端。重点是 `chat/completions`、历史别名路径，以及从兼容入口迁移到 OpenAI 原生入口的建议。

### 协议定位与适用场景

当你满足下面任一条件时，优先阅读本页：

- 现有工程仍然依赖 `chat/completions`
- 你在做旧版 OpenAI 客户端的平滑兼容
- 第三方工具暂时不支持 `responses`

当前程序对 OpenAI 兼容入口的理解是：

- `chat/completions` 的定位是兼容旧生态，而不是主推新设计。
- `/chat/completions` 这类无 `/v1` 别名仍然保留，但不建议新项目继续使用。
- 真正的原生文本主入口已经独立到 `openai-native` 页，不再与兼容入口混写。

### chat/completions 主入口

`chat/completions` 的核心规则如下：

- `POST /v1/chat/completions` 对 OpenAI / Copilot / Grok 平台可直通。
- 对于仍然使用 `messages` 数组的旧应用，这是最省心的入口。
- 如果是全新项目，仍然建议优先改用 `responses`。
- 如果上游账号是 OpenAI Pro，运行时额度会拆成两侧：`gpt-5.3-codex-spark*` 只占用 `Spark` 侧，其它 OpenAI 模型统一占用 `普通` 侧；一侧冷却不会连带阻断另一侧。
- 当 `POST /v1/chat/completions` 的目标模型在所有可路由账号上都只因为对应额度侧冷却而不可服务时，接口会返回 `429 rate_limit_error`；如果是其它 no-account、上游失败或平台不支持，仍然保持现有 `503` / `502` / `400` 语义。
- `/v1/models`、`/v1beta/models` 和对应 detail 读路径会按当前运行时可服务性临时隐藏受限侧模型；这类临时隐藏不会改写账号白名单，也不会把真实调用改成 `404`。
- 管理后台单账号测试 `POST /api/v1/admin/accounts/:id/test` 会在发起上游请求前先做本地预检；命中受限侧时直接返回 `400`，提示 `Spark 冷却中`、`普通额度冷却中` 或整号冷却。

选择 `chat/completions` 的典型场景：

- 现有代码或第三方工具不支持 `responses`
- 你明确依赖旧版 OpenAI SDK 或旧参数结构
- 你在做快速兼容，而不是长期演进

#### Python
```python focus=2-12,15
import requests

response = requests.post(
    "https://api.zyxai.de/v1/chat/completions",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "model": "gpt-4.1",
        "messages": [
            {"role": "system", "content": "你是一个简洁的接口说明助手。"},
            {"role": "user", "content": "解释什么时候还应该使用 chat/completions。"},
        ],
        "stream": False,
    },
    timeout=60,
)

print(response.json())
if response.status_code == 429:
    print("当前模型可能因为对应的 OpenAI Pro 额度侧冷却而暂时不可用。")
```

#### JavaScript
```javascript focus=1-10
const response = await fetch("https://api.zyxai.de/v1/chat/completions", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "gpt-4.1",
    messages: [{ role: "user", content: "解释什么时候还应该使用 chat/completions。" }],
  }),
});

console.log(await response.json());
if (response.status === 429) {
  console.log("当前模型可能因为对应的 OpenAI Pro 额度侧冷却而暂时不可用。");
}
```

#### REST
```bash focus=1-6
curl https://api.zyxai.de/v1/chat/completions \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4.1",
    "messages": [
      { "role": "user", "content": "解释什么时候还应该使用 chat/completions。" }
    ]
  }'
# 如果命中的模型在所有可路由 OpenAI Pro 账号上都只因对应额度侧冷却而不可服务，
# 这里会返回 429 rate_limit_error，而不是 404。
```

### 历史别名与兼容迁移

兼容页还需要记住两个事实：

- `POST /chat/completions` 这类历史别名仍可访问，但只适合旧系统托底。
- 新增功能请优先围绕 `/v1/chat/completions` 或直接迁移到 `openai-native` 页的 `/v1/responses`。

推荐迁移顺序：

1. 先把无 `/v1` 的历史别名迁回 `/v1/chat/completions`。
2. 再评估是否可以切换到 `openai-native` 页的 `responses`。
3. 只有确认第三方工具链仍受限时，才继续长期保留 `chat/completions`。

#### Python
```python focus=1-6
legacy_url = "https://api.zyxai.de/chat/completions"
recommended_url = "https://api.zyxai.de/v1/chat/completions"
native_url = "https://api.zyxai.de/v1/responses"

print("legacy:", legacy_url)
print("recommended:", recommended_url)
print("next step:", native_url)
```

#### JavaScript
```javascript focus=1-6
const urls = {
  legacy: "https://api.zyxai.de/chat/completions",
  compatible: "https://api.zyxai.de/v1/chat/completions",
  native: "https://api.zyxai.de/v1/responses",
};

console.table(urls);
```

#### REST
```bash focus=1-3
curl https://api.zyxai.de/chat/completions
# 建议先迁到 /v1/chat/completions
# 新项目再进一步迁到 /v1/responses
```

### 常见兼容坑

- 兼容入口能跑通，不代表它仍然是最佳长期方案。
- 媒体能力已经在 `grok` 页单独展开，不要继续把图像 / 视频示例堆在兼容页。
- 如果你同时维护新旧两套客户端，建议把新接入统一放到 `openai-native`，旧客户端单独留在本页。
- 对需要长期维护的系统，应该优先围绕 `responses` 设计，而不是继续叠加 `chat/completions` 特性债务。
- 如果是 OpenAI Pro，看到 `429 rate_limit_error` 时要额外考虑“相关额度侧正在冷却”；这时同账号上的另一侧模型通常仍然可以继续调用。
