import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/lib/api'

export interface FileItem {
  name: string
  path: string
  is_dir: boolean
  size: number
  modified_at: string
}

export const useExplorerStore = defineStore('explorer', () => {
  const items = ref<FileItem[]>([])
  const currentPath = ref<string>('')
  const isLoading = ref(false)
  const uploadProgress = ref<number>(0)

  async function listItems(path = '') {
    isLoading.value = true
    try {
      const { data } = await api.get('/explorer/list', { params: { path } })
      items.value = data || []
      currentPath.value = path
    } finally {
      isLoading.value = false
    }
  }

  async function createFolder(parentPath: string, name: string) {
    await api.post('/explorer/folder', { path: parentPath, name })
    await listItems(parentPath)
  }

  async function uploadFile(path: string, file: File) {
    const formData = new FormData()
    formData.append('file', file)
    formData.append('path', path)
    uploadProgress.value = 0
    const { data } = await api.post('/explorer/upload', formData, {
      headers: { 'Content-Type': 'multipart/form-data' },
      onUploadProgress: (e) => {
        if (e.total) {
          uploadProgress.value = Math.round((e.loaded * 100) / e.total)
        }
      },
    })
    uploadProgress.value = 100
    await listItems(path)
    return data
  }

  function downloadFile(path: string) {
    const token = localStorage.getItem('lumbungfs_token')
    const url = `/api/explorer/download?path=${encodeURIComponent(path)}`
    const a = document.createElement('a')
    // Use fetch with auth header then trigger download
    fetch(url, { headers: { Authorization: `Bearer ${token}` } })
      .then((res) => res.blob())
      .then((blob) => {
        const blobUrl = URL.createObjectURL(blob)
        a.href = blobUrl
        a.download = path.split('/').pop() || 'download'
        a.click()
        URL.revokeObjectURL(blobUrl)
      })
  }

  async function deleteItem(path: string) {
    await api.delete('/explorer/delete', { params: { path } })
    await listItems(currentPath.value)
  }

  return { items, currentPath, isLoading, uploadProgress, listItems, createFolder, uploadFile, downloadFile, deleteItem }
})
