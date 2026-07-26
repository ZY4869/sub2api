<template>
  <div
    v-if="supportsRuntimePolicy"
    class="border-t border-gray-200 pt-4 dark:border-dark-400"
  >
    <div class="mb-4 flex items-center justify-between gap-4">
      <div>
        <label class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.groups.openaiRuntimePolicy.title') }}
        </label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.groups.openaiRuntimePolicy.allowLiveHint') }}
        </p>
      </div>
      <Toggle v-model="form.allow_live" />
    </div>

    <div class="space-y-4 border-l-2 border-primary-200 pl-4 dark:border-primary-800">
      <div>
        <label class="input-label">
          {{ t('admin.groups.openaiRuntimePolicy.maxReasoningEffort') }}
        </label>
        <Select
          v-model="form.max_reasoning_effort"
          :options="reasoningEffortOptions"
        />
        <p class="input-hint">
          {{ t('admin.groups.openaiRuntimePolicy.maxReasoningEffortHint') }}
        </p>
      </div>

      <div>
        <div class="mb-2 flex items-center justify-between gap-3">
          <div>
            <label class="input-label">
              {{ t('admin.groups.openaiRuntimePolicy.mappingTitle') }}
            </label>
            <p class="input-hint">
              {{ t('admin.groups.openaiRuntimePolicy.mappingHint') }}
            </p>
          </div>
          <button
            type="button"
            class="flex shrink-0 items-center gap-1.5 text-sm text-primary-600 hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
            @click="$emit('addMapping')"
          >
            <Icon name="plus" size="sm" />
            {{ t('admin.groups.openaiRuntimePolicy.addMapping') }}
          </button>
        </div>

        <div class="space-y-3">
          <div
            v-for="mapping in form.reasoning_effort_mappings"
            :key="getMappingKey(mapping)"
            class="rounded-lg border border-gray-200 p-3 dark:border-dark-600"
          >
            <div class="grid grid-cols-1 gap-3 md:grid-cols-[minmax(0,1.5fr)_minmax(0,1fr)_minmax(0,1fr)_auto] md:items-end">
              <div>
                <label class="input-label text-xs">
                  {{ t('admin.groups.openaiRuntimePolicy.modelPattern') }}
                </label>
                <input
                  v-model="mapping.model"
                  type="text"
                  class="input text-sm"
                  placeholder="gpt-*"
                />
              </div>
              <div>
                <label class="input-label text-xs">
                  {{ t('admin.groups.openaiRuntimePolicy.from') }}
                </label>
                <Select
                  v-model="mapping.from"
                  :options="mappingFromOptions"
                />
              </div>
              <div>
                <label class="input-label text-xs">
                  {{ t('admin.groups.openaiRuntimePolicy.to') }}
                </label>
                <Select
                  v-model="mapping.to"
                  :options="reasoningEffortRequiredOptions"
                />
              </div>
              <button
                type="button"
                class="justify-self-start rounded-lg p-2 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20"
                :title="t('admin.groups.openaiRuntimePolicy.removeMapping')"
                @click="$emit('removeMapping', mapping)"
              >
                <Icon name="trash" size="sm" />
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Select from '@/components/common/Select.vue'
import Toggle from '@/components/common/Toggle.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  form: Record<string, any>
  t: (key: string, params?: Record<string, unknown>) => string
  getMappingKey: (mapping: Record<string, any>) => string
}>()

defineEmits<{
  (e: 'addMapping'): void
  (e: 'removeMapping', mapping: Record<string, any>): void
}>()

const form = props.form
const t = props.t
const supportsRuntimePolicy = computed(() =>
  form.platform === 'openai' || form.platform === 'composite'
)

const effortValues = ['none', 'low', 'medium', 'high', 'xhigh', 'max']
const reasoningEffortOptions = computed(() => [
  { value: '', label: t('admin.groups.openaiRuntimePolicy.noLimit') },
  ...effortValues.map((value) => ({ value, label: value }))
])
const reasoningEffortRequiredOptions = computed(() =>
  effortValues.map((value) => ({ value, label: value }))
)
const mappingFromOptions = computed(() => [
  { value: '', label: t('admin.groups.openaiRuntimePolicy.anyEffort') },
  ...effortValues.map((value) => ({ value, label: value }))
])
</script>
