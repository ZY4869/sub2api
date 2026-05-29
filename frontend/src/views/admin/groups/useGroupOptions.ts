import { computed, type Ref } from 'vue'
import type { AdminGroup, GroupPlatform, OpenAIGroupImageProtocolMode } from '@/types'
import { GROUP_PLATFORM_ORDER } from '@/utils/platformBranding'
export interface PlatformSelectOption {
  value: GroupPlatform | ''
  label: string
  platform?: GroupPlatform
  [key: string]: unknown
}

export interface GroupSelectOption {
  value: number | null
  label: string
  name: string
  platform?: GroupPlatform
  description?: string
  [key: string]: unknown
}

export function normalizeOpenAIGroupImageProtocolMode(value: unknown): OpenAIGroupImageProtocolMode {
  const normalized = String(value || '').trim().toLowerCase()
  if (normalized === 'native' || normalized === 'compat') {
    return normalized
  }
  return 'inherit'
}

function buildPlatformSelectOption(
  t: (key: string) => string,
  platform: GroupPlatform
): PlatformSelectOption {
  return {
    value: platform,
    label: t(`admin.groups.platforms.${platform}`),
    platform
  }
}

export function buildGroupSelectOption(
  group: AdminGroup,
  description?: string
): GroupSelectOption {
  return {
    value: group.id,
    label: group.name,
    name: group.name,
    platform: group.platform,
    description
  }
}

function buildEmptyGroupSelectOption(label: string): GroupSelectOption {
  return {
    value: null,
    label,
    name: label
  }
}

export function useGroupOptions(
  t: (key: string) => string,
  groups: Ref<AdminGroup[]>,
  editingGroup: Ref<AdminGroup | null>
) {
  const statusOptions = computed(() => [
    { value: '', label: t('admin.groups.allStatus') },
    { value: 'active', label: t('admin.accounts.status.active') },
    { value: 'inactive', label: t('admin.accounts.status.inactive') }
  ])

  const exclusiveOptions = computed(() => [
    { value: '', label: t('admin.groups.allGroups') },
    { value: 'true', label: t('admin.groups.exclusive') },
    { value: 'false', label: t('admin.groups.nonExclusive') }
  ])

  const platformOptions = computed<PlatformSelectOption[]>(() =>
    GROUP_PLATFORM_ORDER.map((platform) => buildPlatformSelectOption(t, platform))
  )

  const platformFilterOptions = computed<PlatformSelectOption[]>(() => [
    { value: '', label: t('admin.groups.allPlatforms') },
    ...platformOptions.value
  ])

  const editStatusOptions = computed(() => [
    { value: 'active', label: t('admin.accounts.status.active') },
    { value: 'inactive', label: t('admin.accounts.status.inactive') }
  ])

  const subscriptionTypeOptions = computed(() => [
    { value: 'standard', label: t('admin.groups.subscription.standard') },
    { value: 'subscription', label: t('admin.groups.subscription.subscription') }
  ])

  const openAIGroupImageProtocolModeOptions = computed(() => [
    { value: 'inherit', label: t('admin.groups.imageProtocol.options.inherit') },
    { value: 'native', label: t('admin.groups.imageProtocol.options.native') },
    { value: 'compat', label: t('admin.groups.imageProtocol.options.compat') }
  ])

  const fallbackGroupOptions = computed<GroupSelectOption[]>(() => {
    const options: GroupSelectOption[] = [
      buildEmptyGroupSelectOption(t('admin.groups.claudeCode.noFallback'))
    ]
    const eligibleGroups = groups.value.filter(
      (g) => g.platform === 'anthropic' && !g.claude_code_only && g.status === 'active'
    )
    eligibleGroups.forEach((g) => {
      options.push(buildGroupSelectOption(g))
    })
    return options
  })

  const fallbackGroupOptionsForEdit = computed<GroupSelectOption[]>(() => {
    const options: GroupSelectOption[] = [
      buildEmptyGroupSelectOption(t('admin.groups.claudeCode.noFallback'))
    ]
    const currentId = editingGroup.value?.id
    const eligibleGroups = groups.value.filter(
      (g) => g.platform === 'anthropic' && !g.claude_code_only && g.status === 'active' && g.id !== currentId
    )
    eligibleGroups.forEach((g) => {
      options.push(buildGroupSelectOption(g))
    })
    return options
  })

  const invalidRequestFallbackOptions = computed<GroupSelectOption[]>(() => {
    const options: GroupSelectOption[] = [
      buildEmptyGroupSelectOption(t('admin.groups.invalidRequestFallback.noFallback'))
    ]
    const eligibleGroups = groups.value.filter(
      (g) =>
        g.platform === 'anthropic' &&
        g.status === 'active' &&
        g.subscription_type !== 'subscription' &&
        g.fallback_group_id_on_invalid_request === null
    )
    eligibleGroups.forEach((g) => {
      options.push(buildGroupSelectOption(g))
    })
    return options
  })

  const invalidRequestFallbackOptionsForEdit = computed<GroupSelectOption[]>(() => {
    const options: GroupSelectOption[] = [
      buildEmptyGroupSelectOption(t('admin.groups.invalidRequestFallback.noFallback'))
    ]
    const currentId = editingGroup.value?.id
    const eligibleGroups = groups.value.filter(
      (g) =>
        g.platform === 'anthropic' &&
        g.status === 'active' &&
        g.subscription_type !== 'subscription' &&
        g.fallback_group_id_on_invalid_request === null &&
        g.id !== currentId
    )
    eligibleGroups.forEach((g) => {
      options.push(buildGroupSelectOption(g))
    })
    return options
  })

  return {
    statusOptions,
    exclusiveOptions,
    platformOptions,
    platformFilterOptions,
    editStatusOptions,
    subscriptionTypeOptions,
    openAIGroupImageProtocolModeOptions,
    fallbackGroupOptions,
    fallbackGroupOptionsForEdit,
    invalidRequestFallbackOptions,
    invalidRequestFallbackOptionsForEdit
  }
}
