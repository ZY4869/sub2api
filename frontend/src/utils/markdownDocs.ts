import DOMPurify from 'dompurify'
import { marked } from 'marked'
import {
  buildCompletedCodeTabs,
  inferStandaloneCodeLabel,
  normalizeDocsTabLabel,
  parseFenceMeta,
  resolveDocsCodeLanguage,
  type DocsCodeExampleGroup,
  type DocsCodeExampleTab,
  type DocsCodeLabel,
} from './docsCodeExamples'

export type { DocsCodeExampleGroup, DocsCodeExampleTab, DocsCodeLabel }

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
  'openai-native',
  'openai',
  'anthropic',
  'gemini',
  'grok',
  'antigravity',
  'vertex-batch',
  'document-ai',
] as const

export type DocsPageId = (typeof DOCS_PAGE_ORDER)[number]

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
  description: string
  shortTitle: string
  title: string
}

export const DOCS_PAGE_META: Record<DocsPageId, DocsPageMeta> = {
  common: {
    description: '统一说明基础地址、认证优先级、错误格式、模型目录与接入建议。',
    shortTitle: '通用',
    title: '通用接入',
  },
  'openai-native': {
    description: '聚焦 Responses、Responses 子资源、长连接建议与新项目优先使用的 OpenAI 原生入口。',
    shortTitle: 'OpenAI 原生',
    title: 'OpenAI 原生',
  },
  openai: {
    description: '聚焦 chat/completions、历史别名路径，以及面向旧生态客户端的兼容接入建议。',
    shortTitle: 'OpenAI 兼容',
    title: 'OpenAI 兼容',
  },
  anthropic: {
    description: '说明 Messages、count_tokens、保留请求头与 Claude 客户端约束。',
    shortTitle: 'Claude',
    title: 'Anthropic / Claude',
  },
  gemini: {
    description: '集中展示 models、files、batches、live、authTokens 与 OpenAI compat。',
    shortTitle: 'Gemini',
    title: 'Gemini 原生',
  },
  grok: {
    description: '整理 Grok 的 Responses、聊天、图像与视频能力。',
    shortTitle: 'Grok',
    title: 'Grok',
  },
  antigravity: {
    description: '解释 Antigravity 强制前缀下的 Anthropic 与 Gemini 风格入口。',
    shortTitle: 'AG',
    title: 'Antigravity',
  },
  'vertex-batch': {
    description: '汇总 Vertex 模型动作、Batch Prediction Jobs 与 Google Batch Archive。',
    shortTitle: 'Vertex',
    title: 'Vertex / Batch',
  },
  'document-ai': {
    description: '聚焦百度智能文档接口的模型能力、直连解析、异步任务与权限限制。',
    shortTitle: '百度文档',
    title: '百度智能文档',
  },
}

const DOCS_EMPTY_PAGE_TEXT = '> 当前协议页尚未写入内容，请稍后查看或联系管理员补充。'

marked.setOptions({
  breaks: false,
  gfm: true,
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
    html: DOMPurify.sanitize(doc.body.innerHTML),
  }
}

export function parseDocsMarkdown(markdown: string): ParsedDocsDocument {
  const source = normalizeMarkdown(markdown)
  const lines = source.split('\n')
  const title = extractDocumentTitle(lines) || 'API 文档中心'
  const pageSources = collectPageSources(lines)

  return {
    title,
    pages: DOCS_PAGE_ORDER.map((pageId) => buildDocsPage(pageId, pageSources.get(pageId) ?? [])),
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
      text: normalizeHeadingText(match[2]),
    }))
    .filter((heading) => heading.text.length > 0 && heading.level <= maxLevel)
    .map((heading) => ({
      ...heading,
      id: createHeadingId(heading.text, counters),
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
  const visibleSections = filterSectionsForPage(pageId, sections)

  return {
    description: meta.description,
    id: pageId,
    introBlocks: parseContentBlocks(
      introLines.length > 0 ? introLines : [DOCS_EMPTY_PAGE_TEXT],
      `page-${pageId}`,
      pageId,
      false,
    ),
    isMissing: rawMarkdown.length === 0,
    rawMarkdown,
    sections: visibleSections,
    shortTitle: meta.shortTitle,
    title: meta.title,
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
  pageId: DocsPageId,
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
      contentBlocks: parseContentBlocks(
        currentLines,
        `section-${pageId}-${sections.length + 1}`,
        pageId,
        true,
      ),
      id: createHeadingId(currentTitle, counters),
      level: 3,
      title: currentTitle,
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
  _pageId: DocsPageId,
  completeTabs: boolean,
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
      html: renderMarkdownFragment(markdown),
      id: `${blockPrefix}-markdown-${blockIndex + 1}`,
      kind: 'markdown',
    })
    blockIndex += 1
    markdownBuffer.length = 0
  }

  while (cursor < lines.length) {
    const codeGroup = parseCodeExampleGroup(
      lines,
      cursor,
      `${blockPrefix}-code-${blockIndex + 1}`,
      completeTabs,
    )
    if (codeGroup) {
      pushMarkdownBlock()
      blocks.push({
        group: codeGroup.group,
        id: codeGroup.group.id,
        kind: 'code-group',
      })
      blockIndex += 1
      cursor = codeGroup.nextIndex
      continue
    }

    const standaloneGroup = parseStandaloneCodeGroup(lines, cursor, `${blockPrefix}-code-${blockIndex + 1}`)
    if (standaloneGroup) {
      pushMarkdownBlock()
      blocks.push({
        group: standaloneGroup.group,
        id: standaloneGroup.group.id,
        kind: 'code-group',
      })
      blockIndex += 1
      cursor = standaloneGroup.nextIndex
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
  completeTabs: boolean,
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

  return {
    group: {
      id: blockId,
      tabs: completeTabs ? buildCompletedCodeTabs(tabs, blockId) : tabs,
    },
    nextIndex: cursor,
  }
}

function parseCodeExampleTab(
  lines: string[],
  startIndex: number,
  blockId: string,
  tabIndex: number,
): { tab: DocsCodeExampleTab; nextIndex: number } | null {
  const heading = lines[startIndex]?.match(/^####\s+(.+?)\s*$/)
  if (!heading) {
    return null
  }

  const label = normalizeDocsTabLabel(heading[1])
  if (!label) {
    return null
  }

  let cursor = startIndex + 1
  while (cursor < lines.length && lines[cursor].trim() === '') {
    cursor += 1
  }

  const fence = parseFence(lines[cursor] ?? '')
  if (!fence) {
    return null
  }

  const fenceMeta = parseFenceMeta(extractFenceInfoString(lines[cursor] ?? ''))
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
    nextIndex: cursor,
    tab: {
      code: codeLines.join('\n').replace(/\n+$/g, ''),
      focusLines: fenceMeta.focusLines,
      id: `${blockId}-tab-${tabIndex + 1}`,
      label,
      language: resolveDocsCodeLanguage(label, fenceMeta.language),
    },
  }
}

function parseStandaloneCodeGroup(
  lines: string[],
  startIndex: number,
  blockId: string,
): { group: DocsCodeExampleGroup; nextIndex: number } | null {
  const fence = parseFence(lines[startIndex] ?? '')
  if (!fence) {
    return null
  }

  const fenceMeta = parseFenceMeta(extractFenceInfoString(lines[startIndex] ?? ''))
  const label = inferStandaloneCodeLabel(fenceMeta.language)
  if (!label) {
    return null
  }

  let cursor = startIndex + 1
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
    group: {
      id: blockId,
      tabs: [
        {
          code: codeLines.join('\n').replace(/\n+$/g, ''),
          focusLines: fenceMeta.focusLines,
          id: `${blockId}-tab-1`,
          label,
          language: resolveDocsCodeLanguage(label, fenceMeta.language),
        },
      ],
    },
    nextIndex: cursor,
  }
}

function normalizePageHeading(text: string): DocsPageId | null {
  const normalized = normalizeHeadingText(text).toLowerCase()
  return isDocsPageId(normalized) ? normalized : null
}

function filterSectionsForPage(pageId: DocsPageId, sections: DocsSection[]): DocsSection[] {
  if (pageId !== 'common') {
    return sections
  }

  return sections.filter((section) => {
    if (section.title.includes('Document AI') || section.title.includes('百度智能文档')) {
      return false
    }
    return !section.title.includes('文档同步说明')
  })
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

function extractFenceInfoString(line: string): string {
  const match = line.match(/^\s*(?:```+|~~~+)\s*(.*)$/)
  return match?.[1]?.trim() || ''
}

function escapeForRegExp(value: string): string {
  return value.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}
