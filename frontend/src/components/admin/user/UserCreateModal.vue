<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.createUser')"
    width="normal"
    @close="$emit('close')"
  >
    <form id="create-user-form" @submit.prevent="submit" class="space-y-5">
      <div>
        <label class="input-label">{{ t('admin.users.email') }}</label>
        <input v-model="form.email" type="email" required class="input" :placeholder="t('admin.users.enterEmail')" />
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.password') }}</label>
        <div class="flex gap-2">
          <div class="relative flex-1">
            <input v-model="form.password" type="text" required class="input pr-10" :placeholder="t('admin.users.enterPassword')" />
          </div>
          <button type="button" @click="generateRandomPassword" class="btn btn-secondary px-3">
            <Icon name="refresh" size="md" />
          </button>
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.username') }}</label>
        <input v-model="form.username" type="text" class="input" :placeholder="t('admin.users.enterUsername')" />
      </div>
      <div class="grid grid-cols-1 sm:grid-cols-2 gap-4">
        <div>
          <label class="input-label">{{ t('admin.users.columns.balance') }}</label>
          <input v-model.number="form.balance" type="number" step="any" class="input" />
        </div>
        <div>
          <label class="input-label">{{ t('admin.users.columns.concurrency') }}</label>
          <input v-model.number="form.concurrency" type="number" class="input" />
        </div>
      </div>
      <div>
        <label class="input-label">{{ t('admin.users.apiKeyModelBindingMode') }}</label>
        <select v-model="form.api_key_model_binding_mode" class="input">
          <option value="model_required">{{ t('admin.users.apiKeyModelBindingModeRequired') }}</option>
          <option value="group_allowed">{{ t('admin.users.apiKeyModelBindingModeGroupAllowed') }}</option>
        </select>
        <p class="input-hint">{{ t('admin.users.apiKeyModelBindingModeHint') }}</p>
      </div>
      <div class="space-y-3">
        <label class="flex items-start gap-3 rounded-xl border border-slate-200 bg-slate-50/70 p-4 dark:border-dark-700 dark:bg-dark-900/30">
          <input
            v-model="form.enable_api_key_access_time_policy"
            type="checkbox"
            class="mt-1 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <div>
            <div class="text-sm font-medium text-gray-900 dark:text-white">
              {{ t('admin.users.apiKeyAccessTimePolicy') }}
            </div>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
              {{ t('admin.users.apiKeyAccessTimePolicyHint') }}
            </p>
          </div>
        </label>
        <TimeAccessPolicyEditor
          v-if="form.enable_api_key_access_time_policy"
          v-model="form.api_key_access_time_policy"
          :hint="t('admin.users.apiKeyAccessTimePolicyHint')"
        />
      </div>
    </form>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button @click="$emit('close')" type="button" class="btn btn-secondary">{{ t('common.cancel') }}</button>
        <button type="submit" form="create-user-form" :disabled="loading" class="btn btn-primary">
          {{ loading ? t('admin.users.creating') : t('common.create') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'
import { useI18n } from 'vue-i18n'; import { adminAPI } from '@/api/admin'
import { useForm } from '@/composables/useForm'
import BaseDialog from '@/components/common/BaseDialog.vue'
import TimeAccessPolicyEditor from '@/components/common/TimeAccessPolicyEditor.vue'
import Icon from '@/components/icons/Icon.vue'
import type { APIKeyModelBindingMode, TimeAccessPolicy } from '@/types'
import { buildPresetTimeAccessPolicy, policyToPayload } from '@/utils/timeAccessPolicy'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits(['close', 'success']); const { t } = useI18n()

const form = reactive({
  email: '',
  password: '',
  username: '',
  notes: '',
  balance: 0,
  concurrency: 1,
  api_key_model_binding_mode: 'model_required' as APIKeyModelBindingMode,
  enable_api_key_access_time_policy: false,
  api_key_access_time_policy: buildPresetTimeAccessPolicy('daytime') as TimeAccessPolicy
})

const { loading, submit } = useForm({
  form,
  submitFn: async (data) => {
    const payload = {
      ...data,
      api_key_access_time_policy: data.enable_api_key_access_time_policy
        ? policyToPayload(data.api_key_access_time_policy)
        : undefined,
    }
    delete (payload as { enable_api_key_access_time_policy?: boolean }).enable_api_key_access_time_policy
    await adminAPI.users.create(payload)
    emit('success'); emit('close')
  },
  successMsg: t('admin.users.userCreated')
})

watch(() => props.show, (v) => {
  if(v) Object.assign(form, {
    email: '',
    password: '',
    username: '',
    notes: '',
    balance: 0,
    concurrency: 1,
    api_key_model_binding_mode: 'model_required' as APIKeyModelBindingMode,
    enable_api_key_access_time_policy: false,
    api_key_access_time_policy: buildPresetTimeAccessPolicy('daytime')
  })
})

const generateRandomPassword = () => {
  const chars = 'ABCDEFGHJKLMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz23456789!@#$%^&*'
  let p = ''; for (let i = 0; i < 16; i++) p += chars.charAt(Math.floor(Math.random() * chars.length))
  form.password = p
}
</script>
