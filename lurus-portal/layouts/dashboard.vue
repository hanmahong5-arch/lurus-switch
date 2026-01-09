<script setup lang="ts">
import { Icon } from '@iconify/vue'

const { user, profile, signOut } = useAuth()
const route = useRoute()
const isSidebarOpen = ref(true)
const isMobileSidebarOpen = ref(false)

const navigation = [
  { name: 'Overview', href: '/dashboard', icon: 'lucide:layout-dashboard' },
  { name: 'Usage', href: '/dashboard/usage', icon: 'lucide:bar-chart-3' },
  { name: 'API Keys', href: '/dashboard/api-keys', icon: 'lucide:key' },
  { name: 'Subscription', href: '/dashboard/subscription', icon: 'lucide:credit-card' },
  { name: 'Devices', href: '/dashboard/devices', icon: 'lucide:monitor-smartphone' },
  { name: 'Settings', href: '/dashboard/settings', icon: 'lucide:settings' },
]

const isActive = (href: string) => {
  if (href === '/dashboard') {
    return route.path === '/dashboard'
  }
  return route.path.startsWith(href)
}

const toggleSidebar = () => {
  isSidebarOpen.value = !isSidebarOpen.value
}

const handleSignOut = async () => {
  await signOut()
}
</script>

<template>
  <div class="min-h-screen bg-surface-950">
    <!-- Mobile sidebar backdrop -->
    <Transition
      enter-active-class="transition-opacity duration-300"
      enter-from-class="opacity-0"
      enter-to-class="opacity-100"
      leave-active-class="transition-opacity duration-300"
      leave-from-class="opacity-100"
      leave-to-class="opacity-0"
    >
      <div
        v-if="isMobileSidebarOpen"
        class="fixed inset-0 bg-black/50 z-40 lg:hidden"
        @click="isMobileSidebarOpen = false"
      />
    </Transition>

    <!-- Sidebar -->
    <aside
      :class="[
        'fixed top-0 left-0 z-50 h-full bg-surface-900 border-r border-surface-800 transition-all duration-300',
        isSidebarOpen ? 'w-64' : 'w-20',
        isMobileSidebarOpen ? 'translate-x-0' : '-translate-x-full lg:translate-x-0'
      ]"
    >
      <!-- Logo -->
      <div class="h-16 flex items-center justify-between px-4 border-b border-surface-800">
        <NuxtLink to="/" class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-xl bg-gradient-to-br from-primary-500 to-accent-500 flex items-center justify-center">
            <Icon icon="lucide:zap" class="w-6 h-6 text-white" />
          </div>
          <span v-if="isSidebarOpen" class="text-xl font-bold text-white">
            Lurus<span class="text-primary-400">.AI</span>
          </span>
        </NuxtLink>
        <button
          v-if="isSidebarOpen"
          class="hidden lg:flex p-2 text-surface-400 hover:text-white rounded-lg hover:bg-surface-800 transition-colors"
          @click="toggleSidebar"
        >
          <Icon icon="lucide:panel-left-close" class="w-5 h-5" />
        </button>
      </div>

      <!-- Navigation -->
      <nav class="p-4 space-y-2">
        <NuxtLink
          v-for="item in navigation"
          :key="item.name"
          :to="item.href"
          :class="[
            'flex items-center gap-3 px-3 py-2.5 rounded-lg transition-all duration-200',
            isActive(item.href)
              ? 'bg-primary-500/10 text-primary-400 border border-primary-500/20'
              : 'text-surface-400 hover:text-white hover:bg-surface-800'
          ]"
          @click="isMobileSidebarOpen = false"
        >
          <Icon :icon="item.icon" class="w-5 h-5 flex-shrink-0" />
          <span v-if="isSidebarOpen" class="font-medium">{{ item.name }}</span>
        </NuxtLink>
      </nav>

      <!-- Expand button (collapsed state) -->
      <button
        v-if="!isSidebarOpen"
        class="hidden lg:flex absolute bottom-20 left-1/2 -translate-x-1/2 p-2 text-surface-400 hover:text-white rounded-lg hover:bg-surface-800 transition-colors"
        @click="toggleSidebar"
      >
        <Icon icon="lucide:panel-left-open" class="w-5 h-5" />
      </button>

      <!-- User section -->
      <div class="absolute bottom-0 left-0 right-0 p-4 border-t border-surface-800">
        <div class="flex items-center gap-3">
          <div class="w-10 h-10 rounded-full bg-gradient-to-br from-primary-500 to-accent-500 flex items-center justify-center flex-shrink-0">
            <span class="text-white font-semibold">
              {{ profile?.display_name?.[0]?.toUpperCase() || user?.email?.[0]?.toUpperCase() || 'U' }}
            </span>
          </div>
          <div v-if="isSidebarOpen" class="flex-1 min-w-0">
            <p class="text-sm font-medium text-white truncate">
              {{ profile?.display_name || user?.email?.split('@')[0] }}
            </p>
            <p class="text-xs text-surface-400 truncate">
              {{ user?.email }}
            </p>
          </div>
          <button
            v-if="isSidebarOpen"
            class="p-2 text-surface-400 hover:text-red-400 rounded-lg hover:bg-surface-800 transition-colors"
            title="Sign out"
            @click="handleSignOut"
          >
            <Icon icon="lucide:log-out" class="w-5 h-5" />
          </button>
        </div>
      </div>
    </aside>

    <!-- Main content -->
    <div
      :class="[
        'transition-all duration-300',
        isSidebarOpen ? 'lg:pl-64' : 'lg:pl-20'
      ]"
    >
      <!-- Top bar -->
      <header class="sticky top-0 z-30 h-16 bg-surface-900/80 backdrop-blur-xl border-b border-surface-800">
        <div class="h-full px-4 lg:px-8 flex items-center justify-between">
          <!-- Mobile menu button -->
          <button
            class="lg:hidden p-2 text-surface-400 hover:text-white rounded-lg hover:bg-surface-800 transition-colors"
            @click="isMobileSidebarOpen = true"
          >
            <Icon icon="lucide:menu" class="w-6 h-6" />
          </button>

          <!-- Breadcrumb / Page title -->
          <div class="hidden lg:flex items-center gap-2 text-sm">
            <NuxtLink to="/dashboard" class="text-surface-400 hover:text-white transition-colors">
              Dashboard
            </NuxtLink>
            <template v-if="route.path !== '/dashboard'">
              <Icon icon="lucide:chevron-right" class="w-4 h-4 text-surface-600" />
              <span class="text-white capitalize">
                {{ route.path.split('/').pop()?.replace(/-/g, ' ') }}
              </span>
            </template>
          </div>

          <!-- Right side actions -->
          <div class="flex items-center gap-4">
            <!-- Notifications -->
            <button class="p-2 text-surface-400 hover:text-white rounded-lg hover:bg-surface-800 transition-colors relative">
              <Icon icon="lucide:bell" class="w-5 h-5" />
              <span class="absolute top-1 right-1 w-2 h-2 bg-accent-500 rounded-full" />
            </button>

            <!-- Help -->
            <NuxtLink
              to="/docs"
              class="p-2 text-surface-400 hover:text-white rounded-lg hover:bg-surface-800 transition-colors"
            >
              <Icon icon="lucide:help-circle" class="w-5 h-5" />
            </NuxtLink>

            <!-- Back to marketing site -->
            <NuxtLink
              to="/"
              class="hidden sm:flex items-center gap-2 px-3 py-1.5 text-sm text-surface-400 hover:text-white rounded-lg hover:bg-surface-800 transition-colors"
            >
              <Icon icon="lucide:external-link" class="w-4 h-4" />
              <span>Visit Site</span>
            </NuxtLink>
          </div>
        </div>
      </header>

      <!-- Page content -->
      <main class="p-4 lg:p-8">
        <slot />
      </main>
    </div>
  </div>
</template>
