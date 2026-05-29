import { computed } from 'vue'
import type { Translate } from './publicModelCatalogView'

export function usePublicModelDetailLabels(t: Translate) {
  const overviewLabels = computed(() => ({
    telemetry: t('ui.modelCatalog.detail.telemetry'),
    latency: t('ui.modelCatalog.card.latency'),
    weekSuccess: t('ui.modelCatalog.card.weekSuccess'),
    todaySuccess: t('ui.modelCatalog.card.todaySuccess'),
    pricing: t('ui.modelCatalog.detail.pricingTitle'),
    basePrice: t('ui.modelCatalog.detail.basePrice'),
    routePolicy: t('ui.modelCatalog.detail.routePolicy'),
    modalities: t('ui.modelCatalog.detail.modalities'),
    capabilities: t('ui.modelCatalog.detail.capabilities'),
    specs: t('ui.modelCatalog.detail.specs'),
    context: t('ui.modelCatalog.detail.context'),
    rateLimits: t('ui.modelCatalog.detail.rateLimits'),
  }))

  const monitorLabels = computed(() => ({
    status: t('ui.modelCatalog.detail.monitorStatus'),
    latency: t('ui.modelCatalog.card.latency'),
    todaySuccess: t('ui.modelCatalog.card.todaySuccess'),
    dailyMatrix: t('ui.modelCatalog.detail.dailyMatrix'),
    dailyMatrixCaption: t('ui.modelCatalog.detail.dailyMatrixCaption'),
    successRate: t('ui.modelCatalog.detail.successRate'),
    successTrend: t('ui.modelCatalog.detail.successTrend'),
    successTrendCaption: t('ui.modelCatalog.detail.successTrendCaption'),
    pending: t('ui.modelCatalog.health.pending'),
  }))

  const routingLabels = computed(() => ({
    exampleTitle: t('ui.modelCatalog.detail.exampleTitle'),
    authentication: t('ui.modelCatalog.detail.authentication'),
    authenticationText: t('ui.modelCatalog.detail.authenticationText'),
    rateLimits: t('ui.modelCatalog.detail.rateLimits'),
    group: t('ui.modelCatalog.detail.group'),
    defaultGroup: t('ui.modelCatalog.detail.defaultGroup'),
    parameters: t('ui.modelCatalog.detail.parameters'),
  }))

  const exampleLabels = computed(() => ({
    loading: t('ui.modelCatalog.detail.loading'),
    exampleSourceDocs: t('ui.modelCatalog.detail.exampleSourceDocs'),
    exampleSourceOverride: t('ui.modelCatalog.detail.exampleSourceOverride'),
    exampleUnavailable: t('ui.modelCatalog.detail.exampleUnavailable'),
  }))

  const parameterRows = computed(() => [
    { name: 'temperature', type: 'number', defaultValue: '= 1', description: t('ui.modelCatalog.detail.params.temperature') },
    { name: 'top_p', type: 'number', defaultValue: '= 1', description: t('ui.modelCatalog.detail.params.topP') },
    { name: 'max_tokens', type: 'integer', defaultValue: '-', description: t('ui.modelCatalog.detail.params.maxTokens') },
    { name: 'tools', type: 'array', defaultValue: '-', description: t('ui.modelCatalog.detail.params.tools') },
    { name: 'stream', type: 'boolean', defaultValue: '= false', description: t('ui.modelCatalog.detail.params.stream') },
  ])

  return { overviewLabels, monitorLabels, routingLabels, exampleLabels, parameterRows }
}
