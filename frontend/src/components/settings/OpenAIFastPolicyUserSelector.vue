<template>
  <div>
    <label class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400">
      {{ t('admin.settings.openaiFastPolicy.userIds') }}
    </label>

    <div v-if="normalizedUserIDs.length > 0" class="mb-2 flex flex-wrap gap-1.5">
      <span
        v-for="id in normalizedUserIDs"
        :key="id"
        class="inline-flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
      >
        {{ selectedUserLabels[id] || `#${id}` }}
        <button
          type="button"
          class="ml-0.5 text-primary-500 hover:text-primary-700 dark:hover:text-primary-200"
          :aria-label="t('admin.settings.openaiFastPolicy.removeUser')"
          @click="removeUserID(id)"
        >
          <span aria-hidden="true">x</span>
        </button>
      </span>
    </div>

    <div ref="searchRef" class="relative">
      <input
        v-model="searchKeyword"
        type="text"
        class="input text-xs"
        :placeholder="t('admin.settings.openaiFastPolicy.userSearchPlaceholder')"
        @input="debounceSearch"
        @focus="showDropdown = true"
      />
      <div
        v-if="showDropdown && (searchResults.length > 0 || searchKeyword.trim())"
        class="absolute z-50 mt-1 max-h-48 w-full overflow-auto rounded-lg border bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800"
      >
        <button
          v-for="user in searchResults"
          :key="user.id"
          type="button"
          class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 disabled:cursor-not-allowed disabled:opacity-50 dark:hover:bg-dark-700"
          :disabled="normalizedUserIDs.includes(user.id)"
          @click="selectUser(user)"
        >
          <span>{{ formatUserLabel(user) }}</span>
          <span class="ml-2 text-xs text-gray-400">#{{ user.id }}</span>
        </button>
        <div v-if="searchResults.length === 0" class="px-3 py-2 text-xs text-gray-400">
          {{ t('admin.settings.openaiFastPolicy.userSearchEmpty') }}
        </div>
      </div>
    </div>

    <textarea
      class="input mt-2 min-h-[72px] font-mono text-xs"
      :placeholder="t('admin.settings.openaiFastPolicy.userIdsPlaceholder')"
      :value="manualText"
      @input="handleManualInput"
    ></textarea>
    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
      {{ t('admin.settings.openaiFastPolicy.userIdsHint') }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { SimpleUser } from '@/api/admin/usage'

const props = defineProps<{
  userIds?: number[]
}>()

const emit = defineEmits<{
  'update:userIds': [value: number[]]
}>()

const { t } = useI18n()

const searchRef = ref<HTMLElement | null>(null)
const searchKeyword = ref('')
const searchResults = ref<SimpleUser[]>([])
const showDropdown = ref(false)
const selectedUserLabels = ref<Record<number, string>>({})
let searchTimeout: ReturnType<typeof setTimeout> | null = null

const normalizeUserIDs = (userIDs?: number[]) => {
  const seen = new Set<number>()
  const out: number[] = []
  for (const raw of Array.isArray(userIDs) ? userIDs : []) {
    const id = Number(raw)
    if (!Number.isInteger(id) || id <= 0 || seen.has(id)) continue
    seen.add(id)
    out.push(id)
  }
  return out
}

const normalizedUserIDs = computed(() => normalizeUserIDs(props.userIds))
const manualText = computed(() => normalizedUserIDs.value.join('\n'))

const parseUserIDs = (raw: string) => {
  return normalizeUserIDs(
    raw
      .split(/[\n,]/g)
      .map((item) => Number.parseInt(item.trim(), 10))
  )
}

const formatUserLabel = (user: SimpleUser) => {
  const base = user.email || `#${user.id}`
  return user.deleted ? `${base} (${t('admin.usage.deletedUserSuffix')})` : base
}

const emitUserIDs = (userIDs: number[]) => {
  emit('update:userIds', normalizeUserIDs(userIDs))
}

const selectUser = (user: SimpleUser) => {
  selectedUserLabels.value[user.id] = formatUserLabel(user)
  emitUserIDs([...normalizedUserIDs.value, user.id])
  searchKeyword.value = ''
  searchResults.value = []
  showDropdown.value = false
}

const removeUserID = (userID: number) => {
  emitUserIDs(normalizedUserIDs.value.filter((id) => id !== userID))
}

const handleManualInput = (event: Event) => {
  const el = event.target as HTMLTextAreaElement | null
  if (!el) return
  emitUserIDs(parseUserIDs(el.value || ''))
}

const debounceSearch = () => {
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
  searchTimeout = setTimeout(searchUsers, 300)
}

const searchUsers = async () => {
  const keyword = searchKeyword.value.trim()
  if (!keyword) {
    searchResults.value = []
    return
  }
  try {
    searchResults.value = await adminAPI.usage.searchUsers(keyword)
  } catch {
    searchResults.value = []
  }
}

const handleDocumentClick = (event: MouseEvent) => {
  const target = event.target as Node | null
  if (!target || !searchRef.value?.contains(target)) {
    showDropdown.value = false
  }
}

watch(
  () => props.userIds,
  (userIDs) => {
    for (const id of normalizeUserIDs(userIDs)) {
      if (!selectedUserLabels.value[id]) {
        selectedUserLabels.value[id] = `#${id}`
      }
    }
  },
  { immediate: true, deep: true }
)

onMounted(() => {
  document.addEventListener('click', handleDocumentClick)
})

onUnmounted(() => {
  document.removeEventListener('click', handleDocumentClick)
  if (searchTimeout) {
    clearTimeout(searchTimeout)
  }
})
</script>
