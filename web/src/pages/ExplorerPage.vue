<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useExplorerStore, type FileItem } from '@/stores/explorer'
import { useOriginsStore } from '@/stores/origins'
import SageCard from '@/components/SageCard.vue'
import MossButton from '@/components/MossButton.vue'
import OutlineButton from '@/components/OutlineButton.vue'
import ModalDialog from '@/components/ModalDialog.vue'

const store = useExplorerStore()
const originsStore = useOriginsStore()

const selectedOriginId = ref('')
const showNewFolderModal = ref(false)
const newFolderName = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)

// Delete Confirmation States
const showDeleteConfirm = ref(false)
const deleteTargetItem = ref<FileItem | null>(null)

onMounted(async () => {
  await originsStore.fetchOrigins()
})

const selectedOrigin = computed(() => {
  return originsStore.origins.find((o) => o.id === selectedOriginId.value)
})

// Snake-cased name of the selected origin, which represents the root folder on disk
const baseOriginPath = computed(() => {
  const origin = selectedOrigin.value
  if (!origin) return ''
  return origin.domain.replaceAll('.', '_').replaceAll('-', '_')
})

// When origin changes, reload files scoped to this origin's base path
async function handleOriginChange() {
  if (selectedOriginId.value) {
    await store.listItems(baseOriginPath.value)
  } else {
    store.items = []
    store.currentPath = ''
  }
}

// Breadcrumb segments relative to selected origin
const breadcrumbs = computed(() => {
  if (!baseOriginPath.value) return []
  
  const parts = store.currentPath.split('/').filter(Boolean)
  const relativeParts = parts.slice(1) // skip the base origin folder
  
  return relativeParts.map((part, idx) => {
    const fullPath = [baseOriginPath.value, ...relativeParts.slice(0, idx + 1)].join('/')
    return {
      label: part,
      path: fullPath
    }
  })
})

function navigateTo(path: string) {
  store.listItems(path)
}

function navigateUp() {
  const parts = store.currentPath.split('/').filter(Boolean)
  if (parts.length <= 1) return // Already at origin root
  parts.pop()
  store.listItems(parts.join('/'))
}

function openItem(item: FileItem) {
  if (item.is_dir) {
    store.listItems(item.path)
  }
}

async function handleCreateFolder() {
  if (!newFolderName.value.trim()) return
  await store.createFolder(store.currentPath, newFolderName.value.trim())
  newFolderName.value = ''
  showNewFolderModal.value = false
}

function triggerUpload() {
  fileInputRef.value?.click()
}

async function handleUpload(e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return
  await store.uploadFile(store.currentPath, file)
  target.value = ''
}

function handleDownload(item: FileItem) {
  store.downloadFile(item.path)
}

function openDeleteConfirm(item: FileItem) {
  deleteTargetItem.value = item
  showDeleteConfirm.value = true
}

async function handleDelete() {
  if (!deleteTargetItem.value) return
  await store.deleteItem(deleteTargetItem.value.path)
  showDeleteConfirm.value = false
  deleteTargetItem.value = null
}

function formatSize(bytes: number): string {
  if (bytes === 0) return '—'
  const units = ['B', 'KB', 'MB', 'GB']
  let i = 0
  let size = bytes
  while (size >= 1024 && i < units.length - 1) {
    size /= 1024
    i++
  }
  return `${size.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

function formatDate(dateStr: string): string {
  if (!dateStr) return '—'
  return new Date(dateStr).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

const fileIcon = (item: FileItem) => {
  if (item.is_dir) return '📁'
  const ext = item.name.split('.').pop()?.toLowerCase() || ''
  const icons: Record<string, string> = {
    jpg: '🖼️', jpeg: '🖼️', png: '🖼️', gif: '🖼️', webp: '🖼️', svg: '🖼️',
    pdf: '📑', doc: '📝', docx: '📝', txt: '📝',
    mp4: '🎬', avi: '🎬', mkv: '🎬',
    mp3: '🎵', wav: '🎵',
    zip: '📦', rar: '📦', tar: '📦', gz: '📦',
  }
  return icons[ext] || '📄'
}
</script>

<template>
  <div class="explorer-page">
    <!-- 1. Origin Select Screen (if none selected) -->
    <div v-if="!selectedOriginId" class="origin-select-screen animate-fade-in">
      <div class="origin-select-box">
        <span class="origin-select-icon">🌐</span>
        <h2 class="origin-select-title">Access Storage Explorer</h2>
        <p class="origin-select-desc">Select an origin domain to view and manage its storage bucket</p>
        <select v-model="selectedOriginId" @change="handleOriginChange" class="origin-select-dropdown">
          <option value="" disabled>Select an origin domain…</option>
          <option v-for="o in originsStore.origins" :key="o.id" :value="o.id">
            {{ o.domain }}
          </option>
        </select>
      </div>
    </div>

    <!-- 2. Main File Explorer (when origin selected) -->
    <template v-else>
      <!-- Header -->
      <div class="explorer-page__header">
        <div>
          <h1 class="explorer-page__title">Storage Explorer</h1>
          <div class="explorer-page__subtitle">
            <span class="active-origin-badge">🌐 {{ selectedOrigin?.domain }}</span>
            <button @click="selectedOriginId = ''; handleOriginChange()" class="change-origin-btn">
              [ Switch Origin ]
            </button>
          </div>
        </div>
        <div class="explorer-page__actions">
          <OutlineButton @click="showNewFolderModal = true">📁 New Folder</OutlineButton>
          <MossButton @click="triggerUpload">↑ Upload</MossButton>
          <input ref="fileInputRef" type="file" class="explorer-page__file-input" @change="handleUpload" />
        </div>
      </div>

      <!-- Breadcrumb -->
      <nav class="breadcrumb">
        <button class="breadcrumb__item" @click="navigateTo(baseOriginPath)">
          🏠 root
        </button>
        <template v-for="crumb in breadcrumbs" :key="crumb.path">
          <span class="breadcrumb__sep">/</span>
          <button class="breadcrumb__item" @click="navigateTo(crumb.path)">
            {{ crumb.label }}
          </button>
        </template>
      </nav>

      <!-- Upload Progress -->
      <div v-if="store.uploadProgress > 0 && store.uploadProgress < 100" class="upload-bar">
        <div class="upload-bar__fill" :style="{ width: store.uploadProgress + '%' }"></div>
        <span class="upload-bar__text">Uploading {{ store.uploadProgress }}%</span>
      </div>

      <!-- Loading -->
      <div v-if="store.isLoading" class="explorer-page__loading">
        <span class="explorer-page__spinner"></span>
        Loading…
      </div>

      <!-- Empty -->
      <div v-else-if="store.items.length === 0" class="explorer-page__empty">
        <p>{{ store.currentPath !== baseOriginPath ? 'This folder is empty.' : 'No files in storage yet.' }}</p>
      </div>

      <!-- File Grid -->
      <div v-else class="explorer-page__grid">
        <!-- Back button -->
        <SageCard
          v-if="store.currentPath && store.currentPath !== baseOriginPath"
          class="file-card file-card--back"
          @click="navigateUp()"
        >
          <span class="file-card__icon">⬆️</span>
          <span class="file-card__name">..</span>
        </SageCard>

        <!-- Items -->
        <SageCard
          v-for="item in store.items"
          :key="item.path"
          class="file-card"
          :class="{ 'file-card--dir': item.is_dir }"
        >
          <div class="file-card__main" @click="openItem(item)">
            <span class="file-card__icon">{{ fileIcon(item) }}</span>
            <div class="file-card__info">
              <span class="file-card__name">{{ item.name }}</span>
              <span class="file-card__meta">
                {{ item.is_dir ? 'Folder' : formatSize(item.size) }}
                <template v-if="item.modified_at"> · {{ formatDate(item.modified_at) }}</template>
              </span>
            </div>
          </div>
          <div class="file-card__actions">
            <button v-if="!item.is_dir" class="file-card__action" title="Download" @click.stop="handleDownload(item)">↓</button>
            <button class="file-card__action file-card__action--danger" title="Delete" @click.stop="openDeleteConfirm(item)">✕</button>
          </div>
        </SageCard>
      </div>
    </template>

    <!-- New Folder Modal -->
    <ModalDialog v-if="showNewFolderModal" title="New Folder" @close="showNewFolderModal = false">
      <form @submit.prevent="handleCreateFolder" class="modal-form">
        <div class="field">
          <label class="field__label" for="folder-name">FOLDER NAME</label>
          <input
            id="folder-name"
            v-model="newFolderName"
            type="text"
            class="field__input"
            placeholder="my-folder"
            required
          />
        </div>
      </form>
      <template #footer>
        <OutlineButton @click="showNewFolderModal = false">Cancel</OutlineButton>
        <MossButton @click="handleCreateFolder" :disabled="!newFolderName.trim()">Create Folder</MossButton>
      </template>
    </ModalDialog>

    <!-- Delete Confirmation Modal -->
    <ModalDialog v-if="showDeleteConfirm" :title="'Delete ' + (deleteTargetItem?.is_dir ? 'Folder' : 'File')" max-width="420px" @close="showDeleteConfirm = false">
      <div style="font-family: var(--font-denim); font-size: 14px; color: var(--color-forest-ink); line-height: 1.5;">
        Are you sure you want to delete {{ deleteTargetItem?.is_dir ? 'folder' : 'file' }} <strong>{{ deleteTargetItem?.name }}</strong>? This action cannot be undone.
      </div>
      <template #footer>
        <OutlineButton @click="showDeleteConfirm = false">Cancel</OutlineButton>
        <OutlineButton variant="danger" @click="handleDelete">Delete</OutlineButton>
      </template>
    </ModalDialog>
  </div>
</template>

<style scoped>
.explorer-page {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-20);
}
.explorer-page__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--spacing-16);
  flex-wrap: wrap;
}
.explorer-page__actions {
  display: flex;
  align-items: center;
  gap: var(--spacing-8);
}
.explorer-page__file-input {
  display: none;
}
.explorer-page__title {
  font-family: var(--font-denim);
  font-size: var(--text-heading);
  font-weight: 600;
  color: var(--color-forest-ink);
  margin: 0;
}
.explorer-page__desc {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  margin: 2px 0 0;
}
.explorer-page__loading {
  display: flex;
  align-items: center;
  gap: var(--spacing-12);
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  padding: var(--spacing-48) 0;
}
.explorer-page__spinner {
  width: 18px;
  height: 18px;
  border: 2px solid var(--color-lichen);
  border-top-color: var(--color-moss);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
.explorer-page__empty {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  text-align: center;
  padding: var(--spacing-48) 0;
}

/* ───── Breadcrumb ───── */
.breadcrumb {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
  padding: 10px 16px;
  background: var(--color-bone-white);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-subtle);
}
.breadcrumb__item {
  background: none;
  border: none;
  font-family: var(--font-denim);
  font-size: 13px;
  color: var(--color-forest-ink);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: var(--radius-md);
  transition: background 0.12s ease;
}
.breadcrumb__item:hover {
  background: rgba(133, 192, 147, 0.1);
}
.breadcrumb__sep {
  color: var(--color-lichen);
  font-size: 13px;
}

/* ───── Upload Bar ───── */
.upload-bar {
  position: relative;
  height: 28px;
  background: var(--color-bone-white);
  border-radius: var(--radius-xl);
  overflow: hidden;
  box-shadow: var(--shadow-subtle);
}
.upload-bar__fill {
  position: absolute;
  inset: 0;
  background: linear-gradient(90deg, var(--color-moss), var(--color-deep-fern));
  transition: width 0.3s ease;
}
.upload-bar__text {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  font-family: var(--font-muoto);
  font-size: 12px;
  color: var(--color-forest-ink);
}

/* ───── File Grid ───── */
.explorer-page__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--spacing-12);
}
.file-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--spacing-12);
  cursor: default;
  transition: box-shadow 0.2s ease;
  padding: var(--spacing-16);
}
.file-card:hover {
  box-shadow: var(--shadow-md-2);
}
.file-card--dir .file-card__main {
  cursor: pointer;
}
.file-card--back {
  cursor: pointer;
  opacity: 0.7;
}
.file-card--back:hover {
  opacity: 1;
}
.file-card__main {
  display: flex;
  align-items: center;
  gap: var(--spacing-12);
  min-width: 0;
  flex: 1;
}
.file-card__icon {
  font-size: 24px;
  flex-shrink: 0;
}
.file-card__info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.file-card__name {
  font-family: var(--font-denim);
  font-weight: 500;
  font-size: 14px;
  color: var(--color-forest-ink);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.file-card__meta {
  font-family: var(--font-muoto);
  font-size: 12px;
  color: var(--color-slate-smoke);
}
.file-card__actions {
  display: flex;
  gap: 4px;
  flex-shrink: 0;
}
.file-card__action {
  background: none;
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-md);
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  cursor: pointer;
  color: var(--color-forest-ink);
  transition: background 0.12s ease;
}
.file-card__action:hover {
  background: rgba(133, 192, 147, 0.1);
}
.file-card__action--danger:hover {
  background: rgba(196, 77, 77, 0.08);
  border-color: #c44d4d;
  color: #8b2020;
}

/* ───── Modal ───── */
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
}
.field__input:focus {
  border-color: var(--color-moss);
  box-shadow: 0 0 0 3px rgba(133, 192, 147, 0.15);
}

/* ───── Origin Select Screen ───── */
.origin-select-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--spacing-48) 0;
  min-height: 50vh;
}
.origin-select-box {
  background: var(--color-bone-white);
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-xl);
  padding: var(--spacing-40) var(--spacing-32);
  max-width: 480px;
  width: 100%;
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--spacing-16);
}
.origin-select-icon {
  font-size: 48px;
  line-height: 1;
}
.origin-select-title {
  font-family: var(--font-denim);
  font-size: 22px;
  font-weight: 600;
  color: var(--color-forest-ink);
  margin: 0;
}
.origin-select-desc {
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  margin: 0 0 var(--spacing-8);
  line-height: 1.5;
}
.origin-select-dropdown {
  width: 100%;
  background: var(--color-sage-paper);
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-xl);
  padding: 12px 16px;
  font-family: var(--font-denim);
  font-size: 14px;
  color: var(--color-forest-ink);
  outline: none;
  cursor: pointer;
  transition: border-color 0.15s ease;
}
.origin-select-dropdown:focus {
  border-color: var(--color-moss);
  box-shadow: 0 0 0 3px rgba(133, 192, 147, 0.15);
}

/* ───── Explorer Subtitle ───── */
.explorer-page__subtitle {
  display: flex;
  align-items: center;
  gap: var(--spacing-8);
  margin-top: 6px;
}
.active-origin-badge {
  background: rgba(133, 192, 147, 0.12);
  border: 0.5px solid var(--color-lichen);
  color: var(--color-forest-ink);
  padding: 4px 8px;
  border-radius: var(--radius-md);
  font-family: var(--font-muoto);
  font-size: 12px;
  font-weight: 500;
}
.change-origin-btn {
  background: none;
  border: none;
  color: var(--color-deep-fern);
  font-family: var(--font-denim);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  padding: 2px 6px;
}
.change-origin-btn:hover {
  text-decoration: underline;
}

.animate-fade-in {
  animation: fadeIn 0.25s ease-out;
}
@keyframes fadeIn {
  from { opacity: 0; transform: translateY(4px); }
  to { opacity: 1; transform: translateY(0); }
}
</style>
