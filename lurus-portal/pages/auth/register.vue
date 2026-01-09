<script setup lang="ts">
import { Icon } from '@iconify/vue'

definePageMeta({
  layout: 'auth',
  middleware: ['guest'],
})

useSeoMeta({
  title: 'Create Account - Lurus AI',
  description: 'Create your free Lurus AI account and start using AI infrastructure.',
})

const { signUpWithEmail, signInWithProvider, loading, error } = useAuth()

const form = reactive({
  name: '',
  email: '',
  password: '',
  confirmPassword: '',
  agreeTerms: false,
})

const formError = ref<string | null>(null)
const success = ref(false)

const passwordStrength = computed(() => {
  const password = form.password
  if (!password) return { score: 0, label: '', color: '' }

  let score = 0
  if (password.length >= 8) score++
  if (password.length >= 12) score++
  if (/[a-z]/.test(password) && /[A-Z]/.test(password)) score++
  if (/\d/.test(password)) score++
  if (/[^a-zA-Z0-9]/.test(password)) score++

  const labels = ['Very Weak', 'Weak', 'Fair', 'Good', 'Strong']
  const colors = ['bg-red-500', 'bg-orange-500', 'bg-yellow-500', 'bg-lime-500', 'bg-green-500']

  return {
    score,
    label: labels[Math.min(score, 4)],
    color: colors[Math.min(score, 4)],
  }
})

const handleSubmit = async () => {
  formError.value = null

  // Validation
  if (!form.name || !form.email || !form.password) {
    formError.value = 'Please fill in all required fields'
    return
  }

  if (form.password !== form.confirmPassword) {
    formError.value = 'Passwords do not match'
    return
  }

  if (form.password.length < 8) {
    formError.value = 'Password must be at least 8 characters'
    return
  }

  if (!form.agreeTerms) {
    formError.value = 'Please agree to the Terms of Service'
    return
  }

  try {
    await signUpWithEmail(form.email, form.password, form.name)
    success.value = true
  } catch (e: any) {
    formError.value = e.message || 'Registration failed. Please try again.'
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
    <!-- Success state -->
    <template v-if="success">
      <div class="text-center">
        <div class="w-16 h-16 mx-auto mb-6 rounded-full bg-green-500/20 flex items-center justify-center">
          <Icon icon="lucide:mail-check" class="w-8 h-8 text-green-400" />
        </div>
        <h2 class="text-2xl font-bold text-white mb-2">Check your email</h2>
        <p class="text-surface-400 mb-8">
          We've sent a verification link to <strong class="text-white">{{ form.email }}</strong>.
          Please click the link to verify your account.
        </p>
        <NuxtLink to="/auth/login" class="btn-glow inline-flex items-center gap-2">
          <Icon icon="lucide:arrow-left" class="w-4 h-4" />
          <span>Back to Sign In</span>
        </NuxtLink>
      </div>
    </template>

    <!-- Registration form -->
    <template v-else>
      <div class="text-center mb-8">
        <h2 class="text-2xl font-bold text-white mb-2">Create your account</h2>
        <p class="text-surface-400">Start using AI infrastructure in minutes</p>
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
          <label for="name" class="block text-sm font-medium text-surface-300 mb-2">
            Display name
          </label>
          <div class="relative">
            <Icon
              icon="lucide:user"
              class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-surface-500"
            />
            <input
              id="name"
              v-model="form.name"
              type="text"
              placeholder="Your name"
              class="w-full pl-10 pr-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
              :disabled="loading"
            />
          </div>
        </div>

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
          <label for="password" class="block text-sm font-medium text-surface-300 mb-2">
            Password
          </label>
          <div class="relative">
            <Icon
              icon="lucide:lock"
              class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-surface-500"
            />
            <input
              id="password"
              v-model="form.password"
              type="password"
              placeholder="Create a password"
              class="w-full pl-10 pr-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
              :disabled="loading"
            />
          </div>
          <!-- Password strength indicator -->
          <div v-if="form.password" class="mt-2">
            <div class="flex gap-1 mb-1">
              <div
                v-for="i in 5"
                :key="i"
                class="h-1 flex-1 rounded-full transition-colors"
                :class="i <= passwordStrength.score ? passwordStrength.color : 'bg-surface-700'"
              />
            </div>
            <p class="text-xs" :class="passwordStrength.score >= 3 ? 'text-green-400' : 'text-surface-400'">
              {{ passwordStrength.label }}
            </p>
          </div>
        </div>

        <div>
          <label for="confirmPassword" class="block text-sm font-medium text-surface-300 mb-2">
            Confirm password
          </label>
          <div class="relative">
            <Icon
              icon="lucide:lock"
              class="absolute left-3 top-1/2 -translate-y-1/2 w-5 h-5 text-surface-500"
            />
            <input
              id="confirmPassword"
              v-model="form.confirmPassword"
              type="password"
              placeholder="Confirm your password"
              class="w-full pl-10 pr-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
              :disabled="loading"
            />
          </div>
        </div>

        <div class="flex items-start">
          <input
            id="terms"
            v-model="form.agreeTerms"
            type="checkbox"
            class="mt-1 w-4 h-4 rounded border-surface-600 bg-surface-800 text-primary-500 focus:ring-primary-500/50"
          />
          <label for="terms" class="ml-2 text-sm text-surface-400">
            I agree to the
            <NuxtLink to="/terms" class="text-primary-400 hover:text-primary-300">Terms of Service</NuxtLink>
            and
            <NuxtLink to="/privacy" class="text-primary-400 hover:text-primary-300">Privacy Policy</NuxtLink>
          </label>
        </div>

        <button
          type="submit"
          :disabled="loading"
          class="w-full btn-glow py-3 flex items-center justify-center gap-2"
        >
          <Icon v-if="loading" icon="lucide:loader-2" class="w-5 h-5 animate-spin" />
          <span>{{ loading ? 'Creating account...' : 'Create account' }}</span>
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

      <!-- Sign in link -->
      <p class="mt-8 text-center text-sm text-surface-400">
        Already have an account?
        <NuxtLink to="/auth/login" class="text-primary-400 hover:text-primary-300 font-medium transition-colors">
          Sign in
        </NuxtLink>
      </p>
    </template>
  </div>
</template>
