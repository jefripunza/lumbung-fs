<script setup lang="ts">
defineProps<{
  variant?: 'default' | 'success' | 'warning' | 'danger'
}>()
</script>

<template>
  <span
    class="pill-tag"
    :class="{
      'pill-tag--success': variant === 'success',
      'pill-tag--warning': variant === 'warning',
      'pill-tag--danger': variant === 'danger',
    }"
  >
    <slot />
  </span>
</template>

<style scoped>
.pill-tag {
  display: inline-flex;
  align-items: center;
  border-radius: var(--radius-md);
  border: 1px solid var(--color-lichen);
  background: transparent;
  padding: 2px 10px;
  font-family: var(--font-cinetype);
  font-size: 12px;
  font-weight: 400;
  letter-spacing: 0.30em;
  text-transform: uppercase;
  color: var(--color-forest-ink);
  line-height: 1.4;
  white-space: nowrap;
}
.pill-tag--success {
  border-color: var(--color-moss);
  color: var(--color-deep-fern);
  background: rgba(133, 192, 147, 0.1);
}
.pill-tag--warning {
  border-color: #d4a017;
  color: #8b6914;
  background: rgba(212, 160, 23, 0.08);
}
.pill-tag--danger {
  border-color: #c44d4d;
  color: #8b2020;
  background: rgba(196, 77, 77, 0.08);
}
</style>
