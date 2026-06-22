<script setup lang="ts">
defineProps<{
  label: string
  value: string | number
  trend?: 'up' | 'down' | 'flat'
}>()
</script>

<template>
  <div class="stat-card">
    <span class="stat-card__label">{{ label }}</span>
    <span class="stat-card__value">{{ value }}</span>
    <span v-if="trend" class="stat-card__trend" :class="`stat-card__trend--${trend}`">
      {{ trend === 'up' ? '↑' : trend === 'down' ? '↓' : '—' }}
    </span>
  </div>
</template>

<style scoped>
.stat-card {
  background: var(--color-bone-white);
  border-radius: var(--radius-xl);
  padding: var(--spacing-20);
  box-shadow: var(--shadow-subtle);
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 160px;
}
.stat-card__label {
  font-family: var(--font-muoto);
  font-size: 12px;
  font-weight: 500;
  color: var(--color-slate-smoke);
  text-transform: uppercase;
  letter-spacing: -0.01em;
}
.stat-card__value {
  font-family: var(--font-denim);
  font-size: 32px;
  font-weight: 500;
  line-height: 1.19;
  color: var(--color-forest-ink);
}
.stat-card__trend {
  font-family: var(--font-muoto);
  font-size: 13px;
  font-weight: 400;
}
.stat-card__trend--up {
  color: var(--color-deep-fern);
}
.stat-card__trend--down {
  color: #8b2020;
}
.stat-card__trend--flat {
  color: var(--color-slate-smoke);
}
</style>
