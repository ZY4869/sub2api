<template>
  <BaseDialog
    :show="show"
    :title="t('keys.ccsClientSelect.title')"
    width="narrow"
    @close="$emit('close')"
  >
    <div class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-gray-400">
        {{ t("keys.ccsClientSelect.description") }}
      </p>
      <div class="grid grid-cols-2 gap-3">
        <button
          @click="$emit('select', 'claude')"
          class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
        >
          <Icon name="terminal" size="xl" class="text-gray-600 dark:text-gray-400" />
          <span class="font-medium text-gray-900 dark:text-white">{{ t("keys.ccsClientSelect.claudeCode") }}</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t("keys.ccsClientSelect.claudeCodeDesc") }}</span>
        </button>
        <button
          @click="$emit('select', 'gemini')"
          class="flex flex-col items-center gap-2 p-4 rounded-xl border-2 border-gray-200 dark:border-dark-600 hover:border-primary-500 dark:hover:border-primary-500 hover:bg-primary-50 dark:hover:bg-primary-900/20 transition-all"
        >
          <Icon name="sparkles" size="xl" class="text-gray-600 dark:text-gray-400" />
          <span class="font-medium text-gray-900 dark:text-white">{{ t("keys.ccsClientSelect.geminiCli") }}</span>
          <span class="text-xs text-gray-500 dark:text-gray-400">{{ t("keys.ccsClientSelect.geminiCliDesc") }}</span>
        </button>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end">
        <button @click="$emit('close')" class="btn btn-secondary">
          {{ t("common.cancel") }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from "vue-i18n";
import BaseDialog from "@/components/common/BaseDialog.vue";
import Icon from "@/components/icons/Icon.vue";

defineProps<{
  show: boolean;
}>();

defineEmits<{
  close: [];
  select: [clientType: "claude" | "gemini"];
}>();

const { t } = useI18n();
</script>
