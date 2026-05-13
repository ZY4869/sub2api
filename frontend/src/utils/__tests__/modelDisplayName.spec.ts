import { describe, expect, it } from 'vitest'
import { formatModelDisplayName } from '../modelDisplayName'

describe('modelDisplayName', () => {
  it('formats common model ids into readable names', () => {
    expect(formatModelDisplayName('Gemini-2.5-Pro')).toBe('Gemini 2.5 Pro')
    expect(formatModelDisplayName('claude-opus-4-6')).toBe('Claude Opus 4.6')
    expect(formatModelDisplayName('gpt-4o-mini-2026-03-05')).toBe('GPT 4o Mini')
    expect(formatModelDisplayName('tab_flash_lite_preview')).toBe('Tab Flash Lite Preview')
    expect(formatModelDisplayName('deepseek-v3')).toBe('DeepSeek V3')
    expect(formatModelDisplayName('chatgpt-4o-latest')).toBe('ChatGPT 4o Latest')
    expect(formatModelDisplayName('doubao-pro-256k')).toBe('Doubao Pro 256k')
    expect(formatModelDisplayName('mistral-small-latest')).toBe('Mistral Small Latest')
    expect(formatModelDisplayName('chatglm_turbo')).toBe('ChatGLM Turbo')
  })

  it('keeps non-version numeric suffixes split instead of turning them into decimals', () => {
    expect(formatModelDisplayName('gemini-2.5-computer-use-preview-10-2025')).toBe(
      'Gemini 2.5 Computer Use Preview 10 2025'
    )
  })
})
