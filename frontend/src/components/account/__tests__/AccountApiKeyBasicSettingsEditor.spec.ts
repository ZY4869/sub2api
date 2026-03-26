import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import AccountApiKeyBasicSettingsEditor from '../AccountApiKeyBasicSettingsEditor.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const modelScopeStub = {
  name: 'AccountModelScopeEditor',
  props: ['platform', 'mode', 'disabled', 'allowedModels'],
  emits: ['update:mode', 'update:allowedModels', 'add-mapping', 'remove-mapping', 'add-preset'],
  template: `
    <div data-testid="model-scope">
      <span data-testid="model-scope-platform">{{ platform }}</span>
      <span data-testid="model-scope-mode">{{ mode }}</span>
      <button type="button" data-testid="emit-mode" @click="$emit('update:mode', 'mapping')" />
      <button type="button" data-testid="emit-models" @click="$emit('update:allowedModels', ['gpt-5.4'])" />
      <button type="button" data-testid="emit-add" @click="$emit('add-mapping')" />
      <button type="button" data-testid="emit-remove" @click="$emit('remove-mapping', 1)" />
      <button type="button" data-testid="emit-preset" @click="$emit('add-preset', { from: 'a', to: 'b' })" />
    </div>
  `
}

describe('AccountApiKeyBasicSettingsEditor', () => {
  it('renders create-mode gemini inputs and emits model updates', async () => {
    const wrapper = mount(AccountApiKeyBasicSettingsEditor, {
      props: {
        platform: 'gemini',
        mode: 'create',
        baseUrl: 'https://generativelanguage.googleapis.com',
        apiKey: '',
        modelScopeMode: 'whitelist',
        allowedModels: [],
        geminiTierAiStudio: 'aistudio_free',
        modelMappings: [],
        presetMappings: [],
        getMappingKey: () => 'mapping-1',
        showGeminiTier: true
      },
      global: {
        stubs: {
          AccountModelScopeEditor: modelScopeStub
        }
      }
    })

    expect(wrapper.text()).toContain('admin.accounts.apiKeyRequired')
    expect(wrapper.text()).toContain('admin.accounts.gemini.apiKeyHint')
    expect(wrapper.get('[data-testid="gemini-api-key-tier"]').exists()).toBe(true)
    expect(wrapper.get('input[type="password"]').attributes('required')).toBeDefined()

    await wrapper.get('input[type="text"]').setValue('https://example.com')
    await wrapper.get('input[type="password"]').setValue('AIza-test')
    await wrapper.get('[data-testid="gemini-api-key-tier"]').setValue('aistudio_paid')
    await wrapper.get('[data-testid="emit-mode"]').trigger('click')
    await wrapper.get('[data-testid="emit-models"]').trigger('click')

    expect(wrapper.emitted('update:baseUrl')?.[0]).toEqual(['https://example.com'])
    expect(wrapper.emitted('update:apiKey')?.[0]).toEqual(['AIza-test'])
    expect(wrapper.emitted('update:geminiTierAiStudio')?.[0]).toEqual(['aistudio_paid'])
    expect(wrapper.emitted('update:modelScopeMode')?.[0]).toEqual(['mapping'])
    expect(wrapper.emitted('update:allowedModels')?.[0]).toEqual([['gpt-5.4']])
  })

  it('renders edit-mode hints and forwards model scope events', async () => {
    const wrapper = mount(AccountApiKeyBasicSettingsEditor, {
      props: {
        platform: 'openai',
        mode: 'edit',
        baseUrl: 'https://api.openai.com',
        apiKey: '',
        modelScopeMode: 'whitelist',
        allowedModels: [],
        modelMappings: [],
        presetMappings: [],
        getMappingKey: () => 'mapping-1'
      },
      global: {
        stubs: {
          AccountModelScopeEditor: modelScopeStub
        }
      }
    })

    expect(wrapper.text()).toContain('admin.accounts.leaveEmptyToKeep')
    expect(wrapper.find('[data-testid="gemini-api-key-tier"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="model-scope-platform"]').text()).toBe('openai')

    await wrapper.get('[data-testid="emit-add"]').trigger('click')
    await wrapper.get('[data-testid="emit-remove"]').trigger('click')
    await wrapper.get('[data-testid="emit-preset"]').trigger('click')

    expect(wrapper.emitted('add-mapping')).toHaveLength(1)
    expect(wrapper.emitted('remove-mapping')?.[0]).toEqual([1])
    expect(wrapper.emitted('add-preset')?.[0]).toEqual([{ from: 'a', to: 'b' }])
  })

  it('hides model scope editor when skipModelScopeEditor is enabled', () => {
    const wrapper = mount(AccountApiKeyBasicSettingsEditor, {
      props: {
        platform: 'protocol_gateway',
        gatewayProtocol: 'openai',
        effectivePlatform: 'openai',
        mode: 'create',
        baseUrl: 'https://example.com',
        apiKey: 'sk-test',
        modelScopeMode: 'whitelist',
        allowedModels: [],
        modelMappings: [],
        presetMappings: [],
        getMappingKey: () => 'mapping-1',
        skipModelScopeEditor: true
      },
      global: {
        stubs: {
          AccountModelScopeEditor: modelScopeStub
        }
      }
    })

    expect(wrapper.find('[data-testid="model-scope"]').exists()).toBe(false)
  })
})
