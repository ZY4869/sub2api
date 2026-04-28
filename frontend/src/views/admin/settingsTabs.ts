export const settingsTabKeys = [
  'general',
  'security',
  'users',
  'gateway',
  'notification',
  'email'
] as const

export type SettingsTab = (typeof settingsTabKeys)[number]

export const settingsTabs = [
  { key: 'general' as SettingsTab, icon: 'home' as const },
  { key: 'security' as SettingsTab, icon: 'shield' as const },
  { key: 'users' as SettingsTab, icon: 'user' as const },
  { key: 'gateway' as SettingsTab, icon: 'server' as const },
  { key: 'notification' as SettingsTab, icon: 'bell' as const },
  { key: 'email' as SettingsTab, icon: 'mail' as const }
]

export function resolveSettingsTab(value: unknown): SettingsTab {
  const candidate = Array.isArray(value) ? value[0] : value
  return typeof candidate === 'string' && settingsTabKeys.includes(candidate as SettingsTab)
    ? candidate as SettingsTab
    : 'general'
}
