import { readFileSync } from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'
import { highlightCode } from '../codeHighlight'
import { DOCS_PAGE_ORDER, parseDocsMarkdown } from '../markdownDocs'

function readRepositoryDocsBaseline() {
  const repoRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../../')
  const docsRoot = path.join(repoRoot, 'backend/internal/service/docs')
  const parts = [
    readFileSync(path.join(docsRoot, 'index.md'), 'utf8').trimEnd(),
    ...DOCS_PAGE_ORDER.map((pageId) =>
      readFileSync(path.join(docsRoot, 'pages', `${pageId}.md`), 'utf8').trimEnd(),
    ),
  ]

  return `${parts.join('\n\n')}\n`
}

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

  it('does not leak placeholder glyphs across repository code examples', () => {
    const document = parseDocsMarkdown(readRepositoryDocsBaseline())
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
