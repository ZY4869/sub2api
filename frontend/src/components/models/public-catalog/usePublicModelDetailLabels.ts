import { computed } from 'vue'
import type { Translate } from './publicModelCatalogView'

export function usePublicModelDetailLabels(t: Translate) {
  const overviewLabels = computed(() => ({
    telemetry: t('ui.modelCatalog.detail.telemetry'),
    latency: t('ui.modelCatalog.card.latency'),
    weekSuccess: t('ui.modelCatalog.card.weekSuccess'),
    todaySuccess: t('ui.modelCatalog.card.todaySuccess'),
    pricing: t('ui.modelCatalog.detail.pricingTitle'),
    salePrice: t('ui.modelCatalog.detail.salePrice'),
    officialReferencePrice: t('ui.modelCatalog.detail.officialReferencePrice'),
    officialReferenceMissing: t('ui.modelCatalog.detail.officialReferenceMissing'),
    multiplierRules: t('ui.modelCatalog.detail.multiplierRules'),
    basePrice: t('ui.modelCatalog.detail.basePrice'),
    routePolicy: t('ui.modelCatalog.detail.routePolicy'),
    modalities: t('ui.modelCatalog.detail.modalities'),
    capabilities: t('ui.modelCatalog.detail.capabilities'),
    specs: t('ui.modelCatalog.detail.specs'),
    context: t('ui.modelCatalog.detail.context'),
    contextSource: t('ui.modelCatalog.detail.contextSource'),
    lifecycleSource: t('ui.modelCatalog.detail.lifecycleSource'),
    endpoints: t('ui.modelCatalog.detail.endpoints'),
    capabilityMatrix: t('ui.modelCatalog.detail.capabilityMatrix'),
    capability: t('ui.modelCatalog.detail.capability'),
    endpoint: t('ui.modelCatalog.detail.endpoint'),
    support: t('ui.modelCatalog.detail.support'),
    source: t('ui.modelCatalog.detail.source'),
    verified: t('ui.modelCatalog.detail.verified'),
    declared: t('ui.modelCatalog.detail.declared'),
    pricingSource: t('ui.modelCatalog.source.pricing'),
    snapshotSource: t('ui.modelCatalog.source.snapshot'),
    inferred: t('ui.modelCatalog.source.inferred'),
    supported: t('ui.modelCatalog.support.supported'),
    partial: t('ui.modelCatalog.support.partial'),
    unsupported: t('ui.modelCatalog.support.unsupported'),
    unknown: t('ui.modelCatalog.support.unknown'),
    demo: t('ui.modelCatalog.demo'),
    rateLimits: t('ui.modelCatalog.detail.rateLimits'),
    publishStatus: t('ui.modelCatalog.detail.publishStatus'),
    publishAvailability: t('ui.modelCatalog.detail.publishAvailability'),
    realtimeSource: t('ui.modelCatalog.detail.realtimeSource'),
    healthSourceTraffic: t('ui.modelCatalog.healthSource.traffic'),
    healthSourceProbe: t('ui.modelCatalog.healthSource.probe'),
    healthSourceNone: t('ui.modelCatalog.healthSource.none'),
    healthReasonTrafficRecent: t('ui.modelCatalog.healthReason.trafficRecent'),
    healthReasonProbeRecent: t('ui.modelCatalog.healthReason.probeRecent'),
    healthReasonMonitorDisabled: t('ui.modelCatalog.healthReason.monitorDisabled'),
    healthReasonNoHistory: t('ui.modelCatalog.healthReason.noHistory'),
    healthReasonStaleHistory: t('ui.modelCatalog.healthReason.staleHistory'),
    healthReasonChecking: t('ui.modelCatalog.healthReason.checking'),
    publishPublished: t('ui.modelCatalog.publishStatus.published'),
    publishLiveFallback: t('ui.modelCatalog.publishStatus.liveFallback'),
    publishUnknown: t('ui.modelCatalog.publishStatus.unknown'),
    perMillionTokens: t('ui.modelCatalog.units.perMillionTokens'),
    perImage: t('ui.modelCatalog.units.perImage'),
    perRequest: t('ui.modelCatalog.units.perRequest'),
    perVideo: t('ui.modelCatalog.units.perVideo'),
  }))

  const monitorLabels = computed(() => ({
    status: t('ui.modelCatalog.detail.monitorStatus'),
    latency: t('ui.modelCatalog.card.latency'),
    todaySuccess: t('ui.modelCatalog.card.todaySuccess'),
    dailyMatrix: t('ui.modelCatalog.detail.dailyMatrix'),
    dailyMatrixCaption: t('ui.modelCatalog.detail.dailyMatrixCaption'),
    dailyMatrixCaptionTraffic: t('ui.modelCatalog.detail.dailyMatrixCaptionTraffic'),
    dailyMatrixCaptionProbe: t('ui.modelCatalog.detail.dailyMatrixCaptionProbe'),
    successRate: t('ui.modelCatalog.detail.successRate'),
    successTrend: t('ui.modelCatalog.detail.successTrend'),
    successTrendCaption: t('ui.modelCatalog.detail.successTrendCaption'),
    successTrendCaptionTraffic: t('ui.modelCatalog.detail.successTrendCaptionTraffic'),
    successTrendCaptionProbe: t('ui.modelCatalog.detail.successTrendCaptionProbe'),
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
