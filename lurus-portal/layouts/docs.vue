<script setup lang="ts">
import { Icon } from '@iconify/vue'

const sidebarLinks = [
  {
    title: 'Getting Started',
    items: [
      { name: 'Introduction', href: '/docs' },
      { name: 'Quick Start', href: '/docs/quickstart' },
      { name: 'Installation', href: '/docs/installation' },
    ],
  },
  {
    title: 'Products',
    items: [
      { name: 'CodeSwitch', href: '/docs/codeswitch' },
      { name: 'Gateway', href: '/docs/gateway' },
    ],
  },
  {
    title: 'API Reference',
    items: [
      { name: 'Authentication', href: '/docs/api/auth' },
      { name: 'Chat Completions', href: '/docs/api/chat' },
      { name: 'Models', href: '/docs/api/models' },
    ],
  },
  {
    title: 'Guides',
    items: [
      { name: 'Claude Code Setup', href: '/docs/guides/claude-code' },
      { name: 'Codex Setup', href: '/docs/guides/codex' },
      { name: 'Model Routing', href: '/docs/guides/routing' },
    ],
  },
]

const route = useRoute()
const isSidebarOpen = ref(false)
</script>

<template>
  <div class="flex flex-col min-h-screen">
    <LayoutNavbar />

    <div class="flex-1 pt-20">
      <div class="max-w-8xl mx-auto px-4 sm:px-6 lg:px-8">
        <div class="lg:flex lg:gap-8">
          <!-- Mobile sidebar toggle -->
          <button
            class="lg:hidden fixed bottom-4 right-4 z-50 w-12 h-12 rounded-full bg-primary-500 text-white shadow-lg flex items-center justify-center"
            @click="isSidebarOpen = !isSidebarOpen"
          >
            <Icon :icon="isSidebarOpen ? 'lucide:x' : 'lucide:menu'" class="w-6 h-6" />
          </button>

          <!-- Sidebar -->
          <aside
            class="fixed lg:sticky top-20 left-0 z-40 w-64 h-[calc(100vh-5rem)] overflow-y-auto py-8 pr-4 transition-transform duration-300 lg:translate-x-0"
            :class="[
              isSidebarOpen
                ? 'translate-x-0 bg-surface-900 pl-4'
                : '-translate-x-full lg:translate-x-0'
            ]"
          >
            <nav class="space-y-8">
              <div v-for="section in sidebarLinks" :key="section.title">
                <h4 class="text-sm font-semibold text-surface-400 uppercase tracking-wider mb-3">
                  {{ section.title }}
                </h4>
                <ul class="space-y-1">
                  <li v-for="item in section.items" :key="item.href">
                    <NuxtLink
                      :to="item.href"
                      class="block px-3 py-2 rounded-lg text-sm transition-colors duration-200"
                      :class="[
                        route.path === item.href
                          ? 'bg-primary-500/10 text-primary-400 font-medium'
                          : 'text-surface-400 hover:text-surface-200 hover:bg-surface-800'
                      ]"
                      @click="isSidebarOpen = false"
                    >
                      {{ item.name }}
                    </NuxtLink>
                  </li>
                </ul>
              </div>
            </nav>
          </aside>

          <!-- Main content -->
          <main class="flex-1 min-w-0 py-8 lg:py-12">
            <article class="prose prose-invert prose-primary max-w-none">
              <slot />
            </article>
          </main>

          <!-- Table of contents (optional) -->
          <aside class="hidden xl:block w-56 py-8">
            <!-- TOC will be added later -->
          </aside>
        </div>
      </div>
    </div>

    <LayoutFooter />
  </div>
</template>

<style>
/* Prose overrides for dark theme */
.prose-invert {
  --tw-prose-body: theme('colors.surface.300');
  --tw-prose-headings: theme('colors.white');
  --tw-prose-lead: theme('colors.surface.400');
  --tw-prose-links: theme('colors.primary.400');
  --tw-prose-bold: theme('colors.white');
  --tw-prose-counters: theme('colors.surface.400');
  --tw-prose-bullets: theme('colors.surface.500');
  --tw-prose-hr: theme('colors.surface.700');
  --tw-prose-quotes: theme('colors.surface.200');
  --tw-prose-quote-borders: theme('colors.primary.500');
  --tw-prose-captions: theme('colors.surface.400');
  --tw-prose-code: theme('colors.primary.300');
  --tw-prose-pre-code: theme('colors.surface.200');
  --tw-prose-pre-bg: theme('colors.surface.800');
  --tw-prose-th-borders: theme('colors.surface.600');
  --tw-prose-td-borders: theme('colors.surface.700');
}

.prose h1 {
  @apply text-3xl lg:text-4xl font-bold mb-6;
}

.prose h2 {
  @apply text-2xl font-bold mt-12 mb-4 pb-2 border-b border-surface-700;
}

.prose h3 {
  @apply text-xl font-semibold mt-8 mb-3;
}

.prose pre {
  @apply rounded-xl border border-surface-700;
}

.prose code:not(pre code) {
  @apply px-1.5 py-0.5 rounded bg-surface-800 text-primary-300 font-normal;
}

.prose a {
  @apply text-primary-400 hover:text-primary-300 no-underline;
}

.prose a:hover {
  @apply underline;
}
</style>
