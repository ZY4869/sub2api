<template>
  <LobeStaticIcon
    :sources="sources"
    :badge-text="badgeText"
    :size="pixelSize"
    :alt="platform || badgeText"
    variant="platform"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import LobeStaticIcon from '@/components/common/LobeStaticIcon.vue'
import {
  buildLobeIconSources,
  resolveLobeBadgeText,
  resolveProviderIconSlugs
} from '@/utils/lobeIconResolver'

interface Props {
  platform?: string
  size?: 'xs' | 'sm' | 'md' | 'lg'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'sm'
})

const pixelSize = computed(() => {
  const sizes = {
    xs: '12px',
    sm: '14px',
    md: '16px',
    lg: '20px'
  }
  return sizes[props.size] || sizes.sm
})

const sources = computed(() => buildLobeIconSources(resolveProviderIconSlugs(props.platform)))
const badgeText = computed(() => resolveLobeBadgeText(props.platform))
</script>
