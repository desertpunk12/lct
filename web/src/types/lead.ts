export const australianStates = [
  { code: 'ACT', name: 'Australian Capital Territory' },
  { code: 'NSW', name: 'New South Wales' },
  { code: 'NT', name: 'Northern Territory' },
  { code: 'QLD', name: 'Queensland' },
  { code: 'SA', name: 'South Australia' },
  { code: 'TAS', name: 'Tasmania' },
  { code: 'VIC', name: 'Victoria' },
  { code: 'WA', name: 'Western Australia' },
] as const

export type AustralianState = (typeof australianStates)[number]['code']

export interface Vehicle {
  make: string
  model: string | null
  year: number | null
}

export interface LeadInput {
  first_name: string
  last_name: string
  email: string
  finance_amount: number
  salary: number | null
  state: AustralianState
  vehicle: Vehicle | null
}

export interface Lead extends LeadInput {
  id: number
  created_at: string
}

export type FieldErrors = Record<string, string>
