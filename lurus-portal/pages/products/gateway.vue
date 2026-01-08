<script setup lang="ts">
import { Icon } from '@iconify/vue'

useSeoMeta({
  title: 'Gateway - Unified LLM API Gateway',
  description: 'Enterprise-grade API gateway that unifies 40+ AI providers behind a single, consistent interface.',
})

const features = [
  {
    icon: 'lucide:network',
    title: '40+ Providers',
    description: 'Connect to OpenAI, Anthropic, Google, Azure, AWS Bedrock, and many more through a single API.',
  },
  {
    icon: 'lucide:shield-check',
    title: 'Enterprise Security',
    description: 'SOC 2 compliant. End-to-end encryption, audit logging, and role-based access control.',
  },
  {
    icon: 'lucide:zap',
    title: 'Intelligent Routing',
    description: 'Automatic failover, load balancing, and latency-based routing for optimal performance.',
  },
  {
    icon: 'lucide:credit-card',
    title: 'Usage & Billing',
    description: 'Track usage per user, team, or project. Set quotas and manage billing in real-time.',
  },
  {
    icon: 'lucide:gauge',
    title: 'Rate Limiting',
    description: 'Fine-grained rate limiting and throttling. Protect your infrastructure and manage costs.',
  },
  {
    icon: 'lucide:code',
    title: 'OpenAI Compatible',
    description: 'Drop-in replacement for OpenAI API. No code changes required for existing applications.',
  },
]

const providers = [
  { name: 'OpenAI', icon: 'simple-icons:openai' },
  { name: 'Anthropic', icon: 'simple-icons:anthropic' },
  { name: 'Google', icon: 'simple-icons:google' },
  { name: 'Azure', icon: 'simple-icons:microsoftazure' },
  { name: 'AWS', icon: 'simple-icons:amazonaws' },
  { name: 'Cohere', icon: 'lucide:sparkles' },
  { name: 'Mistral', icon: 'lucide:wind' },
  { name: 'DeepSeek', icon: 'lucide:search' },
]

const codeExample = `# Use Gateway just like OpenAI API
curl https://gateway.lurus.cn/v1/chat/completions \\
  -H "Authorization: Bearer $LURUS_API_KEY" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "claude-3-5-sonnet",
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'`
</script>

<template>
  <div class="min-h-screen pt-24 lg:pt-32">
    <!-- Hero -->
    <section class="relative py-16 lg:py-24 overflow-hidden">
      <div class="absolute inset-0 bg-grid opacity-20" />
      <div
        class="gradient-orb w-[600px] h-[600px] -top-40 right-0"
        style="--orb-color: #22d3ee"
      />
      <div
        class="gradient-orb w-[400px] h-[400px] bottom-0 -left-40"
        style="--orb-color: #10b981"
      />

      <div class="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid lg:grid-cols-2 gap-12 items-center">
          <!-- Left content -->
          <div>
            <div
              v-motion
              :initial="{ opacity: 0, y: 20 }"
              :enter="{ opacity: 1, y: 0 }"
              class="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-accent-500/10 border border-accent-500/20 mb-6"
            >
              <Icon icon="lucide:cloud" class="w-4 h-4 text-accent-400" />
              <span class="text-sm text-accent-300">Cloud Service</span>
            </div>

            <h1
              v-motion
              :initial="{ opacity: 0, y: 20 }"
              :enter="{ opacity: 1, y: 0, transition: { delay: 100 } }"
              class="text-4xl sm:text-5xl lg:text-6xl font-bold text-white mb-6"
            >
              <span class="gradient-text">Gateway</span>
            </h1>

            <p
              v-motion
              :initial="{ opacity: 0, y: 20 }"
              :enter="{ opacity: 1, y: 0, transition: { delay: 200 } }"
              class="text-lg text-surface-300 mb-8"
            >
              Enterprise-grade API gateway that unifies 40+ AI providers behind a single,
              consistent interface. OpenAI-compatible API for seamless integration.
            </p>

            <div
              v-motion
              :initial="{ opacity: 0, y: 20 }"
              :enter="{ opacity: 1, y: 0, transition: { delay: 300 } }"
              class="flex flex-wrap gap-4"
            >
              <NuxtLink to="/login" class="btn-glow flex items-center gap-2">
                <Icon icon="lucide:key" class="w-5 h-5" />
                Get API Key
              </NuxtLink>
              <NuxtLink to="/docs/gateway" class="btn-outline flex items-center gap-2">
                <Icon icon="lucide:book-open" class="w-5 h-5" />
                API Reference
              </NuxtLink>
            </div>
          </div>

          <!-- Right code example -->
          <div
            v-motion
            :initial="{ opacity: 0, x: 50 }"
            :enter="{ opacity: 1, x: 0, transition: { delay: 300 } }"
            class="relative"
          >
            <div class="relative rounded-xl overflow-hidden border border-surface-700 shadow-2xl">
              <!-- Code header -->
              <div class="bg-surface-800 px-4 py-3 flex items-center justify-between">
                <div class="flex items-center gap-2">
                  <Icon icon="lucide:terminal" class="w-4 h-4 text-surface-400" />
                  <span class="text-sm text-surface-400">Quick Start</span>
                </div>
                <button class="text-surface-500 hover:text-surface-300 transition-colors">
                  <Icon icon="lucide:copy" class="w-4 h-4" />
                </button>
              </div>
              <!-- Code content -->
              <div class="bg-surface-900 p-4 overflow-x-auto">
                <pre class="text-sm font-mono text-surface-300 whitespace-pre">{{ codeExample }}</pre>
              </div>
            </div>

            <!-- Glow effect -->
            <div class="absolute -inset-4 bg-gradient-to-r from-accent-500/20 to-emerald-500/20 blur-3xl -z-10" />
          </div>
        </div>
      </div>
    </section>

    <!-- Providers -->
    <section class="py-16 lg:py-24 border-b border-surface-800">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="text-center mb-12">
          <h2 class="text-2xl sm:text-3xl font-bold text-white mb-4">
            Supported Providers
          </h2>
          <p class="text-surface-400">
            Connect to all major AI providers through a single API
          </p>
        </div>

        <div class="flex flex-wrap items-center justify-center gap-8 lg:gap-12">
          <div
            v-for="provider in providers"
            :key="provider.name"
            v-motion
            :initial="{ opacity: 0, scale: 0.8 }"
            :visibleOnce="{ opacity: 1, scale: 1 }"
            class="flex items-center gap-3 text-surface-400 hover:text-surface-200 transition-colors"
          >
            <Icon :icon="provider.icon" class="w-8 h-8" />
            <span class="text-lg font-medium">{{ provider.name }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- Features -->
    <section class="py-16 lg:py-24">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="text-center mb-16">
          <h2 class="text-3xl sm:text-4xl font-bold text-white mb-4">
            Enterprise Features
          </h2>
          <p class="text-lg text-surface-400">
            Built for scale, security, and reliability
          </p>
        </div>

        <div class="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
          <div
            v-for="(feature, index) in features"
            :key="feature.title"
            v-motion
            :initial="{ opacity: 0, y: 30 }"
            :visibleOnce="{ opacity: 1, y: 0, transition: { delay: index * 100 } }"
            class="glass-card"
          >
            <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-accent-500 to-emerald-500 flex items-center justify-center mb-4">
              <Icon :icon="feature.icon" class="w-6 h-6 text-white" />
            </div>
            <h3 class="text-xl font-semibold text-white mb-2">{{ feature.title }}</h3>
            <p class="text-surface-400">{{ feature.description }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- CTA -->
    <section class="py-16 lg:py-24 relative overflow-hidden">
      <div class="absolute inset-0 bg-gradient-to-br from-accent-900/30 via-surface-900 to-emerald-900/20" />

      <div class="relative z-10 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <h2 class="text-3xl sm:text-4xl font-bold text-white mb-6">
          Start using Gateway today
        </h2>
        <p class="text-lg text-surface-300 mb-10">
          Free tier includes 100K tokens/month. No credit card required.
        </p>

        <div class="flex flex-col sm:flex-row items-center justify-center gap-4">
          <NuxtLink to="/login" class="btn-glow flex items-center gap-2">
            <Icon icon="lucide:key" class="w-5 h-5" />
            Get Free API Key
          </NuxtLink>
          <NuxtLink to="/pricing" class="btn-outline flex items-center gap-2">
            <Icon icon="lucide:credit-card" class="w-5 h-5" />
            View Pricing
          </NuxtLink>
        </div>
      </div>
    </section>
  </div>
</template>
