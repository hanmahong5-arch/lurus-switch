<script setup lang="ts">
import { Icon } from '@iconify/vue'

useSeoMeta({
  title: 'Login - Lurus AI',
  description: 'Sign in to your Lurus AI account.',
})

const isLoading = ref(false)
const activeTab = ref<'login' | 'register'>('login')

const loginForm = ref({
  email: '',
  password: '',
  remember: false,
})

const registerForm = ref({
  name: '',
  email: '',
  password: '',
  confirmPassword: '',
})

const handleLogin = async () => {
  isLoading.value = true
  // TODO: Implement actual login with Casdoor
  await new Promise(resolve => setTimeout(resolve, 1000))
  isLoading.value = false
  alert('Login functionality will be connected to Casdoor')
}

const handleRegister = async () => {
  isLoading.value = true
  // TODO: Implement actual registration with Casdoor
  await new Promise(resolve => setTimeout(resolve, 1000))
  isLoading.value = false
  alert('Registration functionality will be connected to Casdoor')
}

const socialProviders = [
  { name: 'GitHub', icon: 'lucide:github' },
  { name: 'Google', icon: 'simple-icons:google' },
  { name: 'WeChat', icon: 'simple-icons:wechat' },
]
</script>

<template>
  <div class="min-h-screen pt-20 flex items-center justify-center relative overflow-hidden">
    <!-- Background -->
    <div class="absolute inset-0 bg-surface-900">
      <div class="absolute inset-0 bg-grid opacity-20" />
      <div
        class="gradient-orb w-[500px] h-[500px] -top-40 -left-40"
        style="--orb-color: #6366f1"
      />
      <div
        class="gradient-orb w-[400px] h-[400px] -bottom-20 -right-20"
        style="--orb-color: #22d3ee"
      />
    </div>

    <div class="relative z-10 w-full max-w-md mx-auto px-4 py-12">
      <!-- Logo -->
      <div class="text-center mb-8">
        <NuxtLink to="/" class="inline-flex items-center gap-3">
          <div class="w-12 h-12 rounded-xl bg-gradient-to-br from-primary-500 to-accent-500 flex items-center justify-center">
            <Icon icon="lucide:zap" class="w-7 h-7 text-white" />
          </div>
          <span class="text-2xl font-bold text-white">
            Lurus<span class="text-primary-400">.AI</span>
          </span>
        </NuxtLink>
      </div>

      <!-- Card -->
      <div class="glass-card">
        <!-- Tabs -->
        <div class="flex mb-6">
          <button
            class="flex-1 py-2 text-center font-medium transition-colors duration-200 border-b-2"
            :class="[
              activeTab === 'login'
                ? 'text-primary-400 border-primary-500'
                : 'text-surface-400 border-transparent hover:text-surface-200'
            ]"
            @click="activeTab = 'login'"
          >
            Sign In
          </button>
          <button
            class="flex-1 py-2 text-center font-medium transition-colors duration-200 border-b-2"
            :class="[
              activeTab === 'register'
                ? 'text-primary-400 border-primary-500'
                : 'text-surface-400 border-transparent hover:text-surface-200'
            ]"
            @click="activeTab = 'register'"
          >
            Register
          </button>
        </div>

        <!-- Login Form -->
        <form v-if="activeTab === 'login'" @submit.prevent="handleLogin">
          <div class="space-y-4">
            <div>
              <label for="email" class="block text-sm font-medium text-surface-300 mb-1">
                Email
              </label>
              <input
                id="email"
                v-model="loginForm.email"
                type="email"
                required
                class="w-full px-4 py-2.5 rounded-lg bg-surface-800 border border-surface-700 text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition-all"
                placeholder="you@example.com"
              />
            </div>

            <div>
              <label for="password" class="block text-sm font-medium text-surface-300 mb-1">
                Password
              </label>
              <input
                id="password"
                v-model="loginForm.password"
                type="password"
                required
                class="w-full px-4 py-2.5 rounded-lg bg-surface-800 border border-surface-700 text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition-all"
                placeholder="••••••••"
              />
            </div>

            <div class="flex items-center justify-between">
              <label class="flex items-center gap-2 text-sm text-surface-400">
                <input
                  v-model="loginForm.remember"
                  type="checkbox"
                  class="w-4 h-4 rounded bg-surface-800 border-surface-600 text-primary-500 focus:ring-primary-500"
                />
                Remember me
              </label>
              <a href="#" class="text-sm text-primary-400 hover:text-primary-300">
                Forgot password?
              </a>
            </div>

            <button
              type="submit"
              :disabled="isLoading"
              class="w-full btn-glow flex items-center justify-center gap-2"
            >
              <Icon v-if="isLoading" icon="lucide:loader-2" class="w-5 h-5 animate-spin" />
              <span>{{ isLoading ? 'Signing in...' : 'Sign In' }}</span>
            </button>
          </div>
        </form>

        <!-- Register Form -->
        <form v-else @submit.prevent="handleRegister">
          <div class="space-y-4">
            <div>
              <label for="name" class="block text-sm font-medium text-surface-300 mb-1">
                Name
              </label>
              <input
                id="name"
                v-model="registerForm.name"
                type="text"
                required
                class="w-full px-4 py-2.5 rounded-lg bg-surface-800 border border-surface-700 text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition-all"
                placeholder="Your name"
              />
            </div>

            <div>
              <label for="reg-email" class="block text-sm font-medium text-surface-300 mb-1">
                Email
              </label>
              <input
                id="reg-email"
                v-model="registerForm.email"
                type="email"
                required
                class="w-full px-4 py-2.5 rounded-lg bg-surface-800 border border-surface-700 text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition-all"
                placeholder="you@example.com"
              />
            </div>

            <div>
              <label for="reg-password" class="block text-sm font-medium text-surface-300 mb-1">
                Password
              </label>
              <input
                id="reg-password"
                v-model="registerForm.password"
                type="password"
                required
                class="w-full px-4 py-2.5 rounded-lg bg-surface-800 border border-surface-700 text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition-all"
                placeholder="••••••••"
              />
            </div>

            <div>
              <label for="confirm-password" class="block text-sm font-medium text-surface-300 mb-1">
                Confirm Password
              </label>
              <input
                id="confirm-password"
                v-model="registerForm.confirmPassword"
                type="password"
                required
                class="w-full px-4 py-2.5 rounded-lg bg-surface-800 border border-surface-700 text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500 focus:border-transparent transition-all"
                placeholder="••••••••"
              />
            </div>

            <button
              type="submit"
              :disabled="isLoading"
              class="w-full btn-glow flex items-center justify-center gap-2"
            >
              <Icon v-if="isLoading" icon="lucide:loader-2" class="w-5 h-5 animate-spin" />
              <span>{{ isLoading ? 'Creating account...' : 'Create Account' }}</span>
            </button>
          </div>
        </form>

        <!-- Divider -->
        <div class="relative my-6">
          <div class="absolute inset-0 flex items-center">
            <div class="w-full border-t border-surface-700" />
          </div>
          <div class="relative flex justify-center text-sm">
            <span class="px-2 bg-surface-800 text-surface-500">Or continue with</span>
          </div>
        </div>

        <!-- Social login -->
        <div class="grid grid-cols-3 gap-3">
          <button
            v-for="provider in socialProviders"
            :key="provider.name"
            type="button"
            class="flex items-center justify-center gap-2 py-2.5 rounded-lg border border-surface-700 text-surface-400 hover:text-white hover:border-surface-600 hover:bg-surface-800 transition-all"
          >
            <Icon :icon="provider.icon" class="w-5 h-5" />
          </button>
        </div>
      </div>

      <!-- Back link -->
      <p class="text-center mt-6 text-surface-500 text-sm">
        <NuxtLink to="/" class="text-primary-400 hover:text-primary-300">
          ← Back to homepage
        </NuxtLink>
      </p>
    </div>
  </div>
</template>
