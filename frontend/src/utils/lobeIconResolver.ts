const LOBE_ICON_BASE_PATH = '/lobehub-icons-static-svg/icons'

const PROVIDER_ICON_ALIASES: Record<string, string> = {
  anthropic: 'anthropic',
  claude: 'anthropic',
  openai: 'openai',
  chatgpt: 'openai',
  codex: 'openai',
  gemini: 'google',
  gemma: 'google',
  google: 'google',
  googleai: 'google',
  aistudio: 'google',
  vertexai: 'google',
  qwen: 'alibaba',
  qwq: 'alibaba',
  alibaba: 'alibaba',
  alibabacloud: 'alibaba',
  bailian: 'alibaba',
  doubao: 'bytedance',
  bytedance: 'bytedance',
  coze: 'bytedance',
  capcut: 'bytedance',
  wenxin: 'baidu',
  ernie: 'baidu',
  baidu: 'baidu',
  baiducloud: 'baidu',
  hunyuan: 'tencent',
  tencent: 'tencent',
  tencentcloud: 'tencent',
  zhipu: 'zhipu',
  chatglm: 'zhipu',
  glm: 'zhipu',
  deepseek: 'deepseek',
  xai: 'xai',
  grok: 'xai',
  meta: 'meta',
  llama: 'meta',
  mistral: 'mistral',
  mixtral: 'mistral',
  moonshot: 'moonshot',
  kimi: 'moonshot',
  perplexity: 'perplexity',
  spark: 'spark',
  yi: 'yi',
  minimax: 'minimax',
  cohere: 'cohere',
  commanda: 'cohere',
  openrouter: 'openrouter',
  ollama: 'ollama',
  cloudflare: 'cloudflare',
  antigravity: 'antigravity',
  bedrock: 'aws',
  aws: 'aws',
  azure: 'azure',
  azureai: 'azureai',
  jina: 'jina',
  midjourney: 'midjourney',
  suno: 'suno',
  sora: 'openai',
  baichuan: 'baichuan',
  stepfun: 'stepfun',
  ai360: 'ai360'
}

type ModelIconMatcher = {
  slug: string
  test: (value: string) => boolean
}

const MODEL_ICON_MATCHERS: ModelIconMatcher[] = [
  { slug: 'sora', test: (value) => value.includes('sora') },
  { slug: 'dalle', test: (value) => value.includes('dall-e') || value.includes('dalle') || value.includes('gpt-image') },
  { slug: 'codex', test: (value) => value.includes('codex') },
  { slug: 'claude', test: (value) => value.includes('claude') },
  { slug: 'gemma', test: (value) => value.includes('gemma') },
  { slug: 'gemini', test: (value) => value.includes('gemini') || value.includes('learnlm') || value.includes('imagen') || value.includes('veo') },
  { slug: 'cogvideo', test: (value) => value.includes('cogvideo') },
  { slug: 'cogview', test: (value) => value.includes('cogview') },
  { slug: 'glmv', test: (value) => value.includes('glmv') },
  {
    slug: 'chatglm',
    test: (value) => value.includes('chatglm') || /^glm(?:[-:/\s]|$)/.test(value)
  },
  { slug: 'qwen', test: (value) => value.includes('qwen') || value.includes('qwq') },
  { slug: 'deepseek', test: (value) => value.includes('deepseek') },
  {
    slug: 'mistral',
    test: (value) =>
      value.includes('mistral') ||
      value.includes('mixtral') ||
      value.includes('codestral') ||
      value.includes('pixtral') ||
      value.includes('magistral') ||
      value.includes('voxtral')
  },
  { slug: 'meta', test: (value) => value.includes('llama') || value.includes('meta') },
  { slug: 'commanda', test: (value) => value.includes('command-a') || value.includes('command a') },
  {
    slug: 'cohere',
    test: (value) =>
      value.includes('cohere') ||
      value.includes('command-r') ||
      value.includes('aya') ||
      value.startsWith('c4ai-') ||
      value.startsWith('embed-')
  },
  { slug: 'yi', test: (value) => /^yi(?:[-\s]|$)/.test(value) || value.includes('yi-lightning') || value.includes('yi-large') },
  { slug: 'grok', test: (value) => value.includes('grok') },
  { slug: 'kimi', test: (value) => value.includes('kimi') },
  { slug: 'moonshot', test: (value) => value.includes('moonshot') },
  { slug: 'doubao', test: (value) => value.includes('doubao') },
  { slug: 'minimax', test: (value) => value.includes('minimax') || value.includes('abab') },
  { slug: 'wenxin', test: (value) => value.includes('wenxin') || value.includes('ernie') },
  { slug: 'spark', test: (value) => value.includes('spark') },
  { slug: 'hunyuan', test: (value) => value.includes('hunyuan') },
  { slug: 'perplexity', test: (value) => value.includes('perplexity') || value.includes('pplx') },
  { slug: 'midjourney', test: (value) => value.includes('midjourney') || value.startsWith('mj-') || value.startsWith('mj_') },
  { slug: 'suno', test: (value) => value.includes('suno') },
  { slug: 'ollama', test: (value) => value.includes('ollama') },
  { slug: 'jina', test: (value) => value.includes('jina') },
  { slug: 'openrouter', test: (value) => value.includes('openrouter') },
  { slug: 'baichuan', test: (value) => value.includes('baichuan') },
  { slug: 'stepfun', test: (value) => value.includes('stepfun') || value.includes('step-') || value.includes('step_') },
  { slug: 'zhipu', test: (value) => value.includes('zhipu') },
  { slug: 'xai', test: (value) => value.includes('xai') },
  { slug: 'openai', test: (value) => isOpenAIModel(value) }
]

export interface ResolveModelIconOptions {
  model?: string
  displayName?: string
  iconKey?: string
  provider?: string
}

export function normalizeLobeIconKey(value?: string | null): string {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[_\s]+/g, '-')
}

export function resolveProviderIconSlugs(value?: string | null): string[] {
  const normalized = normalizeLobeIconKey(value)
  if (!normalized) {
    return []
  }
  const alias = PROVIDER_ICON_ALIASES[normalized]
  return alias ? [alias] : [normalized]
}

export function resolveModelIconSlugs(options: ResolveModelIconOptions): string[] {
  return dedupe([
    resolveModelFamilySlug(options.model),
    resolveModelFamilySlug(options.displayName),
    resolveModelFamilySlug(options.iconKey),
    ...resolveProviderIconSlugs(options.provider)
  ])
}

export function buildLobeIconSources(slugs: string[]): string[] {
  return dedupe(
    slugs.flatMap((slug) => [
      `${LOBE_ICON_BASE_PATH}/${slug}-color.svg`,
      `${LOBE_ICON_BASE_PATH}/${slug}.svg`
    ])
  )
}

export function resolveLobeBadgeText(...values: Array<string | null | undefined>): string {
  const source = values
    .map((value) => String(value || '').trim())
    .find((value) => value.length > 0)

  if (!source) {
    return '?'
  }

  const alphanumeric = source.replace(/[^a-zA-Z0-9]/g, '').slice(0, 2)
  if (alphanumeric) {
    return alphanumeric.toUpperCase()
  }
  return source.slice(0, 1).toUpperCase()
}

function resolveModelFamilySlug(value?: string | null): string | null {
  const normalized = normalizeLobeIconKey(value)
  if (!normalized) {
    return null
  }

  for (const matcher of MODEL_ICON_MATCHERS) {
    if (matcher.test(normalized)) {
      return matcher.slug
    }
  }

  const providerAlias = PROVIDER_ICON_ALIASES[normalized]
  return providerAlias || normalized
}

function isOpenAIModel(value: string) {
  return (
    value.startsWith('gpt') ||
    value.startsWith('o1') ||
    value.startsWith('o3') ||
    value.startsWith('o4') ||
    value.includes('chatgpt') ||
    value.includes('whisper') ||
    value.includes('tts') ||
    value.includes('text-embedding') ||
    value.includes('text-moderation') ||
    value.includes('omni-moderation') ||
    value.includes('babbage') ||
    value.includes('davinci') ||
    value.includes('curie') ||
    value.includes('ada')
  )
}

function dedupe(values: Array<string | null | undefined>): string[] {
  return [...new Set(values.filter((value): value is string => Boolean(value && value.length > 0)))]
}
