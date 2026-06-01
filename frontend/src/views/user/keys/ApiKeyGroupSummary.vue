<script setup lang="ts">
import { computed } from "vue";
import { useI18n } from "vue-i18n";
import GroupBadge from "@/components/common/GroupBadge.vue";
import type { ApiKey, ApiKeyGroup, Group } from "@/types";

const props = defineProps<{
  apiKey: ApiKey;
  bindings: ApiKeyGroup[];
  userGroupRates: Record<number, number>;
  isAdminMode: boolean;
  resolveGroup: (groupId: number | null | undefined) => Group | undefined;
}>();

defineEmits<{
  edit: [key: ApiKey];
}>();

const { t } = useI18n();

const bindingMeta = (binding: ApiKeyGroup) => {
  const group = props.resolveGroup(binding.group_id);
  return {
    id: binding.group_id,
    name: group?.name || binding.group_name || `#${binding.group_id}`,
    platform: group?.platform || binding.platform,
    subscriptionType: group?.subscription_type,
    rateMultiplier: group?.rate_multiplier,
    userRateMultiplier: group ? props.userGroupRates[binding.group_id] : undefined,
    priority: binding.priority ?? group?.priority ?? 1,
  };
};

const visibleBindings = computed(() =>
  props.bindings.slice(0, 2).map((binding) => bindingMeta(binding)),
);
const hiddenCount = computed(() => Math.max(props.bindings.length - visibleBindings.value.length, 0));

const tooltipText = computed(() =>
  props.bindings
    .map((binding) => {
      const meta = bindingMeta(binding);
      const rate = meta.userRateMultiplier ?? meta.rateMultiplier;
      const parts = [
        meta.name,
        meta.platform ? `${t("keys.groupPlatform")}: ${meta.platform}` : "",
        `${t("keys.groupPriority")}: P${meta.priority}`,
        rate !== undefined ? `${t("keys.groupRate")}: ${rate}x` : "",
      ].filter(Boolean);
      return parts.join(" · ");
    })
    .join("\n"),
);
</script>

<template>
  <div v-if="bindings.length" class="max-w-[16rem] space-y-1">
    <div
      class="flex flex-wrap gap-1.5 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:ring-offset-2 dark:focus:ring-offset-dark-800"
      tabindex="0"
      :title="tooltipText"
      :aria-label="tooltipText"
    >
      <div
        v-for="binding in visibleBindings"
        :key="`${apiKey.id}-${binding.id}`"
        class="flex min-w-0 items-center gap-1.5"
      >
        <GroupBadge
          class="max-w-[8rem]"
          :name="binding.name"
          :platform="binding.platform"
          :subscription-type="binding.subscriptionType"
          :rate-multiplier="binding.rateMultiplier"
          :user-rate-multiplier="binding.userRateMultiplier"
        />
        <span class="rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-500 dark:bg-dark-700 dark:text-gray-300">
          P{{ binding.priority }}
        </span>
      </div>
      <span
        v-if="hiddenCount > 0"
        class="rounded-full bg-gray-100 px-2 py-0.5 text-[11px] text-gray-500 dark:bg-dark-700 dark:text-gray-300"
      >
        +{{ hiddenCount }}
      </span>
    </div>
    <button
      type="button"
      class="text-xs text-primary-600 transition-colors hover:text-primary-500 dark:text-primary-400"
      @click="$emit('edit', apiKey)"
    >
      {{ isAdminMode ? t("admin.users.editGroupBindings") : t("keys.editKey") }}
    </button>
  </div>
  <button
    v-else
    type="button"
    class="text-sm text-gray-400 transition-colors hover:text-primary-500 dark:text-dark-500"
    @click="$emit('edit', apiKey)"
  >
    {{ t("keys.noGroup") }}
  </button>
</template>
