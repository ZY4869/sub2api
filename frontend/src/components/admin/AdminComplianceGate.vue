<template>
  <BaseDialog
    :show="dialogVisible"
    :title="t('admin.compliance.title')"
    width="normal"
    :close-on-escape="false"
    :close-on-click-outside="false"
    @close="handleDialogClose"
  >
    <div class="space-y-4 text-sm text-gray-700 dark:text-dark-200">
      <p>{{ t('admin.compliance.message') }}</p>
      <p class="rounded-lg bg-gray-50 px-3 py-2 font-mono text-xs text-gray-600 dark:bg-dark-900 dark:text-dark-300">
        {{ t('admin.compliance.documentMeta', { version: status?.document_version || '-' }) }}
      </p>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          type="button"
          class="btn btn-primary"
          :disabled="submitting"
          @click="acknowledgeCompliance"
        >
          {{ submitting ? t('common.saving') : t('admin.compliance.confirm') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { adminAPI, type AdminComplianceStatus } from '@/api/admin'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'
import BaseDialog from '@/components/common/BaseDialog.vue'

const { t } = useI18n()
const route = useRoute()
const appStore = useAppStore()
const authStore = useAuthStore()

const status = ref<AdminComplianceStatus | null>(null)
const loading = ref(false)
const submitting = ref(false)
const checkedKey = ref('')

const isAdminRoute = computed(() => route.path.startsWith('/admin'))
const shouldCheck = computed(() => authStore.isAdmin && isAdminRoute.value)
const dialogVisible = computed(() => !!status.value?.enabled && !!status.value?.required)

async function refreshStatus(force = false) {
  if (!shouldCheck.value || loading.value) return
  if (!force && checkedKey.value === route.path && status.value && !status.value.required) return

  loading.value = true
  try {
    status.value = await adminAPI.compliance.getStatus()
    checkedKey.value = route.path
  } catch (error: any) {
    const code = error?.reason || error?.code
    if (code === 'ADMIN_COMPLIANCE_REQUIRED') {
      status.value = {
        enabled: true,
        required: true,
        document_version: error?.metadata?.document_version || '',
        document_hash: error?.metadata?.document_hash || '',
      }
      checkedKey.value = route.path
      return
    }
    appStore.showError(error?.message || t('admin.compliance.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function acknowledgeCompliance() {
  submitting.value = true
  try {
    status.value = await adminAPI.compliance.acknowledge()
    appStore.showSuccess(t('admin.compliance.confirmed'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.compliance.confirmFailed'))
  } finally {
    submitting.value = false
  }
}

function handleDialogClose() {
  if (!dialogVisible.value) return
  appStore.showWarning(t('admin.compliance.required'))
}

watch(
  () => [shouldCheck.value, route.path],
  () => {
    void refreshStatus()
  },
)

onMounted(() => {
  void refreshStatus(true)
})
</script>
