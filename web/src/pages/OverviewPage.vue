<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import StatCard from '@/components/StatCard.vue'
import FlowNode from '@/components/FlowNode.vue'
import { useOriginsStore } from '@/stores/origins'
import { useRulesStore } from '@/stores/rules'

const originsStore = useOriginsStore()
const rulesStore = useRulesStore()

const isLoading = ref(true)

onMounted(async () => {
  try {
    await Promise.all([
      originsStore.fetchOrigins(),
      originsStore.fetchUnknownOrigins(),
      rulesStore.fetchRules(),
    ])
  } finally {
    isLoading.value = false
  }
})

const totalOrigins = computed(() => originsStore.origins.length)
const activeOrigins = computed(() => originsStore.origins.filter((o) => !o.is_blocked).length)
const totalRules = computed(() => rulesStore.rules.length)
const unknownCount = computed(() => originsStore.unknownOrigins.length)

// Build flow visualization from first 3 origins
const flowOrigins = computed(() =>
  originsStore.origins.slice(0, 3).map((origin) => {
    const originRules = rulesStore.rules.filter((r) => r.origin_id === origin.id)
    return { origin, rules: originRules }
  }),
)
</script>

<template>
  <div class="overview">
    <!-- Page Header -->
    <div class="overview__header">
      <h1 class="overview__title">System Overview</h1>
      <p class="overview__desc">Infrastructure health and request lifecycle visualization</p>
    </div>

    <!-- Stat Cards Grid -->
    <div class="overview__stats">
      <StatCard label="Total Origins" :value="totalOrigins" />
      <StatCard label="Active Origins" :value="activeOrigins" trend="up" />
      <StatCard label="Total Rules" :value="totalRules" />
      <StatCard
        label="Unknown Origins"
        :value="unknownCount"
        :trend="unknownCount > 0 ? 'down' : 'flat'"
      />
    </div>

    <!-- Architecture Flow Diagram -->
    <div class="overview__flow-section">
      <h2 class="overview__flow-title">Request Lifecycle</h2>
      <p class="overview__flow-desc">Origin → Rule Validation → Storage Bucket → File Access</p>

      <div v-if="isLoading" class="overview__loading">
        <span class="overview__spinner"></span>
        Loading system state…
      </div>

      <div v-else-if="flowOrigins.length === 0" class="overview__empty">
        <p>No origins configured yet. Add your first origin to see the architecture flow.</p>
      </div>

      <div v-else class="overview__flow-canvas">
        <div v-for="flow in flowOrigins" :key="flow.origin.id" class="flow-row">
          <!-- Origin -->
          <FlowNode type="origin" :title="flow.origin.domain" subtitle="Origin Domain" />

          <div class="flow-connector">
            <div class="flow-connector__line"></div>
            <div class="flow-connector__arrow">→</div>
          </div>

          <!-- Rules -->
          <div v-if="flow.rules.length > 0" class="flow-rules-group">
            <FlowNode
              v-for="rule in flow.rules.slice(0, 2)"
              :key="rule.id"
              type="rule"
              :title="rule.path"
              :subtitle="rule.validate_method || 'No validation'"
            />
          </div>
          <FlowNode v-else type="rule" title="No rules" subtitle="Open access" />

          <div class="flow-connector">
            <div class="flow-connector__line"></div>
            <div class="flow-connector__arrow">→</div>
          </div>

          <!-- Bucket -->
          <FlowNode
            type="bucket"
            :title="`bucket/${flow.origin.domain.replace(/\\./g, '_')}`"
            subtitle="Storage Path"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.overview {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-32);
}

/* ───── Header ───── */
.overview__header {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.overview__title {
  font-family: var(--font-denim);
  font-size: var(--text-heading);
  font-weight: 600;
  line-height: var(--leading-heading);
  color: var(--color-forest-ink);
  margin: 0;
}
.overview__desc {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  margin: 0;
}

/* ───── Stats ───── */
.overview__stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--spacing-16);
}

/* ───── Flow Section ───── */
.overview__flow-section {
  background: var(--color-bone-white);
  border-radius: var(--radius-2xl);
  padding: var(--spacing-32);
  box-shadow: var(--shadow-subtle);
}
.overview__flow-title {
  font-family: var(--font-denim);
  font-size: 20px;
  font-weight: 600;
  color: var(--color-forest-ink);
  margin: 0 0 4px;
}
.overview__flow-desc {
  font-family: var(--font-muoto);
  font-size: 13px;
  color: var(--color-slate-smoke);
  margin: 0 0 var(--spacing-24);
}

.overview__loading {
  display: flex;
  align-items: center;
  gap: var(--spacing-12);
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  padding: var(--spacing-32) 0;
}
.overview__spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--color-lichen);
  border-top-color: var(--color-moss);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
.overview__empty {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  padding: var(--spacing-32) 0;
  text-align: center;
}

/* ───── Flow Canvas ───── */
.overview__flow-canvas {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-24);
}
.flow-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-16);
  flex-wrap: wrap;
}
.flow-connector {
  display: flex;
  align-items: center;
  gap: 4px;
}
.flow-connector__line {
  width: 32px;
  height: 1px;
  background: linear-gradient(90deg, var(--color-lichen), var(--color-moss));
}
.flow-connector__arrow {
  font-size: 14px;
  color: var(--color-moss);
  font-family: var(--font-denim);
}
.flow-rules-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
</style>
