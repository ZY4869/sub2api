import { computed, onBeforeUnmount, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { AdminAccountImportJob, AdminAccountImportJobStatus, AdminDataImportResult } from '@/types'

const terminalStatuses = new Set<AdminAccountImportJobStatus>([
  'succeeded',
  'partial_failed',
  'failed',
  'cancelled'
])

export function useAccountImportJobPolling() {
  const job = ref<AdminAccountImportJob | null>(null)
  const result = ref<AdminDataImportResult | null>(null)
  const pollTimer = ref<number | null>(null)

  const canCancelJob = computed(() =>
    Boolean(job.value && !terminalStatuses.has(job.value.status))
  )
  const progressPercent = computed(() => {
    const progress = job.value?.progress
    if (!progress || progress.total <= 0) return job.value ? 100 : 0
    return Math.min(100, Math.round((progress.processed / progress.total) * 100))
  })

  const clearPollTimer = () => {
    if (!pollTimer.value) return
    window.clearTimeout(pollTimer.value)
    pollTimer.value = null
  }

  const resetImportJob = () => {
    clearPollTimer()
    job.value = null
    result.value = null
  }

  const updateFromJob = (nextJob: AdminAccountImportJob) => {
    job.value = nextJob
    result.value = nextJob.result
  }

  const pollImportJob = async (jobId: string): Promise<AdminAccountImportJob> => {
    const nextJob = await adminAPI.accounts.getImportJob(jobId)
    updateFromJob(nextJob)
    if (terminalStatuses.has(nextJob.status)) {
      return nextJob
    }
    await new Promise<void>((resolve) => {
      pollTimer.value = window.setTimeout(resolve, 800)
    })
    return pollImportJob(jobId)
  }

  onBeforeUnmount(clearPollTimer)

  return {
    job,
    result,
    canCancelJob,
    progressPercent,
    resetImportJob,
    updateFromJob,
    pollImportJob,
    clearPollTimer
  }
}
