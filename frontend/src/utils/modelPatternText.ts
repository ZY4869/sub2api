export const parseModelPatternText = (value: string): string[] => {
  const seen = new Set<string>()
  const result: string[] = []

  for (const item of value.split(/[\n,;]+/)) {
    const trimmed = item.trim()
    if (!trimmed || seen.has(trimmed)) {
      continue
    }
    seen.add(trimmed)
    result.push(trimmed)
  }

  return result
}

export const joinModelPatternText = (values: string[] | null | undefined): string =>
  Array.isArray(values) ? values.map((item) => item.trim()).filter(Boolean).join('\n') : ''
