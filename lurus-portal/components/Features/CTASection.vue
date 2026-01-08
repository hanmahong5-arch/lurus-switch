<script setup lang="ts">
import { Icon } from '@iconify/vue'
</script>

<template>
  <section class="py-24 lg:py-32 relative overflow-hidden">
    <!-- Background gradient -->
    <div class="absolute inset-0 bg-gradient-to-br from-primary-900/50 via-surface-900 to-accent-900/30" />

    <!-- Grid overlay -->
    <div class="absolute inset-0 bg-grid opacity-20" />

    <div class="relative z-10 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
      <h2
        v-motion
        :initial="{ opacity: 0, y: 20 }"
        :visibleOnce="{ opacity: 1, y: 0 }"
        class="text-3xl sm:text-4xl lg:text-5xl font-bold text-white mb-6"
      >
        Ready to get started?
      </h2>
      <p
        v-motion
        :initial="{ opacity: 0, y: 20 }"
        :visibleOnce="{ opacity: 1, y: 0, transition: { delay: 100 } }"
        class="text-lg text-surface-300 mb-10 max-w-2xl mx-auto"
      >
        Join thousands of developers building the next generation of AI applications.
        Start free, scale as you grow.
      </p>

      <div
        v-motion
        :initial="{ opacity: 0, y: 20 }"
        :visibleOnce="{ opacity: 1, y: 0, transition: { delay: 200 } }"
        class="flex flex-col sm:flex-row items-center justify-center gap-4"
      >
        <NuxtLink to="/products" class="btn-glow flex items-center gap-2">
          <Icon icon="lucide:rocket" class="w-5 h-5" />
          Start Building
        </NuxtLink>
        <NuxtLink to="/pricing" class="btn-outline flex items-center gap-2">
          <Icon icon="lucide:credit-card" class="w-5 h-5" />
          View Pricing
        </NuxtLink>
      </div>

      <!-- Trust badges -->
      <div
        v-motion
        :initial="{ opacity: 0 }"
        :visibleOnce="{ opacity: 1, transition: { delay: 400 } }"
        class="mt-12 flex flex-wrap items-center justify-center gap-6 text-surface-500 text-sm"
      >
        <div class="flex items-center gap-2">
          <Icon icon="lucide:shield-check" class="w-4 h-4" />
          <span>SOC 2 Compliant</span>
        </div>
        <div class="flex items-center gap-2">
          <Icon icon="lucide:lock" class="w-4 h-4" />
          <span>End-to-end Encrypted</span>
        </div>
        <div class="flex items-center gap-2">
          <Icon icon="lucide:clock" class="w-4 h-4" />
          <span>99.9% Uptime SLA</span>
        </div>
      </div>
    </div>
  </section>
</template>
