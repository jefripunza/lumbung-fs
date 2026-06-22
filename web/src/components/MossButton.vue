<script setup lang="ts">
defineProps<{
  disabled?: boolean
  type?: 'button' | 'submit'
}>()

defineEmits<{
  click: [e: MouseEvent]
}>()
</script>

<template>
  <button
    class="moss-btn"
    :type="type || 'button'"
    :disabled="disabled"
    @click="$emit('click', $event)"
  >
    <slot />
  </button>
</template>

<style scoped>
.moss-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: var(--color-moss);
  color: var(--color-forest-ink);
  border: none;
  border-radius: var(--radius-xl);
  padding: 10px 20px;
  font-family: var(--font-denim);
  font-weight: 500;
  font-size: 14px;
  cursor: pointer;
  transition: opacity 0.15s ease, transform 0.1s ease;
  white-space: nowrap;
}
.moss-btn:hover {
  opacity: 0.88;
}
.moss-btn:active {
  transform: scale(0.97);
}
.moss-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
</style>
