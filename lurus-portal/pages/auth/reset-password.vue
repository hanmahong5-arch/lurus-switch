<script setup lang="ts">
import { Icon } from '@iconify/vue'

definePageMeta({
  layout: 'auth',
})

useSeoMeta({
  title: 'Reset Password - Lurus AI',
  description: 'Reset your Lurus AI account password.',
})

const { resetPassword, loading, error } = useAuth()

const email = ref('')
const formError = ref<string | null>(null)
const success = ref(false)

const handleSubmit = async () => {
  formError.value = null

  if (!email.value) {
    formError.value = 'Please enter your email address'
    return
  }

  try {
    await resetPassword(email.value)
    success.value = true
  } catch (e: any) {
    formError.value = e.message || 'Failed to send reset email. Please try again.'
  }
}
</script>

<template>
  <div>
    <!-- Success state -->
    <template v-if="success">
      <div class="text-center">
        <div class="w-16 h-16 mx-auto mb-6 rounded-full bg-green-500/20 flex items-center justify-center">
          <Icon icon="lucide:mail-check" class="w-8 h-8 text-green-400" />
        </div>
        <h2 class="text-2xl font-bold text-white mb-2">Check your email</h2>
        <p class="text-surface-400 mb-8">
          We've sent a password reset link to <strong class="text-white">{{ email }}</strong>.
          Please check your inbox and follow the instructions.
        </p>
        <NuxtLink to="/auth/login" class="btn-glow inline-flex items-center gap-2">
          <Icon icon="lucide:arrow-left" class="w-4 h-4" />
          <span>Back to Sign In</span>
        </NuxtLink>
      </div>
    </template>

    <!-- Reset form -->
    <template v-else>
      <div class="text-center mb-8">
        <div class="w-16 h-16 mx-auto mb-6 rounded-full bg-primary-500/20 flex items-center justify-center">
          <Icon icon="lucide:key" class="w-8 h-8 text-primary-400" />
        </div>
        <h2 class="text-2xl font-bold text-white mb-2">Forgot your password?</h2>
        <p class="text-surface-400">Enter your email and we'll send you a reset link</p>
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
              v-model="email"
              type="email"
              placeholder="you@example.com"
              class="w-full pl-10 pr-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
              :disabled="loading"
            />
          </div>
        </div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full btn-glow py-3 flex items-center justify-center gap-2"
        >
          <Icon v-if="loading" icon="lucide:loader-2" class="w-5 h-5 animate-spin" />
          <span>{{ loading ? 'Sending...' : 'Send Reset Link' }}</span>
        </button>
      </form>

      <!-- Back to login -->
      <p class="mt-8 text-center text-sm text-surface-400">
        Remember your password?
        <NuxtLink to="/auth/login" class="text-primary-400 hover:text-primary-300 font-medium transition-colors">
          Sign in
        </NuxtLink>
      </p>
    </template>
  </div>
</template>
