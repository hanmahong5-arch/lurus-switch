<script setup lang="ts">
import { Icon } from '@iconify/vue'

const features = [
  {
    icon: 'lucide:layers',
    title: 'Multi-Provider Support',
    description: 'Connect to 40+ AI providers through a single unified API. OpenAI, Anthropic, Google, and more.',
    gradient: 'from-primary-500 to-primary-600',
  },
  {
    icon: 'lucide:zap',
    title: 'Lightning Fast',
    description: 'Sub-50ms latency with intelligent routing. Automatic failover ensures maximum uptime.',
    gradient: 'from-accent-500 to-accent-600',
  },
  {
    icon: 'lucide:shield-check',
    title: 'Enterprise Security',
    description: 'SOC 2 compliant infrastructure. End-to-end encryption and role-based access control.',
    gradient: 'from-emerald-500 to-emerald-600',
  },
  {
    icon: 'lucide:bar-chart-3',
    title: 'Usage Analytics',
    description: 'Real-time dashboards for cost tracking, quota management, and performance monitoring.',
    gradient: 'from-orange-500 to-orange-600',
  },
  {
    icon: 'lucide:git-branch',
    title: 'Model Switching',
    description: 'Seamlessly switch between models without code changes. A/B testing and gradual rollouts.',
    gradient: 'from-pink-500 to-pink-600',
  },
  {
    icon: 'lucide:puzzle',
    title: 'Easy Integration',
    description: 'Drop-in replacement for OpenAI API. Works with existing tools like Claude Code and Codex.',
    gradient: 'from-violet-500 to-violet-600',
  },
]
</script>

<template>
  <section class="py-24 lg:py-32 relative overflow-hidden">
    <!-- Background -->
    <div class="absolute inset-0 bg-surface-900">
      <div class="absolute inset-0 bg-grid opacity-20" />
    </div>

    <div class="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <!-- Section header -->
      <div class="text-center mb-16 lg:mb-20">
        <h2
          v-motion
          :initial="{ opacity: 0, y: 20 }"
          :visibleOnce="{ opacity: 1, y: 0 }"
          class="text-3xl sm:text-4xl lg:text-5xl font-bold text-white mb-4"
        >
          Everything you need to
          <span class="gradient-text">scale AI</span>
        </h2>
        <p
          v-motion
          :initial="{ opacity: 0, y: 20 }"
          :visibleOnce="{ opacity: 1, y: 0, transition: { delay: 100 } }"
          class="text-lg text-surface-400 max-w-2xl mx-auto"
        >
          Built for developers who need reliable, fast, and cost-effective AI infrastructure.
        </p>
      </div>

      <!-- Features grid -->
      <div class="grid md:grid-cols-2 lg:grid-cols-3 gap-6 lg:gap-8">
        <div
          v-for="(feature, index) in features"
          :key="feature.title"
          v-motion
          :initial="{ opacity: 0, y: 30 }"
          :visibleOnce="{ opacity: 1, y: 0, transition: { delay: index * 100 } }"
          class="glass-card group"
        >
          <!-- Icon -->
          <div
            class="w-12 h-12 rounded-xl flex items-center justify-center mb-4 transition-transform duration-300 group-hover:scale-110"
            :class="`bg-gradient-to-br ${feature.gradient}`"
          >
            <Icon :icon="feature.icon" class="w-6 h-6 text-white" />
          </div>

          <!-- Content -->
          <h3 class="text-xl font-semibold text-white mb-2">
            {{ feature.title }}
          </h3>
          <p class="text-surface-400 leading-relaxed">
            {{ feature.description }}
          </p>
        </div>
      </div>
    </div>
  </section>
</template>
