<template>
  <div class="space-y-6">
    <div class="card p-6">
      <div class="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ t('admin.settings.googleBatchGcs.title') }}
          </h3>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
            {{ t('admin.settings.googleBatchGcs.description') }}
          </p>
        </div>
        <div class="flex flex-wrap gap-2">
          <button type="button" class="btn btn-secondary btn-sm" @click="startCreateProfile">
            {{ t('admin.settings.googleBatchGcs.newProfile') }}
          </button>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="loadingProfiles"
            @click="loadProfiles"
          >
            {{ loadingProfiles ? t('common.loading') : t('admin.settings.googleBatchGcs.reloadProfiles') }}
          </button>
        </div>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full min-w-[920px] text-sm">
          <thead>
            <tr class="border-b border-gray-200 text-left text-xs uppercase tracking-wide text-gray-500 dark:border-dark-700 dark:text-gray-400">
              <th class="py-2 pr-4">{{ t('admin.settings.googleBatchGcs.columns.profile') }}</th>
              <th class="py-2 pr-4">{{ t('admin.settings.googleBatchGcs.columns.active') }}</th>
              <th class="py-2 pr-4">{{ t('admin.settings.googleBatchGcs.columns.bucket') }}</th>
              <th class="py-2 pr-4">{{ t('admin.settings.googleBatchGcs.columns.project') }}</th>
              <th class="py-2 pr-4">{{ t('admin.settings.googleBatchGcs.columns.updatedAt') }}</th>
              <th class="py-2">{{ t('admin.settings.googleBatchGcs.columns.actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="profile in profiles"
              :key="profile.profile_id"
              class="border-b border-gray-100 align-top dark:border-dark-800"
            >
              <td class="py-3 pr-4">
                <div class="font-mono text-xs">{{ profile.profile_id }}</div>
                <div class="mt-1 text-xs text-gray-600 dark:text-gray-400">{{ profile.name }}</div>
                <div class="mt-2 flex flex-wrap gap-2">
                  <span
                    class="rounded px-2 py-0.5 text-xs"
                    :class="profile.enabled ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300' : 'bg-gray-100 text-gray-700 dark:bg-dark-800 dark:text-gray-300'"
                  >
                    {{ profile.enabled ? t('common.enabled') : t('common.disabled') }}
                  </span>
                  <span
                    class="rounded px-2 py-0.5 text-xs"
                    :class="profile.service_account_json_configured ? 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'"
                  >
                    {{ profile.service_account_json_configured ? t('admin.settings.googleBatchGcs.configuredBadge') : t('admin.settings.googleBatchGcs.serviceAccountJson') }}
                  </span>
                </div>
              </td>
              <td class="py-3 pr-4">
                <span
                  class="rounded px-2 py-0.5 text-xs"
                  :class="profile.is_active ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' : 'bg-gray-100 text-gray-700 dark:bg-dark-800 dark:text-gray-300'"
                >
                  {{ profile.is_active ? t('common.enabled') : t('common.disabled') }}
                </span>
              </td>
              <td class="py-3 pr-4 text-xs">
                <div>{{ profile.bucket || '-' }}</div>
                <div class="mt-1 text-gray-500 dark:text-gray-400">{{ profile.prefix || '-' }}</div>
              </td>
              <td class="py-3 pr-4 text-xs">{{ profile.project_id || '-' }}</td>
              <td class="py-3 pr-4 text-xs">{{ formatDate(profile.updated_at) }}</td>
              <td class="py-3 text-xs">
                <div class="flex flex-wrap gap-2">
                  <button type="button" class="btn btn-secondary btn-xs" @click="editProfile(profile.profile_id)">
                    {{ t('common.edit') }}
                  </button>
                  <button
                    v-if="!profile.is_active"
                    type="button"
                    class="btn btn-secondary btn-xs"
                    :disabled="activatingProfile"
                    @click="activateProfile(profile.profile_id)"
                  >
                    {{ t('admin.settings.googleBatchGcs.activateProfile') }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-xs"
                    :disabled="testingProfile"
                    @click="testExistingProfile(profile.profile_id)"
                  >
                    {{ testingProfile ? t('admin.settings.googleBatchGcs.testing') : t('admin.settings.googleBatchGcs.testConnection') }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-danger btn-xs"
                    :disabled="deletingProfile"
                    @click="removeProfile(profile.profile_id)"
                  >
                    {{ t('common.delete') }}
                  </button>
                </div>
              </td>
            </tr>
            <tr v-if="profiles.length === 0">
              <td colspan="6" class="py-6 text-center text-sm text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.googleBatchGcs.empty') }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>

  <Teleport to="body">
    <Transition name="gbg-drawer-mask">
      <div
        v-if="profileDrawerOpen"
        class="fixed inset-0 z-[54] bg-black/40 backdrop-blur-sm"
        @click="closeProfileDrawer"
      ></div>
    </Transition>

    <Transition name="gbg-drawer-panel">
      <div
        v-if="profileDrawerOpen"
        class="fixed inset-y-0 right-0 z-[55] flex h-full w-full max-w-2xl flex-col border-l border-gray-200 bg-white shadow-2xl dark:border-dark-700 dark:bg-dark-900"
      >
        <div class="flex items-center justify-between border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <h4 class="text-sm font-semibold text-gray-900 dark:text-white">
            {{ creatingProfile ? t('admin.settings.googleBatchGcs.createTitle') : t('admin.settings.googleBatchGcs.editTitle') }}
          </h4>
          <button
            type="button"
            class="rounded p-1 text-gray-500 hover:bg-gray-100 hover:text-gray-700 dark:text-gray-400 dark:hover:bg-dark-800 dark:hover:text-gray-200"
            @click="closeProfileDrawer"
          >
            x
          </button>
        </div>

        <div class="flex-1 overflow-y-auto p-4">
          <div class="grid grid-cols-1 gap-3 md:grid-cols-2">
            <input
              v-model="profileForm.profile_id"
              class="input w-full"
              :placeholder="t('admin.settings.googleBatchGcs.profileID')"
              :disabled="!creatingProfile"
            />
            <input
              v-model="profileForm.name"
              class="input w-full"
              :placeholder="t('admin.settings.googleBatchGcs.profileName')"
            />
            <label class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 md:col-span-2">
              <input v-model="profileForm.enabled" type="checkbox" />
              <span>{{ t('admin.settings.googleBatchGcs.enabled') }}</span>
            </label>
            <div class="md:col-span-2">
              <p class="text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.googleBatchGcs.enabledHint') }}
              </p>
            </div>
            <input
              v-model="profileForm.bucket"
              class="input w-full"
              :placeholder="t('admin.settings.googleBatchGcs.bucket')"
            />
            <input
              v-model="profileForm.project_id"
              class="input w-full"
              :placeholder="t('admin.settings.googleBatchGcs.projectId')"
            />
            <div class="md:col-span-2">
              <input
                v-model="profileForm.prefix"
                class="input w-full"
                :placeholder="t('admin.settings.googleBatchGcs.prefix')"
              />
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.googleBatchGcs.prefixHint') }}
              </p>
            </div>
            <div class="md:col-span-2">
              <textarea
                v-model="profileForm.service_account_json"
                rows="12"
                class="input w-full font-mono text-xs"
                :placeholder="profileForm.service_account_json_configured ? t('admin.settings.googleBatchGcs.configuredPlaceholder') : t('admin.settings.googleBatchGcs.serviceAccountJson')"
              ></textarea>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                {{ t('admin.settings.googleBatchGcs.serviceAccountHint') }}
              </p>
            </div>
            <label
              v-if="creatingProfile"
              class="inline-flex items-center gap-2 text-sm text-gray-700 dark:text-gray-300 md:col-span-2"
            >
              <input v-model="profileForm.set_active" type="checkbox" />
              <span>{{ t('admin.settings.googleBatchGcs.setActive') }}</span>
            </label>
          </div>
        </div>

        <div class="flex flex-wrap justify-end gap-2 border-t border-gray-200 p-4 dark:border-dark-700">
          <button type="button" class="btn btn-secondary btn-sm" @click="closeProfileDrawer">
            {{ t('common.cancel') }}
          </button>
          <button
            type="button"
            class="btn btn-secondary btn-sm"
            :disabled="testingProfile || !profileForm.enabled"
            @click="testDraftProfile"
          >
            {{ testingProfile ? t('admin.settings.googleBatchGcs.testing') : t('admin.settings.googleBatchGcs.testConnection') }}
          </button>
          <button type="button" class="btn btn-primary btn-sm" :disabled="savingProfile" @click="saveProfile">
            {{ savingProfile ? t('common.loading') : t('admin.settings.googleBatchGcs.saveProfile') }}
          </button>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GoogleBatchGCSProfile } from '@/api/admin/settings'
import { adminAPI } from '@/api'
import { useAppStore } from '@/stores'

const { t } = useI18n()
const appStore = useAppStore()

const loadingProfiles = ref(false)
const savingProfile = ref(false)
const testingProfile = ref(false)
const activatingProfile = ref(false)
const deletingProfile = ref(false)
const creatingProfile = ref(false)
const profileDrawerOpen = ref(false)

const profiles = ref<GoogleBatchGCSProfile[]>([])
const selectedProfileID = ref('')

type GoogleBatchGCSProfileForm = {
  profile_id: string
  name: string
  set_active: boolean
  enabled: boolean
  bucket: string
  prefix: string
  project_id: string
  service_account_json: string
  service_account_json_configured: boolean
}

const profileForm = ref<GoogleBatchGCSProfileForm>(newDefaultProfileForm())

async function loadProfiles() {
  loadingProfiles.value = true
  try {
    const result = await adminAPI.settings.listGoogleBatchGCSProfiles()
    profiles.value = result.items || []
    if (!creatingProfile.value) {
      const stillExists = selectedProfileID.value
        ? profiles.value.some((item) => item.profile_id === selectedProfileID.value)
        : false
      if (!stillExists) {
        selectedProfileID.value = pickPreferredProfileID()
      }
      syncProfileFormWithSelection()
    }
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || t('errors.networkError'))
  } finally {
    loadingProfiles.value = false
  }
}

function startCreateProfile() {
  creatingProfile.value = true
  selectedProfileID.value = ''
  profileForm.value = newDefaultProfileForm()
  profileDrawerOpen.value = true
}

function editProfile(profileID: string) {
  selectedProfileID.value = profileID
  creatingProfile.value = false
  syncProfileFormWithSelection()
  profileDrawerOpen.value = true
}

function closeProfileDrawer() {
  profileDrawerOpen.value = false
  if (creatingProfile.value) {
    creatingProfile.value = false
    selectedProfileID.value = pickPreferredProfileID()
    syncProfileFormWithSelection()
  }
}

async function saveProfile() {
  if (!profileForm.value.name.trim()) {
    appStore.showError(t('admin.settings.googleBatchGcs.profileNameRequired'))
    return
  }
  if (creatingProfile.value && !profileForm.value.profile_id.trim()) {
    appStore.showError(t('admin.settings.googleBatchGcs.profileIDRequired'))
    return
  }
  if (!creatingProfile.value && !selectedProfileID.value) {
    appStore.showError(t('admin.settings.googleBatchGcs.profileSelectRequired'))
    return
  }
  if (profileForm.value.enabled) {
    if (!profileForm.value.bucket.trim()) {
      appStore.showError(t('admin.settings.googleBatchGcs.bucketRequired'))
      return
    }
    if (!profileForm.value.project_id.trim()) {
      appStore.showError(t('admin.settings.googleBatchGcs.projectIDRequired'))
      return
    }
    if (!profileForm.value.service_account_json.trim() && !profileForm.value.service_account_json_configured) {
      appStore.showError(t('admin.settings.googleBatchGcs.serviceAccountRequired'))
      return
    }
  }

  savingProfile.value = true
  try {
    if (creatingProfile.value) {
      const created = await adminAPI.settings.createGoogleBatchGCSProfile({
        profile_id: profileForm.value.profile_id.trim(),
        name: profileForm.value.name.trim(),
        set_active: profileForm.value.set_active,
        enabled: profileForm.value.enabled,
        bucket: profileForm.value.bucket.trim(),
        prefix: profileForm.value.prefix.trim(),
        project_id: profileForm.value.project_id.trim(),
        service_account_json: profileForm.value.service_account_json.trim() || undefined
      })
      selectedProfileID.value = created.profile_id
      creatingProfile.value = false
      profileDrawerOpen.value = false
      appStore.showSuccess(t('admin.settings.googleBatchGcs.profileCreated'))
    } else {
      await adminAPI.settings.updateGoogleBatchGCSProfile(selectedProfileID.value, {
        name: profileForm.value.name.trim(),
        enabled: profileForm.value.enabled,
        bucket: profileForm.value.bucket.trim(),
        prefix: profileForm.value.prefix.trim(),
        project_id: profileForm.value.project_id.trim(),
        service_account_json: profileForm.value.service_account_json.trim() || undefined
      })
      profileDrawerOpen.value = false
      appStore.showSuccess(t('admin.settings.googleBatchGcs.profileSaved'))
    }
    await loadProfiles()
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || t('admin.settings.googleBatchGcs.saveFailed'))
  } finally {
    savingProfile.value = false
  }
}

async function testDraftProfile() {
  testingProfile.value = true
  try {
    const result = await adminAPI.settings.testGoogleBatchGCSConnection({
      profile_id: creatingProfile.value ? undefined : selectedProfileID.value,
      enabled: profileForm.value.enabled,
      bucket: profileForm.value.bucket.trim(),
      prefix: profileForm.value.prefix.trim(),
      project_id: profileForm.value.project_id.trim(),
      service_account_json: profileForm.value.service_account_json.trim() || undefined
    })
    appStore.showSuccess(result.message || t('admin.settings.googleBatchGcs.testSuccess'))
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || t('admin.settings.googleBatchGcs.testFailed'))
  } finally {
    testingProfile.value = false
  }
}

async function testExistingProfile(profileID: string) {
  const profile = profiles.value.find((item) => item.profile_id === profileID)
  if (!profile) {
    return
  }
  testingProfile.value = true
  try {
    const result = await adminAPI.settings.testGoogleBatchGCSConnection({
      profile_id: profileID,
      enabled: profile.enabled,
      bucket: profile.bucket,
      prefix: profile.prefix,
      project_id: profile.project_id
    })
    appStore.showSuccess(result.message || t('admin.settings.googleBatchGcs.testSuccess'))
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || t('admin.settings.googleBatchGcs.testFailed'))
  } finally {
    testingProfile.value = false
  }
}

async function activateProfile(profileID: string) {
  activatingProfile.value = true
  try {
    await adminAPI.settings.setActiveGoogleBatchGCSProfile(profileID)
    appStore.showSuccess(t('admin.settings.googleBatchGcs.profileActivated'))
    await loadProfiles()
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || t('errors.networkError'))
  } finally {
    activatingProfile.value = false
  }
}

async function removeProfile(profileID: string) {
  if (!window.confirm(t('admin.settings.googleBatchGcs.deleteConfirm', { profileID }))) {
    return
  }
  deletingProfile.value = true
  try {
    await adminAPI.settings.deleteGoogleBatchGCSProfile(profileID)
    if (selectedProfileID.value === profileID) {
      selectedProfileID.value = ''
    }
    appStore.showSuccess(t('admin.settings.googleBatchGcs.profileDeleted'))
    await loadProfiles()
  } catch (error) {
    appStore.showError((error as { message?: string })?.message || t('errors.networkError'))
  } finally {
    deletingProfile.value = false
  }
}

function formatDate(value?: string): string {
  if (!value) {
    return '-'
  }
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) {
    return value
  }
  return date.toLocaleString()
}

function pickPreferredProfileID(): string {
  const active = profiles.value.find((item) => item.is_active)
  if (active) {
    return active.profile_id
  }
  return profiles.value[0]?.profile_id || ''
}

function syncProfileFormWithSelection() {
  const profile = profiles.value.find((item) => item.profile_id === selectedProfileID.value)
  profileForm.value = newDefaultProfileForm(profile)
}

function newDefaultProfileForm(profile?: GoogleBatchGCSProfile): GoogleBatchGCSProfileForm {
  if (!profile) {
    return {
      profile_id: '',
      name: '',
      set_active: false,
      enabled: false,
      bucket: '',
      prefix: 'gemini-batch/',
      project_id: '',
      service_account_json: '',
      service_account_json_configured: false
    }
  }

  return {
    profile_id: profile.profile_id,
    name: profile.name,
    set_active: false,
    enabled: profile.enabled,
    bucket: profile.bucket || '',
    prefix: profile.prefix || '',
    project_id: profile.project_id || '',
    service_account_json: '',
    service_account_json_configured: Boolean(profile.service_account_json_configured)
  }
}

onMounted(async () => {
  await loadProfiles()
})
</script>

<style scoped>
.gbg-drawer-mask-enter-active,
.gbg-drawer-mask-leave-active {
  transition: opacity 0.2s ease;
}

.gbg-drawer-mask-enter-from,
.gbg-drawer-mask-leave-to {
  opacity: 0;
}

.gbg-drawer-panel-enter-active,
.gbg-drawer-panel-leave-active {
  transition:
    transform 0.24s cubic-bezier(0.22, 1, 0.36, 1),
    opacity 0.2s ease;
}

.gbg-drawer-panel-enter-from,
.gbg-drawer-panel-leave-to {
  opacity: 0.96;
  transform: translateX(100%);
}

@media (prefers-reduced-motion: reduce) {
  .gbg-drawer-mask-enter-active,
  .gbg-drawer-mask-leave-active,
  .gbg-drawer-panel-enter-active,
  .gbg-drawer-panel-leave-active {
    transition-duration: 0s;
  }
}
</style>
