<template>
  <div class="grid gap-4 xl:grid-cols-[minmax(0,1.1fr)_minmax(340px,0.9fr)]">
    <section class="space-y-4">
      <div class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h2 class="text-xl font-semibold text-gray-900 dark:text-white">规则矩阵</h2>
            <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">这里保留规则编辑与计费模拟器，承接旧计费中心的高级能力。</p>
          </div>
          <button type="button" class="btn btn-secondary btn-sm" @click="resetRuleEditor">新建规则</button>
        </div>

        <div class="mt-4 max-h-[420px] overflow-auto rounded-2xl border border-gray-200 dark:border-dark-700">
          <table class="min-w-full text-sm">
            <tbody>
              <tr
                v-for="rule in rules"
                :key="rule.id"
                class="cursor-pointer border-t border-gray-100 dark:border-dark-700"
                :class="selectedRuleId === rule.id ? 'bg-primary-50 dark:bg-primary-500/10' : ''"
                @click="selectRule(rule)"
              >
                <td class="px-4 py-3">
                  <div class="font-medium text-gray-900 dark:text-white">{{ rule.id }}</div>
                  <div class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ rule.provider }} / {{ rule.layer }} / {{ rule.unit }}</div>
                </td>
                <td class="px-4 py-3 text-right text-gray-700 dark:text-gray-200">{{ rule.price }}</td>
              </tr>
              <tr v-if="rules.length === 0">
                <td colspan="2" class="px-4 py-10 text-center text-sm text-gray-500 dark:text-gray-400">暂无规则。</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center justify-between gap-3">
          <div>
            <h3 class="text-lg font-semibold text-gray-900 dark:text-white">计费模拟</h3>
            <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">使用结构化 JSON 验证规则命中、fallback 与总成本。</p>
          </div>
          <div class="flex gap-2">
            <button type="button" class="btn btn-primary btn-sm" :disabled="busy" @click="runSimulation">运行模拟</button>
            <button type="button" class="btn btn-secondary btn-sm" :disabled="busy" @click="resetSimulation">重置</button>
          </div>
        </div>

        <textarea v-model="simulationDraft" rows="12" class="input mt-4 w-full font-mono text-xs" />

        <div v-if="simulationResult" class="mt-4 rounded-2xl bg-slate-950 p-4 text-sm text-slate-100">
          <div>总成本：{{ simulationResult.total_cost }}</div>
          <div class="mt-1">实际成本：{{ simulationResult.actual_cost }}</div>
          <div v-if="simulationResult.fallback?.reason" class="mt-2 text-slate-300">Fallback：{{ simulationResult.fallback.reason }}</div>
          <div v-if="simulationResult.matched_rule_ids?.length" class="mt-2 text-slate-300">命中规则：{{ simulationResult.matched_rule_ids.join(', ') }}</div>
        </div>
      </div>
    </section>

    <aside class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="flex items-center justify-between gap-3">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">规则编辑器</h3>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ selectedRuleId || '当前为新建规则' }}</p>
        </div>
        <button type="button" class="btn btn-danger btn-sm" :disabled="busy || !selectedRuleId" @click="removeRule">删除</button>
      </div>

      <textarea v-model="ruleDraft" rows="24" class="input mt-4 w-full font-mono text-xs" />
      <div class="mt-4 flex justify-end">
        <button type="button" class="btn btn-primary" :disabled="busy" @click="saveRule">保存规则</button>
      </div>
    </aside>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { deleteBillingRule, listBillingRules, simulateBilling, updateBillingRule, type BillingRule, type BillingSimulationInput, type BillingSimulationResult } from '@/api/admin/billing'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const rules = ref<BillingRule[]>([])
const selectedRuleId = ref('')
const ruleDraft = ref('')
const simulationDraft = ref('')
const simulationResult = ref<BillingSimulationResult | null>(null)
const busy = ref(false)

onMounted(async () => {
  resetRuleEditor()
  resetSimulation()
  await loadRules()
})

async function loadRules() {
  try {
    rules.value = await listBillingRules()
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '加载规则失败'))
  }
}

function selectRule(rule: BillingRule) {
  selectedRuleId.value = rule.id
  ruleDraft.value = JSON.stringify(rule, null, 2)
}

function resetRuleEditor() {
  selectedRuleId.value = ''
  ruleDraft.value = JSON.stringify({
    id: '',
    provider: 'openai',
    layer: 'sale',
    surface: 'any',
    operation_type: 'generate_content',
    service_tier: 'standard',
    batch_mode: 'any',
    matchers: {},
    unit: 'input_token',
    price: 0,
    priority: 2000,
    enabled: true,
  }, null, 2)
}

function resetSimulation() {
  simulationResult.value = null
  simulationDraft.value = JSON.stringify({
    provider: 'gemini',
    layer: 'sale',
    model: 'gemini-2.5-pro',
    surface: 'native',
    operation_type: 'generate_content',
    service_tier: 'standard',
    batch_mode: 'realtime',
    input_modality: 'text',
    output_modality: 'text',
    cache_phase: '',
    grounding_kind: '',
    charges: {
      text_input_tokens: 1000,
      text_output_tokens: 500,
    },
  }, null, 2)
}

async function saveRule() {
  busy.value = true
  try {
    const payload = JSON.parse(ruleDraft.value) as BillingRule
    const saved = await updateBillingRule(payload)
    await loadRules()
    selectRule(saved)
    appStore.showSuccess('规则已保存')
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '保存规则失败'))
  } finally {
    busy.value = false
  }
}

async function removeRule() {
  if (!selectedRuleId.value) return
  busy.value = true
  try {
    await deleteBillingRule(selectedRuleId.value)
    await loadRules()
    resetRuleEditor()
    appStore.showSuccess('规则已删除')
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '删除规则失败'))
  } finally {
    busy.value = false
  }
}

async function runSimulation() {
  busy.value = true
  try {
    simulationResult.value = await simulateBilling(JSON.parse(simulationDraft.value) as BillingSimulationInput)
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, '运行模拟失败'))
  } finally {
    busy.value = false
  }
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error && 'message' in error && typeof (error as { message?: unknown }).message === 'string') {
    return String((error as { message: string }).message)
  }
  return fallback
}
</script>
