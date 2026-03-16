export interface ModelIdLike {
  id: string
  display_name?: string
}

interface ParsedVersion {
  major: number
  minor: number
}

function parseVersionFromId(id: string): ParsedVersion | null {
  const match = id.match(/(\d+)(?:\.(\d+))?/)
  if (!match) return null
  const major = Number.parseInt(match[1], 10)
  const minor = match[2] ? Number.parseInt(match[2], 10) : 0
  if (!Number.isFinite(major) || !Number.isFinite(minor)) return null
  return { major, minor }
}

function isDateVariant(id: string): boolean {
  return /-\d{4}-\d{2}-\d{2}\b/.test(id) || /-\d{8}\b/.test(id)
}

function compareVersionsDesc(left: ParsedVersion, right: ParsedVersion): number {
  if (left.major !== right.major) return right.major - left.major
  return right.minor - left.minor
}

export function pickLatestSeriesModelId<T extends ModelIdLike>(models: T[]): string {
  if (models.length === 0) return ''

  const withVersions = models
    .map((model) => ({ model, version: parseVersionFromId(model.id) }))
    .filter((item): item is { model: T; version: ParsedVersion } => Boolean(item.version))

  if (withVersions.length === 0) {
    const latestCandidates = models.filter((model) => model.id.toLowerCase().includes('latest'))
    if (latestCandidates.length > 0) {
      return [...latestCandidates]
        .sort((a, b) => (a.id.length - b.id.length) || a.id.localeCompare(b.id))[0].id
    }
    return models[0].id
  }

  const sortedByVersion = [...withVersions].sort((a, b) => {
    const versionResult = compareVersionsDesc(a.version, b.version)
    if (versionResult !== 0) return versionResult

    const aDate = isDateVariant(a.model.id)
    const bDate = isDateVariant(b.model.id)
    if (aDate !== bDate) return aDate ? 1 : -1

    return (a.model.id.length - b.model.id.length) || a.model.id.localeCompare(b.model.id)
  })

  return sortedByVersion[0].model.id
}

