<script setup lang="ts">
import { Icon } from '@iconify/vue'

definePageMeta({
  layout: 'auth',
})

useSeoMeta({
  title: 'Authenticating... - Lurus AI',
})

const { syncProfile } = useAuth()
const route = useRoute()
const error = ref<string | null>(null)

onMounted(async () => {
  try {
    // Supabase handles the OAuth callback automatically
    // We just need to sync the profile and redirect
    await syncProfile()

    // Get redirect URL from query or default to dashboard
    const redirect = route.query.redirect as string || '/dashboard'
    await navigateTo(redirect)
  } catch (e: any) {
    error.value = e.message || 'Authentication failed'
  }
})
</script>

<template>
  <div class="text-center">
    <template v-if="error">
      <div class="w-16 h-16 mx-auto mb-6 rounded-full bg-red-500/20 flex items-center justify-center">
        <Icon icon="lucide:x" class="w-8 h-8 text-red-400" />
      </div>
      <h2 class="text-2xl font-bold text-white mb-2">Authentication Failed</h2>
      <p class="text-surface-400 mb-8">{{ error }}</p>
      <NuxtLink to="/auth/login" class="btn-glow inline-flex items-center gap-2">
        <Icon icon="lucide:arrow-left" class="w-4 h-4" />
        <span>Back to Sign In</span>
      </NuxtLink>
    </template>
    <template v-else>
      <div class="w-16 h-16 mx-auto mb-6 rounded-full bg-primary-500/20 flex items-center justify-center">
        <Icon icon="lucide:loader-2" class="w-8 h-8 text-primary-400 animate-spin" />
      </div>
      <h2 class="text-2xl font-bold text-white mb-2">Authenticating...</h2>
      <p class="text-surface-400">Please wait while we complete your sign in.</p>
    </template>
  </div>
</template>
