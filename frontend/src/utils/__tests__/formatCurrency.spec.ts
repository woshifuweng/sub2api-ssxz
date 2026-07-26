import { describe, expect, it } from 'vitest'

import { formatCostFixed, formatCurrency, formatCurrencyExact, formatCurrencyTitle } from '../format'

describe('currency presentation', () => {
  it('uses two decimals for primary money displays', () => {
    expect(formatCurrency(3.8421)).toBe('$3.84')
    expect(formatCurrency(486.7284)).toBe('$486.73')
    expect(formatCostFixed(4.1286)).toBe('4.13')
  })

  it('keeps the precise amount available for detail and hover text', () => {
    expect(formatCurrencyExact(0.00675773)).toBe('$0.00675773')
    expect(formatCurrencyTitle(0.00675773)).toBe('精确金额：$0.00675773')
  })

  it('expands sub-cent charges instead of rounding them to $0.00', () => {
    expect(formatCurrency(0.0003)).toBe('$0.0003')
    expect(formatCurrency(0.004999)).toBe('$0.004999')
    expect(formatCurrency(0.0099)).toBe('$0.0099')
    expect(formatCurrency(0)).toBe('$0.00')
    expect(formatCurrency(0.01)).toBe('$0.01')
    expect(formatCurrency(-0.0003)).toBe('-$0.0003')
  })
})
