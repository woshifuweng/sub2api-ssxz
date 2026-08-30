export function resolvePublicApiBaseUrl(configuredUrl?: string | null, currentOrigin = ''): string {
  const configured = configuredUrl?.trim()
  if (configured) return configured

  const origin = currentOrigin.trim().replace(/\/+$/, '')
  return origin ? `${origin}/v1` : '/v1'
}
