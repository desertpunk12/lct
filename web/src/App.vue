<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import AppIcon from './components/AppIcon.vue'
import LeadForm from './components/LeadForm.vue'
import LeadsView from './components/LeadsView.vue'
import { useLeads } from './composables/useLeads'
import type { Lead } from './types/lead'

const { leads, isLoading, loadError, loadLeads, saveLead } = useLeads()
const drawer = ref<HTMLDialogElement>()
const leadForm = ref<InstanceType<typeof LeadForm>>()
const isDrawerOpen = ref(false)
const isSaving = ref(false)
const savedName = ref('')
const newLeadId = ref<number | null>(null)
let highlightTimer: ReturnType<typeof setTimeout> | undefined
let previousOverflow = ''
let opener: HTMLElement | null = null

onMounted(loadLeads)

async function openDrawer() {
  opener = document.activeElement instanceof HTMLElement ? document.activeElement : null
  previousOverflow = document.body.style.overflow
  document.body.style.overflow = 'hidden'
  isDrawerOpen.value = true
  await nextTick()
  drawer.value?.showModal()
}

function closeDrawer() {
  drawer.value?.close()
  isDrawerOpen.value = false
  isSaving.value = false
  document.body.style.overflow = previousOverflow
  nextTick(() => opener?.focus())
}

function onBackdropClick(event: MouseEvent) {
  if (event.target !== drawer.value || !drawer.value) return
  const rect = drawer.value.getBoundingClientRect()
  if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) {
    leadForm.value?.requestClose()
  }
}

function containDialogFocus(event: KeyboardEvent) {
  const controls = drawer.value?.querySelectorAll<HTMLElement>(
    'button:not(:disabled), input:not(:disabled), select:not(:disabled), a[href], [tabindex="0"]',
  )
  const visible = Array.from(controls ?? []).filter((control) => control.getClientRects().length)
  const first = visible[0]
  const last = visible.at(-1)
  if (!first || !last) {
    event.preventDefault()
    return
  }
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

function onSaved(lead: Lead) {
  savedName.value = `${lead.first_name} ${lead.last_name}`
  newLeadId.value = lead.id
  closeDrawer()
  clearTimeout(highlightTimer)
  highlightTimer = setTimeout(() => { newLeadId.value = null }, 4000)
}

onBeforeUnmount(() => {
  clearTimeout(highlightTimer)
  if (isDrawerOpen.value) document.body.style.overflow = previousOverflow
})
</script>

<template>
  <a href="#main-content" class="skip-link">Skip to leads</a>
  <header class="site-header">
    <div class="page-width flex min-h-20 items-center justify-between gap-4">
      <div class="flex items-center gap-3">
        <span class="brand-mark"><AppIcon name="ledger" class="size-6" /></span>
        <span class="brand-name">Lead ledger</span>
      </div>
      <button type="button" class="button button-header shrink-0" aria-haspopup="dialog" aria-controls="add-lead-dialog" @click="openDrawer">
        <AppIcon name="plus" class="size-4" /> Add a lead
      </button>
    </div>
  </header>

  <main id="main-content" class="page-width main-content" tabindex="-1">
    <div class="page-heading">
      <div>
        <h1 class="flex items-center gap-4">Leads <span v-if="!isLoading && !loadError" class="heading-count">{{ leads.length }}</span></h1>
        <p class="mt-3 max-w-xl text-sm leading-relaxed text-muted sm:text-base">Partner referrals and new enquiries, all in one place.</p>
      </div>
      <div class="hidden items-center gap-2.5 pb-1 text-sm text-muted sm:flex">
        <AppIcon name="car" class="size-5" /> Australian vehicle finance
      </div>
    </div>

    <div v-if="savedName" class="notice notice-success mb-6 flex items-center gap-3" role="status">
      <AppIcon name="check" class="size-5 shrink-0" />
      <p class="flex-1 text-sm"><span class="font-semibold">Lead saved.</span> {{ savedName }} is now in your ledger.</p>
      <button type="button" class="icon-button -my-1" aria-label="Dismiss saved confirmation" @click="savedName = ''"><AppIcon name="close" class="size-4" /></button>
    </div>

    <LeadsView :leads="leads" :loading="isLoading" :error="loadError" :new-lead-id="newLeadId" @add="openDrawer" @retry="loadLeads" />
  </main>

  <dialog id="add-lead-dialog" ref="drawer" class="lead-drawer" aria-labelledby="add-lead-title" aria-describedby="add-lead-description" @cancel.prevent="leadForm?.requestClose()" @click="onBackdropClick" @keydown.tab="containDialogFocus">
    <template v-if="isDrawerOpen">
      <header class="drawer-header">
        <div>
          <h2 id="add-lead-title" class="text-2xl font-semibold tracking-tight">Add a lead</h2>
          <p id="add-lead-description" class="mt-2 text-sm leading-relaxed text-muted">Fields are required unless marked optional.</p>
        </div>
        <button type="button" class="icon-button shrink-0" :disabled="isSaving" aria-label="Close add lead" @click="leadForm?.requestClose()"><AppIcon name="close" class="size-5" /></button>
      </header>
      <LeadForm ref="leadForm" :save="saveLead" @saved="onSaved" @cancel="closeDrawer" @busy-change="isSaving = $event" />
    </template>
  </dialog>
</template>
