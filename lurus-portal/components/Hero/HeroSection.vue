<script setup lang="ts">
import { Icon } from '@iconify/vue'

// Animated particles
const particles = ref<Array<{ id: number; x: number; y: number; size: number; duration: number; delay: number }>>([])

onMounted(() => {
  // Generate random particles
  particles.value = Array.from({ length: 50 }, (_, i) => ({
    id: i,
    x: Math.random() * 100,
    y: Math.random() * 100,
    size: Math.random() * 4 + 1,
    duration: Math.random() * 20 + 10,
    delay: Math.random() * 5,
  }))
})

const stats = [
  { value: '40+', label: 'AI Providers' },
  { value: '99.9%', label: 'Uptime' },
  { value: '10M+', label: 'API Calls' },
  { value: '<50ms', label: 'Latency' },
]

const techLogos = [
  { name: 'OpenAI', icon: 'simple-icons:openai' },
  { name: 'Anthropic', icon: 'simple-icons:anthropic' },
  { name: 'Google', icon: 'simple-icons:google' },
  { name: 'Meta', icon: 'simple-icons:meta' },
  { name: 'Microsoft', icon: 'simple-icons:microsoft' },
  { name: 'Mistral', icon: 'simple-icons:airchina' },
]
</script>

<template>
  <section class="relative min-h-screen flex items-center justify-center overflow-hidden pt-20">
    <!-- Background -->
    <div class="absolute inset-0 bg-surface-900">
      <!-- Grid pattern -->
      <div class="absolute inset-0 bg-grid opacity-40" />

      <!-- Gradient orbs -->
      <div
        class="gradient-orb w-[600px] h-[600px] -top-40 -left-40"
        style="--orb-color: #6366f1"
      />
      <div
        class="gradient-orb w-[500px] h-[500px] top-1/2 -right-40"
        style="--orb-color: #22d3ee"
      />
      <div
        class="gradient-orb w-[400px] h-[400px] -bottom-20 left-1/3"
        style="--orb-color: #8b5cf6"
      />

      <!-- Animated particles -->
      <div
        v-for="particle in particles"
        :key="particle.id"
        class="absolute rounded-full bg-primary-400/20"
        :style="{
          left: `${particle.x}%`,
          top: `${particle.y}%`,
          width: `${particle.size}px`,
          height: `${particle.size}px`,
          animation: `float ${particle.duration}s ease-in-out infinite`,
          animationDelay: `${particle.delay}s`,
        }"
      />
    </div>

    <!-- Content -->
    <div class="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-20 text-center">
      <!-- Badge -->
      <div
        v-motion
        :initial="{ opacity: 0, y: 20 }"
        :enter="{ opacity: 1, y: 0, transition: { delay: 100 } }"
        class="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-surface-800/50 border border-surface-700/50 backdrop-blur-sm mb-8"
      >
        <span class="w-2 h-2 rounded-full bg-accent-400 animate-pulse" />
        <span class="text-sm text-surface-300">Now supporting Claude 3.5 Sonnet</span>
      </div>

      <!-- Main heading -->
      <h1
        v-motion
        :initial="{ opacity: 0, y: 30 }"
        :enter="{ opacity: 1, y: 0, transition: { delay: 200 } }"
        class="text-4xl sm:text-5xl lg:text-7xl font-bold text-white mb-6 leading-tight"
      >
        Intelligent AI
        <br />
        <span class="gradient-text">Infrastructure</span>
      </h1>

      <!-- Subtitle -->
      <p
        v-motion
        :initial="{ opacity: 0, y: 30 }"
        :enter="{ opacity: 1, y: 0, transition: { delay: 300 } }"
        class="text-lg sm:text-xl text-surface-300 max-w-2xl mx-auto mb-10"
      >
        Unified gateway for 40+ AI providers. Seamlessly switch between models,
        manage quotas, and scale your AI applications with enterprise-grade reliability.
      </p>

      <!-- CTA Buttons -->
      <div
        v-motion
        :initial="{ opacity: 0, y: 30 }"
        :enter="{ opacity: 1, y: 0, transition: { delay: 400 } }"
        class="flex flex-col sm:flex-row items-center justify-center gap-4 mb-16"
      >
        <NuxtLink to="/products" class="btn-glow flex items-center gap-2">
          <Icon icon="lucide:rocket" class="w-5 h-5" />
          Get Started
        </NuxtLink>
        <NuxtLink to="/docs" class="btn-outline flex items-center gap-2">
          <Icon icon="lucide:book-open" class="w-5 h-5" />
          Read Documentation
        </NuxtLink>
      </div>

      <!-- Stats -->
      <div
        v-motion
        :initial="{ opacity: 0, y: 40 }"
        :enter="{ opacity: 1, y: 0, transition: { delay: 500 } }"
        class="grid grid-cols-2 md:grid-cols-4 gap-6 lg:gap-12 max-w-4xl mx-auto mb-16"
      >
        <div
          v-for="(stat, index) in stats"
          :key="stat.label"
          class="glass-card text-center"
        >
          <div class="text-3xl lg:text-4xl font-bold gradient-text mb-2">
            {{ stat.value }}
          </div>
          <div class="text-surface-400 text-sm">{{ stat.label }}</div>
        </div>
      </div>

      <!-- Tech logos -->
      <div
        v-motion
        :initial="{ opacity: 0 }"
        :enter="{ opacity: 1, transition: { delay: 600 } }"
        class="pt-8 border-t border-surface-800"
      >
        <p class="text-surface-500 text-sm mb-6">Trusted by developers using</p>
        <div class="flex flex-wrap items-center justify-center gap-8 opacity-60">
          <div
            v-for="logo in techLogos"
            :key="logo.name"
            class="flex items-center gap-2 text-surface-400 hover:text-surface-200 transition-colors duration-200"
          >
            <Icon :icon="logo.icon" class="w-6 h-6" />
            <span class="text-sm font-medium">{{ logo.name }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Scroll indicator -->
    <div class="absolute bottom-8 left-1/2 -translate-x-1/2 animate-bounce">
      <Icon icon="lucide:chevron-down" class="w-6 h-6 text-surface-500" />
    </div>
  </section>
</template>
