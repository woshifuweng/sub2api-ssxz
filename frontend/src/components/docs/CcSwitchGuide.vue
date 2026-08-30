<template>
  <article id="cc-switch-guide" class="space-y-8" data-testid="cc-switch-guide">
    <header>
      <h2 class="text-2xl font-bold text-gray-900 dark:text-white">{{ t('docs.guide.title') }}</h2>
      <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">{{ t('docs.guide.summary') }}</p>
    </header>

    <section v-for="section in sections" :key="section.title" class="space-y-4">
      <h3 class="text-lg font-semibold text-gray-900 dark:text-white">{{ section.title }}</h3>
      <p class="text-sm leading-7 text-gray-600 dark:text-gray-300">{{ section.body }}</p>
      <a
        v-if="section.link"
        class="font-medium text-primary-600 hover:underline dark:text-primary-400"
        :href="section.link"
        target="_blank"
        rel="noopener noreferrer"
      >
        {{ t('docs.guide.officialLink') }}
      </a>
      <p
        v-if="section.warning"
        class="rounded-xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800 dark:border-amber-800/60 dark:bg-amber-950/30 dark:text-amber-200"
      >
        {{ t('docs.guide.warning') }}
      </p>
      <img
        v-for="image in section.images"
        :key="image.src"
        :src="image.src"
        :alt="image.alt"
        class="mx-auto max-h-[42rem] max-w-full rounded-xl border border-gray-200 bg-gray-100 object-contain shadow-sm dark:border-dark-700 dark:bg-dark-900"
        loading="lazy"
      />
    </section>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const assetBase = '/docs/cc-switch'

const sections = computed(() => [
  {
    title: t('docs.guide.prepareTitle'),
    body: t('docs.guide.prepareBody'),
    link: 'https://github.com/farion1231/cc-switch',
    warning: true,
    images: [
      { src: `${assetBase}/01-official-download-windows-macos.png`, alt: t('docs.guide.images.download') },
      { src: `${assetBase}/02-cc-switch-main-window.png`, alt: t('docs.guide.images.window') }
    ]
  },
  {
    title: t('docs.guide.step1Title'),
    body: t('docs.guide.step1Body'),
    images: [
      { src: `${assetBase}/03-ssxz-import-to-ccs-button.png`, alt: t('docs.guide.images.importButton') }
    ]
  },
  {
    title: t('docs.guide.step2Title'),
    body: t('docs.guide.step2Body'),
    images: [
      { src: `${assetBase}/04-browser-open-cc-switch-dialog.png`, alt: t('docs.guide.images.browserPrompt') },
      { src: `${assetBase}/05-cc-switch-import-confirmation.png`, alt: t('docs.guide.images.confirmation') }
    ]
  },
  {
    title: t('docs.guide.step3Title'),
    body: t('docs.guide.step3Body'),
    images: [
      { src: `${assetBase}/06-ssxz-selected-redacted.png`, alt: t('docs.guide.images.selected') },
      { src: `${assetBase}/07-claude-code-success-history.png`, alt: t('docs.guide.images.success') }
    ]
  },
  {
    title: t('docs.guide.finishTitle'),
    body: t('docs.guide.finishBody'),
    images: []
  }
])
</script>
