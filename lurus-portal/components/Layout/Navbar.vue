<script setup lang="ts">
import { Icon } from '@iconify/vue'

const user = useSupabaseUser()
const isScrolled = ref(false)
const isMobileMenuOpen = ref(false)

const navLinks = [
  { name: 'Products', href: '/products' },
  { name: 'Pricing', href: '/pricing' },
  { name: 'Docs', href: '/docs' },
]

onMounted(() => {
  const handleScroll = () => {
    isScrolled.value = window.scrollY > 20
  }
  window.addEventListener('scroll', handleScroll)
  onUnmounted(() => window.removeEventListener('scroll', handleScroll))
})
</script>

<template>
  <header
    class="fixed top-0 left-0 right-0 z-50 transition-all duration-300"
    :class="[
      isScrolled
        ? 'bg-surface-900/80 backdrop-blur-xl border-b border-surface-700/50'
        : 'bg-transparent'
    ]"
  >
    <nav class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
      <div class="flex items-center justify-between h-16 lg:h-20">
        <!-- Logo -->
        <NuxtLink to="/" class="flex items-center gap-3 group">
          <div class="relative">
            <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-primary-500 to-accent-500 flex items-center justify-center transform group-hover:scale-110 transition-transform duration-300">
              <Icon icon="lucide:zap" class="w-6 h-6 text-white" />
            </div>
            <div class="absolute inset-0 rounded-xl bg-primary-500/50 blur-xl opacity-0 group-hover:opacity-100 transition-opacity duration-300" />
          </div>
          <span class="text-xl font-bold text-white">
            Lurus<span class="text-primary-400">.AI</span>
          </span>
        </NuxtLink>

        <!-- Desktop Navigation -->
        <div class="hidden md:flex items-center gap-8">
          <NuxtLink
            v-for="link in navLinks"
            :key="link.name"
            :to="link.href"
            class="text-surface-300 hover:text-white font-medium transition-colors duration-200 relative group"
          >
            {{ link.name }}
            <span class="absolute -bottom-1 left-0 w-0 h-0.5 bg-gradient-to-r from-primary-500 to-accent-500 group-hover:w-full transition-all duration-300" />
          </NuxtLink>
        </div>

        <!-- CTA Buttons -->
        <div class="hidden md:flex items-center gap-4">
          <template v-if="user">
            <NuxtLink
              to="/dashboard"
              class="text-surface-300 hover:text-white font-medium transition-colors duration-200"
            >
              Dashboard
            </NuxtLink>
          </template>
          <template v-else>
            <NuxtLink
              to="/auth/login"
              class="text-surface-300 hover:text-white font-medium transition-colors duration-200"
            >
              Login
            </NuxtLink>
            <NuxtLink to="/auth/register" class="btn-glow text-sm">
              Get Started
            </NuxtLink>
          </template>
        </div>

        <!-- Mobile Menu Button -->
        <button
          class="md:hidden p-2 text-surface-300 hover:text-white transition-colors"
          @click="isMobileMenuOpen = !isMobileMenuOpen"
        >
          <Icon
            :icon="isMobileMenuOpen ? 'lucide:x' : 'lucide:menu'"
            class="w-6 h-6"
          />
        </button>
      </div>

      <!-- Mobile Menu -->
      <Transition
        enter-active-class="transition-all duration-300 ease-out"
        enter-from-class="opacity-0 -translate-y-4"
        enter-to-class="opacity-100 translate-y-0"
        leave-active-class="transition-all duration-200 ease-in"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 -translate-y-4"
      >
        <div
          v-if="isMobileMenuOpen"
          class="md:hidden py-4 border-t border-surface-700/50"
        >
          <div class="flex flex-col gap-4">
            <NuxtLink
              v-for="link in navLinks"
              :key="link.name"
              :to="link.href"
              class="text-surface-300 hover:text-white font-medium py-2 transition-colors"
              @click="isMobileMenuOpen = false"
            >
              {{ link.name }}
            </NuxtLink>
            <div class="flex flex-col gap-3 pt-4 border-t border-surface-700/50">
              <template v-if="user">
                <NuxtLink
                  to="/dashboard"
                  class="btn-glow text-center text-sm"
                  @click="isMobileMenuOpen = false"
                >
                  Dashboard
                </NuxtLink>
              </template>
              <template v-else>
                <NuxtLink
                  to="/auth/login"
                  class="text-surface-300 hover:text-white font-medium py-2"
                  @click="isMobileMenuOpen = false"
                >
                  Login
                </NuxtLink>
                <NuxtLink
                  to="/auth/register"
                  class="btn-glow text-center text-sm"
                  @click="isMobileMenuOpen = false"
                >
                  Get Started
                </NuxtLink>
              </template>
            </div>
          </div>
        </div>
      </Transition>
    </nav>
  </header>
</template>
