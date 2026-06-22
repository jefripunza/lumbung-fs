<script setup lang="ts">
defineProps<{
  title: string
  subtitle?: string
  type: 'origin' | 'rule' | 'bucket' | 'file'
}>()

const typeIcons: Record<string, string> = {
  origin: '🌐',
  rule: '🛡️',
  bucket: '📦',
  file: '📄',
}
</script>

<template>
  <div class="flow-node" :class="`flow-node--${type}`">
    <div class="flow-node__icon">{{ typeIcons[type] || '●' }}</div>
    <div class="flow-node__content">
      <span class="flow-node__title">{{ title }}</span>
      <span v-if="subtitle" class="flow-node__subtitle">{{ subtitle }}</span>
    </div>
  </div>
</template>

<style scoped>
.flow-node {
  display: flex;
  align-items: center;
  gap: var(--spacing-12);
  background: var(--color-bone-white);
  border-radius: var(--radius-xl);
  padding: 14px 18px;
  box-shadow: var(--shadow-subtle);
  min-width: 200px;
  transition: box-shadow 0.2s ease;
}
.flow-node:hover {
  box-shadow: var(--shadow-md-2);
}
.flow-node__icon {
  font-size: 22px;
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--radius-md);
  background: rgba(133, 192, 147, 0.12);
}
.flow-node--origin .flow-node__icon { background: rgba(67, 55, 135, 0.08); }
.flow-node--rule .flow-node__icon { background: rgba(0, 112, 16, 0.08); }
.flow-node--bucket .flow-node__icon { background: rgba(133, 192, 147, 0.12); }
.flow-node--file .flow-node__icon { background: rgba(108, 122, 121, 0.08); }
.flow-node__content {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.flow-node__title {
  font-family: var(--font-denim);
  font-weight: 500;
  font-size: 14px;
  color: var(--color-forest-ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.flow-node__subtitle {
  font-family: var(--font-muoto);
  font-size: 12px;
  color: var(--color-slate-smoke);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
