import { sanitizeUrl } from '@/utils/url'

export const DEFAULT_SITE_NAME = 'SSXZ AI'
export const DEFAULT_SITE_LOGO = '/brand/ssxz-cat-dog-static.svg'

const legacySiteNames = new Set(['sub2api', 'ssxz api'])

export function normalizeSiteName(value?: string | null): string {
  const trimmed = value?.trim()
  if (!trimmed || legacySiteNames.has(trimmed.toLowerCase())) {
    return DEFAULT_SITE_NAME
  }
  return trimmed
}

export function normalizeSiteLogo(value?: string | null): string {
  const trimmed = value?.trim()
  if (!trimmed || /(?:^|\/)logo\.png(?:$|\?)/i.test(trimmed)) {
    return DEFAULT_SITE_LOGO
  }
  return trimmed
}

export function updateFavicon(logoUrl: string): void {
  const sanitizedLogoUrl = sanitizeUrl(logoUrl, {
    allowRelative: true,
    allowDataUrl: true,
  })
  if (!sanitizedLogoUrl) {
    return
  }

  let link = document.querySelector<HTMLLinkElement>('link[rel="icon"]')
  if (!link) {
    link = document.createElement('link')
    link.rel = 'icon'
    document.head.appendChild(link)
  }

  link.type = sanitizedLogoUrl.endsWith('.svg') ? 'image/svg+xml' : 'image/x-icon'
  link.href = sanitizedLogoUrl
}
