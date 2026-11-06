<script setup lang="ts">
import { computed, nextTick, reactive, ref } from 'vue'
import { ApiError } from '../api/leads'
import { emptyLeadDraft, fieldLabels, toLeadInput, validateLeadDraft, type LeadField } from '../lib/leadForm'
import { australianStates, type FieldErrors, type Lead, type LeadInput } from '../types/lead'
import AppIcon from './AppIcon.vue'
import FormField from './FormField.vue'

const props = defineProps<{ save: (input: LeadInput) => Promise<Lead> }>()
const emit = defineEmits<{
  saved: [lead: Lead]
  cancel: []
  'busy-change': [busy: boolean]
}>()

const draft = reactive(emptyLeadDraft())
const errors = ref<FieldErrors>({})
const requestError = ref('')
const isSubmitting = ref(false)
const confirmDiscard = ref(false)
const errorSummary = ref<HTMLElement>()
const keepEditingButton = ref<HTMLButtonElement>()
const formElement = ref<HTMLFormElement>()
const nextYear = new Date().getFullYear() + 1
const errorCount = computed(() => Object.keys(errors.value).length)
const isDirty = computed(() => Object.values(draft).some((value) => value.trim()))
const vehicleMakeRequired = computed(() => Boolean(draft['vehicle.model'].trim() || draft['vehicle.year'].trim()))

function clearError(field: LeadField) {
  delete errors.value[field]
  requestError.value = ''
}

function labelFor(field: string): string {
  return fieldLabels[field as LeadField] ?? field
}

function focusField(field: string) {
  const input = formElement.value?.elements.namedItem(field)
  if (input instanceof HTMLElement) {
    input.focus({ preventScroll: true })
    input.closest('[data-form-field]')?.scrollIntoView({ block: 'center' })
  }
}

async function requestClose() {
  if (isSubmitting.value) return
  if (!isDirty.value) {
    emit('cancel')
    return
  }
  confirmDiscard.value = true
  await nextTick()
  keepEditingButton.value?.focus()
}

function keepEditing() {
  confirmDiscard.value = false
  focusField('first_name')
}

async function submit() {
  if (isSubmitting.value) return
  confirmDiscard.value = false
  requestError.value = ''
  errors.value = validateLeadDraft(draft)

  if (errorCount.value) {
    await nextTick()
    errorSummary.value?.focus()
    return
  }

  isSubmitting.value = true
  emit('busy-change', true)

  try {
    const saved = await props.save(toLeadInput(draft))
    emit('saved', saved)
  } catch (error) {
    if (error instanceof ApiError && Object.keys(error.fields).length) {
      errors.value = error.fields
    } else {
      requestError.value = error instanceof Error ? error.message : 'The lead could not be saved. Try again.'
    }
    await nextTick()
    errorSummary.value?.focus()
  } finally {
    isSubmitting.value = false
    emit('busy-change', false)
  }
}

defineExpose({ requestClose })
</script>

<template>
  <form ref="formElement" class="flex min-h-0 flex-1 flex-col" novalidate :aria-busy="isSubmitting" @submit.prevent="submit">
    <div class="drawer-form-body">
      <div v-if="confirmDiscard" class="notice notice-warning mb-6" role="alert">
        <p class="font-semibold">Discard this draft?</p>
        <p class="mt-1 text-sm">This lead hasn’t been saved yet.</p>
        <div class="mt-4 flex flex-wrap gap-3">
          <button ref="keepEditingButton" type="button" class="button button-secondary button-small" @click="keepEditing">Keep editing</button>
          <button type="button" class="button button-danger button-small" @click="emit('cancel')">Discard draft</button>
        </div>
      </div>

      <div v-if="errorCount || requestError" ref="errorSummary" class="notice notice-error mb-6" role="alert" tabindex="-1">
        <template v-if="errorCount">
          <p class="font-semibold">{{ errorCount }} {{ errorCount === 1 ? 'field needs' : 'fields need' }} attention</p>
          <ul class="mt-2 space-y-1 text-sm">
            <li v-for="(message, field) in errors" :key="field">
              <a :href="`#lead-${field.replace('.', '-')}`" class="error-link" @click.prevent="focusField(field)">{{ labelFor(field) }}: {{ message }}</a>
            </li>
          </ul>
        </template>
        <template v-else>
          <p class="font-semibold">Lead wasn’t saved</p>
          <p class="mt-1 text-sm">{{ requestError }}</p>
          <p class="mt-2 text-sm">Your details are still here.</p>
        </template>
      </div>

      <fieldset :disabled="isSubmitting" class="form-section">
        <legend class="form-section-title">Contact details</legend>
        <div class="grid grid-cols-2 gap-x-4 gap-y-5">
          <FormField v-slot="{ attrs }" name="first_name" :error="errors.first_name">
            <input v-bind="attrs" v-model="draft.first_name" class="field-input" name="first_name" autocomplete="given-name" required autofocus @input="clearError('first_name')">
          </FormField>
          <FormField v-slot="{ attrs }" name="last_name" :error="errors.last_name">
            <input v-bind="attrs" v-model="draft.last_name" class="field-input" name="last_name" autocomplete="family-name" required @input="clearError('last_name')">
          </FormField>
          <FormField v-slot="{ attrs }" name="email" :error="errors.email" class="col-span-2">
            <input v-bind="attrs" v-model="draft.email" class="field-input" name="email" type="email" autocomplete="email" autocapitalize="none" :spellcheck="false" required placeholder="name@example.com" @input="clearError('email')">
          </FormField>
          <FormField v-slot="{ attrs }" name="state" :error="errors.state" class="col-span-2">
            <div class="relative">
              <select v-bind="attrs" v-model="draft.state" class="field-input appearance-none pr-10" name="state" autocomplete="address-level1" required @change="clearError('state')">
                <option disabled value="">Select a state or territory</option>
                <option v-for="state in australianStates" :key="state.code" :value="state.code">{{ state.name }} ({{ state.code }})</option>
              </select>
              <AppIcon name="chevron" class="pointer-events-none absolute top-3 right-3 size-5 text-muted" />
            </div>
          </FormField>
        </div>
      </fieldset>

      <fieldset :disabled="isSubmitting" class="form-section">
        <legend class="form-section-title">Finance details</legend>
        <p class="-mt-1 mb-5 text-sm text-muted">Enter amounts in Australian dollars.</p>
        <div class="grid grid-cols-2 gap-x-4 gap-y-5">
          <FormField v-slot="{ attrs }" name="finance_amount" :error="errors.finance_amount">
            <div class="relative">
              <span class="input-prefix" aria-hidden="true">$</span>
              <input v-bind="attrs" v-model="draft.finance_amount" class="field-input field-money" name="finance_amount" inputmode="decimal" required placeholder="45,000" @input="clearError('finance_amount')">
            </div>
          </FormField>
          <FormField v-slot="{ attrs }" name="salary" :error="errors.salary" optional>
            <div class="relative">
              <span class="input-prefix" aria-hidden="true">$</span>
              <input v-bind="attrs" v-model="draft.salary" class="field-input field-money" name="salary" inputmode="decimal" placeholder="85,000" @input="clearError('salary')">
            </div>
          </FormField>
        </div>
      </fieldset>

      <fieldset :disabled="isSubmitting" class="form-section">
        <legend class="form-section-title flex items-center gap-2">
          Vehicle <span class="section-optional">optional</span>
        </legend>
        <p class="-mt-1 mb-5 text-sm leading-relaxed text-muted">Add what you know. A make is needed if you include a model or year.</p>
        <div class="grid grid-cols-2 gap-x-4 gap-y-5">
          <FormField v-slot="{ attrs }" name="vehicle.make" :error="errors['vehicle.make']">
            <input v-bind="attrs" v-model="draft['vehicle.make']" class="field-input" name="vehicle.make" :required="vehicleMakeRequired" placeholder="e.g. Toyota" @input="clearError('vehicle.make')">
          </FormField>
          <FormField v-slot="{ attrs }" name="vehicle.model" :error="errors['vehicle.model']">
            <input v-bind="attrs" v-model="draft['vehicle.model']" class="field-input" name="vehicle.model" placeholder="e.g. RAV4" @input="clearError('vehicle.model')">
          </FormField>
          <FormField v-slot="{ attrs }" name="vehicle.year" :error="errors['vehicle.year']" :hint="`1990–${nextYear}`">
            <input v-bind="attrs" v-model="draft['vehicle.year']" class="field-input" name="vehicle.year" inputmode="numeric" placeholder="e.g. 2024" @input="clearError('vehicle.year')">
          </FormField>
        </div>
      </fieldset>
    </div>

    <footer class="drawer-footer">
      <button type="button" class="button button-secondary" :disabled="isSubmitting" @click="requestClose">Cancel</button>
      <button type="submit" class="button button-primary min-w-40" :disabled="isSubmitting">
        <span v-if="isSubmitting" class="loading-spinner" aria-hidden="true" />
        <AppIcon v-else name="check" class="size-4" />
        {{ isSubmitting ? 'Saving lead…' : 'Save lead' }}
      </button>
    </footer>
  </form>
</template>
