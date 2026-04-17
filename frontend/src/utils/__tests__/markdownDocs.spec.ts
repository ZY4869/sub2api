import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import {
  DOCS_PAGE_ORDER,
  extractMarkdownHeadings,
  parseDocsMarkdown,
  renderMarkdownDocument,
} from '../markdownDocs'

const fixtureMarkdown = [
  '# API 文档中心',
  '## common',
  '### 概览',
  '这是通用说明。',
  '#### Python',
  '```python',
  'print("common")',
  '```',
  '#### JavaScript',
  '```javascript',
  'console.log("common")',
  '```',
  '#### REST',
  '```bash',
  'echo common',
  '```',
  '## openai',
  '### Responses 规则',
  '这里是 OpenAI 页面。',
  '#### Python',
  '```python',
  'print("openai")',
  '```',
  '#### JavaScript',
  '```javascript',
  'console.log("openai")',
  '```',
  '#### REST',
  '```bash',
  'echo openai',
  '```',
].join('\n')

describe('markdownDocs', () => {
  it('extracts headings with stable deduplicated ids', () => {
    const headings = extractMarkdownHeadings(`
# 总览
## 快速接入
## 快速接入
### Bearer 鉴权
`)

    expect(headings).toEqual([
      { id: '总览', text: '总览', level: 1 },
      { id: '快速接入', text: '快速接入', level: 2 },
      { id: '快速接入-2', text: '快速接入', level: 2 },
      { id: 'bearer-鉴权', text: 'Bearer 鉴权', level: 3 },
    ])
  })

  it('renders html with heading ids for toc anchors', () => {
    const document = renderMarkdownDocument(`
# API 文档

## 概览

内容
`)

    expect(document.headings[0]?.id).toBe('api-文档')
    expect(document.headings[1]?.id).toBe('概览')
    expect(document.html).toContain('<h1 id="api-文档">API 文档</h1>')
    expect(document.html).toContain('<h2 id="概览">概览</h2>')
  })

  it('parses virtual protocol pages and groups code examples into tabs', () => {
    const document = parseDocsMarkdown(fixtureMarkdown)
    const openaiPage = document.pages.find((page) => page.id === 'openai')
    const section = openaiPage?.sections[0]
    const codeGroup = section?.contentBlocks.find((block) => block.kind === 'code-group')

    expect(document.pages.map((page) => page.id)).toEqual(DOCS_PAGE_ORDER)
    expect(openaiPage?.title).toBe('OpenAI 兼容')
    expect(section?.title).toBe('Responses 规则')
    expect(codeGroup?.kind).toBe('code-group')
    if (codeGroup?.kind === 'code-group') {
      expect(codeGroup.group.tabs.map((tab) => tab.label)).toEqual(['Python', 'Javascript', 'Rest'])
      expect(codeGroup.group.tabs[1]?.code).toContain('console.log("openai")')
    }
  })

  it('keeps malformed tab headings as regular markdown content', () => {
    const document = parseDocsMarkdown([
      '# API 文档中心',
      '## common',
      '### 异常结构',
      '#### Python',
      '这里只有普通文本，没有代码块。',
    ].join('\n'))

    const section = document.pages.find((page) => page.id === 'common')?.sections[0]
    expect(section?.contentBlocks).toHaveLength(1)
    expect(section?.contentBlocks[0]?.kind).toBe('markdown')
  })

  it('keeps the repository baseline aligned with the required page set and example tabs', () => {
    const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../../')
    const source = readFileSync(
      path.join(repoRoot, 'backend/internal/service/docs/api_reference.md'),
      'utf8'
    )

    const document = parseDocsMarkdown(source)
    const commonPage = document.pages.find((page) => page.id === 'common')

    expect(document.pages.every((page) => !page.isMissing)).toBe(true)
    expect(document.pages.map((page) => page.id)).toEqual(DOCS_PAGE_ORDER)
    expect(commonPage?.sections.map((section) => section.title)).toEqual([
      '概览',
      '快速接入',
      '基础地址与认证',
      '错误响应与限流',
      '模型与路径兼容差异',
      '接入最佳实践',
      '文档同步说明',
    ])

    for (const page of document.pages) {
      const hasTabbedExamples = page.sections.some((section) =>
        section.contentBlocks.some(
          (block) =>
            block.kind === 'code-group'
            && ['Python', 'Javascript', 'Rest'].every((label) =>
              block.group.tabs.some((tab) => tab.label === label)
            )
        )
      )
      expect(hasTabbedExamples, `${page.id} should contain Python/Javascript/Rest tabs`).toBe(true)
    }
  })
})
