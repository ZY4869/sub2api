import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import AccountServiceAuthVisualCell from '../AccountServiceAuthVisualCell.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const mountCell = (props: Record<string, unknown>) => mount(AccountServiceAuthVisualCell, {
  props: {
    platform: 'openai',
    type: 'oauth',
    ...props,
  } as any,
  global: {
    stubs: {
      PlatformIcon: {
        props: ['platform'],
        template: '<span class="platform-icon-stub" :data-platform="platform" />'
      }
    }
  }
})

describe('AccountServiceAuthVisualCell', () => {
  it.each([
    ['free', ['border-slate-300/80', 'bg-slate-100', 'text-slate-600']],
    ['plus', ['border-emerald-300/80', 'bg-emerald-50', 'text-emerald-700']],
    ['team', ['border-blue-300/80', 'bg-blue-50', 'text-blue-700']],
  ])('maps %s plan to the airy tier palette', (planType, expectedClasses) => {
    const wrapper = mountCell({ planType })
    const badge = wrapper.find('.platform-icon-stub').element.parentElement

    for (const className of expectedClasses) {
      expect(badge?.classList.contains(className)).toBe(true)
    }
  })

  it('keeps short plan labels such as Team readable in compact airy columns', () => {
    const wrapper = mountCell({ planType: 'team', compact: true })
    const label = wrapper.find('.platform-icon-stub').element.nextElementSibling

    expect(label?.textContent).toBe('Team')
    expect(label?.classList.contains('min-w-0')).toBe(true)
    expect(label?.classList.contains('truncate')).toBe(true)
  })

  it('maps Pro and Pro20x to cyan and black-gold palettes', () => {
    const pro = mountCell({ planType: 'pro', proMultiplier: 5 })
    const pro20 = mountCell({ planType: 'pro', proMultiplier: 20 })

    expect(pro.find('.platform-icon-stub').element.parentElement?.classList.contains('bg-cyan-50')).toBe(true)
    expect(pro.find('.platform-icon-stub').element.parentElement?.classList.contains('text-cyan-700')).toBe(true)
    expect(pro20.find('.platform-icon-stub').element.parentElement?.classList.contains('bg-slate-800')).toBe(true)
    expect(pro20.find('.platform-icon-stub').element.parentElement?.classList.contains('text-amber-400')).toBe(true)
  })

  it('uses Key as the main label for API Key accounts and colors by tier', () => {
    const wrapper = mountCell({
      type: 'apikey',
      planType: 'plus',
      extra: {
        account_tier: 'plus',
      },
    })

    const mainBadge = wrapper.get('[data-test="account-service-plan-badge"]')
    const keyIcon = wrapper.get('[data-test="account-key-type-icon"]')
    const authTypeIcon = wrapper.get('[data-test="account-auth-type-icon"]')
    expect(wrapper.text()).toContain('Key')
    expect(wrapper.text()).toContain('Plus')
    expect(wrapper.text()).toContain('admin.accounts.platforms.openai')
    expect(keyIcon.classes()).toContain('bg-emerald-100')
    expect(keyIcon.classes()).toContain('text-emerald-700')
    expect(authTypeIcon.classes()).toContain('bg-amber-50')
    expect(authTypeIcon.classes()).toContain('text-amber-600')
    expect(wrapper.html()).toContain('bg-emerald-50/90')
    expect(wrapper.html()).toContain('bg-amber-50/90')
    expect(mainBadge.classes()).toContain('bg-emerald-50')
    expect(mainBadge.classes()).toContain('text-emerald-700')
    expect(mainBadge.classes()).toContain('w-fit')
  })

  it('maps Gemini Ultra API Key tier to the high-tier palette', () => {
    const wrapper = mountCell({
      platform: 'gemini',
      type: 'apikey',
      extra: {
        account_tier: 'google_ai_ultra',
      },
    })

    const mainBadge = wrapper.get('[data-test="account-service-plan-badge"]')
    expect(wrapper.text()).toContain('Key')
    expect(wrapper.text()).toContain('Ultra')
    expect(mainBadge.classes()).toContain('bg-slate-800')
    expect(mainBadge.classes()).toContain('text-amber-400')
  })
})
