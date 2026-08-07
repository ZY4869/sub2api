<template>
  <div v-if="enabled" class="card p-6">
    <div class="mb-5 flex items-center justify-between gap-4">
      <div>
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">Passkey</h3>
        <p class="mt-1 text-sm text-gray-500 dark:text-dark-400">使用设备生物识别或安全密钥登录账号。</p>
      </div>
      <button type="button" class="btn btn-secondary btn-sm" :disabled="loading" @click="load">
        <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
      </button>
    </div>

    <form class="mb-5 grid gap-3 sm:grid-cols-[1fr_1fr_auto]" @submit.prevent="handleRegister">
      <input
        v-model.trim="newName"
        class="input"
        type="text"
        maxlength="80"
        placeholder="Passkey 名称"
        :disabled="saving"
      />
      <input
        v-model="password"
        class="input"
        type="password"
        autocomplete="current-password"
        placeholder="当前密码"
        :disabled="saving"
      />
      <button class="btn btn-primary" type="submit" :disabled="saving || !password">
        <Icon name="plus" size="sm" class="mr-2" />
        {{ saving ? '添加中' : '添加' }}
      </button>
    </form>

    <div v-if="items.length === 0" class="rounded-lg border border-dashed border-gray-200 p-4 text-sm text-gray-500 dark:border-dark-700 dark:text-dark-400">
      当前没有已注册的 Passkey。
    </div>
    <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
      <div v-for="item in items" :key="item.id" class="flex items-center justify-between gap-4 py-3">
        <div class="min-w-0">
          <div class="truncate font-medium text-gray-900 dark:text-white">{{ item.name || 'Passkey' }}</div>
          <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">
            {{ item.last_used_at ? `最近使用：${formatDate(item.last_used_at)}` : `创建于：${formatDate(item.created_at)}` }}
          </div>
        </div>
        <div class="flex items-center gap-2">
          <button type="button" class="btn btn-secondary btn-sm" :disabled="saving" @click="handleRename(item)">
            <Icon name="edit" size="sm" />
          </button>
          <button type="button" class="btn btn-danger btn-sm" :disabled="saving" @click="handleDelete(item)">
            <Icon name="trash" size="sm" />
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref, watch } from 'vue'
import { passkeyAPI, type PasskeyCredential } from '@/api/passkeys'
import { useAppStore } from '@/stores'
import { formatDate } from '@/utils/format'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{
  enabled: boolean
}>()

const appStore = useAppStore()
const items = ref<PasskeyCredential[]>([])
const loading = ref(false)
const saving = ref(false)
const newName = ref('')
const password = ref('')

async function load(): Promise<void> {
  loading.value = true
  try {
    items.value = await passkeyAPI.listPasskeys()
  } catch (error) {
    console.error('Failed to load passkeys:', error)
    items.value = []
  } finally {
    loading.value = false
  }
}

async function handleRegister(): Promise<void> {
  saving.value = true
  try {
    await passkeyAPI.registerPasskey({
      password: password.value,
      name: newName.value || 'Passkey'
    })
    newName.value = ''
    password.value = ''
    appStore.showSuccess('Passkey 已添加')
    await load()
  } catch (error: unknown) {
    const err = error as { message?: string }
    appStore.showError(err.message || '添加 Passkey 失败')
  } finally {
    saving.value = false
  }
}

async function handleRename(item: PasskeyCredential): Promise<void> {
  const name = window.prompt('新的 Passkey 名称', item.name || 'Passkey')
  if (!name?.trim()) return
  saving.value = true
  try {
    await passkeyAPI.renamePasskey(item.id, name.trim())
    appStore.showSuccess('Passkey 已重命名')
    await load()
  } catch (error: unknown) {
    const err = error as { message?: string }
    appStore.showError(err.message || '重命名 Passkey 失败')
  } finally {
    saving.value = false
  }
}

async function handleDelete(item: PasskeyCredential): Promise<void> {
  const passwordInput = window.prompt(`删除 ${item.name || 'Passkey'} 需要输入当前密码`)
  if (!passwordInput) return
  saving.value = true
  try {
    await passkeyAPI.deletePasskey(item.id, passwordInput)
    appStore.showSuccess('Passkey 已删除')
    await load()
  } catch (error: unknown) {
    const err = error as { message?: string }
    appStore.showError(err.message || '删除 Passkey 失败')
  } finally {
    saving.value = false
  }
}

watch(() => props.enabled, (enabled) => {
  if (enabled) {
    void load()
    return
  }
  items.value = []
})

onMounted(() => {
  if (props.enabled) {
    void load()
  }
})
</script>
