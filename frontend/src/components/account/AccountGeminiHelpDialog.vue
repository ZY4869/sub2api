<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'

defineProps<{
  show: boolean
}>()

const emit = defineEmits<{
  close: []
}>()

const { t } = useI18n()

const geminiQuotaDocs = {
  codeAssist: 'https://developers.google.com/gemini-code-assist/resources/quotas',
  aiStudio: 'https://ai.google.dev/pricing',
  vertex: 'https://cloud.google.com/vertex-ai/generative-ai/docs/quotas'
}

const geminiHelpLinks = {
  apiKey: 'https://aistudio.google.com/app/apikey',
  aiStudioPricing: 'https://ai.google.dev/pricing',
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

const quotaRows = [
  {
    channelKey: 'admin.accounts.gemini.quotaPolicy.rows.googleOne.channel',
    accountLabel: 'Free',
    limitsKey: 'admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsFree'
  },
  {
    accountLabel: 'Pro',
    limitsKey: 'admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsPro'
  },
  {
    accountLabel: 'Ultra',
    limitsKey: 'admin.accounts.gemini.quotaPolicy.rows.googleOne.limitsUltra'
  },
  {
    channelKey: 'admin.accounts.gemini.quotaPolicy.rows.gcp.channel',
    accountLabel: 'Standard',
    limitsKey: 'admin.accounts.gemini.quotaPolicy.rows.gcp.limitsStandard'
  },
  {
    accountLabel: 'Enterprise',
    limitsKey: 'admin.accounts.gemini.quotaPolicy.rows.gcp.limitsEnterprise'
  },
  {
    channelKey: 'admin.accounts.gemini.quotaPolicy.rows.aiStudio.channel',
    accountLabel: 'Free',
    limitsKey: 'admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsFree'
  },
  {
    accountLabel: 'Paid',
    limitsKey: 'admin.accounts.gemini.quotaPolicy.rows.aiStudio.limitsPaid'
  }
]

const quotaDocLinks = [
  { href: geminiQuotaDocs.codeAssist, labelKey: 'admin.accounts.gemini.quotaPolicy.docs.codeAssist' },
  { href: geminiQuotaDocs.aiStudio, labelKey: 'admin.accounts.gemini.quotaPolicy.docs.aiStudio' },
  { href: geminiQuotaDocs.vertex, labelKey: 'admin.accounts.gemini.quotaPolicy.docs.vertex' }
]

const apiKeyLinks = [
  { href: geminiHelpLinks.apiKey, labelKey: 'admin.accounts.gemini.accountType.apiKeyLink' },
  { href: geminiHelpLinks.aiStudioPricing, labelKey: 'admin.accounts.gemini.accountType.quotaLink' }
]
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
          {{ t('admin.accounts.gemini.quotaPolicy.title') }}
        </h3>
        <p class="mb-4 text-xs text-amber-600 dark:text-amber-400">
          {{ t('admin.accounts.gemini.quotaPolicy.note') }}
        </p>
        <div class="overflow-x-auto">
          <table class="w-full text-xs">
            <thead class="bg-gray-50 dark:bg-dark-600">
              <tr>
                <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.accounts.gemini.quotaPolicy.columns.channel') }}
                </th>
                <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.accounts.gemini.quotaPolicy.columns.account') }}
                </th>
                <th class="px-3 py-2 text-left font-medium text-gray-700 dark:text-gray-300">
                  {{ t('admin.accounts.gemini.quotaPolicy.columns.limits') }}
                </th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-200 dark:divide-dark-600">
              <tr v-for="row in quotaRows" :key="`${row.channelKey || 'inherit'}-${row.accountLabel}`">
                <td class="px-3 py-2 text-gray-900 dark:text-white">
                  {{ row.channelKey ? t(row.channelKey) : '' }}
                </td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ row.accountLabel }}</td>
                <td class="px-3 py-2 text-gray-600 dark:text-gray-400">{{ t(row.limitsKey) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="mt-4 flex flex-wrap gap-3">
          <a
            v-for="link in quotaDocLinks"
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
