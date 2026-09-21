/** Exact IDs only. Backend validation rejects wildcard/invalid values. */
export function readOpenAIDisabledModels(extra: Record<string, unknown> | undefined): string {
  const models = extra?.openai_disabled_models
  return Array.isArray(models) ? models.filter((value): value is string => typeof value === 'string').join('\n') : ''
}

export function parseOpenAIDisabledModels(text: string): string[] {
  return [...new Set(text.split(/\r?\n/).map(value => value.trim()).filter(Boolean))]
}
