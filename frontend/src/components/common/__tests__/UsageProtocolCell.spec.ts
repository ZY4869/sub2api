import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'

import UsageProtocolCell from '../UsageProtocolCell.vue'

const translations: Record<string, string> = {
  'usage.protocolFamilies.openai': 'OpenAI',
  'usage.protocolFamilies.anthropic': 'Anthropic',
  'usage.protocolFamilies.gemini': 'Gemini',
  'usage.protocolFamilies.unknown': 'Unknown',
  'usage.protocolModes.native': 'Native',
  'usage.protocolModes.compatible': 'Compatible',
  'usage.protocolTransition': 'Inbound -> Upstream: {transition}'
}

vi.mock('@/i18n', () => ({
  i18n: {
    global: {
      te: (key: string) => key in translations,
      t: (key: string) => translations[key] ?? key
    }
  }
}))

const mountCell = (props: Record<string, unknown>) =>
  mount(UsageProtocolCell, {
    props,
    global: {
      stubs: {
        PlatformIcon: {
          props: ['platform', 'size'],
          template: '<span data-testid="platform-icon" :data-platform="platform" />'
        }
      }
    }
  })

describe('UsageProtocolCell', () => {
  it('shows vendor icon and native mode for native protocols', () => {
    const wrapper = mountCell({
      inboundPath: '/v1/chat/completions',
      upstreamPath: '/v1/chat/completions'
    })

    expect(wrapper.get('[data-testid="platform-icon"]').attributes('data-platform')).toBe('openai')
    expect(wrapper.text()).toContain('OpenAI')
    expect(wrapper.text()).toContain('/v1/chat/completions')
    expect(wrapper.text()).toContain('Native')
    expect(wrapper.get('div.space-y-1').attributes('title')).toBe('/v1/chat/completions')
  })

  it('shows the shared AI icon and compatibility metadata for compatible protocols', () => {
    const wrapper = mountCell({
      inboundPath: '/v1beta/openai/chat/completions',
      upstreamPath: '/v1beta/models'
    })

    expect(wrapper.get('[data-testid="platform-icon"]').attributes('data-platform')).toBe('protocol_gateway')
    expect(wrapper.text()).toContain('Gemini')
    expect(wrapper.text()).toContain('/v1beta/openai/chat/completions')
    expect(wrapper.text()).toContain('Compatible')
    expect(wrapper.get('div.space-y-1').attributes('title')).toContain('Inbound -> Upstream')
  })
})
