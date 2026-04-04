import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountBulkActionsBar from '../AccountBulkActionsBar.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

describe('AccountBulkActionsBar', () => {
  it('emits archive when selected accounts are from one platform', async () => {
    const wrapper = mount(AccountBulkActionsBar, {
      props: {
        selectedIds: [1, 2],
        selectedPlatforms: ['kiro']
      }
    })

    const archiveButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.bulkActions.archive')
    )

    expect(archiveButton).toBeTruthy()
    expect(archiveButton?.attributes('disabled')).toBeUndefined()
    await archiveButton?.trigger('click')
    expect(wrapper.emitted('archive')).toEqual([[]])
  })

  it('disables archive when selected accounts span multiple platforms', () => {
    const wrapper = mount(AccountBulkActionsBar, {
      props: {
        selectedIds: [1, 2],
        selectedPlatforms: ['openai', 'kiro']
      }
    })

    const archiveButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.bulkActions.archive')
    )

    expect(archiveButton).toBeTruthy()
    expect(archiveButton?.attributes('disabled')).toBeDefined()
    expect(archiveButton?.attributes('title')).toBe('admin.accounts.bulkActions.archiveMixedPlatformDisabled')
  })

  it('emits batch-test for selected accounts', async () => {
    const wrapper = mount(AccountBulkActionsBar, {
      props: {
        selectedIds: [1],
        selectedPlatforms: ['openai']
      }
    })

    const batchTestButton = wrapper.findAll('button').find((button) =>
      button.text().includes('admin.accounts.bulkActions.batchTest')
    )

    expect(batchTestButton).toBeTruthy()
    await batchTestButton?.trigger('click')
    expect(wrapper.emitted('batch-test')).toEqual([[]])
  })
})
