import { describe, expect, it } from 'vitest'
import {
  buildAccountModelImportToastPayload,
  extractSyncableRegistryModels,
  mergeAccountModelImportResults,
  resolveAccountModelImportErrorMessage,
  resolveAccountModelImportProbeNoticeMessage,
  shouldInvalidateModelInventory
} from '@/utils/accountModelImport'

const t = (key: string, named?: Record<string, unknown>) => (
  named ? `${key}:${JSON.stringify(named)}` : key
)

describe('resolveAccountModelImportErrorMessage', () => {
  it('prefers response detail over generic error message', () => {
    const message = resolveAccountModelImportErrorMessage(t, {
      message: 'Request failed with status code 400',
      response: {
        data: {
          detail: 'account must be active to import models',
          message: 'bad request'
        }
      }
    })

    expect(message).toBe('account must be active to import models')
  })

  it('maps unsupported probing errors to localized copy', () => {
    const message = resolveAccountModelImportErrorMessage(t, {
      response: {
        data: {
          message: 'current Sora account type does not support real model probing'
        }
      }
    })

    expect(message).toBe('admin.accounts.modelImportUnsupported')
  })

  it('falls back to generic failure copy when no message exists', () => {
    expect(resolveAccountModelImportErrorMessage(t, {})).toBe('admin.accounts.modelImportFailed')
  })
})

describe('resolveAccountModelImportProbeNoticeMessage', () => {
  it('maps Kiro builtin catalog source to localized copy', () => {
    expect(resolveAccountModelImportProbeNoticeMessage(t, {
      imported_count: 4,
      probe_source: 'kiro_builtin_catalog',
      probe_notice: 'using built-in Kiro model catalog'
    })).toBe('admin.accounts.modelImportKiroBuiltinCatalog')
  })

  it('prefers explicit probe notice from backend', () => {
    expect(resolveAccountModelImportProbeNoticeMessage(t, {
      imported_count: 6,
      probe_source: 'gemini_cli_default_fallback',
      probe_notice: 'AI Studio model listing lacks required scopes; imported Gemini CLI default models instead'
    })).toBe('AI Studio model listing lacks required scopes; imported Gemini CLI default models instead')
  })

  it('maps Gemini CLI fallback source to localized copy', () => {
    expect(resolveAccountModelImportProbeNoticeMessage(t, {
      imported_count: 3,
      probe_source: 'gemini_cli_default_fallback'
    })).toBe('admin.accounts.modelImportGeminiFallback:{"count":3}')
  })

  it('returns empty string for upstream probe results', () => {
    expect(resolveAccountModelImportProbeNoticeMessage(t, {
      imported_count: 2,
      probe_source: 'upstream'
    })).toBe('')
  })
})

describe('buildAccountModelImportToastPayload', () => {
  it('builds warning toast with canonical merge and failure details', () => {
    const payload = buildAccountModelImportToastPayload(t, {
      account_id: 1,
      detected_models: ['claude-sonnet-4-5-20250929', 'bad-model'],
      imported_models: [],
      imported_count: 0,
      skipped_count: 1,
      failed_models: [],
      model_results: [
        {
          source_model: 'claude-sonnet-4-5-20250929',
          canonical_model: 'claude-sonnet-4.5',
          status: 'merged',
          reason_code: 'merged_canonical'
        },
        {
          source_model: 'deleted-model',
          canonical_model: 'deleted-model',
          status: 'skipped',
          reason_code: 'blocked_tombstone'
        },
        {
          source_model: 'bad-model',
          status: 'failed',
          reason_code: 'unsupported_runtime_platform',
          detail: 'runtime unsupported'
        }
      ],
      probe_source: 'upstream',
      trigger: 'manual'
    })

    expect(payload.type).toBe('warning')
    expect(payload.message).toBe(
      'admin.accounts.modelImportSummary:{"imported":0,"merged":1,"skipped":1,"failed":1}'
    )
    expect(payload.options.title).toBe('admin.accounts.modelImportResultTitle')
    expect(payload.options.details?.[0]).toContain('claude-sonnet-4-5-20250929')
    expect(payload.options.details?.[0]).toContain('-> claude-sonnet-4.5')
    expect(payload.options.details?.[2]).toContain('bad-model')
    expect(payload.options.copyText).toContain('runtime unsupported')
    expect(payload.options.persistent).toBe(true)
  })
})

describe('mergeAccountModelImportResults', () => {
  it('merges counts, notices, and results', () => {
    const merged = mergeAccountModelImportResults([
      {
        account_id: 1,
        detected_models: ['model-a'],
        imported_models: ['model-a'],
        imported_count: 1,
        skipped_count: 0,
        failed_models: [],
        model_results: [
          {
            source_model: 'model-a',
            canonical_model: 'model-a',
            status: 'imported',
            reason_code: 'imported_new'
          }
        ],
        probe_source: 'upstream',
        probe_notice: '',
        trigger: 'create'
      },
      {
        account_id: 2,
        detected_models: ['model-b'],
        imported_models: [],
        imported_count: 0,
        skipped_count: 1,
        failed_models: [],
        model_results: [
          {
            source_model: 'model-b',
            canonical_model: 'canonical-b',
            status: 'merged',
            reason_code: 'merged_canonical'
          }
        ],
        probe_source: 'gemini_cli_default_fallback',
        probe_notice: 'fallback notice',
        trigger: 'create'
      }
    ])

    expect(merged?.imported_count).toBe(1)
    expect(merged?.skipped_count).toBe(1)
    expect(merged?.probe_source).toBe('gemini_cli_default_fallback')
    expect(merged?.probe_notice).toBe('fallback notice')
    expect(merged?.model_results).toHaveLength(2)
  })

  it('preserves non-upstream Kiro probe sources', () => {
    const merged = mergeAccountModelImportResults([
      {
        account_id: 1,
        detected_models: ['model-a'],
        imported_models: ['model-a'],
        imported_count: 1,
        skipped_count: 0,
        failed_models: [],
        model_results: [],
        probe_source: 'upstream',
        probe_notice: '',
        trigger: 'manual'
      },
      {
        account_id: 1,
        detected_models: ['claude-sonnet-4.5'],
        imported_models: [],
        imported_count: 0,
        skipped_count: 1,
        failed_models: [],
        model_results: [],
        probe_source: 'kiro_builtin_catalog',
        probe_notice: 'using built-in Kiro model catalog',
        trigger: 'manual'
      }
    ])

    expect(merged?.probe_source).toBe('kiro_builtin_catalog')
    expect(merged?.probe_notice).toBe('using built-in Kiro model catalog')
  })
})

describe('shouldInvalidateModelInventory', () => {
  it('returns true when imported or merged models exist', () => {
    expect(shouldInvalidateModelInventory({
      account_id: 1,
      detected_models: [],
      imported_models: [],
      imported_count: 0,
      skipped_count: 0,
      failed_models: [],
      model_results: [
        {
          source_model: 'legacy-model',
          canonical_model: 'canonical-model',
          status: 'merged',
          reason_code: 'merged_canonical'
        }
      ],
      trigger: 'manual'
    })).toBe(true)
  })

  it('returns false for only skipped and failed models', () => {
    expect(shouldInvalidateModelInventory({
      account_id: 1,
      detected_models: [],
      imported_models: [],
      imported_count: 0,
      skipped_count: 1,
      failed_models: [],
      model_results: [
        {
          source_model: 'deleted-model',
          status: 'skipped',
          reason_code: 'blocked_tombstone'
        }
      ],
      trigger: 'manual'
    })).toBe(false)
  })
})


describe('extractSyncableRegistryModels', () => {
  it('returns unique registry models for imported, merged, and eligible skipped results', () => {
    expect(extractSyncableRegistryModels({
      account_id: 1,
      detected_models: [],
      imported_models: [],
      imported_count: 0,
      skipped_count: 2,
      failed_models: [],
      model_results: [
        {
          source_model: 'model-a-raw',
          canonical_model: 'model-a',
          registry_model: 'model-a',
          status: 'imported',
          reason_code: 'imported_new'
        },
        {
          source_model: 'model-a-alias',
          canonical_model: 'model-a',
          registry_model: 'model-a',
          status: 'skipped',
          reason_code: 'duplicate_canonical'
        },
        {
          source_model: 'model-b-old',
          canonical_model: 'model-b',
          registry_model: 'model-b',
          status: 'merged',
          reason_code: 'merged_canonical'
        },
        {
          source_model: 'deleted-model',
          registry_model: 'deleted-model',
          status: 'skipped',
          reason_code: 'blocked_tombstone'
        },
        {
          source_model: 'failed-model',
          status: 'failed',
          reason_code: 'persist_failed'
        }
      ],
      trigger: 'manual'
    })).toEqual(['model-a', 'model-b'])
  })
})
