import type { UsageLog } from '@/types'
import { resolveUsageRequestType } from '@/utils/usageRequestType'

type TranslateFn = (
  key: string,
  params?: Record<string, unknown>,
) => string

type UsageOperationRow = Pick<
  UsageLog,
  | 'operation_type'
  | 'charge_source'
  | 'actual_cost'
  | 'request_type'
  | 'stream'
  | 'openai_ws_mode'
>

export function getUsageOperationLabel(
  row: UsageOperationRow,
  t: TranslateFn,
): string {
  switch (row.operation_type) {
    case 'batch_create':
      return t('usage.operationTypeBatchCreate')
    case 'batch_settlement':
      return t('usage.operationTypeBatchSettlement')
    case 'batch_status':
      return t('usage.operationTypeBatchStatus')
    case 'get_file_metadata':
      return t('usage.operationTypeGetFileMetadata')
    case 'official_result_download':
      return t('usage.operationTypeOfficialResultDownload')
    case 'local_archive_download':
      return t('usage.operationTypeLocalArchiveDownload')
    default: {
      const requestType = resolveUsageRequestType(row)
      if (requestType === 'ws_v2') return t('usage.ws')
      if (requestType === 'stream') return t('usage.stream')
      if (requestType === 'sync') return t('usage.sync')
      return t('usage.unknown')
    }
  }
}

export function getUsageOperationBadgeClass(row: UsageOperationRow): string {
  switch (row.operation_type) {
    case 'batch_create':
      return 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
    case 'batch_settlement':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200'
    case 'batch_status':
      return 'bg-sky-100 text-sky-800 dark:bg-sky-900 dark:text-sky-200'
    case 'get_file_metadata':
      return 'bg-cyan-100 text-cyan-800 dark:bg-cyan-900 dark:text-cyan-200'
    case 'official_result_download':
      return 'bg-slate-100 text-slate-800 dark:bg-slate-800 dark:text-slate-200'
    case 'local_archive_download':
      return 'bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-200'
    default: {
      const requestType = resolveUsageRequestType(row)
      if (requestType === 'ws_v2') {
        return 'bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-200'
      }
      if (requestType === 'stream') {
        return 'bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200'
      }
      if (requestType === 'sync') {
        return 'bg-gray-100 text-gray-800 dark:bg-gray-700 dark:text-gray-200'
      }
      return 'bg-amber-100 text-amber-800 dark:bg-amber-900 dark:text-amber-200'
    }
  }
}

export function getUsageChargeLabel(
  row: UsageOperationRow,
  t: TranslateFn,
): string | null {
  switch (row.charge_source) {
    case 'model_batch':
      return t('usage.chargeSourceModelBatch')
    case 'archive_download':
      return t('usage.chargeSourceArchiveDownload')
    case 'none':
      return t('usage.chargeSourceNone')
    default:
      return row.operation_type ? t('usage.notCharged') : null
  }
}

export function getUsageChargeBadgeClass(row: UsageOperationRow): string {
  switch (row.charge_source) {
    case 'model_batch':
      return 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200'
    case 'archive_download':
      return 'bg-violet-100 text-violet-800 dark:bg-violet-900 dark:text-violet-200'
    case 'none':
      return 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
    default:
      return Number(row.actual_cost || 0) > 0
        ? 'bg-emerald-100 text-emerald-800 dark:bg-emerald-900 dark:text-emerald-200'
        : 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-200'
  }
}
