<template>
        <div class="space-y-6">
        <!-- Default Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.defaults.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.defaults.description') }}
            </p>
          </div>
          <div class="space-y-6 p-6">
            <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.defaults.defaultBalance') }}
                </label>
                <input
                  v-model.number="form.default_balance"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input"
                  placeholder="0.00"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.defaults.defaultBalanceHint') }}
                </p>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.defaults.defaultConcurrency') }}
                </label>
                <input
                  v-model.number="form.default_concurrency"
                  type="number"
                  min="1"
                  class="input"
                  placeholder="1"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.defaults.defaultConcurrencyHint') }}
                </p>
              </div>
            </div>

            <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
              <label class="font-medium text-gray-900 dark:text-white">
                {{ t('admin.settings.defaults.defaultApiKeyModelBindingMode') }}
              </label>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.defaults.defaultApiKeyModelBindingModeHint') }}
              </p>
              <div class="mt-3 grid grid-cols-1 gap-3 md:grid-cols-2">
                <button
                  type="button"
                  class="rounded-lg border px-4 py-3 text-left transition"
                  :class="form.default_api_key_model_binding_mode === 'group_allowed'
                    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/20 dark:text-primary-200'
                    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200'"
                  @click="form.default_api_key_model_binding_mode = 'group_allowed'"
                >
                  <div class="text-sm font-semibold">
                    {{ t('admin.settings.defaults.defaultApiKeyModeGroup') }}
                  </div>
                  <p class="mt-1 text-xs opacity-80">
                    {{ t('admin.settings.defaults.defaultApiKeyModeGroupHint') }}
                  </p>
                </button>
                <button
                  type="button"
                  class="rounded-lg border px-4 py-3 text-left transition"
                  :class="form.default_api_key_model_binding_mode === 'model_required'
                    ? 'border-primary-500 bg-primary-50 text-primary-700 dark:border-primary-400 dark:bg-primary-900/20 dark:text-primary-200'
                    : 'border-gray-200 bg-white text-gray-700 hover:border-primary-300 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-200'"
                  @click="form.default_api_key_model_binding_mode = 'model_required'"
                >
                  <div class="text-sm font-semibold">
                    {{ t('admin.settings.defaults.defaultApiKeyModePublicModel') }}
                  </div>
                  <p class="mt-1 text-xs opacity-80">
                    {{ t('admin.settings.defaults.defaultApiKeyModePublicModelHint') }}
                  </p>
                </button>
              </div>
            </div>

            <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
              <div class="mb-3 flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">
                    {{ t('admin.settings.defaults.defaultSubscriptions') }}
                  </label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.defaults.defaultSubscriptionsHint') }}
                  </p>
                </div>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  @click="addDefaultSubscription"
                  :disabled="subscriptionGroups.length === 0"
                >
                  {{ t('admin.settings.defaults.addDefaultSubscription') }}
                </button>
              </div>

              <div
                v-if="form.default_subscriptions.length === 0"
                class="rounded border border-dashed border-gray-300 px-4 py-3 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
              >
                {{ t('admin.settings.defaults.defaultSubscriptionsEmpty') }}
              </div>

              <div v-else class="space-y-3">
                <div
                  v-for="(item, index) in form.default_subscriptions"
                  :key="`default-sub-${index}`"
                  class="grid grid-cols-1 gap-3 rounded border border-gray-200 p-3 md:grid-cols-[1fr_160px_auto] dark:border-dark-600"
                >
                  <div>
                    <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                      {{ t('admin.settings.defaults.subscriptionGroup') }}
                    </label>
                    <Select
                      v-model="item.group_id"
                      class="default-sub-group-select"
                      :options="defaultSubscriptionGroupOptions"
                      :placeholder="t('admin.settings.defaults.subscriptionGroup')"
                    >
                      <template #selected="{ option }">
                        <GroupBadge
                          v-if="option"
                          :name="(option as unknown as DefaultSubscriptionGroupOption).label"
                          :platform="(option as unknown as DefaultSubscriptionGroupOption).platform"
                          :subscription-type="(option as unknown as DefaultSubscriptionGroupOption).subscriptionType"
                          :rate-multiplier="(option as unknown as DefaultSubscriptionGroupOption).rate"
                        />
                        <span v-else class="text-gray-400">
                          {{ t('admin.settings.defaults.subscriptionGroup') }}
                        </span>
                      </template>
                      <template #option="{ option, selected }">
                        <GroupOptionItem
                          :name="(option as unknown as DefaultSubscriptionGroupOption).label"
                          :platform="(option as unknown as DefaultSubscriptionGroupOption).platform"
                          :subscription-type="(option as unknown as DefaultSubscriptionGroupOption).subscriptionType"
                          :rate-multiplier="(option as unknown as DefaultSubscriptionGroupOption).rate"
                          :description="(option as unknown as DefaultSubscriptionGroupOption).description"
                          :selected="selected"
                        />
                      </template>
                    </Select>
                  </div>
                  <div>
                    <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                      {{ t('admin.settings.defaults.subscriptionValidityDays') }}
                    </label>
                    <input
                      v-model.number="item.validity_days"
                      type="number"
                      min="1"
                      max="36500"
                      class="input h-[42px]"
                    />
                  </div>
                  <div class="flex items-end">
                    <button
                      type="button"
                      class="btn btn-secondary default-sub-delete-btn w-full text-red-600 hover:text-red-700 dark:text-red-400"
                      @click="removeDefaultSubscription(index)"
                    >
                      {{ t('common.delete') }}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Affiliate Settings -->
        <div class="card">
          <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
            <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.affiliate.title') }}
            </h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.affiliate.description') }}
            </p>
          </div>
          <div class="space-y-6 p-6">
            <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
              <div class="flex items-start justify-between gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.affiliate.enabled') }}
                  </label>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.affiliate.enabledHint') }}
                  </p>
                </div>
                <Toggle v-model="form.affiliate_enabled" />
              </div>

              <div class="flex items-start justify-between gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.affiliate.transferEnabled') }}
                  </label>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.affiliate.transferEnabledHint') }}
                  </p>
                </div>
                <Toggle v-model="form.affiliate_transfer_enabled" />
              </div>
            </div>

            <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.affiliate.rate') }}
                </label>
                <input
                  v-model.number="form.affiliate_rebate_rate"
                  type="number"
                  step="0.01"
                  min="0"
                  max="100"
                  class="input"
                />
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.affiliate.freezeHours') }}
                </label>
                <input
                  v-model.number="form.affiliate_rebate_freeze_hours"
                  type="number"
                  step="1"
                  min="0"
                  max="720"
                  class="input"
                />
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.affiliate.durationDays') }}
                </label>
                <input
                  v-model.number="form.affiliate_rebate_duration_days"
                  type="number"
                  step="1"
                  min="0"
                  max="3650"
                  class="input"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.affiliate.durationDaysHint') }}
                </p>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.affiliate.perInviteeCap') }}
                </label>
                <input
                  v-model.number="form.affiliate_rebate_per_invitee_cap"
                  type="number"
                  step="0.01"
                  min="0"
                  class="input"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t('admin.settings.affiliate.perInviteeCapHint') }}
                </p>
              </div>
              <div>
                <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.settings.affiliate.affCodeLength') }}
                </label>
                <input
                  v-model.number="form.affiliate_aff_code_length"
                  type="number"
                  step="1"
                  min="6"
                  max="32"
                  class="input"
                />
              </div>
            </div>

            <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
              <div class="flex items-start justify-between gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.affiliate.onUsage') }}
                  </label>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.affiliate.onUsageHint') }}
                  </p>
                </div>
                <Toggle v-model="form.affiliate_rebate_on_usage_enabled" />
              </div>

              <div class="flex items-start justify-between gap-4">
                <div>
                  <label class="block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.settings.affiliate.onTopup') }}
                  </label>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t('admin.settings.affiliate.onTopupHint') }}
                  </p>
                </div>
                <Toggle v-model="form.affiliate_rebate_on_topup_enabled" />
              </div>
            </div>
          </div>
        </div>
        </div><!-- /Tab: Users -->
</template>

<script setup lang="ts">
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import GroupOptionItem from '@/components/common/GroupOptionItem.vue'
import type { AdminGroup } from '@/types'

interface DefaultSubscriptionGroupOption {
  value: number
  label: string
  description: string | null
  platform: AdminGroup['platform']
  subscriptionType: AdminGroup['subscription_type']
  rate: number
  [key: string]: unknown
}

const props = defineProps<{ ctx: any }>()
const {
  t,
  form,
  subscriptionGroups,
  defaultSubscriptionGroupOptions,
  addDefaultSubscription,
  removeDefaultSubscription,
} = props.ctx
</script>

<style scoped>
.default-sub-group-select :deep(.select-trigger) {
  @apply h-[42px];
}

.default-sub-delete-btn {
  @apply h-[42px];
}
</style>

