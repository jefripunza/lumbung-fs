import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/lib/api'

export interface Origin {
  id: string
  domain: string
  is_blocked: boolean
  api_key?: string
}

export interface UnknownOrigin {
  id: string
  domain: string
  access_at: string
  ip_address: string
}

export const useOriginsStore = defineStore('origins', () => {
  const origins = ref<Origin[]>([])
  const unknownOrigins = ref<UnknownOrigin[]>([])
  const isLoading = ref(false)

  async function fetchOrigins() {
    isLoading.value = true
    try {
      const { data } = await api.get('/origins')
      origins.value = data || []
    } finally {
      isLoading.value = false
    }
  }

  async function createOrigin(domain: string, is_blocked = false) {
    const { data } = await api.post('/origins', { domain, is_blocked })
    origins.value.push(data)
    return data
  }

  async function updateOrigin(id: string, domain: string, is_blocked: boolean) {
    const { data } = await api.put(`/origins?id=${id}`, { domain, is_blocked })
    const idx = origins.value.findIndex((o) => o.id === id)
    if (idx !== -1) origins.value[idx] = data
    return data
  }

  async function deleteOrigin(id: string) {
    await api.delete(`/origins?id=${id}`)
    origins.value = origins.value.filter((o) => o.id !== id)
  }

  async function generateApiKey(id: string) {
    const { data } = await api.post(`/origins/apikey?id=${id}`)
    const idx = origins.value.findIndex((o) => o.id === id)
    if (idx !== -1) origins.value[idx] = data
    return data
  }

  async function fetchUnknownOrigins() {
    isLoading.value = true
    try {
      const { data } = await api.get('/unknown-origins')
      unknownOrigins.value = data || []
    } finally {
      isLoading.value = false
    }
  }

  async function deleteUnknownOrigin(id: string) {
    await api.delete(`/unknown-origins?id=${id}`)
    unknownOrigins.value = unknownOrigins.value.filter((u) => u.id !== id)
  }

  async function clearAllUnknownOrigins() {
    await api.delete('/unknown-origins')
    unknownOrigins.value = []
  }

  async function promoteUnknownOrigin(id: string) {
    const { data } = await api.post(`/unknown-origins/promote?id=${id}`)
    unknownOrigins.value = unknownOrigins.value.filter((u) => u.id !== id)
    origins.value.push(data)
    return data
  }

  return {
    origins,
    unknownOrigins,
    isLoading,
    fetchOrigins,
    createOrigin,
    updateOrigin,
    deleteOrigin,
    fetchUnknownOrigins,
    deleteUnknownOrigin,
    clearAllUnknownOrigins,
    promoteUnknownOrigin,
    generateApiKey,
  }
})
