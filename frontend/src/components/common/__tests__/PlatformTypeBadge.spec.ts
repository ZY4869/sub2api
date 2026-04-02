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
  it('renders localized key and privacy failure labels', () => {
    const keyWrapper = mount(PlatformTypeBadge, {
      props: {
        platform: 'openai',
        type: 'apikey'
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
    expect(keyWrapper.text()).toContain('密钥')
    expect(privacyWrapper.text()).toContain('OpenAI')
    expect(privacyWrapper.text()).toContain('OAuth')
    expect(privacyWrapper.text()).toContain('失败')
  })
})
