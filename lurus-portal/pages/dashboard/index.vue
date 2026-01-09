<script setup lang="ts">
import { Icon } from '@iconify/vue'

definePageMeta({
  layout: 'dashboard',
  middleware: ['auth'],
})

useSeoMeta({
  title: 'Dashboard - Lurus AI',
  description: 'Monitor your AI usage and manage your account.',
})

const { profile, newApiUser } = useAuth()
const {
  quota,
  recentUsage,
  loading,
  connected,
  dailyPercentage,
  monthlyPercentage,
  isQuotaLow,
  isDailyExhausted,
  fetchQuota,
  fetchRecentUsage,
  connectRealtime,
} = useQuota()

// Initialize data
onMounted(async () => {
  await fetchQuota()
  await fetchRecentUsage(5)
  connectRealtime()
})

// Helper functions
const getProgressColor = (percentage: number) => {
  if (percentage >= 90) return 'bg-red-500'
  if (percentage >= 70) return 'bg-yellow-500'
  return 'bg-accent-500'
}

const getTextColor = (percentage: number) => {
  if (percentage >= 90) return 'text-red-400'
  if (percentage >= 70) return 'text-yellow-400'
  return 'text-accent-400'
}

const formatCost = (cost: number) => {
  return cost < 0.01 ? `$${cost.toFixed(4)}` : `$${cost.toFixed(2)}`
}

const quickActions = [
  { name: 'View Usage', href: '/dashboard/usage', icon: 'lucide:bar-chart-3', color: 'from-blue-500 to-cyan-500' },
  { name: 'API Keys', href: '/dashboard/api-keys', icon: 'lucide:key', color: 'from-purple-500 to-pink-500' },
  { name: 'Upgrade Plan', href: '/dashboard/subscription', icon: 'lucide:rocket', color: 'from-orange-500 to-red-500' },
  { name: 'Read Docs', href: '/docs', icon: 'lucide:book-open', color: 'from-green-500 to-emerald-500' },
]
</script>

<template>
  <div class="space-y-8">
    <!-- Welcome header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-white">
          Welcome back, {{ profile?.display_name || 'Developer' }}
        </h1>
        <p class="text-surface-400 mt-1">
          Here's an overview of your AI usage and account status.
        </p>
      </div>
      <div class="flex items-center gap-2">
        <span
          class="flex items-center gap-2 px-3 py-1.5 rounded-full text-sm"
          :class="connected ? 'bg-green-500/20 text-green-400' : 'bg-surface-700 text-surface-400'"
        >
          <span class="w-2 h-2 rounded-full" :class="connected ? 'bg-green-400 animate-pulse' : 'bg-surface-500'" />
          {{ connected ? 'Live' : 'Offline' }}
        </span>
      </div>
    </div>

    <!-- Alert banners -->
    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
    >
      <div
        v-if="isQuotaLow"
        class="p-4 rounded-lg bg-yellow-500/10 border border-yellow-500/20 flex items-center gap-3"
      >
        <Icon icon="lucide:alert-triangle" class="w-5 h-5 text-yellow-400 flex-shrink-0" />
        <div class="flex-1">
          <p class="text-sm text-yellow-400">
            Your monthly quota is running low ({{ monthlyPercentage }}% used).
            <NuxtLink to="/dashboard/subscription" class="underline hover:no-underline">
              Upgrade your plan
            </NuxtLink>
            to avoid service interruption.
          </p>
        </div>
      </div>
    </Transition>

    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
    >
      <div
        v-if="isDailyExhausted && quota.isFallback"
        class="p-4 rounded-lg bg-orange-500/10 border border-orange-500/20 flex items-center gap-3"
      >
        <Icon icon="lucide:info" class="w-5 h-5 text-orange-400 flex-shrink-0" />
        <div class="flex-1">
          <p class="text-sm text-orange-400">
            Daily quota exceeded. You've been temporarily moved to the
            <strong>{{ quota.currentGroup }}</strong> group. Full access will be restored at midnight.
          </p>
        </div>
      </div>
    </Transition>

    <!-- Quota cards -->
    <div class="grid lg:grid-cols-3 gap-6">
      <!-- Account Status Card -->
      <div class="glass-card">
        <div class="flex items-center gap-3 mb-6">
          <div class="w-12 h-12 rounded-full bg-gradient-to-r from-primary-500 to-accent-500 flex items-center justify-center">
            <Icon icon="lucide:user" class="w-6 h-6 text-white" />
          </div>
          <div>
            <h3 class="text-lg font-semibold text-white">Account Status</h3>
            <p class="text-surface-400 text-sm">{{ newApiUser?.group || 'Free' }} Plan</p>
          </div>
        </div>

        <div class="space-y-4">
          <div class="flex justify-between items-center">
            <span class="text-surface-400">Current Group</span>
            <span
              class="px-2 py-1 rounded text-sm font-medium"
              :class="quota.isFallback ? 'bg-yellow-500/20 text-yellow-400' : 'bg-accent-500/20 text-accent-400'"
            >
              {{ quota.currentGroup }}
              <span v-if="quota.isFallback" class="text-xs">(fallback)</span>
            </span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-surface-400">Balance</span>
            <span class="text-white font-semibold">${{ quota.balance.toFixed(2) }}</span>
          </div>
          <div class="flex justify-between items-center">
            <span class="text-surface-400">API Status</span>
            <span
              class="px-2 py-1 rounded text-sm font-medium"
              :class="quota.allowed ? 'bg-green-500/20 text-green-400' : 'bg-red-500/20 text-red-400'"
            >
              {{ quota.allowed ? 'Active' : 'Blocked' }}
            </span>
          </div>
        </div>
      </div>

      <!-- Monthly Quota Card -->
      <div class="glass-card">
        <h3 class="text-lg font-semibold text-white mb-4">Monthly Quota</h3>

        <div class="mb-4">
          <div class="flex justify-between text-sm mb-2">
            <span class="text-surface-400">Used</span>
            <span class="text-white">
              {{ quota.monthlyUsed.toLocaleString() }} / {{ quota.monthlyQuota.toLocaleString() }}
            </span>
          </div>
          <div class="w-full h-3 bg-surface-700 rounded-full overflow-hidden">
            <div
              class="h-full rounded-full transition-all duration-500"
              :class="getProgressColor(monthlyPercentage)"
              :style="{ width: `${Math.min(monthlyPercentage, 100)}%` }"
            />
          </div>
        </div>

        <div class="text-center">
          <span class="text-4xl font-bold" :class="getTextColor(monthlyPercentage)">
            {{ monthlyPercentage }}%
          </span>
          <p class="text-surface-400 text-sm">of monthly quota used</p>
        </div>
      </div>

      <!-- Daily Quota Card -->
      <div class="glass-card">
        <h3 class="text-lg font-semibold text-white mb-4">Daily Quota</h3>

        <div class="mb-4">
          <div class="flex justify-between text-sm mb-2">
            <span class="text-surface-400">Today</span>
            <span class="text-white">
              {{ quota.dailyUsed }} / {{ quota.dailyQuota }}
            </span>
          </div>
          <div class="w-full h-3 bg-surface-700 rounded-full overflow-hidden">
            <div
              class="h-full bg-primary-500 rounded-full transition-all duration-500"
              :style="{ width: `${Math.min(dailyPercentage, 100)}%` }"
            />
          </div>
        </div>

        <div class="text-center">
          <span class="text-4xl font-bold text-primary-400">
            {{ Math.max(quota.dailyRemaining, 0) }}
          </span>
          <p class="text-surface-400 text-sm">remaining (resets at 00:00)</p>
        </div>
      </div>
    </div>

    <!-- Quick Actions -->
    <div>
      <h2 class="text-lg font-semibold text-white mb-4">Quick Actions</h2>
      <div class="grid sm:grid-cols-2 lg:grid-cols-4 gap-4">
        <NuxtLink
          v-for="action in quickActions"
          :key="action.name"
          :to="action.href"
          class="glass-card group hover:scale-105 transition-transform duration-200"
        >
          <div
            class="w-12 h-12 rounded-lg flex items-center justify-center mb-4"
            :class="`bg-gradient-to-br ${action.color}`"
          >
            <Icon :icon="action.icon" class="w-6 h-6 text-white" />
          </div>
          <h3 class="text-white font-medium group-hover:text-primary-400 transition-colors">
            {{ action.name }}
          </h3>
        </NuxtLink>
      </div>
    </div>

    <!-- Recent Usage -->
    <div class="glass-card">
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-lg font-semibold text-white">Recent Usage</h2>
        <NuxtLink
          to="/dashboard/usage"
          class="text-sm text-primary-400 hover:text-primary-300 transition-colors"
        >
          View all
        </NuxtLink>
      </div>

      <div v-if="loading" class="flex items-center justify-center py-8">
        <Icon icon="lucide:loader-2" class="w-8 h-8 text-surface-400 animate-spin" />
      </div>

      <div v-else-if="recentUsage.length === 0" class="text-center py-8">
        <Icon icon="lucide:inbox" class="w-12 h-12 text-surface-600 mx-auto mb-3" />
        <p class="text-surface-400">No usage records yet</p>
        <p class="text-surface-500 text-sm mt-1">Start making API calls to see your usage here</p>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead>
            <tr class="text-left text-surface-400 text-sm border-b border-surface-700">
              <th class="pb-3 font-medium">Time</th>
              <th class="pb-3 font-medium">Model</th>
              <th class="pb-3 font-medium hidden sm:table-cell">Provider</th>
              <th class="pb-3 font-medium text-right">Tokens</th>
              <th class="pb-3 font-medium text-right">Cost</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="usage in recentUsage"
              :key="usage.id"
              class="border-b border-surface-700/50 last:border-0"
            >
              <td class="py-3 text-surface-300">{{ usage.time }}</td>
              <td class="py-3">
                <span class="px-2 py-1 rounded bg-surface-700 text-surface-300 text-sm">
                  {{ usage.model }}
                </span>
              </td>
              <td class="py-3 text-surface-400 hidden sm:table-cell">{{ usage.provider }}</td>
              <td class="py-3 text-right text-white">
                {{ (usage.inputTokens + usage.outputTokens).toLocaleString() }}
              </td>
              <td class="py-3 text-right text-accent-400">{{ formatCost(usage.cost) }}</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
</template>
