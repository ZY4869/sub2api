import DOMPurify from 'dompurify'
import { marked } from 'marked'

marked.setOptions({
  breaks: false,
  gfm: true,
})

export function sanitizeSvg(svg: string): string {
  if (!svg) return ''
  return DOMPurify.sanitize(svg, { USE_PROFILES: { svg: true, svgFilters: true } })
}

export function sanitizeRichHtml(html: string): string {
  return DOMPurify.sanitize(String(html || ''))
}

export function renderSafeMarkdown(markdown: string): string {
  return sanitizeRichHtml(marked.parse(String(markdown || '')) as string)
}
