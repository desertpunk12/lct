import type { Vehicle } from '../types/lead'

const wholeDollars = new Intl.NumberFormat('en-AU', {
  style: 'currency',
  currency: 'AUD',
  maximumFractionDigits: 0,
})

const dollarsAndCents = new Intl.NumberFormat('en-AU', {
  style: 'currency',
  currency: 'AUD',
})

const dateFormat = new Intl.DateTimeFormat('en-AU', {
  day: 'numeric',
  month: 'short',
  year: 'numeric',
})

export function formatMoney(amount: number): string {
  return (Number.isInteger(amount) ? wholeDollars : dollarsAndCents).format(amount)
}

export function formatDate(value: string): string {
  return dateFormat.format(new Date(value))
}

export function vehicleName(vehicle: Vehicle): string {
  return [vehicle.make, vehicle.model].filter(Boolean).join(' ')
}
