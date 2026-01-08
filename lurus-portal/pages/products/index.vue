<script setup lang="ts">
import { Icon } from '@iconify/vue'

useSeoMeta({
  title: 'Products - Lurus AI',
  description: 'Explore Lurus AI products: CodeSwitch desktop app and Gateway unified API service.',
})

const products = [
  {
    id: 'codeswitch',
    name: 'CodeSwitch',
    tagline: 'AI Provider Management Desktop App',
    description: 'A powerful desktop application for managing multiple AI providers, switching between models, and monitoring usage in real-time.',
    icon: 'lucide:app-window',
    gradient: 'from-primary-500 to-violet-500',
    features: [
      'Multi-provider configuration',
      'Real-time usage heatmaps',
      'Request logging & analytics',
      'MCP server integration',
      'Claude Code / Codex support',
    ],
    href: '/products/codeswitch',
  },
  {
    id: 'gateway',
    name: 'Gateway',
    tagline: 'Unified LLM API Gateway',
    description: 'Enterprise-grade API gateway that unifies 40+ AI providers behind a single, consistent interface with advanced features.',
    icon: 'lucide:network',
    gradient: 'from-accent-500 to-emerald-500',
    features: [
      '40+ AI provider support',
      'Intelligent routing & failover',
      'Usage quotas & billing',
      'Rate limiting & caching',
      'SOC 2 compliant infrastructure',
    ],
    href: '/products/gateway',
  },
]
</script>

<template>
  <div class="min-h-screen pt-24 lg:pt-32">
    <!-- Hero -->
    <section class="relative py-16 lg:py-24 overflow-hidden">
      <div class="absolute inset-0 bg-grid opacity-20" />
      <div
        class="gradient-orb w-[500px] h-[500px] -top-40 right-0"
        style="--orb-color: #6366f1"
      />

      <div class="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="text-center max-w-3xl mx-auto">
          <h1
            v-motion
            :initial="{ opacity: 0, y: 20 }"
            :enter="{ opacity: 1, y: 0 }"
            class="text-4xl sm:text-5xl lg:text-6xl font-bold text-white mb-6"
          >
            Our <span class="gradient-text">Products</span>
          </h1>
          <p
            v-motion
            :initial="{ opacity: 0, y: 20 }"
            :enter="{ opacity: 1, y: 0, transition: { delay: 100 } }"
            class="text-lg text-surface-300"
          >
            Choose the tools that fit your workflow. From desktop apps to cloud infrastructure,
            we've got you covered.
          </p>
        </div>
      </div>
    </section>

    <!-- Products -->
    <section class="py-16 lg:py-24">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid lg:grid-cols-2 gap-8 lg:gap-12">
          <div
            v-for="(product, index) in products"
            :key="product.id"
            v-motion
            :initial="{ opacity: 0, y: 30 }"
            :visibleOnce="{ opacity: 1, y: 0, transition: { delay: index * 100 } }"
            class="glass-card group cursor-pointer"
            @click="navigateTo(product.href)"
          >
            <!-- Header -->
            <div class="flex items-start gap-4 mb-6">
              <div
                class="w-14 h-14 rounded-xl flex items-center justify-center transition-transform duration-300 group-hover:scale-110"
                :class="`bg-gradient-to-br ${product.gradient}`"
              >
                <Icon :icon="product.icon" class="w-7 h-7 text-white" />
              </div>
              <div>
                <h2 class="text-2xl font-bold text-white">{{ product.name }}</h2>
                <p class="text-surface-400">{{ product.tagline }}</p>
              </div>
            </div>

            <!-- Description -->
            <p class="text-surface-300 mb-6 leading-relaxed">
              {{ product.description }}
            </p>

            <!-- Features -->
            <ul class="space-y-3 mb-8">
              <li
                v-for="feature in product.features"
                :key="feature"
                class="flex items-center gap-3 text-surface-300"
              >
                <Icon icon="lucide:check-circle" class="w-5 h-5 text-accent-400 flex-shrink-0" />
                <span>{{ feature }}</span>
              </li>
            </ul>

            <!-- CTA -->
            <NuxtLink
              :to="product.href"
              class="inline-flex items-center gap-2 text-primary-400 hover:text-primary-300 font-medium transition-colors"
            >
              Learn more
              <Icon icon="lucide:arrow-right" class="w-4 h-4 transition-transform group-hover:translate-x-1" />
            </NuxtLink>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
