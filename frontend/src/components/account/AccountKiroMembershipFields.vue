<template>
  <section class="space-y-4 rounded-lg border border-amber-200 bg-white/80 p-4 dark:border-amber-900/40 dark:bg-slate-900/60">
    <div class="space-y-1">
      <div class="text-sm font-semibold text-amber-950 dark:text-amber-100">
        {{ t('admin.accounts.kiroMembership.title') }}
      </div>
      <p class="text-xs text-amber-800 dark:text-amber-300">
        {{ t('admin.accounts.kiroMembership.description') }}
      </p>
    </div>

    <div class="grid gap-4 md:grid-cols-2">
      <label class="space-y-1">
        <span class="input-label">{{ t('admin.accounts.kiroMembership.level') }}</span>
        <select v-model="memberLevel" class="input">
          <option v-for="option in levelOptions" :key="option.value" :value="option.value">
            {{ option.label }}
          </option>
        </select>
      </label>

      <label class="space-y-1">
        <span class="input-label">{{ t('admin.accounts.kiroMembership.credits') }}</span>
        <input
          v-model="memberCredits"
          type="number"
          min="0"
          step="1"
          class="input"
          :placeholder="String(defaultCredits)"
        />
        <p
          v-if="showInvalidCredits"
          class="text-xs text-red-600 dark:text-red-400"
        >
          {{ t('admin.accounts.kiroMembership.invalidCredits') }}
        </p>
        <p v-else class="input-hint">
          {{ t('admin.accounts.kiroMembership.creditsHint') }}
        </p>
      </label>
    </div>

    <div class="rounded-md bg-amber-100/80 px-3 py-3 text-sm text-amber-950 dark:bg-amber-950/30 dark:text-amber-100">
      <div class="text-[11px] font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">
        {{ t('admin.accounts.kiroMembership.previewTitle') }}
      </div>
      <div class="mt-2 grid gap-2 md:grid-cols-2">
        <p>
          <span class="font-medium">{{ t('admin.accounts.kiroMembership.previewLevel') }}:</span>
          {{ currentLevelLabel }}
        </p>
        <p>
          <span class="font-medium">{{ t('admin.accounts.kiroMembership.previewCredits') }}:</span>
          {{ parsedCredits ?? '-' }}
        </p>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { KiroMemberLevel } from '@/utils/kiroMembership'
import {
  defaultKiroMemberCredits,
  parseKiroMemberCredits
} from '@/utils/kiroMembership'

const memberLevel = defineModel<KiroMemberLevel>('memberLevel', { required: true })
const memberCredits = defineModel<string>('memberCredits', { required: true })

const { t } = useI18n()

const levelOptions = computed(() => [
  { value: 'kiro_free' as const, label: t('admin.accounts.kiroMembership.levels.kiro_free') },
  { value: 'kiro_pro' as const, label: t('admin.accounts.kiroMembership.levels.kiro_pro') },
  { value: 'kiro_pro_plus' as const, label: t('admin.accounts.kiroMembership.levels.kiro_pro_plus') },
  { value: 'kiro_power' as const, label: t('admin.accounts.kiroMembership.levels.kiro_power') }
])

const parsedCredits = computed(() => parseKiroMemberCredits(memberCredits.value))
const showInvalidCredits = computed(() => memberCredits.value.trim() !== '' && parsedCredits.value === null)
const defaultCredits = computed(() => defaultKiroMemberCredits(memberLevel.value))
const currentLevelLabel = computed(() =>
  levelOptions.value.find((option) => option.value === memberLevel.value)?.label || memberLevel.value
)

watch(
  memberLevel,
  (nextLevel) => {
    memberCredits.value = String(defaultKiroMemberCredits(nextLevel))
  }
)
</script>
