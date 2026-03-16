import { describe, expect, it } from 'vitest'
import { pickLatestSeriesModelId } from '@/utils/pickLatestSeriesModelId'

describe('pickLatestSeriesModelId', () => {
  it('returns empty string when list is empty', () => {
    expect(pickLatestSeriesModelId([])).toBe('')
  })

  it('picks highest major/minor version', () => {
    const picked = pickLatestSeriesModelId([
      { id: 'gpt-5' },
      { id: 'gpt-5.4' },
      { id: 'gpt-5.10' },
      { id: 'gpt-4.9' }
    ])
    expect(picked).toBe('gpt-5.10')
  })

  it('prefers non-date variant when same version exists', () => {
    const picked = pickLatestSeriesModelId([
      { id: 'gpt-5.4-2026-03-05' },
      { id: 'gpt-5.4' },
      { id: 'gpt-5.3-20251211' }
    ])
    expect(picked).toBe('gpt-5.4')
  })

  it('breaks ties by shorter id when versions are equal', () => {
    const picked = pickLatestSeriesModelId([
      { id: 'ernie-4.0-8k-latest' },
      { id: 'ernie-4.0' },
      { id: 'ernie-4.0-20251211' }
    ])
    expect(picked).toBe('ernie-4.0')
  })

  it('falls back to `latest` when no version can be parsed', () => {
    const picked = pickLatestSeriesModelId([{ id: 'foo-beta' }, { id: 'foo-latest' }, { id: 'foo' }])
    expect(picked).toBe('foo-latest')
  })

  it('falls back to first item when nothing can be parsed', () => {
    const picked = pickLatestSeriesModelId([{ id: 'alpha' }, { id: 'beta' }])
    expect(picked).toBe('alpha')
  })
})

