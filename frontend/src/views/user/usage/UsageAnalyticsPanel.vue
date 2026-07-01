<template>
  <section data-testid="usage-analytics-panel" class="space-y-6">
    <div class="grid grid-cols-1 gap-6 xl:grid-cols-2">
      <ModelDistributionChart
        v-model:metric="modelMetric"
        :model-stats="modelStats"
        :loading="modelLoading"
        :show-metric-toggle="true"
        :enable-breakdown="false"
      />
      <GroupDistributionChart
        v-model:metric="groupMetric"
        :group-stats="groupStats"
        :loading="groupLoading"
        :show-metric-toggle="true"
        :enable-breakdown="false"
        :start-date="startDate"
        :end-date="endDate"
      />
    </div>
    <div class="grid grid-cols-1 gap-6 xl:grid-cols-2">
      <EndpointDistributionChart
        v-model:metric="endpointMetric"
        v-model:source="endpointSource"
        :endpoint-stats="endpointStats"
        :upstream-endpoint-stats="upstreamEndpointStats"
        :loading="endpointLoading"
        :show-metric-toggle="true"
        :show-source-toggle="true"
        :enable-breakdown="false"
        :start-date="startDate"
        :end-date="endDate"
      />
      <TokenUsageTrend :trend-data="trendData" :loading="trendLoading" />
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref } from "vue";
import ModelDistributionChart from "@/components/charts/ModelDistributionChart.vue";
import GroupDistributionChart from "@/components/charts/GroupDistributionChart.vue";
import EndpointDistributionChart from "@/components/charts/EndpointDistributionChart.vue";
import TokenUsageTrend from "@/components/charts/TokenUsageTrend.vue";
import type { EndpointStat, GroupStat, ModelStat, TrendDataPoint } from "@/types";

type DistributionMetric = "tokens" | "actual_cost";
type EndpointSource = "inbound" | "upstream" | "path";

defineProps<{
  trendData: TrendDataPoint[];
  modelStats: ModelStat[];
  groupStats: GroupStat[];
  endpointStats: EndpointStat[];
  upstreamEndpointStats: EndpointStat[];
  trendLoading?: boolean;
  modelLoading?: boolean;
  groupLoading?: boolean;
  endpointLoading?: boolean;
  startDate: string;
  endDate: string;
}>();

const modelMetric = ref<DistributionMetric>("tokens");
const groupMetric = ref<DistributionMetric>("tokens");
const endpointMetric = ref<DistributionMetric>("tokens");
const endpointSource = ref<EndpointSource>("inbound");
</script>
