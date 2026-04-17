import DOMPurify from 'dompurify'
import { marked } from 'marked'

export interface MarkdownHeading {
  id: string
  text: string
  level: number
}

export interface RenderedMarkdownDocument {
  html: string
  headings: MarkdownHeading[]
}

export const DOCS_PAGE_ORDER = [
  'common',
  'openai',
  'anthropic',
  'gemini',
  'grok',
  'antigravity',
  'vertex-batch',
] as const

export type DocsPageId = (typeof DOCS_PAGE_ORDER)[number]
export type DocsCodeLabel = 'Python' | 'Javascript' | 'Rest'

export interface DocsCodeExampleTab {
  id: string
  label: DocsCodeLabel
  language: string
  code: string
}

export interface DocsCodeExampleGroup {
  id: string
  tabs: DocsCodeExampleTab[]
}

export interface DocsMarkdownBlock {
  id: string
  kind: 'markdown'
  html: string
}

export interface DocsCodeBlock {
  id: string
  kind: 'code-group'
  group: DocsCodeExampleGroup
}

export type DocsContentBlock = DocsMarkdownBlock | DocsCodeBlock

export interface DocsSection {
  id: string
  title: string
  level: number
  contentBlocks: DocsContentBlock[]
}

export interface DocsPage {
  id: DocsPageId
  title: string
  shortTitle: string
  description: string
  rawMarkdown: string
  introBlocks: DocsContentBlock[]
  sections: DocsSection[]
  isMissing: boolean
}

export interface ParsedDocsDocument {
  title: string
  pages: DocsPage[]
}

export interface DocsPageMeta {
  title: string
  shortTitle: string
  description: string
}

export const DOCS_PAGE_META: Record<DocsPageId, DocsPageMeta> = {
  common: {
    title: '通用接入',
    shortTitle: '通用',
    description: '统一说明基础地址、认证优先级、错误格式、限流语义与文档同步规则。'
  },
  openai: {
    title: 'OpenAI 兼容',
    shortTitle: 'OpenAI',
    description: '聚焦 Responses、Chat Completions、图像与视频入口，以及兼容别名路径。'
  },
  anthropic: {
    title: 'Anthropic / Claude',
    shortTitle: 'Claude',
    description: '说明 Messages、count_tokens、保留头透传，以及 Claude 风格客户端的接入约束。'
  },
  gemini: {
    title: 'Gemini 原生',
    shortTitle: 'Gemini',
    description: '集中展示 models、files、upload/download、batches、live 与 OpenAI compat。'
  },
  grok: {
    title: 'Grok',
    shortTitle: 'Grok',
    description: '整理聊天、responses、图像、视频等 Grok 专用或仅在 Grok 平台生效的能力。'
  },
  antigravity: {
    title: 'Antigravity',
    shortTitle: 'AG',
    description: '解释 Antigravity 前缀下的 Anthropic 风格入口、Gemini 风格入口与能力边界。'
  },
  'vertex-batch': {
    title: 'Vertex / Batch',
    shortTitle: 'Vertex',
    description: '汇总 Vertex Batch Prediction Jobs 与 Google Batch Archive 的特殊调用方式。'
  }
}

const DOCS_TAB_LABELS: readonly DocsCodeLabel[] = ['Python', 'Javascript', 'Rest']
const DOCS_EMPTY_PAGE_TEXT = '> 当前协议页尚未写入内容，请在管理页补齐对应章节。'

marked.setOptions({
  gfm: true,
  breaks: false
})

export function renderMarkdownDocument(markdown: string): RenderedMarkdownDocument {
  const source = normalizeMarkdown(markdown)
  const headings = extractMarkdownHeadings(source)
  const doc = new DOMParser().parseFromString(renderMarkdownFragment(source), 'text/html')
  const headingElements = Array.from(doc.body.querySelectorAll('h1, h2, h3, h4, h5, h6'))

  headingElements.forEach((element, index) => {
    const heading = headings[index]
    if (heading) {
      element.id = heading.id
    }
  })

  return {
    headings,
    html: DOMPurify.sanitize(doc.body.innerHTML)
  }
}

export function parseDocsMarkdown(markdown: string): ParsedDocsDocument {
  const source = normalizeMarkdown(markdown)
  const lines = source.split('\n')
  const title = extractDocumentTitle(lines) || 'API 文档中心'
  const pageSources = collectPageSources(lines)

  return {
    title,
    pages: DOCS_PAGE_ORDER.map((pageId) => buildDocsPage(pageId, pageSources.get(pageId) ?? []))
  }
}

export function extractMarkdownHeadings(markdown: string, maxLevel: number = 4): MarkdownHeading[] {
  const counters = new Map<string, number>()

  return normalizeMarkdown(markdown)
    .split('\n')
    .map((line) => line.match(/^(#{1,6})\s+(.*)$/))
    .filter((match): match is RegExpMatchArray => !!match)
    .map((match) => ({
      level: match[1].length,
      text: normalizeHeadingText(match[2])
    }))
    .filter((heading) => heading.text.length > 0 && heading.level <= maxLevel)
    .map((heading) => ({
      ...heading,
      id: createHeadingId(heading.text, counters)
    }))
}

export function isDocsPageId(value: string | null | undefined): value is DocsPageId {
  return !!value && DOCS_PAGE_ORDER.includes(value as DocsPageId)
}

export function normalizeDocsPageId(value: string | null | undefined): DocsPageId {
  return isDocsPageId(value) ? value : 'common'
}

function normalizeMarkdown(markdown: string): string {
  return String(markdown || '').replace(/^\uFEFF/, '').replace(/\r\n/g, '\n')
}

function buildDocsPage(pageId: DocsPageId, sourceLines: string[]): DocsPage {
  const meta = DOCS_PAGE_META[pageId]
  const counters = new Map<string, number>()
  const rawMarkdown = sourceLines.join('\n').trim()
  const { introLines, sections } = collectSections(sourceLines, counters, pageId)

  return {
    id: pageId,
    title: meta.title,
    shortTitle: meta.shortTitle,
    description: meta.description,
    rawMarkdown,
    introBlocks: parseContentBlocks(
      introLines.length > 0 ? introLines : [DOCS_EMPTY_PAGE_TEXT],
      `page-${pageId}`,
      pageId
    ),
    sections,
    isMissing: rawMarkdown.length === 0
  }
}

function extractDocumentTitle(lines: string[]): string {
  for (const line of lines) {
    const match = line.match(/^#\s+(.+)$/)
    if (match) {
      return normalizeHeadingText(match[1])
    }
  }

  return ''
}

function collectPageSources(lines: string[]): Map<DocsPageId, string[]> {
  const pages = new Map<DocsPageId, string[]>()
  let currentPageId: DocsPageId | null = null
  let inFence = false
  let fenceMarker = ''

  for (const line of lines) {
    const fence = parseFence(line)
    if (fence) {
      if (!inFence) {
        inFence = true
        fenceMarker = fence
      } else if (fence === fenceMarker) {
        inFence = false
        fenceMarker = ''
      }
    }

    if (!inFence) {
      const pageMatch = line.match(/^##\s+(.+)$/)
      const pageId = pageMatch ? normalizePageHeading(pageMatch[1]) : null
      if (pageId) {
        currentPageId = pageId
        if (!pages.has(pageId)) {
          pages.set(pageId, [])
        }
        continue
      }
    }

    if (currentPageId) {
      pages.get(currentPageId)?.push(line)
    }
  }

  return pages
}

function collectSections(
  lines: string[],
  counters: Map<string, number>,
  pageId: DocsPageId
): { introLines: string[]; sections: DocsSection[] } {
  const introLines: string[] = []
  const sections: DocsSection[] = []
  let currentTitle = ''
  let currentLines: string[] = []
  let collectingIntro = true
  let inFence = false
  let fenceMarker = ''

  const pushCurrentSection = () => {
    if (!currentTitle) {
      return
    }

    sections.push({
      id: createHeadingId(currentTitle, counters),
      title: currentTitle,
      level: 3,
      contentBlocks: parseContentBlocks(
        currentLines,
        `section-${pageId}-${sections.length + 1}`,
        pageId
      )
    })
  }

  for (const line of lines) {
    const fence = parseFence(line)
    if (fence) {
      if (!inFence) {
        inFence = true
        fenceMarker = fence
      } else if (fence === fenceMarker) {
        inFence = false
        fenceMarker = ''
      }
    }

    if (!inFence) {
      const sectionMatch = line.match(/^###\s+(.+)$/)
      if (sectionMatch) {
        if (collectingIntro) {
          collectingIntro = false
        } else {
          pushCurrentSection()
        }
        currentTitle = normalizeHeadingText(sectionMatch[1])
        currentLines = []
        continue
      }
    }

    if (collectingIntro) {
      introLines.push(line)
    } else {
      currentLines.push(line)
    }
  }

  pushCurrentSection()

  return { introLines, sections }
}

function parseContentBlocks(
  lines: string[],
  blockPrefix: string,
  pageId?: DocsPageId
): DocsContentBlock[] {
  const blocks: DocsContentBlock[] = []
  const markdownBuffer: string[] = []
  let cursor = 0
  let blockIndex = 0

  const pushMarkdownBlock = () => {
    const markdown = markdownBuffer.join('\n').trim()
    if (!markdown) {
      markdownBuffer.length = 0
      return
    }

    blocks.push({
      id: `${blockPrefix}-markdown-${blockIndex + 1}`,
      kind: 'markdown',
      html: renderMarkdownFragment(markdown)
    })
    blockIndex += 1
    markdownBuffer.length = 0
  }

  while (cursor < lines.length) {
    const codeGroup = parseCodeExampleGroup(
      lines,
      cursor,
      `${blockPrefix}-code-${blockIndex + 1}`,
      pageId
    )

    if (codeGroup) {
      pushMarkdownBlock()
      blocks.push({
        id: codeGroup.group.id,
        kind: 'code-group',
        group: codeGroup.group
      })
      blockIndex += 1
      cursor = codeGroup.nextIndex
      continue
    }

    markdownBuffer.push(lines[cursor])
    cursor += 1
  }

  pushMarkdownBlock()
  return blocks
}

function parseCodeExampleGroup(
  lines: string[],
  startIndex: number,
  blockId: string,
  pageId?: DocsPageId
): { group: DocsCodeExampleGroup; nextIndex: number } | null {
  const firstTab = parseCodeExampleTab(lines, startIndex, blockId, 0)
  if (!firstTab) {
    return null
  }

  const tabs: DocsCodeExampleTab[] = [firstTab.tab]
  let cursor = firstTab.nextIndex
  let tabIndex = 1

  while (true) {
    const nextTab = parseCodeExampleTab(lines, cursor, blockId, tabIndex)
    if (!nextTab) {
      break
    }
    tabs.push(nextTab.tab)
    cursor = nextTab.nextIndex
    tabIndex += 1
  }

  if (pageId) {
    const missingLabels = DOCS_TAB_LABELS.filter((label) => !tabs.some((tab) => tab.label === label))
    for (const label of missingLabels) {
      tabs.push({
        id: `${blockId}-${label.toLowerCase()}`,
        label,
        language: defaultLanguageForTab(label),
        code: notApplicableExample(pageId, label)
      })
    }
  }

  tabs.sort((left, right) => DOCS_TAB_LABELS.indexOf(left.label) - DOCS_TAB_LABELS.indexOf(right.label))

  return {
    group: {
      id: blockId,
      tabs
    },
    nextIndex: cursor
  }
}

function parseCodeExampleTab(
  lines: string[],
  startIndex: number,
  blockId: string,
  tabIndex: number
): { tab: DocsCodeExampleTab; nextIndex: number } | null {
  const heading = lines[startIndex]?.match(/^####\s+(Python|JavaScript|REST)\s*$/i)
  if (!heading) {
    return null
  }

  const label = normalizeTabLabel(heading[1])
  let cursor = startIndex + 1

  while (cursor < lines.length && lines[cursor].trim() === '') {
    cursor += 1
  }

  const fence = parseFence(lines[cursor] ?? '')
  if (!fence) {
    return null
  }

  const infoString = extractFenceInfo(lines[cursor] ?? '')
  cursor += 1

  const codeLines: string[] = []
  while (cursor < lines.length) {
    const line = lines[cursor]
    if (matchesFence(line, fence)) {
      cursor += 1
      break
    }
    codeLines.push(line)
    cursor += 1
  }

  while (cursor < lines.length && lines[cursor].trim() === '') {
    cursor += 1
  }

  return {
    tab: {
      id: `${blockId}-tab-${tabIndex + 1}`,
      label,
      language: infoString || defaultLanguageForTab(label),
      code: codeLines.join('\n').replace(/\n+$/g, '')
    },
    nextIndex: cursor
  }
}

function normalizeTabLabel(value: string): DocsCodeLabel {
  const normalized = value.trim().toLowerCase()
  if (normalized === 'javascript') {
    return 'Javascript'
  }
  if (normalized === 'rest') {
    return 'Rest'
  }
  return 'Python'
}

function defaultLanguageForTab(label: DocsCodeLabel): string {
  switch (label) {
    case 'Python':
      return 'python'
    case 'Javascript':
      return 'javascript'
    case 'Rest':
      return 'bash'
  }
}

function notApplicableExample(pageId: DocsPageId, label: DocsCodeLabel): string {
  switch (label) {
    case 'Javascript':
      return [
        `// ${DOCS_PAGE_META[pageId].title}`,
        `// 当前协议暂不提供 ${label} 示例。`,
        '// 如需补充，请同步更新仓库中的 api_reference.md 基线文档。'
      ].join('\n')
    case 'Rest':
      return [
        `# ${DOCS_PAGE_META[pageId].title}`,
        `# 当前协议暂不提供 ${label} 示例。`,
        '# 如需补充，请同步更新仓库中的 api_reference.md 基线文档。'
      ].join('\n')
    case 'Python':
    default:
      return [
        `# ${DOCS_PAGE_META[pageId].title}`,
        `# 当前协议暂不提供 ${label} 示例。`,
        '# 如需补充，请同步更新仓库中的 api_reference.md 基线文档。'
      ].join('\n')
  }
}

function normalizePageHeading(text: string): DocsPageId | null {
  const normalized = normalizeHeadingText(text).toLowerCase()
  return isDocsPageId(normalized) ? normalized : null
}

function renderMarkdownFragment(markdown: string): string {
  return DOMPurify.sanitize(marked.parse(String(markdown || '')) as string)
}

function normalizeHeadingText(text: string): string {
  return text
    .replace(/\[(.*?)\]\((.*?)\)/g, '$1')
    .replace(/`([^`]+)`/g, '$1')
    .replace(/\*\*(.*?)\*\*/g, '$1')
    .replace(/\*(.*?)\*/g, '$1')
    .replace(/~~(.*?)~~/g, '$1')
    .replace(/#+$/g, '')
    .trim()
}

function createHeadingId(text: string, counters: Map<string, number>): string {
  const base = slugify(text)
  const count = counters.get(base) ?? 0
  counters.set(base, count + 1)
  return count === 0 ? base : `${base}-${count + 1}`
}

function slugify(text: string): string {
  const normalized = text
    .toLowerCase()
    .trim()
    .replace(/[^\p{L}\p{N}\s-]/gu, '')
    .replace(/\s+/g, '-')
    .replace(/-+/g, '-')
    .replace(/^-|-$/g, '')

  return normalized || 'section'
}

function parseFence(line: string): string {
  const match = line.match(/^\s*(```+|~~~+)/)
  return match ? match[1] : ''
}

function matchesFence(line: string, fence: string): boolean {
  return new RegExp(`^\\s*${escapeForRegExp(fence)}\\s*$`).test(line)
}

function extractFenceInfo(line: string): string {
  const match = line.match(/^\s*(```+|~~~+)\s*([^\s]+)?/)
  return match?.[2]?.trim() || ''
}

function escapeForRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
