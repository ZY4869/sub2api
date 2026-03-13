import { computed, ref } from 'vue'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import {
  buildTempUnschedRules,
  createEmptyTempUnschedRule,
  createTempUnschedPresets,
  loadTempUnschedRulesFromCredentials,
  type TempUnschedRuleForm
} from '@/utils/accountFormShared'

interface UseAccountTempUnschedRulesOptions {
  keyPrefix: string
  invalidMessage: () => string
  showError: (message: string) => void
  t: (key: string) => string
}

/**
 * Shared temp-unsched editor state for account modals.
 * The create/edit flows use the same rule semantics, so we keep the
 * validation, serialization and preset handling in one place.
 */
export function useAccountTempUnschedRules(options: UseAccountTempUnschedRulesOptions) {
  const enabled = ref(false)
  const rules = ref<TempUnschedRuleForm[]>([])
  const presets = computed(() => createTempUnschedPresets(options.t))
  const getRuleKey = createStableObjectKeyResolver<TempUnschedRuleForm>(
    `${options.keyPrefix}-temp-unsched-rule`
  )

  const addRule = (preset?: TempUnschedRuleForm) => {
    rules.value.push(preset ? { ...preset } : createEmptyTempUnschedRule())
  }

  const removeRule = (index: number) => {
    rules.value.splice(index, 1)
  }

  const moveRule = (index: number, direction: number) => {
    const target = index + direction
    if (target < 0 || target >= rules.value.length) {
      return
    }

    const current = rules.value[index]
    rules.value[index] = rules.value[target]
    rules.value[target] = current
  }

  const buildRulesPayload = () => buildTempUnschedRules(rules.value)

  const applyToCredentials = (credentials: Record<string, unknown>) => {
    if (!enabled.value) {
      delete credentials.temp_unschedulable_enabled
      delete credentials.temp_unschedulable_rules
      return true
    }

    const payload = buildRulesPayload()
    if (payload.length === 0) {
      options.showError(options.invalidMessage())
      return false
    }

    credentials.temp_unschedulable_enabled = true
    credentials.temp_unschedulable_rules = payload
    return true
  }

  const loadFromCredentials = (credentials?: Record<string, unknown>) => {
    const next = loadTempUnschedRulesFromCredentials(credentials)
    enabled.value = next.enabled
    rules.value = next.rules
  }

  const reset = () => {
    enabled.value = false
    rules.value = []
  }

  return {
    enabled,
    rules,
    presets,
    getRuleKey,
    addRule,
    removeRule,
    moveRule,
    buildRulesPayload,
    applyToCredentials,
    loadFromCredentials,
    reset
  }
}
