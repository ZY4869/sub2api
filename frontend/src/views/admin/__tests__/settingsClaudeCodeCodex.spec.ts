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
      openai_allowed_codex_clients: [] as string[],
      claude_oauth_system_prompt_blocks_enabled: false,
      claude_oauth_system_prompt_blocks: '',
      min_claude_code_version: '',
      max_claude_code_version: '',
      allow_ungrouped_key_scheduling: false,
      maintenance_mode_enabled: false,
      admin_compliance_enabled: false
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
    const clientCheckbox = wrapper.find('input[type="checkbox"][value="claude_code"]')
    expect(clientCheckbox.exists()).toBe(true)

    await clientCheckbox.setValue(true)

    expect(form.openai_allowed_codex_clients).toEqual(['claude_code'])

    const promptBlockSwitch = switches.at(1)
    expect(promptBlockSwitch).toBeTruthy()
    await promptBlockSwitch!.trigger('click')

    expect(form.claude_oauth_system_prompt_blocks_enabled).toBe(true)
    const promptTextarea = wrapper.find('textarea')
    await promptTextarea.setValue('Site safety block')
    expect(form.claude_oauth_system_prompt_blocks).toBe('Site safety block')
  })

  it('keeps i18n labels for both supported locales', () => {
    expect(zhClaudeCode.claudeCode.allowCodexPlugin).toBeTruthy()
    expect(zhClaudeCode.claudeCode.allowCodexPluginHint).toContain('Claude Code')
    expect(zhClaudeCode.claudeCode.allowedClients).toBeTruthy()
    expect(zhClaudeCode.claudeCode.allowedClientClaudeCode).toContain('Claude Code')
    expect(zhClaudeCode.claudeCode.oauthPromptBlocks).toBeTruthy()
    expect(zhClaudeCode.claudeCode.oauthPromptBlocksPriorityHint).toContain('安全')
    expect(enClaudeCode.claudeCode.allowCodexPlugin).toBeTruthy()
    expect(enClaudeCode.claudeCode.allowCodexPluginHint).toContain('Claude Code')
    expect(enClaudeCode.claudeCode.allowedClients).toBeTruthy()
    expect(enClaudeCode.claudeCode.allowedClientClaudeCode).toContain('Claude Code')
    expect(enClaudeCode.claudeCode.oauthPromptBlocks).toBeTruthy()
    expect(enClaudeCode.claudeCode.oauthPromptBlocksPriorityHint).toContain('lower priority')
  })

  it('allows the save payload field in the typed settings request', () => {
    const payload: UpdateSettingsRequest = {
      openai_allow_claude_code_codex_plugin: true,
      openai_allowed_codex_clients: ['claude_code'],
      claude_oauth_system_prompt_blocks_enabled: true,
      claude_oauth_system_prompt_blocks: 'Site safety block',
      content_moderation_cyber_policy_enabled: true,
      content_moderation_cyber_categories: [
        { id: 'credential_theft', keywords: ['steal api key'] }
      ],
      admin_compliance_enabled: true
    }

    expect(payload.openai_allow_claude_code_codex_plugin).toBe(true)
    expect(payload.openai_allowed_codex_clients).toEqual(['claude_code'])
    expect(payload.claude_oauth_system_prompt_blocks).toBe('Site safety block')
    expect(payload.content_moderation_cyber_categories?.[0].id).toBe('credential_theft')
    expect(payload.admin_compliance_enabled).toBe(true)
  })
})
