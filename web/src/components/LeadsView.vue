<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { formatDate, formatMoney, vehicleName } from '../lib/format'
import { australianStates, type AustralianState, type Lead } from '../types/lead'
import AppIcon from './AppIcon.vue'

const props = defineProps<{
  leads: Lead[]
  loading: boolean
  error: string
  newLeadId: number | null
}>()
const emit = defineEmits<{ add: []; retry: [] }>()
const selectedState = ref<AustralianState | 'all'>('all')
const sort = ref('newest')
const stateCounts = computed(() => {
  const counts: Partial<Record<AustralianState, number>> = {}
  for (const lead of props.leads) counts[lead.state] = (counts[lead.state] ?? 0) + 1
  return counts
})
const visibleLeads = computed(() => {
  const filtered = props.leads.filter((lead) => selectedState.value === 'all' || lead.state === selectedState.value)
  return filtered.sort((a, b) => {
    if (sort.value === 'amount-high') return b.finance_amount - a.finance_amount || b.id - a.id
    if (sort.value === 'amount-low') return a.finance_amount - b.finance_amount || a.id - b.id
    if (sort.value === 'name') return `${a.first_name} ${a.last_name}`.localeCompare(`${b.first_name} ${b.last_name}`, 'en-AU') || a.id - b.id
    return Date.parse(b.created_at) - Date.parse(a.created_at) || b.id - a.id
  })
})

const resultsLabel = computed(() => {
  if (props.loading) return 'Loading leads'
  if (props.error && !props.leads.length) return 'Waiting for connection'
  return `Showing ${visibleLeads.value.length} of ${props.leads.length} leads`
})

watch(() => props.newLeadId, (id) => {
  if (id === null) return
  selectedState.value = 'all'
  sort.value = 'newest'
})
</script>

<template>
  <section aria-label="Lead register" :aria-busy="loading">
    <div class="ledger-toolbar">
      <div class="state-filters" role="group" aria-label="Filter leads by state">
        <button type="button" class="state-filter" :class="{ 'is-selected': selectedState === 'all' }" :aria-pressed="selectedState === 'all'" :disabled="loading" @click="selectedState = 'all'">
          All states <span class="filter-count">{{ loading ? '–' : leads.length }}</span>
        </button>
        <button v-for="state in australianStates" :key="state.code" type="button" class="state-filter" :class="{ 'is-selected': selectedState === state.code }" :aria-pressed="selectedState === state.code" :aria-label="`${state.name}, ${stateCounts[state.code] ?? 0} leads`" :disabled="loading" @click="selectedState = state.code">
          {{ state.code }} <span class="filter-count">{{ loading ? '–' : stateCounts[state.code] ?? 0 }}</span>
        </button>
      </div>
      <div class="flex shrink-0 items-center gap-2">
        <label for="lead-sort" class="text-xs text-muted">Sort by</label>
        <div class="relative">
          <select id="lead-sort" v-model="sort" class="sort-select" :disabled="loading">
            <option value="newest">Newest first</option>
            <option value="name">Name A–Z</option>
            <option value="amount-high">Amount: high to low</option>
            <option value="amount-low">Amount: low to high</option>
          </select>
          <AppIcon name="chevron" class="pointer-events-none absolute top-3 right-2 size-4 text-muted" />
        </div>
      </div>
    </div>

    <div v-if="error" class="notice notice-error mb-5 flex flex-wrap items-center gap-4" role="alert">
      <AppIcon name="warning" class="size-5 shrink-0" />
      <div class="min-w-0 flex-1">
        <h2 class="text-sm font-semibold">{{ leads.length ? 'Your list may be incomplete' : 'Your leads couldn’t be loaded' }}</h2>
        <p class="mt-1 text-sm">{{ error }}</p>
      </div>
      <button type="button" class="button button-secondary button-small" :disabled="loading" @click="emit('retry')">
        <AppIcon name="refresh" class="size-4" /> Try again
      </button>
    </div>

    <div class="ledger-surface">
      <div v-if="loading" class="loading-state" role="status">
        <div class="flex items-center gap-3 text-sm text-muted"><span class="loading-spinner" aria-hidden="true" /> Loading your leads…</div>
        <div class="mt-8 space-y-5" aria-hidden="true">
          <div v-for="row in 5" :key="row" class="skeleton-row">
            <span class="skeleton-line w-1/3" /><span class="skeleton-line w-1/6" /><span class="skeleton-line w-1/4" />
          </div>
        </div>
      </div>

      <div v-else-if="!leads.length" class="empty-state">
        <div class="empty-icon"><AppIcon name="ledger" class="size-8" /></div>
        <h2 class="mt-6 text-2xl font-semibold tracking-tight">{{ error ? 'Your ledger will appear here' : 'A fresh page for your leads' }}</h2>
        <p class="mx-auto mt-3 max-w-sm text-sm leading-6 text-muted">
          {{ error ? 'Reconnect to the lead service to see your enquiries. Nothing has been removed.' : 'Add your first lead to get started. Leads imported from partner files will appear here too.' }}
        </p>
        <button v-if="!error" type="button" class="button button-primary mt-7" @click="emit('add')"><AppIcon name="plus" class="size-4" /> Add a lead</button>
      </div>

      <div v-else-if="!visibleLeads.length" class="empty-state">
        <div class="empty-icon"><AppIcon name="ledger" class="size-8" /></div>
        <h2 class="mt-6 text-2xl font-semibold tracking-tight">No leads in {{ selectedState }} yet</h2>
        <p class="mt-3 text-sm text-muted">Try another state, or return to the full ledger.</p>
        <button type="button" class="button button-secondary mt-7" @click="selectedState = 'all'">Show all leads</button>
      </div>

      <template v-else>
        <div class="hidden lg:block">
          <table class="leads-table">
            <caption class="sr-only">Leads with contact details, state, requested finance, annual salary, vehicle, and date added. All amounts are in Australian dollars.</caption>
            <thead>
              <tr>
                <th scope="col">Contact</th>
                <th scope="col">State</th>
                <th scope="col" class="text-right">Finance amount</th>
                <th scope="col" class="text-right">Annual salary</th>
                <th scope="col">Vehicle</th>
                <th scope="col">Added</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="lead in visibleLeads" :key="lead.id" :class="{ 'new-lead': lead.id === newLeadId }">
                <th scope="row" class="contact-cell">
                  <div class="lead-name">{{ lead.first_name }} {{ lead.last_name }} <span v-if="lead.id === newLeadId" class="new-label">New</span></div>
                  <a class="contact-email" :href="`mailto:${lead.email}`">{{ lead.email }}</a>
                </th>
                <td><span class="state-badge">{{ lead.state }}</span></td>
                <td class="text-right"><span class="finance-amount">{{ formatMoney(lead.finance_amount) }}</span></td>
                <td class="text-right"><span v-if="lead.salary !== null" class="salary-amount">{{ formatMoney(lead.salary) }}</span><span v-else class="missing-value"><span aria-hidden="true">—</span><span class="sr-only">Not provided</span></span></td>
                <td>
                  <template v-if="lead.vehicle">
                    <div class="vehicle-name">{{ vehicleName(lead.vehicle) }}</div>
                    <div v-if="lead.vehicle.year !== null" class="mt-1 text-xs text-muted">{{ lead.vehicle.year }}</div>
                  </template>
                  <span v-else class="text-xs text-muted">Not provided</span>
                </td>
                <td><time :datetime="lead.created_at" class="whitespace-nowrap text-xs text-muted">{{ formatDate(lead.created_at) }}</time></td>
              </tr>
            </tbody>
          </table>
        </div>

        <ul class="mobile-leads lg:hidden" aria-label="Leads">
          <li v-for="lead in visibleLeads" :key="lead.id" class="mobile-lead" :class="{ 'new-lead': lead.id === newLeadId }">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0">
                <h2 class="lead-name">{{ lead.first_name }} {{ lead.last_name }} <span v-if="lead.id === newLeadId" class="new-label">New</span></h2>
                <a class="contact-email" :href="`mailto:${lead.email}`">{{ lead.email }}</a>
              </div>
              <span class="state-badge mt-0.5">{{ lead.state }}</span>
            </div>
            <dl class="mt-5 grid grid-cols-2 gap-4">
              <div><dt class="mobile-field-label">Finance amount</dt><dd class="finance-amount mt-1">{{ formatMoney(lead.finance_amount) }}</dd></div>
              <div><dt class="mobile-field-label">Vehicle</dt><dd class="mt-1 text-sm">{{ lead.vehicle ? vehicleName(lead.vehicle) : 'Not provided' }}<span v-if="lead.vehicle?.year" class="ml-1 text-muted">({{ lead.vehicle.year }})</span></dd></div>
            </dl>
            <details class="lead-more-details">
              <summary>More details <AppIcon name="chevron" class="size-3.5" /></summary>
              <dl class="mt-3 grid grid-cols-2 gap-4 pb-1">
                <div><dt class="mobile-field-label">Annual salary</dt><dd class="mt-1 text-sm">{{ lead.salary === null ? 'Not provided' : formatMoney(lead.salary) }}</dd></div>
                <div><dt class="mobile-field-label">Added</dt><dd class="mt-1 text-sm"><time :datetime="lead.created_at">{{ formatDate(lead.created_at) }}</time></dd></div>
              </dl>
            </details>
          </li>
        </ul>
      </template>

      <footer class="ledger-footer">
        <p role="status" aria-live="polite">{{ resultsLabel }}</p>
        <p>All amounts in <span class="font-medium text-ink">AUD</span></p>
      </footer>
    </div>
  </section>
</template>
