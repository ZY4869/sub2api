<template>
  <ConfirmDialog
    :show="show"
    :title="t('admin.accounts.mixedChannelWarningTitle')"
    :message="message"
    :confirm-text="t('common.confirm')"
    :cancel-text="t('common.cancel')"
    :danger="true"
    @confirm="emit('confirm')"
    @cancel="emit('cancel')"
  />
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'

interface Props {
  show: boolean
  message: string
}

defineProps<Props>()
const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const { t } = useI18n()
</script>
