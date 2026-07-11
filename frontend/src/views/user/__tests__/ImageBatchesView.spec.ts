import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

import enImageBatches from '@/i18n/locales/en/imageBatches'
import zhImageBatches from '@/i18n/locales/zh/imageBatches'
import ImageBatchesView from '../ImageBatchesView.vue'

const {
  cancelImageBatch,
  deleteImageBatch,
  deleteImageBatchOutputs,
  downloadImageBatch,
  downloadImageBatchItem,
  listImageBatchItems,
  listImageBatchModels,
  listImageBatches,
  saveAs,
  submitImageBatch,
} = vi.hoisted(() => ({
  cancelImageBatch: vi.fn(),
  deleteImageBatch: vi.fn(),
  deleteImageBatchOutputs: vi.fn(),
  downloadImageBatch: vi.fn(),
  downloadImageBatchItem: vi.fn(),
  listImageBatchItems: vi.fn(),
  listImageBatchModels: vi.fn(),
  listImageBatches: vi.fn(),
  saveAs: vi.fn(),
  submitImageBatch: vi.fn(),
}))

const appStore = vi.hoisted(() => ({
  showError: vi.fn(),
  showSuccess: vi.fn(),
  showWarning: vi.fn(),
  showInfo: vi.fn(),
}))

const messages: Record<string, string> = {
  'common.cancel': 'Cancel',
  'common.delete': 'Delete',
  'common.loading': 'Loading',
  'common.refresh': 'Refresh',
  'imageBatches.actions': 'Actions',
  'imageBatches.addItem': 'Add item',
  'imageBatches.apiKey': 'API Key',
  'imageBatches.apiKeyPlaceholder': 'sk-...',
  'imageBatches.apiKeyRequired': 'Enter an API Key that belongs to your account.',
  'imageBatches.count': 'Count',
  'imageBatches.counts': 'Done',
  'imageBatches.customID': 'Custom ID',
  'imageBatches.deleteOutputs': 'Delete outputs',
  'imageBatches.description': 'Submit, monitor, cancel, download, and clean up batch image jobs.',
  'imageBatches.download': 'Download',
  'imageBatches.genericError': 'Image batch request failed. Please retry or contact support.',
  'imageBatches.itemSize': 'Item size',
  'imageBatches.itemTitle': 'Item {index}',
  'imageBatches.items': 'Items',
  'imageBatches.itemsTitle': 'Items',
  'imageBatches.job': 'Job',
  'imageBatches.jobsTitle': 'Jobs',
  'imageBatches.load': 'Load',
  'imageBatches.model': 'Model',
  'imageBatches.noItems': 'No items',
  'imageBatches.noJobs': 'No image batch jobs',
  'imageBatches.outputs': 'Outputs',
  'imageBatches.prompt': 'Prompt',
  'imageBatches.provider': 'Provider',
  'imageBatches.selectJob': 'Select a job',
  'imageBatches.selectModel': 'Select model',
  'imageBatches.size': 'Size',
  'imageBatches.status': 'Status',
  'imageBatches.statuses.cancelled': 'Cancelled',
  'imageBatches.statuses.completed': 'Completed',
  'imageBatches.statuses.created': 'Created',
  'imageBatches.statuses.failed': 'Failed',
  'imageBatches.statuses.indexing': 'Indexing',
  'imageBatches.statuses.output_deleted': 'Outputs deleted',
  'imageBatches.statuses.running': 'Running',
  'imageBatches.statuses.settling': 'Settling',
  'imageBatches.statuses.submitted': 'Submitted',
  'imageBatches.statuses.uploading': 'Uploading',
  'imageBatches.itemStatuses.cancelled': 'Cancelled',
  'imageBatches.itemStatuses.failed': 'Failed',
  'imageBatches.itemStatuses.pending': 'Pending',
  'imageBatches.itemStatuses.success': 'Success',
  'imageBatches.submit': 'Submit batch',
  'imageBatches.submitRequired': 'Select a model and enter at least one prompt.',
  'imageBatches.submitTitle': 'New batch',
  'imageBatches.submitted': 'Batch submitted',
  'imageBatches.submitting': 'Submitting...',
  'imageBatches.title': 'Image Batches',
}

vi.mock('@/api/imageBatches', () => ({
  cancelImageBatch,
  deleteImageBatch,
  deleteImageBatchOutputs,
  downloadImageBatch,
  downloadImageBatchItem,
  listImageBatchItems,
  listImageBatchModels,
  listImageBatches,
  submitImageBatch,
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => appStore,
}))

vi.mock('file-saver', () => ({
  saveAs,
}))

vi.mock('vue-i18n', () => ({
  createI18n: () => ({
    global: {
      locale: { value: 'en' },
      t: (key: string) => messages[key] ?? key,
    },
  }),
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      const value = messages[key] ?? key
      return Object.entries(params ?? {}).reduce(
        (text, [name, paramValue]) => text.replace(`{${name}}`, String(paramValue)),
        value,
      )
    },
  }),
}))

const AppLayoutStub = { template: '<div><slot /></div>' }
const ModelIconStub = { template: '<span data-test="model-icon" />' }
const ModelPlatformIconStub = { template: '<span data-test="platform-icon" />' }

const makeJob = (overrides: Record<string, unknown> = {}) => ({
  id: 'job-1234567890',
  object: 'image.batch',
  status: 'running',
  provider: 'gemini',
  display_model_id: 'gemini-2.5-flash-image',
  size: '1024x1024',
  counts: { items: 1, succeeded: 1, failed: 0, cancelled: 0 },
  created_at: '2026-07-11T01:00:00Z',
  updated_at: '2026-07-11T01:01:00Z',
  completed_at: '2026-07-11T01:01:00Z',
  ...overrides,
})

const mountView = () =>
  mount(ImageBatchesView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        ModelIcon: ModelIconStub,
        ModelPlatformIcon: ModelPlatformIconStub,
      },
    },
  })

const clickButton = async (wrapper: ReturnType<typeof mount>, label: string, index = 0) => {
  const matches = wrapper
    .findAll('button')
    .filter((button) => button.text().trim() === label)
  expect(matches.length).toBeGreaterThan(index)
  await matches[index].trigger('click')
}

const localeKeyPaths = (value: unknown, prefix = ''): string[] => {
  if (value === null || typeof value !== 'object') {
    return [prefix]
  }
  return Object.entries(value as Record<string, unknown>).flatMap(([key, nested]) =>
    localeKeyPaths(nested, prefix ? `${prefix}.${key}` : key),
  )
}

describe('ImageBatchesView', () => {
  beforeEach(() => {
    vi.stubGlobal('crypto', {
      randomUUID: vi
        .fn()
        .mockReturnValueOnce('draft-00000000')
        .mockReturnValueOnce('submit-idempotency-key')
        .mockReturnValue('draft-next'),
    })

    appStore.showSuccess.mockReset()
    cancelImageBatch.mockReset()
    deleteImageBatch.mockReset()
    deleteImageBatchOutputs.mockReset()
    downloadImageBatch.mockReset()
    downloadImageBatchItem.mockReset()
    listImageBatchItems.mockReset()
    listImageBatchModels.mockReset()
    listImageBatches.mockReset()
    saveAs.mockReset()
    submitImageBatch.mockReset()

    listImageBatchModels.mockResolvedValue([
      {
        id: 'gemini-2.5-flash-image',
        object: 'model',
        display_name: 'Gemini image',
        provider: 'gemini',
      },
    ])
    listImageBatches.mockResolvedValue([makeJob()])
    listImageBatchItems.mockResolvedValue([
      {
        custom_id: 'item-1',
        status: 'success',
        output_count: 1,
        created_at: '2026-07-11T01:00:00Z',
        updated_at: '2026-07-11T01:01:00Z',
      },
    ])
    submitImageBatch.mockResolvedValue(makeJob({ status: 'submitted' }))
    cancelImageBatch.mockResolvedValue(makeJob({ status: 'cancelled' }))
    deleteImageBatch.mockResolvedValue(undefined)
    deleteImageBatchOutputs.mockResolvedValue(undefined)
    downloadImageBatch.mockResolvedValue(new Blob(['zip'], { type: 'application/zip' }))
    downloadImageBatchItem.mockResolvedValue(new Blob(['png'], { type: 'image/png' }))
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads models and jobs, submits a batch, opens details, downloads, cancels, and deletes safely', async () => {
    const wrapper = mountView()

    await wrapper.get('input[type="password"]').setValue('sk-live')
    await clickButton(wrapper, 'Load')
    await flushPromises()

    expect(listImageBatchModels).toHaveBeenCalledWith('sk-live')
    expect(listImageBatches).toHaveBeenCalledWith('sk-live')
    expect(wrapper.text()).toContain('gemini-2.5-flash-image')

    await wrapper.get('textarea').setValue('paint a quiet studio scene')
    await clickButton(wrapper, 'Submit batch')
    await flushPromises()

    expect(submitImageBatch).toHaveBeenCalledWith(
      'sk-live',
      expect.objectContaining({
        model: 'gemini-2.5-flash-image',
        provider: 'gemini',
        items: [
          expect.objectContaining({
            custom_id: 'item-draft-00',
            prompt: 'paint a quiet studio scene',
            n: 1,
            size: '1024x1024',
          }),
        ],
      }),
      'submit-idempotency-key',
    )
    expect(appStore.showSuccess).toHaveBeenCalledWith('Batch submitted')
    expect(listImageBatchItems).toHaveBeenCalledWith('sk-live', 'job-1234567890')

    await clickButton(wrapper, 'Download', 0)
    await flushPromises()
    expect(downloadImageBatch).toHaveBeenCalledWith('sk-live', 'job-1234567890')
    expect(saveAs).toHaveBeenCalledWith(expect.any(Blob), 'job-1234567890.zip')

    await clickButton(wrapper, 'Download', 1)
    await flushPromises()
    expect(downloadImageBatchItem).toHaveBeenCalledWith('sk-live', 'job-1234567890', 'item-1')
    expect(saveAs).toHaveBeenCalledWith(expect.any(Blob), 'job-1234567890-item-1.bin')

    await clickButton(wrapper, 'Cancel')
    await flushPromises()
    expect(cancelImageBatch).toHaveBeenCalledWith('sk-live', 'job-1234567890')

    await clickButton(wrapper, 'Delete outputs')
    await flushPromises()
    expect(deleteImageBatchOutputs).toHaveBeenCalledWith('sk-live', 'job-1234567890')

    await clickButton(wrapper, 'Delete', 1)
    await flushPromises()
    expect(deleteImageBatch).toHaveBeenCalledWith('sk-live', 'job-1234567890')
  })

  it('shows actionable errors for missing API keys and gateway failures', async () => {
    const wrapper = mountView()

    await clickButton(wrapper, 'Load')
    expect(wrapper.text()).toContain('Enter an API Key that belongs to your account.')

    listImageBatchModels.mockRejectedValueOnce({
      response: { data: { error: { message: 'Batch access is disabled for this key.' } } },
    })

    await wrapper.get('input[type="password"]').setValue('sk-disabled')
    await clickButton(wrapper, 'Load')
    await flushPromises()

    expect(wrapper.text()).toContain('Batch access is disabled for this key.')
  })

  it('keeps English and Chinese image batch locale keys aligned', () => {
    expect(localeKeyPaths(enImageBatches).sort()).toEqual(
      localeKeyPaths(zhImageBatches).sort(),
    )
  })
})
