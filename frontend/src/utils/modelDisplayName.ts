const MODEL_DISPLAY_NAME_DATE_SUFFIX_PATTERN = /-(?:\d{8}|\d{4}-\d{2}-\d{2})(?:-[^-\s]+:\d+)?$/;
const OPENAI_REASONING_MODEL_PATTERN = /^o\d/;

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
  if (isFirst) {
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

  if (!/^[a-z]/.test(value)) {
    return value;
  }
  return value.charAt(0).toUpperCase() + value.slice(1);
}
