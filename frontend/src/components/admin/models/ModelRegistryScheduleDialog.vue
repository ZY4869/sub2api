<template>
  <BaseDialog
    :show="show"
    :title="t('admin.models.registry.scheduleDialog.title')"
    width="wide"
    close-on-click-outside
    @close="emit('close')"
  >
    <div v-if="draft && model" class="space-y-4">
      <div class="flex items-start gap-3 rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
        <ModelIcon :model="model.id" :provider="model.provider" :display-name="model.display_name" size="22px" />
        <div class="min-w-0">
          <div class="truncate text-sm font-semibold text-gray-900 dark:text-white">
            {{ model.display_name || model.id }}
          </div>
          <div class="truncate font-mono text-xs text-gray-500 dark:text-gray-400">
            {{ model.id }}
          </div>
        </div>
      </div>

      <div class="grid gap-3 md:grid-cols-2">
        <label class="space-y-1.5 text-sm font-medium text-gray-700 dark:text-gray-200">
          <span>{{ t('admin.models.registry.fields.availableFrom') }}</span>
          <input
            :value="formatDateTimeLocal(draft.available_from)"
            type="datetime-local"
            class="input"
            data-testid="registry-schedule-available-from"
            @input="draft.available_from = dateTimeLocalToISOString(($event.target as HTMLInputElement).value) || ''"
          />
        </label>
        <label class="space-y-1.5 text-sm font-medium text-gray-700 dark:text-gray-200">
          <span>{{ t('admin.models.registry.fields.availableUntil') }}</span>
          <input
            :value="formatDateTimeLocal(draft.available_until)"
            type="datetime-local"
            class="input"
            data-testid="registry-schedule-available-until"
            @input="draft.available_until = dateTimeLocalToISOString(($event.target as HTMLInputElement).value) || ''"
          />
        </label>
      </div>

      <label class="flex items-start gap-3 rounded-xl border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900/40">
        <input
          v-model="timeAccessEnabled"
          type="checkbox"
          class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          data-testid="registry-schedule-time-access-enabled"
        />
        <span>
          <span class="block text-sm font-medium text-gray-900 dark:text-white">
            {{ t('admin.models.registry.fields.accessTimePolicy') }}
          </span>
          <span class="mt-1 block text-xs text-gray-500 dark:text-gray-400">
            {{ t('admin.models.registry.scheduleDialog.hint') }}
          </span>
        </span>
      </label>

      <TimeAccessPolicyEditor
        v-if="timeAccessEnabled"
        v-model="draft.access_time_policy"
        :hint="t('admin.models.registry.scheduleDialog.hint')"
      />
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="emit('close')">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          data-testid="registry-schedule-save"
          :disabled="submitting || !model"
          @click="save"
        >
          {{ t('admin.models.registry.scheduleDialog.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelRegistryDetail } from '@/api/admin/modelRegistry'
import type { TimeAccessPolicy } from '@/types/api-key-groups'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import TimeAccessPolicyEditor from '@/components/common/TimeAccessPolicyEditor.vue'
import {
  buildPresetTimeAccessPolicy,
  dateTimeLocalToISOString,
  ensureEnabledTimeAccessPolicy,
  formatDateTimeLocal,
  normalizeTimeAccessPolicy,
  policyToPayload,
} from '@/utils/timeAccessPolicy'

type ScheduleDraft = {
  available_from: string
  available_until: string
  access_time_policy: TimeAccessPolicy
}

const props = defineProps<{
  show: boolean
  model: ModelRegistryDetail | null
  submitting?: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'submit', model: ModelRegistryDetail, patch: {
    available_from?: string
    available_until?: string
    access_time_policy?: TimeAccessPolicy | null
  }): void
}>()

const { t } = useI18n()
const draft = ref<ScheduleDraft | null>(null)
const timeAccessEnabled = ref(false)

watch(
  () => [props.show, props.model] as const,
  () => {
    if (!props.show || !props.model) {
      draft.value = null
      timeAccessEnabled.value = false
      return
    }
    draft.value = {
      available_from: props.model.available_from || '',
      available_until: props.model.available_until || '',
      access_time_policy: normalizeTimeAccessPolicy(props.model.access_time_policy || null),
    }
    timeAccessEnabled.value = !!props.model.access_time_policy?.enabled
  },
  { immediate: true },
)

function save() {
  if (!props.model || !draft.value) return
  emit('submit', props.model, {
    available_from: draft.value.available_from || '',
    available_until: draft.value.available_until || '',
    access_time_policy: timeAccessEnabled.value
      ? policyToPayload(ensureEnabledTimeAccessPolicy(draft.value.access_time_policy || buildPresetTimeAccessPolicy('daytime')))
      : null,
  })
}
</script>
