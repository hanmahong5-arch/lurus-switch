<script setup lang="ts">
import { Icon } from '@iconify/vue'

definePageMeta({
  layout: 'auth',
  middleware: ['guest'],
})

useSeoMeta({
  title: 'Sign In - Lurus AI',
  description: 'Sign in to your Lurus AI account to access the dashboard.',
})

const { signInWithEmail, signInWithProvider, loading, error } = useAuth()
const route = useRoute()

const form = reactive({
  email: '',
  password: '',
  rememberMe: false,
})

const formError = ref<string | null>(null)

const handleSubmit = async () => {
  formError.value = null

  if (!form.email || !form.password) {
    formError.value = 'Please enter your email and password'
    return
  }

  try {
    await signInWithEmail(form.email, form.password)

    // Redirect to dashboard or original destination
    const redirect = route.query.redirect as string || '/dashboard'
    await navigateTo(redirect)
  } catch (e: any) {
    formError.value = e.message || 'Login failed. Please try again.'
  }
}

const handleOAuth = async (provider: 'github' | 'google') => {
  try {
    await signInWithProvider(provider)
  } catch (e: any) {
    formError.value = e.message || 'OAuth login failed. Please try again.'
  }
}
</script>

<template>
  <div>
    <div class="text-center mb-8">
      <h2 class="text-2xl font-bold text-white mb-2">Welcome back</h2>
      <p class="text-surface-400">Sign in to your account to continue</p>
    </div>

    <!-- Error message -->
    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition-all duration-200"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div
        v-if="formError || error"
        class="mb-6 p-4 rounded-lg bg-red-500/10 border border-red-500/20"
      >
        <p class="text-sm text-red-400">{{ formError || error }}</p>
      </div>
    </Transition>

    <!-- Login form -->
    <form class="space-y-5" @submit.prevent="handleSubmit">
      <div>
        <label for="email" class="block text-sm font-medium text-surface-300 mb-2">
          Email address
        </label>
        <div class="relative">
          <Icon
            icon="lucide:mail"
            class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-surface-500"
          />
          <input
            id="email"
            v-model="form.email"
            type="email"
            placeholder="you@example.com"
            class="w-full pl-10 pr-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
            :disabled="loading"
          />
        </div>
      </div>

      <div>
        <div class="flex items-center justify-between mb-2">
          <label for="password" class="block text-sm font-medium text-surface-300">
            Password
          </label>
          <NuxtLink
            to="/auth/reset-password"
            class="text-sm text-primary-400 hover:text-primary-300 transition-colors"
          >
            Forgot password?
          </NuxtLink>
        </div>
        <div class="relative">
          <Icon
            icon="lucide:lock"
            class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-surface-500"
          />
          <input
            id="password"
            v-model="form.password"
            type="password"
            placeholder="Enter your password"
            class="w-full pl-10 pr-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
            :disabled="loading"
          />
        </div>
      </div>

      <div class="flex items-center">
        <input
          id="remember"
          v-model="form.rememberMe"
          type="checkbox"
          class="w-4 h-4 rounded border-surface-600 bg-surface-800 text-primary-500 focus:ring-primary-500/50"
        />
        <label for="remember" class="ml-2 text-sm text-surface-400">
          Remember me for 30 days
        </label>
      </div>

      <button
        type="submit"
        :disabled="loading"
        class="w-full btn-glow py-3 flex items-center justify-center gap-2"
      >
        <Icon v-if="loading" icon="lucide:loader-2" class="w-5 h-5 animate-spin" />
        <span>{{ loading ? 'Signing in...' : 'Sign in' }}</span>
      </button>
    </form>

    <!-- Divider -->
    <div class="relative my-8">
      <div class="absolute inset-0 flex items-center">
        <div class="w-full border-t border-surface-700" />
      </div>
      <div class="relative flex justify-center text-sm">
        <span class="px-4 bg-surface-950 text-surface-500">Or continue with</span>
      </div>
    </div>

    <!-- Social login -->
    <div class="grid grid-cols-2 gap-4">
      <button
        type="button"
        class="flex items-center justify-center gap-2 px-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white hover:bg-surface-700 transition-colors"
        :disabled="loading"
        @click="handleOAuth('github')"
      >
        <Icon icon="mdi:github" class="w-5 h-5" />
        <span>GitHub</span>
      </button>
      <button
        type="button"
        class="flex items-center justify-center gap-2 px-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white hover:bg-surface-700 transition-colors"
        :disabled="loading"
        @click="handleOAuth('google')"
      >
        <Icon icon="mdi:google" class="w-5 h-5" />
        <span>Google</span>
      </button>
    </div>

    <!-- Sign up link -->
    <p class="mt-8 text-center text-sm text-surface-400">
      Don't have an account?
      <NuxtLink to="/auth/register" class="text-primary-400 hover:text-primary-300 font-medium transition-colors">
        Sign up for free
      </NuxtLink>
    </p>
  </div>
</template>
