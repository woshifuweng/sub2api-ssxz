import { afterEach, describe, expect, it } from 'vitest'
import { compileStyle, parse } from 'vue/compiler-sfc'
import postcss from 'postcss'
import source from '../AppSectionShell.vue?raw'

const { descriptor } = parse(source)
const compiled = compileStyle({
  source: descriptor.styles[0].content, id: 'data-v-fullscreen-test', scoped: true,
})

afterEach(() => { document.body.replaceChildren() })

describe('image tools fullscreen shell stacking contract', () => {
  it('raises BOTH containing stacking contexts only while this shell has expanded image tools', () => {
    // Model the real shell's two stacking contexts, not the standalone QA host.
    expect(descriptor.template?.content).toContain('class="ssxz-app-content min-h-screen"')
    expect(descriptor.template?.content).toContain('class="ssxz-app-main relative z-10"')
    expect(compiled.errors).toEqual([])
    document.body.innerHTML = `<div class="ssxz-app-shell" data-v-fullscreen-test>
      <aside class="ssxz-app-sidebar"></aside>
      <main class="ssxz-app-content" data-v-fullscreen-test>
        <header class="ssxz-app-header"></header>
        <div class="ssxz-app-main" data-v-fullscreen-test><section class="image-tools"></section></div>
      </main></div>`
    const workspace = document.querySelector('.image-tools')!
    const content = document.querySelector('.ssxz-app-content')!
    const main = document.querySelector('.ssxz-app-main')!
    // Evaluate compiled selectors against DOM; jsdom cannot prove paint/hit-testing.
    const layers = new Map<Element, number>([[content, 1], [main, 10]])
    function zIndex(element: Element) {
      let value = layers.get(element)!
      postcss.parse(compiled.code).walkRules(rule => {
        if (!rule.selector.includes(':has(') || !element.matches(rule.selector)) return
        rule.walkDecls('z-index', declaration => { value = Number(declaration.value) })
      })
      return value
    }
    expect([zIndex(content), zIndex(main)]).toEqual([1, 10])
    workspace.classList.add('is-expanded')
    expect(zIndex(content)).toBeGreaterThan(30) // sidebar
    expect(zIndex(main)).toBeGreaterThan(20) // header inside content
    expect(zIndex(content)).toBeLessThan(110) // body-teleported group dialog
    workspace.classList.remove('is-expanded')
    expect([zIndex(content), zIndex(main)]).toEqual([1, 10])
    // An unrelated expanded widget must not change the shell's stacking.
    workspace.className = 'unrelated-widget is-expanded'
    expect([zIndex(content), zIndex(main)]).toEqual([1, 10])
  })
})
