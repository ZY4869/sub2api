import { defineComponent, h, reactive } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAccountsRuntimeSummary } from '@/composables/useAccountsRuntimeSummary'
import type { AccountListRequestParams } from '@/utils/accountListSync'

const intervalMocks = vi.hoisted(() => ({
  pause: vi.fn(),
  resume: vi.fn()
}))

const apiMocks = vi.hoisted(() => ({
  getRuntimeSummaryWithEtag: vi.fn()
}))

vi.mock('@vueuse/core', () => ({
  useIntervalFn: vi.fn(() => ({
    pause: intervalMocks.pause,
    resume: intervalMocks.resume
  }))
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      getRuntimeSummaryWithEtag: apiMocks.getRuntimeSummaryWithEtag
    }
  }
}))

const setDocumentHidden = (hidden: boolean) => {
  Object.defineProperty(document, 'hidden', {
    configurable: true,
    get: () => hidden
  })
}

function mountRuntimeSummary(options: {
  enabled?: boolean
  onSummaryChanged?: (next: { in_use: number }, previous: { in_use: number }) => void | Promise<void>
} = {}) {
  const params = reactive<AccountListRequestParams>({
    platform: '',
    type: '',
    group: '',
    privacy_mode: 'private',
    search: '',
    lifecycle: 'normal',
    limited_view: 'all',
    limited_reason: '',
    runtime_view: 'all'
  })

  let exposed: ReturnType<typeof useAccountsRuntimeSummary> | null = null

  const wrapper = mount(defineComponent({
    setup() {
      exposed = useAccountsRuntimeSummary(params, {
        enabled: options.enabled ?? true,
        onSummaryChanged: options.onSummaryChanged
      })
      return () => h('div')
    }
  }))

  return {
    wrapper,
    params,
    get exposed() {
      if (!exposed) {
        throw new Error('runtime summary composable not exposed')
      }
      return exposed
    }
  }
}

describe('useAccountsRuntimeSummary', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setDocumentHidden(false)
  })

  it('loads runtime summary immediately and updates in-use count', async () => {
    apiMocks.getRuntimeSummaryWithEtag.mockResolvedValue({
      notModified: false,
      etag: '"etag-1"',
      data: {
        in_use: 4
      }
    })

    const { exposed, wrapper } = mountRuntimeSummary()
    await Promise.resolve()
    await Promise.resolve()

    expect(apiMocks.getRuntimeSummaryWithEtag).toHaveBeenCalledWith(
      expect.objectContaining({ runtime_view: 'all', privacy_mode: 'private' }),
      { etag: null }
    )
    expect(exposed.summary.value.in_use).toBe(4)
    wrapper.unmount()
  })

  it('pauses polling when document becomes hidden and resumes with a forced refresh when visible again', async () => {
    apiMocks.getRuntimeSummaryWithEtag.mockResolvedValue({
      notModified: false,
      etag: '"etag-1"',
      data: {
        in_use: 1
      }
    })

    const { wrapper } = mountRuntimeSummary()
    await Promise.resolve()
    await Promise.resolve()

    setDocumentHidden(true)
    document.dispatchEvent(new Event('visibilitychange'))

    expect(intervalMocks.pause).toHaveBeenCalled()

    setDocumentHidden(false)
    document.dispatchEvent(new Event('visibilitychange'))
    await Promise.resolve()
    await Promise.resolve()

    expect(intervalMocks.resume).toHaveBeenCalled()
    expect(apiMocks.getRuntimeSummaryWithEtag).toHaveBeenLastCalledWith(
      expect.objectContaining({ runtime_view: 'all', privacy_mode: 'private' }),
      { etag: null }
    )
    wrapper.unmount()
  })

  it('calls onSummaryChanged when in-use count changes', async () => {
    const onSummaryChanged = vi.fn()
    apiMocks.getRuntimeSummaryWithEtag
      .mockResolvedValueOnce({
        notModified: false,
        etag: '"etag-1"',
        data: {
          in_use: 2
        }
      })
      .mockResolvedValueOnce({
        notModified: false,
        etag: '"etag-2"',
        data: {
          in_use: 5
        }
      })

    const { exposed, wrapper } = mountRuntimeSummary({
      onSummaryChanged
    })
    await Promise.resolve()
    await Promise.resolve()

    await exposed.refresh(true)

    expect(onSummaryChanged).toHaveBeenCalledWith({ in_use: 2 }, { in_use: 0 })
    expect(onSummaryChanged).toHaveBeenCalledWith({ in_use: 5 }, { in_use: 2 })
    wrapper.unmount()
  })
})
