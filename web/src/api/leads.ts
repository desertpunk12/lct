import type { FieldErrors, Lead, LeadInput } from '../types/lead'

export class ApiError extends Error {
  readonly status: number
  readonly fields: FieldErrors

  constructor(message: string, status: number, fields: FieldErrors = {}) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.fields = fields
  }
}

function readFieldErrors(body: unknown): FieldErrors {
  if (!body || typeof body !== 'object' || !('errors' in body)) return {}
  const errors = body.errors
  if (!errors || typeof errors !== 'object' || Array.isArray(errors)) return {}

  return Object.fromEntries(
    Object.entries(errors).filter((entry): entry is [string, string] => typeof entry[1] === 'string'),
  )
}

async function request<T>(options: RequestInit, expectedStatus: number): Promise<T> {
  const timeout = AbortSignal.timeout(15_000)
  const signal = options.signal ? AbortSignal.any([options.signal, timeout]) : timeout
  let response: Response

  try {
    response = await fetch('/api/leads', { ...options, signal })
  } catch (error) {
    if (options.signal?.aborted) throw error
    throw new ApiError(
      timeout.aborted
        ? 'The lead service took too long to respond. Try again in a moment.'
        : 'The lead service could not be reached. Check your connection and try again.',
      0,
    )
  }

  const body: unknown = await response.json().catch(() => null)

  if (!response.ok) {
    const fields = readFieldErrors(body)
    if (response.status === 409 && !fields.email) {
      fields.email = 'A lead with this email address already exists.'
    }
    let message = 'Check the highlighted details and try again.'
    if (response.status >= 500) message = 'The lead service is unavailable. Try again in a moment.'
    if (response.status === 400) message = 'The lead could not be saved. Check the details and try again.'
    throw new ApiError(message, response.status, fields)
  }

  if (response.status !== expectedStatus || body === null) {
    throw new ApiError('The lead service returned an unexpected response. Try again.', response.status)
  }

  return body as T
}

export async function fetchLeads(signal?: AbortSignal): Promise<Lead[]> {
  const leads = await request<Lead[]>({ method: 'GET', signal }, 200)
  if (!Array.isArray(leads)) {
    throw new ApiError('The lead service returned an unexpected response. Try again.', 200)
  }
  return leads
}

export function createLead(input: LeadInput): Promise<Lead> {
  return request<Lead>(
    {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(input),
    },
    201,
  )
}
