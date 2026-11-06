import assert from 'node:assert/strict'
import { test } from 'node:test'
import { emptyLeadDraft, parseDollars, toLeadInput, validateLeadDraft } from '../src/lib/leadForm.ts'
import { formatMoney, vehicleName } from '../src/lib/format.ts'

function validDraft() {
  return {
    ...emptyLeadDraft(),
    first_name: ' Zoë ',
    last_name: ' Martin ',
    email: ' ZOE.MARTIN@EXAMPLE.COM ',
    finance_amount: '38,500.50',
    state: 'QLD',
  }
}

test('requires the five mandatory fields, but not salary or vehicle', () => {
  assert.deepEqual(Object.keys(validateLeadDraft(emptyLeadDraft())).sort(), [
    'email', 'finance_amount', 'first_name', 'last_name', 'state',
  ])
  assert.deepEqual(validateLeadDraft(validDraft()), {})
})

test('trims names, normalizes email, and serializes omitted optional values as null', () => {
  assert.deepEqual(toLeadInput(validDraft()), {
    first_name: 'Zoë', last_name: 'Martin', email: 'zoe.martin@example.com',
    finance_amount: 38500.5, salary: null, state: 'QLD', vehicle: null,
  })
})

test('parses dollars and correctly grouped pasted currency without partial parses', () => {
  for (const [input, amount] of [['$45,000.50', 45000.5], ['.5', 0.5], ['1000', 1000], [' 12.00 ', 12]]) {
    assert.equal(parseDollars(String(input)), amount)
  }
  for (const invalid of ['', '1,2', '1,000,00', '5 dollars', '12abc', 'Infinity', 'NaN', '-1', '1e9']) {
    assert.ok(Number.isNaN(parseDollars(invalid)), invalid)
  }
})

test('finance and optional salary must be finite and strictly positive', () => {
  for (const value of ['0', '-1', 'bad', '9'.repeat(400)]) {
    const errors = validateLeadDraft({ ...validDraft(), finance_amount: value, salary: value })
    assert.equal(errors.finance_amount, 'must be greater than 0')
    assert.equal(errors.salary, 'must be greater than 0')
  }
  assert.deepEqual(validateLeadDraft({ ...validDraft(), finance_amount: '0.001', salary: '0.01' }), {})
  assert.equal(toLeadInput({ ...validDraft(), salary: ' $85,000 ' }).salary, 85000)
})

test('only accepts the eight Australian state and territory codes', () => {
  for (const state of ['ACT', 'NSW', 'NT', 'QLD', 'SA', 'TAS', 'VIC', 'WA']) {
    assert.deepEqual(validateLeadDraft({ ...validDraft(), state }), {})
  }
  assert.ok(validateLeadDraft({ ...validDraft(), state: 'XX' }).state)
})

test('whitespace is not a required value or an optional vehicle detail', () => {
  assert.equal(validateLeadDraft({ ...validDraft(), first_name: '  ' }).first_name, 'is required')
  assert.equal(toLeadInput({ ...validDraft(), 'vehicle.make': ' ', 'vehicle.model': ' ' }).vehicle, null)
})

test('model or year needs a make; a make alone is a valid partial vehicle', () => {
  assert.ok(validateLeadDraft({ ...validDraft(), 'vehicle.model': 'EV6' })['vehicle.make'])
  assert.ok(validateLeadDraft({ ...validDraft(), 'vehicle.year': '2024' })['vehicle.make'])
  const draft = { ...validDraft(), 'vehicle.make': ' Kia ' }
  assert.deepEqual(validateLeadDraft(draft), {})
  assert.deepEqual(toLeadInput(draft).vehicle, { make: 'Kia', model: null, year: null })
})

test('year allows 1990 through next year inclusive, but no fractions or exponents', () => {
  const draft = { ...validDraft(), 'vehicle.make': 'Kia' }
  for (const year of ['1990', '2027']) {
    assert.deepEqual(validateLeadDraft({ ...draft, 'vehicle.year': year }, 2027), {})
  }
  for (const year of ['1989', '2028', '2000.5', '2e3', 'year', '0']) {
    assert.ok(validateLeadDraft({ ...draft, 'vehicle.year': year }, 2027)['vehicle.year'], year)
  }
})

test('currency and partial vehicles are readable without invented optional data', () => {
  assert.equal(formatMoney(52000), '$52,000')
  assert.equal(formatMoney(38500.5), '$38,500.50')
  assert.equal(vehicleName({ make: 'Kia', model: null, year: null }), 'Kia')
  assert.equal(vehicleName({ make: 'Kia', model: 'EV6', year: null }), 'Kia EV6')
})
