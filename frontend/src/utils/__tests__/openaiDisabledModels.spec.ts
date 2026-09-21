import { describe, expect, it } from 'vitest'
import { readOpenAIDisabledModels, parseOpenAIDisabledModels } from '../openaiDisabledModels'

describe('persistent per-account model quarantine', () => {
  it('roundtrips IDs without affecting other account settings', () => {
    const extra = { openai_passthrough: true, openai_disabled_models: ['gpt-5.5', 'alias'] }
    expect(parseOpenAIDisabledModels(readOpenAIDisabledModels(extra))).toEqual(extra.openai_disabled_models)
    expect(extra.openai_passthrough).toBe(true)
  })
  it('deduplicates line input and permits an explicit empty list to restore', () => {
    expect(parseOpenAIDisabledModels(' gpt-5.5\r\ngpt-5.5\n\n gpt-6')).toEqual(['gpt-5.5', 'gpt-6'])
    expect(parseOpenAIDisabledModels('')).toEqual([])
    expect(readOpenAIDisabledModels(undefined)).toBe('')
  })
})
