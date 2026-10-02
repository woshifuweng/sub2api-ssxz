<template>
  <article
    id="cc-switch-guide"
    data-testid="cc-switch-guide"
    class="cc-switch-guide"
  >
    <h1>从零开始，用上 SSXZ</h1>
    <p>不用懂 GitHub，也不用先学配置文件。选好你要用的软件，只看与你有关的步骤。</p>
    <fieldset class="guide-choices">
      <legend>你想用什么？</legend>
      <button v-for="item in tools" :key="item.value" type="button" :aria-pressed="tool === item.value" @click="tool = item.value">{{ item.label }}</button>
    </fieldset>
    <fieldset v-if="tool !== 'image'" class="guide-choices">
      <legend>你的电脑是什么系统？</legend>
      <button type="button" :aria-pressed="system === 'windows'" @click="system = 'windows'">Windows</button>
      <button type="button" :aria-pressed="system === 'mac'" @click="system = 'mac'">Mac</button>
    </fieldset>
    <p class="guide-summary" role="status">{{ tool === 'image' ? '直接在浏览器画图，无需安装客户端。' : `${tool === 'codex' ? 'Codex' : 'Claude Code'} · ${system === 'windows' ? 'Windows' : 'Mac'}：安装 → 准备密钥 → 导入 → 收到第一条回复` }}</p>
    <p class="guide-image-hint">图看不清？点击图片可放大。截图中的版本、账号和模型仅为界面示例。</p>
    <div class="guide-content" @click="openImage" v-html="guideHtml" />
    <section v-if="tool !== 'image'" aria-label="遇到问题怎么办">
      <h2>卡住了？按你看到的现象找答案</h2>
      <details v-for="help in helps" :key="help.title">
        <summary>{{ help.title }}</summary><p>{{ help.body }}</p>
      </details>
    </section>
    <dialog ref="imageDialog" class="guide-lightbox" aria-label="教程图片放大" @click="closeBackdrop">
      <button type="button" autofocus @click="imageDialog?.close()">关闭图片（Esc）</button>
      <img v-if="activeImage" :src="activeImage.src" :alt="activeImage.alt" />
      <p>{{ activeImage?.alt }}</p>
    </dialog>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { renderRichContent } from '@/utils/sanitize'
import { installationGuide, connectionGuide, imageGuide, type GuideTool, type GuideSystem } from './guideContent'
import downloadImage from '../../../../docs/教程/assets/cc-switch/01-official-download-windows-macos.png'
import mainWindowImage from '../../../../docs/教程/assets/cc-switch/02-cc-switch-main-window.png'
import importButtonImage from '../../../../docs/教程/assets/cc-switch/03-ssxz-import-to-ccs-button.png'
import browserPromptImage from '../../../../docs/教程/assets/cc-switch/04-browser-open-cc-switch-dialog.png'
import importConfirmationImage from '../../../../docs/教程/assets/cc-switch/05-cc-switch-import-confirmation.png'
import selectedProviderImage from '../../../../docs/教程/assets/cc-switch/06-ssxz-selected-redacted.png'
import successImage from '../../../../docs/教程/assets/cc-switch/07-claude-code-success-history.png'
import directDownloadImage from '@/assets/docs/08-official-direct-download.jpg'

const guideAssets: Record<string, string> = {
  './assets/cc-switch/08-official-direct-download.jpg': directDownloadImage,
  './assets/cc-switch/01-official-download-windows-macos.png': downloadImage,
  './assets/cc-switch/02-cc-switch-main-window.png': mainWindowImage,
  './assets/cc-switch/03-ssxz-import-to-ccs-button.png': importButtonImage,
  './assets/cc-switch/04-browser-open-cc-switch-dialog.png': browserPromptImage,
  './assets/cc-switch/05-cc-switch-import-confirmation.png': importConfirmationImage,
  './assets/cc-switch/06-ssxz-selected-redacted.png': selectedProviderImage,
  './assets/cc-switch/07-claude-code-success-history.png': successImage
}

const tool = ref<GuideTool>('codex')
const system = ref<GuideSystem>('windows')
const tools: { value: GuideTool; label: string }[] = [{ value: 'codex', label: 'Codex' }, { value: 'claude', label: 'Claude Code' }, { value: 'image', label: '图片工作台' }]
const imageDialog = ref<HTMLDialogElement | null>(null)
const activeImage = ref<{ src: string; alt: string } | null>(null)
const helps = [
  { title: '点“导入到 CCS”没有反应', body: '确认 CC Switch 已安装并打开过一次，再点导入，允许浏览器打开 CC Switch。仍不行，更新官方版本；不要将导入链接发给别人，其中可能包含密钥。' },
  { title: '找不到软件 / 提示命令不存在', body: 'CC Switch 不会自动安装 Codex 或 Claude Code。回到第二步安装对应客户端；安装后关闭并重新打开终端。Claude 仍找不到时，按官方安装输出中的路径提示处理，不要关闭安全保护。' },
  { title: '打开后还是旧账号或旧线路', body: '检查 CC Switch 是否在正确的软件页签、SSXZ 是否启用，然后完全退出并重开客户端。已登录 ChatGPT 不等于已接入 SSXZ；用本站刚产生的使用记录确认。' },
  { title: '401 / 403 / 余额不足', body: '检查本站密钥是否启用、过期、单独额度耗尽、分组有权限，以及账户可用额度。账户有余额也可能受密钥额度或上游错误影响，不要直接认定要充值；保留时间和错误编号联系本站客服，隐藏完整密钥。' },
  { title: '模型不存在 / 没权限 / 没有可用账号', body: '回本站模型页面核对该密钥分组允许的准确模型名称，不能把相似名字互换。若页面显示可用却请求失败，请提供时间、模型、分组和错误编号，不要反复切换随机模型。' },
  { title: '一直等待 / Failed to fetch / 429', body: '先停止连续重试，检查网络和本站使用记录。429 常与请求太频繁或额度限制有关；长时间等待需根据具体错误排查，不能只凭提示认定为某一家网关故障。' },
]

function openImage(event: MouseEvent): void {
  const target = event.target instanceof Element ? event.target.closest('a[data-guide-image]') : null
  if (!(target instanceof HTMLAnchorElement)) return
  event.preventDefault()
  const img = target.querySelector('img')
  if (!img) return
  activeImage.value = { src: img.src, alt: img.alt }
  imageDialog.value?.showModal()
}

function closeBackdrop(event: MouseEvent): void {
  const dialog = imageDialog.value
  if (!dialog || event.target !== dialog) return
  const rect = dialog.getBoundingClientRect()
  if (event.clientX < rect.left || event.clientX > rect.right || event.clientY < rect.top || event.clientY > rect.bottom) dialog.close()
}

const guideHtml = computed(() => {
  const guideMarkdown = tool.value === 'image' ? imageGuide : installationGuide(tool.value, system.value) + '\n\n' + connectionGuide(tool.value)
  const source = Object.entries(guideAssets).reduce(
    (content, [assetPath, assetUrl]) => content.split(assetPath).join(assetUrl),
    guideMarkdown
  )
  // Links remain useful without dialog support and are keyboard accessible.
  return renderRichContent(source)
    .replace(/<a href="(https:\/\/(?:ccswitch\.io\/zh\/download|developers\.openai\.com\/codex\/app\/))"/g, '<a class="guide-download" target="_blank" rel="noopener noreferrer" href="$1"')
    .replace(/<img\b[^>]*src="([^"]+)"[^>]*>/g, (tag, src) => `<a data-guide-image href="${src}" target="_blank" rel="noopener noreferrer" aria-label="放大教程图片">${tag}</a>`)
})
</script>

<style scoped>
.cc-switch-guide {
  min-width: 0;
  color: hsl(var(--foreground));
}

.guide-choices { border: 0; padding: 0; margin: 1.25rem 0; display: flex; flex-wrap: wrap; gap: .6rem; }
.guide-choices legend { font-weight: 650; margin-bottom: .6rem; }
.guide-choices button { border: 1px solid hsl(var(--border)); border-radius: .65rem; padding: .65rem 1rem; background: hsl(var(--card)); color: hsl(var(--foreground)); cursor: pointer; }
.guide-choices button[aria-pressed="true"] { background: hsl(var(--foreground)); color: hsl(var(--background)); }
.guide-choices button:focus-visible, summary:focus-visible { outline: 2px solid #0d9488; outline-offset: 3px; }
.guide-summary { padding: 1rem; background: hsl(var(--muted)); border-radius: .65rem; }
.cc-switch-guide details { margin: .7rem 0; padding: .85rem 1rem; border: 1px solid hsl(var(--border)); border-radius: .65rem; }
.cc-switch-guide summary { cursor: pointer; font-weight: 600; }
.cc-switch-guide :deep(pre) { overflow-x: auto; white-space: pre-wrap; overflow-wrap: anywhere; padding: 1rem; border-radius: .65rem; background: hsl(var(--muted)); }
.cc-switch-guide :deep(a[data-guide-image]) { display: block; cursor: zoom-in; }
.guide-lightbox { width: min(94vw, 1400px); max-height: 90vh; border: 1px solid hsl(var(--border)); border-radius: .75rem; padding: 1rem; background: hsl(var(--background)); color: hsl(var(--foreground)); }
.guide-lightbox::backdrop { background: #000b; }
.guide-lightbox button { position: sticky; top: 0; display: block; margin-left: auto; padding: .7rem 1rem; border-radius: .5rem; background: hsl(var(--foreground)); color: hsl(var(--background)); cursor: pointer; }
.guide-lightbox img { width: 100%; max-height: none !important; }

.cc-switch-guide :deep(h1),
.cc-switch-guide :deep(h2) {
  text-wrap: balance;
}

.cc-switch-guide :deep(h1) {
  margin: 0;
  font-size: clamp(1.5rem, 3vw, 2.25rem);
  line-height: 1.2;
}

.cc-switch-guide :deep(h2) {
  margin: 2.25rem 0 0.75rem;
  padding-top: 0.25rem;
  font-size: 1.125rem;
  line-height: 1.6rem;
}

.cc-switch-guide :deep(p),
.cc-switch-guide :deep(li) {
  color: hsl(var(--muted-foreground));
  font-size: 0.875rem;
  line-height: 1.75;
}

.cc-switch-guide :deep(p) {
  margin: 0.75rem 0;
}

.cc-switch-guide :deep(ol),
.cc-switch-guide :deep(ul) {
  display: grid;
  gap: 0.35rem;
  margin: 0.75rem 0;
  padding-left: 1.4rem;
}

.cc-switch-guide :deep(ol) { list-style-type: decimal; }
.cc-switch-guide :deep(ul) { list-style-type: disc; }

.cc-switch-guide :deep(strong) {
  color: hsl(var(--foreground));
}

.cc-switch-guide :deep(a) {
  color: #2563eb;
  font-weight: 650;
  text-decoration-line: underline;
  text-decoration-thickness: 1px;
  text-underline-offset: 0.2rem;
}

.cc-switch-guide :deep(a:focus-visible) { outline: 3px solid #60a5fa; outline-offset: 4px; }
.cc-switch-guide :deep(a.guide-download) { display: inline-flex; align-items: center; padding: .85rem 1.2rem; border-radius: .7rem; background: #1d4ed8; color: #fff; text-decoration: none; box-shadow: 0 3px 10px #1d4ed826; }
.cc-switch-guide :deep(a.guide-download:hover) { background: #1e40af; }
.cc-switch-guide :deep(a[data-guide-image]) { text-decoration: none; }
:global(.dark) .cc-switch-guide :deep(a:not(.guide-download)) { color: #93c5fd; }
.guide-content :deep(h2) { border-left: 4px solid #2563eb; padding: .4rem 0 .4rem .8rem; margin-top: 2.5rem; }

.cc-switch-guide :deep(code) {
  border: 1px solid hsl(var(--border));
  border-radius: 0.25rem;
  padding: 0.1rem 0.35rem;
  color: hsl(var(--foreground));
  background: hsl(var(--muted));
  font-family: var(--font-mono, "Cascadia Code", monospace);
  font-size: 0.8125rem;
}

.cc-switch-guide :deep(blockquote) {
  margin: 1rem 0;
  border-left: 3px solid hsl(var(--border));
  padding: 0.25rem 0 0.25rem 1rem;
  background: hsl(var(--muted) / 0.45);
}

.cc-switch-guide :deep(blockquote p) {
  margin: 0.4rem 0;
}

.cc-switch-guide :deep(img) {
  display: block;
  width: auto;
  max-width: 100%;
  max-height: 42rem;
  margin: 1rem auto 1.5rem;
  border: 1px solid hsl(var(--border));
  border-radius: var(--radius);
  background: hsl(var(--muted));
  box-shadow: 0 8px 24px hsl(var(--shadow));
  object-fit: contain;
}

@media (max-width: 520px) {
  .cc-switch-guide :deep(h1) {
    font-size: 1.5rem;
  }
}
</style>
