import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { describe, expect, it } from 'vitest'

const componentPath = resolve(dirname(fileURLToPath(import.meta.url)), '../AppSidebar.vue')
const componentSource = readFileSync(componentPath, 'utf8')
const headerSource = readFileSync(resolve(dirname(fileURLToPath(import.meta.url)), '../AppHeader.vue'), 'utf8')
const stylePath = resolve(dirname(fileURLToPath(import.meta.url)), '../../../style.css')
const styleSource = readFileSync(stylePath, 'utf8')

function getStyleBlocks(source: string, selector: string) {
  const escapedSelector = selector.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return [...source.matchAll(new RegExp(`(?:^|\\n)\\s*${escapedSelector}\\s*\\{([^}]*)\\}`, 'g'))]
    .map((match) => match[1])
}

function getStyleBlock(source: string, selector: string, requiredProperty?: string) {
  const blocks = getStyleBlocks(source, selector)
  const block = requiredProperty
    ? blocks.find((candidate) => getStyleDeclaration(candidate, requiredProperty) !== undefined)
    : blocks[0]

  if (block === undefined) {
    throw new Error(`Missing CSS block for ${selector}`)
  }

  return block
}

function getStyleDeclaration(block: string, property: string) {
  const escapedProperty = property.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
  return block.match(new RegExp(`(?:^|\\n)\\s*${escapedProperty}\\s*:\\s*([^;]+);`))?.[1].trim()
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
  it('prevents the shared 56px header height from regressing to 57px', () => {
    const tokenName = '--ssxz-header-height'
    const tokenReference = `var(${tokenName})`
    const tokenReferenceWithFallback = `var(${tokenName}, 56px)`
    const rootBlock = getStyleBlock(styleSource, ':root', tokenName)
    const sidebarHeaderBlock = getStyleBlock(styleSource, '.sidebar-header', 'height')
    const shellBlock = getStyleBlock(headerSource, '.app-header-shell', 'height')
    const innerBlock = getStyleBlock(headerSource, '.app-header-inner', 'height')

    expect(getStyleDeclaration(rootBlock, tokenName)).toBe('56px')
    expect(getStyleDeclaration(sidebarHeaderBlock, 'height')).toBe(tokenReference)
    expect(getStyleDeclaration(sidebarHeaderBlock, 'min-height')).toBe(tokenReference)
    expect(getStyleDeclaration(shellBlock, 'box-sizing')).toBe('border-box')
    expect(getStyleDeclaration(shellBlock, 'height')).toBe(tokenReferenceWithFallback)
    expect(getStyleDeclaration(shellBlock, 'min-height')).toBeUndefined()
    expect(getStyleDeclaration(innerBlock, 'height')).toBe(
      `calc(${tokenReferenceWithFallback} - 1px)`,
    )
    expect(getStyleDeclaration(innerBlock, 'min-height')).toBeUndefined()
  })

  it('keeps the app header title and version vertically centered', () => {
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

    expect(getStyleDeclaration(clusterBlock, 'display')).toBe('flex')
    expect(getStyleDeclaration(clusterBlock, 'align-items')).toBe('center')
    expect(getStyleDeclaration(clusterBlock, 'gap')).toBe('12px')
    expect(getStyleDeclaration(clusterBlock, 'min-width')).toBe('0')
    expect(getStyleDeclaration(copyBlock, 'min-width')).toBe('0')

    expect(getStyleDeclaration(titleBlock, 'margin')).toBe('0')
    expect(getStyleDeclaration(titleBlock, 'font-size')).toBe('18px')
    expect(getStyleDeclaration(titleBlock, 'font-weight')).toBe('600')
    expect(getStyleDeclaration(titleBlock, 'line-height')).toBe('20px')
    expect(getStyleDeclaration(descriptionBlock, 'margin')).toBe('2px 0 0')
    expect(getStyleDeclaration(descriptionBlock, 'font-size')).toBe('12px')
    expect(getStyleDeclaration(descriptionBlock, 'line-height')).toBe('16px')

    expect(getStyleDeclaration(versionBlock, 'flex')).toBe('none')
    expect(getStyleDeclaration(versionBlock, 'align-self')).toBe('center')
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
