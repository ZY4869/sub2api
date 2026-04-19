<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  Chart as ChartJS,
  CategoryScale,
  Filler,
  Legend,
  LineElement,
  LinearScale,
  PointElement,
  Title,
  Tooltip,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import type { OpsRequestSubjectHistoryPoint } from '@/api/admin/ops'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Title, Tooltip, Legend, Filler)

const { t } = useI18n()

const props = defineProps<{
  history: OpsRequestSubjectHistoryPoint[]
  loading?: boolean
}>()

const chartData = computed(() => {
  if (!props.history.length) return null
  return {
    labels: props.history.map((point) => point.label || point.date),
    datasets: [
      {
        label: t('usage.accountBilled'),
        data: props.history.map((point) => point.actual_cost),
        borderColor: '#2563eb',
        backgroundColor: 'rgba(37, 99, 235, 0.08)',
        fill: true,
        tension: 0.3,
        yAxisID: 'y'
      },
      {
        label: t('usage.userBilled'),
        data: props.history.map((point) => point.user_cost),
        borderColor: '#10b981',
        backgroundColor: 'rgba(16, 185, 129, 0.08)',
        fill: false,
        tension: 0.3,
        borderDash: [6, 4],
        yAxisID: 'y'
      },
      {
        label: t('admin.requestDetails.subject.summary.totalRequests'),
        data: props.history.map((point) => point.requests),
        borderColor: '#f97316',
        backgroundColor: 'rgba(249, 115, 22, 0.08)',
        fill: false,
        tension: 0.3,
        yAxisID: 'y1'
      }
    ]
  }
})

const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: {
    intersect: false,
    mode: 'index' as const
  },
  plugins: {
    legend: {
      position: 'top' as const
    }
  },
  scales: {
    x: {
      ticks: {
        maxRotation: 45,
        minRotation: 0
      }
    },
    y: {
      type: 'linear' as const,
      position: 'left' as const,
      title: {
        display: true,
        text: 'USD'
      }
    },
    y1: {
      type: 'linear' as const,
      position: 'right' as const,
      grid: {
        drawOnChartArea: false
      },
      title: {
        display: true,
        text: t('admin.requestDetails.subject.summary.totalRequests')
      }
    }
  }
}))
</script>

<template>
  <section class="rounded-3xl border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-800">
    <div class="mb-4">
      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
        {{ t('admin.requestDetails.subject.trend.title') }}
      </h3>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
        {{ t('admin.requestDetails.subject.trend.description') }}
      </p>
    </div>
    <div v-if="loading" class="flex h-72 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('common.loading') }}
    </div>
    <div v-else-if="chartData" class="h-72">
      <Line :data="chartData" :options="chartOptions" />
    </div>
    <div v-else class="flex h-72 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('common.noData') }}
    </div>
  </section>
</template>
