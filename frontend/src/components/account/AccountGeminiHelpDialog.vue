<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGeminiRateCatalog, type GeminiRateCatalog } from '@/api/admin/settings'
import BaseDialog from '@/components/common/BaseDialog.vue'

const props = defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()
const catalog = ref<GeminiRateCatalog | null>(null)
const catalogLoading = ref(false)

const geminiHelpLinks = {
  apiKey: 'https://aistudio.google.com/app/apikey',
  geminiWebActivation: 'https://gemini.google.com/gems/create?hl=en-US&pli=1',
  countryCheck: 'https://policies.google.com/terms',
  countryChange: 'https://policies.google.com/country-association-form'
}

const setupGuideLinks = [
  { href: geminiHelpLinks.countryCheck, labelKey: 'admin.accounts.gemini.setupGuide.links.countryCheck' },
  { href: geminiHelpLinks.countryChange, labelKey: 'admin.accounts.gemini.setupGuide.links.countryAssociationForm' },
  { href: geminiHelpLinks.geminiWebActivation, labelKey: 'admin.accounts.gemini.setupGuide.links.geminiWebActivation' },
  { href: 'https://console.cloud.google.com', labelKey: 'admin.accounts.gemini.setupGuide.links.gcpProject' }
]

const apiKeyLinks = [
  { href: geminiHelpLinks.apiKey, labelKey: 'admin.accounts.gemini.accountType.apiKeyLink' },
  { href: 'https://ai.google.dev/gemini-api/docs/rate-limits', labelKey: 'admin.accounts.gemini.accountType.quotaLink' }
]

const gemini3Links = [
  { href: 'https://ai.google.dev/gemini-api/docs/gemini-3?hl=zh-cn', labelKey: 'admin.accounts.gemini.gemini3Guide.links.gemini3' },
  { href: 'https://ai.google.dev/gemini-api/docs/media-resolution', labelKey: 'admin.accounts.gemini.gemini3Guide.links.mediaResolution' },
  { href: 'https://ai.google.dev/gemini-api/docs/tool-combination', labelKey: 'admin.accounts.gemini.gemini3Guide.links.toolCombination' },
  { href: 'https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/inference', labelKey: 'admin.accounts.gemini.gemini3Guide.links.vertexInference' }
]

const aiStudioRows = computed(() =>
  (catalog.value?.ai_studio_tiers || []).flatMap((tier) =>
    tier.model_families.map((model) => ({
      tierName: tier.display_name,
      qualification: tier.qualification,
      billingCap: tier.billing_tier_cap,
      modelName: model.display_name,
      rpm: model.rpm,
      tpm: model.tpm,
      rpd: model.rpd,
      notes: model.notes || ''
    }))
  )
)

const batchRows = computed(() =>
  (catalog.value?.batch_limits.by_tier || []).flatMap((tier) =>
    tier.entries.map((entry) => ({
      tierName: tier.tier_id,
      modelName: entry.display_name,
      tokens: entry.enqueued_tokens
    }))
  )
)

function formatTierName(tierID: string) {
  switch (tierID) {
    case 'aistudio_tier_3':
      return 'Tier 3'
    case 'aistudio_tier_2':
      return 'Tier 2'
    case 'aistudio_tier_1':
      return 'Tier 1'
    default:
      return 'Free'
  }
}

function formatLimitValue(value: number) {
  if (value < 0) return 'Unlimited'
  if (value === 0) return '-'
  return value.toLocaleString()
}

function formatBytes(value: number) {
  if (value <= 0) return '-'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let current = value
  let index = 0
  while (current >= 1024 && index < units.length - 1) {
    current /= 1024
    index += 1
  }
  return `${current.toFixed(current >= 10 || index === 0 ? 0 : 1)} ${units[index]}`
}

async function loadCatalog() {
  if (catalogLoading.value) {
    return
  }
  catalogLoading.value = true
  try {
    catalog.value = await getGeminiRateCatalog()
  } finally {
    catalogLoading.value = false
  }
}

watch(
  () => props.show,
  (show) => {
    if (show && !catalog.value) {
      void loadCatalog()
    }
  },
  { immediate: true }
)
</script>

<template>
  <BaseDialog
    :show="show"
    :title="t('admin.accounts.gemini.helpDialog.title')"
    max-width="max-w-3xl"
    @close="emit('close')"
  >
    <div class="space-y-6">
      <div>
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.gemini.setupGuide.title') }}
        </h3>
        <div class="space-y-4">
          <div>
            <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.gemini.setupGuide.checklistTitle') }}
            </p>
            <ul class="list-inside list-disc space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <li>{{ t('admin.accounts.gemini.setupGuide.checklistItems.usIp') }}</li>
              <li>{{ t('admin.accounts.gemini.setupGuide.checklistItems.age') }}</li>
            </ul>
          </div>
          <div>
            <p class="mb-2 text-sm font-medium text-gray-700 dark:text-gray-300">
              {{ t('admin.accounts.gemini.setupGuide.activationTitle') }}
            </p>
            <ul class="list-inside list-disc space-y-1 text-sm text-gray-600 dark:text-gray-400">
              <li>{{ t('admin.accounts.gemini.setupGuide.activationItems.geminiWeb') }}</li>
              <li>{{ t('admin.accounts.gemini.setupGuide.activationItems.gcpProject') }}</li>
            </ul>
            <div class="mt-2 flex flex-wrap gap-2">
              <a
                v-for="link in setupGuideLinks"
                :key="link.href"
                :href="link.href"
                target="_blank"
                rel="noreferrer"
                class="text-sm text-blue-600 hover:underline dark:text-blue-400"
              >
                {{ t(link.labelKey) }}
              </a>
            </div>
          </div>
        </div>
      </div>

      <div class="border-t border-gray-200 pt-6 dark:border-dark-600">
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.gemini.gemini3Guide.title') }}
        </h3>
        <ul class="list-inside list-disc space-y-1 text-sm text-gray-600 dark:text-gray-400">
          <li>{{ t('admin.accounts.gemini.gemini3Guide.items.stableDefault') }}</li>
          <li>{{ t('admin.accounts.gemini.gemini3Guide.items.thinkingLevel') }}</li>
          <li>{{ t('admin.accounts.gemini.gemini3Guide.items.thinkingBudget') }}</li>
          <li>{{ t('admin.accounts.gemini.gemini3Guide.items.mediaResolution') }}</li>
          <li>{{ t('admin.accounts.gemini.gemini3Guide.items.urlContext') }}</li>
          <li>{{ t('admin.accounts.gemini.gemini3Guide.items.toolCombination') }}</li>
        </ul>
        <div class="mt-3 flex flex-wrap gap-3">
          <a
            v-for="link in gemini3Links"
            :key="link.href"
            :href="link.href"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t(link.labelKey) }}
          </a>
        </div>
      </div>

      <div class="border-t border-gray-200 pt-6 dark:border-dark-600">
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.gemini.quotaPolicy.title') }}
        </h3>
        <p class="mb-2 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.gemini.quotaPolicy.effectiveDate', { date: catalog?.effective_date || '-' }) }}
        </p>
        <p class="mb-4 text-xs text-amber-600 dark:text-amber-400">
          {{
            catalog?.remaining_quota_api_supported
              ? t('admin.accounts.gemini.quotaPolicy.remainingApiAvailable')
              : t('admin.accounts.gemini.quotaPolicy.remainingApiUnavailable')
          }}
        </p>

        <div v-if="catalogLoading" class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('common.loading') }}
        </div>

        <template v-else-if="catalog">
          <div class="overflow-x-auto">
            <table class="w-full text-xs">
              <thead class="bg-gray-50 dark:bg-dark-600">
                <tr>
                  <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.accounts.gemini.quotaPolicy.columns.tier') }}
                  </th>
                  <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.accounts.gemini.quotaPolicy.columns.model') }}
                  </th>
                  <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.accounts.gemini.quotaPolicy.columns.limits') }}
                  </th>
                  <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                    {{ t('admin.accounts.gemini.quotaPolicy.columns.qualification') }}
                  </th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-200 dark:divide-dark-600">
                <tr v-for="row in aiStudioRows" :key="`${row.tierName}-${row.modelName}`">
                  <td class="px-3 py-2 text-gray-900 dark:text-white">
                    <div class="font-medium">{{ row.tierName }}</div>
                    <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ row.billingCap }}</div>
                  </td>
                  <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ row.modelName }}</td>
                  <td class="px-3 py-2 text-gray-600 dark:text-gray-400">
                    RPM {{ formatLimitValue(row.rpm) }} / TPM {{ formatLimitValue(row.tpm) }} / RPD {{ formatLimitValue(row.rpd) }}
                    <div v-if="row.notes" class="mt-1 text-[11px] text-gray-500 dark:text-gray-400">
                      {{ row.notes }}
                    </div>
                  </td>
                  <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ row.qualification }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <div class="mt-5">
            <h4 class="mb-2 text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.accounts.gemini.quotaPolicy.batchSection') }}
            </h4>
            <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">
              {{
                t('admin.accounts.gemini.quotaPolicy.batchSummary', {
                  concurrent: catalog.batch_limits.concurrent_batch_requests,
                  inputSize: formatBytes(catalog.batch_limits.input_file_size_limit_bytes),
                  storage: formatBytes(catalog.batch_limits.file_storage_limit_bytes)
                })
              }}
            </p>
            <div class="overflow-x-auto">
              <table class="w-full text-xs">
                <thead class="bg-gray-50 dark:bg-dark-600">
                  <tr>
                    <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                      {{ t('admin.accounts.gemini.quotaPolicy.columns.tier') }}
                    </th>
                    <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                      {{ t('admin.accounts.gemini.quotaPolicy.columns.model') }}
                    </th>
                    <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                      {{ t('admin.accounts.gemini.quotaPolicy.columns.batchTokens') }}
                    </th>
                  </tr>
                </thead>
                <tbody class="divide-y divide-gray-200 dark:divide-dark-600">
                  <tr v-for="row in batchRows" :key="`${row.tierName}-${row.modelName}`">
                    <td class="px-3 py-2 text-gray-900 dark:text-white">{{ formatTierName(row.tierName) }}</td>
                    <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ row.modelName }}</td>
                    <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ row.tokens.toLocaleString() }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <div v-if="catalog.notes.length > 0" class="mt-4 space-y-1 text-xs text-gray-500 dark:text-gray-400">
            <p v-for="note in catalog.notes" :key="note">{{ note }}</p>
          </div>

          <div class="mt-4 flex flex-wrap gap-3">
            <a
              v-for="link in catalog.links"
              :key="link.url"
              :href="link.url"
              target="_blank"
              rel="noreferrer"
              class="text-sm text-blue-600 hover:underline dark:text-blue-400"
            >
              {{ link.label }}
            </a>
          </div>
        </template>

        <div v-else class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.gemini.quotaPolicy.note') }}
        </div>
      </div>

      <div class="border-t border-gray-200 pt-6 dark:border-dark-600">
        <h3 class="mb-3 text-sm font-semibold text-gray-900 dark:text-white">
          {{ t('admin.accounts.gemini.helpDialog.apiKeySection') }}
        </h3>
        <div class="flex flex-wrap gap-3">
          <a
            v-for="link in apiKeyLinks"
            :key="link.href"
            :href="link.href"
            target="_blank"
            rel="noreferrer"
            class="text-sm text-blue-600 hover:underline dark:text-blue-400"
          >
            {{ t(link.labelKey) }}
          </a>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button type="button" class="btn btn-primary" @click="emit('close')">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>
