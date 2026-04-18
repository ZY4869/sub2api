export const DOCS_CODE_TAB_ORDER = [
  'Python',
  'JavaScript',
  'Go',
  'Java',
  'C#',
  'PHP',
  'Shell',
  'REST',
] as const

export const DOCS_STANDALONE_CODE_LABELS = new Set<DocsCodeLabel>(DOCS_CODE_TAB_ORDER)

export type DocsCodeLabel = (typeof DOCS_CODE_TAB_ORDER)[number]

export interface DocsFenceMeta {
  focusLines: number[]
  language: string
}

export interface DocsCodeExampleTab {
  code: string
  focusLines: number[]
  id: string
  label: DocsCodeLabel
  language: string
}

export interface DocsCodeExampleGroup {
  id: string
  tabs: DocsCodeExampleTab[]
}

interface RequestTemplate {
  body: string
  headers: Array<[string, string]>
  method: string
  url: string
}

type HeaderTarget = 'go' | 'java' | 'csharp' | 'php' | 'shell'

const PLACEHOLDER_API_KEY = 'sk-your-key'

const LANGUAGE_BY_LABEL: Record<DocsCodeLabel, string> = {
  Python: 'python',
  JavaScript: 'javascript',
  Go: 'go',
  Java: 'java',
  'C#': 'csharp',
  PHP: 'php',
  Shell: 'bash',
  REST: 'rest',
}

const TAB_LABEL_ALIASES: Record<string, DocsCodeLabel> = {
  'c#': 'C#',
  csharp: 'C#',
  go: 'Go',
  java: 'Java',
  javascript: 'JavaScript',
  js: 'JavaScript',
  php: 'PHP',
  py: 'Python',
  python: 'Python',
  rest: 'REST',
  shell: 'Shell',
  sh: 'Shell',
}

export function buildCompletedCodeTabs(
  tabs: DocsCodeExampleTab[],
  groupId: string,
): DocsCodeExampleTab[] {
  const normalizedTabs = deduplicateTabs(tabs)
  const template = extractRequestTemplate(normalizedTabs)
  const completedTabs = [...normalizedTabs]

  for (const label of DOCS_CODE_TAB_ORDER) {
    if (completedTabs.some((tab) => tab.label === label)) {
      continue
    }

    completedTabs.push({
      code: generateExampleFromTemplate(label, template),
      focusLines: [],
      id: `${groupId}-${slugifyLabel(label)}`,
      label,
      language: defaultLanguageForTab(label),
    })
  }

  return completedTabs.sort(
    (left, right) =>
      DOCS_CODE_TAB_ORDER.indexOf(left.label) - DOCS_CODE_TAB_ORDER.indexOf(right.label),
  )
}

export function defaultLanguageForTab(label: DocsCodeLabel): string {
  return LANGUAGE_BY_LABEL[label]
}

export function resolveDocsCodeLanguage(label: DocsCodeLabel, language: string): string {
  const normalized = normalizeLanguageName(language)

  if (label === 'REST') {
    if (!normalized || normalized === 'bash' || normalized === 'curl' || normalized === 'http') {
      return 'rest'
    }
  }

  if (label === 'Shell' && (!normalized || normalized === 'curl' || normalized === 'http')) {
    return 'bash'
  }

  return normalized || defaultLanguageForTab(label)
}

export function inferStandaloneCodeLabel(language: string): DocsCodeLabel | null {
  const normalized = normalizeLanguageName(language)
  if (!normalized) {
    return null
  }
  if (normalized === 'bash' || normalized === 'curl' || normalized === 'shell' || normalized === 'sh') {
    return 'Shell'
  }
  if (normalized === 'http' || normalized === 'rest') {
    return 'REST'
  }

  return TAB_LABEL_ALIASES[normalized] ?? null
}

export function parseFenceMeta(infoString: string): DocsFenceMeta {
  const parts = String(infoString || '')
    .trim()
    .split(/\s+/)
    .filter(Boolean)
  const language = parts[0] ?? ''
  const focusLines = parts
    .slice(1)
    .flatMap((part) => {
      const match = part.match(/^focus=(.+)$/i)
      return match ? parseFocusRanges(match[1]) : []
    })

  return {
    focusLines,
    language,
  }
}

export function normalizeDocsTabLabel(value: string): DocsCodeLabel | null {
  return TAB_LABEL_ALIASES[String(value || '').trim().toLowerCase()] ?? null
}

function deduplicateTabs(tabs: DocsCodeExampleTab[]): DocsCodeExampleTab[] {
  const seen = new Set<DocsCodeLabel>()
  return tabs.filter((tab) => {
    if (seen.has(tab.label)) {
      return false
    }
    seen.add(tab.label)
    return true
  })
}

function parseFocusRanges(value: string): number[] {
  const lines = new Set<number>()

  for (const chunk of value.split(',')) {
    const trimmed = chunk.trim()
    if (!trimmed) {
      continue
    }

    const rangeMatch = trimmed.match(/^(\d+)-(\d+)$/)
    if (rangeMatch) {
      const start = Number(rangeMatch[1])
      const end = Number(rangeMatch[2])
      if (Number.isFinite(start) && Number.isFinite(end)) {
        for (let index = Math.min(start, end); index <= Math.max(start, end); index += 1) {
          lines.add(index)
        }
      }
      continue
    }

    const lineNumber = Number(trimmed)
    if (Number.isFinite(lineNumber) && lineNumber > 0) {
      lines.add(lineNumber)
    }
  }

  return Array.from(lines).sort((left, right) => left - right)
}

function extractRequestTemplate(tabs: DocsCodeExampleTab[]): RequestTemplate | null {
  return (
    parseRestTemplate(tabs.find((tab) => tab.label === 'REST')?.code) ??
    parseJavaScriptTemplate(tabs.find((tab) => tab.label === 'JavaScript')?.code) ??
    parsePythonTemplate(tabs.find((tab) => tab.label === 'Python')?.code)
  )
}

function parseRestTemplate(code: string | undefined): RequestTemplate | null {
  if (!code) {
    return null
  }

  const urlMatch = code.match(/https?:\/\/[^\s"'\\]+/)
  if (!urlMatch) {
    return null
  }

  const methodMatch = code.match(/-X\s+([A-Z]+)/)
  const headerMatches = Array.from(code.matchAll(/-H\s+(?:"([^"]+)"|'([^']+)')/g))
  const bodyMatch = code.match(/-(?:d|data|data-raw)\s+'([\s\S]*?)'/)

  return {
    body: bodyMatch?.[1]?.trim() ?? '',
    headers: headerMatches
      .map((match) => splitHeader(match[1] ?? match[2] ?? ''))
      .filter((entry): entry is [string, string] => !!entry),
    method: methodMatch?.[1] ?? (bodyMatch ? 'POST' : 'GET'),
    url: urlMatch[0],
  }
}

function parseJavaScriptTemplate(code: string | undefined): RequestTemplate | null {
  if (!code) {
    return null
  }

  const urlMatch = code.match(/fetch\(\s*(?:`([^`]+)`|"([^"]+)"|'([^']+)')/)
  if (!urlMatch) {
    return null
  }

  const methodMatch = code.match(/method:\s*["']([A-Z]+)["']/)
  const headersBlockMatch = code.match(/headers:\s*\{([\s\S]*?)\n\s*\}/)
  const bodyMatch = code.match(/body:\s*JSON\.stringify\(([\s\S]*?)\)\s*[,}]/)

  return {
    body: bodyMatch?.[1]?.trim() ?? '',
    headers: headersBlockMatch ? parseObjectHeaders(headersBlockMatch[1]) : [],
    method: methodMatch?.[1] ?? (bodyMatch ? 'POST' : 'GET'),
    url: urlMatch[1] ?? urlMatch[2] ?? urlMatch[3] ?? 'https://api.zyxai.de',
  }
}

function parsePythonTemplate(code: string | undefined): RequestTemplate | null {
  if (!code) {
    return null
  }

  const requestMatch = code.match(/requests\.(get|post|put|patch|delete)\(\s*(?:f?"([^"]+)"|f?'([^']+)')/)
  if (!requestMatch) {
    return null
  }

  const headersBlockMatch = code.match(/headers\s*=\s*\{([\s\S]*?)\n\s*\}/)
  const bodyMatch = code.match(/json\s*=\s*(\{[\s\S]*?\})\s*[,)]/)

  return {
    body: bodyMatch?.[1]?.trim() ?? '',
    headers: headersBlockMatch ? parseObjectHeaders(headersBlockMatch[1]) : [],
    method: requestMatch[1]?.toUpperCase() ?? 'GET',
    url: requestMatch[2] ?? requestMatch[3] ?? 'https://api.zyxai.de',
  }
}

function parseObjectHeaders(source: string): Array<[string, string]> {
  return Array.from(
    source.matchAll(/["']?([A-Za-z0-9-]+)["']?\s*:\s*([^,\n]+)/g),
  )
    .map((match) => {
      const name = match[1]?.trim()
      const rawValue = match[2]?.trim().replace(/,$/, '')
      if (!name || !rawValue) {
        return null
      }
      return [name, stripQuotePair(rawValue)] as [string, string]
    })
    .filter((entry): entry is [string, string] => !!entry)
}

function splitHeader(header: string): [string, string] | null {
  const separator = header.indexOf(':')
  if (separator <= 0) {
    return null
  }
  return [header.slice(0, separator).trim(), header.slice(separator + 1).trim()]
}

function stripQuotePair(value: string): string {
  const trimmed = value.trim()
  const quoted = trimmed.match(/^(?:[a-z]{0,2})?(["'`])([\s\S]*)\1$/i)
  if (quoted) {
    return quoted[2]
  }
  return trimmed
}

function generateExampleFromTemplate(label: DocsCodeLabel, template: RequestTemplate | null): string {
  if (!template) {
    return fallbackExample(label)
  }

  switch (label) {
    case 'Python':
    case 'JavaScript':
    case 'REST':
      return fallbackExample(label)
    case 'Go':
      return renderGoExample(template)
    case 'Java':
      return renderJavaExample(template)
    case 'C#':
      return renderCSharpExample(template)
    case 'PHP':
      return renderPhpExample(template)
    case 'Shell':
      return renderShellExample(template)
    default:
      return fallbackExample(label)
  }
}

function fallbackExample(label: DocsCodeLabel): string {
  switch (label) {
    case 'Python':
      return [
        'import requests',
        '',
        '# Fill in the request based on the protocol-specific example above.',
      ].join('\n')
    case 'JavaScript':
      return [
        '// Fill in the request based on the protocol-specific example above.',
        'const response = await fetch("https://api.zyxai.de");',
        'console.log(response.status);',
      ].join('\n')
    case 'Go':
      return [
        'package main',
        '',
        '// Fill in the request based on the protocol-specific example above.',
        'func main() {}',
      ].join('\n')
    case 'Java':
      return [
        'public class Example {',
        '  public static void main(String[] args) {',
        '    // Fill in the request based on the protocol-specific example above.',
        '  }',
        '}',
      ].join('\n')
    case 'C#':
      return [
        'using System;',
        '',
        '// Fill in the request based on the protocol-specific example above.',
        'Console.WriteLine("Configure the request from the protocol example.");',
      ].join('\n')
    case 'PHP':
      return [
        '<?php',
        '',
        '// Fill in the request based on the protocol-specific example above.',
        'echo "Configure the request from the protocol example.";',
      ].join('\n')
    case 'Shell':
      return [
        '# Fill in the request based on the protocol-specific example above.',
        `API_KEY="${PLACEHOLDER_API_KEY}"`,
      ].join('\n')
    case 'REST':
      return [
        'curl https://api.zyxai.de \\',
        '  -H "Authorization: Bearer sk-your-key"',
      ].join('\n')
    default:
      return ''
  }
}

function renderGoExample(template: RequestTemplate): string {
  const bodyLines = renderGoBody(template.body)

  return [
    'package main',
    '',
    'import (',
    '  "bytes"',
    '  "fmt"',
    '  "io"',
    '  "net/http"',
    ')',
    '',
    'func main() {',
    `  apiKey := "${PLACEHOLDER_API_KEY}"`,
    ...bodyLines,
    `  req, err := http.NewRequest("${template.method}", "${escapeDoubleQuotes(template.url)}", requestBody)`,
    '  if err != nil {',
    '    panic(err)',
    '  }',
    ...renderGoHeaders(template.headers),
    '  resp, err := (&http.Client{}).Do(req)',
    '  if err != nil {',
    '    panic(err)',
    '  }',
    '  defer resp.Body.Close()',
    '',
    '  responseBody, err := io.ReadAll(resp.Body)',
    '  if err != nil {',
    '    panic(err)',
    '  }',
    '',
    '  fmt.Println(resp.Status)',
    '  fmt.Println(string(responseBody))',
    '}',
  ].join('\n')
}

function renderJavaExample(template: RequestTemplate): string {
  const payloadLines =
    template.method === 'GET'
      ? ['    String payload = "{}";']
      : ['    String payload = """', template.body || '{}', '    """;']

  return [
    'import java.net.URI;',
    'import java.net.http.HttpClient;',
    'import java.net.http.HttpRequest;',
    'import java.net.http.HttpResponse;',
    '',
    'public class Example {',
    '  public static void main(String[] args) throws Exception {',
    `    String apiKey = "${PLACEHOLDER_API_KEY}";`,
    ...payloadLines,
    '',
    '    HttpRequest.Builder builder = HttpRequest.newBuilder()',
    `      .uri(URI.create("${escapeDoubleQuotes(template.url)}"))`,
    renderJavaMethod(template.method),
    ...renderJavaHeaders(template.headers),
    '      ;',
    '',
    '    HttpResponse<String> response = HttpClient.newHttpClient()',
    '      .send(builder.build(), HttpResponse.BodyHandlers.ofString());',
    '',
    '    System.out.println(response.statusCode());',
    '    System.out.println(response.body());',
    '  }',
    '}',
  ].join('\n')
}

function renderCSharpExample(template: RequestTemplate): string {
  const payloadLines =
    template.method === 'GET'
      ? ['var payload = "{}";', '// GET requests usually do not need a request body.']
      : ['var payload = """', template.body || '{}', '""";']

  return [
    'using System;',
    'using System.Net.Http;',
    'using System.Text;',
    '',
    'using var client = new HttpClient();',
    `using var request = new HttpRequestMessage(HttpMethod.${normalizeCSharpMethod(template.method)}, "${escapeDoubleQuotes(template.url)}");`,
    `var apiKey = "${PLACEHOLDER_API_KEY}";`,
    ...renderCSharpHeaders(template.headers, template.method),
    ...payloadLines,
    ...(template.method === 'GET'
      ? []
      : ['request.Content = new StringContent(payload, Encoding.UTF8, "application/json");']),
    '',
    'using var response = await client.SendAsync(request);',
    'var responseBody = await response.Content.ReadAsStringAsync();',
    '',
    'Console.WriteLine(response.StatusCode);',
    'Console.WriteLine(responseBody);',
  ].join('\n')
}

function renderPhpExample(template: RequestTemplate): string {
  const lines = [
    '<?php',
    '',
    `$apiKey = '${escapeSingleQuotes(PLACEHOLDER_API_KEY)}';`,
    '$headers = [',
    ...renderPhpHeaders(template.headers),
    '];',
    '',
    '$ch = curl_init();',
    'curl_setopt_array($ch, [',
    `    CURLOPT_URL => '${escapeSingleQuotes(template.url)}',`,
    `    CURLOPT_CUSTOMREQUEST => '${escapeSingleQuotes(template.method)}',`,
    '    CURLOPT_HTTPHEADER => $headers,',
    '    CURLOPT_RETURNTRANSFER => true,',
  ]

  if (template.method !== 'GET') {
    lines.push("    CURLOPT_POSTFIELDS => <<<'JSON'")
    lines.push(template.body || '{}')
    lines.push('JSON,')
  }

  lines.push(']);')
  lines.push('')
  lines.push('$response = curl_exec($ch);')
  lines.push('echo $response;')
  lines.push('curl_close($ch);')

  return lines.join('\n')
}

function renderShellExample(template: RequestTemplate): string {
  const lines = [
    `API_KEY="${PLACEHOLDER_API_KEY}"`,
    `BASE_URL="${template.url}"`,
  ]

  if (template.method !== 'GET') {
    lines.push('')
    lines.push("read -r -d '' PAYLOAD <<'JSON'")
    lines.push(template.body || '{}')
    lines.push('JSON')
  }

  const commandParts = [
    'curl "$BASE_URL"',
    ...(template.method === 'GET' ? [] : [`-X ${template.method}`]),
    ...renderShellHeaders(template.headers),
    ...(template.method === 'GET' ? [] : ['--data "$PAYLOAD"']),
  ]

  lines.push('')
  lines.push(...renderShellCommand(commandParts))

  return lines.join('\n')
}

function renderGoBody(body: string): string[] {
  if (!body || body === '{}') {
    return ['  requestBody := bytes.NewReader([]byte("{}"))']
  }

  return [
    '  payload := []byte(`' + body + '`)',
    '  requestBody := bytes.NewReader(payload)',
  ]
}

function renderGoHeaders(headers: Array<[string, string]>): string[] {
  return headers.map(([name, value]) => {
    const expression = buildHeaderValueExpression(name, value, 'go')
    return `  req.Header.Set("${escapeDoubleQuotes(name)}", ${expression})`
  })
}

function renderJavaHeaders(headers: Array<[string, string]>): string[] {
  return headers.map(([name, value]) => {
    const expression = buildHeaderValueExpression(name, value, 'java')
    return `      .header("${escapeDoubleQuotes(name)}", ${expression})`
  })
}

function renderCSharpHeaders(headers: Array<[string, string]>, method: string): string[] {
  return headers
    .filter(([name]) => !(method !== 'GET' && normalizeHeaderName(name) === 'content-type'))
    .map(([name, value]) => {
      const expression = buildHeaderValueExpression(name, value, 'csharp')
      return `request.Headers.TryAddWithoutValidation("${escapeDoubleQuotes(name)}", ${expression});`
    })
}

function renderPhpHeaders(headers: Array<[string, string]>): string[] {
  return headers.map(([name, value]) => {
    const expression = buildHeaderValueExpression(name, value, 'php')
    return isPhpExpression(expression)
      ? `    '${escapeSingleQuotes(name)}: ' . ${expression},`
      : `    '${escapeSingleQuotes(name)}: ${escapeSingleQuotes(stripQuotePair(expression))}',`
  })
}

function renderShellHeaders(headers: Array<[string, string]>): string[] {
  return headers.map(([name, value]) => {
    const headerValue = buildShellHeaderValue(name, value)
    return `-H "${escapeDoubleQuotes(name)}: ${escapeDoubleQuotes(headerValue)}"`
  })
}

function renderShellCommand(parts: string[]): string[] {
  return parts.map((part, index) => {
    const line = index === 0 ? part : `  ${part}`
    return index < parts.length - 1 ? `${line} \\` : line
  })
}

function renderJavaMethod(method: string): string {
  if (method === 'GET') {
    return '      .GET()'
  }
  return `      .method("${method}", HttpRequest.BodyPublishers.ofString(payload))`
}

function normalizeCSharpMethod(method: string): string {
  switch (method) {
    case 'DELETE':
      return 'Delete'
    case 'PATCH':
      return 'Patch'
    case 'PUT':
      return 'Put'
    case 'POST':
      return 'Post'
    case 'GET':
    default:
      return 'Get'
  }
}

function buildHeaderValueExpression(name: string, value: string, target: HeaderTarget): string {
  if (isAuthorizationHeader(name)) {
    switch (target) {
      case 'go':
      case 'java':
      case 'csharp':
        return '"Bearer " + apiKey'
      case 'php':
        return "'Bearer ' . $apiKey"
      case 'shell':
        return 'Bearer $API_KEY'
      default:
        return quoteHeaderLiteral(value)
    }
  }

  if (isApiKeyHeader(name)) {
    switch (target) {
      case 'go':
      case 'java':
      case 'csharp':
        return 'apiKey'
      case 'php':
        return '$apiKey'
      case 'shell':
        return '$API_KEY'
      default:
        return quoteHeaderLiteral(value)
    }
  }

  const literal = sanitizeLiteralHeaderValue(value)
  switch (target) {
    case 'go':
    case 'java':
    case 'csharp':
      return `"${escapeDoubleQuotes(literal)}"`
    case 'php':
      return `'${escapeSingleQuotes(literal)}'`
    case 'shell':
      return literal
    default:
      return literal
  }
}

function buildShellHeaderValue(name: string, value: string): string {
  const expression = buildHeaderValueExpression(name, value, 'shell')
  return stripQuotePair(expression)
}

function sanitizeLiteralHeaderValue(value: string): string {
  return stripQuotePair(String(value || ''))
    .replace(/\$\{?\s*api[_-]?key\s*\}?/gi, PLACEHOLDER_API_KEY)
    .replace(/\{\s*api[_-]?key\s*\}/gi, PLACEHOLDER_API_KEY)
    .replace(/\$API_KEY/g, PLACEHOLDER_API_KEY)
    .replace(/Bearer\s+sk-[^\s"'`]+/gi, `Bearer ${PLACEHOLDER_API_KEY}`)
}

function quoteHeaderLiteral(value: string): string {
  return `"${escapeDoubleQuotes(sanitizeLiteralHeaderValue(value))}"`
}

function isAuthorizationHeader(name: string): boolean {
  return normalizeHeaderName(name) === 'authorization'
}

function isApiKeyHeader(name: string): boolean {
  const normalized = normalizeHeaderName(name)
  return normalized === 'x-api-key' || normalized === 'x-goog-api-key'
}

function normalizeHeaderName(name: string): string {
  return String(name || '').trim().toLowerCase()
}

function isPhpExpression(value: string): boolean {
  return value.includes('$apiKey') || value.includes(' . ')
}

function escapeDoubleQuotes(value: string): string {
  return String(value || '').replace(/\\/g, '\\\\').replace(/"/g, '\\"')
}

function escapeSingleQuotes(value: string): string {
  return String(value || '').replace(/\\/g, '\\\\').replace(/'/g, "\\'")
}

function slugifyLabel(label: DocsCodeLabel): string {
  return label.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')
}

function normalizeLanguageName(value: string): string {
  return String(value || '')
    .trim()
    .toLowerCase()
    .replace(/[^a-z0-9#+-]/g, '')
}
