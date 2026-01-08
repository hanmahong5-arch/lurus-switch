<script setup lang="ts">
import { Icon } from '@iconify/vue'

useSeoMeta({
  title: 'Pricing - Lurus AI',
  description: 'Simple, transparent pricing. Start free and scale as you grow.',
})

const plans = [
  {
    name: 'Free',
    price: '0',
    period: 'forever',
    description: 'Perfect for trying out and personal projects',
    features: [
      '100K tokens/month',
      'Basic providers (OpenAI, Claude)',
      'Community support',
      'Basic analytics',
      'Single user',
    ],
    cta: 'Get Started',
    href: '/login',
    popular: false,
  },
  {
    name: 'Pro',
    price: '29',
    period: 'month',
    description: 'For developers and small teams',
    features: [
      '1M tokens/month',
      'All 40+ providers',
      'Priority support',
      'Advanced analytics',
      'Up to 5 team members',
      'Custom model routing',
      'API rate limiting',
    ],
    cta: 'Start Free Trial',
    href: '/login?plan=pro',
    popular: true,
  },
  {
    name: 'Enterprise',
    price: 'Custom',
    period: '',
    description: 'For large organizations with custom needs',
    features: [
      'Unlimited tokens',
      'All features included',
      'Dedicated support',
      'Custom SLA',
      'Unlimited team members',
      'On-premise deployment',
      'Custom integrations',
      'SSO / SAML',
    ],
    cta: 'Contact Sales',
    href: '/contact',
    popular: false,
  },
]

const faqs = [
  {
    question: 'How does token counting work?',
    answer: 'Tokens are counted based on the actual usage from the underlying AI provider. Input and output tokens are both counted towards your monthly limit.',
  },
  {
    question: 'Can I switch plans anytime?',
    answer: 'Yes, you can upgrade or downgrade your plan at any time. Changes take effect immediately, and billing is prorated.',
  },
  {
    question: 'What happens if I exceed my token limit?',
    answer: 'You will receive warnings as you approach your limit. Once reached, API calls will be rate-limited until the next billing cycle or until you upgrade.',
  },
  {
    question: 'Do you offer educational or non-profit discounts?',
    answer: 'Yes! Contact us for special pricing for educational institutions, non-profits, and open-source projects.',
  },
]
</script>

<template>
  <div class="min-h-screen pt-24 lg:pt-32">
    <!-- Hero -->
    <section class="relative py-16 lg:py-24 overflow-hidden">
      <div class="absolute inset-0 bg-grid opacity-20" />
      <div
        class="gradient-orb w-[500px] h-[500px] -top-40 left-1/2 -translate-x-1/2"
        style="--orb-color: #6366f1"
      />

      <div class="relative z-10 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <h1
          v-motion
          :initial="{ opacity: 0, y: 20 }"
          :enter="{ opacity: 1, y: 0 }"
          class="text-4xl sm:text-5xl lg:text-6xl font-bold text-white mb-6"
        >
          Simple, transparent
          <span class="gradient-text">pricing</span>
        </h1>
        <p
          v-motion
          :initial="{ opacity: 0, y: 20 }"
          :enter="{ opacity: 1, y: 0, transition: { delay: 100 } }"
          class="text-lg text-surface-300"
        >
          Start free and scale as you grow. No hidden fees, no surprises.
        </p>
      </div>
    </section>

    <!-- Pricing cards -->
    <section class="py-16 lg:py-24">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid md:grid-cols-3 gap-8">
          <div
            v-for="(plan, index) in plans"
            :key="plan.name"
            v-motion
            :initial="{ opacity: 0, y: 30 }"
            :visibleOnce="{ opacity: 1, y: 0, transition: { delay: index * 100 } }"
            class="relative glass-card"
            :class="{ 'border-primary-500/50 shadow-lg shadow-primary-500/10': plan.popular }"
          >
            <!-- Popular badge -->
            <div
              v-if="plan.popular"
              class="absolute -top-4 left-1/2 -translate-x-1/2 px-4 py-1 rounded-full bg-gradient-to-r from-primary-500 to-primary-600 text-white text-sm font-medium"
            >
              Most Popular
            </div>

            <!-- Plan header -->
            <div class="text-center mb-8">
              <h3 class="text-xl font-semibold text-white mb-2">{{ plan.name }}</h3>
              <div class="mb-2">
                <span v-if="plan.price !== 'Custom'" class="text-4xl font-bold text-white">${{ plan.price }}</span>
                <span v-else class="text-4xl font-bold text-white">{{ plan.price }}</span>
                <span v-if="plan.period" class="text-surface-400">/{{ plan.period }}</span>
              </div>
              <p class="text-surface-400 text-sm">{{ plan.description }}</p>
            </div>

            <!-- Features -->
            <ul class="space-y-3 mb-8">
              <li
                v-for="feature in plan.features"
                :key="feature"
                class="flex items-center gap-3 text-surface-300"
              >
                <Icon icon="lucide:check" class="w-5 h-5 text-accent-400 flex-shrink-0" />
                <span>{{ feature }}</span>
              </li>
            </ul>

            <!-- CTA -->
            <NuxtLink
              :to="plan.href"
              class="block w-full text-center py-3 rounded-lg font-semibold transition-all duration-200"
              :class="plan.popular
                ? 'btn-glow'
                : 'border border-surface-600 text-surface-300 hover:border-primary-500 hover:text-primary-400'"
            >
              {{ plan.cta }}
            </NuxtLink>
          </div>
        </div>
      </div>
    </section>

    <!-- FAQ -->
    <section class="py-16 lg:py-24 border-t border-surface-800">
      <div class="max-w-3xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="text-center mb-12">
          <h2 class="text-3xl font-bold text-white mb-4">
            Frequently Asked Questions
          </h2>
          <p class="text-surface-400">
            Have a question? We've got answers.
          </p>
        </div>

        <div class="space-y-6">
          <div
            v-for="(faq, index) in faqs"
            :key="index"
            v-motion
            :initial="{ opacity: 0, y: 20 }"
            :visibleOnce="{ opacity: 1, y: 0, transition: { delay: index * 100 } }"
            class="glass-card"
          >
            <h3 class="text-lg font-semibold text-white mb-2">
              {{ faq.question }}
            </h3>
            <p class="text-surface-400">
              {{ faq.answer }}
            </p>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
