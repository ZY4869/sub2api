import { computed, ref, watch, type Ref } from 'vue'
import { getModelCatalogDetail, type PublicModelCatalogDetailResponse } from '@/api/meta'
import keysAPI from '@/api/keys'
import userGroupsAPI from '@/api/groups'
import type { ApiKey, UserGroupModelOptionGroup } from '@/types'
import { buildPublicModelExample } from '@/utils/publicModelCatalogExamples'
import { findSupportedKeysForModel } from '@/utils/publicModelCatalogKeys'

interface UsePublicModelDetailOptions {
  show: Ref<boolean>
  model: Ref<string | null>
  isAuthenticated: Ref<boolean>
  baseUrl: Ref<string>
  missingKey: string
  resolveErrorMessage: (error: unknown) => string
}

export function usePublicModelDetail(options: UsePublicModelDetailOptions) {
  const detail = ref<PublicModelCatalogDetailResponse | null>(null)
  const loading = ref(false)
  const errorMessage = ref('')
  const userKeys = ref<ApiKey[]>([])
  const userGroupOptions = ref<UserGroupModelOptionGroup[]>([])
  const selectedKeyID = ref<number | null>(null)
  let requestToken = 0

  const supportedKeys = computed(() => findSupportedKeysForModel(userKeys.value, userGroupOptions.value, detail.value))
  const selectedKey = computed(() => supportedKeys.value.find((item) => item.id === selectedKeyID.value) || supportedKeys.value[0] || null)
  const effectiveAPIKey = computed(() => selectedKey.value?.key || options.missingKey)
  const exampleResult = computed(() => buildPublicModelExample(detail.value, effectiveAPIKey.value, options.baseUrl.value))

  watch(
    () => [options.show.value, options.model.value, options.isAuthenticated.value] as const,
    async ([show, model]) => {
      if (!show || !model) return
      const currentToken = ++requestToken
      loading.value = true
      errorMessage.value = ''
      try {
        const nextDetail = await getModelCatalogDetail(model)
        if (currentToken !== requestToken) return
        detail.value = nextDetail
        if (options.isAuthenticated.value) {
          await loadUserContext(currentToken)
        } else {
          userKeys.value = []
          userGroupOptions.value = []
        }
      } catch (error) {
        if (currentToken !== requestToken) return
        detail.value = null
        errorMessage.value = options.resolveErrorMessage(error)
      } finally {
        if (currentToken === requestToken) loading.value = false
      }
    },
    { immediate: true },
  )

  watch(
    supportedKeys,
    (items) => {
      if (items.length === 0) {
        selectedKeyID.value = null
        return
      }
      if (!items.some((item) => item.id === selectedKeyID.value)) {
        selectedKeyID.value = items[0].id
      }
    },
    { immediate: true },
  )

  async function loadUserContext(currentToken: number) {
    try {
      const [keysResponse, groupOptions] = await Promise.all([
        keysAPI.list(1, 1000),
        userGroupsAPI.getModelOptions(),
      ])
      if (currentToken !== requestToken) return
      userKeys.value = keysResponse.items || []
      userGroupOptions.value = groupOptions || []
    } catch {
      if (currentToken !== requestToken) return
      userKeys.value = []
      userGroupOptions.value = []
    }
  }

  return {
    detail,
    loading,
    errorMessage,
    selectedKey,
    selectedKeyID,
    supportedKeys,
    exampleResult,
  }
}
