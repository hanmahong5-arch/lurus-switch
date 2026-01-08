<script setup lang="ts">
</script>

<template>
  <div class="flex flex-col min-h-screen">
    <LayoutNavbar />
    <main class="flex-1">
      <slot />
    </main>
    <LayoutFooter />
  </div>
</template>
