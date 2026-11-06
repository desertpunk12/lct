import { australianStates } from '../types/lead.ts'
import type { AustralianState, FieldErrors, LeadInput } from '../types/lead'

export const fieldLabels = {
  first_name: 'First name',
  last_name: 'Last name',
  email: 'Email address',
  finance_amount: 'Finance amount',
  salary: 'Annual salary',
  state: 'State or territory',
  'vehicle.make': 'Vehicle make',
  'vehicle.model': 'Vehicle model',
  'vehicle.year': 'Vehicle year',
} as const

export type LeadField = keyof typeof fieldLabels
export type LeadDraft = Record<LeadField, string>

export function emptyLeadDraft(): LeadDraft {
  return {
    first_name: '',
    last_name: '',
    email: '',
    finance_amount: '',
    salary: '',
    state: '',
    'vehicle.make': '',
    'vehicle.model': '',
    'vehicle.year': '',
  }
}

// Accept plain dollars or correctly grouped pasted amounts, not partial parses
// such as "12abc" or "1,2". Do not impose a cents limit absent from the API rules.
export function parseDollars(value: string): number {
  const amount = value.trim().replace(/^\$\s*/, '')
  if (!/^(?:(?:\d+|\d{1,3}(?:,\d{3})+)(?:\.\d*)?|\.\d+)$/.test(amount)) return NaN
  return Number(amount.replaceAll(',', ''))
}

export function validateLeadDraft(draft: LeadDraft, nextYear = new Date().getFullYear() + 1): FieldErrors {
  const errors: FieldErrors = {}

  for (const field of ['first_name', 'last_name', 'email', 'state'] as const) {
    if (!draft[field].trim()) errors[field] = 'is required'
  }

  if (draft.state && !australianStates.some(({ code }) => code === draft.state)) {
    errors.state = 'must be a valid Australian state or territory'
  }

  for (const field of ['finance_amount', 'salary'] as const) {
    if (field === 'salary' && !draft.salary.trim()) continue
    const amount = parseDollars(draft[field])
    if (!Number.isFinite(amount) || amount <= 0) errors[field] = 'must be greater than 0'
  }

  const year = draft['vehicle.year'].trim()
  if (!draft['vehicle.make'].trim() && (draft['vehicle.model'].trim() || year)) {
    errors['vehicle.make'] = 'is required when a model or year is given'
  }
  if (year && (!/^\d+$/.test(year) || Number(year) < 1990 || Number(year) > nextYear)) {
    errors['vehicle.year'] = `must be a whole year between 1990 and ${nextYear}`
  }

  return errors
}

// Call only after validation. Explicit nulls preserve the README JSON contract.
export function toLeadInput(draft: LeadDraft): LeadInput {
  const make = draft['vehicle.make'].trim()
  const model = draft['vehicle.model'].trim()
  const year = draft['vehicle.year'].trim()

  return {
    first_name: draft.first_name.trim(),
    last_name: draft.last_name.trim(),
    email: draft.email.trim().toLowerCase(),
    finance_amount: parseDollars(draft.finance_amount),
    salary: draft.salary.trim() ? parseDollars(draft.salary) : null,
    state: draft.state as AustralianState,
    vehicle: make || model || year ? { make, model: model || null, year: year ? Number(year) : null } : null,
  }
}
