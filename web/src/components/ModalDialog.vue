<script setup lang="ts">
import { ref, onMounted } from 'vue'

defineProps<{
  title?: string
  maxWidth?: string
}>()

defineEmits<{
  close: []
}>()

const isVisible = ref(false)
onMounted(() => {
  isVisible.value = true
})
</script>

<template>
  <Teleport to="body">
    <Transition name="modal" appear>
      <div v-if="isVisible" class="modal-overlay">
        <div class="modal-dialog" :style="maxWidth ? `max-width: ${maxWidth}` : undefined">
          <div v-if="title" class="modal-header">
            <h3 class="modal-title">{{ title }}</h3>
            <button class="modal-close" @click="$emit('close')" aria-label="Close">✕</button>
          </div>
          <div class="modal-body">
            <slot />
          </div>
          <div v-if="$slots.footer" class="modal-footer">
            <slot name="footer" />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(9, 53, 46, 0.35);
  backdrop-filter: blur(4px);
}
.modal-dialog {
  background: var(--color-bone-white);
  border-radius: var(--radius-2xl);
  box-shadow: var(--shadow-md);
  padding: 0;
  max-width: 500px;
  width: 92vw;
  max-height: 85vh;
  overflow-y: auto;
  animation: modal-slide-in 0.2s ease-out;
}
.modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--spacing-20) var(--spacing-24);
  border-bottom: 0.5px solid var(--color-lichen);
}
.modal-title {
  font-family: var(--font-denim);
  font-weight: 600;
  font-size: 18px;
  color: var(--color-forest-ink);
  margin: 0;
}
.modal-close {
  background: none;
  border: none;
  font-size: 18px;
  color: var(--color-slate-smoke);
  cursor: pointer;
  padding: 4px;
  line-height: 1;
}
.modal-close:hover {
  color: var(--color-forest-ink);
}
.modal-body {
  padding: var(--spacing-24);
}
.modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--spacing-12);
  padding: var(--spacing-16) var(--spacing-24);
  border-top: 0.5px solid var(--color-lichen);
}

/* Transition */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease;
}
.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

@keyframes modal-slide-in {
  from {
    transform: translateY(12px);
    opacity: 0;
  }
  to {
    transform: translateY(0);
    opacity: 1;
  }
}
</style>
