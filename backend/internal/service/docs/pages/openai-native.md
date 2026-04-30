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
- `service_tier`（priority/fast/flex）可能会被管理员策略过滤或阻断；详见下文 “OpenAI Fast/Flex Policy（service_tier）”。

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

### OpenAI Fast/Flex Policy（service_tier）

部分 OpenAI 客户端会在请求体里携带 `service_tier`（例如 `priority` / `fast` / `flex`）以请求不同服务层级。网关会在转发前按管理员配置执行策略：

- 默认策略：`priority` / `fast` 会被 **过滤**（从请求体移除），`flex` 默认 **放行**。
- 归一化规则：`fast` 会按 `priority` 处理（因此两者的策略结果等价）。

策略动作：

- `pass`：透传 `service_tier`。
- `filter`：删除 `service_tier` 后再转发（上游将按默认层级处理）。
- `block`：拒绝请求并返回 `403 forbidden_error`，错误码 `openai_fast_policy_blocked`。

覆盖范围：

- HTTP：`POST /v1/responses`、`POST /v1/chat/completions` 等 OpenAI JSON 请求（包含 passthrough 模式）。
- 长连接 / WebSocket：只对 `response.create` 入站 payload 生效；其它 WS 事件不会被改写。

管理员配置入口：`GET/PUT /api/v1/admin/settings` 字段 `openai_fast_policy_settings`。

### OpenAI / Codex 生图

OpenAI 侧现在建议明确区分两类能力：

- 图片链路现在统一由显式 `image_protocol_mode` 决定，不再根据 `model=gpt-image-2` 去猜“原生图片请求”还是 “compat / Codex 图片请求”。
- 生效优先级固定为：OpenAI 分组强制模式（`inherit | native | compat`）> 账号模式 > 默认策略；其中 OpenAI OAuth `free` 默认 `native` 且 `compat` 被禁用，其它已识别付费计划默认 `compat`。
- `gpt-image-2` 这类原生图片模型，走 `/v1/images/generations` 或 `/v1/images/edits`。
- `gpt-5.4`、`gpt-5.4-mini`、`gpt-5.4-pro` 这类主模型，如果要生图，主路径是 `/v1/responses` + `tools:[{type:"image_generation"}]`；顶层 `model` 继续保持主模型本身，不需要改成 `gpt-image-2`。
- `/v1/images/generations`、`/v1/images/edits` 在 `native` 下直连原生 Images 链路，在 `compat` 下会桥接到兼容执行链；Compat 链路内部统一使用 `gpt-image-2` 作为目标图片模型。
- `/v1/images/generations`、`/v1/images/edits` 会继续按请求里的 `model` 解析 provider；如果模型属于 Grok / Gemini 等非 OpenAI 图片模型，会走对应 provider 的图片链路，不会被改写成 Codex / `gpt-image-2`。
- 生图专用 Key 的 `/v1/models` 只枚举原生生图模型；开启按数量计费时，图片请求按 `输出张数 × image_count_weights[分辨率档位]` 预占与结算，默认 `1K=1、2K=1、4K=2`，`auto` 或未识别尺寸按 `2K` 计。
- `/v1/responses` 的 `image_generation` tool 如果显式填写 `model`，网关会先确认它能解析为 OpenAI GPT image profile（含账号模型映射后的目标模型）；只有 OpenAI 图片模型在 compat 模式下才会被内部归一到 `gpt-image-2`。非 OpenAI tool 模型会返回 `400 invalid_request_error`，错误码 `image_tool_model_provider_unsupported`，请改用 provider 专用图片端点。
- 网关现在会解析 `$imagegen ...` 兼容前缀，并自动改写成标准 Responses tool 请求；这是“网关兼容扩展”，不是 OpenAI 官方标准字段。
- 兼容扩展只在命中 `$imagegen` 时生效：JSON 下可额外携带 `image_generation`、`reference_images`；`multipart/form-data` 下可额外使用 `reference_image`、`reference_image_url`。
- `multipart/form-data` 仍然只作为网关扩展，不承诺官方 SDK 兼容；其中 `/v1/images/*` 的 native / compat 两条链路都支持 `stream=true`，但 `/v1/responses` 的 multipart `$imagegen` 扩展当前仍要求 `stream=false`。
- 直传参考图只接受 JPEG / PNG / WebP，最多 4 张；网关会把最长边归一到 `2048px` 以内，再转成标准 `input_image` data URI。URL / data URI 参考图只做格式校验，不做服务端重采样。
- `size` 推荐直接使用明确 `"WIDTHxHEIGHT"`（例如 `"1536x1024"`）。此外网关也接受 shorthand：`"2K 16:9"` / `"16:9 2K"`，以及分字段 `image_size="2K"` + `aspect_ratio="16:9"`（`aspect_ratio` 支持 `W:H` / `W/H`）。`image_size` 默认 `2K`，可选 `1K/2K/4K`（映射最大边 `1024/2048/3840`），`aspect_ratio` 默认 `1:1`；网关会在转发上游前统一换算并写回 `size`（例：`2K 16:9` → `2048x1152`），并剔除扩展字段，避免 OpenAI 上游因未知字段或非法 size 直接 400。
- `/v1/images/edits` 的 `multipart/form-data` 支持 `image[]` 作为输入图片字段（等价于 `image` / `images` / `images[]`）。
- `n`：`/v1/responses` 的 `image_generation` tool 当前不支持 `n>1`；网关会兼容 `n=1` 但不透传，`n>1` 直接返回 `400 invalid_request_error`，错误码 `image_n_not_supported`。
- 原生与 compat 两条图片链路都会统一归一化 `generate | edit`、`images[]`、`mask`、`size`、`quality`、`background`、`output_format`、`output_compression`、`partial_images`、`moderation`、`input_fidelity`；其中 `input_fidelity` 会保留在网关内部 DTO 和 trace，但当前不会继续向 compat 内部固定目标 `gpt-image-2` 的上游 payload 透传。
- 图片能力矩阵现在以单一 GPT image profile 为准：版本化 `gpt-image-*`（例如 `gpt-image-1.5`、`gpt-image-2`）和 `chatgpt-image-latest`，无论 native 还是 compat，都会放开 `generate`、`edit`、`stream`、多图、`mask`、`background=transparent`，以及最大边 `3840px` 的自定义尺寸；未知或旧模型保持保守拒绝路径。
- 能力校验顺序固定在上游请求前完成：`operation` -> `stream/partial_images` -> `mask/multi-image` -> `background/output_format/output_compression` -> `size/custom-resolution`。命中的能力档会写入 `image_capability_profile`，便于观察 `transparent` / `4K` 放量。

标准 Responses 写法与兼容写法都最终会归一成同一类 `image_generation` tool 请求。

#### Python
```python focus=3-18
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
        "tools": [{"type": "image_generation", "size": "1536x1024"}],
        "tool_choice": {"type": "image_generation"},
    },
    timeout=120,
)

print(response.json())
```

#### JavaScript
```javascript focus=1-16
const response = await fetch("https://api.zyxai.de/v1/responses", {
  method: "POST",
  headers: {
    Authorization: "Bearer sk-你的站内Key",
    "Content-Type": "application/json",
  },
  body: JSON.stringify({
    model: "gpt-5.4-mini",
    input: "$imagegen 生成一张带晨雾的山间木屋海报",
    image_generation: { size: "1536x1024", background: "opaque" },
    reference_images: [
      { image_url: "https://example.com/reference.png" },
    ],
  }),
});

console.log(await response.json());
```

#### REST
```bash focus=1-8
curl https://api.zyxai.de/v1/responses \
  -H "Authorization: Bearer sk-你的站内Key" \
  -F 'model=gpt-5.4-mini' \
  -F 'input=$imagegen 生成一张适合产品页首屏的海报图。' \
  -F 'size=1536x1024' \
  -F 'reference_image=@./hero.png' \
  -F 'reference_image_url=https://example.com/reference.png'
```

### `/v1/responses` 生图简写（`model=gpt-image-2`）

当你在 `POST /v1/responses` 的 **JSON** 请求里直接使用 `model=gpt-image-2` 时，网关会把它视为“生图意图”信号：自动注入 `image_generation` tool，并将上游顶层 `model` 内部路由为 `gpt-5.4-mini`（对外响应仍保持 `gpt-image-2`）。这属于网关兼容扩展，不是 OpenAI 官方语义。

#### REST
```bash focus=1-7
curl https://api.zyxai.de/v1/responses \
  -H "Authorization: Bearer sk-你的站内Key" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "input": "生成一张适合产品页首屏的海报图。"
  }'
```

#### REST
```bash focus=1-9
curl https://api.zyxai.de/v1/images/edits \
  -H "Authorization: Bearer sk-你的站内Key" \
  -F 'model=gpt-image-2' \
  -F 'prompt=把天空替换成日落，并保留前景人物' \
  -F 'image[]=@./source.png' \
  -F 'image[]=@./style.png' \
  -F 'mask=@./mask.png' \
  -F 'size=2048x1152' \
  -F 'output_format=png'
```

兼容扩展的固定规则如下：

- JSON 简写：`input` 是以 `$imagegen ` 开头的字符串，或首个 user `input_text` 以 `$imagegen ` 开头。
- JSON `model` 简写：当 `model=gpt-image-2` 时，即使 `input` 不以 `$imagegen ` 开头，也会触发生图归一；并允许 `image_generation` / `reference_images` / `mask` 在无前缀时生效。
- JSON 扩展字段：`image_generation` 支持 `action`、`size`（支持 `"2K 16:9"` 等 shorthand）、`image_size` + `aspect_ratio`（会换算成明确 `size`）、`quality`、`background`、`output_format`、`output_compression`、`partial_images`、`moderation`、`input_fidelity`、`input_image_mask`、`n`（只支持缺省或 `1`；`n>1` 返回 `400 image_n_not_supported`）；`reference_images` 只接受 `{image_url}` 数组，`mask` / `input_image_mask` 可写成字符串或 `{image_url}`。
- 类型与范围校验：`output_compression` / `partial_images` / `n` 会在网关侧做类型归一与范围校验（支持字符串数字，例如 `"2"`）；非法值返回 `400 invalid_request_error`，错误码 `imagegen_compat_invalid_*`。其中 `n=1` 会被忽略（不透传到 `tools`），`n>1` 返回 `400 image_n_not_supported`，避免再触发上游 `Unknown parameter: tools[0].n`。
- multipart 字段：`model`、`input`、可选 `image_generation` JSON 字符串、可重复 `reference_image`、可重复 `reference_image_url`，以及 `action` / `size` / `image_size` / `aspect_ratio` / `quality` / `background` / `output_format` / `output_compression` / `partial_images` / `moderation` / `input_fidelity` / `n` 便捷别名；`mask` 支持文件上传，也支持 `mask_image_url` / `input_image_mask`。
- 如果你已经显式传了 `tools`，网关默认不会再自动注入 `image_generation`，也不会剥离 `$imagegen` 前缀；但当你使用 `model=gpt-image-2` 简写时，为保证语义一致会补齐 `image_generation` 并强制 `tool_choice` 为 `image_generation`。

兼容扩展常见错误码：

- `image_compat_not_allowed`
- `imagegen_compat_requires_prefix`
- `imagegen_compat_conflict`
- `imagegen_compat_tool_choice_conflict`
- `image_n_not_supported`
- `imagegen_compat_invalid_n`
- `imagegen_compat_invalid_output_compression`
- `imagegen_compat_invalid_partial_images`
- `multipart_stream_unsupported`
  只适用于 `/v1/responses` 的 multipart `$imagegen` 扩展；`/v1/images/generations`、`/v1/images/edits` 在 native / compat 下都已经支持流式返回。
- `unsupported_reference_image_type`
- `reference_image_count_exceeded`
- `reference_image_too_large`
- `reference_image_total_too_large`
- `reference_image_too_large_after_normalization`

### 迁移与边界

- 如果你的客户端本来就支持 `responses`，不要再退回 `chat/completions`。
- 如果你只是为了历史兼容而使用旧参数结构，请切到 `openai` 兼容页。
- 如果你要走 `/v1/images/*`，请先确认当前账号 / 分组的 `image_protocol_mode`；同一条 `/v1/images/*` 路径在 `native` 和 `compat` 下会落到不同执行链，但参数语义会保持一致。
- `gpt-5.4-mini` 这类主模型不再承担“也许是生图”的歧义判断；只有当请求意图是图片生成 / 编辑时，才会进入图片路由。
- 如果你要看 Grok 显式媒体入口，请转到 `grok` 页；如果你是在 `/v1/responses` 内通过 `image_generation` tool 生图，仍以本页规则为准。
- 新项目围绕 `responses` 设计，比继续叠加 `chat/completions` 特性债务更稳妥。
