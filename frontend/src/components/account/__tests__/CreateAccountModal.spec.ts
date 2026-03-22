import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/account/CreateAccountModal.vue'),
  'utf-8'
)

describe('CreateAccountModal', () => {
  it('uses an extra-wide dialog with local horizontal overflow protection', () => {
    expect(source).toContain('width="extra-wide"')
    expect(source).toContain('class="min-w-0 overflow-x-hidden"')
  })

  it('keeps the OAuth step indicator responsive', () => {
    expect(source).toContain('flex-col items-center gap-3 sm:w-auto sm:flex-row sm:gap-4')
    expect(source).toContain('hidden h-0.5 w-8 bg-gray-300 dark:bg-dark-600 sm:block')
    expect(source).toContain('min-w-0 break-words text-sm font-medium text-gray-700 dark:text-gray-300')
  })
})
