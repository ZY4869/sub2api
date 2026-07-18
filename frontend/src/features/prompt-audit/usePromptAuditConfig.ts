import { computed, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { PromptAuditConfig, PromptAuditProbeResult, PromptAuditRuntime } from '@/api/admin/promptAudit'
import { defaultGuardModel, newEndpointId, promptAuditScanners } from './constants'
import {
  draftFromConfig,
  endpointPayloadFromDraft,
  payloadFromDraft,
  type PromptAuditConfigDraft,
  type PromptAuditEndpointDraft
} from './types'

export function usePromptAuditConfig() {
  const config = ref<PromptAuditConfig | null>(null)
  const draft = ref<PromptAuditConfigDraft | null>(null)
  const runtime = ref<PromptAuditRuntime | null>(null)
  const loading = ref(false)
  const saving = ref(false)
  const probing = ref<Record<string, boolean>>({})
  const probeResults = ref<Record<string, PromptAuditProbeResult>>({})

  const ready = computed(() => !!draft.value)

  async function load() {
    loading.value = true
    try {
      const [nextConfig, nextRuntime] = await Promise.all([
        adminAPI.promptAudit.getConfig(),
        adminAPI.promptAudit.getRuntime()
      ])
      config.value = nextConfig
      runtime.value = nextRuntime
      draft.value = draftFromConfig(nextConfig)
    } finally {
      loading.value = false
    }
  }

  async function refreshRuntime() {
    runtime.value = await adminAPI.promptAudit.getRuntime()
  }

  async function save(stepUpTotp: string) {
    if (!draft.value) return
    saving.value = true
    try {
      const next = await adminAPI.promptAudit.updateConfig(payloadFromDraft(draft.value), { stepUpTotp })
      config.value = next
      draft.value = draftFromConfig(next)
      await refreshRuntime()
    } finally {
      saving.value = false
    }
  }

  async function probe(endpoint: PromptAuditEndpointDraft) {
    const key = endpoint.id
    probing.value = { ...probing.value, [key]: true }
    try {
      const result = await adminAPI.promptAudit.probeEndpoint(endpointPayloadFromDraft(endpoint))
      probeResults.value = { ...probeResults.value, [key]: result }
      return result
    } finally {
      probing.value = { ...probing.value, [key]: false }
    }
  }

  function addEndpoint() {
    if (!draft.value) return
    draft.value.endpoints.push({
      id: newEndpointId(),
      name: 'Qwen3Guard',
      protocol: 'openai_compatible',
      base_url: '',
      model: defaultGuardModel,
      timeout_ms: 3000,
      input_limit: 12000,
      enabled: true,
      token: '',
      clear_token: false,
      token_mode: 'replace'
    })
  }

  function removeEndpoint(index: number) {
    draft.value?.endpoints.splice(index, 1)
  }

  function selectAllScanners() {
    if (draft.value) draft.value.scanners = [...promptAuditScanners]
  }

  return {
    config,
    draft,
    runtime,
    loading,
    saving,
    probing,
    probeResults,
    ready,
    load,
    refreshRuntime,
    save,
    probe,
    addEndpoint,
    removeEndpoint,
    selectAllScanners
  }
}
