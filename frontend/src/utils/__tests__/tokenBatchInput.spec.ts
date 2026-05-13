import { describe, expect, it } from 'vitest'
import { parseUniqueLineTokens } from '../tokenBatchInput'

describe('tokenBatchInput', () => {
  it('trims, removes empty lines and de-duplicates while preserving order', () => {
    expect(parseUniqueLineTokens('  rt_1  \n\nrt_2\r\nrt_1\n  rt_3  \nrt_2')).toEqual([
      'rt_1',
      'rt_2',
      'rt_3',
    ])
  })
})
