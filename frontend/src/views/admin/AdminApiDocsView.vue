<template>
  <AppLayout>
    <div class="space-y-6">
      <section class="overflow-hidden rounded-[2rem] border border-slate-200 bg-[radial-gradient(circle_at_top_left,_rgba(15,118,110,0.12),_transparent_34%),linear-gradient(135deg,_rgba(255,255,255,0.98),_rgba(240,253,250,0.92))] p-6 shadow-sm dark:border-dark-700 dark:bg-[radial-gradient(circle_at_top_left,_rgba(45,212,191,0.12),_transparent_34%),linear-gradient(135deg,_rgba(17,24,39,0.96),_rgba(15,23,42,0.92))] md:p-8">
        <div class="flex flex-col gap-5 lg:flex-row lg:items-end lg:justify-between">
          <div class="max-w-3xl">
            <p class="text-xs font-semibold uppercase tracking-[0.28em] text-teal-600 dark:text-teal-300">
              {{ t('admin.apiDocs.eyebrow') }}
            </p>
            <h1 class="mt-3 text-3xl font-semibold tracking-tight text-slate-950 dark:text-white">
              {{ t('admin.apiDocs.title') }}
            </h1>
            <p class="mt-3 text-sm leading-7 text-slate-900 dark:text-slate-100 md:text-base">
              {{ t('admin.apiDocs.description') }}
            </p>
          </div>
          <div class="flex flex-wrap items-center gap-2">
            <span class="rounded-full border border-slate-200 bg-white/80 px-4 py-2 text-sm text-slate-900 dark:border-dark-700 dark:bg-dark-900/80 dark:text-slate-100">
              {{ hasOverride ? t('admin.apiDocs.overrideActive') : t('admin.apiDocs.usingDefault') }}
            </span>
            <span
              v-if="isDirty"
              class="rounded-full border border-amber-200 bg-amber-50 px-4 py-2 text-sm text-amber-700 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200"
            >
              {{ t('admin.apiDocs.unsavedChanges') }}
            </span>
          </div>
        </div>
      </section>

      <div class="rounded-2xl border border-slate-200 bg-white/90 p-2 shadow-sm dark:border-dark-700 dark:bg-dark-900/80">
        <div class="flex flex-wrap gap-2">
          <button
            type="button"
            class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
            :class="activeTab === 'preview' ? activeTabClass : inactiveTabClass"
            data-test="api-docs-tab-preview"
            @click="setActiveTab('preview')"
          >
            {{ t('admin.apiDocs.previewTab') }}
          </button>
          <button
            type="button"
            class="rounded-xl px-4 py-2 text-sm font-medium transition-colors"
            :class="activeTab === 'edit' ? activeTabClass : inactiveTabClass"
            data-test="api-docs-tab-edit"
            @click="setActiveTab('edit')"
          >
            {{ t('admin.apiDocs.editorTab') }}
          </button>
        </div>
      </div>

      <div v-if="loading" class="rounded-3xl border border-slate-200 bg-white/90 px-6 py-16 text-center text-sm text-slate-800 dark:border-dark-700 dark:bg-dark-900/80 dark:text-slate-100">
        {{ t('admin.apiDocs.loading') }}
      </div>

      <div v-else-if="errorMessage" class="rounded-3xl border border-rose-200 bg-rose-50 px-6 py-6 text-sm text-rose-700 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200">
        {{ errorMessage }}
      </div>

      <section v-else-if="activeTab === 'preview'" data-test="api-docs-preview" class="min-w-0">
        <DocsMarkdownContent
          :markdown="draft"
          :page-id="routePageId"
          base-path="/admin/api-docs"
          :nav-title="t('admin.apiDocs.protocolsTitle')"
          :toc-title="t('admin.apiDocs.pageTocTitle')"
          preview-mode
        />
      </section>

      <section
        v-else
        data-test="api-docs-editor"
        class="rounded-[2rem] border border-slate-200 bg-white/90 shadow-sm dark:border-dark-700 dark:bg-dark-900/80"
      >
        <div class="flex flex-col gap-3 border-b border-slate-200 px-6 py-5 dark:border-dark-700">
          <div class="flex flex-wrap items-center justify-between gap-3">
            <div>
              <p class="text-xs font-semibold uppercase tracking-[0.24em] text-slate-800 dark:text-slate-200">
                {{ t('admin.apiDocs.editorEyebrow') }}
              </p>
              <h2 class="mt-2 text-xl font-semibold text-slate-950 dark:text-white">
                {{ t('admin.apiDocs.editorTitle') }}
              </h2>
            </div>
            <div class="flex flex-wrap gap-2">
              <button type="button" class="btn btn-primary" :disabled="saving || !isDirty" @click="handleSave">
                {{ saving ? t('admin.apiDocs.saving') : t('admin.apiDocs.save') }}
              </button>
              <button type="button" class="btn btn-secondary" :disabled="saving" @click="handleReset">
                {{ t('admin.apiDocs.restoreDefault') }}
              </button>
              <button type="button" class="btn btn-secondary" @click="handleCopy">
                {{ t('admin.apiDocs.copy') }}
              </button>
              <button type="button" class="btn btn-secondary" @click="openUserPage">
                {{ t('admin.apiDocs.openUserPage') }}
              </button>
            </div>
          </div>
          <p class="text-sm leading-6 text-slate-900 dark:text-slate-100">
            {{ t('admin.apiDocs.editorDescription') }}
          </p>
        </div>

        <div class="p-6">
          <textarea
            v-model="draft"
            class="min-h-[560px] w-full rounded-[1.5rem] border border-slate-200 bg-slate-50 px-4 py-4 font-mono text-sm leading-7 text-slate-900 outline-none transition focus:border-primary-400 focus:ring-2 focus:ring-primary-100 dark:border-dark-700 dark:bg-dark-950 dark:text-slate-100 dark:focus:ring-primary-900/40"
            :placeholder="t('admin.apiDocs.editorPlaceholder')"
            spellcheck="false"
          />
        </div>
      </section>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import DocsMarkdownContent from '@/components/docs/DocsMarkdownContent.vue'
import adminDocsAPI from '@/api/admin/docs'
import { useClipboard } from '@/composables/useClipboard'
import { useAppStore } from '@/stores/app'
import { normalizeDocsPageId } from '@/utils/markdownDocs'

type AdminDocsTab = 'preview' | 'edit'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

const activeTabClass = 'bg-primary-600 text-white'
const inactiveTabClass = 'text-slate-900 hover:bg-slate-100 dark:text-slate-100 dark:hover:bg-dark-800'

const loading = ref(true)
const saving = ref(false)
const errorMessage = ref('')
const draft = ref('')
const effectiveContent = ref('')
const defaultContent = ref('')
const hasOverride = ref(false)

const routePageId = computed(() => normalizeDocsPageId(route.params.pageId as string | undefined))
const activeTab = computed<AdminDocsTab>(() => (
  normalizeAdminDocsTab(route.query.tab)
))
const isDirty = computed(() => draft.value !== effectiveContent.value)

watch(
  () => route.query.tab,
  () => {
    ensureValidTabQuery()
  },
)

function normalizeAdminDocsTab(tab: unknown): AdminDocsTab {
  return tab === 'edit' ? 'edit' : 'preview'
}

function ensureValidTabQuery() {
  const current = route.query.tab
  if (current === 'preview' || current === 'edit') {
    return
  }
  router.replace({
    query: {
      ...route.query,
      tab: 'preview',
    },
  })
}

function setActiveTab(tab: AdminDocsTab) {
  if (activeTab.value === tab) {
    return
  }
  router.replace({
    query: {
      ...route.query,
      tab,
    },
  })
}

function applyDocument(document: {
  effective_content: string
  default_content: string
  has_override: boolean
}) {
  effectiveContent.value = document.effective_content
  defaultContent.value = document.default_content
  hasOverride.value = document.has_override
  draft.value = document.effective_content
}

async function loadDocument() {
  loading.value = true
  errorMessage.value = ''
  try {
    applyDocument(await adminDocsAPI.getAPIDocs(routePageId.value))
  } catch (error: any) {
    errorMessage.value = error?.message || t('admin.apiDocs.loadFailed')
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  if (!isDirty.value || saving.value) {
    return
  }
  saving.value = true
  try {
    applyDocument(await adminDocsAPI.updateAPIDocs(draft.value, routePageId.value))
    appStore.showSuccess(t('admin.apiDocs.saveSuccess'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.apiDocs.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function handleReset() {
  const message = hasOverride.value
    ? t('admin.apiDocs.restoreConfirm')
    : t('admin.apiDocs.restoreDraftConfirm')
  if (!window.confirm(message)) {
    return
  }

  saving.value = true
  try {
    if (hasOverride.value) {
      applyDocument(await adminDocsAPI.clearAPIDocsOverride(routePageId.value))
    } else {
      draft.value = defaultContent.value
      effectiveContent.value = defaultContent.value
    }
    appStore.showSuccess(t('admin.apiDocs.restoreSuccess'))
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.apiDocs.restoreFailed'))
  } finally {
    saving.value = false
  }
}

async function handleCopy() {
  await copyToClipboard(draft.value, t('admin.apiDocs.copySuccess'))
}

function openUserPage() {
  const target = router.resolve(`/api-docs/${routePageId.value}`).href
  window.open(target, '_blank', 'noopener')
}

watch(
  routePageId,
  () => {
    loadDocument().catch((error) => {
      console.error('Failed to load admin API docs:', error)
    })
  },
  { immediate: true }
)

onMounted(() => {
  ensureValidTabQuery()
})
</script>
