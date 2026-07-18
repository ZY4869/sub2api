import { reactive, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type {
  PromptAuditDeletePreview,
  PromptAuditEvent,
  PromptAuditEventFilter
} from '@/api/admin/promptAudit'
import { cleanEventFilter } from './types'

export function usePromptAuditEvents() {
  const loading = ref(false)
  const deleting = ref(false)
  const events = ref<PromptAuditEvent[]>([])
  const selected = ref<PromptAuditEvent | null>(null)
  const total = ref(0)
  const deletePreview = ref<PromptAuditDeletePreview | null>(null)
  const filters = reactive<PromptAuditEventFilter>({
    page: 1,
    page_size: 20,
    decision: '',
    risk_level: '',
    endpoint: '',
    protocol: '',
    request_id: '',
    prompt_hash: '',
    keyword: '',
    user_id: undefined,
    api_key_id: undefined,
    group_id: undefined
  })

  async function load() {
    loading.value = true
    try {
      const result = await adminAPI.promptAudit.listEvents(cleanEventFilter(filters))
      events.value = result.items || []
      total.value = result.total || 0
      filters.page = result.page || filters.page
      filters.page_size = result.page_size || filters.page_size
    } finally {
      loading.value = false
    }
  }

  async function open(id: number) {
    selected.value = await adminAPI.promptAudit.getEvent(id)
  }

  async function remove(id: number, stepUpTotp: string) {
    deleting.value = true
    try {
      await adminAPI.promptAudit.deleteEvent(id, { stepUpTotp })
      if (selected.value?.id === id) selected.value = null
      await load()
    } finally {
      deleting.value = false
    }
  }

  async function previewDeleteCurrentFilter() {
    deletePreview.value = await adminAPI.promptAudit.previewDelete(cleanEventFilter(filters))
  }

  async function deleteCurrentFilter(stepUpTotp: string) {
    if (!deletePreview.value) return
    deleting.value = true
    try {
      await adminAPI.promptAudit.deleteByFilter(cleanEventFilter(filters), deletePreview.value, { stepUpTotp })
      deletePreview.value = null
      selected.value = null
      filters.page = 1
      await load()
    } finally {
      deleting.value = false
    }
  }

  function resetFilters() {
    filters.page = 1
    filters.decision = ''
    filters.risk_level = ''
    filters.endpoint = ''
    filters.protocol = ''
    filters.request_id = ''
    filters.prompt_hash = ''
    filters.keyword = ''
    filters.user_id = undefined
    filters.api_key_id = undefined
    filters.group_id = undefined
    deletePreview.value = null
  }

  return {
    loading,
    deleting,
    events,
    selected,
    total,
    filters,
    deletePreview,
    load,
    open,
    remove,
    previewDeleteCurrentFilter,
    deleteCurrentFilter,
    resetFilters
  }
}
