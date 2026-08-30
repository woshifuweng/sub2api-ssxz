import type { RouteLocationGeneric } from 'vue-router'

export const redirectLegacyRoute = (path: string) => (
  to: Pick<RouteLocationGeneric, 'query' | 'hash'>,
) => ({
  path,
  query: to.query,
  hash: to.hash,
})
