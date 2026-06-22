<script setup lang="ts">
defineProps<{
  disabled?: boolean
  variant?: 'default' | 'danger'
}>()

defineEmits<{
  click: [e: MouseEvent]
}>()
</script>

<template>
  <button
    class="outline-btn"
    :class="{ 'outline-btn--danger': variant === 'danger' }"
    :disabled="disabled"
    @click="$emit('click', $event)"
  >
    <slot />
  </button>
</template>

<style scoped>
.outline-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  background: transparent;
  color: var(--color-forest-ink);
  border-radius: var(--radius-xl);
  padding: 10px 20px;
  font-family: var(--font-denim);
  font-weight: 500;
  font-size: 14px;
  cursor: pointer;
  transition: background 0.15s ease, transform 0.1s ease;
  box-shadow: var(--shadow-subtle-2);
  white-space: nowrap;
}
.outline-btn:hover {
  background: rgba(9, 53, 46, 0.04);
}
.outline-btn:active {
  transform: scale(0.97);
}
.outline-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.outline-btn--danger {
  color: #8b2020;
  box-shadow: inset 0 0 0 0.5px #c44d4d;
}
.outline-btn--danger:hover {
  background: rgba(196, 77, 77, 0.06);
}
</style>
