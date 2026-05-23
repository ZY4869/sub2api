import { nextTick, reactive } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAccountVisualStylePreference } from '../useAccountVisualStylePreference'

const mockState = vi.hoisted(() => ({
  showError: vi.fn(),
  updateProfile: vi.fn(),
  setCurrentUser: vi.fn(),
  setAccountVisualPresetOverride: vi.fn(),
  user: null as any,
  visualPresetDefault: 'classic',
}))

vi.mock('@/api', () => ({
  userAPI: {
    updateProfile: mockState.updateProfile,
  },
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: mockState.showError,
    visualPresetDefault: mockState.visualPresetDefault,
  }),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: mockState.user,
    setCurrentUser: mockState.setCurrentUser,
    setAccountVisualPresetOverride: mockState.setAccountVisualPresetOverride,
  }),
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

describe('useAccountVisualStylePreference', () => {
  beforeEach(() => {
    mockState.showError.mockReset()
    mockState.updateProfile.mockReset()
    mockState.setCurrentUser.mockReset()
    mockState.visualPresetDefault = 'classic'
    mockState.user = reactive({
      id: 1,
      username: 'alice',
      email: 'alice@example.com',
      role: 'user',
      visual_preset_preference: 'inherit' as const,
      account_visual_preset_override: 'inherit' as const,
    })
    mockState.setAccountVisualPresetOverride.mockReset().mockImplementation((preset: 'inherit' | 'classic' | 'airy') => {
      mockState.user.account_visual_preset_override = preset
    })
  })

  it('normalizes the current user override to inherit by default', () => {
    mockState.user.account_visual_preset_override = undefined

    const { accountVisualPresetOverride, resolvedAccountVisualPreset } = useAccountVisualStylePreference()

    expect(accountVisualPresetOverride.value).toBe('inherit')
    expect(resolvedAccountVisualPreset.value).toBe('classic')
  })

  it('optimistically updates and then syncs from the server response', async () => {
    mockState.updateProfile.mockImplementation(async () => {
      await nextTick()
      return {
        ...mockState.user,
        account_visual_preset_override: 'airy',
      }
    })

    const {
      accountVisualPresetOverride,
      resolvedAccountVisualPreset,
      setAccountVisualPresetOverride,
      updatingAccountVisualStyle,
    } =
      useAccountVisualStylePreference()

    const pending = setAccountVisualPresetOverride('airy')

    expect(accountVisualPresetOverride.value).toBe('airy')
    expect(resolvedAccountVisualPreset.value).toBe('airy')
    expect(updatingAccountVisualStyle.value).toBe(true)
    expect(mockState.setAccountVisualPresetOverride).toHaveBeenCalledWith('airy')
    expect(mockState.updateProfile).toHaveBeenCalledWith({
      account_visual_preset_override: 'airy',
    })

    await pending

    expect(mockState.setCurrentUser).toHaveBeenCalledWith(
      expect.objectContaining({
        account_visual_preset_override: 'airy',
      }),
    )
    expect(updatingAccountVisualStyle.value).toBe(false)
  })

  it('keeps the submitted override when a stale profile response contains the previous value', async () => {
    mockState.updateProfile.mockResolvedValueOnce({
      ...mockState.user,
      account_visual_preset_override: 'inherit',
    })

    const { resolvedAccountVisualPreset, setAccountVisualPresetOverride } =
      useAccountVisualStylePreference()

    await setAccountVisualPresetOverride('airy')

    expect(mockState.setCurrentUser).toHaveBeenCalledWith(
      expect.objectContaining({
        account_visual_preset_override: 'airy',
      }),
    )
    expect(resolvedAccountVisualPreset.value).toBe('airy')
  })

  it('rolls back to the previous preset override and reports the save error', async () => {
    mockState.updateProfile.mockRejectedValueOnce(new Error('save failed'))

    const { accountVisualPresetOverride, setAccountVisualPresetOverride } =
      useAccountVisualStylePreference()

    await expect(setAccountVisualPresetOverride('airy')).rejects.toThrow('save failed')

    expect(mockState.setAccountVisualPresetOverride).toHaveBeenNthCalledWith(1, 'airy')
    expect(mockState.setAccountVisualPresetOverride).toHaveBeenNthCalledWith(2, 'inherit')
    expect(accountVisualPresetOverride.value).toBe('inherit')
    expect(mockState.showError).toHaveBeenCalledWith('save failed')
    expect(mockState.setCurrentUser).not.toHaveBeenCalled()
  })
})
