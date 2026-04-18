import DOMPurify from 'dompurify'

export interface HighlightedCodeLine {
  html: string
  isEmpty: boolean
}

const KEYWORDS: Record<string, string[]> = {
  bash: ['case', 'curl', 'do', 'done', 'echo', 'elif', 'else', 'export', 'fi', 'for', 'if', 'in', 'read', 'then', 'while'],
  csharp: ['await', 'class', 'Console', 'HttpClient', 'new', 'public', 'return', 'string', 'using', 'var'],
  go: ['defer', 'func', 'if', 'import', 'package', 'panic', 'return', 'var'],
  java: ['class', 'import', 'new', 'public', 'return', 'static', 'String', 'throws', 'void'],
  javascript: ['await', 'const', 'for', 'if', 'let', 'new', 'return', 'true', 'false'],
  php: ['array', 'curl_close', 'curl_exec', 'curl_init', 'echo', 'false', 'null', 'true'],
  python: ['def', 'elif', 'else', 'False', 'for', 'if', 'import', 'in', 'None', 'print', 'requests', 'return', 'True'],
  rest: ['curl'],
}

const COMMENT_PATTERNS: Record<string, RegExp[]> = {
  bash: [/(^|\s)(#.*)$/],
  csharp: [/(^|\s)(\/\/.*)$/],
  go: [/(^|\s)(\/\/.*)$/],
  java: [/(^|\s)(\/\/.*)$/],
  javascript: [/(^|\s)(\/\/.*)$/],
  php: [/(^|\s)(\/\/.*)$/, /(^|\s)(#.*)$/],
  python: [/(^|\s)(#.*)$/],
  rest: [/(^|\s)(#.*)$/],
}

const URL_PATTERN = /https?:\/\/[^\s"'`]+/g
const NUMBER_PATTERN = /\b\d+(?:\.\d+)?\b/g
const ENV_PATTERN = /\$[A-Z_][A-Z0-9_]*/g
const JSON_KEY_PATTERN = /"(?:\\.|[^"])*"(?=\s*:)|'(?:\\.|[^'])*'(?=\s*:)/g
const HTTP_METHOD_PATTERN = /\b(GET|POST|PUT|PATCH|DELETE|OPTIONS|HEAD)\b/g
const FLAG_PATTERN = /(^|\s)(--?[A-Za-z-]+)/g
const PATH_PATTERN = /(^|[\s(])((?:\/[A-Za-z0-9._:-]+)+(?:\?[^\s"'`]+)?)/g
const PROPERTY_PATTERN = /\b([A-Za-z_][A-Za-z0-9_-]*)(?=\s*:)/g
const FUNCTION_PATTERN = /\b([A-Za-z_][A-Za-z0-9_]*)\s*(?=\()/g
const RESERVED_FUNCTION_NAMES = new Set([
  ...Object.values(KEYWORDS).flat(),
  'if',
  'for',
  'while',
  'switch',
  'catch',
  'return',
])

export function highlightCode(code: string, language: string): HighlightedCodeLine[] {
  const normalizedLanguage = normalizeLanguage(language)
  const lines = String(code || '').replace(/\r\n/g, '\n').split('\n')

  return lines.map((line) => {
    const html = sanitizeHighlightedHtml(highlightLine(line, normalizedLanguage))
    return {
      html: html || '&nbsp;',
      isEmpty: line.length === 0,
    }
  })
}

function highlightLine(line: string, language: string): string {
  const store: Array<{ text: string; type: string }> = []
  let source = String(line || '')

  source = capture(source, JSON_KEY_PATTERN, 'json-key', store)
  source = capture(source, URL_PATTERN, 'url', store)
  source = capture(source, /`[^`]*`|"(?:\\.|[^"])*"|'(?:\\.|[^'])*'/g, 'string', store)

  for (const pattern of COMMENT_PATTERNS[language] ?? []) {
    source = source.replace(pattern, (match, prefix = '', comment = '') => {
      if (!comment) {
        return match
      }
      return `${prefix}${placeToken(store, 'comment', comment)}`
    })
  }

  source = escapeHtml(source)
  source = source.replace(ENV_PATTERN, '<span class="docs-code-token docs-code-token-env">$&</span>')
  source = source.replace(NUMBER_PATTERN, '<span class="docs-code-token docs-code-token-number">$&</span>')
  source = highlightStructure(source, language)
  source = applyKeywordHighlight(source, language)
  source = applyFunctionHighlight(source, language)

  return restoreTokens(source, store, language)
}

function highlightStructure(line: string, language: string): string {
  let highlighted = line

  if (language === 'rest' || language === 'bash') {
    highlighted = highlighted.replace(HTTP_METHOD_PATTERN, '<span class="docs-code-token docs-code-token-method">$1</span>')
    highlighted = highlighted.replace(
      FLAG_PATTERN,
      (_, prefix: string, flag: string) => `${prefix}<span class="docs-code-token docs-code-token-flag">${flag}</span>`,
    )
    highlighted = highlighted.replace(
      PATH_PATTERN,
      (_, prefix: string, path: string) => `${prefix}<span class="docs-code-token docs-code-token-path">${path}</span>`,
    )
  } else if (supportsObjectKeys(language)) {
    highlighted = highlighted.replace(PROPERTY_PATTERN, '<span class="docs-code-token docs-code-token-property">$1</span>')
  }

  return highlighted
}

function applyKeywordHighlight(line: string, language: string): string {
  const keywords = KEYWORDS[language] ?? []
  if (keywords.length === 0) {
    return line
  }

  const pattern = new RegExp(`\\b(${keywords.map(escapeForRegExp).join('|')})\\b`, 'g')
  return line.replace(pattern, '<span class="docs-code-token docs-code-token-keyword">$1</span>')
}

function applyFunctionHighlight(line: string, language: string): string {
  if (!['python', 'javascript', 'go', 'java', 'csharp', 'php'].includes(language)) {
    return line
  }

  return line.replace(FUNCTION_PATTERN, (match, fnName: string) => {
    if (RESERVED_FUNCTION_NAMES.has(fnName)) {
      return match
    }
    return `<span class="docs-code-token docs-code-token-function">${fnName}</span>`
  })
}

function capture(
  source: string,
  pattern: RegExp,
  type: string,
  store: Array<{ text: string; type: string }>,
): string {
  return source.replace(pattern, (match) => placeToken(store, type, match))
}

function placeToken(store: Array<{ text: string; type: string }>, type: string, text: string): string {
  const index = store.push({ text, type }) - 1
  return String.fromCharCode(0xe000 + index)
}

function restoreTokens(source: string, store: Array<{ text: string; type: string }>, language: string): string {
  return source.replace(/[\uE000-\uF8FF]/g, (placeholder) => {
    const token = store[placeholder.charCodeAt(0) - 0xe000]
    if (!token) {
      return ''
    }

    if (token.type === 'string') {
      return formatStringToken(resolveNestedTokenText(token.text, store), language)
    }

    return `<span class="docs-code-token docs-code-token-${token.type}">${escapeHtml(resolveNestedTokenText(token.text, store))}</span>`
  })
}

function resolveNestedTokenText(text: string, store: Array<{ text: string; type: string }>): string {
  return String(text || '').replace(/[\uE000-\uF8FF]/g, (placeholder) => {
    const token = store[placeholder.charCodeAt(0) - 0xe000]
    if (!token) {
      return ''
    }
    return resolveNestedTokenText(token.text, store)
  })
}

function formatStringToken(text: string, language: string): string {
  if (language === 'rest' || language === 'bash') {
    const headerToken = formatHeaderStringToken(text)
    if (headerToken) {
      return headerToken
    }
  }

  return `<span class="docs-code-token docs-code-token-string">${escapeHtml(text)}</span>`
}

function formatHeaderStringToken(text: string): string | null {
  const match = text.match(/^(['"`])([\s\S]*?)\1$/)
  if (!match) {
    return null
  }

  const [, quote, inner] = match
  const separator = inner.indexOf(':')
  if (separator <= 0) {
    return null
  }

  const headerName = inner.slice(0, separator).trim()
  const headerValue = inner.slice(separator + 1).trim()
  if (!/^[A-Za-z-]+$/.test(headerName) || !headerValue) {
    return null
  }

  return [
    '<span class="docs-code-token docs-code-token-string">',
    escapeHtml(quote),
    `<span class="docs-code-token docs-code-token-header">${escapeHtml(headerName)}</span>`,
    ': ',
    `<span class="docs-code-token docs-code-token-string-value">${escapeHtml(headerValue)}</span>`,
    escapeHtml(quote),
    '</span>',
  ].join('')
}

function sanitizeHighlightedHtml(source: string): string {
  return DOMPurify.sanitize(source, {
    ALLOWED_ATTR: ['class'],
    ALLOWED_TAGS: ['span'],
  })
}

function normalizeLanguage(language: string): string {
  const normalized = String(language || '').trim().toLowerCase()
  if (normalized === 'bash' || normalized === 'sh' || normalized === 'shell' || normalized === 'curl') {
    return 'bash'
  }
  if (normalized === 'http' || normalized === 'rest') {
    return 'rest'
  }
  if (normalized === 'c#' || normalized === 'cs' || normalized === 'csharp') {
    return 'csharp'
  }
  return normalized || 'text'
}

function supportsObjectKeys(language: string): boolean {
  return ['python', 'javascript', 'php'].includes(language)
}

function escapeHtml(source: string): string {
  return source
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

function escapeForRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
