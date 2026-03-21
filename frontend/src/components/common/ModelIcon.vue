<template>
  <LobeStaticIcon
    :sources="sources"
    :badge-text="badgeText"
    :size="size"
    :alt="displayName || model || badgeText"
    variant="model"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import LobeStaticIcon from '@/components/common/LobeStaticIcon.vue'
import {
  buildLobeIconSources,
  resolveLobeBadgeText,
  resolveModelIconSlugs
} from '@/utils/lobeIconResolver'

const props = withDefaults(defineProps<{
  model: string
  provider?: string
  iconKey?: string
  displayName?: string
  size?: string
}>(), {
  provider: '',
  iconKey: '',
  displayName: '',
  size: '18px'
})

const sources = computed(() => buildLobeIconSources(resolveModelIconSlugs({
  model: props.model,
  provider: props.provider,
  iconKey: props.iconKey,
  displayName: props.displayName
})))

const badgeText = computed(() => resolveLobeBadgeText(props.displayName, props.model, props.provider))
</script>
