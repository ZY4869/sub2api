<template>
  <AppLayout>
    <TablePageLayout prefer-page-scroll>
      <template #actions>
        <div class="flex flex-wrap items-start justify-between gap-3">
          <div
            class="rounded-2xl border border-gray-200 bg-white px-4 py-3 text-sm text-gray-600 shadow-sm dark:border-dark-700 dark:bg-dark-800 dark:text-gray-300"
          >
            <div class="text-base font-semibold text-gray-900 dark:text-white">
              {{ t('admin.channelMonitors.title') }}
            </div>
            <div class="mt-1 text-sm text-gray-600 dark:text-gray-300">
              {{ t('admin.channelMonitors.description') }}
            </div>
          </div>

          <div class="flex flex-wrap items-center gap-3">
            <div class="flex items-center gap-1 rounded-2xl border border-gray-200 bg-white p-1 shadow-sm dark:border-dark-700 dark:bg-dark-800">
              <button
                type="button"
                class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
                :class="activeTab === 'monitors'
                  ? 'bg-primary-600 text-white shadow-sm'
                  : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'"
                @click="activeTab = 'monitors'"
              >
                {{ t('admin.channelMonitors.tabs.monitors') }}
              </button>
              <button
                type="button"
                class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
                :class="activeTab === 'templates'
                  ? 'bg-primary-600 text-white shadow-sm'
                  : 'text-gray-600 hover:bg-gray-100 hover:text-gray-900 dark:text-gray-300 dark:hover:bg-dark-700 dark:hover:text-white'"
                @click="activeTab = 'templates'"
              >
                {{ t('admin.channelMonitors.tabs.templates') }}
              </button>
            </div>

            <button class="btn btn-secondary" :disabled="loadingAny" @click="refreshAll">
              {{ t('common.refresh') }}
            </button>

            <button
              v-if="activeTab === 'monitors'"
              class="btn btn-primary"
              :disabled="loadingAny"
              @click="openCreateMonitor"
            >
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.channelMonitors.actions.createMonitor') }}
            </button>
            <button
              v-else
              class="btn btn-primary"
              :disabled="loadingAny"
              @click="openCreateTemplate"
            >
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.channelMonitors.actions.createTemplate') }}
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <ChannelMonitorsTable
          v-if="activeTab === 'monitors'"
          :items="monitors"
          :templates="templates"
          :loading="loadingMonitors"
          @edit="openEditMonitor"
          @delete="confirmDeleteMonitor"
          @run="runNow"
          @history="openHistory"
          @toggleEnabled="toggleEnabled"
        />

        <ChannelMonitorTemplatesTable
          v-else
          :items="templates"
          :loading="loadingTemplates"
          @edit="openEditTemplate"
          @delete="confirmDeleteTemplate"
          @apply="openTemplateApply"
          @associated="openAssociated"
        />
      </template>
    </TablePageLayout>

    <ChannelMonitorFormDialog
      :show="monitorDialogOpen"
      :monitor="editingMonitor"
      :templates="templates"
      @close="monitorDialogOpen = false"
      @saved="handleMonitorSaved"
    />

    <ChannelMonitorTemplateFormDialog
      :show="templateDialogOpen"
      :template="editingTemplate"
      @close="templateDialogOpen = false"
      @saved="handleTemplateSaved"
    />

    <ChannelMonitorHistoryDialog
      :show="historyDialogOpen"
      :monitor="historyMonitor"
      :initial-histories="historyInitial"
      @close="closeHistory"
    />

    <ChannelMonitorTemplateApplyDialog
      :show="applyDialogOpen"
      :template="applyTemplate"
      :monitors="monitors"
      @close="applyDialogOpen = false"
      @applied="handleTemplateApplied"
    />

    <ChannelMonitorAssociatedDialog
      :show="associatedDialogOpen"
      :template="associatedTemplate"
      @close="associatedDialogOpen = false"
    />

    <ConfirmDialog
      :show="deleteDialogOpen"
      :title="deleteDialogTitle"
      :message="deleteDialogMessage"
      danger
      @cancel="deleteDialogOpen = false"
      @confirm="handleDeleteConfirmed"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAppStore } from '@/stores'
import { adminAPI } from '@/api/admin'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import ChannelMonitorsTable from '@/components/admin/channel-monitors/ChannelMonitorsTable.vue'
import ChannelMonitorTemplatesTable from '@/components/admin/channel-monitors/ChannelMonitorTemplatesTable.vue'
import ChannelMonitorFormDialog from '@/components/admin/channel-monitors/ChannelMonitorFormDialog.vue'
import ChannelMonitorTemplateFormDialog from '@/components/admin/channel-monitors/ChannelMonitorTemplateFormDialog.vue'
import ChannelMonitorHistoryDialog from '@/components/admin/channel-monitors/ChannelMonitorHistoryDialog.vue'
import ChannelMonitorTemplateApplyDialog from '@/components/admin/channel-monitors/ChannelMonitorTemplateApplyDialog.vue'
import ChannelMonitorAssociatedDialog from '@/components/admin/channel-monitors/ChannelMonitorAssociatedDialog.vue'
import type {
  AdminChannelMonitor,
  AdminChannelMonitorHistory,
  AdminChannelMonitorTemplate
} from '@/api/admin/channelMonitors'

const { t } = useI18n()
const appStore = useAppStore()

const activeTab = ref<'monitors' | 'templates'>('monitors')

const monitors = ref<AdminChannelMonitor[]>([])
const templates = ref<AdminChannelMonitorTemplate[]>([])

const loadingMonitors = ref(false)
const loadingTemplates = ref(false)
const loadingAny = computed(() => loadingMonitors.value || loadingTemplates.value)

const monitorDialogOpen = ref(false)
const editingMonitor = ref<AdminChannelMonitor | null>(null)

const templateDialogOpen = ref(false)
const editingTemplate = ref<AdminChannelMonitorTemplate | null>(null)

const historyDialogOpen = ref(false)
const historyMonitor = ref<AdminChannelMonitor | null>(null)
const historyInitial = ref<AdminChannelMonitorHistory[] | null>(null)

const applyDialogOpen = ref(false)
const applyTemplate = ref<AdminChannelMonitorTemplate | null>(null)

const associatedDialogOpen = ref(false)
const associatedTemplate = ref<AdminChannelMonitorTemplate | null>(null)

const deleteDialogOpen = ref(false)
const deleteTarget = ref<{ type: 'monitor' | 'template'; id: number; name: string } | null>(null)

const deleteDialogTitle = computed(() => t('common.confirm'))
const deleteDialogMessage = computed(() => {
  const target = deleteTarget.value
  if (!target) return ''
  if (target.type === 'monitor') {
    return t('admin.channelMonitors.confirm.deleteMonitor', { name: target.name })
  }
  return t('admin.channelMonitors.confirm.deleteTemplate', { name: target.name })
})

async function loadMonitors() {
  loadingMonitors.value = true
  try {
    monitors.value = await adminAPI.channelMonitors.listMonitors()
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.messages.loadFailed'))
  } finally {
    loadingMonitors.value = false
  }
}

async function loadTemplates() {
  loadingTemplates.value = true
  try {
    templates.value = await adminAPI.channelMonitors.listTemplates()
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.messages.loadFailed'))
  } finally {
    loadingTemplates.value = false
  }
}

async function refreshAll() {
  await Promise.all([loadMonitors(), loadTemplates()])
}

function openCreateMonitor() {
  editingMonitor.value = null
  monitorDialogOpen.value = true
}

function openEditMonitor(m: AdminChannelMonitor) {
  editingMonitor.value = m
  monitorDialogOpen.value = true
}

async function handleMonitorSaved() {
  monitorDialogOpen.value = false
  await loadMonitors()
}

function openCreateTemplate() {
  editingTemplate.value = null
  templateDialogOpen.value = true
}

function openEditTemplate(tpl: AdminChannelMonitorTemplate) {
  editingTemplate.value = tpl
  templateDialogOpen.value = true
}

async function handleTemplateSaved() {
  templateDialogOpen.value = false
  await loadTemplates()
}

function openHistory(m: AdminChannelMonitor) {
  historyMonitor.value = m
  historyInitial.value = null
  historyDialogOpen.value = true
}

function closeHistory() {
  historyDialogOpen.value = false
  historyInitial.value = null
  historyMonitor.value = null
}

async function runNow(m: AdminChannelMonitor) {
  historyMonitor.value = m
  historyDialogOpen.value = true
  historyInitial.value = null
  try {
    historyInitial.value = await adminAPI.channelMonitors.runMonitor(m.id)
    appStore.showSuccess(t('admin.channelMonitors.messages.ran'))
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.messages.runFailed'))
  }
}

async function toggleEnabled(m: AdminChannelMonitor, enabled: boolean) {
  try {
    await adminAPI.channelMonitors.updateMonitor(m.id, { enabled })
    await loadMonitors()
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.messages.saveFailed'))
  }
}

function openTemplateApply(tpl: AdminChannelMonitorTemplate) {
  applyTemplate.value = tpl
  applyDialogOpen.value = true
}

async function handleTemplateApplied() {
  applyDialogOpen.value = false
  await loadMonitors()
}

function openAssociated(tpl: AdminChannelMonitorTemplate) {
  associatedTemplate.value = tpl
  associatedDialogOpen.value = true
}

function confirmDeleteMonitor(m: AdminChannelMonitor) {
  deleteTarget.value = { type: 'monitor', id: m.id, name: m.name }
  deleteDialogOpen.value = true
}

function confirmDeleteTemplate(tpl: AdminChannelMonitorTemplate) {
  deleteTarget.value = { type: 'template', id: tpl.id, name: tpl.name }
  deleteDialogOpen.value = true
}

async function handleDeleteConfirmed() {
  const target = deleteTarget.value
  if (!target) return

  try {
    if (target.type === 'monitor') {
      await adminAPI.channelMonitors.deleteMonitor(target.id)
      await loadMonitors()
    } else {
      await adminAPI.channelMonitors.deleteTemplate(target.id)
      await loadTemplates()
    }
    appStore.showSuccess(t('admin.channelMonitors.messages.deleted'))
  } catch (err) {
    appStore.showError(t('admin.channelMonitors.messages.deleteFailed'))
  } finally {
    deleteDialogOpen.value = false
    deleteTarget.value = null
  }
}

onMounted(() => {
  refreshAll()
})
</script>

