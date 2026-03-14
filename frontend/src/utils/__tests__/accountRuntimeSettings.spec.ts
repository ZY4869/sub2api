import { describe, expect, it } from 'vitest'
import {
  normalizeAccountConcurrency,
  normalizeAccountLoadFactor
} from '../accountRuntimeSettings'

describe('accountRuntimeSettings', () => {
  it('normalizes concurrency to at least one', () => {
    expect(normalizeAccountConcurrency(undefined)).toBe(1)
    expect(normalizeAccountConcurrency(0)).toBe(1)
    expect(normalizeAccountConcurrency(-1)).toBe(1)
    expect(normalizeAccountConcurrency(8)).toBe(8)
  })

  it('normalizes load factor to null when invalid', () => {
    expect(normalizeAccountLoadFactor(undefined)).toBeNull()
    expect(normalizeAccountLoadFactor(0)).toBeNull()
    expect(normalizeAccountLoadFactor(-1)).toBeNull()
    expect(normalizeAccountLoadFactor(2)).toBe(2)
  })
})
