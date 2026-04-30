<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ t('admin.settings.openaiFastPolicy.title') }}
      </h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.settings.openaiFastPolicy.description') }}
      </p>
    </div>

    <div class="space-y-5 p-6">
      <div class="flex items-center justify-between gap-4">
        <div>
          <label class="font-medium text-gray-900 dark:text-white">
            {{ t('admin.settings.openaiFastPolicy.anthropicTTL1hInjection') }}
          </label>
          <p class="text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.openaiFastPolicy.anthropicTTL1hInjectionHint') }}
          </p>
        </div>
        <Toggle v-model="enableInjection" />
      </div>

      <div class="space-y-4 border-t border-gray-100 pt-5 dark:border-dark-700">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
              {{ t('admin.settings.openaiFastPolicy.rulesTitle') }}
            </h3>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.openaiFastPolicy.rulesHint') }}
            </p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" @click="addRule">
            {{ t('admin.settings.openaiFastPolicy.addRule') }}
          </button>
        </div>

        <div
          v-for="(rule, idx) in policy.rules"
          :key="idx"
          class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="flex flex-wrap items-center gap-2">
                <span class="text-sm font-medium text-gray-900 dark:text-white">
                  {{ t('admin.settings.openaiFastPolicy.rule') }} #{{ idx + 1 }}
                </span>
                <span class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-600 dark:bg-dark-700 dark:text-gray-300">
                  {{ displayTier(rule.service_tier) }}
                </span>
              </div>
            </div>
            <button type="button" class="btn btn-secondary btn-sm" @click="removeRule(idx)">
              {{ t('common.remove') }}
            </button>
          </div>

          <div class="mt-3 grid grid-cols-1 gap-4 md:grid-cols-2">
            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.settings.openaiFastPolicy.serviceTier') }}
              </label>
              <Select
                :modelValue="rule.service_tier"
                @update:modelValue="rule.service_tier = String($event || '')"
                :options="serviceTierOptions"
              />
            </div>

            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.settings.openaiFastPolicy.scope') }}
              </label>
              <Select
                :modelValue="rule.scope"
                @update:modelValue="rule.scope = String($event || '')"
                :options="scopeOptions"
              />
            </div>

            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.settings.openaiFastPolicy.action') }}
              </label>
              <Select
                :modelValue="rule.action"
                @update:modelValue="rule.action = String($event || '')"
                :options="actionOptions"
              />
            </div>

            <div>
              <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
                {{ t('admin.settings.openaiFastPolicy.fallbackAction') }}
              </label>
              <Select
                :modelValue="rule.fallback_action || ''"
                @update:modelValue="rule.fallback_action = String($event || '')"
                :options="actionOptions"
              />
            </div>
          </div>

          <div class="mt-3">
            <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
              {{ t('admin.settings.openaiFastPolicy.modelWhitelist') }}
            </label>
            <textarea
              class="input min-h-[96px] font-mono text-xs"
              :placeholder="t('admin.settings.openaiFastPolicy.modelWhitelistPlaceholder')"
              :value="formatWhitelist(rule.model_whitelist)"
              @input="handleWhitelistInput(idx, $event)"
            ></textarea>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.settings.openaiFastPolicy.modelWhitelistHint') }}
            </p>
          </div>
        </div>

        <p class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.settings.openaiFastPolicy.defaultBehaviorHint') }}
        </p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Toggle from '@/components/common/Toggle.vue'
import Select from '@/components/common/Select.vue'
import type { OpenAIFastPolicySettings, OpenAIFastPolicyRule } from '@/api/admin/settings'

const { t } = useI18n()

const policy = defineModel<OpenAIFastPolicySettings>({
  required: true
})

const enableInjection = defineModel<boolean>('enableInjection', {
  required: true
})

const serviceTierOptions = computed(() => [
  { value: 'priority', label: t('admin.settings.openaiFastPolicy.tierPriority') },
  { value: 'fast', label: t('admin.settings.openaiFastPolicy.tierFast') },
  { value: 'flex', label: t('admin.settings.openaiFastPolicy.tierFlex') }
])

const actionOptions = computed(() => [
  { value: 'pass', label: t('admin.settings.openaiFastPolicy.actionPass') },
  { value: 'filter', label: t('admin.settings.openaiFastPolicy.actionFilter') },
  { value: 'block', label: t('admin.settings.openaiFastPolicy.actionBlock') }
])

const scopeOptions = computed(() => [
  { value: 'all', label: t('admin.settings.openaiFastPolicy.scopeAll') },
  { value: 'oauth', label: t('admin.settings.openaiFastPolicy.scopeOAuth') },
  { value: 'apikey', label: t('admin.settings.openaiFastPolicy.scopeAPIKey') }
])

const defaultRule = (): OpenAIFastPolicyRule => ({
  service_tier: 'priority',
  action: 'filter',
  scope: 'all',
  model_whitelist: [],
  fallback_action: 'filter'
})

const addRule = () => {
  if (!policy.value) {
    return
  }
  if (!Array.isArray(policy.value.rules)) {
    policy.value.rules = []
  }
  policy.value.rules.push(defaultRule())
}

const removeRule = (idx: number) => {
  if (!policy.value || !Array.isArray(policy.value.rules)) {
    return
  }
  policy.value.rules.splice(idx, 1)
}

const displayTier = (tier: string) => {
  const normalized = String(tier || '').trim().toLowerCase()
  if (normalized === 'priority') return t('admin.settings.openaiFastPolicy.tierPriority')
  if (normalized === 'fast') return t('admin.settings.openaiFastPolicy.tierFast')
  if (normalized === 'flex') return t('admin.settings.openaiFastPolicy.tierFlex')
  return normalized || t('common.unknown')
}

const formatWhitelist = (whitelist?: string[]) => {
  if (!Array.isArray(whitelist) || whitelist.length === 0) {
    return ''
  }
  return whitelist.join('\n')
}

const parseWhitelist = (raw: string): string[] => {
  const tokens = raw
    .split(/[\n,]/g)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)

  const seen = new Set<string>()
  const out: string[] = []
  for (const token of tokens) {
    if (seen.has(token)) continue
    seen.add(token)
    out.push(token)
  }
  return out
}

const handleWhitelistInput = (idx: number, event: Event) => {
  if (!policy.value || !Array.isArray(policy.value.rules)) {
    return
  }
  const el = event.target as HTMLTextAreaElement | null
  if (!el) return
  const next = parseWhitelist(el.value || '')
  policy.value.rules[idx].model_whitelist = next
}
</script>

