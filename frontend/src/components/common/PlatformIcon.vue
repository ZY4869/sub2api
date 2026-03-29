<template>
  <LobeStaticIcon
    :sources="sources"
    :badge-text="badgeText"
    :size="sizePx"
    variant="platform"
    :alt="badgeText"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { AccountPlatform, GroupPlatform } from '@/types'
import {
  getPlatformBadgeText,
  getPlatformIconSources
} from '@/utils/platformBranding'
import LobeStaticIcon from '@/components/common/LobeStaticIcon.vue'

interface Props {
  platform?: AccountPlatform | GroupPlatform
  size?: 'xs' | 'sm' | 'md' | 'lg'
}

const props = withDefaults(defineProps<Props>(), {
  size: 'sm'
})

const sizePx = computed(() => ({
  xs: '12px',
  sm: '14px',
  md: '16px',
  lg: '20px'
}[props.size]))

const sources = computed(() => getPlatformIconSources(props.platform))
const badgeText = computed(() => getPlatformBadgeText(props.platform))
</script>
