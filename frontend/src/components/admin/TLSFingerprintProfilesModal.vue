<template>
  <BaseDialog
    :show="show"
    :title="t('admin.tlsFingerprintProfiles.title')"
    width="extra-wide"
    @close="$emit('close')"
  >
    <div class="space-y-4">
      <div class="flex items-center justify-between gap-3">
        <p class="text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.tlsFingerprintProfiles.description') }}
        </p>
        <button type="button" class="btn btn-primary btn-sm" @click="openCreateDialog">
          <Icon name="plus" size="sm" class="mr-1" />
          {{ t('admin.tlsFingerprintProfiles.create') }}
        </button>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-10">
        <Icon name="refresh" size="lg" class="animate-spin text-gray-400" />
      </div>

      <div v-else-if="profiles.length === 0" class="rounded-lg border border-dashed border-gray-300 p-8 text-center dark:border-dark-500">
        <p class="text-sm font-medium text-gray-900 dark:text-white">
          {{ t('admin.tlsFingerprintProfiles.emptyTitle') }}
        </p>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
          {{ t('admin.tlsFingerprintProfiles.emptyDescription') }}
        </p>
      </div>

      <div v-else class="max-h-[28rem] overflow-auto rounded-lg border border-gray-200 dark:border-dark-600">
        <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
          <thead class="sticky top-0 bg-gray-50 dark:bg-dark-700">
            <tr>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.tlsFingerprintProfiles.columns.name') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.tlsFingerprintProfiles.columns.grease') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.tlsFingerprintProfiles.columns.summary') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.tlsFingerprintProfiles.columns.updatedAt') }}
              </th>
              <th class="px-3 py-2 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
                {{ t('admin.tlsFingerprintProfiles.columns.actions') }}
              </th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
            <tr v-for="profile in profiles" :key="profile.id" class="hover:bg-gray-50 dark:hover:bg-dark-700">
              <td class="px-3 py-2">
                <div class="text-sm font-medium text-gray-900 dark:text-white">{{ profile.name }}</div>
                <div v-if="profile.description" class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ profile.description }}
                </div>
              </td>
              <td class="whitespace-nowrap px-3 py-2 text-sm text-gray-700 dark:text-gray-300">
                {{ profile.enable_grease ? t('common.enabled') : t('common.disabled') }}
              </td>
              <td class="px-3 py-2 text-xs text-gray-600 dark:text-gray-300">
                {{ summarizeProfile(profile) }}
              </td>
              <td class="whitespace-nowrap px-3 py-2 text-xs text-gray-500 dark:text-gray-400">
                {{ formatDateTime(profile.updated_at) }}
              </td>
              <td class="px-3 py-2">
                <div class="flex items-center gap-1">
                  <button
                    type="button"
                    class="p-1 text-gray-500 hover:text-primary-600 dark:hover:text-primary-400"
                    :title="t('common.edit')"
                    @click="openEditDialog(profile)"
                  >
                    <Icon name="edit" size="sm" />
                  </button>
                  <button
                    type="button"
                    class="p-1 text-gray-500 hover:text-red-600 dark:hover:text-red-400"
                    :title="t('common.delete')"
                    @click="openDeleteDialog(profile)"
                  >
                    <Icon name="trash" size="sm" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end">
        <button type="button" class="btn btn-secondary" @click="$emit('close')">
          {{ t('common.close') }}
        </button>
      </div>
    </template>

    <BaseDialog
      :show="showFormDialog"
      :title="editingProfile ? t('admin.tlsFingerprintProfiles.edit') : t('admin.tlsFingerprintProfiles.create')"
      width="wide"
      @close="closeFormDialog"
    >
      <form class="space-y-4" @submit.prevent="submitForm">
        <div class="grid grid-cols-2 gap-4">
          <div>
            <label class="input-label">{{ t('admin.tlsFingerprintProfiles.form.name') }}</label>
            <input v-model="form.name" type="text" required class="input" />
          </div>
          <label class="flex items-center gap-2 self-end pb-2 text-sm text-gray-700 dark:text-gray-300">
            <input v-model="form.enable_grease" type="checkbox" class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('admin.tlsFingerprintProfiles.form.enableGrease') }}</span>
          </label>
        </div>

        <div>
          <label class="input-label">{{ t('admin.tlsFingerprintProfiles.form.description') }}</label>
          <textarea v-model="form.description" rows="2" class="input" />
        </div>

        <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
          <div v-for="field in numericFields" :key="field.key">
            <label class="input-label text-xs">{{ t(field.labelKey) }}</label>
            <textarea v-model="form[field.key]" rows="2" class="input font-mono text-xs" :placeholder="t('admin.tlsFingerprintProfiles.form.numberListPlaceholder')" />
          </div>
          <div v-for="field in stringFields" :key="field.key">
            <label class="input-label text-xs">{{ t(field.labelKey) }}</label>
            <textarea v-model="form[field.key]" rows="2" class="input font-mono text-xs" :placeholder="t('admin.tlsFingerprintProfiles.form.stringListPlaceholder')" />
          </div>
        </div>
      </form>

      <template #footer>
        <div class="flex justify-end gap-3">
          <button type="button" class="btn btn-secondary" @click="closeFormDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="button" class="btn btn-primary" :disabled="submitting" @click="submitForm">
            <Icon v-if="submitting" name="refresh" size="sm" class="mr-1 animate-spin" />
            {{ editingProfile ? t('common.update') : t('common.create') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <ConfirmDialog
      :show="showDeleteDialog"
      :title="t('admin.tlsFingerprintProfiles.deleteTitle')"
      :message="t('admin.tlsFingerprintProfiles.deleteConfirm', { name: deletingProfile?.name || '' })"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmDelete"
      @cancel="showDeleteDialog = false"
    />
  </BaseDialog>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  CreateTLSFingerprintProfileRequest,
  TLSFingerprintProfile,
  UpdateTLSFingerprintProfileRequest
} from '@/api/admin/tlsFingerprintProfile'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ show: boolean }>()
defineEmits<{ close: [] }>()

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(false)
const submitting = ref(false)
const profiles = ref<TLSFingerprintProfile[]>([])
const editingProfile = ref<TLSFingerprintProfile | null>(null)
const deletingProfile = ref<TLSFingerprintProfile | null>(null)
const showFormDialog = ref(false)
const showDeleteDialog = ref(false)

const numericFields = [
  { key: 'cipher_suites', labelKey: 'admin.tlsFingerprintProfiles.form.cipherSuites' },
  { key: 'curves', labelKey: 'admin.tlsFingerprintProfiles.form.curves' },
  { key: 'point_formats', labelKey: 'admin.tlsFingerprintProfiles.form.pointFormats' },
  { key: 'signature_algorithms', labelKey: 'admin.tlsFingerprintProfiles.form.signatureAlgorithms' },
  { key: 'supported_versions', labelKey: 'admin.tlsFingerprintProfiles.form.supportedVersions' },
  { key: 'key_share_groups', labelKey: 'admin.tlsFingerprintProfiles.form.keyShareGroups' },
  { key: 'psk_modes', labelKey: 'admin.tlsFingerprintProfiles.form.pskModes' },
  { key: 'extensions', labelKey: 'admin.tlsFingerprintProfiles.form.extensions' }
] as const
const stringFields = [
  { key: 'alpn_protocols', labelKey: 'admin.tlsFingerprintProfiles.form.alpnProtocols' }
] as const

interface ProfileFormState {
  name: string
  description: string
  enable_grease: boolean
  cipher_suites: string
  curves: string
  point_formats: string
  signature_algorithms: string
  alpn_protocols: string
  supported_versions: string
  key_share_groups: string
  psk_modes: string
  extensions: string
}

const form = reactive<ProfileFormState>({
  name: '',
  description: '',
  enable_grease: false,
  cipher_suites: '',
  curves: '',
  point_formats: '',
  signature_algorithms: '',
  alpn_protocols: '',
  supported_versions: '',
  key_share_groups: '',
  psk_modes: '',
  extensions: ''
})

const loadProfiles = async () => {
  loading.value = true
  try {
    profiles.value = await adminAPI.tlsFingerprintProfiles.list()
  } catch (error: any) {
    console.error('Failed to load TLS fingerprint profiles:', error)
    appStore.showError(error?.message || t('admin.tlsFingerprintProfiles.loadFailed'))
  } finally {
    loading.value = false
  }
}

watch(
  () => props.show,
  (show) => {
    if (show) {
      loadProfiles().catch((error) => console.error('Failed to refresh TLS fingerprint profiles:', error))
      return
    }
    closeFormDialog()
    showDeleteDialog.value = false
    deletingProfile.value = null
  },
  { immediate: true }
)

const resetForm = () => {
  form.name = ''
  form.description = ''
  form.enable_grease = false
  for (const field of [...numericFields, ...stringFields]) {
    form[field.key] = ''
  }
}

const populateForm = (profile: TLSFingerprintProfile) => {
  form.name = profile.name
  form.description = profile.description || ''
  form.enable_grease = profile.enable_grease
  for (const field of numericFields) {
    form[field.key] = profile[field.key].join(', ')
  }
  for (const field of stringFields) {
    form[field.key] = profile[field.key].join(', ')
  }
}

const openCreateDialog = () => {
  editingProfile.value = null
  resetForm()
  showFormDialog.value = true
}

const openEditDialog = (profile: TLSFingerprintProfile) => {
  editingProfile.value = profile
  populateForm(profile)
  showFormDialog.value = true
}

const closeFormDialog = () => {
  showFormDialog.value = false
  editingProfile.value = null
  resetForm()
}

const openDeleteDialog = (profile: TLSFingerprintProfile) => {
  deletingProfile.value = profile
  showDeleteDialog.value = true
}

const parseNumberList = (value: string): number[] =>
  value
    .split(/[,\n]/)
    .map((item) => Number.parseInt(item.trim(), 10))
    .filter((item) => Number.isFinite(item) && item >= 0 && item <= 65535)

const parseStringList = (value: string): string[] =>
  value
    .split(/[,\n]/)
    .map((item) => item.trim())
    .filter((item) => item.length > 0)

const buildPayload = (): CreateTLSFingerprintProfileRequest | UpdateTLSFingerprintProfileRequest => {
  const payload: CreateTLSFingerprintProfileRequest = {
    name: String(form.name).trim(),
    description: String(form.description).trim() || null,
    enable_grease: Boolean(form.enable_grease)
  }
  for (const field of numericFields) {
    payload[field.key] = parseNumberList(String(form[field.key] || ''))
  }
  for (const field of stringFields) {
    payload[field.key] = parseStringList(String(form[field.key] || ''))
  }
  return payload
}

const submitForm = async () => {
  if (!String(form.name).trim()) {
    appStore.showError(t('admin.tlsFingerprintProfiles.nameRequired'))
    return
  }
  submitting.value = true
  try {
    const payload = buildPayload()
    if (editingProfile.value) {
      await adminAPI.tlsFingerprintProfiles.update(editingProfile.value.id, payload)
      appStore.showSuccess(t('admin.tlsFingerprintProfiles.updated'))
    } else {
      await adminAPI.tlsFingerprintProfiles.create(payload as CreateTLSFingerprintProfileRequest)
      appStore.showSuccess(t('admin.tlsFingerprintProfiles.created'))
    }
    closeFormDialog()
    await loadProfiles()
  } catch (error: any) {
    console.error('Failed to save TLS fingerprint profile:', error)
    appStore.showError(error?.message || t('admin.tlsFingerprintProfiles.saveFailed'))
  } finally {
    submitting.value = false
  }
}

const confirmDelete = async () => {
  if (!deletingProfile.value) {
    return
  }
  try {
    await adminAPI.tlsFingerprintProfiles.delete(deletingProfile.value.id)
    appStore.showSuccess(t('admin.tlsFingerprintProfiles.deleted'))
    showDeleteDialog.value = false
    deletingProfile.value = null
    await loadProfiles()
  } catch (error: any) {
    console.error('Failed to delete TLS fingerprint profile:', error)
    appStore.showError(error?.message || t('admin.tlsFingerprintProfiles.deleteFailed'))
  }
}

const summarizeProfile = (profile: TLSFingerprintProfile) =>
  t('admin.tlsFingerprintProfiles.summary', {
    cipherSuites: profile.cipher_suites.length,
    curves: profile.curves.length,
    extensions: profile.extensions.length,
    alpn: profile.alpn_protocols.length
  })

const formatDateTime = (value: string) => {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
</script>
