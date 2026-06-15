import type { ChannelMonitorRequestProtocol } from '@/api/admin/channelMonitors'

export function inferChannelMonitorProtocol(provider: string): ChannelMonitorRequestProtocol {
  if (provider === 'anthropic' || provider === 'antigravity') return 'anthropic'
  if (provider === 'gemini' || provider === 'google') return 'gemini'
  return 'openai'
}

export function parseJsonRecord(text: string, fallback: Record<string, any> = {}): Record<string, any> {
  const trimmed = String(text || '').trim()
  if (!trimmed) return fallback
  const parsed = JSON.parse(trimmed)
  if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
    return parsed as Record<string, any>
  }
  throw new Error('invalid_json_object')
}

export function parseHeaderRecord(text: string): Record<string, string> {
  const record = parseJsonRecord(text, {})
  for (const value of Object.values(record)) {
    if (typeof value !== 'string') throw new Error('invalid_header_value')
  }
  return record as Record<string, string>
}

export function normalizeMonitorModels(text: string): string[] {
  return String(text || '')
    .split(/[\s,]+/)
    .map((item) => item.trim())
    .filter(Boolean)
}

export function resolveChannelMonitorSaveErrorMessage(err: any, fallback: string): string {
  return err?.response?.data?.detail || err?.response?.data?.message || err?.message || fallback
}
