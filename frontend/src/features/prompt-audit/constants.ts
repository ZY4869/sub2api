export const promptAuditScanners = [
  'violent',
  'non_violent_illegal_acts',
  'sexual_content_or_sexual_acts',
  'pii',
  'suicide_and_self_harm',
  'unethical_acts',
  'politically_sensitive_topics',
  'copyright_violation',
  'jailbreak'
] as const

export const promptAuditProtocols = [
  'openai_chat_completions',
  'openai_responses',
  'openai_images',
  'anthropic_messages',
  'gemini',
  'grok_media',
  'openai_responses_ws'
] as const

export const defaultGuardModel = 'sileader/qwen3guard:0.6b'

export function newEndpointId(): string {
  return `guard-${Math.random().toString(36).slice(2, 8)}`
}
