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
    expect(label?.classList.contains('min-w-[2.6rem]')).toBe(true)
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

    const mainBadge = wrapper.find('svg').element.parentElement
    expect(wrapper.text()).toContain('Key')
    expect(wrapper.text()).toContain('Plus')
    expect(wrapper.text()).toContain('admin.accounts.platforms.openai')
    expect(mainBadge?.classList.contains('bg-emerald-50')).toBe(true)
    expect(mainBadge?.classList.contains('text-emerald-700')).toBe(true)
  })

  it('maps Gemini Ultra API Key tier to the high-tier palette', () => {
    const wrapper = mountCell({
      platform: 'gemini',
      type: 'apikey',
      extra: {
        account_tier: 'google_ai_ultra',
      },
    })

    const mainBadge = wrapper.find('svg').element.parentElement
    expect(wrapper.text()).toContain('Key')
    expect(wrapper.text()).toContain('Ultra')
    expect(mainBadge?.classList.contains('bg-slate-800')).toBe(true)
    expect(mainBadge?.classList.contains('text-amber-400')).toBe(true)
  })
})
