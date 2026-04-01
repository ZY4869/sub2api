import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/components/account/CreateAccountModal.vue'),
  'utf-8'
)

describe('CreateAccountModal', () => {
  it('uses the dedicated account-wide dialog with local horizontal overflow protection', () => {
    expect(source).toContain('width="account-wide"')
    expect(source).toContain('class="min-w-0 overflow-x-hidden"')
  })

  it('keeps the OAuth step indicator responsive', () => {
    expect(source).toContain('flex-col items-center gap-3 sm:w-auto sm:flex-row sm:gap-4')
    expect(source).toContain('hidden h-0.5 w-8 bg-gray-300 dark:bg-dark-600 sm:block')
    expect(source).toContain('min-w-0 break-words text-sm font-medium text-gray-700 dark:text-gray-300')
  })

  it('uses the protocol gateway probe editor and hides the generic auto-import toggle for that platform', () => {
    expect(source).toContain('AccountProtocolGatewayModelProbeEditor')
    expect(source).toContain(":skip-model-scope-editor=\"form.platform === 'protocol_gateway'\"")
    expect(source).toContain(":show-auto-import=\"form.platform !== 'protocol_gateway'\"")
  })

  it('embeds the Grok batch import panel alongside the single-account Grok fields', () => {
    expect(source).toContain('AccountGrokImportPanel')
    expect(source).toContain("@imported=\"handleGrokImportCompleted\"")
  })

  it('defaults Grok to API Key mode and only persists grok_tier for SSO submissions', () => {
    expect(source).toContain("if (newPlatform === 'grok')")
    expect(source).toContain("accountCategory.value = 'apikey'")
    expect(source).toContain("form.type = 'apikey'")
    expect(source).toContain("form.platform === 'grok' && form.type === 'sso'")
  })

  it('shows generic quota controls and protocol gateway batch controls in account creation', () => {
    expect(source).toContain('const showQuotaLimitSection = computed(() =>')
    expect(source).toContain('const showQuotaLimitSection = computed(() => true)')
    expect(source).toContain('AccountProtocolGatewayBatchEditor')
    expect(source).toContain('const showProtocolGatewayBatchEditor = computed(() =>')
  })
})
