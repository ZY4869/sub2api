<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <SubscriptionsToolbar
          v-model:filter-user-keyword="filterUserKeyword"
          v-model:show-filter-user-dropdown="showFilterUserDropdown"
          v-model:show-column-dropdown="showColumnDropdown"
          :filter-user-results="filterUserResults"
          :filter-user-loading="filterUserLoading"
          :selected-filter-user="selectedFilterUser"
          :filters="filters"
          :status-options="statusOptions"
          :group-options="groupOptions"
          :platform-filter-options="platformFilterOptions"
          :loading="loading"
          :user-column-mode="userColumnMode"
          :toggleable-columns="toggleableColumns"
          :is-column-visible="isColumnVisible"
          @search-filter-users="debounceSearchFilterUsers"
          @select-filter-user="selectFilterUser"
          @clear-filter-user="clearFilterUser"
          @apply-filters="applyFilters"
          @load="loadSubscriptions"
          @set-user-column-mode="setUserColumnMode"
          @toggle-column="toggleColumn"
          @show-guide="showGuideModal = true"
          @assign="showAssignModal = true"
        />
      </template>

      <!-- Subscriptions Table -->
      <template #table>
        <SubscriptionsTable
          :columns="columns"
          :subscriptions="subscriptions"
          :loading="loading"
          :user-column-mode="userColumnMode"
          :resetting-quota="resettingQuota"
          :resetting-subscription-id="resettingSubscription?.id ?? null"
          @sort="handleSort"
          @extend="handleExtend"
          @reset-quota="handleResetQuota"
          @revoke="handleRevoke"
          @assign="showAssignModal = true"
        />
      </template>

      <!-- Pagination -->
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

    <!-- Assign Subscription Modal -->
    <BaseDialog
      :show="showAssignModal"
      :title="t('admin.subscriptions.assignSubscription')"
      width="normal"
      @close="closeAssignModal"
    >
      <form
        id="assign-subscription-form"
        @submit.prevent="handleAssignSubscription"
        class="space-y-5"
      >
        <div>
          <label class="input-label">{{ t('admin.subscriptions.form.user') }}</label>
          <div class="relative" data-assign-user-search>
            <input
              v-model="userSearchKeyword"
              type="text"
              class="input pr-8"
              :placeholder="t('admin.usage.searchUserPlaceholder')"
              @input="debounceSearchUsers"
              @focus="showUserDropdown = true"
            />
            <button
              v-if="selectedUser"
              @click="clearUserSelection"
              type="button"
              class="absolute right-2 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            >
              <Icon name="x" size="sm" :stroke-width="2" />
            </button>
            <!-- User Dropdown -->
            <div
              v-if="showUserDropdown && (userSearchResults.length > 0 || userSearchKeyword)"
              class="absolute z-50 mt-1 max-h-60 w-full overflow-auto rounded-lg border border-gray-200 bg-white shadow-lg dark:border-gray-700 dark:bg-gray-800"
            >
              <div
                v-if="userSearchLoading"
                class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400"
              >
                {{ t('common.loading') }}
              </div>
              <div
                v-else-if="userSearchResults.length === 0 && userSearchKeyword"
                class="px-4 py-3 text-sm text-gray-500 dark:text-gray-400"
              >
                {{ t('common.noOptionsFound') }}
              </div>
              <button
                v-for="user in userSearchResults"
                :key="user.id"
                type="button"
                @click="selectUser(user)"
                class="w-full px-4 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-gray-700"
              >
                <span class="font-medium text-gray-900 dark:text-white">{{ user.email }}</span>
                <span class="ml-2 text-gray-500 dark:text-gray-400">#{{ user.id }}</span>
              </button>
            </div>
          </div>
        </div>
        <div>
          <label class="input-label">{{ t('admin.subscriptions.form.group') }}</label>
          <Select
            v-model="assignForm.group_id"
            :options="subscriptionGroupOptions"
            :placeholder="t('admin.subscriptions.selectGroup')"
          >
            <template #selected="{ option }">
              <GroupBadge
                v-if="option"
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
              />
              <span v-else class="text-gray-400">{{ t('admin.subscriptions.selectGroup') }}</span>
            </template>
            <template #option="{ option, selected }">
              <GroupOptionItem
                :name="(option as unknown as GroupOption).label"
                :platform="(option as unknown as GroupOption).platform"
                :subscription-type="(option as unknown as GroupOption).subscriptionType"
                :rate-multiplier="(option as unknown as GroupOption).rate"
                :description="(option as unknown as GroupOption).description"
                :selected="selected"
              />
            </template>
          </Select>
          <p class="input-hint">{{ t('admin.subscriptions.groupHint') }}</p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.subscriptions.form.validityDays') }}</label>
          <input v-model.number="assignForm.validity_days" type="number" min="1" class="input" />
          <p class="input-hint">{{ t('admin.subscriptions.validityHint') }}</p>
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button @click="closeAssignModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            form="assign-subscription-form"
            :disabled="submitting"
            class="btn btn-primary"
          >
            <svg
              v-if="submitting"
              class="-ml-1 mr-2 h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{ submitting ? t('admin.subscriptions.assigning') : t('admin.subscriptions.assign') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Adjust Subscription Modal -->
    <BaseDialog
      :show="showExtendModal"
      :title="t('admin.subscriptions.adjustSubscription')"
      width="narrow"
      @close="closeExtendModal"
    >
      <form
        v-if="extendingSubscription"
        id="extend-subscription-form"
        @submit.prevent="handleExtendSubscription"
        class="space-y-5"
      >
        <div class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
          <p class="text-sm text-gray-600 dark:text-gray-400">
            {{ t('admin.subscriptions.adjustingFor') }}
            <span class="font-medium text-gray-900 dark:text-white">{{
              extendingSubscription.user?.email
            }}</span>
          </p>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-400">
            {{ t('admin.subscriptions.currentExpiration') }}:
            <span class="font-medium text-gray-900 dark:text-white">
              {{
                extendingSubscription.expires_at
                  ? formatDateOnly(extendingSubscription.expires_at)
                  : t('admin.subscriptions.noExpiration')
              }}
            </span>
          </p>
          <p v-if="extendingSubscription.expires_at" class="mt-1 text-sm text-gray-600 dark:text-gray-400">
            {{ t('admin.subscriptions.remainingDays') }}:
            <span class="font-medium text-gray-900 dark:text-white">
              {{ getDaysRemaining(extendingSubscription.expires_at) ?? 0 }}
            </span>
          </p>
        </div>
        <div>
          <label class="input-label">{{ t('admin.subscriptions.form.adjustDays') }}</label>
          <div class="flex items-center gap-2">
            <input
              v-model.number="extendForm.days"
              type="number"
              required
              class="input text-center"
              :placeholder="t('admin.subscriptions.adjustDaysPlaceholder')"
            />
          </div>
          <p class="input-hint">{{ t('admin.subscriptions.adjustHint') }}</p>
        </div>
      </form>
      <template #footer>
        <div v-if="extendingSubscription" class="flex justify-end gap-3">
          <button @click="closeExtendModal" type="button" class="btn btn-secondary">
            {{ t('common.cancel') }}
          </button>
          <button
            type="submit"
            form="extend-subscription-form"
            :disabled="submitting"
            class="btn btn-primary"
          >
            {{ submitting ? t('admin.subscriptions.adjusting') : t('admin.subscriptions.adjust') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <!-- Revoke Confirmation Dialog -->
    <ConfirmDialog
      :show="showRevokeDialog"
      :title="t('admin.subscriptions.revokeSubscription')"
      :message="t('admin.subscriptions.revokeConfirm', { user: revokingSubscription?.user?.email })"
      :confirm-text="t('admin.subscriptions.revoke')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmRevoke"
      @cancel="showRevokeDialog = false"
    />

    <!-- Reset Quota Confirmation Dialog -->
    <ConfirmDialog
      :show="showResetQuotaConfirm"
      :title="t('admin.subscriptions.resetQuotaTitle')"
      :message="t('admin.subscriptions.resetQuotaConfirm', { user: resettingSubscription?.user?.email })"
      :confirm-text="t('admin.subscriptions.resetQuota')"
      :cancel-text="t('common.cancel')"
      @confirm="confirmResetQuota"
      @cancel="showResetQuotaConfirm = false"
    />
    <!-- Subscription Guide Modal -->
    <teleport to="body">
      <transition name="modal">
        <div v-if="showGuideModal" class="fixed inset-0 z-50 flex items-center justify-center p-4" @mousedown.self="showGuideModal = false">
          <div class="fixed inset-0 bg-black/50" @click="showGuideModal = false"></div>
          <div class="relative max-h-[85vh] w-full max-w-2xl overflow-y-auto rounded-xl bg-white p-6 shadow-2xl dark:bg-dark-800">
            <button type="button" class="absolute right-4 top-4 text-gray-400 hover:text-gray-600 dark:hover:text-gray-200" @click="showGuideModal = false">
              <svg class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" stroke-width="2"><path stroke-linecap="round" stroke-linejoin="round" d="M6 18L18 6M6 6l12 12" /></svg>
            </button>

            <h2 class="mb-4 text-lg font-bold text-gray-900 dark:text-white">{{ t('admin.subscriptions.guide.title') }}</h2>
            <p class="mb-5 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.subscriptions.guide.subtitle') }}</p>

            <!-- Step 1 -->
            <div class="mb-5">
              <h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <span class="flex h-6 w-6 items-center justify-center rounded-full bg-primary-100 text-xs font-bold text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">1</span>
                {{ t('admin.subscriptions.guide.step1.title') }}
              </h3>
              <ol class="ml-8 list-decimal space-y-1 text-sm text-gray-600 dark:text-gray-300">
                <li>{{ t('admin.subscriptions.guide.step1.line1') }}</li>
                <li>{{ t('admin.subscriptions.guide.step1.line2') }}</li>
                <li>{{ t('admin.subscriptions.guide.step1.line3') }}</li>
              </ol>
              <div class="ml-8 mt-2">
                <router-link
                  to="/admin/groups"
                  @click="showGuideModal = false"
                  class="inline-flex items-center gap-1 text-sm font-medium text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                >
                  {{ t('admin.subscriptions.guide.step1.link') }}
                  <Icon name="arrowRight" size="xs" />
                </router-link>
              </div>
            </div>

            <!-- Step 2 -->
            <div class="mb-5">
              <h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <span class="flex h-6 w-6 items-center justify-center rounded-full bg-primary-100 text-xs font-bold text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">2</span>
                {{ t('admin.subscriptions.guide.step2.title') }}
              </h3>
              <ol class="ml-8 list-decimal space-y-1 text-sm text-gray-600 dark:text-gray-300">
                <li>{{ t('admin.subscriptions.guide.step2.line1') }}</li>
                <li>{{ t('admin.subscriptions.guide.step2.line2') }}</li>
                <li>{{ t('admin.subscriptions.guide.step2.line3') }}</li>
              </ol>
            </div>

            <!-- Step 3 -->
            <div class="mb-5">
              <h3 class="mb-2 flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
                <span class="flex h-6 w-6 items-center justify-center rounded-full bg-primary-100 text-xs font-bold text-primary-700 dark:bg-primary-900/40 dark:text-primary-300">3</span>
                {{ t('admin.subscriptions.guide.step3.title') }}
              </h3>
              <div class="ml-8 overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600">
                <table class="w-full text-sm">
                  <tbody>
                    <tr v-for="(row, i) in guideActionRows" :key="i" class="border-b border-gray-100 dark:border-dark-700 last:border-0">
                      <td class="whitespace-nowrap bg-gray-50 px-3 py-2 font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-300">{{ row.action }}</td>
                      <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ row.desc }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>

            <!-- Tip -->
            <div class="rounded-lg bg-blue-50 p-3 text-xs text-blue-700 dark:bg-blue-900/20 dark:text-blue-300">
              {{ t('admin.subscriptions.guide.tip') }}
            </div>

            <div class="mt-4 text-right">
              <button type="button" class="btn btn-primary btn-sm" @click="showGuideModal = false">{{ t('common.close') }}</button>
            </div>
          </div>
        </div>
      </transition>
    </teleport>
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { UserSubscription, Group, GroupPlatform, SubscriptionType } from '@/types'
import type { SimpleUser } from '@/api/admin/usage'
import type { Column } from '@/components/common/types'
import { formatDateOnly } from '@/utils/format'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { FILTER_PLATFORM_ORDER } from '@/utils/platformBranding'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Select from '@/components/common/Select.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import Icon from '@/components/icons/Icon.vue'
import SubscriptionsTable from './subscriptions/SubscriptionsTable.vue'
import SubscriptionsToolbar from './subscriptions/SubscriptionsToolbar.vue'
import { getSubscriptionDaysRemaining } from './subscriptions/utils'

const { t } = useI18n()
const appStore = useAppStore()

interface GroupOption {
  value: number
  label: string
  description: string | null
  platform: GroupPlatform
  subscriptionType: SubscriptionType
  rate: number
}

// Guide modal state
const showGuideModal = ref(false)

const guideActionRows = computed(() => [
  { action: t('admin.subscriptions.guide.actions.adjust'), desc: t('admin.subscriptions.guide.actions.adjustDesc') },
  { action: t('admin.subscriptions.guide.actions.resetQuota'), desc: t('admin.subscriptions.guide.actions.resetQuotaDesc') },
  { action: t('admin.subscriptions.guide.actions.revoke'), desc: t('admin.subscriptions.guide.actions.revokeDesc') }
])

// User column display mode: 'email' or 'username'
const userColumnMode = ref<'email' | 'username'>('email')
const USER_COLUMN_MODE_KEY = 'subscription-user-column-mode'

const loadUserColumnMode = () => {
  try {
    const saved = localStorage.getItem(USER_COLUMN_MODE_KEY)
    if (saved === 'email' || saved === 'username') {
      userColumnMode.value = saved
    }
  } catch (e) {
    console.error('Failed to load user column mode:', e)
  }
}

const saveUserColumnMode = () => {
  try {
    localStorage.setItem(USER_COLUMN_MODE_KEY, userColumnMode.value)
  } catch (e) {
    console.error('Failed to save user column mode:', e)
  }
}

const setUserColumnMode = (mode: 'email' | 'username') => {
  userColumnMode.value = mode
  saveUserColumnMode()
}

// All available columns
const allColumns = computed<Column[]>(() => [
  {
    key: 'user',
    label: userColumnMode.value === 'email'
      ? t('admin.subscriptions.columns.user')
      : t('admin.users.columns.username'),
    sortable: false
  },
  { key: 'group', label: t('admin.subscriptions.columns.group'), sortable: false },
  { key: 'usage', label: t('admin.subscriptions.columns.usage'), sortable: false },
  { key: 'expires_at', label: t('admin.subscriptions.columns.expires'), sortable: true },
  { key: 'status', label: t('admin.subscriptions.columns.status'), sortable: true },
  { key: 'actions', label: t('admin.subscriptions.columns.actions'), sortable: false }
])

// Columns that can be toggled (exclude user and actions which are always visible)
const toggleableColumns = computed(() =>
  allColumns.value.filter(col => col.key !== 'user' && col.key !== 'actions')
)

// Hidden columns set
const hiddenColumns = reactive<Set<string>>(new Set())

// Default hidden columns
const DEFAULT_HIDDEN_COLUMNS: string[] = []

// localStorage key
const HIDDEN_COLUMNS_KEY = 'subscription-hidden-columns'

// Load saved column settings
const loadSavedColumns = () => {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      parsed.forEach(key => hiddenColumns.add(key))
    } else {
      DEFAULT_HIDDEN_COLUMNS.forEach(key => hiddenColumns.add(key))
    }
  } catch (e) {
    console.error('Failed to load saved columns:', e)
    DEFAULT_HIDDEN_COLUMNS.forEach(key => hiddenColumns.add(key))
  }
}

// Save column settings to localStorage
const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
  } catch (e) {
    console.error('Failed to save columns:', e)
  }
}

// Toggle column visibility
const toggleColumn = (key: string) => {
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
}

// Check if column is visible
const isColumnVisible = (key: string) => !hiddenColumns.has(key)

// Filtered columns for display
const columns = computed<Column[]>(() =>
  allColumns.value.filter(col =>
    col.key === 'user' || col.key === 'actions' || !hiddenColumns.has(col.key)
  )
)

// Column dropdown state
const showColumnDropdown = ref(false)
// Filter options
const statusOptions = computed(() => [
  { value: '', label: t('admin.subscriptions.allStatus') },
  { value: 'active', label: t('admin.subscriptions.status.active') },
  { value: 'expired', label: t('admin.subscriptions.status.expired') },
  { value: 'revoked', label: t('admin.subscriptions.status.revoked') }
])

const subscriptions = ref<UserSubscription[]>([])
const groups = ref<Group[]>([])
const loading = ref(false)
let abortController: AbortController | null = null

// Toolbar user filter (fuzzy search -> select user_id)
const filterUserKeyword = ref('')
const filterUserResults = ref<SimpleUser[]>([])
const filterUserLoading = ref(false)
const showFilterUserDropdown = ref(false)
const selectedFilterUser = ref<SimpleUser | null>(null)
let filterUserSearchTimeout: ReturnType<typeof setTimeout> | null = null

// User search state
const userSearchKeyword = ref('')
const userSearchResults = ref<SimpleUser[]>([])
const userSearchLoading = ref(false)
const showUserDropdown = ref(false)
const selectedUser = ref<SimpleUser | null>(null)
let userSearchTimeout: ReturnType<typeof setTimeout> | null = null

const filters = reactive({
  status: 'active',
  group_id: '',
  platform: '',
  user_id: null as number | null
})

// Sorting state
const sortState = reactive({
  sort_by: 'created_at',
  sort_order: 'desc' as 'asc' | 'desc'
})

const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const showAssignModal = ref(false)
const showExtendModal = ref(false)
const showRevokeDialog = ref(false)
const showResetQuotaConfirm = ref(false)
const submitting = ref(false)
const resettingSubscription = ref<UserSubscription | null>(null)
const resettingQuota = ref(false)
const extendingSubscription = ref<UserSubscription | null>(null)
const revokingSubscription = ref<UserSubscription | null>(null)

const assignForm = reactive({
  user_id: null as number | null,
  group_id: null as number | null,
  validity_days: 30
})

const extendForm = reactive({
  days: 30
})

// Group options for filter (all groups)
const groupOptions = computed(() => [
  { value: '', label: t('admin.subscriptions.allGroups') },
  ...groups.value.map((g) => ({ value: g.id.toString(), label: g.name }))
])

const platformFilterOptions = computed(() => [
  { value: '', label: t('admin.subscriptions.allPlatforms') },
  ...FILTER_PLATFORM_ORDER.map((platform) => ({
    value: platform,
    label: t(`admin.accounts.platforms.${platform}`)
  }))
])

// Group options for assign (only subscription type groups)
const subscriptionGroupOptions = computed(() =>
  groups.value
    .filter((g) => g.subscription_type === 'subscription' && g.status === 'active')
    .map((g) => ({
      value: g.id,
      label: g.name,
      description: g.description,
      platform: g.platform,
      subscriptionType: g.subscription_type,
      rate: g.rate_multiplier
    }))
)

const applyFilters = () => {
  pagination.page = 1
  loadSubscriptions()
}

const loadSubscriptions = async () => {
  if (abortController) {
    abortController.abort()
  }
  const requestController = new AbortController()
  abortController = requestController
  const { signal } = requestController

  loading.value = true
  try {
    const response = await adminAPI.subscriptions.list(
      pagination.page,
      pagination.page_size,
      {
        status: (filters.status as any) || undefined,
        group_id: filters.group_id ? parseInt(filters.group_id) : undefined,
        platform: filters.platform || undefined,
        user_id: filters.user_id || undefined,
        sort_by: sortState.sort_by,
        sort_order: sortState.sort_order
      },
      {
        signal
      }
    )
    if (signal.aborted || abortController !== requestController) return
    subscriptions.value = response.items
    pagination.total = response.total
    pagination.pages = response.pages
  } catch (error: any) {
    if (signal.aborted || error?.name === 'AbortError' || error?.code === 'ERR_CANCELED') {
      return
    }
    appStore.showError(t('admin.subscriptions.failedToLoad'))
    console.error('Error loading subscriptions:', error)
  } finally {
    if (abortController === requestController) {
      loading.value = false
      abortController = null
    }
  }
}

const loadGroups = async () => {
  try {
    groups.value = await adminAPI.groups.getAll()
  } catch (error) {
    console.error('Error loading groups:', error)
  }
}

// Toolbar user filter search with debounce
const debounceSearchFilterUsers = () => {
  if (filterUserSearchTimeout) {
    clearTimeout(filterUserSearchTimeout)
  }
  filterUserSearchTimeout = setTimeout(searchFilterUsers, 300)
}

const searchFilterUsers = async () => {
  const keyword = filterUserKeyword.value.trim()

  // Clear active user filter if user modified the search keyword
  if (selectedFilterUser.value && keyword !== selectedFilterUser.value.email) {
    selectedFilterUser.value = null
    filters.user_id = null
    applyFilters()
  }

  if (!keyword) {
    filterUserResults.value = []
    return
  }

  filterUserLoading.value = true
  try {
    filterUserResults.value = await adminAPI.usage.searchUsers(keyword)
  } catch (error) {
    console.error('Failed to search users:', error)
    filterUserResults.value = []
  } finally {
    filterUserLoading.value = false
  }
}

const selectFilterUser = (user: SimpleUser) => {
  selectedFilterUser.value = user
  filterUserKeyword.value = user.email
  showFilterUserDropdown.value = false
  filters.user_id = user.id
  applyFilters()
}

const clearFilterUser = () => {
  selectedFilterUser.value = null
  filterUserKeyword.value = ''
  filterUserResults.value = []
  showFilterUserDropdown.value = false
  filters.user_id = null
  applyFilters()
}

// User search with debounce
const debounceSearchUsers = () => {
  if (userSearchTimeout) {
    clearTimeout(userSearchTimeout)
  }
  userSearchTimeout = setTimeout(searchUsers, 300)
}

const searchUsers = async () => {
  const keyword = userSearchKeyword.value.trim()

  // Clear selection if user modified the search keyword
  if (selectedUser.value && keyword !== selectedUser.value.email) {
    selectedUser.value = null
    assignForm.user_id = null
  }

  if (!keyword) {
    userSearchResults.value = []
    return
  }

  userSearchLoading.value = true
  try {
    userSearchResults.value = await adminAPI.usage.searchUsers(keyword)
  } catch (error) {
    console.error('Failed to search users:', error)
    userSearchResults.value = []
  } finally {
    userSearchLoading.value = false
  }
}

const selectUser = (user: SimpleUser) => {
  selectedUser.value = user
  userSearchKeyword.value = user.email
  showUserDropdown.value = false
  assignForm.user_id = user.id
}

const clearUserSelection = () => {
  selectedUser.value = null
  userSearchKeyword.value = ''
  userSearchResults.value = []
  assignForm.user_id = null
}

const handlePageChange = (page: number) => {
  pagination.page = page
  loadSubscriptions()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  loadSubscriptions()
}

const handleSort = (key: string, order: 'asc' | 'desc') => {
  sortState.sort_by = key
  sortState.sort_order = order
  pagination.page = 1
  loadSubscriptions()
}

const closeAssignModal = () => {
  showAssignModal.value = false
  assignForm.user_id = null
  assignForm.group_id = null
  assignForm.validity_days = 30
  // Clear user search state
  selectedUser.value = null
  userSearchKeyword.value = ''
  userSearchResults.value = []
  showUserDropdown.value = false
}

const handleAssignSubscription = async () => {
  if (!assignForm.user_id) {
    appStore.showError(t('admin.subscriptions.pleaseSelectUser'))
    return
  }
  if (!assignForm.group_id) {
    appStore.showError(t('admin.subscriptions.pleaseSelectGroup'))
    return
  }
  if (!assignForm.validity_days || assignForm.validity_days < 1) {
    appStore.showError(t('admin.subscriptions.validityDaysRequired'))
    return
  }

  submitting.value = true
  try {
    await adminAPI.subscriptions.assign({
      user_id: assignForm.user_id,
      group_id: assignForm.group_id,
      validity_days: assignForm.validity_days
    })
    appStore.showSuccess(t('admin.subscriptions.subscriptionAssigned'))
    closeAssignModal()
    loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.subscriptions.failedToAssign'))
    console.error('Error assigning subscription:', error)
  } finally {
    submitting.value = false
  }
}

const handleExtend = (subscription: UserSubscription) => {
  extendingSubscription.value = subscription
  extendForm.days = 30
  showExtendModal.value = true
}

const closeExtendModal = () => {
  showExtendModal.value = false
  extendingSubscription.value = null
}

const handleExtendSubscription = async () => {
  if (!extendingSubscription.value) return

  // 前端验证：调整后的过期时间必须在未来
  if (extendingSubscription.value.expires_at) {
    const expiresAt = new Date(extendingSubscription.value.expires_at)
    const newExpiresAt = new Date(expiresAt.getTime() + extendForm.days * 24 * 60 * 60 * 1000)
    if (newExpiresAt <= new Date()) {
      appStore.showError(t('admin.subscriptions.adjustWouldExpire'))
      return
    }
  }

  submitting.value = true
  try {
    await adminAPI.subscriptions.extend(extendingSubscription.value.id, {
      days: extendForm.days
    })
    appStore.showSuccess(t('admin.subscriptions.subscriptionAdjusted'))
    closeExtendModal()
    loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.subscriptions.failedToAdjust'))
    console.error('Error adjusting subscription:', error)
  } finally {
    submitting.value = false
  }
}

const handleRevoke = (subscription: UserSubscription) => {
  revokingSubscription.value = subscription
  showRevokeDialog.value = true
}

const confirmRevoke = async () => {
  if (!revokingSubscription.value) return

  try {
    await adminAPI.subscriptions.revoke(revokingSubscription.value.id)
    appStore.showSuccess(t('admin.subscriptions.subscriptionRevoked'))
    showRevokeDialog.value = false
    revokingSubscription.value = null
    loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.subscriptions.failedToRevoke'))
    console.error('Error revoking subscription:', error)
  }
}

const handleResetQuota = (subscription: UserSubscription) => {
  resettingSubscription.value = subscription
  showResetQuotaConfirm.value = true
}

const confirmResetQuota = async () => {
  if (!resettingSubscription.value) return
  if (resettingQuota.value) return
  resettingQuota.value = true
  try {
    await adminAPI.subscriptions.resetQuota(resettingSubscription.value.id, { daily: true, weekly: true, monthly: true })
    appStore.showSuccess(t('admin.subscriptions.quotaResetSuccess'))
    showResetQuotaConfirm.value = false
    resettingSubscription.value = null
    await loadSubscriptions()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.subscriptions.failedToResetQuota'))
    console.error('Error resetting quota:', error)
  } finally {
    resettingQuota.value = false
  }
}

const getDaysRemaining = getSubscriptionDaysRemaining

// Handle click outside to close dropdowns
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (!target.closest('[data-assign-user-search]')) showUserDropdown.value = false
  if (!target.closest('[data-filter-user-search]')) showFilterUserDropdown.value = false
  if (!target.closest('[data-subscriptions-column-dropdown]')) {
    showColumnDropdown.value = false
  }
}

onMounted(() => {
  loadUserColumnMode()
  loadSavedColumns()
  loadSubscriptions()
  loadGroups()
  document.addEventListener('click', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  if (filterUserSearchTimeout) {
    clearTimeout(filterUserSearchTimeout)
  }
  if (userSearchTimeout) {
    clearTimeout(userSearchTimeout)
  }
})
</script>
