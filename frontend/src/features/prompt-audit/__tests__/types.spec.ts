import { describe, expect, it } from 'vitest'
import { draftFromConfig, payloadFromDraft } from '../types'
import type { PromptAuditConfig } from '@/api/admin/promptAudit'

describe('prompt audit draft mapping', () => {
  it('keeps disabled config saveable without endpoints', () => {
    const config: PromptAuditConfig = {
      enabled: false,
      blocking_enabled: false,
      store_pass_events: false,
      effective_mode: 'off',
      strategy: 'priority',
      worker_count: 4,
      queue_capacity: 32768,
      scanners: ['jailbreak'],
      all_groups: true,
      group_ids: [],
      endpoints: [],
      config_version: 7,
      updated_at: '',
      updated_by: 0,
      change_summary: ''
    }

    const draft = draftFromConfig(config)
    const payload = payloadFromDraft(draft)

    expect(payload).toMatchObject({
      expected_config_version: 7,
      enabled: false,
      blocking_enabled: false,
      endpoints: []
    })
  })

  it('preserves existing endpoint tokens unless replace or clear is selected', () => {
    const draft = draftFromConfig({
      enabled: true,
      blocking_enabled: true,
      store_pass_events: false,
      effective_mode: 'blocking',
      strategy: 'priority',
      worker_count: 4,
      queue_capacity: 32768,
      scanners: ['jailbreak'],
      all_groups: true,
      group_ids: [],
      config_version: 1,
      updated_at: '',
      updated_by: 0,
      change_summary: '',
      endpoints: [
        {
          id: 'primary',
          name: 'Primary',
          protocol: 'openai_compatible',
          base_url: 'https://guard.example.com',
          model: 'sileader/qwen3guard:0.6b',
          timeout_ms: 3000,
          input_limit: 12000,
          enabled: true,
          has_token: true,
          token_status: 'stored'
        }
      ]
    })

    expect(payloadFromDraft(draft).endpoints[0]).toMatchObject({
      token: undefined,
      clear_token: false
    })

    draft.endpoints[0].token_mode = 'clear'
    expect(payloadFromDraft(draft).endpoints[0]).toMatchObject({ clear_token: true })

    draft.endpoints[0].token_mode = 'replace'
    draft.endpoints[0].token = 'sk-new'
    expect(payloadFromDraft(draft).endpoints[0]).toMatchObject({ token: 'sk-new' })
  })
})
