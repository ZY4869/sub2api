import type {
  PublicModelCatalogPriceDisplay,
  PublicModelCatalogPriceEntry,
} from '@/api/meta'

export function clonePriceDisplay(display?: PublicModelCatalogPriceDisplay): PublicModelCatalogPriceDisplay {
  return {
    primary: cloneEntries(display?.primary),
    secondary: cloneEntries(display?.secondary),
  }
}

export function scalePriceDisplay(
  display: PublicModelCatalogPriceDisplay,
  ratio: number,
): PublicModelCatalogPriceDisplay {
  return {
    primary: scaleEntries(display.primary, ratio),
    secondary: scaleEntries(display.secondary, ratio),
  }
}

function cloneEntries(entries?: PublicModelCatalogPriceEntry[]): PublicModelCatalogPriceEntry[] {
  return (entries || []).map((entry) => ({ ...entry }))
}

function scaleEntries(
  entries: PublicModelCatalogPriceEntry[] | undefined,
  ratio: number,
): PublicModelCatalogPriceEntry[] {
  return (entries || []).map((entry) => {
    if (entry.supported_unpriced || entry.configured === false) {
      return { ...entry }
    }
    return {
      ...entry,
      value: Number((entry.value * ratio).toPrecision(12)),
      configured: true,
      supported_unpriced: false,
    }
  })
}
