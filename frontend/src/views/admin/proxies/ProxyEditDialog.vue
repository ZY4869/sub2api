<template>
  <BaseDialog
    :show="show"
    :title="t('admin.proxies.editProxy')"
    width="normal"
    @close="emit('close')"
  >
    <form
      v-if="editingProxy"
      id="edit-proxy-form"
      @submit.prevent="emit('update')"
      class="space-y-5"
    >
      <div>
        <label class="input-label">{{ t('admin.proxies.name') }}</label>
        <input v-model="editForm.name" type="text" required class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.protocol') }}</label>
        <Select v-model="editForm.protocol" :options="protocolSelectOptions" />
      </div>
      <div class="grid grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('admin.proxies.host') }}</label>
          <input v-model="editForm.host" type="text" required class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.proxies.port') }}</label>
          <input
            v-model.number="editForm.port"
            type="number"
            required
            min="1"
            max="65535"
            class="input"
          />
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.username') }}</label>
        <input v-model="editForm.username" type="text" class="input" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.password') }}</label>
        <div class="relative">
          <input
            v-model="editForm.password"
            :type="editPasswordVisible ? 'text' : 'password'"
            :placeholder="t('admin.proxies.leaveEmptyToKeep')"
            class="input pr-10"
            @input="emit('password-dirty')"
          />
          <button
            type="button"
            class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
            @click="emit('update:editPasswordVisible', !editPasswordVisible)"
          >
            <Icon :name="editPasswordVisible ? 'eyeOff' : 'eye'" size="md" />
          </button>
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.proxies.status') }}</label>
        <Select v-model="editForm.status" :options="editStatusOptions" />
      </div>
    </form>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="emit('close')" type="button" class="btn btn-secondary">
          {{ t('common.cancel') }}
        </button>
        <button
          v-if="editingProxy"
          type="submit"
          form="edit-proxy-form"
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
          {{ submitting ? t('admin.proxies.updating') : t('common.update') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { Proxy, ProxyProtocol } from '@/types'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'

interface ProxyEditForm {
  name: string
  protocol: ProxyProtocol
  host: string
  port: number
  username: string
  password: string
  status: 'active' | 'inactive'
}

defineProps<{
  show: boolean
  editingProxy: Proxy | null
  editForm: ProxyEditForm
  protocolSelectOptions: Array<{ value: string; label: string }>
  editStatusOptions: Array<{ value: string; label: string }>
  editPasswordVisible: boolean
  submitting: boolean
}>()

const emit = defineEmits<{
  close: []
  update: []
  'update:editPasswordVisible': [value: boolean]
  'password-dirty': []
}>()

const { t } = useI18n()
</script>
