import type { Account, ClaudeModel } from '@/types'
import { buildAccountTestModelOptionKeyFromModel } from '@/utils/accountTestModelOptions'
import { normalizeGatewayAcceptedProtocol } from '@/utils/accountProtocolGateway'
import { normalizeProviderSlug } from '@/utils/providerLabels'

export interface AccountGatewayStoredDefault {
  provider: string
  modelId: string
}

export interface AccountGatewayCatalogTarget {
  sourceProtocol?: 'openai' | 'anthropic' | 'gemini'
  targetProvider?: string
  targetModelId?: string
}

export function ensureOpenAIOAuthGatewayTestDefaults(
  extra?: Record<string, unknown> | null
): Record<string, unknown> {
  const provider = normalizeProviderSlug(String(extra?.gateway_test_provider || ''))
  const modelId = String(extra?.gateway_test_model_id || '').trim()
  return {
    ...(extra || {}),
    gateway_test_provider: provider || 'openai',
    gateway_test_model_id: modelId || 'gpt-5.4'
  }
}

function normalizeModelID(value?: string | null): string {
  return String(value || '').trim().toLowerCase()
}

function getStoredDefault(account?: Pick<Account, 'extra'> | null): AccountGatewayStoredDefault {
  return {
    provider: normalizeProviderSlug(account?.extra?.gateway_test_provider as string | undefined),
    modelId: String(account?.extra?.gateway_test_model_id || '').trim()
  }
}

function modelMatchesStoredDefault(model: ClaudeModel | null | undefined, stored: AccountGatewayStoredDefault): boolean {
  if (!model || !stored.modelId) {
    return false
  }

  const normalizedProvider = normalizeProviderSlug(model.provider)
  if (stored.provider && normalizedProvider && stored.provider !== normalizedProvider) {
    return false
  }

  const targetModelID = normalizeModelID(stored.modelId)
  return [
    normalizeModelID(model.id),
    normalizeModelID(model.canonical_id)
  ].includes(targetModelID)
}

function findMatchingModel(models: ClaudeModel[], stored: AccountGatewayStoredDefault): ClaudeModel | null {
  return models.find((model) => modelMatchesStoredDefault(model, stored)) || null
}

export function findDefaultGatewayTestModel(
  accounts: Array<Pick<Account, 'extra'>>,
  models: ClaudeModel[]
): ClaudeModel | null {
  if (accounts.length === 0 || models.length === 0) {
    return null
  }

  if (accounts.length === 1) {
    return findMatchingModel(models, getStoredDefault(accounts[0]))
  }

  const defaults = accounts.map((account) => getStoredDefault(account))
  const first = defaults[0]
  if (!first?.provider || !first?.modelId) {
    return null
  }
  const allShared = defaults.every((current) => current.provider === first.provider && current.modelId === first.modelId)
  if (!allShared) {
    return null
  }
  return findMatchingModel(models, first)
}

export function resolveGatewayTestSelectedModelKey(
  accounts: Array<Pick<Account, 'extra'>>,
  models: ClaudeModel[],
  fallbackToFirst = true
): string {
  const defaultModel = findDefaultGatewayTestModel(accounts, models)
  if (defaultModel) {
    return buildAccountTestModelOptionKeyFromModel(defaultModel)
  }
  if (fallbackToFirst && models[0]) {
    return buildAccountTestModelOptionKeyFromModel(models[0])
  }
  return ''
}

export function resolveCatalogTargetFromModel(model?: ClaudeModel | null): AccountGatewayCatalogTarget {
  if (!model) {
    return {}
  }

  const sourceProtocol = normalizeGatewayAcceptedProtocol(model.source_protocol)
  const targetProvider = normalizeProviderSlug(model.provider)
  const targetModelId = String(model.canonical_id || model.id || '').trim()

  return {
    sourceProtocol: sourceProtocol || undefined,
    targetProvider: targetProvider || undefined,
    targetModelId: targetModelId || undefined
  }
}
