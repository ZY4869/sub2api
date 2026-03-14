import { defineComponent, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import BulkEditModelRestrictionSection from '../BulkEditModelRestrictionSection.vue'

const showInfo = vi.fn()

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showInfo
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

function mountSection(initialMode: 'whitelist' | 'mapping' = 'whitelist') {
  const enabled = ref(true)
  const mode = ref<'whitelist' | 'mapping'>(initialMode)
  const allowedModels = ref<string[]>([])
  const modelMappings = ref<{ from: string; to: string }[]>([])

  const wrapper = mount(
    defineComponent({
      components: { BulkEditModelRestrictionSection },
      setup() {
        return {
          enabled,
          mode,
          allowedModels,
          modelMappings,
          models: [
            { value: 'claude-sonnet-4.5', label: 'claude-sonnet-4.5' },
            { value: 'gemini-2.5-flash-image', label: 'gemini-2.5-flash-image' }
          ],
          presets: [
            {
              label: 'Sonnet 4.5',
              from: 'claude-3.7-sonnet',
              to: 'claude-sonnet-4.5',
              color: 'bg-purple-100'
            }
          ]
        }
      },
      template: `
        <BulkEditModelRestrictionSection
          v-model:enabled="enabled"
          v-model:mode="mode"
          v-model:allowed-models="allowedModels"
          v-model:model-mappings="modelMappings"
          :models="models"
          :presets="presets"
        />
      `
    })
  )

  return { wrapper, enabled, mode, allowedModels, modelMappings }
}

describe('BulkEditModelRestrictionSection', () => {
  beforeEach(() => {
    showInfo.mockReset()
  })

  it('updates whitelist selections through v-model', async () => {
    const { wrapper, allowedModels } = mountSection()

    await wrapper.get('input[value="claude-sonnet-4.5"]').setValue(true)
    expect(allowedModels.value).toEqual(['claude-sonnet-4.5'])

    await wrapper.get('input[value="claude-sonnet-4.5"]').setValue(false)
    expect(allowedModels.value).toEqual([])
  })

  it('adds preset mappings and blocks duplicates', async () => {
    const { wrapper, modelMappings } = mountSection('mapping')

    const presetButton = wrapper
      .findAll('button')
      .find((button) => button.text().includes('Sonnet 4.5'))

    expect(presetButton).toBeTruthy()

    await presetButton?.trigger('click')
    expect(modelMappings.value).toEqual([
      { from: 'claude-3.7-sonnet', to: 'claude-sonnet-4.5' }
    ])

    await presetButton?.trigger('click')
    expect(showInfo).toHaveBeenCalledWith('admin.accounts.mappingExists')
    expect(modelMappings.value).toHaveLength(1)
  })
})
