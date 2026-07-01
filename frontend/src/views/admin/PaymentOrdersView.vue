<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <Select v-model="filters.status" :options="statusOptions" class="w-44" @change="resetAndLoad" />
          <Select v-model="filters.product_type" :options="productOptions" class="w-44" @change="resetAndLoad" />
          <input
            v-model.number="filters.user_id"
            type="number"
            min="1"
            class="input w-36"
            :placeholder="t('admin.payment.filters.userId')"
            @keyup.enter="resetAndLoad"
          />
          <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadOrders">
            <Icon name="refresh" size="sm" class="mr-2" :class="loading ? 'animate-spin' : ''" />
            {{ t('admin.payment.refresh') }}
          </button>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="orders" :loading="loading">
          <template #cell-order="{ row }">
            <div class="min-w-0">
              <p class="truncate font-mono text-xs text-gray-900 dark:text-white">{{ row.order_no }}</p>
              <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ row.provider_intent_id || '-' }}</p>
            </div>
          </template>
          <template #cell-user="{ row }">
            <span class="font-mono text-xs">#{{ row.user_id }}</span>
          </template>
          <template #cell-product="{ row }">
            {{ productLabel(row.product_type) }}
          </template>
          <template #cell-amount="{ row }">
            <div>
              <span class="font-semibold">{{ formatAmount(row.amount, row.currency) }}</span>
              <p v-if="convertedAmountText(row)" class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                {{ t('admin.payment.convertedAmount', { amount: convertedAmountText(row) }) }}
              </p>
            </div>
          </template>
          <template #cell-refund="{ row }">
            <div class="text-sm">
              <p class="font-medium text-gray-900 dark:text-white">
                {{ formatMinor(row.refunded_amount_minor || 0, row.currency) }}
              </p>
              <p class="text-xs text-gray-500 dark:text-dark-400">
                {{ t('admin.payment.refund.refundable') }} {{ formatMinor(row.refundable_amount_minor || 0, row.currency) }}
              </p>
            </div>
          </template>
          <template #cell-status="{ row }">
            <span :class="statusClass(row.status)">
              {{ t(`purchase.status.${row.status}`) }}
            </span>
          </template>
          <template #cell-provider="{ row }">
            <span class="text-sm text-gray-600 dark:text-dark-300">{{ row.provider }} · {{ row.provider_env }}</span>
          </template>
          <template #cell-created_at="{ row }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDate(row.created_at) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              :disabled="!canRefund(row)"
              @click="openRefund(row)"
            >
              <Icon name="refresh" size="xs" class="mr-1.5" />
              {{ t('admin.payment.refund.action') }}
            </button>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination
          v-if="pagination.total > 0"
          :page="pagination.page"
          :total="pagination.total"
          :page-size="pagination.page_size"
          @update:page="handlePageChange"
          @update:pageSize="handlePageSizeChange"
        />
      </template>
    </TablePageLayout>

    <BaseDialog :show="refundDialogOpen" :title="t('admin.payment.refund.title')" @close="closeRefund">
      <form id="payment-refund-form" class="space-y-4" @submit.prevent="submitRefund">
        <PaymentStatusPanel :order="selectedOrder" />
        <label class="space-y-2">
          <span class="input-label">{{ t('admin.payment.refund.amountMinor') }}</span>
          <input
            v-model.number="refundForm.amount_minor"
            type="number"
            min="1"
            :max="selectedOrder?.refundable_amount_minor || undefined"
            class="input"
          />
          <span class="input-hint">{{ refundAmountHint }}</span>
        </label>
        <label class="space-y-2">
          <span class="input-label">{{ t('admin.payment.refund.reason') }}</span>
          <input v-model.trim="refundForm.reason" type="text" class="input" :placeholder="t('admin.payment.refund.reasonPlaceholder')" />
        </label>
      </form>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="refunding" @click="closeRefund">
          {{ t('common.cancel') }}
        </button>
        <button type="submit" form="payment-refund-form" class="btn btn-primary" :disabled="refunding || !refundAmountValid">
          {{ refunding ? t('admin.payment.refund.submitting') : t('admin.payment.refund.submit') }}
        </button>
      </template>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import PaymentStatusPanel from '@/components/payment/PaymentStatusPanel.vue'
import { adminAPI } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import type { Column } from '@/components/common/types'
import type { PaymentOrder } from '@/types'

const { t, locale } = useI18n()
const appStore = useAppStore()

const orders = ref<PaymentOrder[]>([])
const loading = ref(false)
const refunding = ref(false)
const refundDialogOpen = ref(false)
const selectedOrder = ref<PaymentOrder | null>(null)
const filters = reactive({
  status: '',
  product_type: '',
  user_id: undefined as number | undefined
})
const pagination = reactive({ page: 1, page_size: 20, total: 0, pages: 1 })
const refundForm = reactive({ amount_minor: 0, reason: '' })

const columns = computed<Column[]>(() => [
  { key: 'order', label: t('admin.payment.columns.order') },
  { key: 'user', label: t('admin.payment.columns.user') },
  { key: 'product', label: t('admin.payment.columns.product') },
  { key: 'amount', label: t('admin.payment.columns.amount') },
  { key: 'refund', label: t('admin.payment.columns.refund') },
  { key: 'status', label: t('admin.payment.columns.status') },
  { key: 'provider', label: t('admin.payment.columns.provider') },
  { key: 'created_at', label: t('admin.payment.columns.createdAt') },
  { key: 'actions', label: t('admin.payment.columns.actions') }
])

const statusOptions = computed(() => [
  { value: '', label: t('admin.payment.filters.allStatuses') },
  ...['created', 'pending', 'paid', 'failed', 'cancelled', 'expired', 'partial_refunded', 'refunded'].map((status) => ({
    value: status,
    label: t(`purchase.status.${status}`)
  }))
])
const productOptions = computed(() => [
  { value: '', label: t('admin.payment.filters.allProducts') },
  { value: 'balance_topup', label: t('admin.payment.product.balance_topup') },
  { value: 'subscription', label: t('admin.payment.product.subscription') }
])
const selectedRefundableAmount = computed(() => selectedOrder.value?.refundable_amount_minor || 0)
const refundAmountValid = computed(() =>
  Boolean(selectedOrder.value && refundForm.amount_minor > 0 && refundForm.amount_minor <= selectedRefundableAmount.value)
)
const refundAmountHint = computed(() =>
  t('admin.payment.refund.amountHint', {
    amount: formatMinor(selectedRefundableAmount.value, selectedOrder.value?.currency || 'USD')
  })
)

async function loadOrders() {
  loading.value = true
  try {
    const result = await adminAPI.payment.listOrders(pagination.page, pagination.page_size, {
      status: filters.status || undefined,
      product_type: filters.product_type || undefined,
      user_id: filters.user_id || undefined
    })
    orders.value = result.items || []
    pagination.total = result.total || 0
    pagination.pages = result.pages || 1
    pagination.page = result.page || pagination.page
    pagination.page_size = result.page_size || pagination.page_size
  } catch (err) {
    appStore.showError(resolveErrorMessage(err, t('admin.payment.loadFailed')))
  } finally {
    loading.value = false
  }
}

function resetAndLoad() {
  pagination.page = 1
  void loadOrders()
}

function handlePageChange(page: number) {
  pagination.page = page
  void loadOrders()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page_size = pageSize
  pagination.page = 1
  void loadOrders()
}

function openRefund(order: PaymentOrder) {
  selectedOrder.value = order
  refundForm.amount_minor = order.refundable_amount_minor || Math.max(order.amount_minor - (order.refunded_amount_minor || 0), 0)
  refundForm.reason = ''
  refundDialogOpen.value = true
}

function closeRefund() {
  if (refunding.value) return
  refundDialogOpen.value = false
  selectedOrder.value = null
}

async function submitRefund() {
  if (!selectedOrder.value) return
  if (!refundAmountValid.value) {
    appStore.showError(t('admin.payment.refund.invalidAmount'))
    return
  }
  refunding.value = true
  try {
    await adminAPI.payment.refundOrder(
      selectedOrder.value.order_no,
      { amount_minor: refundForm.amount_minor, reason: refundForm.reason || undefined },
      randomIdempotencyKey()
    )
    appStore.showSuccess(t('admin.payment.refund.success'))
    refundDialogOpen.value = false
    await loadOrders()
  } catch (err) {
    appStore.showError(resolveErrorMessage(err, t('admin.payment.refund.failed')))
  } finally {
    refunding.value = false
  }
}

function canRefund(order: PaymentOrder): boolean {
  return (order.status === 'paid' || order.status === 'partial_refunded') && (order.refundable_amount_minor || 0) > 0
}

function productLabel(productType: string): string {
  return productType === 'subscription'
    ? t('admin.payment.product.subscription')
    : t('admin.payment.product.balance_topup')
}

function statusClass(status: string): string {
  const base = 'inline-flex rounded-full px-2.5 py-1 text-xs font-semibold'
  if (status === 'paid') return `${base} bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300`
  if (status === 'failed' || status === 'cancelled' || status === 'expired') {
    return `${base} bg-red-100 text-red-700 dark:bg-red-500/15 dark:text-red-300`
  }
  if (status === 'refunded' || status === 'partial_refunded') {
    return `${base} bg-sky-100 text-sky-700 dark:bg-sky-500/15 dark:text-sky-300`
  }
  return `${base} bg-amber-100 text-amber-700 dark:bg-amber-500/15 dark:text-amber-300`
}

function formatAmount(amount: number, currency: string): string {
  try {
    return new Intl.NumberFormat(locale.value, { style: 'currency', currency }).format(amount)
  } catch {
    return `${amount.toFixed(2)} ${currency}`
  }
}

function formatMinor(amountMinor: number, currency: string): string {
  const amount = amountMinor / minorUnitFactor(currency)
  return formatAmount(amount, currency)
}

function convertedAmountText(order: PaymentOrder): string {
  if (!order || order.product_type !== 'subscription') return ''
  const currency = String(order.currency || '').trim().toUpperCase()
  if (currency === 'CNY') return ''
  const snapshot = order.snapshot || {}
  const prices = snapshot.prices_by_currency
  if (!prices || typeof prices !== 'object') return ''
  const amount = Number((prices as Record<string, unknown>).CNY)
  if (!Number.isFinite(amount) || amount <= 0) return ''
  return formatAmount(amount, 'CNY')
}

function minorUnitFactor(currency: string): number {
  return ['JPY', 'KRW', 'VND'].includes(currency.toUpperCase()) ? 1 : 100
}

function formatDate(value?: string): string {
  if (!value) return '-'
  return new Intl.DateTimeFormat(locale.value, { dateStyle: 'short', timeStyle: 'short' }).format(new Date(value))
}

function randomIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return `refund-${Date.now()}-${Math.random().toString(36).slice(2)}`
}

function resolveErrorMessage(err: unknown, fallback: string): string {
  if (err && typeof err === 'object' && 'message' in err) {
    return String((err as { message?: unknown }).message || fallback)
  }
  return fallback
}

onMounted(loadOrders)
</script>
