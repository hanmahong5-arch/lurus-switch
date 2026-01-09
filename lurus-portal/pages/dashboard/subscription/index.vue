<script setup lang="ts">
import { Icon } from '@iconify/vue'

definePageMeta({
  layout: 'dashboard',
  middleware: ['auth'],
})

useSeoMeta({
  title: 'Subscription - Lurus AI',
  description: 'Manage your Lurus AI subscription and billing.',
})

const { quota } = useQuota()
const { newApiUser } = useAuth()

const currentPlan = computed(() => {
  const group = newApiUser.value?.group || quota.value.currentGroup
  return plans.find(p => p.group === group) || plans[0]
})

const plans = [
  {
    name: 'Free',
    group: 'free',
    price: 0,
    period: 'forever',
    monthlyQuota: 500,
    dailyQuota: 50,
    features: [
      'Access to all models',
      '500 tokens/month',
      '50 tokens/day limit',
      'Community support',
    ],
    cta: 'Current Plan',
    popular: false,
  },
  {
    name: 'Basic',
    group: 'basic',
    price: 29,
    period: '/month',
    monthlyQuota: 2900,
    dailyQuota: 100,
    features: [
      'Everything in Free',
      '2,900 tokens/month',
      '100 tokens/day limit',
      'Priority support',
      'Usage analytics',
    ],
    cta: 'Upgrade',
    popular: false,
  },
  {
    name: 'Pro',
    group: 'pro',
    price: 99,
    period: '/month',
    monthlyQuota: 9900,
    dailyQuota: 330,
    features: [
      'Everything in Basic',
      '9,900 tokens/month',
      '330 tokens/day limit',
      'Premium models access',
      'Dedicated support',
      'Custom integrations',
    ],
    cta: 'Upgrade',
    popular: true,
  },
  {
    name: 'Enterprise',
    group: 'premium',
    price: 299,
    period: '/month',
    monthlyQuota: 29900,
    dailyQuota: 1000,
    features: [
      'Everything in Pro',
      '29,900 tokens/month',
      '1,000 tokens/day limit',
      'Unlimited team members',
      'SLA guarantee',
      'Custom contract',
    ],
    cta: 'Contact Sales',
    popular: false,
  },
]

const billingHistory = ref([
  { id: 1, date: '2026-01-01', amount: 99, status: 'paid', plan: 'Pro Monthly' },
  { id: 2, date: '2025-12-01', amount: 99, status: 'paid', plan: 'Pro Monthly' },
  { id: 3, date: '2025-11-01', amount: 99, status: 'paid', plan: 'Pro Monthly' },
])

const selectedPlan = ref<string | null>(null)

const handleUpgrade = (planGroup: string) => {
  selectedPlan.value = planGroup
  // In production, this would redirect to payment
}
</script>

<template>
  <div class="space-y-8">
    <!-- Header -->
    <div>
      <h1 class="text-2xl font-bold text-white">Subscription</h1>
      <p class="text-surface-400 mt-1">Manage your plan and billing</p>
    </div>

    <!-- Current plan card -->
    <div class="glass-card">
      <div class="flex flex-col lg:flex-row lg:items-center lg:justify-between gap-6">
        <div>
          <div class="flex items-center gap-3 mb-2">
            <h2 class="text-xl font-semibold text-white">{{ currentPlan.name }} Plan</h2>
            <span
              v-if="quota.isFallback"
              class="px-2 py-1 text-xs rounded-full bg-yellow-500/20 text-yellow-400"
            >
              Temporarily downgraded
            </span>
          </div>
          <p class="text-surface-400">
            {{ currentPlan.monthlyQuota.toLocaleString() }} tokens/month,
            {{ currentPlan.dailyQuota }} tokens/day
          </p>
        </div>

        <div class="flex items-center gap-6">
          <div class="text-right">
            <p class="text-3xl font-bold text-white">
              ${{ currentPlan.price }}
              <span class="text-lg font-normal text-surface-400">{{ currentPlan.period }}</span>
            </p>
            <p v-if="currentPlan.price > 0" class="text-sm text-surface-400">
              Next billing: Feb 1, 2026
            </p>
          </div>
          <NuxtLink
            v-if="currentPlan.group !== 'premium'"
            to="#plans"
            class="btn-glow"
          >
            Upgrade Plan
          </NuxtLink>
        </div>
      </div>

      <!-- Usage bars -->
      <div class="grid sm:grid-cols-2 gap-6 mt-8 pt-6 border-t border-surface-700">
        <div>
          <div class="flex justify-between text-sm mb-2">
            <span class="text-surface-400">Monthly Usage</span>
            <span class="text-white">
              {{ quota.monthlyUsed.toLocaleString() }} / {{ quota.monthlyQuota.toLocaleString() }}
            </span>
          </div>
          <div class="w-full h-2 bg-surface-700 rounded-full overflow-hidden">
            <div
              class="h-full bg-primary-500 rounded-full"
              :style="{ width: `${Math.min((quota.monthlyUsed / quota.monthlyQuota) * 100, 100)}%` }"
            />
          </div>
        </div>
        <div>
          <div class="flex justify-between text-sm mb-2">
            <span class="text-surface-400">Daily Usage</span>
            <span class="text-white">
              {{ quota.dailyUsed }} / {{ quota.dailyQuota }}
            </span>
          </div>
          <div class="w-full h-2 bg-surface-700 rounded-full overflow-hidden">
            <div
              class="h-full bg-accent-500 rounded-full"
              :style="{ width: `${Math.min((quota.dailyUsed / quota.dailyQuota) * 100, 100)}%` }"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- Plans comparison -->
    <div id="plans">
      <h2 class="text-xl font-semibold text-white mb-6">Available Plans</h2>

      <div class="grid md:grid-cols-2 xl:grid-cols-4 gap-6">
        <div
          v-for="plan in plans"
          :key="plan.name"
          class="glass-card relative"
          :class="plan.popular ? 'ring-2 ring-primary-500' : ''"
        >
          <!-- Popular badge -->
          <div
            v-if="plan.popular"
            class="absolute -top-3 left-1/2 -translate-x-1/2 px-3 py-1 bg-primary-500 text-white text-xs font-medium rounded-full"
          >
            Most Popular
          </div>

          <div class="text-center mb-6">
            <h3 class="text-lg font-semibold text-white mb-2">{{ plan.name }}</h3>
            <p class="text-3xl font-bold text-white">
              ${{ plan.price }}
              <span class="text-base font-normal text-surface-400">{{ plan.period }}</span>
            </p>
          </div>

          <ul class="space-y-3 mb-6">
            <li
              v-for="feature in plan.features"
              :key="feature"
              class="flex items-start gap-2 text-sm text-surface-300"
            >
              <Icon icon="lucide:check" class="w-4 h-4 text-accent-400 mt-0.5 flex-shrink-0" />
              <span>{{ feature }}</span>
            </li>
          </ul>

          <button
            v-if="plan.group === currentPlan.group"
            class="w-full py-3 px-4 bg-surface-700 text-surface-400 rounded-lg cursor-not-allowed"
            disabled
          >
            Current Plan
          </button>
          <button
            v-else-if="plan.group === 'premium'"
            class="w-full py-3 px-4 bg-surface-700 text-white rounded-lg hover:bg-surface-600 transition-colors"
            @click="handleUpgrade(plan.group)"
          >
            Contact Sales
          </button>
          <button
            v-else
            class="w-full btn-glow py-3"
            @click="handleUpgrade(plan.group)"
          >
            {{ plan.price > currentPlan.price ? 'Upgrade' : 'Downgrade' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Billing history -->
    <div class="glass-card">
      <div class="flex items-center justify-between mb-6">
        <h2 class="text-lg font-semibold text-white">Billing History</h2>
        <button class="text-sm text-primary-400 hover:text-primary-300 transition-colors">
          Download All
        </button>
      </div>

      <div class="overflow-x-auto">
        <table class="w-full">
          <thead>
            <tr class="text-left text-surface-400 text-sm border-b border-surface-700">
              <th class="pb-3 font-medium">Date</th>
              <th class="pb-3 font-medium">Plan</th>
              <th class="pb-3 font-medium">Amount</th>
              <th class="pb-3 font-medium">Status</th>
              <th class="pb-3 font-medium text-right">Invoice</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="invoice in billingHistory"
              :key="invoice.id"
              class="border-b border-surface-700/50 last:border-0"
            >
              <td class="py-3 text-white">{{ invoice.date }}</td>
              <td class="py-3 text-surface-300">{{ invoice.plan }}</td>
              <td class="py-3 text-white">${{ invoice.amount }}</td>
              <td class="py-3">
                <span class="px-2 py-1 text-xs rounded-full bg-green-500/20 text-green-400 capitalize">
                  {{ invoice.status }}
                </span>
              </td>
              <td class="py-3 text-right">
                <button class="text-primary-400 hover:text-primary-300 transition-colors">
                  <Icon icon="lucide:download" class="w-4 h-4" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- FAQ -->
    <div class="glass-card">
      <h2 class="text-lg font-semibold text-white mb-6">Frequently Asked Questions</h2>

      <div class="space-y-4">
        <details class="group">
          <summary class="flex items-center justify-between cursor-pointer text-white font-medium py-2">
            <span>What happens when I exceed my daily quota?</span>
            <Icon icon="lucide:chevron-down" class="w-5 h-5 text-surface-400 group-open:rotate-180 transition-transform" />
          </summary>
          <p class="text-surface-400 pb-4">
            When you exceed your daily quota, you'll be temporarily downgraded to a lower tier group.
            Your access will be restored at midnight (00:00 UTC). Your monthly quota remains unaffected.
          </p>
        </details>

        <details class="group">
          <summary class="flex items-center justify-between cursor-pointer text-white font-medium py-2">
            <span>Can I upgrade or downgrade at any time?</span>
            <Icon icon="lucide:chevron-down" class="w-5 h-5 text-surface-400 group-open:rotate-180 transition-transform" />
          </summary>
          <p class="text-surface-400 pb-4">
            Yes, you can change your plan at any time. Upgrades take effect immediately,
            while downgrades will apply at the start of your next billing cycle.
          </p>
        </details>

        <details class="group">
          <summary class="flex items-center justify-between cursor-pointer text-white font-medium py-2">
            <span>What payment methods do you accept?</span>
            <Icon icon="lucide:chevron-down" class="w-5 h-5 text-surface-400 group-open:rotate-180 transition-transform" />
          </summary>
          <p class="text-surface-400 pb-4">
            We accept all major credit cards, Alipay, and WeChat Pay. Enterprise customers
            can also pay via bank transfer or invoice.
          </p>
        </details>
      </div>
    </div>
  </div>
</template>
