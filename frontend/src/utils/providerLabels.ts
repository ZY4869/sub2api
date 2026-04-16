import { generatedModelRegistrySnapshot } from '@/generated/modelRegistry'

const GENERATED_PROVIDER_LABELS = generatedModelRegistrySnapshot.provider_labels || {}
const GENERATED_MODEL_PROVIDERS = Array.from(
  new Set(
    (generatedModelRegistrySnapshot.models || [])
      .map((model) => normalizeProviderSlug(model.provider))
      .filter(Boolean)
  )
)

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

export function listKnownProviders(extraProviders: Array<string | null | undefined> = []): string[] {
  const providers = new Set<string>([
    ...Object.keys(GENERATED_PROVIDER_LABELS).map((provider) => normalizeProviderSlug(provider)),
    ...GENERATED_MODEL_PROVIDERS
  ])

  for (const provider of extraProviders) {
    const normalized = normalizeProviderSlug(provider)
    if (normalized) {
      providers.add(normalized)
    }
  }

  return Array.from(providers).sort((left, right) =>
    formatProviderLabel(left).localeCompare(formatProviderLabel(right))
  )
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
