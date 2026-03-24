import { nextTick } from 'vue'
import { beforeEach, describe, expect, it } from 'vitest'
import { useAccountViewMode } from '../useAccountViewMode'

describe('useAccountViewMode', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('reads persisted values and writes updates back to localStorage', async () => {
    localStorage.setItem('account-view-mode', 'card')
    localStorage.setItem('account-group-view-enabled', 'true')

    const { viewMode, groupViewEnabled } = useAccountViewMode()

    expect(viewMode.value).toBe('card')
    expect(groupViewEnabled.value).toBe(true)

    viewMode.value = 'table'
    groupViewEnabled.value = false
    await nextTick()

    expect(localStorage.getItem('account-view-mode')).toBe('table')
    expect(localStorage.getItem('account-group-view-enabled')).toBe('false')
  })
})
