import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import { reactive } from 'vue'
import type { UpdateSettingsRequest } from '@/api/admin/settings'
import enClaudeCode from '@/i18n/locales/en/admin/settings/claudeCode'
import zhClaudeCode from '@/i18n/locales/zh/admin/settings/claudeCode'
import SettingsGatewayExtraTab from '../settings/SettingsGatewayExtraTab.vue'

describe('settings Claude Code Codex plugin control', () => {
  it('renders the switch and updates the form field', async () => {
    const form = reactive({
      openai_allow_claude_code_codex_plugin: false,
      min_claude_code_version: '',
      max_claude_code_version: '',
      allow_ungrouped_key_scheduling: false,
      maintenance_mode_enabled: false
    })
    const wrapper = mount(SettingsGatewayExtraTab, {
      props: {
        ctx: {
          form,
          t: (key: string) => key
        }
      }
    })

    expect(wrapper.text()).toContain('admin.settings.claudeCode.allowCodexPlugin')
    const switches = wrapper.findAll('[role="switch"]')
    expect(switches.length).toBeGreaterThan(0)

    await switches[0].trigger('click')

    expect(form.openai_allow_claude_code_codex_plugin).toBe(true)
  })

  it('keeps i18n labels for both supported locales', () => {
    expect(zhClaudeCode.claudeCode.allowCodexPlugin).toBeTruthy()
    expect(zhClaudeCode.claudeCode.allowCodexPluginHint).toContain('Claude Code')
    expect(enClaudeCode.claudeCode.allowCodexPlugin).toBeTruthy()
    expect(enClaudeCode.claudeCode.allowCodexPluginHint).toContain('Claude Code')
  })

  it('allows the save payload field in the typed settings request', () => {
    const payload: UpdateSettingsRequest = {
      openai_allow_claude_code_codex_plugin: true
    }

    expect(payload.openai_allow_claude_code_codex_plugin).toBe(true)
  })
})
