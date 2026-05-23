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

- `POST /v1/chat/completions` 对 OpenAI / Grok 平台可直通。
- 对于仍然使用 `messages` 数组的旧应用，这是最省心的入口。
- 如果是全新项目，仍然建议优先改用 `responses`。
- 如果上游账号是 OpenAI Pro，运行时额度会拆成两侧：`gpt-5.3-codex-spark*` 只占用 `Spark` 侧，其它 OpenAI 模型统一占用 `普通` 侧；一侧冷却不会连带阻断另一侧。
- 当 `POST /v1/chat/completions` 的目标模型在所有可路由账号上都只因为对应额度侧冷却而不可服务时，接口会返回 `429 rate_limit_error`；如果是其它 no-account、上游失败或平台不支持，仍然保持现有 `503` / `502` / `400` 语义。
- `/v1/models`、`/v1beta/models` 和对应 detail 读路径会按当前运行时可服务性临时隐藏受限侧模型；这类临时隐藏不会改写账号白名单，也不会把真实调用改成 `404`。
- 管理后台单账号测试 `POST /api/v1/admin/accounts/:id/test` 会在发起上游请求前先做本地预检；命中受限侧时直接返回 `400`，提示 `Spark 冷却中`、`普通额度冷却中` 或整号冷却。
- 如果账号上游 `base_url` 不符合当前出站安全策略，保存阶段就会被拒绝，返回 `400 ACCOUNT_INVALID_BASE_URL`；保存成功后不会自动探测模型，需由管理员手动执行 Probe/Test。
- 管理端单账号测试、手动 Probe 与运行态转发都不会跟随上游 `3xx`；命中重定向时会返回受控错误 `502 UPSTREAM_REDIRECT_NOT_ALLOWED`。
- `service_tier`（priority/fast/flex）可能会被管理员策略过滤或阻断；默认 `priority/fast` 被过滤、`flex` 放行。命中阻断时返回 `403 forbidden_error`，错误码 `openai_fast_policy_blocked`。

Thinking / reasoning 强度补充规则：

- 顶层 `effortLevel` 现在只保留为 Claude 定向补位字段，不再作为 OpenAI / Responses 的公开承诺能力。
- OpenAI 兼容入口只认 `reasoning.effort` / `reasoning_effort`；请求详情与使用记录仍会展示 `reasoning_effort_raw` / `reasoning_effort_effective`。
- 兼容入口转 Responses 上游时，如果目标是 reasoning 模型，网关会自动剔除上游不接受的 `temperature` / `top_p`；非 reasoning 模型保持原样。
- OpenAI 入口当前会接受并记录顶层 `effortLevel` 与 `model[1m]`，并把 `[1m]` 先剥离后再参与模型映射、路由和上游请求构造。
- 但在当前运行时矩阵下，OpenAI 文本入口不会新增直达 Anthropic runtime 的跨平台转发能力；因此这两类字段默认只会体现为“用户请求过”，不承诺在该入口直接转成 Claude 上游生效。
- 对纯 OpenAI / DeepSeek 等当前可运行目标来说，请求详情与使用记录里通常会看到 `million_context_requested=true`、`million_context_effective=false`；这属于当前能力边界内的正常行为，而不是报错。
- 即使请求里写了 `deepseek-v4-pro[1m]` 或顶层 `effortLevel=max`，它们也不会让 `/v1/models`、策略投影或公开模型目录出现新的 `[1m]` 变体。

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

#### Python
```python focus=1-14
import requests

response = requests.post(
    "https://api.zyxai.de/v1/chat/completions",
    headers={
        "Authorization": "Bearer sk-你的站内Key",
        "Content-Type": "application/json",
    },
    json={
        "model": "deepseek-v4-pro[1m]",
        "effortLevel": "max",
        "messages": [{"role": "user", "content": "当前入口会记录 1M 与 Claude 定向 effort 请求意图，但不会因为它们新增直达 Claude runtime 的转发能力。"}],
    },
    timeout=60,
)

print(response.json())
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

### 图片双协议补充

兼容页还需要额外记住三条图片规则：

- `chat/completions` 仍然只定位为文本兼容入口；如果你的请求意图是生图或编辑图，请改走 `openai-native` 页里的 `/v1/responses` 或 `/v1/images/*`。
- OpenAI 图片链路现在统一受 `image_protocol_mode` 控制：账号 / Protocol Gateway 可以设置默认值，OpenAI 分组还能用 `inherit | native | compat` 再做覆盖；不再根据 `gpt-image-2` 模型名猜链路。
- 当当前模式是 `compat` 时，`/v1/images/generations`、`/v1/images/edits` 会桥接到 compat 图片执行链；如果当前账号没有 compat 图片权限，会返回 `403 forbidden_error`，错误码 `image_compat_not_allowed`。
- Compat 图片桥接现在已经补齐 `stream=true`：`/v1/images/generations` 会输出 `image_generation.partial_image` / `image_generation.completed`，`/v1/images/edits` 会输出 `image_edit.partial_image` / `image_edit.completed`。
- `/v1/images/generations` 与 `/v1/images/edits` 的 native 路径会透传支持的 `n`；compat 路径不支持多图时会在上游前返回 `400 image_n_not_supported`，不会静默丢弃 `n`。
- Compat 执行链内部固定把目标图片模型归一到 `gpt-image-2`，并在真正请求上游前按能力矩阵校验 `size`、`background`、`output_format`、`output_compression`、`partial_images`、`mask` 与多图输入；`input_fidelity` 仍保留在网关内部 trace，但不会继续向 `gpt-image-2` 上游透传。
- `/v1/responses` 的 `image_generation` tool 只有在显式 tool `model` 解析为 OpenAI GPT image profile 时才允许进入 compat 归一路径；Grok / Gemini 等非 OpenAI 图片模型不会被静默转成 Codex 生图，会返回 `400 image_tool_model_provider_unsupported`。
- 对 GPT image profile（版本化 `gpt-image-*`，例如 `gpt-image-1.5`、`gpt-image-2`，以及 `chatgpt-image-latest`）来说，native / compat 两条链都会放开 `stream`、多图、`mask`、`background=transparent` 与最大边 `3840px` 的自定义尺寸；未知或旧模型保持保守拒绝。

### 常见兼容坑

- 兼容入口能跑通，不代表它仍然是最佳长期方案。
- 媒体能力已经在 `grok` 页单独展开，不要继续把图像 / 视频示例堆在兼容页。
- 如果你同时维护新旧两套客户端，建议把新接入统一放到 `openai-native`，旧客户端单独留在本页。
- 对需要长期维护的系统，应该优先围绕 `responses` 设计，而不是继续叠加 `chat/completions` 特性债务。
- 如果是 OpenAI Pro，看到 `429 rate_limit_error` 时要额外考虑“相关额度侧正在冷却”；这时同账号上的另一侧模型通常仍然可以继续调用。
