import { describe, expect, it } from 'vitest'
import { readdir, readFile } from 'node:fs/promises'
import { extname, join, resolve } from 'node:path'

import zh from '../locales/zh'
import en from '../locales/en'

const SRC_ROOT = resolve(process.cwd(), 'src')
const SEARCH_DIRS = ['components', 'views', 'composables', 'utils']
const VALID_EXTENSIONS = new Set(['.vue', '.ts', '.tsx', '.js', '.jsx'])

function collectKeys(tree: unknown, prefix = ''): Set<string> {
  const result = new Set<string>()
  if (!tree || typeof tree !== 'object') return result
  for (const [key, value] of Object.entries(tree as Record<string, unknown>)) {
    const nextKey = prefix ? `${prefix}.${key}` : key
    result.add(nextKey)
    if (value && typeof value === 'object' && !Array.isArray(value)) {
      for (const child of collectKeys(value, nextKey)) result.add(child)
    }
  }
  return result
}

async function collectSourceFiles(dir: string): Promise<string[]> {
  const entries = await readdir(dir, { withFileTypes: true })
  const files: string[] = []
  for (const entry of entries) {
    const absPath = join(dir, entry.name)
    if (entry.isDirectory()) {
      if (entry.name === '__tests__') continue
      files.push(...await collectSourceFiles(absPath))
      continue
    }
    if (!VALID_EXTENSIONS.has(extname(entry.name))) continue
    if (entry.name.endsWith('.spec.ts') || entry.name.endsWith('.spec.tsx') || entry.name.endsWith('.spec.vue')) continue
    files.push(absPath)
  }
  return files
}

function extractStaticKeys(code: string): string[] {
  const withoutComments = code
    .replace(/\/\*[\s\S]*?\*\//g, '')
    .replace(/(^|\s)\/\/.*$/gm, '$1')
  const pattern =
    /\b(?:t|\$t|i18n\.global\.t)\(\s*['"]([a-zA-Z0-9_.-]+)['"]\s*(?:[),])/g
  const keys: string[] = []
  let match: RegExpExecArray | null
  while ((match = pattern.exec(withoutComments))) {
    keys.push(match[1])
  }
  return keys
}

describe('i18n locale key audit', () => {
  it('keeps static translation keys present in zh and en locale trees', async () => {
    const files = await Promise.all(
      SEARCH_DIRS.map(async (dir) => {
        const absDir = join(SRC_ROOT, dir)
        const sourceFiles = await collectSourceFiles(absDir)
        return Promise.all(sourceFiles.map((path) => readFile(path, 'utf8')))
      }),
    )

    const keys = [...new Set(files.flat().flatMap(extractStaticKeys))]
    const zhKeys = collectKeys(zh)
    const enKeys = collectKeys(en)

    const missingZh = keys.filter((key) => !zhKeys.has(key))
    const missingEn = keys.filter((key) => !enKeys.has(key))

    expect(missingZh).toEqual([])
    expect(missingEn).toEqual([])
  })
})
