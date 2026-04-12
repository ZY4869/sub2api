<template>
  <div class="space-y-6">
    <section class="grid gap-4 xl:grid-cols-[minmax(0,1.3fr)_minmax(280px,0.7fr)]">
      <div class="rounded-3xl border border-sky-200 bg-gradient-to-br from-sky-50 via-white to-cyan-50 p-6 shadow-sm dark:border-sky-500/20 dark:from-sky-500/10 dark:via-dark-800 dark:to-cyan-500/10">
        <div class="flex flex-wrap items-center gap-2">
          <span class="inline-flex rounded-full bg-sky-600 px-3 py-1 text-xs font-semibold uppercase tracking-[0.16em] text-white">{{ t('admin.models.pages.billing.badge') }}</span>
          <span class="inline-flex rounded-full bg-white/80 px-3 py-1 text-xs font-medium text-sky-700 shadow-sm dark:bg-dark-900/60 dark:text-sky-200">{{ t('admin.models.pages.billing.compatibilityBadge') }}</span>
        </div>
        <h2 class="mt-4 text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.heroTitle') }}</h2>
        <p class="mt-3 max-w-3xl text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('admin.models.pages.billing.heroDescription') }}</p>
      </div>

      <div class="grid gap-3">
        <button class="btn btn-primary justify-center" :disabled="loading" @click="loadBillingCenter">{{ t('common.refresh') }}</button>
        <div class="rounded-3xl border border-amber-200 bg-amber-50/80 p-5 shadow-sm dark:border-amber-500/20 dark:bg-amber-500/10">
          <h3 class="text-sm font-semibold text-amber-900 dark:text-amber-200">{{ t('admin.models.pages.billing.classificationTitle') }}</h3>
          <p class="mt-2 text-sm leading-6 text-amber-800/90 dark:text-amber-100/90">{{ t('admin.models.pages.billing.classificationDescription') }}</p>
        </div>
        <div class="rounded-3xl border border-emerald-200 bg-emerald-50/80 p-5 shadow-sm dark:border-emerald-500/20 dark:bg-emerald-500/10">
          <h3 class="text-sm font-semibold text-emerald-900 dark:text-emerald-200">{{ t('admin.models.pages.billing.catalogSplitTitle') }}</h3>
          <p class="mt-2 text-sm leading-6 text-emerald-800/90 dark:text-emerald-100/90">{{ t('admin.models.pages.billing.catalogSplitDescription') }}</p>
        </div>
      </div>
    </section>

    <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.priceMatrixTitle') }}</h3>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ t('admin.models.pages.billing.priceMatrixDescription') }}</p>
        </div>
        <input v-model.trim="sheetSearch" type="text" class="input w-full max-w-xs" :placeholder="t('admin.models.pages.billing.sheetSearchPlaceholder')" />
      </div>

      <div class="mt-4 grid gap-4 xl:grid-cols-2">
        <article class="rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-900/40">
          <h4 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.official.title') }}</h4>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ t('admin.models.pages.official.description') }}</p>
          <div class="mt-4 max-h-[320px] overflow-auto rounded-2xl border border-gray-200 dark:border-dark-700">
            <table class="min-w-full text-sm">
              <tbody>
                <tr v-for="sheet in filteredSheets" :key="`official-${sheet.model}`" class="cursor-pointer border-t border-gray-200 transition-colors hover:bg-sky-50 dark:border-dark-700 dark:hover:bg-sky-500/10" :class="officialSelectedModel === sheet.model ? 'bg-sky-50 dark:bg-sky-500/10' : ''" @click="selectSheet('official', sheet.model)">
                  <td class="px-3 py-3">
                    <p class="font-medium text-gray-900 dark:text-white">{{ sheet.display_name || sheet.model }}</p>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ sheet.model }}</p>
                  </td>
                  <td class="px-3 py-3 text-right text-xs text-gray-500 dark:text-gray-400">{{ formatSheetLayerPreview(sheet, 'official') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="mt-4 space-y-3">
            <div class="flex flex-wrap items-center gap-2">
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ officialSelectedModel || t('admin.models.pages.billing.selectModelHint') }}</p>
              <span v-if="officialCurrentSheet && isGeminiSheet(officialCurrentSheet)" class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-[11px] font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-200">
                {{ t('admin.models.pages.billing.matrixBadge') }}
              </span>
              <span v-else-if="officialCurrentSheet" class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-[11px] font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                {{ t('admin.models.pages.billing.legacyBadge') }}
              </span>
            </div>
            <BillingMatrixEditor
              v-if="officialCurrentSheet && isGeminiSheet(officialCurrentSheet)"
              v-model="officialMatrixDraft"
              :left-header="t('admin.models.pages.billing.matrixLeftHeader')"
              :rule-label="t('admin.models.pages.billing.ruleIdLabel')"
              :derived-label="t('admin.models.pages.billing.derivedViaLabel')"
            />
            <textarea v-else v-model="officialDraft" rows="10" class="input w-full font-mono text-xs" :disabled="!officialSelectedModel" />
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-primary" :disabled="savingSheet || !officialSelectedModel" @click="saveSheet('official')">{{ t('common.save') }}</button>
              <button class="btn btn-secondary" :disabled="savingSheet || !officialSelectedModel" @click="deleteSheetAction('official')">{{ t('common.delete') }}</button>
              <button class="btn btn-secondary" :disabled="!officialSelectedModel" @click="syncDraft('official')">{{ t('common.reset') }}</button>
            </div>
          </div>
        </article>

        <article class="rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-900/40">
          <h4 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.sale.title') }}</h4>
          <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ t('admin.models.pages.sale.description') }}</p>
          <div class="mt-4 max-h-[320px] overflow-auto rounded-2xl border border-gray-200 dark:border-dark-700">
            <table class="min-w-full text-sm">
              <tbody>
                <tr v-for="sheet in filteredSheets" :key="`sale-${sheet.model}`" class="cursor-pointer border-t border-gray-200 transition-colors hover:bg-sky-50 dark:border-dark-700 dark:hover:bg-sky-500/10" :class="saleSelectedModel === sheet.model ? 'bg-sky-50 dark:bg-sky-500/10' : ''" @click="selectSheet('sale', sheet.model)">
                  <td class="px-3 py-3">
                    <p class="font-medium text-gray-900 dark:text-white">{{ sheet.display_name || sheet.model }}</p>
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ sheet.model }}</p>
                  </td>
                  <td class="px-3 py-3 text-right text-xs text-gray-500 dark:text-gray-400">{{ formatSheetLayerPreview(sheet, 'sale') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="mt-4 space-y-3">
            <div class="flex flex-wrap items-center gap-2">
              <p class="text-sm font-medium text-gray-900 dark:text-white">{{ saleSelectedModel || t('admin.models.pages.billing.selectModelHint') }}</p>
              <span v-if="saleCurrentSheet && isGeminiSheet(saleCurrentSheet)" class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-[11px] font-medium text-sky-700 dark:bg-sky-500/15 dark:text-sky-200">
                {{ t('admin.models.pages.billing.matrixBadge') }}
              </span>
              <span v-else-if="saleCurrentSheet" class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-[11px] font-medium text-gray-700 dark:bg-dark-700 dark:text-gray-200">
                {{ t('admin.models.pages.billing.legacyBadge') }}
              </span>
            </div>
            <BillingMatrixEditor
              v-if="saleCurrentSheet && isGeminiSheet(saleCurrentSheet)"
              v-model="saleMatrixDraft"
              :left-header="t('admin.models.pages.billing.matrixLeftHeader')"
              :rule-label="t('admin.models.pages.billing.ruleIdLabel')"
              :derived-label="t('admin.models.pages.billing.derivedViaLabel')"
            />
            <textarea v-else v-model="saleDraft" rows="10" class="input w-full font-mono text-xs" :disabled="!saleSelectedModel" />
            <div class="flex flex-wrap gap-2">
              <button class="btn btn-primary" :disabled="savingSheet || !saleSelectedModel" @click="saveSheet('sale')">{{ t('common.save') }}</button>
              <button class="btn btn-secondary" :disabled="savingSheet || !saleSelectedModel" @click="deleteSheetAction('sale')">{{ t('common.delete') }}</button>
              <button class="btn btn-secondary" :disabled="savingSheet || !saleSelectedModel" @click="copyOfficialToSaleAction">{{ t('admin.models.pages.billing.copyOfficialToSale') }}</button>
              <button class="btn btn-secondary" :disabled="!saleSelectedModel" @click="syncDraft('sale')">{{ t('common.reset') }}</button>
            </div>
          </div>
        </article>
      </div>
    </section>

    <section class="grid gap-4 xl:grid-cols-[minmax(0,1.15fr)_minmax(320px,0.85fr)]">
      <div class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.ruleMatrixTitle') }}</h3>
        <div class="mt-4 grid gap-3 md:grid-cols-2">
          <article v-for="card in ruleCards" :key="card.title" class="rounded-2xl border border-gray-200 bg-gray-50/80 p-4 dark:border-dark-700 dark:bg-dark-900/40">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ card.title }}</h4>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ card.body }}</p>
          </article>
        </div>
        <div class="mt-4 max-h-[360px] overflow-auto rounded-2xl border border-gray-200 dark:border-dark-700">
          <table class="min-w-full text-sm">
            <tbody>
              <tr v-for="rule in rules" :key="rule.id" class="cursor-pointer border-t border-gray-200 transition-colors hover:bg-sky-50 dark:border-dark-700 dark:hover:bg-sky-500/10" :class="selectedRuleId === rule.id ? 'bg-sky-50 dark:bg-sky-500/10' : ''" @click="selectRule(rule)">
                <td class="px-3 py-3 text-gray-700 dark:text-gray-200">{{ humanize(rule.surface) }}</td>
                <td class="px-3 py-3 text-gray-700 dark:text-gray-200">{{ humanize(rule.operation_type) }}</td>
                <td class="px-3 py-3 text-gray-700 dark:text-gray-200">{{ humanize(rule.unit) }}</td>
                <td class="px-3 py-3 text-right text-gray-700 dark:text-gray-200">{{ formatNumber(rule.price) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <aside class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <div class="flex items-center justify-between gap-3">
          <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.ruleEditorTitle') }}</h3>
          <button class="btn btn-secondary btn-sm" :disabled="savingRule" @click="resetRuleEditor">{{ t('admin.models.pages.billing.newRule') }}</button>
        </div>
        <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ selectedRuleId || t('admin.models.pages.billing.ruleEditorNew') }}</p>
        <textarea v-model="ruleDraft" rows="18" class="input mt-4 w-full font-mono text-xs" />
        <div class="mt-4 flex flex-wrap gap-2">
          <button class="btn btn-primary" :disabled="savingRule" @click="saveRule">{{ t('common.save') }}</button>
          <button class="btn btn-secondary" :disabled="savingRule || !selectedRuleId" @click="deleteRuleAction">{{ t('common.delete') }}</button>
        </div>
      </aside>
    </section>

    <section class="grid gap-4 xl:grid-cols-[minmax(0,1fr)_minmax(320px,0.95fr)]">
      <div class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.simulatorTitle') }}</h3>
        <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ t('admin.models.pages.billing.simulatorDescription') }}</p>
        <textarea v-model="simulationDraft" rows="18" class="input mt-4 w-full font-mono text-xs" />
        <div class="mt-4 flex flex-wrap gap-2">
          <button class="btn btn-primary" :disabled="simulating" @click="runSimulation">{{ t('common.search') }}</button>
          <button class="btn btn-secondary" :disabled="simulating" @click="resetSimulation">{{ t('common.reset') }}</button>
        </div>
      </div>

      <div class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
        <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.simulatorResultTitle') }}</h3>
        <div v-if="simulationResult" class="mt-4 space-y-4">
          <div class="rounded-2xl bg-slate-950 p-4 text-sm text-slate-100">
            <p class="font-medium">{{ t('admin.models.pages.billing.simulatorTotalCost', { total: formatNumber(simulationResult.total_cost) }) }}</p>
            <p class="mt-1 text-slate-300">{{ t('admin.models.pages.billing.actualCostLabel') }}{{ formatNumber(simulationResult.actual_cost) }}</p>
            <p v-if="simulationHeadline" class="mt-3 text-slate-200">{{ simulationHeadline }}</p>
            <p v-if="simulationSummary" class="mt-1 text-slate-400">{{ simulationSummary }}</p>
          </div>

          <section v-if="simulationResult.classification" class="space-y-2">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.classificationResultTitle') }}</h4>
            <div class="flex flex-wrap gap-2 text-xs">
              <span class="inline-flex rounded-full bg-sky-100 px-2.5 py-1 text-sky-700 dark:bg-sky-500/15 dark:text-sky-200">{{ resolveSurfaceLabel(simulationResult.classification.surface) }}</span>
              <span class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ humanize(simulationResult.classification.operation_type) }}</span>
              <span class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ humanize(simulationResult.classification.service_tier || 'standard') }}</span>
              <span class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ humanize(simulationResult.classification.batch_mode || 'realtime') }}</span>
              <span v-if="simulationResult.classification.input_modality" class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ t('admin.models.pages.billing.inputModalityLabel') }}{{ humanize(simulationResult.classification.input_modality) }}</span>
              <span v-if="simulationResult.classification.output_modality" class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ t('admin.models.pages.billing.outputModalityLabel') }}{{ humanize(simulationResult.classification.output_modality) }}</span>
              <span v-if="simulationResult.classification.grounding_kind" class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ t('admin.models.pages.billing.groundingKindLabel') }}{{ humanize(simulationResult.classification.grounding_kind) }}</span>
              <span v-if="simulationResult.classification.cache_phase" class="inline-flex rounded-full bg-gray-200 px-2.5 py-1 text-gray-700 dark:bg-dark-700 dark:text-gray-200">{{ t('admin.models.pages.billing.cachePhaseLabel') }}{{ humanize(simulationResult.classification.cache_phase) }}</span>
            </div>
          </section>

          <section v-if="simulationResult.matched_rules?.length" class="space-y-2">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.matchedRulesTitle') }}</h4>
            <div class="space-y-2">
              <div v-for="rule in simulationResult.matched_rules" :key="rule.id" class="rounded-2xl border border-gray-200 bg-gray-50/80 p-3 text-sm dark:border-dark-700 dark:bg-dark-900/40">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div class="font-medium text-gray-900 dark:text-white">{{ rule.id }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ formatNumber(rule.price) }}</div>
                </div>
                <div class="mt-2 text-xs text-gray-600 dark:text-gray-300">{{ humanize(rule.surface) }} / {{ humanize(rule.operation_type) }} / {{ humanize(rule.unit) }}</div>
              </div>
            </div>
          </section>

          <section class="space-y-2">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.costBreakdownTitle') }}</h4>
            <div class="overflow-auto rounded-2xl border border-gray-200 dark:border-dark-700">
              <table class="min-w-full text-sm">
                <thead class="bg-gray-100/80 dark:bg-dark-900/60">
                  <tr>
                    <th class="px-3 py-2 text-left font-semibold text-gray-700 dark:text-gray-200">{{ t('admin.models.pages.billing.chargeSlotLabel') }}</th>
                    <th class="px-3 py-2 text-left font-semibold text-gray-700 dark:text-gray-200">{{ t('admin.models.pages.billing.unitLabel') }}</th>
                    <th class="px-3 py-2 text-right font-semibold text-gray-700 dark:text-gray-200">{{ t('admin.models.pages.billing.unitsLabel') }}</th>
                    <th class="px-3 py-2 text-right font-semibold text-gray-700 dark:text-gray-200">{{ t('admin.models.pages.billing.priceLabel') }}</th>
                    <th class="px-3 py-2 text-right font-semibold text-gray-700 dark:text-gray-200">{{ t('admin.models.pages.billing.costLabel') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="line in simulationResult.lines" :key="`${line.charge_slot}:${line.rule_id || 'na'}`" class="border-t border-gray-200 dark:border-dark-700">
                    <td class="px-3 py-2 text-gray-700 dark:text-gray-200">{{ humanize(line.charge_slot) }}</td>
                    <td class="px-3 py-2 text-gray-700 dark:text-gray-200">{{ humanize(line.unit) }}</td>
                    <td class="px-3 py-2 text-right text-gray-700 dark:text-gray-200">{{ formatNumber(line.units) }}</td>
                    <td class="px-3 py-2 text-right text-gray-700 dark:text-gray-200">{{ formatNumber(line.price) }}</td>
                    <td class="px-3 py-2 text-right text-gray-700 dark:text-gray-200">
                      <div>{{ formatNumber(line.cost) }}</div>
                      <div class="text-[11px] text-gray-500 dark:text-gray-400">{{ t('admin.models.pages.billing.actualCostShortLabel') }}{{ formatNumber(line.actual_cost) }}</div>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <section v-if="simulationResult.unmatched_demands?.length" class="space-y-2">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.unmatchedDemandsTitle') }}</h4>
            <div class="space-y-2">
              <div v-for="demand in simulationResult.unmatched_demands" :key="`${demand.charge_slot}:${demand.reason}`" class="rounded-2xl border border-amber-200 bg-amber-50/80 p-3 text-sm dark:border-amber-500/20 dark:bg-amber-500/10">
                <div class="flex flex-wrap items-center justify-between gap-2">
                  <div class="font-medium text-amber-900 dark:text-amber-100">{{ humanize(demand.charge_slot) }}</div>
                  <div class="text-xs text-amber-700 dark:text-amber-200">{{ humanize(demand.reason) }}</div>
                </div>
                <div class="mt-2 text-xs text-amber-800/90 dark:text-amber-100/90">{{ t('admin.models.pages.billing.unmatchedDemandSummary', { unit: humanize(demand.unit), units: formatNumber(demand.units) }) }}</div>
                <div v-if="demand.missing_dimensions?.length" class="mt-1 text-xs text-amber-700 dark:text-amber-200">{{ t('admin.models.pages.billing.missingDimensionsLabel') }}{{ demand.missing_dimensions.map((item) => humanize(item)).join(', ') }}</div>
              </div>
            </div>
          </section>

          <section v-if="simulationResult.fallback" class="space-y-2">
            <h4 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('admin.models.pages.billing.fallbackTitle') }}</h4>
            <div class="rounded-2xl border border-indigo-200 bg-indigo-50/80 p-4 text-sm dark:border-indigo-500/20 dark:bg-indigo-500/10">
              <div class="flex flex-wrap gap-3 text-indigo-900 dark:text-indigo-100">
                <span>{{ t('admin.models.pages.billing.fallbackPolicyLabel') }}{{ humanize(simulationResult.fallback.policy || '-') }}</span>
                <span>{{ t('admin.models.pages.billing.fallbackAppliedLabel') }}{{ simulationResult.fallback.applied ? t('common.yes') : t('common.no') }}</span>
              </div>
              <div v-if="simulationResult.fallback.reason" class="mt-2 text-indigo-800 dark:text-indigo-200">{{ t('admin.models.pages.billing.fallbackReasonLabel') }}{{ humanize(simulationResult.fallback.reason) }}</div>
              <div v-if="simulationResult.fallback.derived_from" class="mt-1 text-indigo-800 dark:text-indigo-200">{{ t('admin.models.pages.billing.fallbackDerivedFromLabel') }}{{ humanize(simulationResult.fallback.derived_from) }}</div>

              <div v-if="simulationResult.fallback.cost_lines?.length" class="mt-4 overflow-auto rounded-2xl border border-indigo-200/70 bg-white/70 dark:border-indigo-500/20 dark:bg-dark-900/50">
                <table class="min-w-full text-xs">
                  <thead class="bg-white/80 dark:bg-dark-900/60">
                    <tr>
                      <th class="px-3 py-2 text-left font-semibold text-indigo-900 dark:text-indigo-100">{{ t('admin.models.pages.billing.chargeSlotLabel') }}</th>
                      <th class="px-3 py-2 text-left font-semibold text-indigo-900 dark:text-indigo-100">{{ t('admin.models.pages.billing.unitLabel') }}</th>
                      <th class="px-3 py-2 text-right font-semibold text-indigo-900 dark:text-indigo-100">{{ t('admin.models.pages.billing.costLabel') }}</th>
                    </tr>
                  </thead>
                  <tbody>
                    <tr v-for="line in simulationResult.fallback.cost_lines" :key="`${line.charge_slot}:${line.unit}`" class="border-t border-indigo-200/60 dark:border-indigo-500/20">
                      <td class="px-3 py-2 text-indigo-900 dark:text-indigo-100">{{ humanize(line.charge_slot) }}</td>
                      <td class="px-3 py-2 text-indigo-900 dark:text-indigo-100">{{ humanize(line.unit) }}</td>
                      <td class="px-3 py-2 text-right text-indigo-900 dark:text-indigo-100">{{ formatNumber(line.cost) }}</td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </section>
        </div>
        <p v-else class="mt-4 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.models.pages.billing.simulatorEmpty') }}</p>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BillingMatrixEditor from '@/components/admin/models/BillingMatrixEditor.vue'
import {
  copyBillingSheetOfficialToSale,
  deleteBillingRule,
  deleteBillingSheet,
  getBillingCenter,
  simulateBilling,
  updateBillingRule,
  updateBillingSheet,
  type BillingRule,
  type BillingSimulationInput,
  type BillingSimulationResult,
  type GeminiBillingMatrix,
  type ModelBillingSheet,
  type ModelCatalogPricing
} from '@/api/admin/models'
import { useAppStore } from '@/stores/app'

type SheetLayer = 'official' | 'sale'

const { t } = useI18n()
const appStore = useAppStore()

const sheets = ref<ModelBillingSheet[]>([])
const rules = ref<BillingRule[]>([])
const loading = ref(false)
const savingSheet = ref(false)
const savingRule = ref(false)
const simulating = ref(false)
const sheetSearch = ref('')
const officialSelectedModel = ref('')
const saleSelectedModel = ref('')
const officialDraft = ref('{}')
const saleDraft = ref('{}')
const officialMatrixDraft = ref<GeminiBillingMatrix | null>(null)
const saleMatrixDraft = ref<GeminiBillingMatrix | null>(null)
const selectedRuleId = ref('')
const ruleDraft = ref('')
const simulationDraft = ref('')
const simulationResult = ref<BillingSimulationResult | null>(null)

const filteredSheets = computed(() => {
  const query = sheetSearch.value.trim().toLowerCase()
  if (!query) return sheets.value
  return sheets.value.filter((sheet) => [sheet.model, sheet.display_name, sheet.model_family].some((value) => String(value || '').toLowerCase().includes(query)))
})

const officialCurrentSheet = computed(() => sheets.value.find((sheet) => sheet.model === officialSelectedModel.value) || null)
const saleCurrentSheet = computed(() => sheets.value.find((sheet) => sheet.model === saleSelectedModel.value) || null)

const ruleCards = computed(() => [
  { title: t('admin.models.pages.billing.ruleCards.surface.title'), body: t('admin.models.pages.billing.ruleCards.surface.body') },
  { title: t('admin.models.pages.billing.ruleCards.serviceTier.title'), body: t('admin.models.pages.billing.ruleCards.serviceTier.body') },
  { title: t('admin.models.pages.billing.ruleCards.modality.title'), body: t('admin.models.pages.billing.ruleCards.modality.body') },
  { title: t('admin.models.pages.billing.ruleCards.cache.title'), body: t('admin.models.pages.billing.ruleCards.cache.body') }
  ])

const simulationHeadline = computed(() => {
  const surface = simulationResult.value?.classification?.surface
  if (!surface) return ''
  switch (surface) {
    case 'native':
      return t('admin.models.pages.billing.simulatorHeadlines.native')
    case 'openai_compat':
      return t('admin.models.pages.billing.simulatorHeadlines.compat')
    case 'live':
      return t('admin.models.pages.billing.simulatorHeadlines.live')
    case 'interactions':
      return t('admin.models.pages.billing.simulatorHeadlines.interactions')
    default:
      return ''
  }
})

const simulationSummary = computed(() => {
  const classification = simulationResult.value?.classification
  if (!classification) return ''
  return t('admin.models.pages.billing.simulatorSummary', {
    surface: resolveSurfaceLabel(classification.surface),
    tier: humanize(classification.service_tier || 'standard'),
    batch: humanize(classification.batch_mode || 'realtime')
  })
})

watch(officialSelectedModel, () => syncDraft('official'))
watch(saleSelectedModel, () => syncDraft('sale'))

onMounted(() => {
  resetRuleEditor()
  resetSimulation()
  void loadBillingCenter()
})

async function loadBillingCenter() {
  loading.value = true
  try {
    const payload = await getBillingCenter()
    sheets.value = payload.sheets || []
    rules.value = payload.rules || []
    if (!officialSelectedModel.value && sheets.value[0]) officialSelectedModel.value = sheets.value[0].model
    if (!saleSelectedModel.value && sheets.value[0]) saleSelectedModel.value = sheets.value[0].model
    syncDraft('official')
    syncDraft('sale')
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, t('admin.models.pages.billing.loadFailed')))
  } finally {
    loading.value = false
  }
}

function selectSheet(layer: SheetLayer, model: string) {
  if (layer === 'official') officialSelectedModel.value = model
  else saleSelectedModel.value = model
}

function syncDraft(layer: SheetLayer) {
  const sheet = layer === 'official' ? officialCurrentSheet.value : saleCurrentSheet.value
  if (!sheet) {
    if (layer === 'official') {
      officialDraft.value = '{}'
      officialMatrixDraft.value = null
    } else {
      saleDraft.value = '{}'
      saleMatrixDraft.value = null
    }
    return
  }
  if (isGeminiSheet(sheet)) {
    const matrix = layer === 'official' ? sheet.official_matrix : sheet.sale_matrix
    if (layer === 'official') {
      officialMatrixDraft.value = cloneMatrix(matrix)
      officialDraft.value = '{}'
    } else {
      saleMatrixDraft.value = cloneMatrix(matrix)
      saleDraft.value = '{}'
    }
    return
  }
  const pricing = layer === 'official' ? sheet.official_pricing : saleCurrentSheet.value?.sale_pricing
  const next = JSON.stringify(pricing || {}, null, 2)
  if (layer === 'official') {
    officialDraft.value = next
    officialMatrixDraft.value = null
  } else {
    saleDraft.value = next
    saleMatrixDraft.value = null
  }
}

async function saveSheet(layer: SheetLayer) {
  const sheet = layer === 'official' ? officialCurrentSheet.value : saleCurrentSheet.value
  if (!sheet) return
  savingSheet.value = true
  try {
    if (isGeminiSheet(sheet)) {
      await updateBillingSheet({
        model: sheet.model,
        layer,
        matrix: cloneMatrix(layer === 'official' ? officialMatrixDraft.value : saleMatrixDraft.value) || undefined
      })
    } else {
      await updateBillingSheet({
        model: sheet.model,
        layer,
        pricing:
          parseJSON<ModelCatalogPricing>(
            layer === 'official' ? officialDraft.value : saleDraft.value,
            t('admin.models.pages.billing.invalidJson')
          ) || {}
      })
    }
    appStore.showSuccess(t('admin.models.pages.billing.saveSheetSuccess'))
    await loadBillingCenter()
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, t('admin.models.pages.billing.saveSheetFailed')))
  } finally {
    savingSheet.value = false
  }
}

async function deleteSheetAction(layer: SheetLayer) {
  const sheet = layer === 'official' ? officialCurrentSheet.value : saleCurrentSheet.value
  if (!sheet) return
  savingSheet.value = true
  try {
    await deleteBillingSheet(sheet.model, layer)
    appStore.showSuccess(t('admin.models.pages.billing.deleteSheetSuccess'))
    await loadBillingCenter()
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, t('admin.models.pages.billing.deleteSheetFailed')))
  } finally {
    savingSheet.value = false
  }
}

async function copyOfficialToSaleAction() {
  if (!saleCurrentSheet.value) return
  savingSheet.value = true
  try {
    await copyBillingSheetOfficialToSale(saleCurrentSheet.value.model)
    appStore.showSuccess(t('admin.models.pages.billing.copyOfficialToSaleSuccess'))
    await loadBillingCenter()
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, t('admin.models.pages.billing.copyOfficialToSaleFailed')))
  } finally {
    savingSheet.value = false
  }
}

function selectRule(rule: BillingRule) {
  selectedRuleId.value = rule.id
  ruleDraft.value = JSON.stringify(rule, null, 2)
}

function resetRuleEditor() {
  selectedRuleId.value = ''
  ruleDraft.value = JSON.stringify({
    provider: 'gemini',
    layer: 'sale',
    surface: 'native',
    operation_type: 'generate_content',
    service_tier: '',
    batch_mode: 'any',
    matchers: {},
    unit: 'input_token',
    price: 0,
    priority: 0,
    enabled: true
  }, null, 2)
}

async function saveRule() {
  savingRule.value = true
  try {
    const payload = parseJSON<Partial<BillingRule>>(ruleDraft.value, t('admin.models.pages.billing.invalidJson'))
    await updateBillingRule({
      id: String(payload?.id || '').trim(),
      provider: String(payload?.provider || 'gemini'),
      layer: String(payload?.layer || 'sale'),
      surface: String(payload?.surface || 'native'),
      operation_type: String(payload?.operation_type || 'generate_content'),
      service_tier: String(payload?.service_tier || ''),
      batch_mode: String(payload?.batch_mode || 'any'),
      matchers: payload?.matchers || {},
      unit: String(payload?.unit || 'input_token'),
      price: Number(payload?.price || 0),
      priority: Number(payload?.priority || 0),
      enabled: payload?.enabled !== false
    })
    appStore.showSuccess(t('admin.models.pages.billing.saveRuleSuccess'))
    resetRuleEditor()
    await loadBillingCenter()
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, t('admin.models.pages.billing.saveRuleFailed')))
  } finally {
    savingRule.value = false
  }
}

async function deleteRuleAction() {
  if (!selectedRuleId.value) return
  savingRule.value = true
  try {
    await deleteBillingRule(selectedRuleId.value)
    appStore.showSuccess(t('admin.models.pages.billing.deleteRuleSuccess'))
    resetRuleEditor()
    await loadBillingCenter()
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, t('admin.models.pages.billing.deleteRuleFailed')))
  } finally {
    savingRule.value = false
  }
}

function resetSimulation() {
  const payload: BillingSimulationInput = {
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
      text_input_tokens: 1024,
      text_output_tokens: 512,
      audio_input_tokens: 0,
      audio_output_tokens: 0,
      cache_create_tokens: 0,
      cache_read_tokens: 0,
      cache_storage_token_hours: 0,
      image_outputs: 0,
      video_requests: 0,
      file_search_embedding_tokens: 0,
      file_search_retrieval_tokens: 0,
      grounding_search_queries: 0,
      grounding_maps_queries: 0
    }
  }
  simulationDraft.value = JSON.stringify(payload, null, 2)
  simulationResult.value = null
}

async function runSimulation() {
  simulating.value = true
  try {
    simulationResult.value = await simulateBilling(parseJSON<BillingSimulationInput>(simulationDraft.value, t('admin.models.pages.billing.invalidJson')) as BillingSimulationInput)
  } catch (error) {
    appStore.showError(resolveErrorMessage(error, t('admin.models.pages.billing.simulateFailed')))
  } finally {
    simulating.value = false
  }
}

function parseJSON<T>(raw: string, fallbackMessage: string): T | null {
  try {
    return JSON.parse(raw) as T
  } catch {
    throw new Error(fallbackMessage)
  }
}

function cloneMatrix(matrix?: GeminiBillingMatrix | null): GeminiBillingMatrix | null {
  if (!matrix) return null
  return JSON.parse(JSON.stringify(matrix)) as GeminiBillingMatrix
}

function isGeminiSheet(sheet: ModelBillingSheet): boolean {
  return sheet.provider === 'gemini'
}

function formatPricingPreview(pricing?: ModelCatalogPricing): string {
  if (!pricing) return '-'
  const values = [pricing.input_cost_per_token, pricing.output_cost_per_token, pricing.output_cost_per_image, pricing.output_cost_per_video_request].filter((value) => value !== undefined)
  return values.length ? values.map((value) => formatNumber(value)).join(' / ') : '-'
}

function formatSheetLayerPreview(sheet: ModelBillingSheet, layer: SheetLayer): string {
  if (isGeminiSheet(sheet)) {
    const matrix = layer === 'official' ? sheet.official_matrix : sheet.sale_matrix
    if (!matrix) return '-'
    return t('admin.models.pages.billing.matrixPreview', {
      rows: String(matrix.rows?.length || 0),
      cols: String(matrix.charge_slots?.length || 0)
    })
  }
  const pricing = layer === 'official' ? sheet.official_pricing : sheet.sale_pricing
  return formatPricingPreview(pricing)
}

function formatNumber(value?: number): string {
  if (value === undefined || value === null || Number.isNaN(value)) return '-'
  return new Intl.NumberFormat(undefined, { minimumFractionDigits: value === 0 ? 0 : 4, maximumFractionDigits: 8 }).format(value)
}

function humanize(value: string): string {
  const normalized = String(value || '').trim()
  if (!normalized) return t('admin.models.pages.billing.ruleValues.any')
  return normalized.replace(/_/g, ' ').replace(/\b\w/g, (letter) => letter.toUpperCase())
}

function resolveSurfaceLabel(surface: string): string {
  switch (surface) {
    case 'native':
      return t('admin.models.pages.billing.surfaces.native')
    case 'openai_compat':
      return t('admin.models.pages.billing.surfaces.compat')
    case 'live':
      return t('admin.models.pages.billing.surfaces.live')
    case 'interactions':
      return t('admin.models.pages.billing.surfaces.interactions')
    case 'vertex_existing':
      return t('admin.models.pages.billing.surfaces.vertex')
    default:
      return humanize(surface)
  }
}

function resolveErrorMessage(error: unknown, fallback: string): string {
  if (typeof error === 'object' && error !== null && 'message' in error) {
    const message = String((error as { message?: string }).message || '').trim()
    if (message) return message
  }
  return fallback
}
</script>
