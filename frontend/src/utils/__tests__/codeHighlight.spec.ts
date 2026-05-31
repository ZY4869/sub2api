import { describe, expect, it } from 'vitest'
import { highlightCode } from '../codeHighlight'
import { parseDocsMarkdown } from '../markdownDocs'

const modelExampleMarkdown = [
  '# 模型库协议示例',
  '## common',
  '### OpenAI Responses',
  '#### Python',
  '```python',
  'import requests',
  '',
  'base_url = "https://api.zyxai.de"',
  'api_key = "sk-你的站内Key"',
  '',
  'response = requests.post(',
  '    f"{base_url}/v1/responses",',
  '    headers={"Authorization": f"Bearer {api_key}"},',
  '    json={"model": "gpt-5.4", "input": "ping"},',
  ')',
  '```',
  '#### REST',
  '```bash',
  'curl https://api.zyxai.de/v1/responses \\',
  '  -H "Authorization: Bearer sk-你的站内Key" \\',
  '  -H "Content-Type: application/json" \\',
  "  -d '{",
  '    "model": "gpt-5.4",',
  '    "input": "ping"',
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
  '    "contents": [{ "role": "user", "parts": [{ "text": "hello" }] }]',
  "  }'",
  '```',
].join('\n')

describe('codeHighlight', () => {
  it('applies multi-color highlighting for REST requests', () => {
    const [line] = highlightCode('curl https://api.zyxai.de/v1/responses -H "Authorization: Bearer sk-test"', 'rest')

    expect(line.html).toContain('docs-code-token-keyword')
    expect(line.html).toContain('docs-code-token-url')
    expect(line.html).toContain('docs-code-token-flag')
    expect(line.html).toContain('docs-code-token-header')
    expect(line.html).toContain('docs-code-token-string-value')
  })

  it('highlights functions and object keys for JavaScript', () => {
    const [line] = highlightCode('const response = await fetch(url, { headers: { Authorization: token } })', 'javascript')

    expect(line.html).toContain('docs-code-token-keyword')
    expect(line.html).toContain('docs-code-token-function')
    expect(line.html).toContain('docs-code-token-property')
  })

  it('renders urls inside quoted strings without leaking placeholder glyphs', () => {
    const [line] = highlightCode('base_url = "https://api.zyxai.de"', 'python')

    expect(line.html).toContain('https://api.zyxai.de')
    expect(line.html).not.toMatch(/[\uE000-\uF8FF]/)
  })

  it('does not leak placeholder glyphs across public model example code', () => {
    const document = parseDocsMarkdown(modelExampleMarkdown)
    const leakedLines: string[] = []

    const scanBlocks = (blocks: Array<{ kind: string; group?: { tabs: Array<{ code: string; language: string; id: string }> } }>, pageId: string) => {
      for (const block of blocks) {
        if (block.kind !== 'code-group' || !block.group) {
          continue
        }

        for (const tab of block.group.tabs) {
          const lines = highlightCode(tab.code, tab.language)
          lines.forEach((line, index) => {
            if (/[\uE000-\uF8FF]/.test(line.html)) {
              leakedLines.push(`${pageId}:${tab.id}:${index + 1}`)
            }
          })
        }
      }
    }

    for (const page of document.pages) {
      scanBlocks(page.introBlocks, page.id)
      for (const section of page.sections) {
        scanBlocks(section.contentBlocks, page.id)
      }
    }

    expect(leakedLines).toEqual([])
  })

  it('keeps unsupported languages readable as plain text', () => {
    const [line] = highlightCode('plain text only', 'unknown')
    expect(line.html).toContain('plain text only')
    expect(line.html).not.toContain('docs-code-token-keyword')
  })
})
