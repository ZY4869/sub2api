<template>
  <Select
    :model-value="modelValue"
    :options="options"
    :placeholder="t('keys.selectGroup')"
    :search-placeholder="t('keys.searchGroup')"
    :empty-text="t('keys.noGroupFound')"
    :aria-label="t('keys.groupLabel')"
    searchable
    @update:model-value="emit('update:modelValue', normalizeGroupId($event))"
  >
    <template #selected="{ option }">
      <ApiKeyGroupPill
        v-if="selectedGroup(option)"
        class="max-w-full"
        :name="selectedGroup(option)!.name"
        :platform="selectedGroup(option)!.platform"
        :subscription-type="selectedGroup(option)!.subscription_type"
        :rate-multiplier="selectedGroup(option)!.rate_multiplier"
        :user-rate-multiplier="selectedUserRateMultiplier(option)"
      />
      <span v-else class="text-gray-400 dark:text-dark-400">
        {{ t("keys.selectGroup") }}
      </span>
    </template>

    <template #option="{ option, selected }">
      <div v-if="selectedGroup(option)" class="flex min-w-0 flex-1 items-center justify-between gap-3">
        <ApiKeyGroupPill
          class="min-w-0 flex-1"
          :name="selectedGroup(option)!.name"
          :platform="selectedGroup(option)!.platform"
          :subscription-type="selectedGroup(option)!.subscription_type"
          :rate-multiplier="selectedGroup(option)!.rate_multiplier"
          :user-rate-multiplier="selectedUserRateMultiplier(option)"
        />
        <Icon
          v-if="selected"
          name="check"
          size="sm"
          class="flex-shrink-0 text-primary-500"
          :stroke-width="2"
        />
      </div>
      <span v-else class="select-option-label">{{ option.label }}</span>
    </template>
  </Select>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import Select from "@/components/common/Select.vue";
import Icon from "@/components/icons/Icon.vue";
import type { BindableGroup } from "./apiKeyGroupBindings";
import ApiKeyGroupPill from "./ApiKeyGroupPill.vue";

interface GroupSelectOption extends Record<string, unknown> {
  value: number;
  label: string;
  description?: string;
  disabled?: boolean;
  group?: BindableGroup;
  userRateMultiplier?: number;
}

const props = withDefaults(
  defineProps<{
    modelValue: number;
    groups: BindableGroup[];
    disabledGroupIds?: number[];
    userGroupRates?: Record<number, number>;
  }>(),
  {
    disabledGroupIds: () => [],
    userGroupRates: () => ({}),
  },
);

const emit = defineEmits<{
  (e: "update:modelValue", value: number): void;
}>();

const { t } = useI18n();

const disabledGroupIds = computed(() => new Set(props.disabledGroupIds));

const options = computed<GroupSelectOption[]>(() => [
  {
    value: 0,
    label: t("keys.selectGroup"),
    searchText: t("keys.selectGroup"),
  },
  ...props.groups.map((group) => {
    const userRateMultiplier = props.userGroupRates[group.id];
    const effectiveRate = userRateMultiplier ?? group.rate_multiplier;
    return {
      value: group.id,
      label: `${group.name} ${effectiveRate}x`,
      description: `${group.name} ${group.platform} ${effectiveRate}x`,
      disabled: disabledGroupIds.value.has(group.id),
      group,
      userRateMultiplier,
    };
  }),
]);

const selectedGroup = (option: unknown): BindableGroup | undefined => {
  return (option as GroupSelectOption | null | undefined)?.group;
};

const selectedUserRateMultiplier = (option: unknown): number | undefined => {
  return (option as GroupSelectOption | null | undefined)?.userRateMultiplier;
};

const normalizeGroupId = (value: string | number | boolean | null): number => {
  const parsed = Number(value);
  return Number.isFinite(parsed) ? parsed : 0;
};
</script>
