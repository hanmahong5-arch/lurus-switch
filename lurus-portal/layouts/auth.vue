<script setup lang="ts">
import { Icon } from '@iconify/vue'
</script>

<template>
  <div class="min-h-screen bg-surface-950 flex">
    <!-- Left side - Branding -->
    <div class="hidden lg:flex lg:w-1/2 relative overflow-hidden">
      <!-- Background effects -->
      <div class="absolute inset-0 bg-gradient-to-br from-primary-900/20 via-surface-900 to-accent-900/20" />
      <div class="gradient-orb w-[600px] h-[600px] -top-40 -left-40" style="--orb-color: #6366f1" />
      <div class="gradient-orb w-[500px] h-[500px] bottom-0 right-0" style="--orb-color: #22d3ee" />
      <div class="absolute inset-0 bg-grid opacity-10" />

      <!-- Content -->
      <div class="relative z-10 flex flex-col justify-center p-12 xl:p-20">
        <!-- Logo -->
        <NuxtLink to="/" class="flex items-center gap-3 mb-12">
          <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-primary-500 to-accent-500 flex items-center justify-center">
            <Icon icon="lucide:zap" class="w-7 h-7 text-white" />
          </div>
          <span class="text-2xl font-bold text-white">
            Lurus<span class="text-primary-400">.AI</span>
          </span>
        </NuxtLink>

        <!-- Headline -->
        <h1 class="text-4xl xl:text-5xl font-bold text-white mb-6">
          Intelligent AI
          <span class="gradient-text">Infrastructure</span>
        </h1>
        <p class="text-lg text-surface-300 mb-12 max-w-md">
          Unified gateway for 40+ AI providers. Real-time quota management,
          multi-device sync, and enterprise-grade reliability.
        </p>

        <!-- Features -->
        <div class="space-y-4">
          <div class="flex items-center gap-3 text-surface-300">
            <div class="w-8 h-8 rounded-lg bg-primary-500/20 flex items-center justify-center">
              <Icon icon="lucide:check" class="w-4 h-4 text-primary-400" />
            </div>
            <span>Multi-provider AI routing</span>
          </div>
          <div class="flex items-center gap-3 text-surface-300">
            <div class="w-8 h-8 rounded-lg bg-accent-500/20 flex items-center justify-center">
              <Icon icon="lucide:check" class="w-4 h-4 text-accent-400" />
            </div>
            <span>Real-time quota tracking</span>
          </div>
          <div class="flex items-center gap-3 text-surface-300">
            <div class="w-8 h-8 rounded-lg bg-primary-500/20 flex items-center justify-center">
              <Icon icon="lucide:check" class="w-4 h-4 text-primary-400" />
            </div>
            <span>99.9% uptime SLA</span>
          </div>
        </div>
      </div>
    </div>

    <!-- Right side - Form -->
    <div class="w-full lg:w-1/2 flex items-center justify-center p-6 sm:p-12">
      <div class="w-full max-w-md">
        <!-- Mobile logo -->
        <NuxtLink to="/" class="lg:hidden flex items-center justify-center gap-3 mb-8">
          <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-primary-500 to-accent-500 flex items-center justify-center">
            <Icon icon="lucide:zap" class="w-6 h-6 text-white" />
          </div>
          <span class="text-xl font-bold text-white">
            Lurus<span class="text-primary-400">.AI</span>
          </span>
        </NuxtLink>

        <slot />
      </div>
    </div>
  </div>
</template>
