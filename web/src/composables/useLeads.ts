import { onBeforeUnmount, ref } from 'vue'
import { createLead, fetchLeads } from '../api/leads'
import type { Lead, LeadInput } from '../types/lead'

export function useLeads() {
  const leads = ref<Lead[]>([])
  const isLoading = ref(true)
  const loadError = ref('')
  let controller: AbortController | undefined
  let savedDuringLoad: Lead[] = []

  async function loadLeads() {
    controller?.abort()
    const request = new AbortController()
    controller = request
    savedDuringLoad = []
    isLoading.value = true
    loadError.value = ''

    try {
      const result = await fetchLeads(request.signal)
      if (request.signal.aborted) return
      // A slow initial GET must not overwrite a lead saved while it was loading.
      const savedIds = new Set(savedDuringLoad.map((lead) => lead.id))
      leads.value = [...savedDuringLoad, ...result.filter((lead) => !savedIds.has(lead.id))]
    } catch (error) {
      if (!request.signal.aborted) {
        loadError.value = error instanceof Error ? error.message : 'Leads could not be loaded. Try again.'
      }
    } finally {
      if (!request.signal.aborted) isLoading.value = false
    }
  }

  async function saveLead(input: LeadInput): Promise<Lead> {
    const saved = await createLead(input)
    if (isLoading.value) savedDuringLoad.push(saved)
    leads.value = [saved, ...leads.value.filter((lead) => lead.id !== saved.id)]
    return saved
  }

  onBeforeUnmount(() => controller?.abort())

  return { leads, isLoading, loadError, loadLeads, saveLead }
}
