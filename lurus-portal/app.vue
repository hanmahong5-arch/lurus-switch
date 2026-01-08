<script setup lang="ts">
// Force dark mode
useHead({
  htmlAttrs: {
    class: 'dark',
  },
})
</script>

<template>
  <div class="min-h-screen bg-surface-900">
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </div>
</template>

<style>
/* Global styles handled by tailwind.css */
</style>
