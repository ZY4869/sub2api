export interface ModelRegistryEntry {
  id: string
  display_name: string
  provider: string
  platforms: string[]
  protocol_ids: string[]
  aliases: string[]
  pricing_lookup_ids: string[]
  context_window_tokens?: number
  preferred_protocol_ids?: Record<string, string>
  modalities: string[]
  capabilities: string[]
  ui_priority: number
  exposed_in: string[]
  status?: string
  deprecated_at?: string
  replaced_by?: string
  deprecation_notice?: string
}

export interface ModelRegistryPreset {
  platform: string
  label: string
  from: string
  to: string
  color: string
  order?: number
}

export interface ModelRegistrySnapshot {
  etag: string
  updated_at: string
  provider_labels: Record<string, string>
  models: ModelRegistryEntry[]
  presets: ModelRegistryPreset[]
}

export const generatedModelRegistryBuiltAt = "2026-05-12T11:33:34Z"

export const generatedModelRegistrySnapshot: ModelRegistrySnapshot = {
  "etag": "W/\"8991f57dde6c61150892232210e687ba8ce173f3650b95fa58e8bacaa782d495\"",
  "updated_at": "2026-05-12T11:33:34Z",
  "provider_labels": {
    "anthropic": "Anthropic-Claude",
    "antigravity": "Antigravity",
    "baidu": "Baidu-Document-AI",
    "deepseek": "DeepSeek",
    "gemini": "Google-Gemini",
    "grok": "xAI-Grok",
    "kiro": "Kiro",
    "openai": "OpenAI-GPT"
  },
  "models": [
    {
      "id": "claude-opus-4.1",
      "display_name": "Claude Opus 4.1",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-opus-4-1-20250805",
        "claude-opus-4.1"
      ],
      "aliases": [
        "claude-opus-4-1-20250805",
        "claude-opus-4-5",
        "claude-opus-4-5-20251101",
        "claude-opus-4-5-thinking"
      ],
      "pricing_lookup_ids": [
        "claude-opus-4-1-20250805"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 0,
      "exposed_in": [
        "use_key",
        "whitelist"
      ]
    },
    {
      "id": "gemini-3.1-flash-image",
      "display_name": "Gemini 3.1 Flash Image",
      "provider": "gemini",
      "platforms": [
        "antigravity",
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.1-flash-image"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-flash-image"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 0,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-3.5-turbo",
      "display_name": "GPT-3.5-turbo",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-3.5-turbo"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-3.5-turbo"
      ],
      "context_window_tokens": 16385,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 0,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "claude-sonnet-4.5",
      "display_name": "Claude Sonnet 4.5",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-sonnet-4.5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-sonnet-4-5-20250929"
      ],
      "context_window_tokens": 200000,
      "preferred_protocol_ids": {
        "anthropic_oauth": "claude-sonnet-4-5-20250929",
        "antigravity": "claude-sonnet-4-5",
        "kiro": "claude-sonnet-4-5-20250929"
      },
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 1,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-2.5-flash-image",
      "display_name": "Gemini 2.5 Flash Image",
      "provider": "gemini",
      "platforms": [
        "antigravity",
        "gemini"
      ],
      "protocol_ids": [
        "gemini-2.5-flash-image"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-flash-image"
      ],
      "context_window_tokens": 32768,
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 1,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-3.5-turbo-0125",
      "display_name": "GPT-3.5-turbo-0125",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-3.5-turbo-0125"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-3.5-turbo-0125"
      ],
      "context_window_tokens": 16385,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 1,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "claude-haiku-4.5",
      "display_name": "Claude Haiku 4.5",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-haiku-4.5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-haiku-4-5-20251001"
      ],
      "context_window_tokens": 200000,
      "preferred_protocol_ids": {
        "anthropic_oauth": "claude-haiku-4-5-20251001",
        "antigravity": "claude-sonnet-4-5",
        "kiro": "claude-haiku-4-5-20251001"
      },
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 2,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-2.0-flash",
      "display_name": "Gemini 2.0 Flash",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-2.0-flash"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.0-flash"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 2,
      "exposed_in": [
        "runtime",
        "test",
        "use_key",
        "whitelist"
      ]
    },
    {
      "id": "gpt-3.5-turbo-1106",
      "display_name": "GPT-3.5-turbo-1106",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-3.5-turbo-1106"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-3.5-turbo-1106"
      ],
      "context_window_tokens": 16385,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 2,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "claude-opus-4-7",
      "display_name": "Claude Opus 4.7",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-opus-4-7"
      ],
      "aliases": [
        "claude-opus-4.7"
      ],
      "pricing_lookup_ids": [
        "claude-opus-4-7"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 3,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gemini-2.5-flash",
      "display_name": "Gemini 2.5 Flash",
      "provider": "gemini",
      "platforms": [
        "antigravity",
        "gemini"
      ],
      "protocol_ids": [
        "gemini-2.5-flash"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-flash"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 3,
      "exposed_in": [
        "runtime",
        "test",
        "use_key",
        "whitelist"
      ]
    },
    {
      "id": "gpt-3.5-turbo-16k",
      "display_name": "GPT-3.5-turbo-16k",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-3.5-turbo-16k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-3.5-turbo-16k"
      ],
      "context_window_tokens": 16385,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 3,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-2.5-pro",
      "display_name": "Gemini 2.5 Pro",
      "provider": "gemini",
      "platforms": [
        "antigravity",
        "gemini"
      ],
      "protocol_ids": [
        "gemini-2.5-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-pro"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 4,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-4",
      "display_name": "GPT-4",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4"
      ],
      "context_window_tokens": 8192,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 4,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-3-flash-preview",
      "display_name": "Gemini 3 Flash Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3-flash-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3-flash-preview"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 5,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-4-turbo",
      "display_name": "GPT-4-turbo",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4-turbo"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4-turbo"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 5,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-2.5-flash-lite",
      "display_name": "Gemini 2.5 Flash Lite",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-2.5-flash-lite"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-flash-lite"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 6,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-4-turbo-preview",
      "display_name": "GPT-4-turbo-preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4-turbo-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4-turbo-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 6,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-4o",
      "display_name": "GPT-4o",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4o"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 7,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-4o-2024-08-06",
      "display_name": "GPT-4o-2024-08-06",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-2024-08-06"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4o-2024-08-06"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 8,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-3-flash",
      "display_name": "Gemini 3 Flash",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-3-flash"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3-flash-preview"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 9,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-4o-2024-11-20",
      "display_name": "GPT-4o-2024-11-20",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-2024-11-20"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4o-2024-11-20"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 9,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-4o-mini",
      "display_name": "GPT-4o-mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-mini"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4o-mini"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 10,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-4o-mini-2024-07-18",
      "display_name": "GPT-4o-mini-2024-07-18",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-mini-2024-07-18"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4o-mini-2024-07-18"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 11,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-3.1-pro-high",
      "display_name": "Gemini 3.1 Pro High",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-3.1-pro-high"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-pro-high"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 12,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-4.5-preview",
      "display_name": "GPT-4.5-preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4.5-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4.5-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 12,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-3.1-pro-low",
      "display_name": "Gemini 3.1 Pro Low",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-3.1-pro-low"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-pro-low"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 13,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-4.1",
      "display_name": "GPT-4.1",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4.1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4.1"
      ],
      "context_window_tokens": 1047576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 13,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-3-pro-image",
      "display_name": "Gemini 3 Pro Image",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-3-pro-image-preview",
        "gemini-3-pro-image"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3-pro-image-preview"
      ],
      "context_window_tokens": 65536,
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 14,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-4.1-mini",
      "display_name": "GPT-4.1-mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4.1-mini"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4.1-mini"
      ],
      "context_window_tokens": 1047576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 14,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-4.1-nano",
      "display_name": "GPT-4.1-nano",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4.1-nano"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4.1-nano"
      ],
      "context_window_tokens": 1047576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 15,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-oss-120b-medium",
      "display_name": "GPT-oss-120b-medium",
      "provider": "openai",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gpt-oss-120b-medium"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-oss-120b-medium"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 15,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o1",
      "display_name": "O1",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "o1"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 16,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o1-pro",
      "display_name": "O1-pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o1-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "o1-pro"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 19,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o3",
      "display_name": "O3",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o3"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "o3"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 20,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o3-mini",
      "display_name": "O3-mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o3-mini"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "o3-mini"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 21,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o3-pro",
      "display_name": "O3-pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o3-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "o3-pro"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 22,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o4-mini",
      "display_name": "O4-mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o4-mini"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "o4-mini"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 23,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5",
      "display_name": "GPT-5",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 24,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-5-2025-08-07",
      "display_name": "GPT-5-2025-08-07",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-2025-08-07"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-2025-08-07"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 25,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5-chat",
      "display_name": "GPT-5-chat",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-chat"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 26,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5-chat-latest",
      "display_name": "GPT-5-chat-latest",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-chat-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-chat-latest"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 27,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.3-codex-spark",
      "display_name": "GPT-5.3 Codex Spark",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.3-codex-spark"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.3-codex-spark"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 29,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-5-pro",
      "display_name": "GPT-5-pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-pro"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 30,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5-pro-2025-10-06",
      "display_name": "GPT-5-pro-2025-10-06",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-pro-2025-10-06"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-pro-2025-10-06"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 31,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5-mini",
      "display_name": "GPT-5-mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-mini"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-mini"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 32,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5-mini-2025-08-07",
      "display_name": "GPT-5-mini-2025-08-07",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-mini-2025-08-07"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-mini-2025-08-07"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 33,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5-nano",
      "display_name": "GPT-5-nano",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-nano"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-nano"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 34,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5-nano-2025-08-07",
      "display_name": "GPT-5-nano-2025-08-07",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-nano-2025-08-07"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-nano-2025-08-07"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 35,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.2",
      "display_name": "GPT-5.2",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.2"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.2"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 42,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.2-2025-12-11",
      "display_name": "GPT-5.2-2025-12-11",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.2-2025-12-11"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.2-2025-12-11"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 43,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.2-chat-latest",
      "display_name": "GPT-5.2-chat-latest",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.2-chat-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.2-chat-latest"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 44,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.2-pro",
      "display_name": "GPT-5.2-pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.2-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.2-pro"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 46,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.2-pro-2025-12-11",
      "display_name": "GPT-5.2-pro-2025-12-11",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.2-pro-2025-12-11"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.2-pro-2025-12-11"
      ],
      "context_window_tokens": 272000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 47,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.4",
      "display_name": "GPT-5.4",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.4"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.4"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "image_generation_tool"
      ],
      "ui_priority": 48,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.5",
      "display_name": "GPT-5.5",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.5"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "image_generation_tool"
      ],
      "ui_priority": 48,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.4-2026-03-05",
      "display_name": "GPT-5.4-2026-03-05",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.4-2026-03-05"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.4-2026-03-05"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "image_generation_tool"
      ],
      "ui_priority": 49,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.4-mini",
      "display_name": "GPT-5.4 Mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.4-mini"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.4-mini"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "image_generation_tool"
      ],
      "ui_priority": 49,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist",
        "use_key"
      ]
    },
    {
      "id": "gpt-5.4-nano",
      "display_name": "GPT-5.4 Nano",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.4-nano"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.4-nano"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 50,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist",
        "use_key"
      ]
    },
    {
      "id": "gpt-5.4-pro",
      "display_name": "GPT-5.4-pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.4-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.4-pro"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "image_generation_tool"
      ],
      "ui_priority": 50,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-5.4-pro-2026-03-05",
      "display_name": "GPT-5.4-pro-2026-03-05",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.4-pro-2026-03-05"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.4-pro"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "image_generation_tool"
      ],
      "ui_priority": 51,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-image-2",
      "display_name": "GPT Image 2",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-image-2"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-image-2"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 52,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist",
        "use_key"
      ]
    },
    {
      "id": "gpt-4o-audio-preview",
      "display_name": "GPT-4o-audio-preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-audio-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4o-audio-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 53,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gpt-4o-realtime-preview",
      "display_name": "GPT-4o-realtime-preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-realtime-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4o-realtime-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 54,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "claude-opus-4-6",
      "display_name": "Claude Opus 4.6",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-opus-4-6"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-opus-4-6"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 200,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "claude-opus-4-6-thinking",
      "display_name": "Claude Opus 4.6 Thinking",
      "provider": "anthropic",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "claude-opus-4-6-thinking"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-opus-4-6-thinking"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 200,
      "exposed_in": [
        "runtime",
        "test"
      ]
    },
    {
      "id": "claude-sonnet-4-6",
      "display_name": "Claude Sonnet 4.6",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-sonnet-4-6"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-sonnet-4-6"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 200,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "deepseek-coder",
      "display_name": "Deepseek-coder",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-coder"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-coder"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "deepseek-r1",
      "display_name": "Deepseek-r1",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-r1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-r1"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "deepseek-r1-0528",
      "display_name": "Deepseek-r1-0528",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-r1-0528"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-r1-0528"
      ],
      "context_window_tokens": 163840,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "deepseek-r1-distill-llama-70b",
      "display_name": "Deepseek-r1-distill-llama-70b",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-r1-distill-llama-70b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-r1-distill-llama-70b"
      ],
      "context_window_tokens": 131072,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "deepseek-r1-distill-llama-8b",
      "display_name": "Deepseek-r1-distill-llama-8b",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-r1-distill-llama-8b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-r1-distill-llama-8b"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "deepseek-r1-distill-qwen-14b",
      "display_name": "Deepseek-r1-distill-qwen-14b",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-r1-distill-qwen-14b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-r1-distill-qwen-14b"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "deepseek-r1-distill-qwen-32b",
      "display_name": "Deepseek-r1-distill-qwen-32b",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-r1-distill-qwen-32b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-r1-distill-qwen-32b"
      ],
      "context_window_tokens": 32768,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "deepseek-r1-distill-qwen-7b",
      "display_name": "Deepseek-r1-distill-qwen-7b",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-r1-distill-qwen-7b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-r1-distill-qwen-7b"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "deepseek-v3",
      "display_name": "Deepseek-v3",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-v3"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-v3"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "deepseek-v3-0324",
      "display_name": "Deepseek-v3-0324",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-v3-0324"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-v3-0324"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": []
    },
    {
      "id": "gemini-3.1-flash-image-preview",
      "display_name": "Gemini 3.1 Flash Image Preview",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-3.1-flash-image-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-flash-image-preview"
      ],
      "context_window_tokens": 65536,
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 200,
      "exposed_in": [
        "runtime",
        "test"
      ]
    },
    {
      "id": "gemini-3.1-pro-preview",
      "display_name": "Gemini 3.1 Pro Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.1-pro-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-pro-preview"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 200,
      "exposed_in": [
        "runtime",
        "test"
      ]
    }
  ],
  "presets": [
    {
      "platform": "anthropic",
      "label": "Opus 4.1",
      "from": "claude-opus-4.1",
      "to": "claude-opus-4.1",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 1
    },
    {
      "platform": "anthropic",
      "label": "Sonnet 4.5",
      "from": "claude-sonnet-4.5",
      "to": "claude-sonnet-4.5",
      "color": "bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400",
      "order": 2
    },
    {
      "platform": "anthropic",
      "label": "Haiku 4.5",
      "from": "claude-haiku-4.5",
      "to": "claude-haiku-4.5",
      "color": "bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400",
      "order": 3
    },
    {
      "platform": "anthropic",
      "label": "Opus-\u003eSonnet",
      "from": "claude-opus-4.1",
      "to": "claude-sonnet-4.5",
      "color": "bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400",
      "order": 4
    },
    {
      "platform": "anthropic",
      "label": "Haiku-\u003eSonnet",
      "from": "claude-haiku-4.5",
      "to": "claude-sonnet-4.5",
      "color": "bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400",
      "order": 5
    },
    {
      "platform": "anthropic",
      "label": "Opus 4.7",
      "from": "claude-opus-4-7",
      "to": "claude-opus-4-7",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 6
    },
    {
      "platform": "openai",
      "label": "GPT-4o",
      "from": "gpt-4o",
      "to": "gpt-4o",
      "color": "bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400",
      "order": 6
    },
    {
      "platform": "openai",
      "label": "GPT-4o Mini",
      "from": "gpt-4o-mini",
      "to": "gpt-4o-mini",
      "color": "bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400",
      "order": 7
    },
    {
      "platform": "openai",
      "label": "GPT-4.1",
      "from": "gpt-4.1",
      "to": "gpt-4.1",
      "color": "bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400",
      "order": 8
    },
    {
      "platform": "openai",
      "label": "o1",
      "from": "o1",
      "to": "o1",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 9
    },
    {
      "platform": "openai",
      "label": "o3",
      "from": "o3",
      "to": "o3",
      "color": "bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400",
      "order": 10
    },
    {
      "platform": "openai",
      "label": "GPT-5",
      "from": "gpt-5",
      "to": "gpt-5",
      "color": "bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400",
      "order": 11
    },
    {
      "platform": "openai",
      "label": "GPT-5.3 Codex Spark",
      "from": "gpt-5.3-codex-spark",
      "to": "gpt-5.3-codex-spark",
      "color": "bg-teal-100 text-teal-700 hover:bg-teal-200 dark:bg-teal-900/30 dark:text-teal-400",
      "order": 12
    },
    {
      "platform": "openai",
      "label": "GPT-5.2",
      "from": "gpt-5.2",
      "to": "gpt-5.2",
      "color": "bg-red-100 text-red-700 hover:bg-red-200 dark:bg-red-900/30 dark:text-red-400",
      "order": 14
    },
    {
      "platform": "openai",
      "label": "GPT-5.4",
      "from": "gpt-5.4",
      "to": "gpt-5.4",
      "color": "bg-rose-100 text-rose-700 hover:bg-rose-200 dark:bg-rose-900/30 dark:text-rose-400",
      "order": 15
    },
    {
      "platform": "openai",
      "label": "Haiku-\u003e5.4",
      "from": "claude-haiku-4.5",
      "to": "gpt-5.4",
      "color": "bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400",
      "order": 17
    },
    {
      "platform": "openai",
      "label": "Opus-\u003e5.4",
      "from": "claude-opus-4.1",
      "to": "gpt-5.4",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 18
    },
    {
      "platform": "openai",
      "label": "Sonnet-\u003e5.4",
      "from": "claude-sonnet-4.5",
      "to": "gpt-5.4",
      "color": "bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400",
      "order": 19
    },
    {
      "platform": "gemini",
      "label": "Flash 2.0",
      "from": "gemini-2.0-flash",
      "to": "gemini-2.0-flash",
      "color": "bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400",
      "order": 20
    },
    {
      "platform": "gemini",
      "label": "2.5 Flash",
      "from": "gemini-2.5-flash",
      "to": "gemini-2.5-flash",
      "color": "bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400",
      "order": 21
    },
    {
      "platform": "gemini",
      "label": "2.5 Image",
      "from": "gemini-2.5-flash-image",
      "to": "gemini-2.5-flash-image",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 22
    },
    {
      "platform": "gemini",
      "label": "2.5 Pro",
      "from": "gemini-2.5-pro",
      "to": "gemini-2.5-pro",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 23
    },
    {
      "platform": "gemini",
      "label": "3.1 Image",
      "from": "gemini-3.1-flash-image",
      "to": "gemini-3.1-flash-image",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 24
    },
    {
      "platform": "antigravity",
      "label": "Claude-\u003eSonnet",
      "from": "claude-*",
      "to": "claude-sonnet-4.5",
      "color": "bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400",
      "order": 30
    },
    {
      "platform": "antigravity",
      "label": "Sonnet-\u003eSonnet",
      "from": "claude-sonnet-*",
      "to": "claude-sonnet-4.5",
      "color": "bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400",
      "order": 31
    },
    {
      "platform": "antigravity",
      "label": "Opus-\u003eOpus",
      "from": "claude-opus-*",
      "to": "claude-opus-4.1",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 32
    },
    {
      "platform": "antigravity",
      "label": "Haiku-\u003eSonnet",
      "from": "claude-haiku-*",
      "to": "claude-sonnet-4.5",
      "color": "bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400",
      "order": 33
    },
    {
      "platform": "antigravity",
      "label": "Sonnet4-\u003e4.5",
      "from": "claude-sonnet-4-20250514",
      "to": "claude-sonnet-4.5",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 34
    },
    {
      "platform": "antigravity",
      "label": "Sonnet3.5-\u003e4.5",
      "from": "claude-3-5-sonnet-20241022",
      "to": "claude-sonnet-4.5",
      "color": "bg-teal-100 text-teal-700 hover:bg-teal-200 dark:bg-teal-900/30 dark:text-teal-400",
      "order": 35
    },
    {
      "platform": "antigravity",
      "label": "3.1-Pro-High passthrough",
      "from": "gemini-3.1-pro-high",
      "to": "gemini-3.1-pro-high",
      "color": "bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-400",
      "order": 39
    },
    {
      "platform": "antigravity",
      "label": "3.1-Pro-Low passthrough",
      "from": "gemini-3.1-pro-low",
      "to": "gemini-3.1-pro-low",
      "color": "bg-yellow-100 text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-400",
      "order": 40
    },
    {
      "platform": "antigravity",
      "label": "2.5-Flash-Image passthrough",
      "from": "gemini-2.5-flash-image",
      "to": "gemini-2.5-flash-image",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 41
    },
    {
      "platform": "antigravity",
      "label": "3.1-Flash-Image passthrough",
      "from": "gemini-3.1-flash-image",
      "to": "gemini-3.1-flash-image",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 42
    },
    {
      "platform": "antigravity",
      "label": "3-Pro-Image-\u003e3.1",
      "from": "gemini-3-pro-image",
      "to": "gemini-3.1-flash-image",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 43
    },
    {
      "platform": "antigravity",
      "label": "Gemini 3-\u003eFlash",
      "from": "gemini-3*",
      "to": "gemini-3-flash",
      "color": "bg-yellow-100 text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-400",
      "order": 44
    },
    {
      "platform": "antigravity",
      "label": "Gemini 2.5-\u003eFlash",
      "from": "gemini-2.5*",
      "to": "gemini-2.5-flash",
      "color": "bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-400",
      "order": 45
    },
    {
      "platform": "antigravity",
      "label": "3-Flash passthrough",
      "from": "gemini-3-flash",
      "to": "gemini-3-flash",
      "color": "bg-lime-100 text-lime-700 hover:bg-lime-200 dark:bg-lime-900/30 dark:text-lime-400",
      "order": 46
    },
    {
      "platform": "antigravity",
      "label": "2.5-Flash-Lite passthrough",
      "from": "gemini-2.5-flash-lite",
      "to": "gemini-2.5-flash-lite",
      "color": "bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400",
      "order": 47
    },
    {
      "platform": "antigravity",
      "label": "Sonnet 4.5",
      "from": "claude-sonnet-4.5",
      "to": "claude-sonnet-4.5",
      "color": "bg-cyan-100 text-cyan-700 hover:bg-cyan-200 dark:bg-cyan-900/30 dark:text-cyan-400",
      "order": 48
    },
    {
      "platform": "antigravity",
      "label": "Haiku 4.5",
      "from": "claude-haiku-4.5",
      "to": "claude-haiku-4.5",
      "color": "bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400",
      "order": 49
    },
    {
      "platform": "antigravity",
      "label": "Opus 4.1",
      "from": "claude-opus-4.1",
      "to": "claude-opus-4.1",
      "color": "bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400",
      "order": 50
    },
    {
      "platform": "antigravity",
      "label": "Opus 4.7",
      "from": "claude-opus-4-7",
      "to": "claude-opus-4-7",
      "color": "bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400",
      "order": 51
    }
  ]
}
