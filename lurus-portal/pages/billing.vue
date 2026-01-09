<script setup lang="ts">
// Redirect old billing URL to dashboard/usage
definePageMeta({
  middleware: [
    function () {
      return navigateTo('/dashboard/usage', { redirectCode: 301 })
    }
  ],
})
</script>

<template>
  <div />
</template>
