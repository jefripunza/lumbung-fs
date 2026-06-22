<script setup lang="ts">
import { onMounted } from 'vue'
import { useOriginsStore } from '@/stores/origins'
import SageCard from '@/components/SageCard.vue'
import PillTag from '@/components/PillTag.vue'
import MossButton from '@/components/MossButton.vue'
import OutlineButton from '@/components/OutlineButton.vue'

const store = useOriginsStore()

onMounted(() => {
  store.fetchUnknownOrigins()
})

async function handlePromote(id: string) {
  await store.promoteUnknownOrigin(id)
}

async function handleDelete(id: string) {
  if (confirm('Remove this unknown origin log entry?')) {
    await store.deleteUnknownOrigin(id)
  }
}

async function handleClearAll() {
  if (confirm('Clear all unknown origin entries? This cannot be undone.')) {
    await store.clearAllUnknownOrigins()
  }
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}
</script>

<template>
  <div class="unknown-page">
    <!-- Header -->
    <div class="unknown-page__header">
      <div>
        <h1 class="unknown-page__title">Unknown Origins</h1>
        <p class="unknown-page__desc">Unregistered domains that attempted to access the system</p>
      </div>
      <OutlineButton v-if="store.unknownOrigins.length > 0" variant="danger" @click="handleClearAll">
        Clear All
      </OutlineButton>
    </div>

    <!-- Loading -->
    <div v-if="store.isLoading" class="unknown-page__loading">
      <span class="unknown-page__spinner"></span>
      Loading…
    </div>

    <!-- Empty -->
    <div v-else-if="store.unknownOrigins.length === 0" class="unknown-page__empty">
      <p>No unknown origin access attempts recorded.</p>
    </div>

    <!-- List -->
    <div v-else class="unknown-page__list">
      <SageCard
        v-for="entry in store.unknownOrigins"
        :key="entry.id"
        class="unknown-card"
      >
        <div class="unknown-card__main">
          <span class="unknown-card__icon">⚠️</span>
          <div class="unknown-card__info">
            <span class="unknown-card__domain">{{ entry.domain }}</span>
            <div class="unknown-card__meta">
              <PillTag>{{ entry.ip_address || 'unknown ip' }}</PillTag>
              <span class="unknown-card__time">{{ formatDate(entry.access_at) }}</span>
            </div>
          </div>
        </div>
        <div class="unknown-card__actions">
          <MossButton @click="handlePromote(entry.id)">Approve</MossButton>
          <OutlineButton variant="danger" @click="handleDelete(entry.id)">Dismiss</OutlineButton>
        </div>
      </SageCard>
    </div>
  </div>
</template>

<style scoped>
.unknown-page {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-24);
}
.unknown-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-16);
}
.unknown-page__title {
  font-family: var(--font-denim);
  font-size: var(--text-heading);
  font-weight: 600;
  color: var(--color-forest-ink);
  margin: 0;
}
.unknown-page__desc {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  margin: 2px 0 0;
}
.unknown-page__loading {
  display: flex;
  align-items: center;
  gap: var(--spacing-12);
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  padding: var(--spacing-48) 0;
}
.unknown-page__spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--color-lichen);
  border-top-color: var(--color-moss);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.unknown-page__empty {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  text-align: center;
  padding: var(--spacing-48) 0;
}

/* ───── List ───── */
.unknown-page__list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-12);
}

/* ───── Card ───── */
.unknown-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-16);
  padding: var(--spacing-16) var(--spacing-20);
  transition: box-shadow 0.2s ease;
}
.unknown-card:hover {
  box-shadow: var(--shadow-md-2);
}
.unknown-card__main {
  display: flex;
  align-items: center;
  gap: var(--spacing-12);
  min-width: 0;
  flex: 1;
}
.unknown-card__icon {
  font-size: 22px;
  flex-shrink: 0;
}
.unknown-card__info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.unknown-card__domain {
  font-family: var(--font-denim);
  font-weight: 500;
  font-size: 16px;
  color: var(--color-forest-ink);
  word-break: break-all;
}
.unknown-card__meta {
  display: flex;
  align-items: center;
  gap: var(--spacing-8);
  flex-wrap: wrap;
}
.unknown-card__time {
  font-family: var(--font-muoto);
  font-size: 12px;
  color: var(--color-slate-smoke);
}
.unknown-card__actions {
  display: flex;
  gap: var(--spacing-8);
  flex-shrink: 0;
}
</style>
