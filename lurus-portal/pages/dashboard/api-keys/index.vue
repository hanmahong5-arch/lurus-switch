<script setup lang="ts">
import { Icon } from '@iconify/vue'

definePageMeta({
  layout: 'dashboard',
  middleware: ['auth'],
})

useSeoMeta({
  title: 'API Keys - Lurus AI',
  description: 'Manage your API keys for accessing Lurus AI services.',
})

interface ApiKey {
  id: string
  name: string
  prefix: string
  createdAt: string
  lastUsedAt: string | null
  status: 'active' | 'revoked' | 'expired'
}

const { profile } = useAuth()

const apiKeys = ref<ApiKey[]>([
  {
    id: '1',
    name: 'Production Key',
    prefix: 'sk-lur...x7Kp',
    createdAt: '2026-01-05',
    lastUsedAt: '2026-01-08 14:30',
    status: 'active',
  },
  {
    id: '2',
    name: 'Development Key',
    prefix: 'sk-lur...m3Nq',
    createdAt: '2026-01-02',
    lastUsedAt: '2026-01-07 09:15',
    status: 'active',
  },
])

const isCreateModalOpen = ref(false)
const newKeyName = ref('')
const createdKey = ref<string | null>(null)
const loading = ref(false)

const createKey = async () => {
  if (!newKeyName.value.trim()) return

  loading.value = true

  // Simulate API call
  await new Promise(resolve => setTimeout(resolve, 1000))

  const newKey = {
    id: String(Date.now()),
    name: newKeyName.value,
    prefix: 'sk-lur...' + Math.random().toString(36).slice(2, 6),
    createdAt: new Date().toISOString().split('T')[0],
    lastUsedAt: null,
    status: 'active' as const,
  }

  apiKeys.value.unshift(newKey)
  createdKey.value = 'sk-lur-' + Math.random().toString(36).slice(2) + Math.random().toString(36).slice(2)
  newKeyName.value = ''
  loading.value = false
}

const copyKey = async (key: string) => {
  await navigator.clipboard.writeText(key)
  // Show toast notification
}

const closeCreateModal = () => {
  isCreateModalOpen.value = false
  createdKey.value = null
  newKeyName.value = ''
}

const revokeKey = async (id: string) => {
  const key = apiKeys.value.find(k => k.id === id)
  if (key) {
    key.status = 'revoked'
  }
}

const deleteKey = async (id: string) => {
  apiKeys.value = apiKeys.value.filter(k => k.id !== id)
}

const getStatusColor = (status: string) => {
  switch (status) {
    case 'active': return 'bg-green-500/20 text-green-400'
    case 'revoked': return 'bg-red-500/20 text-red-400'
    case 'expired': return 'bg-yellow-500/20 text-yellow-400'
    default: return 'bg-surface-700 text-surface-400'
  }
}
</script>

<template>
  <div class="space-y-8">
    <!-- Header -->
    <div class="flex flex-col sm:flex-row sm:items-center sm:justify-between gap-4">
      <div>
        <h1 class="text-2xl font-bold text-white">API Keys</h1>
        <p class="text-surface-400 mt-1">Manage your API keys for accessing Lurus AI services</p>
      </div>
      <button
        class="btn-glow flex items-center gap-2"
        @click="isCreateModalOpen = true"
      >
        <Icon icon="lucide:plus" class="w-5 h-5" />
        <span>Create New Key</span>
      </button>
    </div>

    <!-- Info banner -->
    <div class="p-4 rounded-lg bg-primary-500/10 border border-primary-500/20 flex items-start gap-3">
      <Icon icon="lucide:info" class="w-5 h-5 text-primary-400 flex-shrink-0 mt-0.5" />
      <div>
        <p class="text-sm text-primary-300">
          API keys provide access to Lurus AI services. Keep your keys secure and never share them publicly.
        </p>
      </div>
    </div>

    <!-- API Keys list -->
    <div class="glass-card">
      <div v-if="apiKeys.length === 0" class="text-center py-12">
        <Icon icon="lucide:key" class="w-12 h-12 text-surface-600 mx-auto mb-4" />
        <h3 class="text-lg font-medium text-white mb-2">No API Keys</h3>
        <p class="text-surface-400 mb-6">Create your first API key to get started</p>
        <button
          class="btn-glow inline-flex items-center gap-2"
          @click="isCreateModalOpen = true"
        >
          <Icon icon="lucide:plus" class="w-5 h-5" />
          <span>Create API Key</span>
        </button>
      </div>

      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead>
            <tr class="text-left text-surface-400 text-sm border-b border-surface-700">
              <th class="pb-4 font-medium">Name</th>
              <th class="pb-4 font-medium">Key</th>
              <th class="pb-4 font-medium hidden sm:table-cell">Created</th>
              <th class="pb-4 font-medium hidden md:table-cell">Last Used</th>
              <th class="pb-4 font-medium">Status</th>
              <th class="pb-4 font-medium text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="key in apiKeys"
              :key="key.id"
              class="border-b border-surface-700/50 last:border-0"
            >
              <td class="py-4">
                <span class="text-white font-medium">{{ key.name }}</span>
              </td>
              <td class="py-4">
                <code class="px-2 py-1 rounded bg-surface-700 text-surface-300 text-sm font-mono">
                  {{ key.prefix }}
                </code>
              </td>
              <td class="py-4 text-surface-400 hidden sm:table-cell">{{ key.createdAt }}</td>
              <td class="py-4 text-surface-400 hidden md:table-cell">
                {{ key.lastUsedAt || 'Never' }}
              </td>
              <td class="py-4">
                <span
                  class="px-2 py-1 rounded text-xs font-medium capitalize"
                  :class="getStatusColor(key.status)"
                >
                  {{ key.status }}
                </span>
              </td>
              <td class="py-4 text-right">
                <div class="flex items-center justify-end gap-2">
                  <button
                    v-if="key.status === 'active'"
                    class="p-2 text-surface-400 hover:text-yellow-400 rounded-lg hover:bg-surface-700 transition-colors"
                    title="Revoke key"
                    @click="revokeKey(key.id)"
                  >
                    <Icon icon="lucide:ban" class="w-4 h-4" />
                  </button>
                  <button
                    class="p-2 text-surface-400 hover:text-red-400 rounded-lg hover:bg-surface-700 transition-colors"
                    title="Delete key"
                    @click="deleteKey(key.id)"
                  >
                    <Icon icon="lucide:trash-2" class="w-4 h-4" />
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <!-- Usage example -->
    <div class="glass-card">
      <h2 class="text-lg font-semibold text-white mb-4">Quick Start</h2>
      <p class="text-surface-400 mb-4">Use your API key to make requests to Lurus AI:</p>

      <div class="bg-surface-800 rounded-lg p-4 overflow-x-auto">
        <pre class="text-sm text-surface-300 font-mono"><code>curl https://ai.lurus.cn/v1/chat/completions \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-4-turbo",
    "messages": [{"role": "user", "content": "Hello!"}]
  }'</code></pre>
      </div>

      <NuxtLink
        to="/docs/quickstart"
        class="inline-flex items-center gap-2 mt-4 text-sm text-primary-400 hover:text-primary-300 transition-colors"
      >
        <span>View full documentation</span>
        <Icon icon="lucide:arrow-right" class="w-4 h-4" />
      </NuxtLink>
    </div>

    <!-- Create Key Modal -->
    <Teleport to="body">
      <Transition
        enter-active-class="transition-all duration-300"
        enter-from-class="opacity-0"
        enter-to-class="opacity-100"
        leave-active-class="transition-all duration-200"
        leave-from-class="opacity-100"
        leave-to-class="opacity-0"
      >
        <div
          v-if="isCreateModalOpen"
          class="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/50 backdrop-blur-sm"
          @click.self="closeCreateModal"
        >
          <div
            class="w-full max-w-md bg-surface-900 rounded-xl border border-surface-700 shadow-xl"
          >
            <div class="p-6">
              <div class="flex items-center justify-between mb-6">
                <h3 class="text-xl font-semibold text-white">
                  {{ createdKey ? 'API Key Created' : 'Create New API Key' }}
                </h3>
                <button
                  class="p-2 text-surface-400 hover:text-white rounded-lg hover:bg-surface-800 transition-colors"
                  @click="closeCreateModal"
                >
                  <Icon icon="lucide:x" class="w-5 h-5" />
                </button>
              </div>

              <!-- Key created success -->
              <template v-if="createdKey">
                <div class="p-4 rounded-lg bg-green-500/10 border border-green-500/20 mb-6">
                  <div class="flex items-center gap-2 mb-2">
                    <Icon icon="lucide:check-circle" class="w-5 h-5 text-green-400" />
                    <span class="text-green-400 font-medium">Key created successfully!</span>
                  </div>
                  <p class="text-sm text-surface-400">
                    Make sure to copy your API key now. You won't be able to see it again.
                  </p>
                </div>

                <div class="relative">
                  <input
                    type="text"
                    :value="createdKey"
                    readonly
                    class="w-full px-4 py-3 pr-12 bg-surface-800 border border-surface-700 rounded-lg text-white font-mono text-sm"
                  />
                  <button
                    class="absolute right-2 top-1/2 -translate-y-1/2 p-2 text-surface-400 hover:text-white rounded transition-colors"
                    @click="copyKey(createdKey)"
                  >
                    <Icon icon="lucide:copy" class="w-5 h-5" />
                  </button>
                </div>

                <button
                  class="w-full mt-6 btn-glow py-3"
                  @click="closeCreateModal"
                >
                  Done
                </button>
              </template>

              <!-- Create form -->
              <template v-else>
                <div class="space-y-4">
                  <div>
                    <label class="block text-sm font-medium text-surface-300 mb-2">
                      Key Name
                    </label>
                    <input
                      v-model="newKeyName"
                      type="text"
                      placeholder="e.g., Production Key"
                      class="w-full px-4 py-3 bg-surface-800 border border-surface-700 rounded-lg text-white placeholder-surface-500 focus:outline-none focus:ring-2 focus:ring-primary-500/50 focus:border-primary-500 transition-colors"
                    />
                    <p class="mt-2 text-xs text-surface-500">
                      Give your key a descriptive name to identify it later
                    </p>
                  </div>
                </div>

                <div class="flex gap-3 mt-6">
                  <button
                    class="flex-1 px-4 py-3 bg-surface-800 text-white rounded-lg hover:bg-surface-700 transition-colors"
                    @click="closeCreateModal"
                  >
                    Cancel
                  </button>
                  <button
                    class="flex-1 btn-glow py-3 flex items-center justify-center gap-2"
                    :disabled="loading || !newKeyName.trim()"
                    @click="createKey"
                  >
                    <Icon v-if="loading" icon="lucide:loader-2" class="w-5 h-5 animate-spin" />
                    <span>{{ loading ? 'Creating...' : 'Create Key' }}</span>
                  </button>
                </div>
              </template>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>
