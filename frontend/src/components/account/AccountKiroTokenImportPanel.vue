<template>
  <div class="space-y-4 rounded-xl border border-amber-200 bg-amber-50/80 p-5 dark:border-amber-900/40 dark:bg-amber-950/20">
    <div class="space-y-1">
      <h3 class="text-base font-semibold text-amber-950 dark:text-amber-100">
        {{ t('admin.accounts.kiroImport.title') }}
      </h3>
      <p class="text-sm text-amber-800 dark:text-amber-300">
        {{ t('admin.accounts.kiroImport.description') }}
      </p>
    </div>

    <textarea
      v-model="rawInput"
      rows="10"
      class="input w-full resize-y font-mono text-sm"
      :placeholder="t('admin.accounts.kiroImport.placeholder')"
    ></textarea>

    <p class="text-xs text-amber-700 dark:text-amber-400">
      {{ t('admin.accounts.kiroImport.hint') }}
    </p>

    <AccountKiroMembershipFields
      v-model:member-level="memberLevel"
      v-model:member-credits="memberCredits"
    />

    <div v-if="parseError" class="rounded-lg border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-900/50 dark:bg-red-950/20 dark:text-red-300">
      {{ parseError }}
    </div>

    <div v-if="preview" class="rounded-lg border border-amber-200 bg-white/80 p-4 text-sm text-slate-700 dark:border-amber-900/40 dark:bg-slate-900/60 dark:text-slate-200">
      <div class="grid gap-3 md:grid-cols-2">
        <div>
          <div class="text-xs font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">
            {{ t('admin.accounts.kiroImport.previewCredentials') }}
          </div>
          <div class="mt-2 space-y-1">
            <p><span class="font-medium">AT:</span> {{ preview.credentials.access_token ? 'yes' : 'no' }}</p>
            <p><span class="font-medium">RT:</span> {{ preview.credentials.refresh_token ? 'yes' : 'no' }}</p>
            <p><span class="font-medium">{{ t('admin.accounts.kiroImport.expiresAt') }}:</span> {{ preview.credentials.expires_at || '-' }}</p>
          </div>
        </div>
        <div>
          <div class="text-xs font-semibold uppercase tracking-wide text-amber-700 dark:text-amber-300">
            {{ t('admin.accounts.kiroImport.previewIdentity') }}
          </div>
          <div class="mt-2 space-y-1">
            <p><span class="font-medium">{{ t('admin.accounts.kiroImport.email') }}:</span> {{ preview.extra?.email || '-' }}</p>
            <p><span class="font-medium">{{ t('admin.accounts.kiroImport.username') }}:</span> {{ preview.extra?.username || '-' }}</p>
            <p><span class="font-medium">{{ t('admin.accounts.kiroImport.provider') }}:</span> {{ preview.extra?.provider || 'kiro' }}</p>
          </div>
        </div>
      </div>
    </div>

    <button type="button" class="btn btn-primary" :disabled="submitting" @click="handleSubmit">
      {{ submitting ? t('common.loading') : submitLabel }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AccountKiroMembershipFields from '@/components/account/AccountKiroMembershipFields.vue'
import {
  parseKiroTokenImport,
  type ParsedKiroTokenImport
} from '@/utils/kiroTokenImport'
import type { KiroMemberLevel } from '@/utils/kiroMembership'
import {
  buildKiroMembershipExtra,
  parseKiroMemberCredits,
  readKiroMembershipFromExtra
} from '@/utils/kiroMembership'

interface Props {
  submitLabel: string
  submitting?: boolean
  initialExtra?: Record<string, unknown> | null
}

const props = withDefaults(defineProps<Props>(), {
  submitting: false,
  initialExtra: null
})

const emit = defineEmits<{
  submit: [payload: ParsedKiroTokenImport]
}>()

const { t } = useI18n()

const rawInput = ref('')
const parseError = ref('')
const preview = ref<ParsedKiroTokenImport | null>(null)
const memberLevel = ref<KiroMemberLevel>('kiro_free')
const memberCredits = ref('50')

function resetMembership() {
  const membership = readKiroMembershipFromExtra(props.initialExtra)
  memberLevel.value = membership.level
  memberCredits.value = String(membership.credits)
}

watch(
  () => props.initialExtra,
  () => {
    resetMembership()
  },
  { immediate: true }
)

function handleSubmit() {
  try {
    const parsed = parseKiroTokenImport(rawInput.value)
    const credits = parseKiroMemberCredits(memberCredits.value)
    if (credits === null) {
      throw new Error(t('admin.accounts.kiroMembership.invalidCredits'))
    }
    const nextPayload: ParsedKiroTokenImport = {
      ...parsed,
      extra: buildKiroMembershipExtra(memberLevel.value, credits, parsed.extra)
    }
    preview.value = nextPayload
    parseError.value = ''
    emit('submit', nextPayload)
  } catch (error: any) {
    preview.value = null
    parseError.value = error?.message || t('admin.accounts.kiroImport.parseFailed')
  }
}

function reset() {
  rawInput.value = ''
  parseError.value = ''
  preview.value = null
  resetMembership()
}

defineExpose({ reset })
</script>
