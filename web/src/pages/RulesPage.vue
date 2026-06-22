<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useRulesStore, type Rule } from '@/stores/rules'
import { useOriginsStore } from '@/stores/origins'
import SageCard from '@/components/SageCard.vue'
import PillTag from '@/components/PillTag.vue'
import MossButton from '@/components/MossButton.vue'
import OutlineButton from '@/components/OutlineButton.vue'
import ModalDialog from '@/components/ModalDialog.vue'

const rulesStore = useRulesStore()
const originsStore = useOriginsStore()

const showCreateModal = ref(false)
const showEditModal = ref(false)
const editTarget = ref<Rule | null>(null)
const filterOriginId = ref('')

// New rule form
const form = ref({
  origin_id: '',
  path: '',
  validate_method: '',
  validate_url: '',
  validate_fallback_url: '',
  is_max_size: false,
  value_max_size: 0,
  value_unit_size: 'MB',
  is_extensions: false,
  value_extensions: '',
})

function resetForm() {
  form.value = {
    origin_id: '',
    path: '',
    validate_method: '',
    validate_url: '',
    validate_fallback_url: '',
    is_max_size: false,
    value_max_size: 0,
    value_unit_size: 'MB',
    is_extensions: false,
    value_extensions: '',
  }
}

onMounted(async () => {
  await originsStore.fetchOrigins()
  await rulesStore.fetchRules()
})

const filteredRules = computed(() => {
  if (!filterOriginId.value) return rulesStore.rules
  return rulesStore.rules.filter((r) => r.origin_id === filterOriginId.value)
})

function getDomainName(originId: string) {
  return originsStore.origins.find((o) => o.id === originId)?.domain || originId
}

async function handleCreate() {
  if (!form.value.origin_id || !form.value.path) return
  await rulesStore.createRule(form.value)
  resetForm()
  showCreateModal.value = false
}

function openEdit(rule: Rule) {
  editTarget.value = rule
  form.value = { ...rule }
  showEditModal.value = true
}

async function handleEdit() {
  if (!editTarget.value) return
  await rulesStore.updateRule(editTarget.value.id, form.value)
  showEditModal.value = false
  editTarget.value = null
  resetForm()
}

async function handleDelete(id: string) {
  if (confirm('Delete this rule?')) {
    await rulesStore.deleteRule(id)
  }
}
</script>

<template>
  <div class="rules-page">
    <!-- Header -->
    <div class="rules-page__header">
      <div>
        <h1 class="rules-page__title">Rules</h1>
        <p class="rules-page__desc">Validation pipeline for each origin path</p>
      </div>
      <div class="rules-page__header-actions">
        <select v-model="filterOriginId" class="rules-page__filter">
          <option value="">All Origins</option>
          <option v-for="o in originsStore.origins" :key="o.id" :value="o.id">
            {{ o.domain }}
          </option>
        </select>
        <MossButton @click="resetForm(); showCreateModal = true">+ Add Rule</MossButton>
      </div>
    </div>

    <!-- Rules List -->
    <div v-if="rulesStore.isLoading" class="rules-page__loading">
      <span class="rules-page__spinner"></span>
      Loading rules…
    </div>

    <div v-else-if="filteredRules.length === 0" class="rules-page__empty">
      <p>No rules found. Create a new rule to protect your origins.</p>
    </div>

    <div v-else class="rules-page__list">
      <SageCard
        v-for="rule in filteredRules"
        :key="rule.id"
        class="rule-card"
      >
        <div class="rule-card__header">
          <div class="rule-card__path-row">
            <PillTag>{{ getDomainName(rule.origin_id) }}</PillTag>
            <span class="rule-card__arrow">→</span>
            <span class="rule-card__path">{{ rule.path }}</span>
          </div>
        </div>

        <div class="rule-card__details">
          <div v-if="rule.validate_method" class="rule-card__detail">
            <span class="rule-card__detail-label">Method</span>
            <span class="rule-card__detail-value">{{ rule.validate_method }}</span>
          </div>
          <div v-if="rule.is_max_size" class="rule-card__detail">
            <span class="rule-card__detail-label">Max Size</span>
            <span class="rule-card__detail-value">{{ rule.value_max_size }} {{ rule.value_unit_size }}</span>
          </div>
          <div v-if="rule.is_extensions" class="rule-card__detail">
            <span class="rule-card__detail-label">Extensions</span>
            <span class="rule-card__detail-value">{{ rule.value_extensions }}</span>
          </div>
          <div v-if="rule.validate_url" class="rule-card__detail">
            <span class="rule-card__detail-label">Validate URL</span>
            <span class="rule-card__detail-value rule-card__detail-value--url">{{ rule.validate_url }}</span>
          </div>
        </div>

        <div class="rule-card__actions">
          <OutlineButton @click="openEdit(rule)">Edit</OutlineButton>
          <OutlineButton variant="danger" @click="handleDelete(rule.id)">Delete</OutlineButton>
        </div>
      </SageCard>
    </div>

    <!-- Create/Edit Modal (shared) -->
    <ModalDialog
      v-if="showCreateModal || showEditModal"
      :title="showEditModal ? 'Edit Rule' : 'Add Rule'"
      max-width="560px"
      @close="showCreateModal = false; showEditModal = false; editTarget = null; resetForm()"
    >
      <form @submit.prevent="showEditModal ? handleEdit() : handleCreate()" class="modal-form">
        <div class="field" v-if="!showEditModal">
          <label class="field__label" for="rule-origin">ORIGIN</label>
          <select id="rule-origin" v-model="form.origin_id" class="field__input" required>
            <option value="" disabled>Select origin…</option>
            <option v-for="o in originsStore.origins" :key="o.id" :value="o.id">{{ o.domain }}</option>
          </select>
        </div>
        <div class="field">
          <label class="field__label" for="rule-path">PATH</label>
          <input id="rule-path" v-model="form.path" class="field__input" placeholder="/file/uploads" required />
        </div>
        <div class="field">
          <label class="field__label" for="rule-method">VALIDATE METHOD</label>
          <input id="rule-method" v-model="form.validate_method" class="field__input" placeholder="GET, POST" />
        </div>
        <div class="field">
          <label class="field__label" for="rule-url">VALIDATE URL</label>
          <input id="rule-url" v-model="form.validate_url" class="field__input" placeholder="https://auth.example.com/validate" />
        </div>

        <div class="field__row">
          <label class="field__checkbox">
            <input type="checkbox" v-model="form.is_max_size" />
            <span>Limit file size</span>
          </label>
          <div v-if="form.is_max_size" class="field__inline">
            <input v-model.number="form.value_max_size" type="number" class="field__input field__input--small" min="1" />
            <select v-model="form.value_unit_size" class="field__input field__input--small">
              <option>KB</option>
              <option>MB</option>
              <option>GB</option>
            </select>
          </div>
        </div>

        <div class="field__row">
          <label class="field__checkbox">
            <input type="checkbox" v-model="form.is_extensions" />
            <span>Restrict extensions</span>
          </label>
          <input v-if="form.is_extensions" v-model="form.value_extensions" class="field__input" placeholder=".jpg,.png,.pdf" />
        </div>
      </form>
      <template #footer>
        <OutlineButton @click="showCreateModal = false; showEditModal = false; resetForm()">Cancel</OutlineButton>
        <MossButton @click="showEditModal ? handleEdit() : handleCreate()">
          {{ showEditModal ? 'Save Changes' : 'Create Rule' }}
        </MossButton>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.rules-page {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-24);
}
.rules-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-16);
  flex-wrap: wrap;
}
.rules-page__header-actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-12);
}
.rules-page__title {
  font-family: var(--font-denim);
  font-size: var(--text-heading);
  font-weight: 600;
  color: var(--color-forest-ink);
  margin: 0;
}
.rules-page__desc {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  margin: 2px 0 0;
}
.rules-page__filter {
  background: var(--color-bone-white);
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-xl);
  padding: 10px 14px;
  font-family: var(--font-denim);
  font-size: 13px;
  color: var(--color-forest-ink);
  outline: none;
}
.rules-page__loading,
.rules-page__empty {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  text-align: center;
  padding: var(--spacing-48) 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-12);
}
.rules-page__spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--color-lichen);
  border-top-color: var(--color-moss);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }

/* ───── List ───── */
.rules-page__list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-12);
}

/* ───── Card ───── */
.rule-card {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-12);
  transition: box-shadow 0.2s ease;
}
.rule-card:hover {
  box-shadow: var(--shadow-md-2);
}
.rule-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.rule-card__path-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-8);
}
.rule-card__arrow {
  color: var(--color-moss);
  font-family: var(--font-denim);
}
.rule-card__path {
  font-family: var(--font-denim);
  font-weight: 500;
  font-size: 16px;
  color: var(--color-forest-ink);
}
.rule-card__details {
  display: flex;
  flex-wrap: wrap;
  gap: var(--spacing-16);
}
.rule-card__detail {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.rule-card__detail-label {
  font-family: var(--font-cinetype);
  font-size: 10px;
  letter-spacing: 0.20em;
  color: var(--color-slate-smoke);
  text-transform: uppercase;
}
.rule-card__detail-value {
  font-family: var(--font-denim);
  font-size: 13px;
  color: var(--color-forest-ink);
}
.rule-card__detail-value--url {
  font-size: 12px;
  color: var(--color-indigo-accent);
  word-break: break-all;
}
.rule-card__actions {
  display: flex;
  gap: var(--spacing-8);
  margin-top: var(--spacing-4);
}

/* ───── Modal Form ───── */
.modal-form {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-16);
}
.field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.field__label {
  font-family: var(--font-cinetype);
  font-size: 11px;
  letter-spacing: 0.20em;
  color: var(--color-slate-smoke);
  text-transform: uppercase;
}
.field__input {
  background: var(--color-sage-paper);
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-xl);
  padding: 12px 16px;
  font-family: var(--font-denim);
  font-size: 14px;
  color: var(--color-forest-ink);
  outline: none;
  transition: border-color 0.15s ease;
}
.field__input:focus {
  border-color: var(--color-moss);
  box-shadow: 0 0 0 3px rgba(133, 192, 147, 0.15);
}
.field__input--small {
  max-width: 100px;
}
.field__row {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-8);
}
.field__inline {
  display: flex;
  gap: var(--spacing-8);
}
.field__checkbox {
  display: flex;
  align-items: center;
  gap: var(--spacing-8);
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-forest-ink);
  cursor: pointer;
}
</style>
