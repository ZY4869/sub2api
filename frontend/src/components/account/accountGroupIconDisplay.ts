import type { Group } from '@/types'

export type AccountGroupIconDisplay = {
  group: Group
  abbreviation: string
  paletteClass: string
}

const PALETTE_CLASSES = [
  'border-sky-200 bg-sky-50 text-sky-700 dark:border-sky-400/25 dark:bg-sky-400/10 dark:text-sky-100',
  'border-emerald-200 bg-emerald-50 text-emerald-700 dark:border-emerald-400/25 dark:bg-emerald-400/10 dark:text-emerald-100',
  'border-violet-200 bg-violet-50 text-violet-700 dark:border-violet-400/25 dark:bg-violet-400/10 dark:text-violet-100',
  'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-400/25 dark:bg-amber-400/10 dark:text-amber-100',
  'border-rose-200 bg-rose-50 text-rose-700 dark:border-rose-400/25 dark:bg-rose-400/10 dark:text-rose-100',
]

const isAsciiWord = (value: string) => /^[a-z0-9]+$/i.test(value)

const stableHash = (value: string): number => {
  let hash = 0
  for (const char of value) {
    hash = (hash * 31 + char.charCodeAt(0)) >>> 0
  }
  return hash
}

const groupIdentity = (group: Group): string => `${group.id}:${group.name}`

export const resolveGroupAbbreviation = (name: string): string => {
  const trimmed = String(name || '').trim()
  if (!trimmed) return '#'

  const firstSegment = trimmed
    .split(/[-_/\s|:\uFF1A]+/u)
    .find((segment) => segment.trim())
    ?.trim() || trimmed
  const chars = Array.from(firstSegment.replace(/[^\p{L}\p{N}]/gu, ''))
  if (chars.length === 0) return '#'

  if (isAsciiWord(chars.join(''))) {
    return chars.slice(0, 3).join('').toUpperCase()
  }
  return chars.slice(0, 2).join('')
}

export const createAccountGroupIconDisplay = (
  groups: Group[],
): AccountGroupIconDisplay[] => {
  const abbreviations = groups.map((group) => resolveGroupAbbreviation(group.name))
  const counts = abbreviations.reduce<Record<string, number>>((acc, value) => {
    acc[value] = (acc[value] || 0) + 1
    return acc
  }, {})
  const duplicateOrdinals = new Map<string, number>()

  for (const abbreviation of Object.keys(counts)) {
    if (counts[abbreviation] <= 1) continue
    groups
      .filter((_, index) => abbreviations[index] === abbreviation)
      .map(groupIdentity)
      .sort((first, second) => first.localeCompare(second))
      .forEach((identity, index) => {
        duplicateOrdinals.set(identity, index)
      })
  }

  return groups.map((group, index) => {
    const abbreviation = abbreviations[index]
    const identity = groupIdentity(group)
    const paletteIndex = counts[abbreviation] > 1
      ? (stableHash(abbreviation) + (duplicateOrdinals.get(identity) || 0)) % PALETTE_CLASSES.length
      : index % PALETTE_CLASSES.length

    return {
      group,
      abbreviation,
      paletteClass: PALETTE_CLASSES[paletteIndex],
    }
  })
}
