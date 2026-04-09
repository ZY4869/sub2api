import { generatedModelRegistrySnapshot } from '@/generated/modelRegistry'

const GENERATED_PROVIDER_LABELS = generatedModelRegistrySnapshot.provider_labels || {}

export function normalizeProviderSlug(provider?: string | null): string {
  return String(provider || '').trim().toLowerCase()
}

function toTitleCase(value: string): string {
  return value
    .split(/[-_\s]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join('-')
}

export function getProviderLabelCatalog(): Record<string, string> {
  return { ...GENERATED_PROVIDER_LABELS }
}

export function formatProviderLabel(provider?: string | null, providerLabel?: string | null): string {
  const explicitLabel = String(providerLabel || '').trim()
  if (explicitLabel) {
    return explicitLabel
  }

  const normalized = normalizeProviderSlug(provider)
  if (!normalized) {
    return 'Unknown'
  }

  return GENERATED_PROVIDER_LABELS[normalized] || toTitleCase(normalized)
}

export function buildProviderDisplayName(options: {
  provider?: string | null
  providerLabel?: string | null
  displayName?: string | null
  fallbackId?: string | null
}): string {
  const label = formatProviderLabel(options.provider, options.providerLabel)
  const name = String(options.displayName || options.fallbackId || '').trim()

  if (!name) {
    return label
  }
  if (!label || label === 'Unknown') {
    return name
  }
  return `${label} ${name}`
}

export function buildProviderDisplaySortKey(options: {
  provider?: string | null
  providerLabel?: string | null
  displayName?: string | null
  fallbackId?: string | null
}): string {
  return buildProviderDisplayName(options).trim().toLowerCase()
}
