import { mount } from '@vue/test-utils'
import { defineComponent, ref } from 'vue'
import { describe, expect, it, vi } from 'vitest'
import PlatformTypeBadge from '../PlatformTypeBadge.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: ref('zh'),
      t: (key: string) => ({
        'admin.accounts.platforms.openai': 'OpenAI',
        'admin.accounts.platforms.protocol_gateway': '协议网关',
        'ui.platformType.oauth': 'OAuth',
        'ui.platformType.token': '令牌',
        'ui.platformType.key': '密钥',
        'ui.platformType.sso': 'SSO',
        'ui.platformType.aws': 'AWS',
        'ui.platformType.privacy': '隐私',
        'ui.platformType.fail': '失败',
        'admin.accounts.privacyTrainingOff': '已关闭训练数据共享',
        'admin.accounts.privacyCfBlocked': '被 Cloudflare 拦截',
        'admin.accounts.privacyFailed': '关闭失败',
        'admin.accounts.keyUsage.keyAccountTooltip': 'Key 账号',
      }[key] || key)
    })
  }
})

const PlatformIconStub = defineComponent({
  template: '<span class="platform-icon">icon</span>'
})

const IconStub = defineComponent({
  template: '<span class="icon-stub">icon</span>'
})

describe('PlatformTypeBadge', () => {
  it('renders API Key accounts with Key as the primary label and platform as secondary text', () => {
    const keyWrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai',
        type: 'apikey',
        planType: 'plus'
      } as any,
      global: {
        stubs: {
          PlatformIcon: PlatformIconStub,
          Icon: IconStub
        }
      }
    })

    const privacyWrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai',
        type: 'oauth',
        privacyMode: 'training_set_failed'
      } as any,
      global: {
        stubs: {
          PlatformIcon: PlatformIconStub,
          Icon: IconStub
        }
      }
    })

    expect(keyWrapper.text()).toContain('OpenAI')
    expect(keyWrapper.text()).toContain('Key')
    expect(keyWrapper.text()).toContain('Plus')
    expect(keyWrapper.text()).not.toContain('密钥')
    expect(keyWrapper.find('[title="Key 账号"]').exists()).toBe(true)
    expect(privacyWrapper.text()).toContain('OpenAI')
    expect(privacyWrapper.text()).toContain('OAuth')
    expect(privacyWrapper.text()).toContain('失败')
  })

  it('prefers multiplier-specific Pro label when plan type is Pro and falls back to pro multiplier formatting', () => {
    const explicitWrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai',
        type: 'oauth',
        planType: 'pro',
        planTypeLabel: 'Pro',
        proMultiplier: 20
      } as any,
      global: {
        stubs: {
          PlatformIcon: PlatformIconStub,
          Icon: IconStub
        }
      }
    })

    const multiplierWrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai',
        type: 'oauth',
        planType: 'pro',
        proMultiplier: 5
      } as any,
      global: {
        stubs: {
          PlatformIcon: PlatformIconStub,
          Icon: IconStub
        }
      }
    })

    expect(explicitWrapper.text()).toContain('Pro20x')
    expect(multiplierWrapper.text()).toContain('Pro5x')
  })

  it('expands generic Pro label with pro multiplier when available', () => {
    const wrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai',
        type: 'oauth',
        planType: 'pro',
        planTypeLabel: 'Pro',
        proMultiplier: 20
      } as any,
      global: {
        stubs: {
          PlatformIcon: PlatformIconStub,
          Icon: IconStub
        }
      }
    })

    expect(wrapper.text()).toContain('Pro20x')
    expect(wrapper.text()).not.toContain('ProOAuth')
  })

  it('prefers multiplier-specific Pro label even when explicit label is only generic Pro', () => {
    const wrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai',
        type: 'oauth',
        planType: 'pro',
        planTypeLabel: 'Pro',
        proMultiplier: 5
      } as any,
      global: {
        stubs: {
          PlatformIcon: PlatformIconStub,
          Icon: IconStub
        }
      }
    })

    expect(wrapper.text()).toContain('Pro5x')
  })

  it('renders protocol gateway mixed badge with localized label', () => {
    const wrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'protocol_gateway',
        gatewayProtocol: 'mixed',
        type: 'apikey'
      } as any,
      global: {
        stubs: {
          PlatformIcon: PlatformIconStub,
          Icon: IconStub
        }
      }
    })

    expect(wrapper.text()).toContain('协议网关')
    expect(wrapper.text()).toContain('混合')
    expect(wrapper.text()).toContain('Key')
    expect(wrapper.text()).not.toContain('密钥')
  })

  it('colors API Key tier badges by plan and Gemini tier', () => {
    const plus = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai',
        type: 'apikey',
        planType: 'plus'
      } as any,
      global: {
        stubs: {
          PlatformIcon: PlatformIconStub,
          Icon: IconStub
        }
      }
    })
    const ultra = mount(PlatformTypeBadge, {
      props: {
        platform: 'gemini',
        type: 'apikey',
        extra: { account_tier: 'google_ai_ultra' }
      } as any,
      global: {
        stubs: {
          PlatformIcon: PlatformIconStub,
          Icon: IconStub
        }
      }
    })

    expect(plus.find('[title="Key 账号"]').classes()).toContain('bg-emerald-50')
    expect(ultra.find('[title="Key 账号"]').classes()).toContain('bg-slate-800')
    expect(ultra.text()).toContain('Ultra')
  })
})
