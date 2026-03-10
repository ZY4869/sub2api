import claudeIcon from '@/assets/model-icons/claude.png'
import chatgptIcon from '@/assets/model-icons/chatgpt.png'
import geminiIcon from '@/assets/model-icons/gemini.png'

export const MODEL_CATALOG_DEFAULT_THRESHOLD = 200000

const MODEL_ICON_MAP: Record<string, string> = {
  claude: claudeIcon,
  chatgpt: chatgptIcon,
  gemini: geminiIcon
}

export function resolveModelCatalogIcon(iconKey?: string): string | undefined {
  return iconKey ? MODEL_ICON_MAP[iconKey] : undefined
}

export function resolveModelCatalogDisplayName(model: string, displayName?: string): string {
  return displayName || model
}

export function buildModelCatalogTierDescription(threshold = MODEL_CATALOG_DEFAULT_THRESHOLD) {
  return {
    low: `<= ${threshold.toLocaleString()}`,
    high: `>= ${(threshold + 1).toLocaleString()}`
  }
}
