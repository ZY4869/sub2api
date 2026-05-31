import { describe, expect, it } from 'vitest'
import {
  DOCS_PAGE_ORDER,
  extractMarkdownHeadings,
  parseDocsMarkdown,
  renderMarkdownDocument,
} from '../markdownDocs'

const modelExampleMarkdown = [
  '# 模型库协议示例',
  '## common',
  '### OpenAI Responses',
  '#### REST',
  '```bash',
  'curl https://api.zyxai.de/v1/responses \\',
  '  -H "Authorization: Bearer sk-你的站内Key" \\',
  '  -H "Content-Type: application/json" \\',
  "  -d '{",
  '    "model": "gpt-5.4",',
  '    "input": "请用一句话确认网关已经联通。"',
  "  }'",
  '```',
  '## gemini',
  '### generateContent',
  '#### REST',
  '```bash',
  'curl https://api.zyxai.de/v1beta/models/gemini-2.5-pro:generateContent \\',
  '  -H "Authorization: Bearer sk-你的站内Key" \\',
  '  -H "Content-Type: application/json" \\',
  "  -d '{",
  '    "contents": [',
  '      { "role": "user", "parts": [{ "text": "用一句话介绍这个模型。" }] }',
  '    ]',
  "  }'",
  '```',
].join('\n')

const fixtureMarkdown = [
  '# API 文档中心',
  '## common',
  '### 概览',
  '这里是通用说明。',
  '## openai-native',
  '### Responses',
  '这里是 OpenAI 原生页面。',
  '```python focus=1-2',
  'print("responses")',
  'print("native")',
  '```',
  '## document-ai',
  '### 异步任务',
  '这里是独立的百度智能文档页面。',
  '#### Python',
  '```python focus=2-3',
  'print("document-ai")',
  'print("job")',
  'print("result")',
  '```',
  '#### Javascript',
  '```javascript',
  'console.log("document-ai")',
  '```',
  '#### Rest',
  '```bash',
  'curl https://api.zyxai.de/document-ai/v1/models',
  '```',
  '## openai',
  '### Chat Completions',
  '这里是 OpenAI 兼容页面。',
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

  it('parses openai-native and document-ai pages with aliases and focus metadata', () => {
    const document = parseDocsMarkdown(fixtureMarkdown)
    const openAINativePage = document.pages.find((page) => page.id === 'openai-native')
    const documentAIPage = document.pages.find((page) => page.id === 'document-ai')
    const asyncSection = documentAIPage?.sections[0]
    const tabGroup = asyncSection?.contentBlocks.find((block) => block.kind === 'code-group')

    expect(document.pages.map((page) => page.id)).toEqual(DOCS_PAGE_ORDER)
    expect(openAINativePage?.title).toBe('OpenAI 原生')
    expect(documentAIPage?.title).toBe('百度智能文档')
    expect(asyncSection?.title).toBe('异步任务')
    expect(tabGroup?.kind).toBe('code-group')
    if (tabGroup?.kind === 'code-group') {
      expect(tabGroup.group.tabs.map((tab) => tab.label)).toEqual([
        'Python',
        'JavaScript',
        'Go',
        'Java',
        'C#',
        'PHP',
        'Shell',
        'REST',
      ])
      expect(tabGroup.group.tabs[0]?.focusLines).toEqual([2, 3])
      expect(tabGroup.group.tabs[1]?.code).toContain('console.log("document-ai")')
      expect(tabGroup.group.tabs[7]?.language).toBe('rest')
    }
  })

  it('uses a neutral empty-page placeholder without sync messaging', () => {
    const document = parseDocsMarkdown('# API 文档中心\n\n## common\n### 概览\n只有通用页。')
    const documentAIPage = document.pages.find((page) => page.id === 'document-ai')
    const emptyIntro = documentAIPage?.introBlocks[0]

    expect(documentAIPage?.isMissing).toBe(true)
    expect(emptyIntro?.kind).toBe('markdown')
    if (emptyIntro?.kind === 'markdown') {
      expect(emptyIntro.html).toContain('当前协议页尚未写入内容')
      expect(emptyIntro.html).not.toContain('同步')
    }
  })

  it('parses public model example template markdown without repository docs files', () => {
    const document = parseDocsMarkdown(modelExampleMarkdown)
    const commonPage = document.pages.find((page) => page.id === 'common')
    const geminiPage = document.pages.find((page) => page.id === 'gemini')
    const commonGroup = commonPage?.sections[0]?.contentBlocks.find((block) => block.kind === 'code-group')
    const geminiGroup = geminiPage?.sections[0]?.contentBlocks.find((block) => block.kind === 'code-group')

    expect(document.title).toBe('模型库协议示例')
    expect(document.pages.map((page) => page.id)).toEqual(DOCS_PAGE_ORDER)
    expect(commonPage?.isMissing).toBe(false)
    expect(geminiPage?.isMissing).toBe(false)
    expect(document.pages.find((page) => page.id === 'openai')?.isMissing).toBe(true)
    expect(commonGroup?.kind).toBe('code-group')
    expect(geminiGroup?.kind).toBe('code-group')
    if (commonGroup?.kind === 'code-group') {
      expect(commonGroup.group.tabs.map((tab) => tab.label)).toEqual([
        'Python',
        'JavaScript',
        'Go',
        'Java',
        'C#',
        'PHP',
        'Shell',
        'REST',
      ])
      expect(commonGroup.group.tabs.find((tab) => tab.label === 'REST')?.code).toContain('/v1/responses')
    }
    if (geminiGroup?.kind === 'code-group') {
      expect(geminiGroup.group.tabs.find((tab) => tab.label === 'REST')?.code).toContain(':generateContent')
    }
  })
})
