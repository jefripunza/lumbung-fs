import { defineStore } from 'pinia'
import { ref } from 'vue'
import api from '@/lib/api'

export interface Rule {
  id: string
  origin_id: string
  path: string
  validate_method: string
  validate_url: string
  validate_fallback_url: string
  is_max_size: boolean
  value_max_size: number
  value_unit_size: string
  is_extensions: boolean
  value_extensions: string
}

export const useRulesStore = defineStore('rules', () => {
  const rules = ref<Rule[]>([])
  const isLoading = ref(false)

  async function fetchRules(originId?: string) {
    isLoading.value = true
    try {
      const url = originId ? `/rules?origin_id=${originId}` : '/rules'
      const { data } = await api.get(url)
      rules.value = data || []
    } finally {
      isLoading.value = false
    }
  }

  async function createRule(rule: Omit<Rule, 'id'>) {
    const { data } = await api.post('/rules', rule)
    rules.value.push(data)
    return data
  }

  async function updateRule(id: string, rule: Partial<Rule>) {
    const { data } = await api.put(`/rules?id=${id}`, rule)
    const idx = rules.value.findIndex((r) => r.id === id)
    if (idx !== -1) rules.value[idx] = data
    return data
  }

  async function deleteRule(id: string) {
    await api.delete(`/rules?id=${id}`)
    rules.value = rules.value.filter((r) => r.id !== id)
  }

  return { rules, isLoading, fetchRules, createRule, updateRule, deleteRule }
})
