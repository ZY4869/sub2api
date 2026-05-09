import { describe, expect, it } from 'vitest'
import { renderSafeMarkdown, sanitizeRichHtml } from '../sanitize'

describe('sanitize helpers', () => {
  it('removes scripts, inline handlers, and javascript urls from rich content', () => {
    const sanitized = sanitizeRichHtml([
      '<div onclick="alert(1)">safe</div>',
      '<script>alert(1)</script>',
      '<a href="javascript:alert(1)">bad</a>',
      '<img src="x" onerror="alert(1)" />',
    ].join(''))

    expect(sanitized).toContain('<div>safe</div>')
    expect(sanitized).not.toContain('<script')
    expect(sanitized).not.toContain('onclick=')
    expect(sanitized).not.toContain('onerror=')
    expect(sanitized).not.toContain('javascript:alert(1)')
  })

  it('renders markdown through the same sanitization chain', () => {
    const rendered = renderSafeMarkdown([
      '# Title',
      '',
      '<script>alert(1)</script>',
      '<div onclick="alert(1)">safe</div>',
      '',
      '[Safe](https://example.com)',
    ].join('\n'))

    expect(rendered).toContain('<h1>Title</h1>')
    expect(rendered).toContain('<div>safe</div>')
    expect(rendered).toContain('https://example.com')
    expect(rendered).not.toContain('<script')
    expect(rendered).not.toContain('onclick=')
  })
})
