export const DEFAULT_SITE_NAME = 'SSXZ AI'
export const DEFAULT_SITE_SUBTITLE = '智能服务控制台'

const legacySiteNames = new Set(['sub2api', 'ssxz api'])
const legacySubtitles = new Set(['subscription to api conversion platform'])

export function normalizeSiteName(value?: string | null): string {
  const trimmed = value?.trim()
  if (!trimmed) return DEFAULT_SITE_NAME
  if (legacySiteNames.has(trimmed.toLowerCase())) return DEFAULT_SITE_NAME
  return trimmed
}

export function normalizeSiteSubtitle(value?: string | null): string {
  const trimmed = value?.trim()
  if (!trimmed) return DEFAULT_SITE_SUBTITLE
  if (legacySubtitles.has(trimmed.toLowerCase())) return DEFAULT_SITE_SUBTITLE
  return trimmed
}
