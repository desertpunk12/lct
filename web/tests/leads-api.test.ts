import assert from 'node:assert/strict'
import { afterEach, mock, test } from 'node:test'
import { ApiError, createLead, fetchLeads } from '../src/api/leads.ts'
import type { LeadInput } from '../src/types/lead.ts'

const input: LeadInput = {
  first_name: 'Zoë', last_name: 'Martin', email: 'zoe@example.com',
  finance_amount: 38500.5, state: 'QLD', salary: null, vehicle: null,
}

afterEach(() => mock.restoreAll())

test('GET uses a relative API URL and preserves an empty array', async () => {
  mock.method(globalThis, 'fetch', async (url: string, options: RequestInit) => {
    assert.equal(url, '/api/leads')
    assert.equal(options.method, 'GET')
    return Response.json([])
  })
  assert.deepEqual(await fetchLeads(), [])
})

test('POST preserves the contract, including explicit nulls, and reads the saved lead', async () => {
  const saved = { ...input, id: 12, created_at: '2026-11-03T10:15:00+11:00' }
  mock.method(globalThis, 'fetch', async (url: string, options: RequestInit) => {
    assert.equal(url, '/api/leads')
    assert.equal(options.method, 'POST')
    assert.deepEqual(options.headers, { 'Content-Type': 'application/json' })
    assert.deepEqual(JSON.parse(String(options.body)), input)
    return Response.json(saved, { status: 201 })
  })
  assert.deepEqual(await createLead(input), saved)
})

test('422 exposes API messages verbatim including dotted vehicle fields', async () => {
  const errors = { 'vehicle.make': 'is required when a model or year is given', finance_amount: 'must be greater than 0' }
  mock.method(globalThis, 'fetch', async () => Response.json({ errors }, { status: 422 }))
  await assert.rejects(createLead(input), (error: unknown) => {
    assert.ok(error instanceof ApiError)
    assert.equal(error.status, 422)
    assert.deepEqual(error.fields, errors)
    return true
  })
})

test('409 without a field payload still points to the email field', async () => {
  mock.method(globalThis, 'fetch', async () => new Response(null, { status: 409 }))
  await assert.rejects(createLead(input), (error: unknown) => {
    assert.ok(error instanceof ApiError)
    assert.match(error.fields.email ?? '', /already exists/)
    return true
  })
})

test('500 and malformed successful responses provide actionable errors', async () => {
  const fetchMock = mock.method(globalThis, 'fetch', async () => new Response('Unavailable', { status: 500 }))
  await assert.rejects(fetchLeads(), /unavailable/)
  fetchMock.mock.mockImplementation(async () => Response.json({ leads: [] }))
  await assert.rejects(fetchLeads(), /unexpected response/)
  fetchMock.mock.mockImplementation(async () => new Response('<html>not JSON</html>'))
  await assert.rejects(fetchLeads(), /unexpected response/)
})

test('network failures explain how to recover; caller cancellation is preserved', async () => {
  mock.method(globalThis, 'fetch', async () => { throw new TypeError('Failed to fetch') })
  await assert.rejects(fetchLeads(), /Check your connection and try again/)

  const controller = new AbortController()
  controller.abort()
  await assert.rejects(fetchLeads(controller.signal), TypeError)
})
