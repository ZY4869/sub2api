export function parseUniqueLineTokens(input: string): string[] {
  const seen = new Set<string>()
  const tokens: string[] = []

  for (const line of input.split(/\r?\n/)) {
    const token = line.trim()
    if (!token || seen.has(token)) continue
    seen.add(token)
    tokens.push(token)
  }

  return tokens
}
