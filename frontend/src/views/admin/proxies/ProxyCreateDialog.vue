<template>
  <BaseDialog
    :show="show"
    :title="t('admin.proxies.createProxy')"
    width="normal"
    @close="emit('close')"
  >
    <div class="mb-6 flex border-b border-gray-200 dark:border-dark-600">
      <button
        type="button"
        @click="emit('update:createMode', 'standard')"
        :class="[
          '-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors',
          createMode === 'standard'
            ? 'border-primary-500 text-primary-600 dark:text-primary-400'
            : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
        ]"
      >
        <Icon name="plus" size="sm" class="mr-1.5 inline" />
        {{ t('admin.proxies.standardAdd') }}
      </button>
      <button
        type="button"
        @click="emit('update:createMode', 'batch')"
        :class="[
          '-mb-px border-b-2 px-4 py-2 text-sm font-medium transition-colors',
          createMode === 'batch'
            ? 'border-primary-500 text-primary-600 dark:text-primary-400'
            : 'border-transparent text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-300'
        ]"
      >
        <svg
          class="mr-1.5 inline h-4 w-4"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          stroke-width="1.5"
        >
          <path
            stroke-linecap="round"
            stroke-linejoin="round"
            d="M3.75 12h16.5m-16.5 3.75h16.5M3.75 19.5h16.5M5.625 4.5h12.75a1.875 1.875 0 010 3.75H5.625a1.875 1.875 0 010-3.75z"
          />
        </svg>
        {{ t('admin.proxies.batchAdd') }}
      </button>
    </div>

    <form
      v-if="createMode === 'standard'"
      id="create-proxy-form"
      @submit.prevent="emit('create')"
      class="space-y-5"
    >
      <div>
        <label class="input-label">{{ t('admin.proxies.name') }}</label>
        <input
          :value="createForm.name"
          type="text"
          required
          class="input"
          :placeholder="t('admin.proxies.enterProxyName')"
          @input="(event) => updateCreateFormField('name', (event.target as HTMLInputElement).value)"
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.protocol') }}</label>
        <Select
          :model-value="createForm.protocol"
          :options="protocolSelectOptions"
          @update:model-value="(value) => updateCreateFormField('protocol', value)"
        />
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('admin.proxies.host') }}</label>
          <input
            :value="createForm.host"
            type="text"
            required
            :placeholder="t('admin.proxies.form.hostPlaceholder')"
            class="input"
            @input="(event) => updateCreateFormField('host', (event.target as HTMLInputElement).value)"
          />
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.port') }}</label>
          <input
            :value="createForm.port"
            type="number"
            required
            min="1"
            max="65535"
            :placeholder="t('admin.proxies.form.portPlaceholder')"
            class="input"
            @input="(event) => updateCreateFormField('port', Number((event.target as HTMLInputElement).value))"
          />
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.username') }}</label>
        <input
          :value="createForm.username"
          type="text"
          class="input"
          :placeholder="t('admin.proxies.optionalAuth')"
          @input="(event) => updateCreateFormField('username', (event.target as HTMLInputElement).value)"
        />
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.password') }}</label>
        <div class="relative">
          <input
            :value="createForm.password"
            :type="createPasswordVisible ? 'text' : 'password'"
            class="input pr-10"
            :placeholder="t('admin.proxies.optionalAuth')"
            @input="(event) => updateCreateFormField('password', (event.target as HTMLInputElement).value)"
          />
          <button
            type="button"
            class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            @click="emit('update:createPasswordVisible', !createPasswordVisible)"
          >
            <Icon :name="createPasswordVisible ? 'eyeOff' : 'eye'" size="md" />
          </button>
        </div>
      </div>
    </form>

    <div v-else class="space-y-5">
      <div>
        <label class="input-label">{{ t('admin.proxies.batchInput') }}</label>
        <textarea
          v-model="batchInputModel"
          rows="10"
          class="input font-mono text-sm"
          :placeholder="t('admin.proxies.batchInputPlaceholder')"
          @input="emit('parse-batch')"
        ></textarea>
        <p class="input-hint mt-2">
          {{ t('admin.proxies.batchInputHint') }}
        </p>
      </div>

      <div v-if="batchParseResult.total > 0" class="rounded-lg bg-gray-50 p-4 dark:bg-dark-700">
        <div class="flex items-center gap-4 text-sm">
          <div class="flex items-center gap-1.5">
            <Icon name="checkCircle" size="sm" :stroke-width="2" class="text-primary-500" />
            <span class="text-gray-700 dark:text-gray-300">
              {{ t('admin.proxies.parsedCount', { count: batchParseResult.valid }) }}
            </span>
          </div>
          <div v-if="batchParseResult.invalid > 0" class="flex items-center gap-1.5">
            <Icon
              name="exclamationCircle"
              size="sm"
              :stroke-width="2"
              class="text-amber-500"
            />
            <span class="text-amber-600 dark:text-amber-400">
              {{ t('admin.proxies.invalidCount', { count: batchParseResult.invalid }) }}
            </span>
          </div>
          <div v-if="batchParseResult.duplicate > 0" class="flex items-center gap-1.5">
            <svg
              class="h-4 w-4 text-gray-400"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              stroke-width="2"
            >
              <path
                stroke-linecap="round"
                stroke-linejoin="round"
                d="M15.75 17.25v3.375c0 .621-.504 1.125-1.125 1.125h-9.75a1.125 1.125 0 01-1.125-1.125V7.875c0-.621.504-1.125 1.125-1.125H6.75a9.06 9.06 0 011.5.124m7.5 10.376h3.375c.621 0 1.125-.504 1.125-1.125V11.25c0-4.46-3.243-8.161-7.5-8.876a9.06 9.06 0 00-1.5-.124H9.375c-.621 0-1.125.504-1.125 1.125v3.5m7.5 10.375H9.375a1.125 1.125 0 01-1.125-1.125v-9.25m12 6.625v-1.875a3.375 3.375 0 00-3.375-3.375h-1.5a1.125 1.125 0 01-1.125-1.125v-1.5a3.375 3.375 0 00-3.375-3.375H9.75"
              />
            </svg>
            <span class="text-gray-500 dark:text-gray-400">
              {{ t('admin.proxies.duplicateCount', { count: batchParseResult.duplicate }) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="emit('close')" type="button" class="btn btn-secondary">
          {{ t('common.cancel') }}
        </button>
        <button
          v-if="createMode === 'standard'"
          type="submit"
          form="create-proxy-form"
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
          {{ submitting ? t('admin.proxies.creating') : t('common.create') }}
        </button>
        <button
          v-else
          @click="emit('batch-create')"
          type="button"
          :disabled="submitting || batchParseResult.valid === 0"
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
          {{
            submitting
              ? t('admin.proxies.importing')
              : t('admin.proxies.importProxies', { count: batchParseResult.valid })
          }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ProxyProtocol } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

interface ProxyForm {
  name: string
  protocol: ProxyProtocol
  host: string
  port: number
  username: string
  password: string
}

interface BatchParseResult {
  total: number
  valid: number
  invalid: number
  duplicate: number
}

const props = defineProps<{
  show: boolean
  createMode: 'standard' | 'batch'
  createForm: ProxyForm
  protocolSelectOptions: Array<{ value: string; label: string }>
  createPasswordVisible: boolean
  batchInput: string
  batchParseResult: BatchParseResult
  submitting: boolean
}>()

const emit = defineEmits<{
  close: []
  'update:createMode': [mode: 'standard' | 'batch']
  'update:createPasswordVisible': [value: boolean]
  'update:batchInput': [value: string]
  'update:createForm': [value: ProxyForm]
  'parse-batch': []
  create: []
  'batch-create': []
}>()

const { t } = useI18n()

const batchInputModel = computed({
  get: () => props.batchInput,
  set: (value: string) => emit('update:batchInput', value)
})

const updateCreateFormField = (key: keyof ProxyForm, value: unknown) => {
  emit('update:createForm', { ...props.createForm, [key]: value } as ProxyForm)
}
</script>
