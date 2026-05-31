import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import TimeAccessPolicyEditor from '../TimeAccessPolicyEditor.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'common.timeAccess.windowTitle') return `Window ${params?.index}`
      return key
    },
  }),
}))

const IconStub = {
  template: '<span />',
}

function mountEditor(modelValue = {
  enabled: true,
  timezone: 'Asia/Singapore',
  weekly_windows: [{ days: [1, 2, 3, 4, 5], start: '08:00', end: '20:00' }],
  daily_allowed_minutes: 720,
}) {
  return mount(TimeAccessPolicyEditor, {
    props: { modelValue },
    global: {
      stubs: {
        Icon: IconStub,
      },
    },
  })
}

function lastPolicy(wrapper: ReturnType<typeof mountEditor>) {
  const events = wrapper.emitted('update:modelValue') || []
  return events.at(-1)?.[0] as any
}

function localDateTimeToISOString(value: string) {
  return new Date(value).toISOString()
}

describe('TimeAccessPolicyEditor', () => {
  it('adds and removes weekly windows', async () => {
    const wrapper = mountEditor()

    await wrapper.get('[data-testid="time-access-add-window"]').trigger('click')
    expect(lastPolicy(wrapper).weekly_windows).toHaveLength(2)

    await wrapper.setProps({ modelValue: lastPolicy(wrapper) })
    await wrapper.get('[data-testid="time-access-remove-window-1"]').trigger('click')
    expect(lastPolicy(wrapper).weekly_windows).toHaveLength(1)
  })

  it('toggles days without allowing an empty day set', async () => {
    const wrapper = mountEditor({
      enabled: true,
      timezone: 'Asia/Singapore',
      weekly_windows: [{ days: [1], start: '08:00', end: '20:00' }],
      daily_allowed_minutes: null,
    })

    await wrapper.get('[data-testid="time-access-day-0-1"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    await wrapper.get('[data-testid="time-access-day-0-2"]').trigger('click')
    expect(lastPolicy(wrapper).weekly_windows[0].days).toEqual([1, 2])
  })

  it('emits cross-midnight windows and boundary fields', async () => {
    const wrapper = mountEditor()

    await wrapper.get('[data-testid="time-access-start-0"]').setValue('22:00')
    await wrapper.setProps({ modelValue: lastPolicy(wrapper) })
    await wrapper.get('[data-testid="time-access-end-0"]').setValue('02:00')
    await wrapper.setProps({ modelValue: lastPolicy(wrapper) })
    await wrapper.get('[data-testid="time-access-not-before"]').setValue('2026-06-01T08:00')
    await wrapper.setProps({ modelValue: lastPolicy(wrapper) })
    await wrapper.get('[data-testid="time-access-daily-allowed-minutes"]').setValue('240')

    const policy = lastPolicy(wrapper)
    expect(policy.weekly_windows[0]).toEqual(expect.objectContaining({
      start: '22:00',
      end: '02:00',
    }))
    expect(policy.not_before).toBe(localDateTimeToISOString('2026-06-01T08:00'))
    expect(policy.daily_allowed_minutes).toBe(240)
  })

  it('applies presets through the shared editor', async () => {
    const wrapper = mountEditor()
    const deepNight = wrapper.findAll('button').find((button) =>
      button.text().includes('common.timeAccess.presets.deep_night')
    )

    expect(deepNight).toBeDefined()
    await deepNight!.trigger('click')

    expect(lastPolicy(wrapper)).toEqual(expect.objectContaining({
      enabled: true,
      daily_allowed_minutes: 360,
      weekly_windows: [
        { days: [0, 1, 2, 3, 4, 5, 6], start: '00:00', end: '06:00' },
      ],
    }))
  })
})
