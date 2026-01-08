<script setup lang="ts">
import { Icon } from '@iconify/vue'

const footerLinks = {
  products: [
    { name: 'CodeSwitch', href: '/products/codeswitch' },
    { name: 'Gateway', href: '/products/gateway' },
    { name: 'Pricing', href: '/pricing' },
  ],
  resources: [
    { name: 'Documentation', href: '/docs' },
    { name: 'API Reference', href: '/docs/api' },
    { name: 'Changelog', href: '/docs/changelog' },
  ],
  company: [
    { name: 'About', href: '/about' },
    { name: 'Contact', href: '/contact' },
    { name: 'Privacy', href: '/privacy' },
  ],
}

const socialLinks = [
  { icon: 'lucide:github', href: 'https://github.com/lurus-ai', label: 'GitHub' },
  { icon: 'lucide:twitter', href: 'https://twitter.com/lurus_ai', label: 'Twitter' },
  { icon: 'lucide:mail', href: 'mailto:support@lurus.cn', label: 'Email' },
]
</script>

<template>
  <footer class="bg-surface-900 border-t border-surface-800">
    <div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-12 lg:py-16">
      <div class="grid grid-cols-2 md:grid-cols-4 gap-8 lg:gap-12">
        <!-- Brand -->
        <div class="col-span-2 md:col-span-1">
          <NuxtLink to="/" class="flex items-center gap-3 mb-4">
            <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-primary-500 to-accent-500 flex items-center justify-center">
              <Icon icon="lucide:zap" class="w-6 h-6 text-white" />
            </div>
            <span class="text-xl font-bold text-white">
              Lurus<span class="text-primary-400">.AI</span>
            </span>
          </NuxtLink>
          <p class="text-surface-400 text-sm mb-6">
            Intelligent AI Infrastructure for modern developers. Unified gateway supporting 40+ AI providers.
          </p>
          <div class="flex gap-4">
            <a
              v-for="social in socialLinks"
              :key="social.label"
              :href="social.href"
              :aria-label="social.label"
              target="_blank"
              rel="noopener noreferrer"
              class="w-10 h-10 rounded-lg bg-surface-800 flex items-center justify-center text-surface-400 hover:text-primary-400 hover:bg-surface-700 transition-all duration-200"
            >
              <Icon :icon="social.icon" class="w-5 h-5" />
            </a>
          </div>
        </div>

        <!-- Products -->
        <div>
          <h3 class="text-white font-semibold mb-4">Products</h3>
          <ul class="space-y-3">
            <li v-for="link in footerLinks.products" :key="link.name">
              <NuxtLink
                :to="link.href"
                class="text-surface-400 hover:text-primary-400 text-sm transition-colors duration-200"
              >
                {{ link.name }}
              </NuxtLink>
            </li>
          </ul>
        </div>

        <!-- Resources -->
        <div>
          <h3 class="text-white font-semibold mb-4">Resources</h3>
          <ul class="space-y-3">
            <li v-for="link in footerLinks.resources" :key="link.name">
              <NuxtLink
                :to="link.href"
                class="text-surface-400 hover:text-primary-400 text-sm transition-colors duration-200"
              >
                {{ link.name }}
              </NuxtLink>
            </li>
          </ul>
        </div>

        <!-- Company -->
        <div>
          <h3 class="text-white font-semibold mb-4">Company</h3>
          <ul class="space-y-3">
            <li v-for="link in footerLinks.company" :key="link.name">
              <NuxtLink
                :to="link.href"
                class="text-surface-400 hover:text-primary-400 text-sm transition-colors duration-200"
              >
                {{ link.name }}
              </NuxtLink>
            </li>
          </ul>
        </div>
      </div>

      <!-- Bottom -->
      <div class="mt-12 pt-8 border-t border-surface-800 flex flex-col md:flex-row justify-between items-center gap-4">
        <p class="text-surface-500 text-sm">
          &copy; {{ new Date().getFullYear() }} Lurus AI. All rights reserved.
        </p>
        <p class="text-surface-600 text-xs">
          Powered by AI Infrastructure
        </p>
      </div>
    </div>
  </footer>
</template>
