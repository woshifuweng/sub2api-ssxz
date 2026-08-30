import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const headerSource = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../AppHeader.vue'), 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

function getStyleBlock(source: string, selector: string) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return source.match(new RegExp(`${escapedSelector}\\s*\\{[^}]*\\}`))?.[0] ?? ''
}

describe('AppSidebar custom SVG styles', () => {
  it('does not override uploaded SVG fill or stroke colors', () => {
    expect(componentSource).toContain('.sidebar-svg-icon {')
    expect(componentSource).toContain('color: currentColor;')
    expect(componentSource).toContain('display: block;')
    expect(componentSource).not.toContain('stroke: currentColor;')
    expect(componentSource).not.toContain('fill: none;')
  })
})

describe('AppSidebar scroll position persistence', () => {
  it('binds a template ref to the sidebar nav element', () => {
    expect(componentSource).toContain('ref="sidebarNavRef"')
    expect(componentSource).toContain('sidebar-nav')
  })

  it('declares sidebarNavRef in script setup', () => {
    expect(componentSource).toContain("const sidebarNavRef = ref<HTMLElement | null>(null)")
  })

  it('saves scroll position on beforeUnmount', () => {
    expect(componentSource).toContain('onBeforeUnmount')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('sidebarNavRef.value.scrollTop')
  })

  it('restores scroll position on mount', () => {
    expect(componentSource).toContain('onMounted')
    expect(componentSource).toContain('appStore.sidebarScrollTop')
    expect(componentSource).toContain('nextTick')
  })
})

describe('AppSidebar header styles', () => {
  it('keeps the app header title, version, and divider aligned to the shared height', () => {
    const shellBlock = getStyleBlock(headerSource, '.app-header-shell')
    const innerBlock = getStyleBlock(headerSource, '.app-header-inner')
    const clusterBlock = getStyleBlock(headerSource, '.app-header-title-cluster')
    const copyBlock = getStyleBlock(headerSource, '.app-header-title-copy')
    const titleBlock = getStyleBlock(headerSource, '.app-header-title-copy h1')
    const descriptionBlock = getStyleBlock(headerSource, '.app-header-title-copy p')
    const versionBlock = getStyleBlock(headerSource, '.app-header-version')

    expect(headerSource).toContain('class="app-header-title-cluster"')
    expect(headerSource).toContain('class="app-header-title-copy hidden lg:block"')
    expect(headerSource).toMatch(
      /<VersionBadge\s+v-if="authStore\.isAdmin"\s+:runtime-actions-enabled="false"\s+class="app-header-version"\s*\/>/,
    )
    expect(headerSource).not.toContain('class="mt-0.5"')

    expect(shellBlock).toContain('height: var(--ssxz-header-height, 56px);')
    expect(shellBlock).not.toContain('min-height:')
    expect(innerBlock).toContain('height: var(--ssxz-header-height, 56px);')
    expect(innerBlock).not.toContain('min-height:')

    expect(clusterBlock).toContain('display: flex;')
    expect(clusterBlock).toContain('align-items: center;')
    expect(clusterBlock).toContain('gap: 12px;')
    expect(clusterBlock).toContain('min-width: 0;')
    expect(copyBlock).toContain('min-width: 0;')

    expect(titleBlock).toContain('margin: 0;')
    expect(titleBlock).toContain('font-size: 18px;')
    expect(titleBlock).toContain('font-weight: 600;')
    expect(titleBlock).toContain('line-height: 20px;')
    expect(descriptionBlock).toContain('margin: 2px 0 0;')
    expect(descriptionBlock).toContain('font-size: 12px;')
    expect(descriptionBlock).toContain('line-height: 16px;')

    expect(versionBlock).toContain('flex: none;')
    expect(versionBlock).toContain('align-self: center;')
  })

  it('does not clip the version badge dropdown', () => {
    const sidebarHeaderBlockMatch = styleSource.match(/\.sidebar-header\s*\{[\s\S]*?\n {2}\}/)
    const sidebarBrandBlockMatch = componentSource.match(/\.ssxz-sidebar-brand-copy\s*\{[\s\S]*?\n\}/)
    const appHeaderBlockMatch = headerSource.match(/\.app-header-shell\s*\{[\s\S]*?\n\}/)

    expect(sidebarHeaderBlockMatch).not.toBeNull()
    expect(sidebarBrandBlockMatch).not.toBeNull()
    expect(headerSource).toContain('<VersionBadge')
    expect(sidebarHeaderBlockMatch?.[0]).not.toContain('@apply overflow-hidden;')
    expect(sidebarBrandBlockMatch?.[0]).not.toContain('overflow: hidden;')
    expect(appHeaderBlockMatch?.[0]).not.toContain('overflow: hidden;')
  })
})
