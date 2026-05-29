<template>
  <BaseDialog
    :show="show"
    :title="showEditModal ? t('keys.editKey') : t('keys.createKey')"
    width="wide"
    @close="$emit('close')"
  >
    <form
      id="key-form"
      @submit.prevent="$emit('submit')"
      class="grid items-start gap-4 lg:grid-cols-2"
    >
      <div class="lg:col-span-2">
        <label class="input-label">{{ t("keys.nameLabel") }}</label>
        <input
          :value="formData.name"
          type="text"
          required
          class="input"
          :placeholder="t('keys.namePlaceholder')"
          data-tour="key-form-name"
          @input="(event) => updateField('name', (event.target as HTMLInputElement).value)"
        />
      </div>

      <div class="lg:col-span-2">
        <label class="input-label">
          {{ isAdminMode ? t("admin.users.groupBindings") : t("keys.groupLabel") }}
        </label>
        <APIKeyGroupBindingsEditor
          :model-value="formData.group_bindings"
          :groups="groups"
          :group-model-catalog-items="groupModelCatalogItems"
          :group-model-options="groupModelOptions"
          :group-model-options-loading="groupModelOptionsLoading"
          :admin-mode="isAdminMode"
          :image-only="formData.image_only_enabled"
          :model-selection-required="apiKeyModelSelectionRequired"
          @update:model-value="(value) => updateField('group_bindings', value)"
        />
      </div>

      <div v-if="!showEditModal" class="space-y-3">
        <ToggleField
          :label="t('keys.customKeyLabel')"
          :model-value="formData.use_custom_key"
          @update:model-value="(value) => updateField('use_custom_key', value)"
        />
        <div v-if="formData.use_custom_key">
          <input
            :value="formData.custom_key"
            type="text"
            class="input font-mono"
            :placeholder="t('keys.customKeyPlaceholder')"
            :class="{ 'border-red-500 dark:border-red-500': customKeyError }"
            @input="(event) => updateField('custom_key', (event.target as HTMLInputElement).value)"
          />
          <p v-if="customKeyError" class="mt-1 text-sm text-red-500">{{ customKeyError }}</p>
          <p v-else class="input-hint">{{ t("keys.customKeyHint") }}</p>
        </div>
      </div>

      <div v-if="showEditModal">
        <label class="input-label">{{ t("keys.statusLabel") }}</label>
        <Select
          :model-value="formData.status"
          :options="statusOptions"
          :placeholder="t('keys.selectStatus')"
          @update:model-value="(value) => updateField('status', normalizeStatus(value))"
        />
      </div>

      <section class="space-y-3 lg:col-span-2">
        <ToggleField
          :label="t('keys.ipRestriction')"
          :model-value="formData.enable_ip_restriction"
          @update:model-value="(value) => updateField('enable_ip_restriction', value)"
        />
        <div v-if="formData.enable_ip_restriction" class="grid gap-3 pt-2 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t("keys.ipWhitelist") }}</label>
            <textarea
              :value="formData.ip_whitelist"
              rows="3"
              class="input font-mono text-sm"
              :placeholder="t('keys.ipWhitelistPlaceholder')"
              @input="(event) => updateField('ip_whitelist', (event.target as HTMLTextAreaElement).value)"
            />
            <p class="input-hint">{{ t("keys.ipWhitelistHint") }}</p>
          </div>
          <div>
            <label class="input-label">{{ t("keys.ipBlacklist") }}</label>
            <textarea
              :value="formData.ip_blacklist"
              rows="3"
              class="input font-mono text-sm"
              :placeholder="t('keys.ipBlacklistPlaceholder')"
              @input="(event) => updateField('ip_blacklist', (event.target as HTMLTextAreaElement).value)"
            />
            <p class="input-hint">{{ t("keys.ipBlacklistHint") }}</p>
          </div>
        </div>
      </section>

      <section class="space-y-3 lg:col-span-2">
        <label class="input-label">{{ t("keys.quotaLimit") }}</label>
        <div class="space-y-4">
          <div>
            <div class="relative">
              <span class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-500">$</span>
              <input
                :value="formData.quota"
                type="number"
                step="0.01"
                min="0"
                class="input pl-7"
                :placeholder="t('keys.quotaAmountPlaceholder')"
                @input="(event) => updateField('quota', parseNullableNumber((event.target as HTMLInputElement).value))"
              />
            </div>
            <p class="input-hint">{{ t("keys.quotaAmountHint") }}</p>
          </div>

          <div v-if="showEditModal && selectedKey && selectedKey.quota > 0">
            <label class="input-label">{{ t("keys.quotaUsed") }}</label>
            <div class="flex items-center gap-2">
              <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700">
                <span class="font-medium text-gray-900 dark:text-white">
                  ${{ selectedKey.quota_used?.toFixed(4) || "0.0000" }}
                </span>
                <span class="mx-2 text-gray-400">/</span>
                <span class="text-gray-500 dark:text-gray-400">
                  ${{ selectedKey.quota?.toFixed(2) || "0.00" }}
                </span>
              </div>
              <button
                type="button"
                @click="$emit('confirm-reset-quota')"
                class="btn btn-secondary text-sm"
                :title="t('keys.resetQuotaUsed')"
              >
                {{ t("keys.reset") }}
              </button>
            </div>
          </div>
        </div>
      </section>

      <section class="space-y-3 lg:col-span-2">
        <ToggleField
          :label="t('keys.imageOnlyKey')"
          :hint="t('keys.imageOnlyKeyHint')"
          :model-value="formData.image_only_enabled"
          align-start
          @update:model-value="(value) => updateField('image_only_enabled', value)"
        />
        <div v-if="formData.image_only_enabled" class="space-y-4 pt-2">
          <ToggleField
            :label="t('keys.imageCountBilling')"
            :hint="t('keys.imageCountBillingHint')"
            :model-value="formData.image_count_billing_enabled"
            align-start
            @update:model-value="(value) => updateField('image_count_billing_enabled', value)"
          />
          <div v-if="formData.image_count_billing_enabled" class="grid gap-3 md:grid-cols-2">
            <div>
              <label class="input-label">{{ t("keys.imageMaxCount") }}</label>
              <input
                :value="formData.image_max_count"
                type="number"
                step="1"
                min="1"
                class="input"
                :placeholder="t('keys.imageMaxCountPlaceholder')"
                @input="(event) => updateField('image_max_count', parseNullableNumber((event.target as HTMLInputElement).value))"
              />
              <p class="input-hint">{{ t("keys.imageMaxCountHint") }}</p>
            </div>

            <div
              v-if="
                showEditModal &&
                selectedKey &&
                selectedKey.image_only_enabled &&
                selectedKey.image_count_billing_enabled &&
                selectedKey.image_max_count > 0
              "
            >
              <label class="input-label">{{ t("keys.imageCountUsage") }}</label>
              <div class="flex-1 rounded-lg bg-gray-100 px-3 py-2 dark:bg-dark-700">
                <span class="font-medium text-gray-900 dark:text-white">
                  {{ selectedKey.image_count_used || 0 }}
                </span>
                <span class="mx-2 text-gray-400">/</span>
                <span class="text-gray-500 dark:text-gray-400">{{ selectedKey.image_max_count }}</span>
                <span class="ml-3 text-gray-500 dark:text-gray-400">
                  {{ t("keys.imageCountRemaining") }}:
                  {{ Math.max((selectedKey.image_max_count || 0) - (selectedKey.image_count_used || 0), 0) }}
                </span>
              </div>
            </div>

            <div class="md:col-span-2">
              <label class="input-label">{{ t("keys.imageCountWeights") }}</label>
              <div class="grid gap-3 sm:grid-cols-3">
                <label v-for="tier in imageCountWeightTiers" :key="tier" class="space-y-1">
                  <span class="text-xs font-medium text-gray-500 dark:text-gray-400">
                    {{ t(`keys.imageCountWeight${tier}`) }}
                  </span>
                  <input
                    :value="formData.image_count_weights[tier]"
                    type="number"
                    step="1"
                    min="1"
                    class="input"
                    @input="(event) => updateImageCountWeight(tier, (event.target as HTMLInputElement).value)"
                  />
                </label>
              </div>
              <p class="input-hint">{{ t("keys.imageCountWeightsHint") }}</p>
            </div>
          </div>
        </div>
      </section>

      <section class="space-y-3 lg:col-span-2">
        <ToggleField
          :label="t('keys.rateLimitSection')"
          :model-value="formData.enable_rate_limit"
          @update:model-value="(value) => updateField('enable_rate_limit', value)"
        />
        <div v-if="formData.enable_rate_limit" class="space-y-3 pt-2">
          <p class="input-hint -mt-2">{{ t("keys.rateLimitHint") }}</p>
          <div class="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
            <RateLimitEditField
              :model-value="formData.rate_limit_5h"
              :label="t('keys.rateLimit5h')"
              :selected-key="selectedKey"
              usage-key="usage_5h"
              limit-key="rate_limit_5h"
              :show-edit-modal="showEditModal"
              @update:model-value="(value) => updateField('rate_limit_5h', value)"
            />
            <RateLimitEditField
              :model-value="formData.rate_limit_1d"
              :label="t('keys.rateLimit1d')"
              :selected-key="selectedKey"
              usage-key="usage_1d"
              limit-key="rate_limit_1d"
              :show-edit-modal="showEditModal"
              @update:model-value="(value) => updateField('rate_limit_1d', value)"
            />
            <RateLimitEditField
              :model-value="formData.rate_limit_7d"
              :label="t('keys.rateLimit7d')"
              :selected-key="selectedKey"
              usage-key="usage_7d"
              limit-key="rate_limit_7d"
              :show-edit-modal="showEditModal"
              @update:model-value="(value) => updateField('rate_limit_7d', value)"
            />
          </div>

          <div
            v-if="
              showEditModal &&
              selectedKey &&
              (selectedKey.rate_limit_5h > 0 ||
                selectedKey.rate_limit_1d > 0 ||
                selectedKey.rate_limit_7d > 0)
            "
          >
            <button type="button" @click="$emit('confirm-reset-rate-limit')" class="btn btn-secondary text-sm">
              {{ t("keys.resetRateLimitUsage") }}
            </button>
          </div>
        </div>
      </section>

      <section class="space-y-3">
        <ToggleField
          :label="t('keys.expiration')"
          :model-value="formData.enable_expiration"
          @update:model-value="(value) => updateField('enable_expiration', value)"
        />
        <div v-if="formData.enable_expiration" class="space-y-4 pt-2">
          <div class="flex flex-wrap gap-2">
            <button
              v-for="days in ['7', '30', '90']"
              :key="days"
              type="button"
              @click="$emit('set-expiration-days', parseInt(days))"
              :class="[
                'rounded-lg px-3 py-1.5 text-sm transition-colors',
                formData.expiration_preset === days
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600',
              ]"
            >
              {{ showEditModal ? t("keys.extendDays", { days }) : t("keys.expiresInDays", { days }) }}
            </button>
            <button
              type="button"
              @click="updateField('expiration_preset', 'custom')"
              :class="[
                'rounded-lg px-3 py-1.5 text-sm transition-colors',
                formData.expiration_preset === 'custom'
                  ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-400'
                  : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-400 dark:hover:bg-dark-600',
              ]"
            >
              {{ t("keys.customDate") }}
            </button>
          </div>

          <div>
            <label class="input-label">{{ t("keys.expirationDate") }}</label>
            <input
              :value="formData.expiration_date"
              type="datetime-local"
              class="input"
              @input="(event) => updateField('expiration_date', (event.target as HTMLInputElement).value)"
            />
            <p class="input-hint">{{ t("keys.expirationDateHint") }}</p>
          </div>

          <div v-if="showEditModal && selectedKey?.expires_at" class="text-sm">
            <span class="text-gray-500 dark:text-gray-400">{{ t("keys.currentExpiration") }}: </span>
            <span class="font-medium text-gray-900 dark:text-white">
              {{ formatDateTime(selectedKey.expires_at) }}
            </span>
          </div>
        </div>
      </section>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" type="button" class="btn btn-secondary">
          {{ t("common.cancel") }}
        </button>
        <button
          form="key-form"
          type="submit"
          :disabled="submitting"
          class="btn btn-primary"
          data-tour="key-form-submit"
        >
          <svg
            v-if="submitting"
            class="-ml-1 mr-2 h-4 w-4 animate-spin"
            fill="none"
            viewBox="0 0 24 24"
          >
            <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
            <path
              class="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
            />
          </svg>
          {{ submitting ? t("keys.saving") : showEditModal ? t("common.update") : t("common.create") }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import BaseDialog from "@/components/common/BaseDialog.vue";
import Select from "@/components/common/Select.vue";
import APIKeyGroupBindingsEditor from "@/components/keys/APIKeyGroupBindingsEditor.vue";
import type { PublicModelCatalogItem } from "@/api/meta";
import type { ApiKey, Group, UserGroupModelOption } from "@/types";
import { formatDateTime } from "@/utils/format";
import type { ApiKeyFormData, ImageCountWeightTier } from "./types";
import { imageCountWeightTiers } from "./types";
import RateLimitEditField from "./RateLimitEditField.vue";
import ToggleField from "./ToggleField.vue";

const props = defineProps<{
  show: boolean;
  showEditModal: boolean;
  submitting: boolean;
  formData: ApiKeyFormData;
  selectedKey: ApiKey | null;
  groups: Group[];
  groupModelCatalogItems: Record<number, PublicModelCatalogItem[]>;
  groupModelOptions: Record<number, UserGroupModelOption[]>;
  groupModelOptionsLoading: boolean;
  isAdminMode: boolean;
  apiKeyModelSelectionRequired: boolean;
  customKeyError: string;
  statusOptions: Array<{ value: string; label: string }>;
}>();

const emit = defineEmits<{
  close: [];
  submit: [];
  "update:formData": [value: ApiKeyFormData];
  "confirm-reset-quota": [];
  "confirm-reset-rate-limit": [];
  "set-expiration-days": [days: number];
}>();

const { t } = useI18n();

function updateField<K extends keyof ApiKeyFormData>(
  key: K,
  value: ApiKeyFormData[K],
) {
  emit("update:formData", { ...props.formData, [key]: value });
}

function parseNullableNumber(value: string): number | null {
  if (value === "") return null;
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : null;
}

function normalizeStatus(value: unknown): ApiKeyFormData["status"] {
  return value === "inactive" ? "inactive" : "active";
}

function updateImageCountWeight(tier: ImageCountWeightTier, value: string) {
  const parsed = Number(value);
  const nextValue = Number.isFinite(parsed)
    ? parsed
    : props.formData.image_count_weights[tier];
  updateField("image_count_weights", {
    ...props.formData.image_count_weights,
    [tier]: nextValue,
  });
}
</script>
