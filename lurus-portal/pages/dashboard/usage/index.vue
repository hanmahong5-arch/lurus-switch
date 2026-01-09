<script setup lang="ts">
import { Icon } from '@iconify/vue'
import { Line, Doughnut } from 'vue-chartjs'
import {
  Chart as ChartJS,
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler,
} from 'chart.js'

// Register Chart.js components
ChartJS.register(
  CategoryScale,
  LinearScale,
  PointElement,
  LineElement,
  ArcElement,
  Title,
  Tooltip,
  Legend,
  Filler
)

definePageMeta({
  layout: 'dashboard',
  middleware: ['auth'],
})

useSeoMeta({
  title: 'Usage Statistics - Lurus AI',
  description: 'View your AI usage statistics and analytics.',
})

const { quota, fetchQuota } = useQuota()

// Time range selector
const timeRange = ref<'7d' | '30d' | '90d'>('7d')

// Demo data for charts
const usageData = computed(() => {
  const days = timeRange.value === '7d' ? 7 : timeRange.value === '30d' ? 30 : 90
  const labels = []
  const data = []

  for (let i = days - 1; i >= 0; i--) {
    const date = new Date()
    date.setDate(date.getDate() - i)
    labels.push(date.toLocaleDateString('en-US', { month: 'short', day: 'numeric' }))
    data.push(Math.floor(Math.random() * 5000) + 500)
  }

  return { labels, data }
})

const lineChartData = computed(() => ({
  labels: usageData.value.labels,
  datasets: [
    {
      label: 'Tokens Used',
      data: usageData.value.data,
      borderColor: '#6366f1',
      backgroundColor: 'rgba(99, 102, 241, 0.1)',
      fill: true,
      tension: 0.4,
      pointRadius: 0,
      pointHoverRadius: 6,
      pointHoverBackgroundColor: '#6366f1',
      pointHoverBorderColor: '#fff',
      pointHoverBorderWidth: 2,
    },
  ],
}))

const lineChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      display: false,
    },
    tooltip: {
      backgroundColor: '#1e293b',
      titleColor: '#fff',
      bodyColor: '#94a3b8',
      borderColor: '#334155',
      borderWidth: 1,
      padding: 12,
      displayColors: false,
    },
  },
  scales: {
    x: {
      grid: {
        display: false,
      },
      ticks: {
        color: '#64748b',
        maxTicksLimit: 7,
      },
    },
    y: {
      grid: {
        color: '#1e293b',
      },
      ticks: {
        color: '#64748b',
        callback: (value: number) => {
          if (value >= 1000) return `${value / 1000}k`
          return value
        },
      },
    },
  },
  interaction: {
    intersect: false,
    mode: 'index' as const,
  },
}

// Model distribution data
const modelData = ref({
  labels: ['Claude 3 Opus', 'GPT-4 Turbo', 'Claude 3 Sonnet', 'Gemini Pro', 'Other'],
  datasets: [
    {
      data: [35, 28, 20, 12, 5],
      backgroundColor: [
        '#6366f1',
        '#22d3ee',
        '#a855f7',
        '#f59e0b',
        '#64748b',
      ],
      borderWidth: 0,
      hoverOffset: 4,
    },
  ],
})

const doughnutChartOptions = {
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: {
      position: 'right' as const,
      labels: {
        color: '#94a3b8',
        padding: 16,
        usePointStyle: true,
        pointStyle: 'circle',
      },
    },
    tooltip: {
      backgroundColor: '#1e293b',
      titleColor: '#fff',
      bodyColor: '#94a3b8',
      borderColor: '#334155',
      borderWidth: 1,
      padding: 12,
    },
  },
  cutout: '70%',
}

// Stats
const stats = computed(() => {
  const totalTokens = usageData.value.data.reduce((a, b) => a + b, 0)
  const avgDaily = Math.round(totalTokens / usageData.value.data.length)
  const totalCost = (totalTokens / 1000000) * 15 // Rough estimate

  return {
    totalTokens,
    avgDaily,
    totalCost,
    totalRequests: Math.round(totalTokens / 800), // Avg tokens per request
  }
})

onMounted(() => {
  fetchQuota()
})
</script>

<template>
  <div class="space-y-8">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-white">Usage Statistics</h1>
        <p class="text-surface-400 mt-1">Monitor your AI usage and costs</p>
      </div>

      <!-- Time range selector -->
      <div class="flex items-center gap-2 bg-surface-800 rounded-lg p-1">
        <button
          v-for="range in ['7d', '30d', '90d']"
          :key="range"
          class="px-4 py-2 rounded-md text-sm font-medium transition-colors"
          :class="timeRange === range ? 'bg-primary-500 text-white' : 'text-surface-400 hover:text-white'"
          @click="timeRange = range as '7d' | '30d' | '90d'"
        >
          {{ range === '7d' ? '7 Days' : range === '30d' ? '30 Days' : '90 Days' }}
        </button>
      </div>
    </div>

    <!-- Stats cards -->
    <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-6">
      <div class="glass-card">
        <div class="flex items-center gap-3 mb-2">
          <div class="w-10 h-10 rounded-lg bg-primary-500/20 flex items-center justify-center">
            <Icon icon="lucide:hash" class="w-5 h-5 text-primary-400" />
          </div>
          <span class="text-surface-400 text-sm">Total Tokens</span>
        </div>
        <p class="text-2xl font-bold text-white">{{ stats.totalTokens.toLocaleString() }}</p>
        <p class="text-xs text-surface-500 mt-1">Last {{ timeRange }}</p>
      </div>

      <div class="glass-card">
        <div class="flex items-center gap-3 mb-2">
          <div class="w-10 h-10 rounded-lg bg-accent-500/20 flex items-center justify-center">
            <Icon icon="lucide:activity" class="w-5 h-5 text-accent-400" />
          </div>
          <span class="text-surface-400 text-sm">Avg. Daily</span>
        </div>
        <p class="text-2xl font-bold text-white">{{ stats.avgDaily.toLocaleString() }}</p>
        <p class="text-xs text-surface-500 mt-1">Tokens per day</p>
      </div>

      <div class="glass-card">
        <div class="flex items-center gap-3 mb-2">
          <div class="w-10 h-10 rounded-lg bg-green-500/20 flex items-center justify-center">
            <Icon icon="lucide:send" class="w-5 h-5 text-green-400" />
          </div>
          <span class="text-surface-400 text-sm">Total Requests</span>
        </div>
        <p class="text-2xl font-bold text-white">{{ stats.totalRequests.toLocaleString() }}</p>
        <p class="text-xs text-surface-500 mt-1">API calls made</p>
      </div>

      <div class="glass-card">
        <div class="flex items-center gap-3 mb-2">
          <div class="w-10 h-10 rounded-lg bg-yellow-500/20 flex items-center justify-center">
            <Icon icon="lucide:dollar-sign" class="w-5 h-5 text-yellow-400" />
          </div>
          <span class="text-surface-400 text-sm">Est. Cost</span>
        </div>
        <p class="text-2xl font-bold text-white">${{ stats.totalCost.toFixed(2) }}</p>
        <p class="text-xs text-surface-500 mt-1">Last {{ timeRange }}</p>
      </div>
    </div>

    <!-- Charts -->
    <div class="grid lg:grid-cols-3 gap-6">
      <!-- Usage trend chart -->
      <div class="lg:col-span-2 glass-card">
        <h2 class="text-lg font-semibold text-white mb-6">Usage Trend</h2>
        <div class="h-80">
          <Line :data="lineChartData" :options="lineChartOptions" />
        </div>
      </div>

      <!-- Model distribution -->
      <div class="glass-card">
        <h2 class="text-lg font-semibold text-white mb-6">By Model</h2>
        <div class="h-80">
          <Doughnut :data="modelData" :options="doughnutChartOptions" />
        </div>
      </div>
    </div>

    <!-- Quota status -->
    <div class="glass-card">
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-lg font-semibold text-white">Current Quota Status</h2>
        <NuxtLink
          to="/dashboard/subscription"
          class="text-sm text-primary-400 hover:text-primary-300 transition-colors"
        >
          Upgrade Plan
        </NuxtLink>
      </div>

      <div class="grid sm:grid-cols-2 gap-8">
        <!-- Monthly quota -->
        <div>
          <div class="flex justify-between text-sm mb-2">
            <span class="text-surface-400">Monthly Quota</span>
            <span class="text-white">
              {{ quota.monthlyUsed.toLocaleString() }} / {{ quota.monthlyQuota.toLocaleString() }}
            </span>
          </div>
          <div class="w-full h-4 bg-surface-700 rounded-full overflow-hidden">
            <div
              class="h-full bg-gradient-to-r from-primary-500 to-accent-500 rounded-full transition-all duration-500"
              :style="{ width: `${Math.min((quota.monthlyUsed / quota.monthlyQuota) * 100, 100)}%` }"
            />
          </div>
          <p class="text-xs text-surface-500 mt-2">
            {{ quota.monthlyRemaining.toLocaleString() }} tokens remaining this month
          </p>
        </div>

        <!-- Daily quota -->
        <div>
          <div class="flex justify-between text-sm mb-2">
            <span class="text-surface-400">Daily Quota</span>
            <span class="text-white">
              {{ quota.dailyUsed }} / {{ quota.dailyQuota }}
            </span>
          </div>
          <div class="w-full h-4 bg-surface-700 rounded-full overflow-hidden">
            <div
              class="h-full bg-gradient-to-r from-accent-500 to-green-500 rounded-full transition-all duration-500"
              :style="{ width: `${Math.min((quota.dailyUsed / quota.dailyQuota) * 100, 100)}%` }"
            />
          </div>
          <p class="text-xs text-surface-500 mt-2">
            Resets daily at 00:00 UTC
          </p>
        </div>
      </div>
    </div>

    <!-- View history link -->
    <div class="text-center">
      <NuxtLink
        to="/dashboard/usage/history"
        class="inline-flex items-center gap-2 text-primary-400 hover:text-primary-300 transition-colors"
      >
        <span>View detailed history</span>
        <Icon icon="lucide:arrow-right" class="w-4 h-4" />
      </NuxtLink>
    </div>
  </div>
</template>
