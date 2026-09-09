import type { TimeAccessPolicy } from '../types/api-key-groups'

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
  available_from?: string
  available_until?: string
  access_time_policy?: TimeAccessPolicy | null
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

export const generatedModelRegistryBuiltAt = "2026-09-09T02:48:07Z"

export const generatedModelRegistrySnapshot: ModelRegistrySnapshot = {
  "etag": "W/\"8149d0175a383fffea325165a4f49970fbbdb52dbf30f8c86cbb505b2052b217\"",
  "updated_at": "2026-09-09T02:48:07Z",
  "provider_labels": {
    "anthropic": "Anthropic-Claude",
    "antigravity": "Antigravity",
    "baidu": "Baidu-Document-AI",
    "deepseek": "DeepSeek",
    "gemini": "Google-Gemini",
    "grok": "xAI-Grok",
    "kimi": "Kimi",
    "kiro": "Kiro",
    "openai": "OpenAI-GPT",
    "openrouter": "OpenRouter"
  },
  "models": [
    {
      "id": "abab6.5-chat",
      "display_name": "Abab6.5-chat",
      "provider": "minimax",
      "platforms": [
        "minimax"
      ],
      "protocol_ids": [
        "abab6.5-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "abab6.5-chat"
      ],
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
      "id": "command-a-03-2025",
      "display_name": "Command-a-03-2025",
      "provider": "cohere",
      "platforms": [
        "cohere"
      ],
      "protocol_ids": [
        "command-a-03-2025"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "command-a-03-2025"
      ],
      "context_window_tokens": 256000,
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
      "id": "deepseek-v4-flash",
      "display_name": "DeepSeek V4 Flash",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-v4-flash"
      ],
      "aliases": [
        "deepseek v4 flash",
        "deepseek_v4_flash",
        "deepseek_v4_flash_free",
        "deepseek/deepseek v4 flash:free",
        "deepseek/deepseek-v4-flash:free"
      ],
      "pricing_lookup_ids": [
        "deepseek-v4-flash"
      ],
      "context_window_tokens": 1048576,
      "preferred_protocol_ids": {
        "anthropic_apikey": "deepseek-v4-flash",
        "deepseek": "deepseek-v4-flash",
        "openai": "deepseek-v4-flash"
      },
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 0,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "doubao-pro-256k",
      "display_name": "Doubao-pro-256k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-pro-256k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-pro-256k"
      ],
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
      "id": "ernie-4.0-8k-latest",
      "display_name": "Ernie-4.0-8k-latest",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-4.0-8k-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-4.0-8k-latest"
      ],
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
        "image_generation"
      ],
      "ui_priority": 0,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "glm-4",
      "display_name": "Glm-4",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4"
      ],
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
      "id": "gpt-3.5-turbo",
      "display_name": "GPT-3.5 Turbo",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-3.5-turbo"
      ],
      "aliases": [
        "gpt-3.5-turbo-0125",
        "gpt-3.5-turbo-1106",
        "gpt-3.5-turbo-instruct"
      ],
      "pricing_lookup_ids": [
        "gpt-3.5-turbo"
      ],
      "context_window_tokens": 16385,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 0,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "grok-auto",
      "display_name": "Grok Auto",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-auto"
      ],
      "aliases": [
        "grok-beta"
      ],
      "pricing_lookup_ids": [
        "grok-auto"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 0,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "hunyuan-lite",
      "display_name": "Hunyuan-lite",
      "provider": "hunyuan",
      "platforms": [
        "hunyuan"
      ],
      "protocol_ids": [
        "hunyuan-lite"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "hunyuan-lite"
      ],
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
      "id": "llama-3.3-70b-instruct",
      "display_name": "Llama-3.3-70b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3.3-70b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3.3-70b-instruct"
      ],
      "context_window_tokens": 128000,
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
      "id": "mistral-small-latest",
      "display_name": "Mistral-small-latest",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "mistral-small-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "mistral-small-latest"
      ],
      "context_window_tokens": 131072,
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
      "id": "moonshot-v1-8k",
      "display_name": "Moonshot-v1-8k",
      "provider": "moonshot",
      "platforms": [
        "moonshot"
      ],
      "protocol_ids": [
        "moonshot-v1-8k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "moonshot-v1-8k"
      ],
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
      "id": "qwen-turbo",
      "display_name": "Qwen-turbo",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen-turbo"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen-turbo"
      ],
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
      "id": "sonar",
      "display_name": "Sonar",
      "provider": "perplexity",
      "platforms": [
        "perplexity"
      ],
      "protocol_ids": [
        "sonar"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "sonar"
      ],
      "context_window_tokens": 128000,
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
      "id": "spark-desk",
      "display_name": "Spark-desk",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-desk"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-desk"
      ],
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
      "id": "yi-large",
      "display_name": "Yi-large",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-large"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-large"
      ],
      "context_window_tokens": 32768,
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
      "id": "abab6.5s-chat",
      "display_name": "Abab6.5s-chat",
      "provider": "minimax",
      "platforms": [
        "minimax"
      ],
      "protocol_ids": [
        "abab6.5s-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "abab6.5s-chat"
      ],
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
      "id": "claude-opus-5",
      "display_name": "Claude Opus 5",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-opus-5"
      ],
      "aliases": [
        "claude-opus-5.0"
      ],
      "pricing_lookup_ids": [
        "claude-opus-5"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 1,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
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
        "claude-sonnet-4-5-20250929",
        "claude-sonnet-4.5"
      ],
      "aliases": [
        "claude-sonnet-4-5",
        "claude-sonnet-4-5-20250929",
        "claude-sonnet-4-5-thinking"
      ],
      "pricing_lookup_ids": [
        "claude-sonnet-4-5-20250929"
      ],
      "context_window_tokens": 200000,
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
      "id": "command-r",
      "display_name": "Command-r",
      "provider": "cohere",
      "platforms": [
        "cohere"
      ],
      "protocol_ids": [
        "command-r"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "command-r"
      ],
      "context_window_tokens": 128000,
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
      "id": "deepseek-v4-pro",
      "display_name": "DeepSeek V4 Pro",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-v4-pro"
      ],
      "aliases": [
        "deepseek v4 pro",
        "deepseek_v4_pro",
        "deepseek_v4_pro_free",
        "deepseek/deepseek v4 pro:free",
        "deepseek/deepseek-v4-pro:free"
      ],
      "pricing_lookup_ids": [
        "deepseek-v4-pro"
      ],
      "context_window_tokens": 1048576,
      "preferred_protocol_ids": {
        "anthropic_apikey": "deepseek-v4-pro",
        "deepseek": "deepseek-v4-pro",
        "openai": "deepseek-v4-pro"
      },
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 1,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "doubao-pro-128k",
      "display_name": "Doubao-pro-128k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-pro-128k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-pro-128k"
      ],
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
      "id": "ernie-4.0-8k",
      "display_name": "Ernie-4.0-8k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-4.0-8k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-4.0-8k"
      ],
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
      "aliases": [
        "gemini-2.5-flash-image-preview"
      ],
      "pricing_lookup_ids": [
        "gemini-2.5-flash-image"
      ],
      "context_window_tokens": 32768,
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 1,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "glm-4v",
      "display_name": "Glm-4v",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4v"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4v"
      ],
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
      "id": "grok-3-fast",
      "display_name": "Grok 3 Fast",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-3-fast",
        "grok-3-fast-beta"
      ],
      "aliases": [
        "grok-3-fast-beta"
      ],
      "pricing_lookup_ids": [
        "grok-3-fast"
      ],
      "context_window_tokens": 131072,
      "preferred_protocol_ids": {
        "grok": "grok-3-fast-beta"
      },
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 1,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "grok-4.6",
      "display_name": "Grok 4.6",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4.6",
        "grok",
        "grok-latest",
        "grok-4.6-latest",
        "grok-build-latest"
      ],
      "aliases": [
        "grok",
        "grok-latest",
        "grok-4.6-latest",
        "grok-build-latest"
      ],
      "pricing_lookup_ids": [
        "grok-4.6"
      ],
      "context_window_tokens": 500000,
      "preferred_protocol_ids": {
        "grok": "grok-4.6"
      },
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "x_search"
      ],
      "ui_priority": 1,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "hunyuan-standard",
      "display_name": "Hunyuan-standard",
      "provider": "hunyuan",
      "platforms": [
        "hunyuan"
      ],
      "protocol_ids": [
        "hunyuan-standard"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "hunyuan-standard"
      ],
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
      "id": "llama-3.2-90b-vision-instruct",
      "display_name": "Llama-3.2-90b-vision-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3.2-90b-vision-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3.2-90b-vision-instruct"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 1,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "mistral-medium-latest",
      "display_name": "Mistral-medium-latest",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "mistral-medium-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "mistral-medium-latest"
      ],
      "context_window_tokens": 131072,
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
      "id": "moonshot-v1-32k",
      "display_name": "Moonshot-v1-32k",
      "provider": "moonshot",
      "platforms": [
        "moonshot"
      ],
      "protocol_ids": [
        "moonshot-v1-32k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "moonshot-v1-32k"
      ],
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
      "id": "qwen-plus",
      "display_name": "Qwen-plus",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen-plus"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen-plus"
      ],
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
      "id": "sonar-pro",
      "display_name": "Sonar-pro",
      "provider": "perplexity",
      "platforms": [
        "perplexity"
      ],
      "protocol_ids": [
        "sonar-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "sonar-pro"
      ],
      "context_window_tokens": 200000,
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
      "id": "spark-desk-v1.1",
      "display_name": "Spark-desk-v1.1",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-desk-v1.1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-desk-v1.1"
      ],
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
      "id": "yi-large-turbo",
      "display_name": "Yi-large-turbo",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-large-turbo"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-large-turbo"
      ],
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
      "id": "abab6.5s-chat-pro",
      "display_name": "Abab6.5s-chat-pro",
      "provider": "minimax",
      "platforms": [
        "minimax"
      ],
      "protocol_ids": [
        "abab6.5s-chat-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "abab6.5s-chat-pro"
      ],
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
      "id": "claude-haiku-4.5",
      "display_name": "Claude Haiku 4.5",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-haiku-4-5-20251001",
        "claude-haiku-4.5"
      ],
      "aliases": [
        "claude-haiku-4-5",
        "claude-haiku-4-5-20251001"
      ],
      "pricing_lookup_ids": [
        "claude-haiku-4-5-20251001"
      ],
      "context_window_tokens": 200000,
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
      "id": "claude-opus-4-8",
      "display_name": "Claude Opus 4.8",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-opus-4-8"
      ],
      "aliases": [
        "claude-opus-4.8"
      ],
      "pricing_lookup_ids": [
        "claude-opus-4-8"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 2,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "command-r-plus",
      "display_name": "Command-r-plus",
      "provider": "cohere",
      "platforms": [
        "cohere"
      ],
      "protocol_ids": [
        "command-r-plus"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "command-r-plus"
      ],
      "context_window_tokens": 128000,
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
      "id": "doubao-pro-32k",
      "display_name": "Doubao-pro-32k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-pro-32k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-pro-32k"
      ],
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
      "id": "ernie-4.0-turbo-8k",
      "display_name": "Ernie-4.0-turbo-8k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-4.0-turbo-8k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-4.0-turbo-8k"
      ],
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
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding"
      ],
      "ui_priority": 2,
      "exposed_in": [
        "runtime",
        "test",
        "use_key",
        "whitelist"
      ],
      "status": "deprecated"
    },
    {
      "id": "glm-4-plus",
      "display_name": "Glm-4-plus",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4-plus"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4-plus"
      ],
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
      "id": "grok-4-expert",
      "display_name": "Grok 4 Expert",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4-expert",
        "grok-4",
        "grok-4-0709"
      ],
      "aliases": [
        "grok-4",
        "grok-4-0709"
      ],
      "pricing_lookup_ids": [
        "grok-4-expert"
      ],
      "preferred_protocol_ids": {
        "grok": "grok-4"
      },
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 2,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "grok-4.5",
      "display_name": "Grok 4.5",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4.5",
        "grok-4.5-latest"
      ],
      "aliases": [
        "grok-4.5-latest"
      ],
      "pricing_lookup_ids": [
        "grok-4.5"
      ],
      "context_window_tokens": 256000,
      "preferred_protocol_ids": {
        "grok": "grok-4.5"
      },
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 2,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "hunyuan-standard-256k",
      "display_name": "Hunyuan-standard-256k",
      "provider": "hunyuan",
      "platforms": [
        "hunyuan"
      ],
      "protocol_ids": [
        "hunyuan-standard-256k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "hunyuan-standard-256k"
      ],
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
      "id": "llama-3.2-11b-vision-instruct",
      "display_name": "Llama-3.2-11b-vision-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3.2-11b-vision-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3.2-11b-vision-instruct"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 2,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "mistral-large-latest",
      "display_name": "Mistral-large-latest",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "mistral-large-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "mistral-large-latest"
      ],
      "context_window_tokens": 128000,
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
      "id": "moonshot-v1-128k",
      "display_name": "Moonshot-v1-128k",
      "provider": "moonshot",
      "platforms": [
        "moonshot"
      ],
      "protocol_ids": [
        "moonshot-v1-128k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "moonshot-v1-128k"
      ],
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
      "id": "qwen-max",
      "display_name": "Qwen-max",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen-max"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen-max"
      ],
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
      "id": "sonar-reasoning",
      "display_name": "Sonar-reasoning",
      "provider": "perplexity",
      "platforms": [
        "perplexity"
      ],
      "protocol_ids": [
        "sonar-reasoning"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "sonar-reasoning"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 2,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "spark-desk-v2.1",
      "display_name": "Spark-desk-v2.1",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-desk-v2.1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-desk-v2.1"
      ],
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
      "id": "yi-large-rag",
      "display_name": "Yi-large-rag",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-large-rag"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-large-rag"
      ],
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
      "id": "abab6-chat",
      "display_name": "Abab6-chat",
      "provider": "minimax",
      "platforms": [
        "minimax"
      ],
      "protocol_ids": [
        "abab6-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "abab6-chat"
      ],
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
      "id": "command-r-08-2024",
      "display_name": "Command-r-08-2024",
      "provider": "cohere",
      "platforms": [
        "cohere"
      ],
      "protocol_ids": [
        "command-r-08-2024"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "command-r-08-2024"
      ],
      "context_window_tokens": 128000,
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
      "id": "doubao-pro-4k",
      "display_name": "Doubao-pro-4k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-pro-4k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-pro-4k"
      ],
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
      "id": "ernie-3.5-8k",
      "display_name": "Ernie-3.5-8k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-3.5-8k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-3.5-8k"
      ],
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
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding"
      ],
      "ui_priority": 3,
      "exposed_in": [
        "runtime",
        "test",
        "use_key",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "glm-4-0520",
      "display_name": "Glm-4-0520",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4-0520"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4-0520"
      ],
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
      "id": "grok-4-heavy",
      "display_name": "Grok 4 Heavy",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4-heavy"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-4-heavy"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 3,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "grok-4.3",
      "display_name": "Grok 4.3",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4.3"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-4.3"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 3,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "hunyuan-pro",
      "display_name": "Hunyuan-pro",
      "provider": "hunyuan",
      "platforms": [
        "hunyuan"
      ],
      "protocol_ids": [
        "hunyuan-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "hunyuan-pro"
      ],
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
      "id": "kimi-latest",
      "display_name": "Kimi-latest",
      "provider": "moonshot",
      "platforms": [
        "moonshot"
      ],
      "protocol_ids": [
        "kimi-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "kimi-latest"
      ],
      "context_window_tokens": 131072,
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
      "id": "llama-3-sonar-small-32k-online",
      "display_name": "Llama-3-sonar-small-32k-online",
      "provider": "perplexity",
      "platforms": [
        "perplexity"
      ],
      "protocol_ids": [
        "llama-3-sonar-small-32k-online"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3-sonar-small-32k-online"
      ],
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
      "id": "llama-3.2-3b-instruct",
      "display_name": "Llama-3.2-3b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3.2-3b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3.2-3b-instruct"
      ],
      "context_window_tokens": 131072,
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
      "id": "open-mistral-7b",
      "display_name": "Open-mistral-7b",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "open-mistral-7b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "open-mistral-7b"
      ],
      "context_window_tokens": 32000,
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
      "id": "qwen-max-longcontext",
      "display_name": "Qwen-max-longcontext",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen-max-longcontext"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen-max-longcontext"
      ],
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
      "id": "spark-desk-v3.1",
      "display_name": "Spark-desk-v3.1",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-desk-v3.1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-desk-v3.1"
      ],
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
      "id": "yi-medium",
      "display_name": "Yi-medium",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-medium"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-medium"
      ],
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
      "id": "abab5.5-chat",
      "display_name": "Abab5.5-chat",
      "provider": "minimax",
      "platforms": [
        "minimax"
      ],
      "protocol_ids": [
        "abab5.5-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "abab5.5-chat"
      ],
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
      "id": "claude-fable-5",
      "display_name": "Claude Fable 5",
      "provider": "anthropic",
      "platforms": [
        "anthropic",
        "antigravity"
      ],
      "protocol_ids": [
        "claude-fable-5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-fable-5"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 4,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "command-r-plus-08-2024",
      "display_name": "Command-r-plus-08-2024",
      "provider": "cohere",
      "platforms": [
        "cohere"
      ],
      "protocol_ids": [
        "command-r-plus-08-2024"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "command-r-plus-08-2024"
      ],
      "context_window_tokens": 128000,
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
      "id": "doubao-lite-128k",
      "display_name": "Doubao-lite-128k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-lite-128k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-lite-128k"
      ],
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
      "id": "ernie-3.5-128k",
      "display_name": "Ernie-3.5-128k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-3.5-128k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-3.5-128k"
      ],
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
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding"
      ],
      "ui_priority": 4,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "glm-4-air",
      "display_name": "Glm-4-air",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4-air"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4-air"
      ],
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
      "id": "gpt-4",
      "display_name": "GPT-4",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4"
      ],
      "aliases": [
        "gpt-4-0613",
        "gpt-4-0314"
      ],
      "pricing_lookup_ids": [
        "gpt-4"
      ],
      "context_window_tokens": 8192,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 4,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "grok-build-0.1",
      "display_name": "Grok Build 0.1",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-build-0.1",
        "grok-build"
      ],
      "aliases": [
        "grok-build"
      ],
      "pricing_lookup_ids": [
        "grok-build-0.1"
      ],
      "preferred_protocol_ids": {
        "grok": "grok-build-0.1"
      },
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
      "id": "grok-imagine-1.0-fast",
      "display_name": "Grok Imagine 1.0 Fast",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-imagine-1.0-fast"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-imagine-1.0-fast"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 4,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "hunyuan-turbo",
      "display_name": "Hunyuan-turbo",
      "provider": "hunyuan",
      "platforms": [
        "hunyuan"
      ],
      "protocol_ids": [
        "hunyuan-turbo"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "hunyuan-turbo"
      ],
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
      "id": "kimi-k3",
      "display_name": "Kimi K3",
      "provider": "moonshot",
      "platforms": [
        "moonshot"
      ],
      "protocol_ids": [
        "kimi-k3"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "kimi-k3"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 4,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "llama-3-sonar-large-32k-online",
      "display_name": "Llama-3-sonar-large-32k-online",
      "provider": "perplexity",
      "platforms": [
        "perplexity"
      ],
      "protocol_ids": [
        "llama-3-sonar-large-32k-online"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3-sonar-large-32k-online"
      ],
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
      "id": "llama-3.2-1b-instruct",
      "display_name": "Llama-3.2-1b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3.2-1b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3.2-1b-instruct"
      ],
      "context_window_tokens": 60000,
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
      "id": "open-mixtral-8x7b",
      "display_name": "Open-mixtral-8x7b",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "open-mixtral-8x7b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "open-mixtral-8x7b"
      ],
      "context_window_tokens": 32000,
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
      "id": "qwen-long",
      "display_name": "Qwen-long",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen-long"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen-long"
      ],
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
      "id": "spark-desk-v3.5",
      "display_name": "Spark-desk-v3.5",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-desk-v3.5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-desk-v3.5"
      ],
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
      "id": "yi-medium-200k",
      "display_name": "Yi-medium-200k",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-medium-200k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-medium-200k"
      ],
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
      "id": "abab5.5s-chat",
      "display_name": "Abab5.5s-chat",
      "provider": "minimax",
      "platforms": [
        "minimax"
      ],
      "protocol_ids": [
        "abab5.5s-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "abab5.5s-chat"
      ],
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
      "id": "c4ai-aya-23-35b",
      "display_name": "C4ai-aya-23-35b",
      "provider": "cohere",
      "platforms": [
        "cohere"
      ],
      "protocol_ids": [
        "c4ai-aya-23-35b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "c4ai-aya-23-35b"
      ],
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
      "id": "claude-sonnet-5",
      "display_name": "Claude Sonnet 5",
      "provider": "anthropic",
      "platforms": [
        "anthropic"
      ],
      "protocol_ids": [
        "claude-sonnet-5"
      ],
      "aliases": [
        "claude-sonnet-5.0"
      ],
      "pricing_lookup_ids": [
        "claude-sonnet-5"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 5,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "doubao-lite-32k",
      "display_name": "Doubao-lite-32k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-lite-32k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-lite-32k"
      ],
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
      "id": "ernie-speed-8k",
      "display_name": "Ernie-speed-8k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-speed-8k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-speed-8k"
      ],
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
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding",
        "video_understanding"
      ],
      "ui_priority": 5,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "beta"
    },
    {
      "id": "glm-4-airx",
      "display_name": "Glm-4-airx",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4-airx"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4-airx"
      ],
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
      "id": "gpt-4-turbo",
      "display_name": "GPT-4 Turbo",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4-turbo"
      ],
      "aliases": [
        "gpt-4-turbo-2024-04-09"
      ],
      "pricing_lookup_ids": [
        "gpt-4-turbo"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 5,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "grok-composer-2.5-fast",
      "display_name": "Grok Composer 2.5 Fast",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-composer-2.5-fast",
        "grok-composer",
        "composer-2.5"
      ],
      "aliases": [
        "grok-composer",
        "composer-2.5"
      ],
      "pricing_lookup_ids": [
        "grok-composer-2.5-fast"
      ],
      "preferred_protocol_ids": {
        "grok": "grok-composer-2.5-fast"
      },
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
      "id": "grok-imagine-1.0",
      "display_name": "Grok Imagine 1.0",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-imagine-1.0",
        "grok-imagine-image"
      ],
      "aliases": [
        "grok-imagine-image"
      ],
      "pricing_lookup_ids": [
        "grok-imagine-1.0"
      ],
      "preferred_protocol_ids": {
        "grok": "grok-imagine-image"
      },
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 5,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "hunyuan-large",
      "display_name": "Hunyuan-large",
      "provider": "hunyuan",
      "platforms": [
        "hunyuan"
      ],
      "protocol_ids": [
        "hunyuan-large"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "hunyuan-large"
      ],
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
      "id": "llama-3-sonar-small-32k-chat",
      "display_name": "Llama-3-sonar-small-32k-chat",
      "provider": "perplexity",
      "platforms": [
        "perplexity"
      ],
      "protocol_ids": [
        "llama-3-sonar-small-32k-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3-sonar-small-32k-chat"
      ],
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
      "id": "llama-3.1-405b-instruct",
      "display_name": "Llama-3.1-405b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3.1-405b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3.1-405b-instruct"
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
      "id": "open-mixtral-8x22b",
      "display_name": "Open-mixtral-8x22b",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "open-mixtral-8x22b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "open-mixtral-8x22b"
      ],
      "context_window_tokens": 65336,
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
      "id": "qwen2-72b-instruct",
      "display_name": "Qwen2-72b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2-72b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2-72b-instruct"
      ],
      "context_window_tokens": 32768,
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
      "id": "spark-desk-v4.0",
      "display_name": "Spark-desk-v4.0",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-desk-v4.0"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-desk-v4.0"
      ],
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
      "id": "yi-spark",
      "display_name": "Yi-spark",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-spark"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-spark"
      ],
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
      "id": "c4ai-aya-23-8b",
      "display_name": "C4ai-aya-23-8b",
      "provider": "cohere",
      "platforms": [
        "cohere"
      ],
      "protocol_ids": [
        "c4ai-aya-23-8b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "c4ai-aya-23-8b"
      ],
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
      "id": "claude-fable-5-1",
      "display_name": "Claude Fable 5.1",
      "provider": "anthropic",
      "platforms": [
        "anthropic"
      ],
      "protocol_ids": [
        "claude-fable-5-1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-fable-5-1"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 6,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "codestral-latest",
      "display_name": "Codestral-latest",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "codestral-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "codestral-latest"
      ],
      "context_window_tokens": 32000,
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
      "id": "doubao-lite-4k",
      "display_name": "Doubao-lite-4k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-lite-4k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-lite-4k"
      ],
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
      "id": "ernie-speed-128k",
      "display_name": "Ernie-speed-128k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-speed-128k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-speed-128k"
      ],
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
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding"
      ],
      "ui_priority": 6,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-3-pro-preview",
      "display_name": "Gemini 3 Pro Preview",
      "provider": "gemini",
      "platforms": [
        "antigravity",
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3-pro-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3-pro-preview"
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
      "id": "gemini-3.6-flash",
      "display_name": "Gemini 3.6 Flash",
      "provider": "gemini",
      "platforms": [
        "antigravity",
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.6-flash"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.6-flash"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text",
        "image",
        "audio",
        "video"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding",
        "video_understanding"
      ],
      "ui_priority": 6,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "glm-4-long",
      "display_name": "Glm-4-long",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4-long"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4-long"
      ],
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
      "id": "gpt-4-turbo-preview",
      "display_name": "GPT-4 Turbo Preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4-turbo-preview"
      ],
      "aliases": [
        "gpt-4-0125-preview",
        "gpt-4-1106-vision-preview"
      ],
      "pricing_lookup_ids": [
        "gpt-4-turbo-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 6,
      "exposed_in": [
        "whitelist"
      ],
      "status": "beta"
    },
    {
      "id": "grok-4.20-0309-reasoning",
      "display_name": "Grok 4.20 Reasoning",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4.20-0309-reasoning",
        "grok-4.20-reasoning"
      ],
      "aliases": [
        "grok-4.20-reasoning"
      ],
      "pricing_lookup_ids": [
        "grok-4.20-0309-reasoning"
      ],
      "context_window_tokens": 1000000,
      "preferred_protocol_ids": {
        "grok": "grok-4.20-0309-reasoning"
      },
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
      "id": "grok-imagine-1.0-edit",
      "display_name": "Grok Imagine 1.0 Edit",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-imagine-1.0-edit"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-imagine-1.0-edit"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 6,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "hunyuan-vision",
      "display_name": "Hunyuan-vision",
      "provider": "hunyuan",
      "platforms": [
        "hunyuan"
      ],
      "protocol_ids": [
        "hunyuan-vision"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "hunyuan-vision"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 6,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "llama-3-sonar-large-32k-chat",
      "display_name": "Llama-3-sonar-large-32k-chat",
      "provider": "perplexity",
      "platforms": [
        "perplexity"
      ],
      "protocol_ids": [
        "llama-3-sonar-large-32k-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3-sonar-large-32k-chat"
      ],
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
      "id": "llama-3.1-70b-instruct",
      "display_name": "Llama-3.1-70b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3.1-70b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3.1-70b-instruct"
      ],
      "context_window_tokens": 131072,
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
      "id": "qwen2-57b-a14b-instruct",
      "display_name": "Qwen2-57b-a14b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2-57b-a14b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2-57b-a14b-instruct"
      ],
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
      "id": "spark-lite",
      "display_name": "Spark-lite",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-lite"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-lite"
      ],
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
      "id": "yi-vision",
      "display_name": "Yi-vision",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-vision"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-vision"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 6,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "claude-mythos-5",
      "display_name": "Claude Mythos 5",
      "provider": "anthropic",
      "platforms": [
        "anthropic"
      ],
      "protocol_ids": [
        "claude-mythos-5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-mythos-5"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 7,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "unknown"
    },
    {
      "id": "codestral-mamba",
      "display_name": "Codestral-mamba",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "codestral-mamba"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "codestral-mamba"
      ],
      "context_window_tokens": 256000,
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
      "id": "command",
      "display_name": "Command",
      "provider": "cohere",
      "platforms": [
        "cohere"
      ],
      "protocol_ids": [
        "command"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "command"
      ],
      "context_window_tokens": 4096,
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
      "id": "doubao-vision-pro-32k",
      "display_name": "Doubao-vision-pro-32k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-vision-pro-32k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-vision-pro-32k"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 7,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "ernie-speed-pro-128k",
      "display_name": "Ernie-speed-pro-128k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-speed-pro-128k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-speed-pro-128k"
      ],
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
      "id": "gemini-2.5-flash-thinking",
      "display_name": "Gemini 2.5 Flash Thinking",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-2.5-flash-thinking"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-flash-thinking"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 7,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "glm-4-flash",
      "display_name": "Glm-4-flash",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4-flash"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4-flash"
      ],
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
      "id": "gpt-4o",
      "display_name": "GPT-4o",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o"
      ],
      "aliases": [
        "gpt-4o-2024-11-20",
        "gpt-4o-2024-08-06",
        "gpt-4o-2024-05-13"
      ],
      "pricing_lookup_ids": [
        "gpt-4o"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 7,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "grok-4.20-0309-non-reasoning",
      "display_name": "Grok 4.20 Non Reasoning",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4.20-0309-non-reasoning",
        "grok-4.20-non-reasoning"
      ],
      "aliases": [
        "grok-4.20-non-reasoning"
      ],
      "pricing_lookup_ids": [
        "grok-4.20-0309-non-reasoning"
      ],
      "context_window_tokens": 1000000,
      "preferred_protocol_ids": {
        "grok": "grok-4.20-0309-non-reasoning"
      },
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 7,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "grok-imagine-1.0-video",
      "display_name": "Grok Imagine 1.0 Video",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-imagine-1.0-video",
        "grok-imagine-video"
      ],
      "aliases": [
        "grok-imagine-video"
      ],
      "pricing_lookup_ids": [
        "grok-imagine-1.0-video"
      ],
      "preferred_protocol_ids": {
        "grok": "grok-imagine-video"
      },
      "modalities": [
        "text",
        "video"
      ],
      "capabilities": [
        "video_generation"
      ],
      "ui_priority": 7,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "hunyuan-code",
      "display_name": "Hunyuan-code",
      "provider": "hunyuan",
      "platforms": [
        "hunyuan"
      ],
      "protocol_ids": [
        "hunyuan-code"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "hunyuan-code"
      ],
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
      "id": "llama-3.1-8b-instruct",
      "display_name": "Llama-3.1-8b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3.1-8b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3.1-8b-instruct"
      ],
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
      "id": "qwen2-7b-instruct",
      "display_name": "Qwen2-7b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2-7b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2-7b-instruct"
      ],
      "context_window_tokens": 32768,
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
      "id": "spark-pro",
      "display_name": "Spark-pro",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-pro"
      ],
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
      "id": "yi-1.5-34b-chat",
      "display_name": "Yi-1.5-34b-chat",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-1.5-34b-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-1.5-34b-chat"
      ],
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
      "id": "claude-mythos-5-1",
      "display_name": "Claude Mythos 5.1",
      "provider": "anthropic",
      "platforms": [
        "anthropic"
      ],
      "protocol_ids": [
        "claude-mythos-5-1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-mythos-5-1"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 8,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "unknown"
    },
    {
      "id": "command-light",
      "display_name": "Command-light",
      "provider": "cohere",
      "platforms": [
        "cohere"
      ],
      "protocol_ids": [
        "command-light"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "command-light"
      ],
      "context_window_tokens": 4096,
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
      "id": "doubao-vision-lite-32k",
      "display_name": "Doubao-vision-lite-32k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-vision-lite-32k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-vision-lite-32k"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 8,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "ernie-lite-8k",
      "display_name": "Ernie-lite-8k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-lite-8k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-lite-8k"
      ],
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
      "id": "glm-4v-plus",
      "display_name": "Glm-4v-plus",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4v-plus"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4v-plus"
      ],
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
      "id": "grok-4.20-multi-agent-0309",
      "display_name": "Grok 4.20 Multi Agent",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4.20-multi-agent-0309"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-4.20-multi-agent-0309"
      ],
      "context_window_tokens": 1000000,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 8,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "llama-3-70b-instruct",
      "display_name": "Llama-3-70b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3-70b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3-70b-instruct"
      ],
      "context_window_tokens": 8192,
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
      "id": "pixtral-12b-2409",
      "display_name": "Pixtral-12b-2409",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "pixtral-12b-2409"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "pixtral-12b-2409"
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
      "id": "qwen2.5-72b-instruct",
      "display_name": "Qwen2.5-72b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2.5-72b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2.5-72b-instruct"
      ],
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
      "id": "spark-max",
      "display_name": "Spark-max",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-max"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-max"
      ],
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
      "id": "yi-1.5-9b-chat",
      "display_name": "Yi-1.5-9b-chat",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-1.5-9b-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-1.5-9b-chat"
      ],
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
      "id": "doubao-1.5-pro-256k",
      "display_name": "Doubao-1.5-pro-256k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-1.5-pro-256k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-1.5-pro-256k"
      ],
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
      "id": "ernie-lite-pro-128k",
      "display_name": "Ernie-lite-pro-128k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-lite-pro-128k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-lite-pro-128k"
      ],
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
      "id": "glm-4.5",
      "display_name": "Glm-4.5",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4.5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4.5"
      ],
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
      "id": "llama-3-8b-instruct",
      "display_name": "Llama-3-8b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "llama-3-8b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "llama-3-8b-instruct"
      ],
      "context_window_tokens": 8192,
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
      "id": "pixtral-large-latest",
      "display_name": "Pixtral-large-latest",
      "provider": "mistral",
      "platforms": [
        "mistral"
      ],
      "protocol_ids": [
        "pixtral-large-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "pixtral-large-latest"
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
      "id": "qwen2.5-32b-instruct",
      "display_name": "Qwen2.5-32b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2.5-32b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2.5-32b-instruct"
      ],
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
      "id": "spark-ultra",
      "display_name": "Spark-ultra",
      "provider": "spark",
      "platforms": [
        "spark"
      ],
      "protocol_ids": [
        "spark-ultra"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "spark-ultra"
      ],
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
      "id": "yi-1.5-6b-chat",
      "display_name": "Yi-1.5-6b-chat",
      "provider": "yi",
      "platforms": [
        "yi"
      ],
      "protocol_ids": [
        "yi-1.5-6b-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "yi-1.5-6b-chat"
      ],
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
      "id": "codellama-70b-instruct",
      "display_name": "Codellama-70b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "codellama-70b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "codellama-70b-instruct"
      ],
      "context_window_tokens": 16384,
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
      "id": "doubao-1.5-pro-32k",
      "display_name": "Doubao-1.5-pro-32k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-1.5-pro-32k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-1.5-pro-32k"
      ],
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
      "id": "ernie-tiny-8k",
      "display_name": "Ernie-tiny-8k",
      "provider": "baidu",
      "platforms": [
        "baidu_document_ai"
      ],
      "protocol_ids": [
        "ernie-tiny-8k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "ernie-tiny-8k"
      ],
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
      "id": "gemini-3-pro-high",
      "display_name": "Gemini 3 Pro High",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-3-pro-high"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3-pro-high"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 10,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "glm-4.6",
      "display_name": "Glm-4.6",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4.6"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4.6"
      ],
      "context_window_tokens": 204800,
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
      "id": "gpt-4o-mini",
      "display_name": "GPT-4o Mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-mini"
      ],
      "aliases": [
        "gpt-4o-mini-2024-07-18"
      ],
      "pricing_lookup_ids": [
        "gpt-4o-mini"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 10,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "qwen2.5-14b-instruct",
      "display_name": "Qwen2.5-14b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2.5-14b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2.5-14b-instruct"
      ],
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
      "id": "codellama-34b-instruct",
      "display_name": "Codellama-34b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "codellama-34b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "codellama-34b-instruct"
      ],
      "context_window_tokens": 16384,
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
      "id": "doubao-1.5-lite-32k",
      "display_name": "Doubao-1.5-lite-32k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-1.5-lite-32k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-1.5-lite-32k"
      ],
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
      "id": "gemini-3-pro-low",
      "display_name": "Gemini 3 Pro Low",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-3-pro-low"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3-pro-low"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 11,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "glm-3-turbo",
      "display_name": "Glm-3-turbo",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-3-turbo"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-3-turbo"
      ],
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
      "id": "glm-5.2",
      "display_name": "GLM-5.2",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-5.2"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-5.2"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 11,
      "exposed_in": [
        "runtime",
        "test",
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
      "id": "qwen2.5-7b-instruct",
      "display_name": "Qwen2.5-7b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2.5-7b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2.5-7b-instruct"
      ],
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
      "id": "codellama-13b-instruct",
      "display_name": "Codellama-13b-instruct",
      "provider": "meta",
      "platforms": [
        "meta"
      ],
      "protocol_ids": [
        "codellama-13b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "codellama-13b-instruct"
      ],
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
      "id": "doubao-1.5-pro-vision-32k",
      "display_name": "Doubao-1.5-pro-vision-32k",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-1.5-pro-vision-32k"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-1.5-pro-vision-32k"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 12,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "gemini-3.1-pro",
      "display_name": "Gemini 3.1 Pro",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-3.1-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-pro"
      ],
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
      "id": "glm-4-alltools",
      "display_name": "Glm-4-alltools",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "glm-4-alltools"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "glm-4-alltools"
      ],
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
      "id": "gpt-4.5-preview",
      "display_name": "GPT-4.5 Preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4.5-preview"
      ],
      "aliases": [
        "gpt-4.5-preview-2025-02-27"
      ],
      "pricing_lookup_ids": [
        "gpt-4.5-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 12,
      "exposed_in": [
        "whitelist"
      ],
      "status": "deprecated"
    },
    {
      "id": "qwen2.5-3b-instruct",
      "display_name": "Qwen2.5-3b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2.5-3b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2.5-3b-instruct"
      ],
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
      "id": "chatglm_turbo",
      "display_name": "Chatglm_turbo",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "chatglm_turbo"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "chatglm_turbo"
      ],
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
      "id": "doubao-1.5-thinking-pro",
      "display_name": "Doubao-1.5-thinking-pro",
      "provider": "doubao",
      "platforms": [
        "doubao"
      ],
      "protocol_ids": [
        "doubao-1.5-thinking-pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "doubao-1.5-thinking-pro"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 13,
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
      "aliases": [
        "gpt-4.1-2025-04-14"
      ],
      "pricing_lookup_ids": [
        "gpt-4.1"
      ],
      "context_window_tokens": 1047576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 13,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "qwen2.5-1.5b-instruct",
      "display_name": "Qwen2.5-1.5b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2.5-1.5b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2.5-1.5b-instruct"
      ],
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
      "id": "chatglm_pro",
      "display_name": "Chatglm_pro",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "chatglm_pro"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "chatglm_pro"
      ],
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
        "image_generation"
      ],
      "ui_priority": 14,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
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
      "ui_priority": 14,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ]
    },
    {
      "id": "gpt-4.1-mini",
      "display_name": "GPT-4.1 Mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4.1-mini"
      ],
      "aliases": [
        "gpt-4.1-mini-2025-04-14"
      ],
      "pricing_lookup_ids": [
        "gpt-4.1-mini"
      ],
      "context_window_tokens": 1047576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 14,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "qwen2.5-coder-32b-instruct",
      "display_name": "Qwen2.5-coder-32b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2.5-coder-32b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2.5-coder-32b-instruct"
      ],
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
      "id": "chatglm_std",
      "display_name": "Chatglm_std",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "chatglm_std"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "chatglm_std"
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
      "id": "gpt-4.1-nano",
      "display_name": "GPT-4.1 nano",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4.1-nano"
      ],
      "aliases": [
        "gpt-4.1-nano-2025-04-14"
      ],
      "pricing_lookup_ids": [
        "gpt-4.1-nano"
      ],
      "context_window_tokens": 1047576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 15,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
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
      "id": "qwen2.5-coder-14b-instruct",
      "display_name": "Qwen2.5-coder-14b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2.5-coder-14b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2.5-coder-14b-instruct"
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
      "id": "chatglm_lite",
      "display_name": "Chatglm_lite",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "chatglm_lite"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "chatglm_lite"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 16,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o1",
      "display_name": "o1",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o1"
      ],
      "aliases": [
        "o1-2024-12-17"
      ],
      "pricing_lookup_ids": [
        "o1"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 16,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "qwen2.5-coder-7b-instruct",
      "display_name": "Qwen2.5-coder-7b-instruct",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen2.5-coder-7b-instruct"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen2.5-coder-7b-instruct"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 16,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "tab_flash_lite_preview",
      "display_name": "Tab_flash_lite_preview",
      "provider": "antigravity",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "tab_flash_lite_preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "tab_flash_lite_preview"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 16,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "cogview-3",
      "display_name": "Cogview-3",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "cogview-3"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "cogview-3"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 17,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o1-preview",
      "display_name": "o1 Preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o1-preview"
      ],
      "aliases": [
        "o1-preview-2024-09-12"
      ],
      "pricing_lookup_ids": [
        "o1-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 17,
      "exposed_in": [
        "whitelist"
      ],
      "status": "beta"
    },
    {
      "id": "qwen3-235b-a22b",
      "display_name": "Qwen3-235b-a22b",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwen3-235b-a22b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwen3-235b-a22b"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 17,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "cogvideo",
      "display_name": "Cogvideo",
      "provider": "zhipu",
      "platforms": [
        "zhipu"
      ],
      "protocol_ids": [
        "cogvideo"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "cogvideo"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 18,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o1-mini",
      "display_name": "o1-mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o1-mini"
      ],
      "aliases": [
        "o1-mini-2024-09-12"
      ],
      "pricing_lookup_ids": [
        "o1-mini"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 18,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "qwq-32b",
      "display_name": "Qwq-32b",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwq-32b"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwq-32b"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 18,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "o1-pro",
      "display_name": "o1-pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o1-pro"
      ],
      "aliases": [
        "o1-pro-2025-03-19"
      ],
      "pricing_lookup_ids": [
        "o1-pro"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 19,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "qwq-32b-preview",
      "display_name": "Qwq-32b-preview",
      "provider": "qwen",
      "platforms": [
        "qwen"
      ],
      "protocol_ids": [
        "qwq-32b-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "qwq-32b-preview"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 19,
      "exposed_in": [
        "whitelist"
      ]
    },
    {
      "id": "deepseek-chat",
      "display_name": "DeepSeek Chat",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-chat"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-chat"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 20,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "deprecated_at": "2026-07-24",
      "replaced_by": "deepseek-v4-flash",
      "deprecation_notice": "Official DeepSeek legacy model ID remains callable until 2026-07-24."
    },
    {
      "id": "o3",
      "display_name": "o3",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o3"
      ],
      "aliases": [
        "o3-2025-04-16"
      ],
      "pricing_lookup_ids": [
        "o3"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 20,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "deepseek-reasoner",
      "display_name": "DeepSeek Reasoner",
      "provider": "deepseek",
      "platforms": [
        "deepseek"
      ],
      "protocol_ids": [
        "deepseek-reasoner"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deepseek-reasoner"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "reasoning"
      ],
      "ui_priority": 21,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "deprecated_at": "2026-07-24",
      "replaced_by": "deepseek-v4-flash",
      "deprecation_notice": "Official DeepSeek legacy model ID remains callable until 2026-07-24."
    },
    {
      "id": "o3-mini",
      "display_name": "o3-mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o3-mini"
      ],
      "aliases": [
        "o3-mini-2025-01-31"
      ],
      "pricing_lookup_ids": [
        "o3-mini"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 21,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "o3-pro",
      "display_name": "o3-pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o3-pro"
      ],
      "aliases": [
        "o3-pro-2025-06-10"
      ],
      "pricing_lookup_ids": [
        "o3-pro"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 22,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "o4-mini",
      "display_name": "o4-mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o4-mini"
      ],
      "aliases": [
        "o4-mini-2025-04-16"
      ],
      "pricing_lookup_ids": [
        "o4-mini"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 23,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
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
      "aliases": [
        "gpt-5-2025-08-07"
      ],
      "pricing_lookup_ids": [
        "gpt-5"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 24,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
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
      "context_window_tokens": 400000,
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
      "display_name": "GPT-5 Chat",
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
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 27,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
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
      "display_name": "GPT-5 Pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-pro"
      ],
      "aliases": [
        "gpt-5-pro-2025-10-06"
      ],
      "pricing_lookup_ids": [
        "gpt-5-pro"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 30,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "grok-4",
      "display_name": "Grok-4",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-4"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 30,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-4-expert"
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
      "context_window_tokens": 400000,
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
      "id": "grok-4-0709",
      "display_name": "Grok-4-0709",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-4-0709"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-4-0709"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 31,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-4-expert"
    },
    {
      "id": "gpt-5-mini",
      "display_name": "GPT-5 Mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-mini"
      ],
      "aliases": [
        "gpt-5-mini-2025-08-07"
      ],
      "pricing_lookup_ids": [
        "gpt-5-mini"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 32,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "grok-3-beta",
      "display_name": "Grok-3-beta",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-3-beta"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-3-beta"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 32,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-auto"
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
      "context_window_tokens": 400000,
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
      "id": "grok-3-mini-beta",
      "display_name": "Grok-3-mini-beta",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-3-mini-beta"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-3-mini-beta"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 33,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-auto"
    },
    {
      "id": "gpt-5-nano",
      "display_name": "GPT-5 nano",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-nano"
      ],
      "aliases": [
        "gpt-5-nano-2025-08-07"
      ],
      "pricing_lookup_ids": [
        "gpt-5-nano"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 34,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "grok-3-fast-beta",
      "display_name": "Grok-3-fast-beta",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-3-fast-beta"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-3-fast-beta"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 34,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-3-fast"
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
      "context_window_tokens": 400000,
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
      "id": "grok-2",
      "display_name": "Grok-2",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-2"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-2"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 35,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-auto"
    },
    {
      "id": "grok-2-vision",
      "display_name": "Grok-2-vision",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-2-vision"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-2-vision"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 36,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-auto"
    },
    {
      "id": "grok-imagine-image",
      "display_name": "Grok Imagine Image",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-imagine-image"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-imagine-image"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 37,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-imagine-1.0"
    },
    {
      "id": "grok-imagine-video",
      "display_name": "Grok Imagine Video",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-imagine-video"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-imagine-video"
      ],
      "modalities": [
        "text",
        "video"
      ],
      "capabilities": [
        "video_generation"
      ],
      "ui_priority": 38,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-imagine-1.0-video"
    },
    {
      "id": "grok-2-image",
      "display_name": "Grok-2-image",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-2-image"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-2-image"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 39,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-imagine-1.0"
    },
    {
      "id": "grok-beta",
      "display_name": "Grok-beta",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-beta"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-beta"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 40,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-auto"
    },
    {
      "id": "grok-vision-beta",
      "display_name": "Grok-vision-beta",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-vision-beta"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-vision-beta"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image"
      ],
      "ui_priority": 41,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "deprecated",
      "replaced_by": "grok-auto"
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
      "aliases": [
        "gpt-5.2-2025-12-11"
      ],
      "pricing_lookup_ids": [
        "gpt-5.2"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 42,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
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
      "context_window_tokens": 400000,
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
      "display_name": "GPT-5.2 Chat",
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
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 44,
      "exposed_in": [
        "whitelist"
      ],
      "status": "deprecated"
    },
    {
      "id": "gpt-5.2-pro",
      "display_name": "GPT-5.2 Pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.2-pro"
      ],
      "aliases": [
        "gpt-5.2-pro-2025-12-11"
      ],
      "pricing_lookup_ids": [
        "gpt-5.2-pro"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 46,
      "exposed_in": [
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.6-luna",
      "display_name": "GPT-5.6 Luna",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.6-luna"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.6-luna"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "image_generation_tool"
      ],
      "ui_priority": 46,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.6-sol",
      "display_name": "GPT-5.6 Sol",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.6-sol"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.6-sol"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "image_generation_tool"
      ],
      "ui_priority": 46,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.6-terra",
      "display_name": "GPT-5.6 Terra",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.6-terra"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.6-terra"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "image_generation_tool"
      ],
      "ui_priority": 46,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
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
      "context_window_tokens": 400000,
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
      "aliases": [
        "gpt-5.4-2026-03-05"
      ],
      "pricing_lookup_ids": [
        "gpt-5.4"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "image_generation_tool"
      ],
      "ui_priority": 48,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
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
      "aliases": [
        "gpt-5.5-2026-04-23"
      ],
      "pricing_lookup_ids": [
        "gpt-5.5"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "image_generation_tool"
      ],
      "ui_priority": 48,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
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
      "aliases": [
        "gpt-5.4-mini-2026-03-17"
      ],
      "pricing_lookup_ids": [
        "gpt-5.4-mini"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "image_generation_tool"
      ],
      "ui_priority": 49,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist",
        "use_key"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-6-astra",
      "display_name": "GPT-6 Astra",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-6-astra"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-6-astra"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 49,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.4-nano",
      "display_name": "GPT-5.4 nano",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.4-nano"
      ],
      "aliases": [
        "gpt-5.4-nano-2026-03-17"
      ],
      "pricing_lookup_ids": [
        "gpt-5.4-nano"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 50,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist",
        "use_key"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.4-pro",
      "display_name": "GPT-5.4 Pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.4-pro"
      ],
      "aliases": [
        "gpt-5.4-pro-2026-03-05"
      ],
      "pricing_lookup_ids": [
        "gpt-5.4-pro"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "image_generation_tool"
      ],
      "ui_priority": 50,
      "exposed_in": [
        "runtime",
        "test",
        "whitelist"
      ],
      "status": "stable"
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
      "id": "chatgpt-4o-latest",
      "display_name": "ChatGPT-4o",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "chatgpt-4o-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "chatgpt-4o-latest"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 52,
      "exposed_in": [
        "whitelist"
      ],
      "status": "deprecated"
    },
    {
      "id": "gpt-image-2",
      "display_name": "GPT-Image-2",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-image-2"
      ],
      "aliases": [
        "gpt-image-2-2026-04-21"
      ],
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
      ],
      "status": "stable"
    },
    {
      "id": "gpt-4o-audio-preview",
      "display_name": "GPT-4o Audio",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-audio-preview"
      ],
      "aliases": [
        "gpt-4o-audio-preview-2025-06-03",
        "gpt-4o-audio-preview-2024-12-17",
        "gpt-4o-audio-preview-2024-10-01"
      ],
      "pricing_lookup_ids": [
        "gpt-4o-audio-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 53,
      "exposed_in": [
        "whitelist"
      ],
      "status": "beta"
    },
    {
      "id": "gpt-4o-realtime-preview",
      "display_name": "GPT-4o Realtime",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-realtime-preview"
      ],
      "aliases": [
        "gpt-4o-realtime-preview-2025-06-03",
        "gpt-4o-realtime-preview-2024-12-17",
        "gpt-4o-realtime-preview-2024-10-01"
      ],
      "pricing_lookup_ids": [
        "gpt-4o-realtime-preview"
      ],
      "context_window_tokens": 32000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 54,
      "exposed_in": [
        "whitelist"
      ],
      "status": "beta"
    },
    {
      "id": "babbage-002",
      "display_name": "babbage-002",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "babbage-002"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "babbage-002"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 100,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "chat-latest",
      "display_name": "Chat Latest",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "chat-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "chat-latest"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 101,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "chatgpt-image-latest",
      "display_name": "chatgpt-image-latest",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "chatgpt-image-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "chatgpt-image-latest"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 102,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "codex-mini-latest",
      "display_name": "codex-mini-latest",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "codex-mini-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "codex-mini-latest"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 103,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "computer-use-preview",
      "display_name": "computer-use-preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "computer-use-preview"
      ],
      "aliases": [
        "computer-use-preview-2025-03-11"
      ],
      "pricing_lookup_ids": [
        "computer-use-preview"
      ],
      "context_window_tokens": 8192,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 104,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "beta"
    },
    {
      "id": "davinci-002",
      "display_name": "davinci-002",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "davinci-002"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "davinci-002"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 105,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-4o-mini-audio-preview",
      "display_name": "GPT-4o Mini Audio",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-mini-audio-preview"
      ],
      "aliases": [
        "gpt-4o-mini-audio-preview-2024-12-17"
      ],
      "pricing_lookup_ids": [
        "gpt-4o-mini-audio-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 106,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "gpt-4o-mini-realtime-preview",
      "display_name": "GPT-4o Mini Realtime",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-mini-realtime-preview"
      ],
      "aliases": [
        "gpt-4o-mini-realtime-preview-2024-12-17"
      ],
      "pricing_lookup_ids": [
        "gpt-4o-mini-realtime-preview"
      ],
      "context_window_tokens": 16000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 107,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "gpt-4o-mini-search-preview",
      "display_name": "GPT-4o Mini Search Preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-mini-search-preview"
      ],
      "aliases": [
        "gpt-4o-mini-search-preview-2025-03-11"
      ],
      "pricing_lookup_ids": [
        "gpt-4o-mini-search-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 108,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "beta"
    },
    {
      "id": "gpt-4o-mini-transcribe",
      "display_name": "GPT-4o Mini Transcribe",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-mini-transcribe"
      ],
      "aliases": [
        "gpt-4o-mini-transcribe-2025-03-20",
        "gpt-4o-mini-transcribe-2025-12-15"
      ],
      "pricing_lookup_ids": [
        "gpt-4o-mini-transcribe"
      ],
      "context_window_tokens": 16000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 109,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-4o-mini-tts",
      "display_name": "GPT-4o Mini TTS",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-mini-tts"
      ],
      "aliases": [
        "gpt-4o-mini-tts-2025-03-20",
        "gpt-4o-mini-tts-2025-12-15"
      ],
      "pricing_lookup_ids": [
        "gpt-4o-mini-tts"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 110,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-4o-search-preview",
      "display_name": "GPT-4o Search Preview",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-search-preview"
      ],
      "aliases": [
        "gpt-4o-search-preview-2025-03-11"
      ],
      "pricing_lookup_ids": [
        "gpt-4o-search-preview"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 111,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "beta"
    },
    {
      "id": "gpt-4o-transcribe-diarize",
      "display_name": "GPT-4o Transcribe Diarize",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-transcribe-diarize"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4o-transcribe-diarize"
      ],
      "context_window_tokens": 16000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 112,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-4o-transcribe",
      "display_name": "GPT-4o Transcribe",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-4o-transcribe"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-4o-transcribe"
      ],
      "context_window_tokens": 16000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 113,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5-codex",
      "display_name": "GPT-5-Codex",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5-codex"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5-codex"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 114,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.1-chat-latest",
      "display_name": "GPT-5.1 Chat",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.1-chat-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.1-chat-latest"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 115,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.1-codex-max",
      "display_name": "GPT-5.1-Codex-Max",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.1-codex-max"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.1-codex-max"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 116,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.1-codex-mini",
      "display_name": "GPT-5.1-Codex Mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.1-codex-mini"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.1-codex-mini"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 117,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.1-codex",
      "display_name": "GPT-5.1-Codex",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.1-codex"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.1-codex"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 118,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.1",
      "display_name": "GPT-5.1",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.1"
      ],
      "aliases": [
        "gpt-5.1-2025-11-13"
      ],
      "pricing_lookup_ids": [
        "gpt-5.1"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 119,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.2-codex",
      "display_name": "GPT-5.2-Codex",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.2-codex"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.2-codex"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 120,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.3-codex",
      "display_name": "GPT-5.3-Codex",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.3-codex"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-5.3-codex"
      ],
      "context_window_tokens": 400000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 121,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-5.5-pro",
      "display_name": "GPT-5.5 Pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-5.5-pro"
      ],
      "aliases": [
        "gpt-5.5-pro-2026-04-23"
      ],
      "pricing_lookup_ids": [
        "gpt-5.5-pro"
      ],
      "context_window_tokens": 1050000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 122,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-audio-1.5",
      "display_name": "GPT-Audio-1.5",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-audio-1.5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-audio-1.5"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 123,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-audio-mini",
      "display_name": "GPT-Audio Mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-audio-mini"
      ],
      "aliases": [
        "gpt-audio-mini-2025-10-06",
        "gpt-audio-mini-2025-12-15"
      ],
      "pricing_lookup_ids": [
        "gpt-audio-mini"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 124,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-audio",
      "display_name": "GPT-Audio",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-audio"
      ],
      "aliases": [
        "gpt-audio-2025-08-28"
      ],
      "pricing_lookup_ids": [
        "gpt-audio"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 125,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-image-1-mini",
      "display_name": "GPT-Image-1 Mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-image-1-mini"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-image-1-mini"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 126,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-image-1.5",
      "display_name": "GPT-Image-1.5",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-image-1.5"
      ],
      "aliases": [
        "gpt-image-1.5-2025-12-16"
      ],
      "pricing_lookup_ids": [
        "gpt-image-1.5"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 127,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-image-1",
      "display_name": "GPT-Image-1",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-image-1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-image-1"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 128,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-live-transcribe",
      "display_name": "GPT-Live-Transcribe",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-live-transcribe"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-live-transcribe"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 129,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-realtime-1.5",
      "display_name": "GPT-Realtime-1.5",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-realtime-1.5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-realtime-1.5"
      ],
      "context_window_tokens": 32000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 130,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-realtime-2.1-mini",
      "display_name": "GPT-Realtime-2.1 Mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-realtime-2.1-mini"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-realtime-2.1-mini"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 131,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-realtime-2.1",
      "display_name": "GPT-Realtime-2.1",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-realtime-2.1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-realtime-2.1"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 132,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-realtime-2",
      "display_name": "GPT-Realtime-2",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-realtime-2"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-realtime-2"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 133,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-realtime-mini",
      "display_name": "GPT-Realtime Mini",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-realtime-mini"
      ],
      "aliases": [
        "gpt-realtime-mini-2025-10-06",
        "gpt-realtime-mini-2025-12-15"
      ],
      "pricing_lookup_ids": [
        "gpt-realtime-mini"
      ],
      "context_window_tokens": 32000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 134,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-realtime-translate",
      "display_name": "GPT-Realtime-Translate",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-realtime-translate"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-realtime-translate"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 135,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-realtime-whisper",
      "display_name": "GPT-Realtime-Whisper",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-realtime-whisper"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-realtime-whisper"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 136,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-realtime",
      "display_name": "GPT-Realtime",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-realtime"
      ],
      "aliases": [
        "gpt-realtime-2025-08-28"
      ],
      "pricing_lookup_ids": [
        "gpt-realtime"
      ],
      "context_window_tokens": 32000,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 137,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gpt-transcribe",
      "display_name": "GPT-Transcribe",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "gpt-transcribe"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gpt-transcribe"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 138,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "o3-deep-research",
      "display_name": "o3-deep-research",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o3-deep-research"
      ],
      "aliases": [
        "o3-deep-research-2025-06-26"
      ],
      "pricing_lookup_ids": [
        "o3-deep-research"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 139,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "o4-mini-deep-research",
      "display_name": "o4-mini-deep-research",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "o4-mini-deep-research"
      ],
      "aliases": [
        "o4-mini-deep-research-2025-06-26"
      ],
      "pricing_lookup_ids": [
        "o4-mini-deep-research"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search"
      ],
      "ui_priority": 140,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "omni-moderation-latest",
      "display_name": "omni-moderation",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "omni-moderation-latest"
      ],
      "aliases": [
        "omni-moderation-2024-09-26"
      ],
      "pricing_lookup_ids": [
        "omni-moderation-latest"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "moderation"
      ],
      "ui_priority": 141,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "sora-2-pro",
      "display_name": "Sora 2 Pro",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "sora-2-pro"
      ],
      "aliases": [
        "sora-2-pro-2025-10-06"
      ],
      "pricing_lookup_ids": [
        "sora-2-pro"
      ],
      "modalities": [
        "video"
      ],
      "capabilities": [
        "video_generation"
      ],
      "ui_priority": 142,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "sora-2",
      "display_name": "Sora 2",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "sora-2"
      ],
      "aliases": [
        "sora-2-2025-12-08",
        "sora-2-2025-10-06"
      ],
      "pricing_lookup_ids": [
        "sora-2"
      ],
      "modalities": [
        "video"
      ],
      "capabilities": [
        "video_generation"
      ],
      "ui_priority": 143,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "text-embedding-3-large",
      "display_name": "text-embedding-3-large",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "text-embedding-3-large"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "text-embedding-3-large"
      ],
      "modalities": [
        "embedding"
      ],
      "capabilities": [
        "embedding"
      ],
      "ui_priority": 144,
      "exposed_in": [
        "runtime",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "text-embedding-3-small",
      "display_name": "text-embedding-3-small",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "text-embedding-3-small"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "text-embedding-3-small"
      ],
      "modalities": [
        "embedding"
      ],
      "capabilities": [
        "embedding"
      ],
      "ui_priority": 145,
      "exposed_in": [
        "runtime",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "text-embedding-ada-002",
      "display_name": "text-embedding-ada-002",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "text-embedding-ada-002"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "text-embedding-ada-002"
      ],
      "modalities": [
        "embedding"
      ],
      "capabilities": [
        "embedding"
      ],
      "ui_priority": 146,
      "exposed_in": [
        "runtime",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "text-moderation-latest",
      "display_name": "text-moderation",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "text-moderation-latest"
      ],
      "aliases": [
        "text-moderation-007"
      ],
      "pricing_lookup_ids": [
        "text-moderation-latest"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "moderation"
      ],
      "ui_priority": 147,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "text-moderation-stable",
      "display_name": "text-moderation-stable",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "text-moderation-stable"
      ],
      "aliases": [
        "text-moderation-007"
      ],
      "pricing_lookup_ids": [
        "text-moderation-stable"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "moderation"
      ],
      "ui_priority": 148,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "tts-1-hd",
      "display_name": "TTS-1 HD",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "tts-1-hd"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "tts-1-hd"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 149,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "tts-1",
      "display_name": "TTS-1",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "tts-1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "tts-1"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 150,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "whisper-1",
      "display_name": "Whisper",
      "provider": "openai",
      "platforms": [
        "openai"
      ],
      "protocol_ids": [
        "whisper-1"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "whisper-1"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 151,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-2.5-computer-use-preview-10-2025",
      "display_name": "Gemini 2.5 Computer Use Preview 10 2025",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-2.5-computer-use-preview-10-2025"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-computer-use-preview-10-2025"
      ],
      "context_window_tokens": 128000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 152,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "beta"
    },
    {
      "id": "gemini-2.5-flash-native-audio-preview-12-2025",
      "display_name": "Gemini 2.5 Flash Native Audio Preview 12 2025",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-2.5-flash-native-audio-preview-12-2025"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-flash-native-audio-preview-12-2025"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 153,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "gemini-2.5-flash-preview-tts",
      "display_name": "Gemini 2.5 Flash Preview TTS",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-2.5-flash-preview-tts"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-flash-preview-tts"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 154,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "gemini-2.5-pro-preview-tts",
      "display_name": "Gemini 2.5 Pro Preview TTS",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-2.5-pro-preview-tts"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-pro-preview-tts"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 155,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "gemini-3.1-flash-lite-image",
      "display_name": "Gemini 3.1 Flash Lite Image",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.1-flash-lite-image"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-flash-lite-image"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 156,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-3.1-flash-lite",
      "display_name": "Gemini 3.1 Flash Lite",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.1-flash-lite"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-flash-lite"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding",
        "video_understanding"
      ],
      "ui_priority": 157,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-3.1-flash-live-preview",
      "display_name": "Gemini 3.1 Flash Live Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.1-flash-live-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-flash-live-preview"
      ],
      "context_window_tokens": 131072,
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 158,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "gemini-3.1-flash-tts-preview",
      "display_name": "Gemini 3.1 Flash TTS Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.1-flash-tts-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.1-flash-tts-preview"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 159,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "gemini-3.5-flash-lite",
      "display_name": "Gemini 3.5 Flash Lite",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.5-flash-lite"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.5-flash-lite"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding",
        "video_understanding"
      ],
      "ui_priority": 160,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-3.5-flash",
      "display_name": "Gemini 3.5 Flash",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.5-flash"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.5-flash"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding",
        "video_understanding"
      ],
      "ui_priority": 161,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-3.5-live-translate-preview",
      "display_name": "Gemini 3.5 Live Translate Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.5-live-translate-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.5-live-translate-preview"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 162,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "gemini-3.5-transcribe",
      "display_name": "Gemini 3.5 Transcribe",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.5-transcribe"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.5-transcribe"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 163,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-3.7-flash",
      "display_name": "Gemini 3.7 Flash",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.7-flash"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.7-flash"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding",
        "video_understanding"
      ],
      "ui_priority": 164,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-3.8-flash",
      "display_name": "Gemini 3.8 Flash",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.8-flash"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.8-flash"
      ],
      "context_window_tokens": 1048576,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding",
        "video_understanding"
      ],
      "ui_priority": 165,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-embedding-001",
      "display_name": "Gemini Embedding 001",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-embedding-001"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-embedding-001"
      ],
      "context_window_tokens": 2048,
      "modalities": [
        "embedding"
      ],
      "capabilities": [
        "embedding"
      ],
      "ui_priority": 166,
      "exposed_in": [
        "runtime",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-embedding-2",
      "display_name": "Gemini Embedding 2",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-embedding-2"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-embedding-2"
      ],
      "context_window_tokens": 8192,
      "modalities": [
        "embedding"
      ],
      "capabilities": [
        "embedding"
      ],
      "ui_priority": 167,
      "exposed_in": [
        "runtime",
        "whitelist"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-omni-1.1-flash",
      "display_name": "Gemini Omni 1.1 Flash",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-omni-1.1-flash"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-omni-1.1-flash"
      ],
      "modalities": [
        "video"
      ],
      "capabilities": [
        "video_generation"
      ],
      "ui_priority": 168,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-robotics-er-1.6-preview",
      "display_name": "Gemini Robotics Er 1.6 Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-robotics-er-1.6-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-robotics-er-1.6-preview"
      ],
      "context_window_tokens": 131072,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding"
      ],
      "ui_priority": 169,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "beta"
    },
    {
      "id": "gemini-robotics-er-2-preview",
      "display_name": "Gemini Robotics Er 2 Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-robotics-er-2-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-robotics-er-2-preview"
      ],
      "context_window_tokens": 131072,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding"
      ],
      "ui_priority": 170,
      "exposed_in": [
        "runtime",
        "whitelist",
        "test"
      ],
      "status": "beta"
    },
    {
      "id": "lyria-3-clip-preview",
      "display_name": "Lyria 3 Clip Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "lyria-3-clip-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "lyria-3-clip-preview"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 171,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "lyria-3.5",
      "display_name": "Lyria 3.5",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "lyria-3.5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "lyria-3.5"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_generation"
      ],
      "ui_priority": 172,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "lyria-realtime-exp",
      "display_name": "Lyria Realtime Exp",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "lyria-realtime-exp"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "lyria-realtime-exp"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 173,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "veo-3.1-generate-preview",
      "display_name": "Veo 3.1 Generate Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "veo-3.1-generate-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "veo-3.1-generate-preview"
      ],
      "modalities": [
        "video"
      ],
      "capabilities": [
        "video_generation"
      ],
      "ui_priority": 174,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "veo-3.1-fast-generate-preview",
      "display_name": "Veo 3.1 Fast Generate Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "veo-3.1-fast-generate-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "veo-3.1-fast-generate-preview"
      ],
      "modalities": [
        "video"
      ],
      "capabilities": [
        "video_generation"
      ],
      "ui_priority": 175,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "veo-3.1-lite-generate-preview",
      "display_name": "Veo 3.1 Lite Generate Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "veo-3.1-lite-generate-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "veo-3.1-lite-generate-preview"
      ],
      "modalities": [
        "video"
      ],
      "capabilities": [
        "video_generation"
      ],
      "ui_priority": 176,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "grok-imagine-image-2.0",
      "display_name": "Grok Imagine Image 2.0",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-imagine-image-2.0"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-imagine-image-2.0"
      ],
      "modalities": [
        "text",
        "image"
      ],
      "capabilities": [
        "image_generation"
      ],
      "ui_priority": 177,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "grok-imagine-video-1.5",
      "display_name": "Grok Imagine Video 1.5",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-imagine-video-1.5"
      ],
      "aliases": [
        "grok-imagine-video-1.5-preview",
        "grok-imagine-video-1.5-2026-05-30"
      ],
      "pricing_lookup_ids": [
        "grok-imagine-video-1.5"
      ],
      "modalities": [
        "video"
      ],
      "capabilities": [
        "video_generation"
      ],
      "ui_priority": 178,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "grok-voice-latest",
      "display_name": "Grok Voice Latest",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-voice-latest"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-voice-latest"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 179,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "grok-voice-think-fast-1.0",
      "display_name": "Grok Voice Think Fast 1.0",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-voice-think-fast-1.0"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-voice-think-fast-1.0"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 180,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "grok-voice-think-fast-2.0",
      "display_name": "Grok Voice Think Fast 2.0",
      "provider": "grok",
      "platforms": [
        "grok"
      ],
      "protocol_ids": [
        "grok-voice-think-fast-2.0"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "grok-voice-think-fast-2.0"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding",
        "audio_generation"
      ],
      "ui_priority": 181,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "claude-haiku-4-5-20251001",
      "display_name": "Claude Haiku 4.5",
      "provider": "anthropic",
      "platforms": [
        "anthropic"
      ],
      "protocol_ids": [
        "claude-haiku-4-5-20251001"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-haiku-4-5-20251001"
      ],
      "context_window_tokens": 200000,
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text",
        "vision"
      ],
      "ui_priority": 200,
      "exposed_in": [
        "runtime",
        "test"
      ],
      "status": "deprecated",
      "replaced_by": "claude-haiku-4.5"
    },
    {
      "id": "claude-opus-4-5-20251101",
      "display_name": "Claude Opus 4.5",
      "provider": "anthropic",
      "platforms": [
        "anthropic"
      ],
      "protocol_ids": [
        "claude-opus-4-5-20251101"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-opus-4-5-20251101"
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
        "test"
      ],
      "status": "deprecated",
      "replaced_by": "claude-opus-4.1"
    },
    {
      "id": "claude-opus-4-5-thinking",
      "display_name": "Claude Opus 4.5 Thinking",
      "provider": "anthropic",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "claude-opus-4-5-thinking"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-opus-4-5-thinking"
      ],
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
      ],
      "status": "deprecated",
      "replaced_by": "claude-opus-4.1"
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
      "id": "claude-sonnet-4-5",
      "display_name": "Claude Sonnet 4.5",
      "provider": "anthropic",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "claude-sonnet-4-5"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-sonnet-4-5"
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
        "test"
      ],
      "status": "deprecated",
      "replaced_by": "claude-sonnet-4.5"
    },
    {
      "id": "claude-sonnet-4-5-20250929",
      "display_name": "Claude Sonnet 4.5",
      "provider": "anthropic",
      "platforms": [
        "anthropic"
      ],
      "protocol_ids": [
        "claude-sonnet-4-5-20250929"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-sonnet-4-5-20250929"
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
        "test"
      ],
      "status": "deprecated",
      "replaced_by": "claude-sonnet-4.5"
    },
    {
      "id": "claude-sonnet-4-5-thinking",
      "display_name": "Claude Sonnet 4.5 Thinking",
      "provider": "anthropic",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "claude-sonnet-4-5-thinking"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "claude-sonnet-4-5-thinking"
      ],
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
      ],
      "status": "deprecated",
      "replaced_by": "claude-sonnet-4.5"
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
      "id": "gemini-2.5-flash-image-preview",
      "display_name": "Gemini 2.5 Flash Image Preview",
      "provider": "gemini",
      "platforms": [
        "antigravity"
      ],
      "protocol_ids": [
        "gemini-2.5-flash-image-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-2.5-flash-image-preview"
      ],
      "context_window_tokens": 1048576,
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
      "capabilities": [
        "text",
        "vision",
        "web_search",
        "audio_understanding",
        "video_understanding"
      ],
      "ui_priority": 200,
      "exposed_in": [
        "runtime",
        "test"
      ],
      "status": "beta"
    },
    {
      "id": "antigravity-preview-05-2026",
      "display_name": "Antigravity Agent Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "antigravity-preview-05-2026"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "antigravity-preview-05-2026"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 300,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "deep-research-max-preview-04-2026",
      "display_name": "Deep Research Max Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "deep-research-max-preview-04-2026"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deep-research-max-preview-04-2026"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 300,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "deep-research-preview-04-2026",
      "display_name": "Deep Research Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "deep-research-preview-04-2026"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "deep-research-preview-04-2026"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [
        "text"
      ],
      "ui_priority": 300,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "gemini-3.5-transcribe-live",
      "display_name": "Gemini 3.5 Transcribe Live",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-3.5-transcribe-live"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-3.5-transcribe-live"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding"
      ],
      "ui_priority": 300,
      "exposed_in": [
        "catalog"
      ],
      "status": "stable"
    },
    {
      "id": "gemini-robotics-er-2-streaming-preview",
      "display_name": "Gemini Robotics ER 2 Streaming Preview",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "gemini-robotics-er-2-streaming-preview"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "gemini-robotics-er-2-streaming-preview"
      ],
      "modalities": [
        "audio"
      ],
      "capabilities": [
        "audio_understanding"
      ],
      "ui_priority": 300,
      "exposed_in": [
        "catalog"
      ],
      "status": "beta"
    },
    {
      "id": "unknown",
      "display_name": "Unknown",
      "provider": "gemini",
      "platforms": [
        "gemini"
      ],
      "protocol_ids": [
        "unknown"
      ],
      "aliases": [],
      "pricing_lookup_ids": [
        "unknown"
      ],
      "modalities": [
        "text"
      ],
      "capabilities": [],
      "ui_priority": 9999,
      "exposed_in": [
        "runtime"
      ]
    }
  ],
  "presets": [
    {
      "platform": "anthropic",
      "label": "Opus 5",
      "from": "claude-opus-5",
      "to": "claude-opus-5",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 1
    },
    {
      "platform": "grok",
      "label": "Grok Auto",
      "from": "grok-auto",
      "to": "grok-auto",
      "color": "bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400",
      "order": 1
    },
    {
      "platform": "anthropic",
      "label": "Opus 4.1",
      "from": "claude-opus-4.1",
      "to": "claude-opus-4.1",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 2
    },
    {
      "platform": "grok",
      "label": "Grok 4 Expert",
      "from": "grok-4-expert",
      "to": "grok-4-expert",
      "color": "bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400",
      "order": 2
    },
    {
      "platform": "anthropic",
      "label": "Sonnet 4.5",
      "from": "claude-sonnet-4.5",
      "to": "claude-sonnet-4.5",
      "color": "bg-indigo-100 text-indigo-700 hover:bg-indigo-200 dark:bg-indigo-900/30 dark:text-indigo-400",
      "order": 3
    },
    {
      "platform": "grok",
      "label": "Imagine Image",
      "from": "grok-imagine-image",
      "to": "grok-imagine-1.0",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 3
    },
    {
      "platform": "anthropic",
      "label": "Haiku 4.5",
      "from": "claude-haiku-4.5",
      "to": "claude-haiku-4.5",
      "color": "bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400",
      "order": 4
    },
    {
      "platform": "grok",
      "label": "Imagine Edit",
      "from": "grok-imagine-1.0-edit",
      "to": "grok-imagine-1.0-edit",
      "color": "bg-rose-100 text-rose-700 hover:bg-rose-200 dark:bg-rose-900/30 dark:text-rose-400",
      "order": 4
    },
    {
      "platform": "anthropic",
      "label": "Opus-\u003eSonnet",
      "from": "claude-opus-4.1",
      "to": "claude-sonnet-4.5",
      "color": "bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400",
      "order": 5
    },
    {
      "platform": "grok",
      "label": "Imagine Video",
      "from": "grok-imagine-video",
      "to": "grok-imagine-1.0-video",
      "color": "bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400",
      "order": 5
    },
    {
      "platform": "anthropic",
      "label": "Haiku-\u003eSonnet",
      "from": "claude-haiku-4.5",
      "to": "claude-sonnet-4.5",
      "color": "bg-blue-100 text-blue-700 hover:bg-blue-200 dark:bg-blue-900/30 dark:text-blue-400",
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
      "platform": "anthropic",
      "label": "Opus 4.8",
      "from": "claude-opus-4-8",
      "to": "claude-opus-4-8",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 7
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
      "platform": "anthropic",
      "label": "Opus 4.7",
      "from": "claude-opus-4-7",
      "to": "claude-opus-4-7",
      "color": "bg-purple-100 text-purple-700 hover:bg-purple-200 dark:bg-purple-900/30 dark:text-purple-400",
      "order": 8
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
      "platform": "gemini",
      "label": "3.6 Flash",
      "from": "gemini-3.6-flash",
      "to": "gemini-3.6-flash",
      "color": "bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400",
      "order": 25
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
      "label": "Opus4.5-\u003e4.1",
      "from": "claude-opus-4-5-20251101",
      "to": "claude-opus-4.1",
      "color": "bg-violet-100 text-violet-700 hover:bg-violet-200 dark:bg-violet-900/30 dark:text-violet-400",
      "order": 36
    },
    {
      "platform": "antigravity",
      "label": "3-Pro-Preview-\u003ePro-Agent",
      "from": "gemini-3-pro-preview",
      "to": "gemini-pro-agent",
      "color": "bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400",
      "order": 37
    },
    {
      "platform": "antigravity",
      "label": "3-Pro-High-\u003ePro-Agent",
      "from": "gemini-3-pro-high",
      "to": "gemini-pro-agent",
      "color": "bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-400",
      "order": 38
    },
    {
      "platform": "antigravity",
      "label": "3-Pro-Low-\u003e3.1-Pro-Low",
      "from": "gemini-3-pro-low",
      "to": "gemini-3.1-pro-low",
      "color": "bg-yellow-100 text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-400",
      "order": 39
    },
    {
      "platform": "antigravity",
      "label": "3.1-Pro-\u003ePro-Agent",
      "from": "gemini-3.1-pro",
      "to": "gemini-pro-agent",
      "color": "bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-400",
      "order": 40
    },
    {
      "platform": "antigravity",
      "label": "3.1-Pro-High-\u003ePro-Agent",
      "from": "gemini-3.1-pro-high",
      "to": "gemini-pro-agent",
      "color": "bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-400",
      "order": 41
    },
    {
      "platform": "antigravity",
      "label": "3.1-Pro-Low passthrough",
      "from": "gemini-3.1-pro-low",
      "to": "gemini-3.1-pro-low",
      "color": "bg-yellow-100 text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-400",
      "order": 42
    },
    {
      "platform": "antigravity",
      "label": "3.1-Pro-Preview-\u003ePro-Agent",
      "from": "gemini-3.1-pro-preview",
      "to": "gemini-pro-agent",
      "color": "bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400",
      "order": 43
    },
    {
      "platform": "antigravity",
      "label": "2.5-Flash-Image passthrough",
      "from": "gemini-2.5-flash-image",
      "to": "gemini-2.5-flash-image",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 44
    },
    {
      "platform": "antigravity",
      "label": "3.1-Flash-Image passthrough",
      "from": "gemini-3.1-flash-image",
      "to": "gemini-3.1-flash-image",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 45
    },
    {
      "platform": "antigravity",
      "label": "3.6-Flash passthrough",
      "from": "gemini-3.6-flash",
      "to": "gemini-3.6-flash",
      "color": "bg-emerald-100 text-emerald-700 hover:bg-emerald-200 dark:bg-emerald-900/30 dark:text-emerald-400",
      "order": 46
    },
    {
      "platform": "antigravity",
      "label": "3-Pro-Image-\u003e3.1",
      "from": "gemini-3-pro-image",
      "to": "gemini-3.1-flash-image",
      "color": "bg-sky-100 text-sky-700 hover:bg-sky-200 dark:bg-sky-900/30 dark:text-sky-400",
      "order": 47
    },
    {
      "platform": "antigravity",
      "label": "Gemini 2.5-\u003eFlash",
      "from": "gemini-2.5*",
      "to": "gemini-2.5-flash",
      "color": "bg-orange-100 text-orange-700 hover:bg-orange-200 dark:bg-orange-900/30 dark:text-orange-400",
      "order": 48
    },
    {
      "platform": "antigravity",
      "label": "Gemini 3-\u003eFlash",
      "from": "gemini-3*",
      "to": "gemini-3-flash",
      "color": "bg-yellow-100 text-yellow-700 hover:bg-yellow-200 dark:bg-yellow-900/30 dark:text-yellow-400",
      "order": 48
    },
    {
      "platform": "antigravity",
      "label": "3-Flash passthrough",
      "from": "gemini-3-flash",
      "to": "gemini-3-flash",
      "color": "bg-lime-100 text-lime-700 hover:bg-lime-200 dark:bg-lime-900/30 dark:text-lime-400",
      "order": 49
    },
    {
      "platform": "antigravity",
      "label": "2.5-Flash-Lite passthrough",
      "from": "gemini-2.5-flash-lite",
      "to": "gemini-2.5-flash-lite",
      "color": "bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400",
      "order": 50
    },
    {
      "platform": "antigravity",
      "label": "Sonnet 4.5",
      "from": "claude-sonnet-4.5",
      "to": "claude-sonnet-4.5",
      "color": "bg-cyan-100 text-cyan-700 hover:bg-cyan-200 dark:bg-cyan-900/30 dark:text-cyan-400",
      "order": 51
    },
    {
      "platform": "antigravity",
      "label": "Haiku 4.5",
      "from": "claude-haiku-4.5",
      "to": "claude-haiku-4.5",
      "color": "bg-green-100 text-green-700 hover:bg-green-200 dark:bg-green-900/30 dark:text-green-400",
      "order": 52
    },
    {
      "platform": "antigravity",
      "label": "Opus 5",
      "from": "claude-opus-5",
      "to": "claude-opus-5",
      "color": "bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400",
      "order": 53
    },
    {
      "platform": "antigravity",
      "label": "Opus 4.1",
      "from": "claude-opus-4.1",
      "to": "claude-opus-4.1",
      "color": "bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400",
      "order": 54
    },
    {
      "platform": "antigravity",
      "label": "Opus 4.7",
      "from": "claude-opus-4-7",
      "to": "claude-opus-4-7",
      "color": "bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400",
      "order": 55
    },
    {
      "platform": "antigravity",
      "label": "Opus 4.8",
      "from": "claude-opus-4-8",
      "to": "claude-opus-4-8",
      "color": "bg-pink-100 text-pink-700 hover:bg-pink-200 dark:bg-pink-900/30 dark:text-pink-400",
      "order": 56
    }
  ]
}
