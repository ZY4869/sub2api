const MODEL_DISPLAY_NAME_DATE_SUFFIX_PATTERN = /-(?:\d{8}|\d{4}-\d{2}-\d{2})(?:-[^-\s]+:\d+)?$/;
const OPENAI_REASONING_MODEL_PATTERN = /^o\d/;
const MODEL_DISPLAY_TOKEN_OVERRIDES: Record<string, string> = {
  abab: "ABAB",
  airx: "AirX",
  aya: "Aya",
  c4ai: "C4AI",
  chatglm: "ChatGLM",
  chatgpt: "ChatGPT",
  codestral: "Codestral",
  codellama: "CodeLlama",
  cogvideo: "CogVideo",
  cogview: "CogView",
  deepseek: "DeepSeek",
  distill: "Distill",
  doubao: "Doubao",
  ernie: "ERNIE",
  flash: "Flash",
  glm: "GLM",
  hunyuan: "Hunyuan",
  kimi: "Kimi",
  latest: "Latest",
  lite: "Lite",
  llama: "Llama",
  longcontext: "LongContext",
  max: "Max",
  medium: "Medium",
  mistral: "Mistral",
  mini: "Mini",
  mixtral: "Mixtral",
  moonshot: "Moonshot",
  nano: "Nano",
  online: "Online",
  open: "Open",
  oss: "OSS",
  pixtral: "Pixtral",
  plus: "Plus",
  preview: "Preview",
  pro: "Pro",
  qwen: "Qwen",
  qwq: "QwQ",
  r1: "R1",
  rag: "RAG",
  reasoner: "Reasoner",
  realtime: "Realtime",
  small: "Small",
  sonar: "Sonar",
  spark: "Spark",
  speed: "Speed",
  std: "STD",
  tab: "Tab",
  thinking: "Thinking",
  tiny: "Tiny",
  tools: "Tools",
  turbo: "Turbo",
  ultra: "Ultra",
  vision: "Vision",
  yi: "Yi",
};

export function formatModelDisplayName(modelId?: string | null): string {
  const canonical = normalizeModelDisplayNameSource(modelId);
  if (!canonical) {
    return "";
  }

  const parts = canonical.split(/[-_\s]+/).filter(Boolean);
  if (parts.length === 0) {
    return canonical;
  }

  const formattedParts: string[] = [];
  for (let index = 0; index < parts.length; index += 1) {
    const current = parts[index];
    const next = parts[index + 1];
    if (shouldMergeVersionTokens(current, next)) {
      formattedParts.push(`${current}.${next}`);
      index += 1;
      continue;
    }
    formattedParts.push(formatModelDisplayToken(current, formattedParts.length === 0));
  }

  return formattedParts.join(" ");
}

function normalizeModelDisplayNameSource(modelId?: string | null): string {
  return String(modelId || "")
    .trim()
    .toLowerCase()
    .replace(MODEL_DISPLAY_NAME_DATE_SUFFIX_PATTERN, "");
}

function shouldMergeVersionTokens(current?: string, next?: string): boolean {
  return isShortNumericToken(current) && isShortNumericToken(next);
}

function isShortNumericToken(value?: string): boolean {
  return Boolean(value && /^[0-9]{1,2}$/.test(value));
}

function formatModelDisplayToken(value: string, isFirst: boolean): string {
  const override = MODEL_DISPLAY_TOKEN_OVERRIDES[value];
  if (isFirst) {
    if (override) {
      return override;
    }
    switch (value) {
      case "claude":
        return "Claude";
      case "gpt":
        return "GPT";
      case "gemini":
        return "Gemini";
      case "codex":
        return "Codex";
      default:
        if (OPENAI_REASONING_MODEL_PATTERN.test(value)) {
          return value.toUpperCase();
        }
    }
  }

  if (override) {
    return override;
  }
  if (!/^[a-z]/.test(value)) {
    return value;
  }
  return value.charAt(0).toUpperCase() + value.slice(1);
}
