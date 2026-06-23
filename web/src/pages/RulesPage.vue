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

const showDeleteConfirm = ref(false)
const deleteTarget = ref<Rule | null>(null)

// New rule form with headers keys list
const form = ref({
  origin_id: '',
  path: '',
  validate_method: '',
  validate_headers: '',
  validate_url: '',
  validate_fallback_url: '',
  is_max_size: false,
  value_max_size: 0,
  value_unit_size: 'MB',
  is_extensions: false,
  value_extensions: '',
  is_compress: false,
  compress_level: 3,
  is_encrypt: false,
  encryption_key: '',
  is_cache: false,
  value_cache: 1,
  unit_cache: 'year',
  headers: [{ id: Math.random().toString(36).substring(2), value: '' }],
})

function resetForm() {
  form.value = {
    origin_id: '',
    path: '',
    validate_method: '',
    validate_headers: '',
    validate_url: '',
    validate_fallback_url: '',
    is_max_size: false,
    value_max_size: 0,
    value_unit_size: 'MB',
    is_extensions: false,
    value_extensions: '',
    is_compress: false,
    compress_level: 3,
    is_encrypt: false,
    encryption_key: '',
    is_cache: false,
    value_cache: 1,
    unit_cache: 'year',
    headers: [{ id: Math.random().toString(36).substring(2), value: '' }],
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

function addHeaderKey() {
  form.value.headers.push({ id: Math.random().toString(36).substring(2), value: '' })
}

function removeHeaderKey(index: number) {
  form.value.headers.splice(index, 1)
  if (form.value.headers.length === 0) {
    form.value.headers.push({ id: Math.random().toString(36).substring(2), value: '' })
  }
}

function cleanFormPayload() {
  let pathVal = form.value.path.trim()
  if (pathVal.startsWith('/file/')) {
    pathVal = pathVal.substring(6)
  } else if (pathVal.startsWith('file/')) {
    pathVal = pathVal.substring(5)
  } else if (pathVal.startsWith('/file')) {
    pathVal = pathVal.substring(5)
  } else if (pathVal.startsWith('file')) {
    pathVal = pathVal.substring(4)
  }
  pathVal = pathVal.replace(/^\/+|\/+$/g, '')

  let headersStr = ''
  if (form.value.validate_method === 'headers') {
    headersStr = form.value.headers
      .map((h) => h.value.trim())
      .filter(Boolean)
      .join(',')
  }

  return {
    origin_id: form.value.origin_id,
    path: pathVal,
    validate_method: form.value.validate_method,
    validate_headers: headersStr,
    validate_url:
      form.value.validate_method === 'JWT' ||
      form.value.validate_method === 'headers' ||
      form.value.validate_method === 'cookies'
        ? form.value.validate_url.trim()
        : '',
    validate_fallback_url:
      form.value.validate_method === 'JWT' ||
      form.value.validate_method === 'headers' ||
      form.value.validate_method === 'cookies'
        ? form.value.validate_fallback_url.trim()
        : '',
    is_max_size: form.value.is_max_size,
    value_max_size: form.value.value_max_size,
    value_unit_size: form.value.value_unit_size,
    is_extensions: form.value.is_extensions,
    value_extensions: form.value.value_extensions.trim(),
    is_compress: form.value.is_compress,
    compress_level: form.value.compress_level,
    is_encrypt: form.value.is_encrypt,
    encryption_key: form.value.encryption_key.trim(),
    is_cache: form.value.is_cache,
    value_cache: form.value.is_cache ? Math.max(1, form.value.value_cache || 1) : 1,
    unit_cache: form.value.unit_cache || 'year',
  }
}

async function handleCreate() {
  if (!form.value.origin_id || !form.value.path) return
  const payload = cleanFormPayload()
  await rulesStore.createRule(payload)
  resetForm()
  showCreateModal.value = false
}

function openEdit(rule: Rule) {
  editTarget.value = rule
  const hdrs = rule.validate_headers
    ? rule.validate_headers
        .split(',')
        .map((h) => ({ id: Math.random().toString(36).substring(2), value: h }))
    : [{ id: Math.random().toString(36).substring(2), value: '' }]
  form.value = {
    origin_id: rule.origin_id,
    path: rule.path,
    validate_method: rule.validate_method || '',
    validate_headers: rule.validate_headers || '',
    validate_url: rule.validate_url || '',
    validate_fallback_url: rule.validate_fallback_url || '',
    is_max_size: rule.is_max_size,
    value_max_size: rule.value_max_size,
    value_unit_size: rule.value_unit_size,
    is_extensions: rule.is_extensions,
    value_extensions: rule.value_extensions || '',
    is_compress: rule.is_compress,
    compress_level: rule.compress_level || 3,
    is_encrypt: rule.is_encrypt,
    encryption_key: rule.encryption_key || '',
    is_cache: rule.is_cache || false,
    value_cache: rule.value_cache || 1,
    unit_cache: rule.unit_cache || 'year',
    headers: hdrs,
  }
  showEditModal.value = true
}

async function handleEdit() {
  if (!editTarget.value) return
  const payload = cleanFormPayload()
  await rulesStore.updateRule(editTarget.value.id, payload)
  showEditModal.value = false
  editTarget.value = null
  resetForm()
}

function openDeleteConfirm(rule: Rule) {
  deleteTarget.value = rule
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deleteTarget.value) return
  await rulesStore.deleteRule(deleteTarget.value.id)
  showDeleteConfirm.value = false
  deleteTarget.value = null
}

function openCreateModal() {
  resetForm()
  showCreateModal.value = true
}

function closeFormModal() {
  showCreateModal.value = false
  showEditModal.value = false
  editTarget.value = null
  resetForm()
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
        <MossButton @click="openCreateModal">+ Add Rule</MossButton>
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
      <SageCard v-for="rule in filteredRules" :key="rule.id" class="rule-card">
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
            <span class="rule-card__detail-value"
              >{{ rule.value_max_size }} {{ rule.value_unit_size }}</span
            >
          </div>
          <div v-if="rule.is_extensions" class="rule-card__detail">
            <span class="rule-card__detail-label">Extensions</span>
            <span class="rule-card__detail-value">{{ rule.value_extensions }}</span>
          </div>
          <div v-if="rule.validate_url" class="rule-card__detail">
            <span class="rule-card__detail-label">Validate URL</span>
            <span class="rule-card__detail-value rule-card__detail-value--url">{{
              rule.validate_url
            }}</span>
          </div>
          <div v-if="rule.is_compress" class="rule-card__detail">
            <span class="rule-card__detail-label">Compress</span>
            <span class="rule-card__detail-value">Level {{ rule.compress_level }}</span>
          </div>
          <div v-if="rule.is_encrypt" class="rule-card__detail">
            <span class="rule-card__detail-label">Encrypt</span>
            <span class="rule-card__detail-value">{{
              rule.encryption_key ? 'Custom Key' : 'Default Key'
            }}</span>
          </div>
          <div v-if="rule.is_cache" class="rule-card__detail">
            <span class="rule-card__detail-label">Cache</span>
            <span class="rule-card__detail-value"
              >{{ rule.value_cache }} {{ rule.unit_cache
              }}{{ rule.value_cache > 1 ? 's' : '' }}</span
            >
          </div>
        </div>

        <div class="rule-card__actions">
          <OutlineButton @click="openEdit(rule)">Edit</OutlineButton>
          <OutlineButton variant="danger" @click="openDeleteConfirm(rule)">Delete</OutlineButton>
        </div>
      </SageCard>
    </div>

    <ModalDialog
      v-if="showCreateModal || showEditModal"
      :title="showEditModal ? 'Edit Rule' : 'Add Rule'"
      max-width="560px"
      @close="closeFormModal"
    >
      <form @submit.prevent="showEditModal ? handleEdit() : handleCreate()" class="modal-form">
        <div class="field" v-if="!showEditModal">
          <label class="field__label" for="rule-origin">ORIGIN</label>
          <select id="rule-origin" v-model="form.origin_id" class="field__input" required>
            <option value="" disabled>Select origin…</option>
            <option v-for="o in originsStore.origins" :key="o.id" :value="o.id">
              {{ o.domain }}
            </option>
          </select>
        </div>

        <div class="field">
          <label class="field__label" for="rule-path">PATH</label>
          <div class="path-input-wrapper">
            <span class="path-prefix">/file/</span>
            <input
              id="rule-path"
              v-model="form.path"
              class="field__input path-field"
              placeholder="uploads"
              required
            />
          </div>
        </div>

        <div class="field">
          <label class="field__label" for="rule-method">VALIDATE METHOD</label>
          <select id="rule-method" v-model="form.validate_method" class="field__input">
            <option value="">None (Public)</option>
            <option value="JWT">JWT</option>
            <option value="headers">Headers</option>
            <option value="cookies">Cookies</option>
          </select>
        </div>

        <!-- Headers list card -->
        <div v-if="form.validate_method === 'headers'" class="headers-card">
          <div class="headers-card__header">
            <span class="headers-card__title">REQUIRED HEADERS</span>
            <button type="button" class="headers-card__add" @click="addHeaderKey">
              + Add Header
            </button>
          </div>
          <div class="headers-card__list">
            <div v-for="(hdr, idx) in form.headers" :key="hdr.id" class="header-field-row">
              <input
                v-model="hdr.value"
                type="text"
                class="field__input header-key-input"
                placeholder="X-Api-Key"
                required
              />
              <button type="button" class="header-remove-btn" @click="removeHeaderKey(idx)">
                ✕
              </button>
            </div>
          </div>
        </div>

        <!-- Validate URL fields -->
        <div
          v-if="
            form.validate_method === 'JWT' ||
            form.validate_method === 'headers' ||
            form.validate_method === 'cookies'
          "
          class="field"
        >
          <label class="field__label" for="rule-url">VALIDATE URL (EXTERNAL TARGET)</label>
          <input
            id="rule-url"
            v-model="form.validate_url"
            class="field__input"
            placeholder="https://api.example.com/auth/validate"
            required
          />
        </div>

        <div
          v-if="
            form.validate_method === 'JWT' ||
            form.validate_method === 'headers' ||
            form.validate_method === 'cookies'
          "
          class="field"
        >
          <label class="field__label" for="rule-fallback">FALLBACK REDIRECT URL (OPTIONAL)</label>
          <input
            id="rule-fallback"
            v-model="form.validate_fallback_url"
            class="field__input"
            placeholder="https://example.com/login"
          />
        </div>

        <div class="field__row">
          <label class="field__checkbox">
            <input type="checkbox" v-model="form.is_max_size" />
            <span>Limit file size</span>
          </label>
          <div v-if="form.is_max_size" class="field__inline">
            <input
              v-model.number="form.value_max_size"
              type="number"
              class="field__input field__input--small"
              min="1"
            />
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
          <input
            v-if="form.is_extensions"
            v-model="form.value_extensions"
            class="field__input"
            placeholder="jpg,png,pdf"
          />
        </div>

        <div class="field__row" style="flex-direction: column; align-items: flex-start; gap: 8px">
          <label class="field__checkbox">
            <input type="checkbox" v-model="form.is_compress" />
            <span>Compress File (zstd)</span>
          </label>
          <div
            v-if="form.is_compress"
            style="width: 100%; display: flex; align-items: center; gap: 8px; padding-left: 24px"
          >
            <span class="field__label" style="font-size: 11px; margin-bottom: 0"
              >compress level</span
            >
            <select
              v-model.number="form.compress_level"
              class="field__input field__input--small"
              style="width: 100px"
            >
              <option v-for="level in 22" :key="level" :value="level">{{ level }}</option>
            </select>
          </div>
        </div>

        <div class="field__row" style="flex-direction: column; align-items: flex-start; gap: 8px">
          <label class="field__checkbox">
            <input type="checkbox" v-model="form.is_encrypt" />
            <span>Encrypt File</span>
          </label>
          <div v-if="form.is_encrypt" style="width: 100%; padding-left: 24px">
            <input
              v-model="form.encryption_key"
              class="field__input"
              placeholder="encryption_key (optional)"
            />
          </div>
        </div>

        <div class="field__row" style="flex-direction: column; align-items: flex-start; gap: 8px">
          <label class="field__checkbox">
            <input type="checkbox" v-model="form.is_cache" />
            <span>Cache (Max-Age)</span>
          </label>
          <div
            v-if="form.is_cache"
            style="width: 100%; display: flex; align-items: center; gap: 8px; padding-left: 24px"
          >
            <input
              type="number"
              v-model.number="form.value_cache"
              class="field__input field__input--small"
              style="width: 100px"
              min="1"
              required
            />
            <select
              v-model="form.unit_cache"
              class="field__input field__input--small"
              style="width: 120px"
            >
              <option value="hour">hour</option>
              <option value="day">day</option>
              <option value="month">month</option>
              <option value="year">year</option>
            </select>
          </div>
        </div>
      </form>
      <template #footer>
        <OutlineButton @click="closeFormModal">Cancel</OutlineButton>
        <MossButton @click="showEditModal ? handleEdit() : handleCreate()">
          {{ showEditModal ? 'Save Changes' : 'Create Rule' }}
        </MossButton>
      </template>
    </ModalDialog>

    <!-- Delete Confirmation Modal -->
    <ModalDialog
      v-if="showDeleteConfirm"
      title="Delete Rule"
      max-width="420px"
      @close="showDeleteConfirm = false"
    >
      <div
        style="
          font-family: var(--font-denim);
          font-size: 14px;
          color: var(--color-forest-ink);
          line-height: 1.5;
        "
      >
        Are you sure you want to delete rule for path <strong>/file/{{ deleteTarget?.path }}</strong
        >? This action cannot be undone.
      </div>
      <template #footer>
        <OutlineButton @click="showDeleteConfirm = false">Cancel</OutlineButton>
        <OutlineButton variant="danger" @click="handleDelete">Delete</OutlineButton>
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
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

/* ───── List ───── */
.rules-page__list {
  display: grid;
  grid-template-columns: repeat(1, minmax(0, 1fr));
  gap: var(--spacing-16);
}
@media (min-width: 640px) {
  .rules-page__list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (min-width: 1024px) {
  .rules-page__list {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (min-width: 1280px) {
  .rules-page__list {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}

/* ───── Card ───── */
.rule-card {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-16);
  height: 100%;
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
  flex-direction: column;
  gap: var(--spacing-12);
  flex: 1;
}
.rule-card__detail {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.rule-card__detail-label {
  font-family: var(--font-cinetype);
  font-size: 10px;
  letter-spacing: 0.2em;
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
  margin-top: auto;
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
  letter-spacing: 0.2em;
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

/* ───── Path Prefix Input ───── */
.path-input-wrapper {
  display: flex;
  align-items: center;
  background: var(--color-sage-paper);
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-xl);
  overflow: hidden;
}
.path-prefix {
  padding: 12px 0 12px 16px;
  font-family: var(--font-denim);
  font-size: 14px;
  color: var(--color-slate-smoke);
  user-select: none;
}
.path-field {
  flex: 1;
  background: transparent !important;
  border: none !important;
  box-shadow: none !important;
  padding-left: 4px !important;
}
.path-input-wrapper:focus-within {
  border-color: var(--color-moss);
  box-shadow: 0 0 0 3px rgba(133, 192, 147, 0.15);
}

/* ───── Headers configuration card ───── */
.headers-card {
  background: var(--color-bone-white);
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-xl);
  padding: var(--spacing-16);
  display: flex;
  flex-direction: column;
  gap: var(--spacing-12);
}
.headers-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.headers-card__title {
  font-family: var(--font-cinetype);
  font-size: 11px;
  letter-spacing: 0.2em;
  color: var(--color-slate-smoke);
}
.headers-card__add {
  background: none;
  border: none;
  color: var(--color-deep-fern);
  font-family: var(--font-denim);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
}
.headers-card__add:hover {
  text-decoration: underline;
}
.headers-card__list {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-8);
}
.header-field-row {
  display: flex;
  align-items: center;
  gap: var(--spacing-8);
}
.header-key-input {
  flex: 1;
}
.header-remove-btn {
  background: none;
  border: none;
  font-size: 16px;
  color: var(--color-slate-smoke);
  cursor: pointer;
  padding: var(--spacing-4);
}
.header-remove-btn:hover {
  color: #8b2020;
}
</style>
