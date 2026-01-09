<script setup lang="ts">
import { Icon } from '@iconify/vue'

definePageMeta({
  layout: 'dashboard',
  middleware: ['auth'],
})

useSeoMeta({
  title: 'Settings - Lurus AI',
  description: 'Manage your account settings and preferences.',
})

const { user, profile } = useAuth()
const supabase = useSupabaseClient()

const form = reactive({
  displayName: profile.value?.display_name || '',
  email: user.value?.email || '',
  timezone: profile.value?.timezone || 'Asia/Shanghai',
  language: profile.value?.language || 'zh-CN',
})

const loading = ref(false)
const success = ref(false)

const timezones = [
  { value: 'Asia/Shanghai', label: 'China Standard Time (UTC+8)' },
  { value: 'Asia/Tokyo', label: 'Japan Standard Time (UTC+9)' },
  { value: 'America/New_York', label: 'Eastern Time (UTC-5)' },
  { value: 'America/Los_Angeles', label: 'Pacific Time (UTC-8)' },
  { value: 'Europe/London', label: 'Greenwich Mean Time (UTC+0)' },
  { value: 'Europe/Paris', label: 'Central European Time (UTC+1)' },
]

const languages = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English (US)' },
  { value: 'ja-JP', label: '日本語' },
]

const saveSettings = async () => {
  loading.value = true
  success.value = false

  try {
    const { error } = await supabase
      .from('user_profiles')
      .update({
        display_name: form.displayName,
        timezone: form.timezone,
        language: form.language,
      })
      .eq('id', user.value?.id)

    if (error) throw error

    success.value = true
    setTimeout(() => { success.value = false }, 3000)
  } catch (e) {
    console.error('Failed to save settings:', e)
  } finally {
    loading.value = false
  }
}

// Watch for profile changes
watch(() => profile.value, (newProfile) => {
  if (newProfile) {
    form.displayName = newProfile.display_name || ''
    form.timezone = newProfile.timezone || 'Asia/Shanghai'
    form.language = newProfile.language || 'zh-CN'
  }
}, { immediate: true })
</script>

<template>
  <div class="max-w-3xl space-y-8">
    <!-- Header -->
    <div>
      <h1 class="text-2xl font-bold text-white">Account Settings</h1>
      <p class="text-surface-400 mt-1">Manage your account information and preferences</p>
    </div>

    <!-- Success message -->
    <Transition
      enter-active-class="transition-all duration-300"
      enter-from-class="opacity-0 -translate-y-2"
      enter-to-class="opacity-100 translate-y-0"
      leave-active-class="transition-all duration-200"
      leave-from-class="opacity-100 translate-y-0"
      leave-to-class="opacity-0 -translate-y-2"
    >
      <div
        v-if="success"
        class="p-4 rounded-lg bg-green-500/10 border border-green-500/20 flex items-center gap-3"
      >
        <Icon icon="lucide:check-circle" class="w-5 h-5 text-green-400" />
        <span class="text-green-400">Settings saved successfully</span>
      </div>
    </Transition>

    <!-- Profile section -->
    <div class="glass-card">
      <h2 class="text-lg font-semibold text-white mb-6">Profile</h2>

      <div class="flex items-start gap-6 mb-8">
        <!-- Avatar -->
        <div class="relative">
          <div class="w-20 h-20 rounded-full bg-gradient-to-br from-primary-500 to-accent-500 flex items-center justify-center">
            <span class="text-2xl font-bold text-white">
              {{ form.displayName?.[0]?.toUpperCase() || user?.email?.[0]?.toUpperCase() || 'U' }}
            </span>
          </div>
          <button
            class="absolute -bottom-1 -right-1 w-8 h-8 bg-surface-700 rounded-full flex items-center justify-center border-2 border-surface-900 hover:bg-surface-600 transition-colors"
            title="Change avatar"
          >
            <Icon icon="lucide:camera" class="w-4 h-4 text-surface-300" />
          </button>
        </div>

        <div class="flex-1">
          <h3 class="text-white font-medium">{{ form.displayName || 'Set your name' }}</h3>
          <p class="text-surface-400 text-sm">{{ user?.email }}</p>
          <p class="text-surface-500 text-xs mt-1">
            Member since {{ new Date(user?.created_at || Date.now()).toLocaleDateString() }}
          </p>
        </div>
      </div>

      <form class="space-y-6" @submit.prevent="saveSettings">
        <div>
          <label class="block text-sm font-medium text-surface-300 mb-2">
            Display Name
          </label>
          <input
            v-model="form.displayName"
            type="text"
            placeholder="Your name"
            class="w-full px-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
          />
        </div>

        <div>
          <label class="block text-sm font-medium text-surface-300 mb-2">
            Email Address
          </label>
          <input
            v-model="form.email"
            type="email"
            disabled
            class="w-full px-4 py-3 bg-surface-800/50 border border-surface-700 rounded-lg text-surface-400 cursor-not-allowed"
          />
          <p class="mt-2 text-xs text-surface-500">
            Contact support to change your email address
          </p>
        </div>

        <div class="grid sm:grid-cols-2 gap-6">
          <div>
            <label class="block text-sm font-medium text-surface-300 mb-2">
              Timezone
            </label>
            <select
              v-model="form.timezone"
              class="w-full px-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
            >
              <option
                v-for="tz in timezones"
                :key="tz.value"
                :value="tz.value"
              >
                {{ tz.label }}
              </option>
            </select>
          </div>

          <div>
            <label class="block text-sm font-medium text-surface-300 mb-2">
              Language
            </label>
            <select
              v-model="form.language"
              class="w-full px-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
            >
              <option
                v-for="lang in languages"
                :key="lang.value"
                :value="lang.value"
              >
                {{ lang.label }}
              </option>
            </select>
          </div>
        </div>

        <div class="pt-4">
          <button
            type="submit"
            :disabled="loading"
            class="btn-glow flex items-center gap-2"
          >
            <Icon v-if="loading" icon="lucide:loader-2" class="w-5 h-5 animate-spin" />
            <span>{{ loading ? 'Saving...' : 'Save Changes' }}</span>
          </button>
        </div>
      </form>
    </div>

    <!-- Security section -->
    <div class="glass-card">
      <h2 class="text-lg font-semibold text-white mb-6">Security</h2>

      <div class="space-y-4">
        <div class="flex items-center justify-between p-4 bg-surface-800 rounded-lg">
          <div class="flex items-center gap-3">
            <Icon icon="lucide:lock" class="w-5 h-5 text-surface-400" />
            <div>
              <h3 class="text-white font-medium">Password</h3>
              <p class="text-surface-400 text-sm">Last changed 30 days ago</p>
            </div>
          </div>
          <NuxtLink
            to="/auth/reset-password"
            class="text-primary-400 hover:text-primary-300 text-sm transition-colors"
          >
            Change
          </NuxtLink>
        </div>

        <div class="flex items-center justify-between p-4 bg-surface-800 rounded-lg">
          <div class="flex items-center gap-3">
            <Icon icon="lucide:smartphone" class="w-5 h-5 text-surface-400" />
            <div>
              <h3 class="text-white font-medium">Two-Factor Authentication</h3>
              <p class="text-surface-400 text-sm">Add an extra layer of security</p>
            </div>
          </div>
          <button class="text-primary-400 hover:text-primary-300 text-sm transition-colors">
            Enable
          </button>
        </div>

        <div class="flex items-center justify-between p-4 bg-surface-800 rounded-lg">
          <div class="flex items-center gap-3">
            <Icon icon="lucide:monitor-smartphone" class="w-5 h-5 text-surface-400" />
            <div>
              <h3 class="text-white font-medium">Active Sessions</h3>
              <p class="text-surface-400 text-sm">Manage your active devices</p>
            </div>
          </div>
          <NuxtLink
            to="/dashboard/devices"
            class="text-primary-400 hover:text-primary-300 text-sm transition-colors"
          >
            View
          </NuxtLink>
        </div>
      </div>
    </div>

    <!-- Danger zone -->
    <div class="glass-card border-red-500/20">
      <h2 class="text-lg font-semibold text-red-400 mb-6">Danger Zone</h2>

      <div class="flex items-center justify-between p-4 bg-red-500/5 rounded-lg border border-red-500/20">
        <div>
          <h3 class="text-white font-medium">Delete Account</h3>
          <p class="text-surface-400 text-sm">Permanently delete your account and all data</p>
        </div>
        <button class="px-4 py-2 bg-red-500/20 text-red-400 rounded-lg hover:bg-red-500/30 transition-colors">
          Delete
        </button>
      </div>
    </div>
  </div>
</template>
