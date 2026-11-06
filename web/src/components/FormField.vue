<script setup lang="ts">
import { computed } from 'vue'
import { fieldLabels, type LeadField } from '../lib/leadForm'

const props = defineProps<{
  name: LeadField
  error?: string
  hint?: string
  optional?: boolean
}>()

const id = computed(() => `lead-${props.name.replace('.', '-')}`)
const attrs = computed(() => ({
  id: id.value,
  'aria-invalid': Boolean(props.error),
  'aria-describedby': [props.hint && `${id.value}-hint`, props.error && `${id.value}-error`]
    .filter(Boolean).join(' ') || undefined,
}))
</script>

<template>
  <div class="min-w-0" data-form-field>
    <label :for="id" class="field-label">
      {{ fieldLabels[name] }} <span v-if="optional" class="field-optional">optional</span>
    </label>
    <slot :attrs="attrs" />
    <p v-if="hint" :id="`${id}-hint`" class="field-hint">{{ hint }}</p>
    <p v-if="error" :id="`${id}-error`" class="field-error">{{ error }}</p>
  </div>
</template>
