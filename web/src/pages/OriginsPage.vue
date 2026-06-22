<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useOriginsStore, type Origin } from '@/stores/origins'
import SageCard from '@/components/SageCard.vue'
import PillTag from '@/components/PillTag.vue'
import MossButton from '@/components/MossButton.vue'
import OutlineButton from '@/components/OutlineButton.vue'
import ModalDialog from '@/components/ModalDialog.vue'

const store = useOriginsStore()

const showCreateModal = ref(false)
const showEditModal = ref(false)
const editTarget = ref<Origin | null>(null)

const newDomain = ref('')
const editDomain = ref('')
const editBlocked = ref(false)

const showDeleteConfirm = ref(false)
const deleteTarget = ref<Origin | null>(null)

onMounted(() => {
  store.fetchOrigins()
})

function cleanDomain(val: string): string {
  let domain = val.trim().toLowerCase()
  if (domain.startsWith('https://')) {
    domain = domain.substring(8)
  } else if (domain.startsWith('http://')) {
    domain = domain.substring(7)
  }
  return domain.split('/')[0] || ''
}

async function handleCreate() {
  const cleaned = cleanDomain(newDomain.value)
  if (!cleaned) return
  await store.createOrigin(cleaned)
  newDomain.value = ''
  showCreateModal.value = false
}

function openEdit(origin: Origin) {
  editTarget.value = origin
  editDomain.value = origin.domain
  editBlocked.value = origin.is_blocked
  showEditModal.value = true
}

async function handleEdit() {
  if (!editTarget.value) return
  const cleaned = cleanDomain(editDomain.value)
  if (!cleaned) return
  await store.updateOrigin(editTarget.value.id, cleaned, editBlocked.value)
  showEditModal.value = false
  editTarget.value = null
}

function openDeleteConfirm(origin: Origin) {
  deleteTarget.value = origin
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deleteTarget.value) return
  await store.deleteOrigin(deleteTarget.value.id)
  showDeleteConfirm.value = false
  deleteTarget.value = null
}

async function handleToggleBlock(origin: Origin) {
  await store.updateOrigin(origin.id, origin.domain, !origin.is_blocked)
}
</script>

<template>
  <div class="origins-page">
    <!-- Header -->
    <div class="origins-page__header">
      <div>
        <h1 class="origins-page__title">Origins</h1>
        <p class="origins-page__desc">Manage connected domains and their access permissions</p>
      </div>
      <MossButton @click="showCreateModal = true">+ Add Origin</MossButton>
    </div>

    <!-- Grid -->
    <div v-if="store.isLoading" class="origins-page__loading">
      <span class="origins-page__spinner"></span>
      Loading origins…
    </div>

    <div v-else-if="store.origins.length === 0" class="origins-page__empty">
      <p>No origins configured. Add your first domain to get started.</p>
    </div>

    <div v-else class="origins-page__grid">
      <SageCard
        v-for="origin in store.origins"
        :key="origin.id"
        class="origin-card"
        :class="{ 'origin-card--blocked': origin.is_blocked }"
      >
        <div class="origin-card__header">
          <span class="origin-card__icon">{{ origin.is_blocked ? '⊘' : '🌐' }}</span>
          <PillTag :variant="origin.is_blocked ? 'danger' : 'success'">
            {{ origin.is_blocked ? 'BLOCKED' : 'ACTIVE' }}
          </PillTag>
        </div>
        <h3 class="origin-card__domain">{{ origin.domain }}</h3>
        <p class="origin-card__id">{{ origin.id }}</p>
        <div class="origin-card__actions">
          <OutlineButton @click="openEdit(origin)">Edit</OutlineButton>
          <OutlineButton @click="handleToggleBlock(origin)">
            {{ origin.is_blocked ? 'Unblock' : 'Block' }}
          </OutlineButton>
          <OutlineButton variant="danger" @click="openDeleteConfirm(origin)">Delete</OutlineButton>
        </div>
      </SageCard>
    </div>

    <!-- Create Modal -->
    <ModalDialog v-if="showCreateModal" title="Add Origin" @close="showCreateModal = false">
      <form @submit.prevent="handleCreate" class="modal-form">
        <div class="field">
          <label class="field__label" for="new-origin-domain">DOMAIN</label>
          <input
            id="new-origin-domain"
            v-model="newDomain"
            type="text"
            class="field__input"
            placeholder="example.com"
            required
          />
        </div>
      </form>
      <template #footer>
        <OutlineButton @click="showCreateModal = false">Cancel</OutlineButton>
        <MossButton @click="handleCreate" :disabled="!newDomain.trim()">Create Origin</MossButton>
      </template>
    </ModalDialog>

    <!-- Edit Modal -->
    <ModalDialog v-if="showEditModal" title="Edit Origin" @close="showEditModal = false">
      <form @submit.prevent="handleEdit" class="modal-form">
        <div class="field">
          <label class="field__label" for="edit-origin-domain">DOMAIN</label>
          <input
            id="edit-origin-domain"
            v-model="editDomain"
            type="text"
            class="field__input"
            required
          />
        </div>
      </form>
      <template #footer>
        <OutlineButton @click="showEditModal = false">Cancel</OutlineButton>
        <MossButton @click="handleEdit" :disabled="!editDomain.trim()">Save Changes</MossButton>
      </template>
    </ModalDialog>

    <!-- Delete Confirmation Modal -->
    <ModalDialog v-if="showDeleteConfirm" title="Delete Origin" max-width="420px" @close="showDeleteConfirm = false">
      <div style="font-family: var(--font-denim); font-size: 14px; color: var(--color-forest-ink); line-height: 1.5;">
        Are you sure you want to delete origin <strong>{{ deleteTarget?.domain }}</strong>? This action cannot be undone and will prevent clients from serving files under this domain.
      </div>
      <template #footer>
        <OutlineButton @click="showDeleteConfirm = false">Cancel</OutlineButton>
        <OutlineButton variant="danger" @click="handleDelete">Delete</OutlineButton>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.origins-page {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-24);
}
.origins-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-16);
}
.origins-page__title {
  font-family: var(--font-denim);
  font-size: var(--text-heading);
  font-weight: 600;
  color: var(--color-forest-ink);
  margin: 0;
}
.origins-page__desc {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  margin: 2px 0 0;
}
.origins-page__loading {
  display: flex;
  align-items: center;
  gap: var(--spacing-12);
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  padding: var(--spacing-48) 0;
}
.origins-page__spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--color-lichen);
  border-top-color: var(--color-moss);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin {
  to { transform: rotate(360deg); }
}
.origins-page__empty {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  text-align: center;
  padding: var(--spacing-48) 0;
}

/* ───── Grid ───── */
.origins-page__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--spacing-16);
}

/* ───── Card ───── */
.origin-card {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-12);
  transition: box-shadow 0.2s ease;
}
.origin-card:hover {
  box-shadow: var(--shadow-md-2);
}
.origin-card--blocked {
  opacity: 0.6;
}
.origin-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.origin-card__icon {
  font-size: 22px;
}
.origin-card__domain {
  font-family: var(--font-denim);
  font-size: 18px;
  font-weight: 500;
  color: var(--color-forest-ink);
  margin: 0;
  word-break: break-all;
}
.origin-card__id {
  font-family: var(--font-cinetype);
  font-size: 11px;
  color: var(--color-lichen);
  margin: 0;
  letter-spacing: 0.04em;
}
.origin-card__actions {
  display: flex;
  gap: var(--spacing-8);
  flex-wrap: wrap;
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
