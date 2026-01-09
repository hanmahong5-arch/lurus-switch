<script setup lang="ts">
import { Icon } from '@iconify/vue'

definePageMeta({
  layout: 'dashboard',
  middleware: ['auth'],
})

useSeoMeta({
  title: 'Devices - Lurus AI',
  description: 'Manage your connected devices and sessions.',
})

interface Device {
  id: string
  name: string
  type: 'desktop' | 'mobile' | 'cli' | 'api'
  platform: string
  lastActive: string
  location: string
  isActive: boolean
  isCurrent: boolean
}

const devices = ref<Device[]>([
  {
    id: '1',
    name: 'Windows PC',
    type: 'desktop',
    platform: 'Windows 11',
    lastActive: 'Now',
    location: 'Beijing, China',
    isActive: true,
    isCurrent: true,
  },
  {
    id: '2',
    name: 'Claude Code CLI',
    type: 'cli',
    platform: 'Windows Terminal',
    lastActive: '2 hours ago',
    location: 'Beijing, China',
    isActive: true,
    isCurrent: false,
  },
  {
    id: '3',
    name: 'MacBook Pro',
    type: 'desktop',
    platform: 'macOS Ventura',
    lastActive: '3 days ago',
    location: 'Shanghai, China',
    isActive: false,
    isCurrent: false,
  },
  {
    id: '4',
    name: 'iPhone 15',
    type: 'mobile',
    platform: 'iOS 17',
    lastActive: '1 week ago',
    location: 'Hangzhou, China',
    isActive: false,
    isCurrent: false,
  },
])

const getDeviceIcon = (type: string) => {
  switch (type) {
    case 'desktop': return 'lucide:monitor'
    case 'mobile': return 'lucide:smartphone'
    case 'cli': return 'lucide:terminal'
    case 'api': return 'lucide:code'
    default: return 'lucide:laptop'
  }
}

const revokeDevice = async (id: string) => {
  devices.value = devices.value.filter(d => d.id !== id)
}

const revokeAllOther = async () => {
  devices.value = devices.value.filter(d => d.isCurrent)
}
</script>

<template>
  <div class="space-y-8">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-white">Connected Devices</h1>
        <p class="text-surface-400 mt-1">Manage devices and sessions connected to your account</p>
      </div>
      <button
        class="px-4 py-2 bg-surface-800 text-white rounded-lg hover:bg-surface-700 transition-colors"
        @click="revokeAllOther"
      >
        Sign out all other devices
      </button>
    </div>

    <!-- Device list -->
    <div class="space-y-4">
      <div
        v-for="device in devices"
        :key="device.id"
        class="glass-card"
      >
        <div class="flex items-start gap-4">
          <!-- Device icon -->
          <div
            class="w-12 h-12 rounded-lg flex items-center justify-center"
            :class="device.isActive ? 'bg-green-500/20' : 'bg-surface-700'"
          >
            <Icon
              :icon="getDeviceIcon(device.type)"
              class="w-6 h-6"
              :class="device.isActive ? 'text-green-400' : 'text-surface-400'"
            />
          </div>

          <!-- Device info -->
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 mb-1">
              <h3 class="text-white font-medium truncate">{{ device.name }}</h3>
              <span
                v-if="device.isCurrent"
                class="px-2 py-0.5 text-xs rounded-full bg-primary-500/20 text-primary-400"
              >
                This device
              </span>
              <span
                v-else-if="device.isActive"
                class="px-2 py-0.5 text-xs rounded-full bg-green-500/20 text-green-400"
              >
                Active
              </span>
            </div>
            <p class="text-surface-400 text-sm">{{ device.platform }}</p>
            <div class="flex items-center gap-4 mt-2 text-xs text-surface-500">
              <span class="flex items-center gap-1">
                <Icon icon="lucide:clock" class="w-3 h-3" />
                {{ device.lastActive }}
              </span>
              <span class="flex items-center gap-1">
                <Icon icon="lucide:map-pin" class="w-3 h-3" />
                {{ device.location }}
              </span>
            </div>
          </div>

          <!-- Actions -->
          <div v-if="!device.isCurrent" class="flex items-center">
            <button
              class="px-3 py-1.5 text-sm text-red-400 hover:text-red-300 hover:bg-red-500/10 rounded-lg transition-colors"
              @click="revokeDevice(device.id)"
            >
              Sign out
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Info box -->
    <div class="p-4 rounded-lg bg-surface-800/50 border border-surface-700">
      <div class="flex items-start gap-3">
        <Icon icon="lucide:info" class="w-5 h-5 text-surface-400 flex-shrink-0 mt-0.5" />
        <div>
          <h4 class="text-white font-medium mb-1">About connected devices</h4>
          <p class="text-sm text-surface-400">
            This list shows all devices and applications that have accessed your Lurus AI account.
            If you don't recognize a device, sign it out immediately and change your password.
          </p>
        </div>
      </div>
    </div>
  </div>
</template>
