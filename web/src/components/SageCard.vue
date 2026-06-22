<script setup lang="ts">
defineProps<{
  padding?: string
}>()
</script>

<template>
  <div class="sage-card" :style="padding ? `padding: ${padding}` : undefined">
    <slot />
  </div>
</template>

<style scoped>
.sage-card {
  background: var(--color-bone-white);
  border-radius: var(--radius-xl);
  padding: var(--spacing-20);
  box-shadow: var(--shadow-subtle);
}
</style>
