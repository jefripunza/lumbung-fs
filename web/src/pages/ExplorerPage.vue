<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { useExplorerStore, type FileItem } from '@/stores/explorer'
import { useOriginsStore } from '@/stores/origins'
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
  return origin.domain
    .replace(/[^a-zA-Z0-9]+/g, '_')
    .replace(/^_+|_+$/g, '')
    .toLowerCase()
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

const showImagePreviewModal = ref(false)
const previewImageUrl = ref('')
const isPreviewLoading = ref(false)
const previewImageName = ref('')
const previewType = ref<'image' | 'video' | 'audio'>('image')

function isImage(name: string): boolean {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return ['jpg', 'jpeg', 'png', 'gif', 'webp', 'svg'].includes(ext)
}

function isVideo(name: string): boolean {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return ['mp4', 'webm', 'ogg', 'mov', 'm4v'].includes(ext)
}

function isAudio(name: string): boolean {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return ['mp3', 'wav', 'ogg', 'aac', 'm4a', 'flac'].includes(ext)
}

function isPdf(name: string): boolean {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  return ext === 'pdf'
}

function getMimeType(name: string): string {
  const ext = name.split('.').pop()?.toLowerCase() || ''
  const mimes: Record<string, string> = {
    jpg: 'image/jpeg',
    jpeg: 'image/jpeg',
    png: 'image/png',
    gif: 'image/gif',
    webp: 'image/webp',
    svg: 'image/svg+xml',
    mp4: 'video/mp4',
    webm: 'video/webm',
    ogg: 'video/ogg',
    mp3: 'audio/mpeg',
    wav: 'audio/wav',
    aac: 'audio/aac',
    m4a: 'audio/mp4',
    flac: 'audio/flac',
  }
  return mimes[ext] || 'application/octet-stream'
}

async function openItem(item: FileItem) {
  if (item.is_dir) {
    store.listItems(item.path)
  } else if (isImage(item.name)) {
    previewType.value = 'image'
    openImagePreview(item)
  } else if (isVideo(item.name)) {
    previewType.value = 'video'
    openImagePreview(item)
  } else if (isAudio(item.name)) {
    previewType.value = 'audio'
    openImagePreview(item)
  } else if (isPdf(item.name)) {
    try {
      const token = localStorage.getItem('lumbungfs_token')
      const url = `/api/explorer/download?path=${encodeURIComponent(item.path)}`
      const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
      if (!res.ok) throw new Error('Failed to load PDF')
      const blob = await res.blob()
      const pdfBlob = new Blob([blob], { type: 'application/pdf' })
      const blobUrl = URL.createObjectURL(pdfBlob)
      window.open(blobUrl, '_blank')
    } catch (err) {
      console.error('Error opening PDF:', err)
    }
  }
}

async function openImagePreview(item: FileItem) {
  isPreviewLoading.value = true
  previewImageUrl.value = ''
  previewImageName.value = item.name
  showImagePreviewModal.value = true

  try {
    const token = localStorage.getItem('lumbungfs_token')
    const url = `/api/explorer/download?path=${encodeURIComponent(item.path)}`
    const res = await fetch(url, { headers: { Authorization: `Bearer ${token}` } })
    if (!res.ok) throw new Error('Failed to load media')
    const blob = await res.blob()
    const mime = getMimeType(item.name)
    const mediaBlob = new Blob([blob], { type: mime })
    previewImageUrl.value = URL.createObjectURL(mediaBlob)
  } catch (err) {
    console.error('Error loading preview media:', err)
  } finally {
    isPreviewLoading.value = false
  }
}

function closeImagePreview() {
  showImagePreviewModal.value = false
  if (previewImageUrl.value) {
    URL.revokeObjectURL(previewImageUrl.value)
    previewImageUrl.value = ''
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
  try {
    await store.uploadFile(store.currentPath, file)
  } catch (err) {
    console.error('Upload failed:', err)
  } finally {
    target.value = ''
  }
}

const isGeneratingPresigned = ref(false)
const presignedUrlCopied = ref(false)

async function handleGeneratePresignedUrl() {
  if (!selectedOriginId.value) return
  isGeneratingPresigned.value = true
  try {
    const data = await store.generatePresignedUrl(selectedOriginId.value, store.currentPath)
    if (data && data.url) {
      let fullUrl = data.url
      if (fullUrl.startsWith('/')) {
        fullUrl = window.location.origin + fullUrl
      }
      await navigator.clipboard.writeText(fullUrl)
      presignedUrlCopied.value = true
      setTimeout(() => {
        presignedUrlCopied.value = false
      }, 2000)
    }
  } catch (err) {
    console.error('Failed to generate presigned URL:', err)
  } finally {
    isGeneratingPresigned.value = false
  }
}

const sortedItems = computed(() => {
  const folders = store.items.filter(item => item.is_dir).sort((a, b) => a.name.localeCompare(b.name))
  const files = store.items.filter(item => !item.is_dir).sort((a, b) => a.name.localeCompare(b.name))
  return [...folders, ...files]
})

function getPublicUrl(itemPath: string): string {
  const origin = selectedOrigin.value
  if (!origin) return ''
  const prefix = baseOriginPath.value + '/'
  let rel = itemPath
  if (itemPath.startsWith(prefix)) {
    rel = itemPath.substring(prefix.length)
  }
  return `https://${origin.domain}/file/${rel}`
}

const copiedFileId = ref<string | null>(null)

async function handleCopyUrl(item: FileItem) {
  const url = getPublicUrl(item.path)
  if (!url) return
  try {
    await navigator.clipboard.writeText(url)
    copiedFileId.value = item.path
    setTimeout(() => {
      if (copiedFileId.value === item.path) {
        copiedFileId.value = null
      }
    }, 2000)
  } catch (err) {
    console.error('Failed to copy file URL', err)
  }
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
          <OutlineButton @click="handleGeneratePresignedUrl" :disabled="isGeneratingPresigned">
            {{ presignedUrlCopied ? '📋 Copied!' : '🔗 Generate Presigned URL' }}
          </OutlineButton>
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

      <!-- Detail List Table -->
      <div v-else class="explorer-table-container animate-fade-in">
        <table class="explorer-table">
          <thead>
            <tr>
              <th class="explorer-table__th">NAME</th>
              <th class="explorer-table__th">SIZE</th>
              <th class="explorer-table__th">CREATED DATE</th>
              <th class="explorer-table__th text-right">ACTIONS</th>
            </tr>
          </thead>
          <tbody>
            <!-- Back row -->
            <tr
              v-if="store.currentPath && store.currentPath !== baseOriginPath"
              class="explorer-table__tr explorer-table__tr--back"
              @click="navigateUp()"
            >
              <td colspan="4" class="explorer-table__td">
                <span class="explorer-table__icon">⬆️</span>
                <span class="explorer-table__back-text">.. (Parent Directory)</span>
              </td>
            </tr>

            <!-- Sorted Items -->
            <tr
              v-for="item in sortedItems"
              :key="item.path"
              class="explorer-table__tr"
              :class="{ 'explorer-table__tr--dir': item.is_dir }"
              @click="openItem(item)"
            >
              <td class="explorer-table__td">
                <div class="explorer-table__name-cell">
                  <span class="explorer-table__icon">{{ fileIcon(item) }}</span>
                  <span class="explorer-table__name">{{ item.name }}</span>
                </div>
              </td>
              <td class="explorer-table__td explorer-table__size">
                {{ item.is_dir ? '—' : formatSize(item.size) }}
              </td>
              <td class="explorer-table__td explorer-table__date">
                {{ formatDate(item.modified_at) }}
              </td>
              <td class="explorer-table__td text-right" @click.stop>
                <div class="explorer-table__actions">
                  <button
                    v-if="!item.is_dir"
                    type="button"
                    class="explorer-table__action"
                    title="Copy URL"
                    @click="handleCopyUrl(item)"
                  >
                    {{ copiedFileId === item.path ? 'Copied!' : 'Copy URL' }}
                  </button>
                  <button
                    v-if="!item.is_dir"
                    type="button"
                    class="explorer-table__action"
                    title="Download"
                    @click="handleDownload(item)"
                  >
                    Download
                  </button>
                  <button
                    type="button"
                    class="explorer-table__action explorer-table__action--danger"
                    title="Delete"
                    @click="openDeleteConfirm(item)"
                  >
                    Delete
                  </button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
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

    <!-- Image Preview Modal -->
    <ModalDialog
      v-if="showImagePreviewModal"
      :title="previewImageName"
      max-width="800px"
      @close="closeImagePreview"
    >
      <div v-if="isPreviewLoading" class="image-preview-loading">
        <span class="explorer-page__spinner"></span>
        Loading preview…
      </div>
      <div v-else class="image-preview-modal-body">
        <div class="image-preview-container">
          <video
            v-if="previewType === 'video'"
            :src="previewImageUrl"
            controls
            autoplay
            class="preview-video"
          ></video>
          <div v-else-if="previewType === 'audio'" class="preview-audio-container">
            <span class="audio-player-icon">🎵</span>
            <audio
              :src="previewImageUrl"
              controls
              autoplay
              class="preview-audio"
            ></audio>
          </div>
          <img
            v-else
            :src="previewImageUrl"
            class="preview-img"
            alt="Preview"
          />
        </div>
      </div>
      <template #footer>
        <OutlineButton @click="closeImagePreview">Close</OutlineButton>
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

.image-preview-loading {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-12);
  font-family: var(--font-muoto);
  font-size: 14px;
  color: var(--color-slate-smoke);
  padding: var(--spacing-48) 0;
}
.image-preview-modal-body {
  display: flex;
  flex-direction: column;
  gap: var(--spacing-16);
}
.image-preview-container {
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: auto;
  max-height: 70vh;
  min-height: 320px;
  background: var(--color-sage-paper);
  border-radius: var(--radius-xl);
  padding: var(--spacing-16);
  border: 0.5px solid var(--color-lichen);
}
.preview-img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}
.preview-video {
  max-width: 100%;
  max-height: 100%;
  border-radius: var(--radius-lg);
  outline: none;
}
.preview-audio-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--spacing-16);
  width: 100%;
  padding: var(--spacing-24);
}
.audio-player-icon {
  font-size: 48px;
  animation: pulse 2s infinite ease-in-out;
}
@keyframes pulse {
  0%, 100% { transform: scale(1); opacity: 0.8; }
  50% { transform: scale(1.1); opacity: 1; }
}
.preview-audio {
  width: 100%;
  max-width: 400px;
  outline: none;
}

/* ───── Explorer Table ───── */
.explorer-table-container {
  background: var(--color-bone-white);
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-subtle);
  overflow-x: auto;
}
.explorer-table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
}
.explorer-table__th {
  font-family: var(--font-cinetype);
  font-size: 11px;
  letter-spacing: 0.15em;
  color: var(--color-slate-smoke);
  padding: 16px 20px;
  border-bottom: 0.5px solid var(--color-lichen);
  font-weight: 600;
}
.explorer-table__tr {
  transition: background 0.12s ease;
  cursor: default;
}
.explorer-table__tr--dir {
  cursor: pointer;
}
.explorer-table__tr--back {
  cursor: pointer;
}
.explorer-table__tr:hover {
  background: rgba(133, 192, 147, 0.05);
}
.explorer-table__td {
  padding: 14px 20px;
  font-family: var(--font-muoto);
  font-size: 13.5px;
  color: var(--color-forest-ink);
  border-bottom: 0.5px solid rgba(202, 211, 210, 0.4);
}
.explorer-table__tr:last-child .explorer-table__td {
  border-bottom: none;
}
.explorer-table__name-cell {
  display: flex;
  align-items: center;
  gap: 12px;
}
.explorer-table__icon {
  font-size: 20px;
  flex-shrink: 0;
}
.explorer-table__name {
  font-family: var(--font-denim);
  font-weight: 500;
  color: var(--color-forest-ink);
  word-break: break-all;
}
.explorer-table__back-text {
  font-family: var(--font-denim);
  font-weight: 500;
  color: var(--color-slate-smoke);
}
.explorer-table__size,
.explorer-table__date {
  color: var(--color-slate-smoke);
  font-size: 13px;
}
.explorer-table__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--spacing-8);
  flex-wrap: wrap;
}
.explorer-table__action {
  background: var(--color-bone-white);
  border: 0.5px solid var(--color-lichen);
  border-radius: var(--radius-md);
  padding: 4px 8px;
  font-family: var(--font-denim);
  font-size: 11px;
  font-weight: 500;
  color: var(--color-forest-ink);
  cursor: pointer;
  transition: all 0.12s ease;
}
.explorer-table__action:hover {
  background: var(--color-lichen);
}
.explorer-table__action--danger {
  border-color: rgba(196, 77, 77, 0.3);
  color: #8b2020;
}
.explorer-table__action--danger:hover {
  background: rgba(196, 77, 77, 0.08);
  border-color: #c44d4d;
}
.text-right {
  text-align: right;
}
</style>
