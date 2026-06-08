export const DEFAULT_AUTH_REDIRECT = '/app'

const CANONICAL_APP_REDIRECTS = new Set([
  '/app',
  '/app/chat',
  '/app/image'
])

const LEGACY_AUTH_REDIRECTS: Record<string, string> = {
  '/dashboard': '/app',
  '/home': '/',
  '/ai-chat': '/app/chat',
  '/image-studio': '/app/image',
  '/apps': '/app',
  '/monitor': '/app',
  '/sora': '/app/image'
}

function firstQueryValue(value: unknown): string | undefined {
  if (Array.isArray(value)) {
    const first = value.find((item) => typeof item === 'string')
    return typeof first === 'string' ? first : undefined
  }
  return typeof value === 'string' ? value : undefined
}

function normalizeInternalPath(rawPath: string, fallback: string): string {
  if (!rawPath) return fallback
  if (!rawPath.startsWith('/')) return fallback
  if (rawPath.startsWith('//')) return fallback
  if (rawPath.includes('://')) return fallback
  if (rawPath.includes('\n') || rawPath.includes('\r')) return fallback

  const hashIndex = rawPath.indexOf('#')
  const beforeHash = hashIndex >= 0 ? rawPath.slice(0, hashIndex) : rawPath
  const hash = hashIndex >= 0 ? rawPath.slice(hashIndex) : ''
  const queryIndex = beforeHash.indexOf('?')
  const pathname = queryIndex >= 0 ? beforeHash.slice(0, queryIndex) : beforeHash
  const query = queryIndex >= 0 ? beforeHash.slice(queryIndex) : ''

  if (pathname === '/login' || pathname === '/register') {
    return fallback
  }

  const mappedPath = LEGACY_AUTH_REDIRECTS[pathname] || pathname
  if (!CANONICAL_APP_REDIRECTS.has(mappedPath)) {
    return fallback
  }

  return `${mappedPath}${query}${hash}`
}

export function resolveAuthRedirect(value: unknown, fallback = DEFAULT_AUTH_REDIRECT): string {
  const rawValue = firstQueryValue(value)
  if (!rawValue) return fallback

  try {
    if (rawValue.includes('://')) {
      return fallback
    }
  } catch {
    return fallback
  }

  return normalizeInternalPath(rawValue, fallback)
}

export function resolveRouteAuthRedirect(
  query: Record<string, unknown>,
  fallback = DEFAULT_AUTH_REDIRECT
): string {
  return resolveAuthRedirect(query.returnTo ?? query.redirect, fallback)
}
