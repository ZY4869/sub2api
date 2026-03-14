import { defineComponent, ref } from 'vue'
import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import BulkEditStatusGroupSection from '../BulkEditStatusGroupSection.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const groupSelectorStub = {
  name: 'GroupSelector',
  props: ['modelValue', 'groups'],
  emits: ['update:modelValue'],
  template:
    '<button type="button" data-testid="group-selector" @click="$emit(\'update:modelValue\', [1, 2])">groups</button>'
}

const selectStub = {
  name: 'Select',
  props: ['modelValue', 'options'],
  emits: ['update:modelValue'],
  template:
    '<button type="button" data-testid="status-select" @click="$emit(\'update:modelValue\', \'inactive\')">status</button>'
}

function mountSection() {
  const enableStatus = ref(true)
  const status = ref<'active' | 'inactive'>('active')
  const enableGroups = ref(true)
  const groupIds = ref<number[]>([])

  const wrapper = mount(
    defineComponent({
      components: { BulkEditStatusGroupSection },
      setup() {
        return {
          enableStatus,
          status,
          enableGroups,
          groupIds,
          groups: []
        }
      },
      template: `
        <BulkEditStatusGroupSection
          v-model:enable-status="enableStatus"
          v-model:status="status"
          v-model:enable-groups="enableGroups"
          v-model:group-ids="groupIds"
          :groups="groups"
        />
      `
    }),
    {
      global: {
        stubs: {
          GroupSelector: groupSelectorStub,
          Select: selectStub
        }
      }
    }
  )

  return { wrapper, status, groupIds }
}

describe('BulkEditStatusGroupSection', () => {
  it('forwards status and group updates', async () => {
    const { wrapper, status, groupIds } = mountSection()

    await wrapper.get('[data-testid="status-select"]').trigger('click')
    await wrapper.get('[data-testid="group-selector"]').trigger('click')

    expect(status.value).toBe('inactive')
    expect(groupIds.value).toEqual([1, 2])
  })
})
