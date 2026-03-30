import type { ClaudeModel } from '@/types'
import { normalizeGatewayAcceptedProtocol } from '@/utils/accountProtocolGateway'

export function buildAccountTestModelOptionKey(modelId?: unknown, sourceProtocol?: unknown): string {
  return `${normalizeGatewayAcceptedProtocol(sourceProtocol) || 'default'}::${String(modelId || '').trim()}`
}

export function buildAccountTestModelOptionKeyFromModel(
  model?: Pick<ClaudeModel, 'id' | 'source_protocol'> | null
): string {
  if (!model) {
    return ''
  }
  return buildAccountTestModelOptionKey(model.id, model.source_protocol)
}

export function findAccountTestModelByKey(models: ClaudeModel[], key: string): ClaudeModel | null {
  return models.find((model) => buildAccountTestModelOptionKeyFromModel(model) === key) || null
}
