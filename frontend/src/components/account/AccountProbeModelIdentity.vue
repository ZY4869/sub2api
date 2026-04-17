<template>
  <div class="min-w-0 space-y-2">
    <div
      v-if="provider"
      class="flex items-center gap-1.5 text-gray-400 dark:text-gray-500"
      data-testid="probe-identity-icons"
    >
      <ModelPlatformIcon
        :platform="provider"
        size="sm"
        data-testid="probe-provider-icon"
      />
      <span class="sr-only">{{ providerText }}</span>
    </div>

    <div class="min-w-0 space-y-1">
      <div
        class="flex min-w-0 items-start gap-2 text-sm font-semibold text-gray-900 dark:text-gray-100"
        :title="resolvedDisplayName"
        data-testid="probe-model-display-name"
      >
        <ModelIcon
          :model="modelId"
          :provider="provider"
          :display-name="resolvedDisplayName"
          size="16px"
          data-testid="probe-model-icon"
        />
        <span class="min-w-0 break-words">{{ resolvedDisplayName }}</span>
      </div>

      <div
        class="flex min-w-0 items-start gap-2 text-xs text-gray-500 dark:text-gray-400"
        :title="modelId"
        data-testid="probe-model-id"
      >
        <ModelIcon
          :model="modelId"
          :provider="provider"
          :display-name="resolvedDisplayName"
          size="12px"
        />
        <span class="min-w-0 break-words">{{ modelId }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import ModelPlatformIcon from '@/components/common/ModelPlatformIcon.vue'
import { formatModelDisplayName } from '@/utils/modelDisplayName'

const props = defineProps<{
  modelId: string
  displayName?: string
  provider?: string
  providerText?: string
}>()

const resolvedDisplayName = computed(
  () => props.displayName?.trim() || formatModelDisplayName(props.modelId) || props.modelId
)
const provider = computed(() => String(props.provider || '').trim())
const providerText = computed(
  () => String(props.providerText || props.provider || '').trim()
)
</script>
