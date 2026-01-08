<script setup lang="ts">
import { Icon } from '@iconify/vue'

useSeoMeta({
  title: 'CodeSwitch - AI Provider Management Desktop App',
  description: 'Powerful desktop application for managing multiple AI providers, switching between models, and monitoring usage in real-time.',
})

const features = [
  {
    icon: 'lucide:layers',
    title: 'Multi-Provider Support',
    description: 'Configure and manage multiple AI providers in one place. OpenAI, Anthropic, Google, DeepSeek, and more.',
  },
  {
    icon: 'lucide:bar-chart-2',
    title: 'Usage Analytics',
    description: 'Real-time heatmaps and statistics. Track tokens, costs, and request patterns across all providers.',
  },
  {
    icon: 'lucide:git-branch',
    title: 'Smart Routing',
    description: 'Automatic failover and load balancing. Round-robin or priority-based model selection.',
  },
  {
    icon: 'lucide:terminal',
    title: 'CLI Integration',
    description: 'Works seamlessly with Claude Code, Codex CLI, and other AI development tools.',
  },
  {
    icon: 'lucide:puzzle',
    title: 'MCP Servers',
    description: 'Built-in MCP server management. Configure and monitor your Model Context Protocol servers.',
  },
  {
    icon: 'lucide:database',
    title: 'Request Logging',
    description: 'Comprehensive request logs with filtering, search, and export capabilities.',
  },
]

const screenshots = [
  { title: 'Dashboard', description: 'Real-time usage heatmap and analytics' },
  { title: 'Providers', description: 'Multi-provider configuration panel' },
  { title: 'Logs', description: 'Detailed request logging and analysis' },
]
</script>

<template>
  <div class="min-h-screen pt-24 lg:pt-32">
    <!-- Hero -->
    <section class="relative py-16 lg:py-24 overflow-hidden">
      <div class="absolute inset-0 bg-grid opacity-20" />
      <div
        class="gradient-orb w-[600px] h-[600px] -top-40 -left-40"
        style="--orb-color: #6366f1"
      />
      <div
        class="gradient-orb w-[400px] h-[400px] bottom-0 right-0"
        style="--orb-color: #8b5cf6"
      />

      <div class="relative z-10 max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="grid lg:grid-cols-2 gap-12 items-center">
          <!-- Left content -->
          <div>
            <div
              v-motion
              :initial="{ opacity: 0, y: 20 }"
              :enter="{ opacity: 1, y: 0 }"
              class="inline-flex items-center gap-2 px-4 py-2 rounded-full bg-primary-500/10 border border-primary-500/20 mb-6"
            >
              <Icon icon="lucide:app-window" class="w-4 h-4 text-primary-400" />
              <span class="text-sm text-primary-300">Desktop Application</span>
            </div>

            <h1
              v-motion
              :initial="{ opacity: 0, y: 20 }"
              :enter="{ opacity: 1, y: 0, transition: { delay: 100 } }"
              class="text-4xl sm:text-5xl lg:text-6xl font-bold text-white mb-6"
            >
              Code<span class="gradient-text">Switch</span>
            </h1>

            <p
              v-motion
              :initial="{ opacity: 0, y: 20 }"
              :enter="{ opacity: 1, y: 0, transition: { delay: 200 } }"
              class="text-lg text-surface-300 mb-8"
            >
              The ultimate AI provider management tool. Switch between models instantly,
              monitor usage in real-time, and optimize your AI development workflow.
            </p>

            <div
              v-motion
              :initial="{ opacity: 0, y: 20 }"
              :enter="{ opacity: 1, y: 0, transition: { delay: 300 } }"
              class="flex flex-wrap gap-4"
            >
              <a
                href="#download"
                class="btn-glow flex items-center gap-2"
              >
                <Icon icon="lucide:download" class="w-5 h-5" />
                Download for Windows
              </a>
              <NuxtLink to="/docs/codeswitch" class="btn-outline flex items-center gap-2">
                <Icon icon="lucide:book-open" class="w-5 h-5" />
                Documentation
              </NuxtLink>
            </div>

            <!-- Platform badges -->
            <div
              v-motion
              :initial="{ opacity: 0 }"
              :enter="{ opacity: 1, transition: { delay: 400 } }"
              class="mt-8 flex items-center gap-6 text-surface-500 text-sm"
            >
              <div class="flex items-center gap-2">
                <Icon icon="simple-icons:windows" class="w-4 h-4" />
                <span>Windows</span>
              </div>
              <div class="flex items-center gap-2">
                <Icon icon="simple-icons:apple" class="w-4 h-4" />
                <span>macOS (Soon)</span>
              </div>
              <div class="flex items-center gap-2">
                <Icon icon="simple-icons:linux" class="w-4 h-4" />
                <span>Linux (Soon)</span>
              </div>
            </div>
          </div>

          <!-- Right screenshot -->
          <div
            v-motion
            :initial="{ opacity: 0, x: 50 }"
            :enter="{ opacity: 1, x: 0, transition: { delay: 300 } }"
            class="relative"
          >
            <div class="relative rounded-xl overflow-hidden border border-surface-700 shadow-2xl">
              <!-- Fake window chrome -->
              <div class="bg-surface-800 px-4 py-3 flex items-center gap-2">
                <div class="flex gap-1.5">
                  <div class="w-3 h-3 rounded-full bg-red-500/80" />
                  <div class="w-3 h-3 rounded-full bg-yellow-500/80" />
                  <div class="w-3 h-3 rounded-full bg-green-500/80" />
                </div>
                <span class="text-xs text-surface-500 ml-2">CodeSwitch</span>
              </div>
              <!-- Screenshot placeholder -->
              <div class="bg-surface-900 aspect-[16/10] flex items-center justify-center">
                <div class="text-center p-8">
                  <Icon icon="lucide:image" class="w-16 h-16 text-surface-600 mx-auto mb-4" />
                  <p class="text-surface-500">Application Screenshot</p>
                </div>
              </div>
            </div>

            <!-- Glow effect -->
            <div class="absolute -inset-4 bg-gradient-to-r from-primary-500/20 to-violet-500/20 blur-3xl -z-10" />
          </div>
        </div>
      </div>
    </section>

    <!-- Features -->
    <section class="py-16 lg:py-24">
      <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="text-center mb-16">
          <h2 class="text-3xl sm:text-4xl font-bold text-white mb-4">
            Powerful Features
          </h2>
          <p class="text-lg text-surface-400">
            Everything you need to manage your AI providers
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
            <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-primary-500 to-primary-600 flex items-center justify-center mb-4">
              <Icon :icon="feature.icon" class="w-6 h-6 text-white" />
            </div>
            <h3 class="text-xl font-semibold text-white mb-2">{{ feature.title }}</h3>
            <p class="text-surface-400">{{ feature.description }}</p>
          </div>
        </div>
      </div>
    </section>

    <!-- Download section -->
    <section id="download" class="py-16 lg:py-24 relative overflow-hidden">
      <div class="absolute inset-0 bg-gradient-to-br from-primary-900/30 via-surface-900 to-violet-900/20" />

      <div class="relative z-10 max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 text-center">
        <h2 class="text-3xl sm:text-4xl font-bold text-white mb-6">
          Ready to get started?
        </h2>
        <p class="text-lg text-surface-300 mb-10">
          Download CodeSwitch and start managing your AI providers today.
        </p>

        <div class="flex flex-col sm:flex-row items-center justify-center gap-4">
          <a
            href="https://github.com/lurus-ai/codeswitch/releases"
            target="_blank"
            class="btn-glow flex items-center gap-2"
          >
            <Icon icon="lucide:download" class="w-5 h-5" />
            Download v1.0.0
          </a>
          <a
            href="https://github.com/lurus-ai/codeswitch"
            target="_blank"
            class="btn-outline flex items-center gap-2"
          >
            <Icon icon="lucide:github" class="w-5 h-5" />
            View on GitHub
          </a>
        </div>
      </div>
    </section>
  </div>
</template>
