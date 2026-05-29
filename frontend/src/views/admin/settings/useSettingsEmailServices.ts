import { computed, reactive, ref } from 'vue'
import { adminAPI } from '@/api'
import type { EmailTemplate, EmailTemplateDefinition } from '@/api/admin/settings'

type Translate = (key: string) => string
type MessageStore = {
  showError: (message: string) => void
  showSuccess: (message: string) => void
}

type EmailSettingsForm = {
  smtp_host: string
  smtp_port: number
  smtp_username: string
  smtp_password: string
  smtp_from_email: string
  smtp_from_name: string
  smtp_use_tls: boolean
  telegram_bot_token: string
  telegram_chat_id: string
}

export function useSettingsEmailServices(
  t: Translate,
  appStore: MessageStore,
  form: EmailSettingsForm
) {
  const testingSmtp = ref(false)
  const testingTelegram = ref(false)
  const sendingTestEmail = ref(false)
  const testEmailAddress = ref('')
  const emailTemplatesLoading = ref(false)
  const emailTemplateSaving = ref(false)
  const emailTemplateTesting = ref(false)
  const emailTemplateDefinitions = ref<EmailTemplateDefinition[]>([])
  const selectedEmailTemplateKey = ref('')
  const selectedEmailTemplateLocale = ref('zh')
  const emailTemplateTestAddress = ref('')
  const emailTemplateDraft = reactive({
    subject: '',
    body: '',
    enabled: true
  })

  const emailTemplateOptions = computed(() =>
    emailTemplateDefinitions.value.map((item) => ({
      value: item.key,
      label: t(`admin.settings.emailTemplates.names.${item.key}`)
    }))
  )

  const emailTemplateLocaleOptions = computed(() => [
    { value: 'zh', label: t('admin.settings.emailTemplates.localeZh') },
    { value: 'en', label: t('admin.settings.emailTemplates.localeEn') }
  ])

  const selectedEmailTemplate = computed(
    () => emailTemplateDefinitions.value.find((item) => item.key === selectedEmailTemplateKey.value) || null
  )

  const formatEmailTemplateVariable = (name: string) => `{${`{.${name}}`}}`

  function emailTemplateBuiltIn(
    def: EmailTemplateDefinition | null,
    locale: string
  ): EmailTemplate | null {
    if (!def) return null
    const builtIn = def.built_in || def.BuiltIn || {}
    return builtIn[locale] || builtIn.en || builtIn.zh || null
  }

  function syncEmailTemplateDraft() {
    const tmpl = emailTemplateBuiltIn(selectedEmailTemplate.value, selectedEmailTemplateLocale.value)
    emailTemplateDraft.subject = tmpl?.subject || ''
    emailTemplateDraft.body = tmpl?.body || ''
    emailTemplateDraft.enabled = tmpl?.enabled !== false
  }

  async function testSmtpConnection() {
    testingSmtp.value = true
    try {
      const result = await adminAPI.settings.testSmtpConnection({
        smtp_host: form.smtp_host,
        smtp_port: form.smtp_port,
        smtp_username: form.smtp_username,
        smtp_password: form.smtp_password,
        smtp_use_tls: form.smtp_use_tls
      })
      appStore.showSuccess(result.message || t('admin.settings.smtpConnectionSuccess'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.failedToTestSmtp') + ': ' + (error.message || t('common.unknownError'))
      )
    } finally {
      testingSmtp.value = false
    }
  }

  async function testTelegramConnection() {
    testingTelegram.value = true
    try {
      const result = await adminAPI.settings.testTelegramConnection({
        bot_token: form.telegram_bot_token || undefined,
        chat_id: form.telegram_chat_id || undefined
      })
      appStore.showSuccess(result.message || t('admin.settings.telegram.testSuccess'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.telegram.testFailed') +
          ': ' +
          (error.message || t('common.unknownError'))
      )
    } finally {
      testingTelegram.value = false
    }
  }

  async function sendTestEmail() {
    if (!testEmailAddress.value) {
      appStore.showError(t('admin.settings.testEmail.enterRecipientHint'))
      return
    }

    sendingTestEmail.value = true
    try {
      const result = await adminAPI.settings.sendTestEmail({
        email: testEmailAddress.value,
        smtp_host: form.smtp_host,
        smtp_port: form.smtp_port,
        smtp_username: form.smtp_username,
        smtp_password: form.smtp_password,
        smtp_from_email: form.smtp_from_email,
        smtp_from_name: form.smtp_from_name,
        smtp_use_tls: form.smtp_use_tls
      })
      appStore.showSuccess(result.message || t('admin.settings.testEmailSent'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.failedToSendTestEmail') + ': ' + (error.message || t('common.unknownError'))
      )
    } finally {
      sendingTestEmail.value = false
    }
  }

  async function loadEmailTemplates() {
    emailTemplatesLoading.value = true
    try {
      const items = await adminAPI.settings.getEmailTemplates()
      emailTemplateDefinitions.value = items
      if (!selectedEmailTemplateKey.value && items.length > 0) {
        selectedEmailTemplateKey.value = items[0].key
      }
      syncEmailTemplateDraft()
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.emailTemplates.loadFailed') +
          ': ' +
          (error.message || t('common.unknownError'))
      )
    } finally {
      emailTemplatesLoading.value = false
    }
  }

  function selectEmailTemplate(key: string) {
    selectedEmailTemplateKey.value = key
    syncEmailTemplateDraft()
  }

  function selectEmailTemplateLocale(locale: string) {
    selectedEmailTemplateLocale.value = locale === 'zh' ? 'zh' : 'en'
    syncEmailTemplateDraft()
  }

  async function saveSelectedEmailTemplate() {
    if (!selectedEmailTemplateKey.value) return
    emailTemplateSaving.value = true
    try {
      await adminAPI.settings.updateEmailTemplate(
        selectedEmailTemplateKey.value,
        selectedEmailTemplateLocale.value,
        {
          subject: emailTemplateDraft.subject,
          body: emailTemplateDraft.body,
          enabled: emailTemplateDraft.enabled
        }
      )
      await loadEmailTemplates()
      appStore.showSuccess(t('admin.settings.emailTemplates.saved'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.emailTemplates.saveFailed') +
          ': ' +
          (error.message || t('common.unknownError'))
      )
    } finally {
      emailTemplateSaving.value = false
    }
  }

  async function resetSelectedEmailTemplate() {
    if (!selectedEmailTemplateKey.value) return
    emailTemplateSaving.value = true
    try {
      await adminAPI.settings.resetEmailTemplate(
        selectedEmailTemplateKey.value,
        selectedEmailTemplateLocale.value
      )
      await loadEmailTemplates()
      appStore.showSuccess(t('admin.settings.emailTemplates.resetDone'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.emailTemplates.resetFailed') +
          ': ' +
          (error.message || t('common.unknownError'))
      )
    } finally {
      emailTemplateSaving.value = false
    }
  }

  async function testSelectedEmailTemplate() {
    if (!selectedEmailTemplateKey.value || !emailTemplateTestAddress.value) return
    emailTemplateTesting.value = true
    try {
      const result = await adminAPI.settings.testEmailTemplate(
        selectedEmailTemplateKey.value,
        selectedEmailTemplateLocale.value,
        emailTemplateTestAddress.value
      )
      appStore.showSuccess(result.message || t('admin.settings.emailTemplates.testSent'))
    } catch (error: any) {
      appStore.showError(
        t('admin.settings.emailTemplates.testFailed') +
          ': ' +
          (error.message || t('common.unknownError'))
      )
    } finally {
      emailTemplateTesting.value = false
    }
  }

  return {
    testingSmtp,
    testingTelegram,
    sendingTestEmail,
    testEmailAddress,
    emailTemplatesLoading,
    emailTemplateSaving,
    emailTemplateTesting,
    selectedEmailTemplateKey,
    selectedEmailTemplateLocale,
    emailTemplateTestAddress,
    emailTemplateDraft,
    emailTemplateOptions,
    emailTemplateLocaleOptions,
    selectedEmailTemplate,
    formatEmailTemplateVariable,
    testSmtpConnection,
    testTelegramConnection,
    sendTestEmail,
    loadEmailTemplates,
    selectEmailTemplate,
    selectEmailTemplateLocale,
    saveSelectedEmailTemplate,
    resetSelectedEmailTemplate,
    testSelectedEmailTemplate
  }
}
