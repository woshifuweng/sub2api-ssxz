<template>
  <AppLayout>
    <AdminPageHeader
      title="站点设置"
      description="全局参数、注册策略与认证配置"
    />

    <div class="mx-auto max-w-4xl space-y-6 admin-b4-settings-scope">
      <!-- Loading State -->
      <div v-if="loading" class="flex items-center justify-center py-12">
        <div
          class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"
        ></div>
      </div>

      <!-- Settings Form -->
      <form v-else @submit.prevent="saveSettings" class="space-y-6" novalidate>
        <!-- Tab Navigation -->
        <div class="sticky top-0 z-10 overflow-x-auto settings-tabs-scroll">
          <nav class="settings-tabs">
            <LiquidButton
              v-for="tab in settingsTabs"
              :key="tab.key"
              type="button"
              :class="[
                'settings-tab',
                activeTab === tab.key && 'settings-tab-active',
              ]"
              @click="activeTab = tab.key"
              variant="plain"
              size="sm"
            >
              <span class="settings-tab-icon">
                <Icon :name="tab.icon" size="sm" />
              </span>
              <span>{{ t(`admin.settings.tabs.${tab.key}`) }}</span>
            </LiquidButton>
          </nav>
        </div>

        <!-- Tab: Security — Admin API Key -->
        <!-- OpenAI scheduler settings are intentionally visible on the default tab. -->
        <div data-testid="openai-scheduler-settings" class="card">
          <div class="space-y-5 p-6">
            <div
              v-if="!form.openai_advanced_scheduler_enabled"
              class="flex items-start justify-between gap-6"
            >
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.openaiExperimentalScheduler.lowRatePriorityTitle") }}
                </h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.openaiExperimentalScheduler.lowRatePriorityDescription") }}
                </p>
              </div>
              <Toggle
                data-testid="openai-low-rate-priority-toggle"
                :model-value="Boolean(form.openai_low_upstream_rate_priority_enabled)"
                @update:model-value="form.openai_low_upstream_rate_priority_enabled = $event"
              />
            </div>

            <div v-else class="space-y-4">
              <div class="flex items-start justify-between gap-6">
                <div>
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ t("admin.settings.openaiExperimentalScheduler.subscriptionPriorityTitle") }}
                  </h3>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.openaiExperimentalScheduler.subscriptionPriorityDescription") }}
                  </p>
                </div>
                <Toggle
                  data-testid="openai-subscription-priority-toggle"
                  :model-value="Boolean(form.openai_advanced_scheduler_subscription_priority_enabled)"
                  @update:model-value="form.openai_advanced_scheduler_subscription_priority_enabled = $event"
                />
              </div>
              <div class="flex items-start justify-between gap-6">
                <div>
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ t("admin.settings.openaiExperimentalScheduler.stickyWeightedTitle") }}
                  </h3>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.openaiExperimentalScheduler.stickyWeightedDescription") }}
                  </p>
                </div>
                <Toggle
                  data-testid="openai-sticky-weighted-toggle"
                  :model-value="Boolean(form.openai_advanced_scheduler_sticky_weighted_enabled)"
                  @update:model-value="form.openai_advanced_scheduler_sticky_weighted_enabled = $event"
                />
              </div>
            </div>

            <div class="flex items-start justify-between gap-6">
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.openaiExperimentalScheduler.oauthRateTitle") }}
                </h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{
                    t(
                      form.openai_advanced_scheduler_enabled
                        ? "admin.settings.openaiExperimentalScheduler.oauthRateWeightedDescription"
                        : "admin.settings.openaiExperimentalScheduler.oauthRatePriorityDescription",
                    )
                  }}
                </p>
              </div>
              <input
                v-if="form.openai_low_upstream_rate_priority_enabled || form.openai_advanced_scheduler_enabled"
                v-model.number="form.openai_oauth_scheduling_rate_multiplier"
                data-testid="openai-oauth-scheduling-rate-multiplier"
                type="number"
                min="0"
                step="0.01"
                class="input w-32"
              />
            </div>

            <div v-if="form.openai_advanced_scheduler_enabled" class="space-y-4">
              <div>
                <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.openaiExperimentalScheduler.weightsTitle") }}
                </h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.openaiExperimentalScheduler.weightsDescription") }}
                </p>
              </div>
              <div class="grid gap-4 sm:grid-cols-2">
                <div v-for="field in openAIAdvancedSchedulerWeightFields" :key="field.key">
                  <label class="mb-1 block text-sm text-gray-700 dark:text-gray-300">
                    {{ field.label }}
                  </label>
                  <input
                    :value="schedulerFieldValue(field.key)"
                    :placeholder="field.placeholder"
                    type="text"
                    class="input"
                    @input="setSchedulerFieldFromEvent(field.key, $event)"
                  />
                </div>
              </div>
            </div>

            <div class="flex items-start justify-between gap-6 border-t border-gray-100 pt-5 dark:border-dark-700">
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.openaiExperimentalScheduler.title") }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.openaiExperimentalScheduler.description") }}
                </p>
              </div>
              <Toggle
                data-testid="openai-advanced-scheduler-toggle"
                :model-value="Boolean(form.openai_advanced_scheduler_enabled)"
                @update:model-value="form.openai_advanced_scheduler_enabled = $event"
              />
            </div>
          </div>
        </div>

        <div v-show="activeTab === 'security'" class="space-y-6">
          <!-- Admin API Key Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.adminApiKey.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.adminApiKey.description") }}
              </p>
            </div>
            <div class="space-y-4 p-6">
              <!-- Security Warning -->
              <div
                class="rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
              >
                <div class="flex items-start">
                  <Icon
                    name="exclamationTriangle"
                    size="md"
                    class="mt-0.5 flex-shrink-0 text-amber-500"
                  />
                  <p class="ml-3 text-sm text-amber-700 dark:text-amber-300">
                    {{ t("admin.settings.adminApiKey.securityWarning") }}
                  </p>
                </div>
              </div>

              <!-- Loading State -->
              <div
                v-if="adminApiKeyLoading"
                class="flex items-center gap-2 text-gray-500"
              >
                <div
                  class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                ></div>
                {{ t("common.loading") }}
              </div>

              <!-- No Key Configured -->
              <div
                v-else-if="!adminApiKeyExists"
                class="flex items-center justify-between"
              >
                <span class="text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.adminApiKey.notConfigured") }}
                </span>
                <LiquidButton
                  type="button"
                  @click="createAdminApiKey"
                  :disabled="adminApiKeyOperating"
                  size="sm"
                >
                  <svg
                    v-if="adminApiKeyOperating"
                    class="h-4 w-4 animate-spin"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      class="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      stroke-width="4"
                    ></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  {{
                    adminApiKeyOperating
                      ? t("admin.settings.adminApiKey.creating")
                      : t("admin.settings.adminApiKey.create")
                  }}
                </LiquidButton>
              </div>

              <!-- Key Exists -->
              <div v-else class="space-y-4">
                <div class="flex items-center justify-between">
                  <div>
                    <label
                      class="mb-1 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.adminApiKey.currentKey") }}
                    </label>
                    <code
                      class="rounded bg-gray-100 px-2 py-1 font-mono text-sm text-gray-900 dark:bg-dark-700 dark:text-gray-100"
                    >
                      {{ adminApiKeyMasked }}
                    </code>
                  </div>
                  <div class="flex gap-2">
                    <LiquidButton
                      type="button"
                      @click="regenerateAdminApiKey"
                      :disabled="adminApiKeyOperating"
                      variant="outline"
                      size="sm"
                    >
                      {{
                        adminApiKeyOperating
                          ? t("admin.settings.adminApiKey.regenerating")
                          : t("admin.settings.adminApiKey.regenerate")
                      }}
                    </LiquidButton>
                    <LiquidButton
                      type="button"
                      @click="deleteAdminApiKey"
                      :disabled="adminApiKeyOperating"
                      class="text-red-600 hover:text-red-700 dark:text-red-400"
                      variant="outline"
                      size="sm"
                    >
                      {{ t("admin.settings.adminApiKey.delete") }}
                    </LiquidButton>
                  </div>
                </div>

                <!-- Newly Generated Key Display -->
                <div
                  v-if="newAdminApiKey"
                  class="space-y-3 rounded-lg border border-green-200 bg-green-50 p-4 dark:border-green-800 dark:bg-green-900/20"
                >
                  <p
                    class="text-sm font-medium text-green-700 dark:text-green-300"
                  >
                    {{ t("admin.settings.adminApiKey.keyWarning") }}
                  </p>
                  <div class="flex items-center gap-2">
                    <code
                      class="flex-1 select-all break-all rounded border border-green-300 bg-white px-3 py-2 font-mono text-sm dark:border-green-700 dark:bg-dark-800"
                    >
                      {{ newAdminApiKey }}
                    </code>
                    <LiquidButton
                      type="button"
                      @click="copyNewKey"
                      class="flex-shrink-0"
                      size="sm"
                    >
                      {{ t("admin.settings.adminApiKey.copyKey") }}
                    </LiquidButton>
                  </div>
                  <p class="text-xs text-green-600 dark:text-green-400">
                    {{ t("admin.settings.adminApiKey.usage") }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
        <!-- /Tab: Security — Admin API Key -->

        <!-- Tab: Gateway -->
        <div v-show="activeTab === 'gateway'" class="space-y-6">
          <!-- Overload Cooldown (529) Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.overloadCooldown.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.overloadCooldown.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div
                v-if="overloadCooldownLoading"
                class="flex items-center gap-2 text-gray-500"
              >
                <div
                  class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                ></div>
                {{ t("common.loading") }}
              </div>

              <template v-else>
                <div class="flex items-center justify-between">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.overloadCooldown.enabled")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.overloadCooldown.enabledHint") }}
                    </p>
                  </div>
                  <Toggle v-model="overloadCooldownForm.enabled" />
                </div>

                <div
                  v-if="overloadCooldownForm.enabled"
                  class="space-y-4 border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.overloadCooldown.cooldownMinutes") }}
                    </label>
                    <input
                      v-model.number="overloadCooldownForm.cooldown_minutes"
                      type="number"
                      min="1"
                      max="120"
                      class="input w-32"
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{
                        t("admin.settings.overloadCooldown.cooldownMinutesHint")
                      }}
                    </p>
                  </div>
                </div>

                <div
                  class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <LiquidButton
                    type="button"
                    @click="saveOverloadCooldownSettings"
                    :disabled="overloadCooldownSaving"
                    size="sm"
                  >
                    <svg
                      v-if="overloadCooldownSaving"
                      class="h-4 w-4 animate-spin"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        class="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        stroke-width="4"
                      ></circle>
                      <path
                        class="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    {{
                      overloadCooldownSaving
                        ? t("common.saving")
                        : t("common.save")
                    }}
                  </LiquidButton>
                </div>
              </template>
            </div>
          </div>

          <!-- Panel API Rate Limit Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.panelRateLimit.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.panelRateLimit.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div
                v-if="panelRateLimitLoading"
                class="flex items-center gap-2 text-gray-500"
              >
                <div
                  class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                ></div>
                {{ t("common.loading") }}
              </div>
              <template v-else>
                <div class="flex items-center justify-between">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">
                      {{ t("admin.settings.panelRateLimit.enabled") }}
                    </label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.panelRateLimit.enabledHint") }}
                    </p>
                  </div>
                  <Toggle v-model="panelRateLimitForm.enabled" />
                </div>

                <div
                  v-if="panelRateLimitForm.enabled"
                  class="space-y-5 border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div class="grid grid-cols-1 gap-5 sm:grid-cols-3">
                    <label
                      class="block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.panelRateLimit.userRpm") }}
                      <input
                        v-model.number="panelRateLimitForm.user_rpm"
                        type="number"
                        min="0"
                        max="100000"
                        class="input mt-2 w-32"
                      />
                      <span
                        class="ml-2 text-xs text-gray-500 dark:text-gray-400"
                        >{{
                          t("admin.settings.panelRateLimit.perMinute")
                        }}</span
                      >
                    </label>
                    <label
                      class="block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.panelRateLimit.heavyRpm") }}
                      <input
                        v-model.number="panelRateLimitForm.heavy_rpm"
                        type="number"
                        min="0"
                        max="100000"
                        class="input mt-2 w-32"
                      />
                      <span
                        class="ml-2 text-xs text-gray-500 dark:text-gray-400"
                        >{{
                          t("admin.settings.panelRateLimit.perMinute")
                        }}</span
                      >
                    </label>
                    <label
                      class="block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.panelRateLimit.publicIpRpm") }}
                      <input
                        v-model.number="panelRateLimitForm.public_ip_rpm"
                        type="number"
                        min="0"
                        max="100000"
                        class="input mt-2 w-32"
                      />
                      <span
                        class="ml-2 text-xs text-gray-500 dark:text-gray-400"
                        >{{
                          t("admin.settings.panelRateLimit.perMinute")
                        }}</span
                      >
                    </label>
                  </div>

                  <div
                    class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                  >
                    <div>
                      <label
                        class="font-medium text-gray-900 dark:text-white"
                        >{{
                          t("admin.settings.panelRateLimit.exemptAdmin")
                        }}</label
                      >
                      <p class="text-sm text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.panelRateLimit.exemptAdminHint") }}
                      </p>
                    </div>
                    <Toggle v-model="panelRateLimitForm.exempt_admin" />
                  </div>
                </div>

                <div
                  class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <LiquidButton
                    type="button"
                    @click="savePanelRateLimitSettings"
                    :disabled="panelRateLimitSaving"
                    size="sm"
                  >
                    {{
                      panelRateLimitSaving
                        ? t("common.saving")
                        : t("common.save")
                    }}
                  </LiquidButton>
                </div>
              </template>
            </div>
          </div>

          <!-- Stream Timeout Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.streamTimeout.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.streamTimeout.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <!-- Loading State -->
              <div
                v-if="streamTimeoutLoading"
                class="flex items-center gap-2 text-gray-500"
              >
                <div
                  class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                ></div>
                {{ t("common.loading") }}
              </div>

              <template v-else>
                <!-- Enable Stream Timeout -->
                <div class="flex items-center justify-between">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.streamTimeout.enabled")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.streamTimeout.enabledHint") }}
                    </p>
                  </div>
                  <Toggle v-model="streamTimeoutForm.enabled" />
                </div>

                <!-- Settings - Only show when enabled -->
                <div
                  v-if="streamTimeoutForm.enabled"
                  class="space-y-4 border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <!-- Action -->
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.streamTimeout.action") }}
                    </label>
                    <select
                      v-model="streamTimeoutForm.action"
                      class="input w-64"
                    >
                      <option value="temp_unsched">
                        {{
                          t("admin.settings.streamTimeout.actionTempUnsched")
                        }}
                      </option>
                      <option value="error">
                        {{ t("admin.settings.streamTimeout.actionError") }}
                      </option>
                      <option value="none">
                        {{ t("admin.settings.streamTimeout.actionNone") }}
                      </option>
                    </select>
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.streamTimeout.actionHint") }}
                    </p>
                  </div>

                  <!-- Temp Unsched Minutes (only show when action is temp_unsched) -->
                  <div v-if="streamTimeoutForm.action === 'temp_unsched'">
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.streamTimeout.tempUnschedMinutes") }}
                    </label>
                    <input
                      v-model.number="streamTimeoutForm.temp_unsched_minutes"
                      type="number"
                      min="1"
                      max="60"
                      class="input w-32"
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{
                        t("admin.settings.streamTimeout.tempUnschedMinutesHint")
                      }}
                    </p>
                  </div>

                  <!-- Threshold Count -->
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.streamTimeout.thresholdCount") }}
                    </label>
                    <input
                      v-model.number="streamTimeoutForm.threshold_count"
                      type="number"
                      min="1"
                      max="10"
                      class="input w-32"
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.streamTimeout.thresholdCountHint") }}
                    </p>
                  </div>

                  <!-- Threshold Window Minutes -->
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{
                        t("admin.settings.streamTimeout.thresholdWindowMinutes")
                      }}
                    </label>
                    <input
                      v-model.number="
                        streamTimeoutForm.threshold_window_minutes
                      "
                      type="number"
                      min="1"
                      max="60"
                      class="input w-32"
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{
                        t(
                          "admin.settings.streamTimeout.thresholdWindowMinutesHint",
                        )
                      }}
                    </p>
                  </div>
                </div>

                <!-- Save Button -->
                <div
                  class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <LiquidButton
                    type="button"
                    @click="saveStreamTimeoutSettings"
                    :disabled="streamTimeoutSaving"
                    size="sm"
                  >
                    <svg
                      v-if="streamTimeoutSaving"
                      class="h-4 w-4 animate-spin"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        class="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        stroke-width="4"
                      ></circle>
                      <path
                        class="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    {{
                      streamTimeoutSaving
                        ? t("common.saving")
                        : t("common.save")
                    }}
                  </LiquidButton>
                </div>
              </template>
            </div>
          </div>

          <!-- Request Rectifier Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.rectifier.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.rectifier.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <!-- Loading State -->
              <div
                v-if="rectifierLoading"
                class="flex items-center gap-2 text-gray-500"
              >
                <div
                  class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                ></div>
                {{ t("common.loading") }}
              </div>

              <template v-else>
                <!-- Master Toggle -->
                <div class="flex items-center justify-between">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.rectifier.enabled")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.rectifier.enabledHint") }}
                    </p>
                  </div>
                  <Toggle v-model="rectifierForm.enabled" />
                </div>

                <!-- Sub-toggles (only show when master is enabled) -->
                <div
                  v-if="rectifierForm.enabled"
                  class="space-y-4 border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <!-- Thinking Signature Rectifier -->
                  <div class="flex items-center justify-between">
                    <div>
                      <label
                        class="text-sm font-medium text-gray-700 dark:text-gray-300"
                        >{{
                          t("admin.settings.rectifier.thinkingSignature")
                        }}</label
                      >
                      <p class="text-xs text-gray-500 dark:text-gray-400">
                        {{
                          t("admin.settings.rectifier.thinkingSignatureHint")
                        }}
                      </p>
                    </div>
                    <Toggle
                      v-model="rectifierForm.thinking_signature_enabled"
                    />
                  </div>

                  <!-- Thinking Budget Rectifier -->
                  <div class="flex items-center justify-between">
                    <div>
                      <label
                        class="text-sm font-medium text-gray-700 dark:text-gray-300"
                        >{{
                          t("admin.settings.rectifier.thinkingBudget")
                        }}</label
                      >
                      <p class="text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.rectifier.thinkingBudgetHint") }}
                      </p>
                    </div>
                    <Toggle v-model="rectifierForm.thinking_budget_enabled" />
                  </div>
                </div>

                <!-- Save Button -->
                <div
                  class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <LiquidButton
                    type="button"
                    @click="saveRectifierSettings"
                    :disabled="rectifierSaving"
                    size="sm"
                  >
                    <svg
                      v-if="rectifierSaving"
                      class="h-4 w-4 animate-spin"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        class="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        stroke-width="4"
                      ></circle>
                      <path
                        class="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    {{
                      rectifierSaving ? t("common.saving") : t("common.save")
                    }}
                  </LiquidButton>
                </div>
              </template>
            </div>
          </div>
          <!-- Beta Policy Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.betaPolicy.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.betaPolicy.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <!-- Loading State -->
              <div
                v-if="betaPolicyLoading"
                class="flex items-center gap-2 text-gray-500"
              >
                <div
                  class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                ></div>
                {{ t("common.loading") }}
              </div>

              <template v-else>
                <!-- Rule Cards -->
                <div
                  v-for="rule in betaPolicyForm.rules"
                  :key="rule.beta_token"
                  class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
                >
                  <div class="mb-3 flex items-center gap-2">
                    <span
                      class="text-sm font-medium text-gray-900 dark:text-white"
                    >
                      {{ getBetaDisplayName(rule.beta_token) }}
                    </span>
                    <span
                      class="rounded bg-gray-100 px-2 py-0.5 text-xs text-gray-500 dark:bg-dark-700 dark:text-gray-400"
                    >
                      {{ rule.beta_token }}
                    </span>
                  </div>

                  <div class="grid grid-cols-2 gap-4">
                    <!-- Action -->
                    <div>
                      <label
                        class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                      >
                        {{ t("admin.settings.betaPolicy.action") }}
                      </label>
                      <Select
                        :modelValue="rule.action"
                        @update:modelValue="rule.action = $event as any"
                        :options="betaPolicyActionOptions"
                      />
                    </div>

                    <!-- Scope -->
                    <div>
                      <label
                        class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                      >
                        {{ t("admin.settings.betaPolicy.scope") }}
                      </label>
                      <Select
                        :modelValue="rule.scope"
                        @update:modelValue="rule.scope = $event as any"
                        :options="betaPolicyScopeOptions"
                      />
                    </div>
                  </div>

                  <!-- Error Message (only when action=block) -->
                  <div v-if="rule.action === 'block'" class="mt-3">
                    <label
                      class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                    >
                      {{ t("admin.settings.betaPolicy.errorMessage") }}
                    </label>
                    <input
                      v-model="rule.error_message"
                      type="text"
                      class="input"
                      :placeholder="
                        t('admin.settings.betaPolicy.errorMessagePlaceholder')
                      "
                    />
                    <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                      {{ t("admin.settings.betaPolicy.errorMessageHint") }}
                    </p>
                  </div>
                </div>

                <!-- Save Button -->
                <div
                  class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <LiquidButton
                    type="button"
                    @click="saveBetaPolicySettings"
                    :disabled="betaPolicySaving"
                    size="sm"
                  >
                    <svg
                      v-if="betaPolicySaving"
                      class="h-4 w-4 animate-spin"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        class="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        stroke-width="4"
                      ></circle>
                      <path
                        class="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    {{
                      betaPolicySaving ? t("common.saving") : t("common.save")
                    }}
                  </LiquidButton>
                </div>
              </template>
            </div>
          </div>

          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.openaiFastPolicy.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.openaiFastPolicy.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div
                v-if="openaiFastPolicyForm.rules.length === 0"
                class="rounded-lg border border-dashed border-gray-200 p-6 text-center text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
              >
                {{ t("admin.settings.openaiFastPolicy.empty") }}
              </div>

              <!-- Rule Cards -->
              <div
                v-for="(rule, ruleIndex) in openaiFastPolicyForm.rules"
                :key="ruleIndex"
                class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
              >
                <div class="mb-3 flex items-center justify-between">
                  <span
                    class="text-sm font-medium text-gray-900 dark:text-white"
                  >
                    {{
                      t("admin.settings.openaiFastPolicy.ruleHeader", {
                        index: ruleIndex + 1,
                      })
                    }}
                  </span>
                  <button
                    type="button"
                    @click="removeOpenAIFastPolicyRule(ruleIndex)"
                    class="rounded p-1 text-red-400 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                    :title="t('admin.settings.openaiFastPolicy.removeRule')"
                  >
                    <svg
                      class="h-4 w-4"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M6 18L18 6M6 6l12 12"
                      />
                    </svg>
                  </button>
                </div>

                <div
                  class="mb-4 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-gray-500 dark:text-gray-400"
                  :data-testid="`openai-fast-policy-summary-${ruleIndex}`"
                >
                  <span class="font-medium text-gray-700 dark:text-gray-300">
                    {{
                      t(
                        hasOpenAIFastPolicyTargetModels(rule)
                          ? "admin.settings.openaiFastPolicy.summaryTargetModels"
                          : "admin.settings.openaiFastPolicy.summaryAllModels",
                      )
                    }}
                  </span>
                  <span aria-hidden="true">→</span>
                  <span
                    class="inline-flex items-center rounded bg-primary-50 px-2 py-0.5 font-medium text-primary-700 dark:bg-primary-900/30 dark:text-primary-300"
                  >
                    {{ openaiFastPolicyActionSummary(rule.action) }}
                  </span>
                  <template v-if="hasOpenAIFastPolicyTargetModels(rule)">
                    <span aria-hidden="true">·</span>
                    <span class="font-medium text-gray-700 dark:text-gray-300">
                      {{
                        t(
                          "admin.settings.openaiFastPolicy.summaryOtherModels",
                        )
                      }}
                    </span>
                    <span aria-hidden="true">→</span>
                    <span
                      class="inline-flex items-center rounded bg-gray-100 px-2 py-0.5 font-medium text-gray-700 dark:bg-dark-600 dark:text-gray-300"
                    >
                      {{
                        openaiFastPolicyActionSummary(
                          rule.fallback_action || "pass",
                        )
                      }}
                    </span>
                  </template>
                </div>

                <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
                  <!-- Service Tier -->
                  <div>
                    <label
                      class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                    >
                      {{ t("admin.settings.openaiFastPolicy.serviceTier") }}
                    </label>
                    <Select
                      :modelValue="rule.service_tier"
                      @update:modelValue="
                        rule.service_tier = $event as
                          | 'all'
                          | 'priority'
                          | 'flex'
                      "
                      :options="openaiFastPolicyTierOptions"
                    />
                  </div>

                  <!-- Action -->
                  <div>
                    <label
                      class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                    >
                      {{ t("admin.settings.openaiFastPolicy.action") }}
                    </label>
                    <Select
                      :modelValue="rule.action"
                      @update:modelValue="
                        rule.action = $event as
                          | 'pass'
                          | 'filter'
                          | 'block'
                          | 'force_priority'
                      "
                      :options="openaiFastPolicyActionOptions"
                    />
                  </div>

                  <!-- Scope -->
                  <div>
                    <label
                      class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                    >
                      {{ t("admin.settings.openaiFastPolicy.scope") }}
                    </label>
                    <Select
                      :modelValue="rule.scope"
                      @update:modelValue="
                        rule.scope = $event as
                          | 'all'
                          | 'oauth'
                          | 'apikey'
                          | 'bedrock'
                      "
                      :options="openaiFastPolicyScopeOptions"
                    />
                  </div>
                </div>

                <!-- User Scope -->
                <div class="mt-3">
                  <label
                    class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                  >
                    {{ t("admin.settings.openaiFastPolicy.userIds") }}
                  </label>
                  <p class="mb-2 text-xs text-gray-400 dark:text-gray-500">
                    {{ t("admin.settings.openaiFastPolicy.userIdsHint") }}
                  </p>
                  <OpenAIFastPolicyUserSelector
                    :model-value="rule.user_ids || []"
                    @update:model-value="rule.user_ids = $event"
                  />
                </div>

                <!-- Error Message (only when action=block) -->
                <div v-if="rule.action === 'block'" class="mt-3">
                  <label
                    class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                  >
                    {{ t("admin.settings.openaiFastPolicy.errorMessage") }}
                  </label>
                  <input
                    v-model="rule.error_message"
                    type="text"
                    class="input"
                    :placeholder="
                      t(
                        'admin.settings.openaiFastPolicy.errorMessagePlaceholder',
                      )
                    "
                  />
                  <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                    {{ t("admin.settings.openaiFastPolicy.errorMessageHint") }}
                  </p>
                </div>

                <!-- Target Models -->
                <div
                  class="mt-3"
                  role="group"
                  :aria-labelledby="`openai-fast-policy-models-label-${ruleIndex}`"
                  :aria-describedby="`openai-fast-policy-models-hint-${ruleIndex}`"
                >
                  <label
                    :id="`openai-fast-policy-models-label-${ruleIndex}`"
                    class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                  >
                    {{ t("admin.settings.openaiFastPolicy.modelWhitelist") }}
                  </label>
                  <p
                    :id="`openai-fast-policy-models-hint-${ruleIndex}`"
                    class="mb-2 text-xs text-gray-400 dark:text-gray-500"
                  >
                    {{
                      t("admin.settings.openaiFastPolicy.modelWhitelistHint")
                    }}
                  </p>
                  <div
                    v-for="(_, patternIdx) in rule.model_whitelist || []"
                    :key="patternIdx"
                    class="mb-1.5 flex items-center gap-2"
                  >
                    <input
                      v-model="rule.model_whitelist![patternIdx]"
                      type="text"
                      class="input input-sm flex-1"
                      :placeholder="
                        t(
                          'admin.settings.openaiFastPolicy.modelPatternPlaceholder',
                        )
                      "
                    />
                    <button
                      type="button"
                      @click="
                        removeOpenAIFastPolicyModelPattern(rule, patternIdx)
                      "
                      class="shrink-0 rounded p-1 text-red-400 transition-colors hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                    >
                      <svg
                        class="h-4 w-4"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          d="M6 18L18 6M6 6l12 12"
                        />
                      </svg>
                    </button>
                  </div>
                  <button
                    type="button"
                    @click="addOpenAIFastPolicyModelPattern(rule)"
                    class="mb-2 inline-flex items-center gap-1 text-xs text-primary-600 transition-colors hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300"
                  >
                    <svg
                      class="h-3.5 w-3.5"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <path
                        stroke-linecap="round"
                        stroke-linejoin="round"
                        d="M12 4v16m8-8H4"
                      />
                    </svg>
                    {{ t("admin.settings.openaiFastPolicy.addModelPattern") }}
                  </button>
                </div>

                <!-- Other Models Action (only when target models are non-empty) -->
                <div
                  v-if="hasOpenAIFastPolicyTargetModels(rule)"
                  class="mt-3"
                >
                  <label
                    class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                  >
                    {{ t("admin.settings.openaiFastPolicy.fallbackAction") }}
                  </label>
                  <Select
                    :modelValue="rule.fallback_action || 'pass'"
                    @update:modelValue="
                      rule.fallback_action = $event as
                        | 'pass'
                        | 'filter'
                        | 'block'
                        | 'force_priority'
                    "
                    :options="openaiFastPolicyActionOptions"
                  />
                  <p class="mt-1 text-xs text-gray-400 dark:text-gray-500">
                    {{
                      t("admin.settings.openaiFastPolicy.fallbackActionHint")
                    }}
                  </p>
                  <div v-if="rule.fallback_action === 'block'" class="mt-2">
                    <input
                      v-model="rule.fallback_error_message"
                      type="text"
                      class="input"
                      :placeholder="
                        t(
                          'admin.settings.openaiFastPolicy.fallbackErrorMessagePlaceholder',
                        )
                      "
                    />
                  </div>
                </div>
              </div>

              <div>
                <LiquidButton
                  type="button"
                  @click="addOpenAIFastPolicyRule"
                  variant="outline"
                  size="sm"
                >
                  {{ t("admin.settings.openaiFastPolicy.addRule") }}
                </LiquidButton>
                <p class="mt-2 text-xs text-gray-400 dark:text-gray-500">
                  {{ t("admin.settings.openaiFastPolicy.saveHint") }}
                </p>
              </div>
            </div>
          </div>

          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <div class="flex items-start justify-between gap-4">
                <div>
                  <h2
                    class="text-lg font-semibold text-gray-900 dark:text-white"
                  >
                    {{ t("admin.settings.tlsFingerprint.title") }}
                  </h2>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.tlsFingerprint.description") }}
                  </p>
                </div>
                <LiquidButton
                  type="button"
                  @click="openCreateTLSFingerprintModal"
                  variant="outline"
                  size="sm"
                >
                  {{ t("admin.settings.tlsFingerprint.newProfile") }}
                </LiquidButton>
              </div>
            </div>
            <div class="space-y-5 p-6">
              <div
                v-if="tlsFingerprintLoading"
                class="flex items-center gap-2 text-gray-500"
              >
                <div
                  class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                ></div>
                {{ t("common.loading") }}
              </div>

              <template v-else>
                <div class="flex items-center justify-between">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">
                      {{ t("admin.settings.tlsFingerprint.enabled") }}
                    </label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.tlsFingerprint.enabledHint") }}
                    </p>
                  </div>
                  <Toggle v-model="tlsFingerprintGlobalEnabled" />
                </div>

                <div
                  class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <LiquidButton
                    type="button"
                    :disabled="tlsFingerprintSaving"
                    @click="saveTLSFingerprintGlobalSettings"
                    size="sm"
                  >
                    {{
                      tlsFingerprintSaving
                        ? t("common.saving")
                        : t("common.save")
                    }}
                  </LiquidButton>
                </div>

                <div
                  v-if="tlsFingerprintProfiles.length === 0"
                  class="rounded border border-dashed border-gray-300 px-4 py-3 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
                >
                  {{ t("admin.settings.tlsFingerprint.empty") }}
                </div>

                <div
                  v-else
                  class="overflow-x-auto rounded-xl border border-gray-200 dark:border-dark-600"
                >
                  <table
                    class="min-w-full divide-y divide-gray-200 dark:divide-dark-600"
                  >
                    <thead class="bg-gray-50 dark:bg-dark-800/60">
                      <tr
                        class="text-left text-xs uppercase tracking-wide text-gray-500 dark:text-gray-400"
                      >
                        <th class="px-4 py-3">
                          {{
                            t("admin.settings.tlsFingerprint.columns.profile")
                          }}
                        </th>
                        <th class="px-4 py-3">
                          {{
                            t("admin.settings.tlsFingerprint.columns.status")
                          }}
                        </th>
                        <th class="px-4 py-3">
                          {{
                            t("admin.settings.tlsFingerprint.columns.grease")
                          }}
                        </th>
                        <th class="px-4 py-3">
                          {{
                            t(
                              "admin.settings.tlsFingerprint.columns.cipherSuites",
                            )
                          }}
                        </th>
                        <th class="px-4 py-3">
                          {{
                            t("admin.settings.tlsFingerprint.columns.curves")
                          }}
                        </th>
                        <th class="px-4 py-3">
                          {{
                            t(
                              "admin.settings.tlsFingerprint.columns.pointFormats",
                            )
                          }}
                        </th>
                        <th class="px-4 py-3">
                          {{
                            t("admin.settings.tlsFingerprint.columns.updatedAt")
                          }}
                        </th>
                        <th class="px-4 py-3">
                          {{
                            t("admin.settings.tlsFingerprint.columns.actions")
                          }}
                        </th>
                      </tr>
                    </thead>
                    <tbody
                      class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-900"
                    >
                      <tr
                        v-for="profile in tlsFingerprintProfiles"
                        :key="profile.profile_id"
                      >
                        <td class="px-4 py-3 align-top">
                          <div
                            class="font-medium text-gray-900 dark:text-white"
                          >
                            {{ profile.name }}
                          </div>
                          <div
                            class="mt-1 font-mono text-xs text-gray-500 dark:text-gray-400"
                          >
                            {{ profile.profile_id }}
                          </div>
                        </td>
                        <td class="px-4 py-3 align-top">
                          <span
                            class="rounded-full px-2.5 py-1 text-xs font-medium"
                            :class="
                              profile.enabled
                                ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-500/15 dark:text-emerald-300'
                                : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
                            "
                          >
                            {{
                              profile.enabled
                                ? t("common.enabled")
                                : t("common.disabled")
                            }}
                          </span>
                        </td>
                        <td
                          class="px-4 py-3 align-top text-sm text-gray-700 dark:text-gray-300"
                        >
                          {{
                            profile.enable_grease
                              ? t("common.enabled")
                              : t("common.disabled")
                          }}
                        </td>
                        <td
                          class="px-4 py-3 align-top text-xs text-gray-600 dark:text-gray-300"
                        >
                          {{
                            profile.cipher_suites.length
                              ? profile.cipher_suites.join(", ")
                              : t("admin.settings.tlsFingerprint.defaultValues")
                          }}
                        </td>
                        <td
                          class="px-4 py-3 align-top text-xs text-gray-600 dark:text-gray-300"
                        >
                          {{
                            profile.curves.length
                              ? profile.curves.join(", ")
                              : t("admin.settings.tlsFingerprint.defaultValues")
                          }}
                        </td>
                        <td
                          class="px-4 py-3 align-top text-xs text-gray-600 dark:text-gray-300"
                        >
                          {{
                            profile.point_formats.length
                              ? profile.point_formats.join(", ")
                              : t("admin.settings.tlsFingerprint.defaultValues")
                          }}
                        </td>
                        <td
                          class="px-4 py-3 align-top text-xs text-gray-500 dark:text-gray-400"
                        >
                          {{ profile.updated_at || "-" }}
                        </td>
                        <td class="px-4 py-3 align-top">
                          <div class="flex flex-wrap gap-2">
                            <LiquidButton
                              type="button"
                              @click="openEditTLSFingerprintModal(profile)"
                              variant="outline"
                              size="sm"
                            >
                              {{ t("common.edit") }}
                            </LiquidButton>
                            <LiquidButton
                              type="button"
                              :disabled="tlsFingerprintSaving"
                              @click="toggleTLSFingerprintProfile(profile)"
                              variant="outline"
                              size="sm"
                            >
                              {{
                                profile.enabled
                                  ? t(
                                      "admin.settings.tlsFingerprint.disableProfile",
                                    )
                                  : t(
                                      "admin.settings.tlsFingerprint.enableProfile",
                                    )
                              }}
                            </LiquidButton>
                            <LiquidButton
                              type="button"
                              :disabled="tlsFingerprintSaving"
                              @click="deleteTLSFingerprintProfile(profile)"
                              variant="destructive"
                              size="sm"
                            >
                              {{ t("common.delete") }}
                            </LiquidButton>
                          </div>
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </template>
            </div>
          </div>
        </div>
        <!-- /Tab: Gateway -->

        <div
          v-if="showTLSFingerprintModal"
          class="fixed inset-0 z-50 flex items-center justify-center p-4"
          @mousedown.self="closeTLSFingerprintModal"
        >
          <div
            class="w-full max-w-2xl rounded-2xl bg-white shadow-2xl dark:bg-dark-800"
          >
            <div
              class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <div>
                <h3 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{
                    editingTLSFingerprintProfileID
                      ? t("admin.settings.tlsFingerprint.editTitle")
                      : t("admin.settings.tlsFingerprint.createTitle")
                  }}
                </h3>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.tlsFingerprint.modalHint") }}
                </p>
              </div>
              <LiquidButton
                type="button"
                :aria-label="t('common.close')"
                class="text-gray-400 hover:text-gray-600 dark:hover:text-gray-200"
                @click="closeTLSFingerprintModal"
                variant="plain"
                size="icon"
              >
                <Icon name="x" size="sm" />
              </LiquidButton>
            </div>
            <div class="space-y-4 px-6 py-5">
              <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.tlsFingerprint.profileID") }}
                  </label>
                  <input
                    v-model="tlsFingerprintForm.profile_id"
                    type="text"
                    class="input"
                    :disabled="Boolean(editingTLSFingerprintProfileID)"
                  />
                </div>
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.tlsFingerprint.profileName") }}
                  </label>
                  <input
                    v-model="tlsFingerprintForm.name"
                    type="text"
                    class="input"
                  />
                </div>
              </div>

              <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
                <div
                  class="flex items-center justify-between rounded-xl border border-gray-200 px-4 py-3 dark:border-dark-600"
                >
                  <div>
                    <div
                      class="text-sm font-medium text-gray-900 dark:text-white"
                    >
                      {{ t("admin.settings.tlsFingerprint.profileEnabled") }}
                    </div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">
                      {{
                        t("admin.settings.tlsFingerprint.profileEnabledHint")
                      }}
                    </div>
                  </div>
                  <Toggle v-model="tlsFingerprintForm.enabled" />
                </div>
                <div
                  class="flex items-center justify-between rounded-xl border border-gray-200 px-4 py-3 dark:border-dark-600"
                >
                  <div>
                    <div
                      class="text-sm font-medium text-gray-900 dark:text-white"
                    >
                      {{ t("admin.settings.tlsFingerprint.enableGrease") }}
                    </div>
                    <div class="text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.tlsFingerprint.enableGreaseHint") }}
                    </div>
                  </div>
                  <Toggle v-model="tlsFingerprintForm.enable_grease" />
                </div>
              </div>

              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.tlsFingerprint.cipherSuites") }}
                </label>
                <textarea
                  v-model="tlsFingerprintForm.cipher_suites_text"
                  rows="3"
                  class="input min-h-[84px]"
                  :placeholder="
                    t('admin.settings.tlsFingerprint.numberListPlaceholder')
                  "
                ></textarea>
                <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.tlsFingerprint.defaultHint") }}
                </p>
              </div>

              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.tlsFingerprint.curves") }}
                </label>
                <textarea
                  v-model="tlsFingerprintForm.curves_text"
                  rows="3"
                  class="input min-h-[84px]"
                  :placeholder="
                    t('admin.settings.tlsFingerprint.numberListPlaceholder')
                  "
                ></textarea>
              </div>

              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.tlsFingerprint.pointFormats") }}
                </label>
                <textarea
                  v-model="tlsFingerprintForm.point_formats_text"
                  rows="2"
                  class="input min-h-[64px]"
                  :placeholder="
                    t('admin.settings.tlsFingerprint.numberListPlaceholder')
                  "
                ></textarea>
              </div>
            </div>
            <div
              class="flex justify-end gap-3 border-t border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <LiquidButton
                type="button"
                @click="closeTLSFingerprintModal"
                variant="outline"
                size="sm"
              >
                {{ t("common.cancel") }}
              </LiquidButton>
              <LiquidButton
                type="button"
                :disabled="tlsFingerprintSaving"
                @click="submitTLSFingerprintProfile"
                size="default"
              >
                {{
                  tlsFingerprintSaving ? t("common.saving") : t("common.save")
                }}
              </LiquidButton>
            </div>
          </div>
        </div>

        <!-- Tab: Security — Registration, Turnstile, LinuxDo -->
        <div v-show="activeTab === 'security'" class="space-y-6">
          <!-- Registration Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.registration.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.registration.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <!-- Enable Registration -->
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.registration.enableRegistration")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{
                      t("admin.settings.registration.enableRegistrationHint")
                    }}
                  </p>
                </div>
                <Toggle v-model="form.registration_enabled" />
              </div>

              <!-- Email Verification -->
              <div
                class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.registration.emailVerification")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.registration.emailVerificationHint") }}
                  </p>
                </div>
                <Toggle v-model="form.email_verify_enabled" />
              </div>

              <!-- Email Suffix Whitelist -->
              <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
                <label class="font-medium text-gray-900 dark:text-white">{{
                  t("admin.settings.registration.emailSuffixWhitelist")
                }}</label>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{
                    t("admin.settings.registration.emailSuffixWhitelistHint")
                  }}
                </p>
                <div
                  class="mt-3 rounded-lg border border-gray-300 bg-white p-2 dark:border-dark-500 dark:bg-dark-700"
                >
                  <div class="flex flex-wrap items-center gap-2">
                    <span
                      v-for="suffix in registrationEmailSuffixWhitelistTags"
                      :key="suffix"
                      class="inline-flex items-center gap-1 rounded bg-gray-100 px-2 py-1 text-xs font-mono text-gray-700 dark:bg-dark-600 dark:text-gray-200"
                    >
                      <span class="text-gray-400 dark:text-gray-500">@</span>
                      <span>{{ suffix }}</span>
                      <LiquidButton
                        type="button"
                        :aria-label="t('common.delete')"
                        class="rounded-full text-gray-500 hover:bg-gray-200 hover:text-gray-700 dark:text-gray-300 dark:hover:bg-dark-500 dark:hover:text-white"
                        @click="
                          removeRegistrationEmailSuffixWhitelistTag(suffix)
                        "
                        variant="plain"
                        size="icon"
                      >
                        <Icon
                          name="x"
                          size="xs"
                          class="h-3.5 w-3.5"
                          :stroke-width="2"
                        />
                      </LiquidButton>
                    </span>

                    <div
                      class="flex min-w-[220px] flex-1 items-center gap-1 rounded border border-transparent px-2 py-1 focus-within:border-primary-300 dark:focus-within:border-primary-700"
                    >
                      <span
                        class="font-mono text-sm text-gray-400 dark:text-gray-500"
                        >@</span
                      >
                      <input
                        v-model="registrationEmailSuffixWhitelistDraft"
                        type="text"
                        class="w-full bg-transparent text-sm font-mono text-gray-900 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
                        :placeholder="
                          t(
                            'admin.settings.registration.emailSuffixWhitelistPlaceholder',
                          )
                        "
                        @input="
                          handleRegistrationEmailSuffixWhitelistDraftInput
                        "
                        @keydown="
                          handleRegistrationEmailSuffixWhitelistDraftKeydown
                        "
                        @blur="commitRegistrationEmailSuffixWhitelistDraft"
                        @paste="handleRegistrationEmailSuffixWhitelistPaste"
                      />
                    </div>
                  </div>
                </div>
                <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                  {{
                    t(
                      "admin.settings.registration.emailSuffixWhitelistInputHint",
                    )
                  }}
                </p>
              </div>

              <!-- Promo Code -->
              <div
                class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.registration.promoCode")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.registration.promoCodeHint") }}
                  </p>
                </div>
                <Toggle v-model="form.promo_code_enabled" />
              </div>

              <!-- Invitation Code -->
              <div
                class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.registration.invitationCode")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.registration.invitationCodeHint") }}
                  </p>
                </div>
                <Toggle v-model="form.invitation_code_enabled" />
              </div>
              <!-- Password Reset - Only show when email verification is enabled -->
              <div
                v-if="form.email_verify_enabled"
                class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.registration.passwordReset")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.registration.passwordResetHint") }}
                  </p>
                </div>
                <Toggle v-model="form.password_reset_enabled" />
              </div>
              <!-- Frontend URL - 判据用 passwordResetIntended（基于 password_reset_enabled_stored），
                 不能用 form.password_reset_enabled：后者是与 email_verify_enabled 取与后的生效值，
                 邮箱验证关闭时恒为 false，会把「配置开着但未生效」的隐患整块藏起来 -->
              <div
                v-if="passwordResetIntended"
                class="border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <!-- 状态 A：正在静默失败 —— 邮箱验证开着、重置开着、前端地址为空，
                   客户此刻点重置就收不到邮件，页面却显示发送成功 -->
                <div
                  v-if="passwordResetSilentlyFailing"
                  data-testid="settings-frontend-url-missing-warning"
                  class="mb-4 rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
                >
                  <div class="flex items-start">
                    <Icon
                      name="exclamationTriangle"
                      size="md"
                      class="mt-0.5 flex-shrink-0 text-amber-500"
                    />
                    <p class="ml-3 text-sm text-amber-700 dark:text-amber-300">
                      {{
                        t(
                          "admin.settings.registration.frontendUrlMissingWarning",
                        )
                      }}
                    </p>
                  </div>
                </div>

                <!-- 状态 B：潜伏 —— 重置在 DB 里是开的，但邮箱验证关着，功能当前未生效，
                   所以现在并没有在静默失败。文案必须与状态 A 区分，不能报未发生的故障 -->
                <div
                  v-else-if="passwordResetLatentlyMisconfigured"
                  data-testid="settings-frontend-url-latent-warning"
                  class="mb-4 rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
                >
                  <div class="flex items-start">
                    <Icon
                      name="exclamationTriangle"
                      size="md"
                      class="mt-0.5 flex-shrink-0 text-amber-500"
                    />
                    <p class="ml-3 text-sm text-amber-700 dark:text-amber-300">
                      {{
                        t(
                          "admin.settings.registration.frontendUrlLatentWarning",
                        )
                      }}
                    </p>
                  </div>
                </div>

                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.registration.frontendUrl") }}
                </label>
                <input
                  v-model="form.frontend_url"
                  type="url"
                  class="input"
                  data-testid="settings-frontend-url-input"
                  :placeholder="
                    t('admin.settings.registration.frontendUrlPlaceholder')
                  "
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.registration.frontendUrlHint") }}
                </p>
                <p
                  v-if="frontendUrlFormatInvalid"
                  data-testid="settings-frontend-url-format-hint"
                  class="mt-2 text-xs text-amber-600 dark:text-amber-400"
                >
                  {{ t("admin.settings.registration.frontendUrlInvalidHint") }}
                </p>
              </div>

              <!-- TOTP 2FA -->
              <div
                class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.registration.totp")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.registration.totpHint") }}
                  </p>
                  <!-- Warning when encryption key not configured -->
                  <p
                    v-if="!form.totp_encryption_key_configured"
                    class="mt-2 text-sm text-amber-600 dark:text-amber-400"
                  >
                    {{ t("admin.settings.registration.totpKeyNotConfigured") }}
                  </p>
                </div>
                <Toggle
                  v-model="form.totp_enabled"
                  :disabled="!form.totp_encryption_key_configured"
                />
              </div>

              <!-- Passkey -->
              <div
                class="border-t border-gray-100 pt-4 dark:border-dark-700"
                data-testid="passkey-settings"
              >
                <div class="flex items-start justify-between gap-4">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.security.passkey")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.security.passkeyHint") }}
                    </p>
                  </div>
                  <Toggle
                    v-model="form.passkey_enabled"
                    data-testid="passkey-toggle"
                    :disabled="!form.passkey_configured"
                  />
                </div>
                <div
                  class="mt-3 rounded-lg border px-3 py-2 text-sm"
                  :class="
                    form.passkey_configured
                      ? 'border-green-200 bg-green-50 text-green-800 dark:border-green-900 dark:bg-green-950/40 dark:text-green-300'
                      : 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900 dark:bg-amber-950/40 dark:text-amber-300'
                  "
                  data-testid="passkey-config-status"
                >
                  <p class="font-medium">
                    {{
                      form.passkey_configured
                        ? t("admin.settings.security.passkeyConfigured")
                        : t("admin.settings.security.passkeyNotConfigured")
                    }}
                  </p>
                  <p class="mt-1 break-all">
                    {{ t("admin.settings.security.passkeyRPID") }}:
                    {{
                      form.passkey_rp_id ||
                      t("admin.settings.security.passkeyValueNotConfigured")
                    }}
                  </p>
                  <p class="mt-1 break-all">
                    {{ t("admin.settings.security.passkeyOrigins") }}:
                    {{
                      form.passkey_rp_origins.length > 0
                        ? form.passkey_rp_origins.join(", ")
                        : t(
                            "admin.settings.security.passkeyValueNotConfigured",
                          )
                    }}
                  </p>
                  <p v-if="!form.passkey_configured" class="mt-2">
                    {{ t("admin.settings.security.passkeyDeploymentHint") }}
                  </p>
                </div>
              </div>

              <!-- 敏感操作 step-up 2FA -->
              <div
                class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="font-medium text-gray-900 dark:text-white"
                    >Passkey 免密登录</label
                  >
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    允许用户在个人资料页注册通行密钥并用于登录。
                  </p>
                  <p
                    v-if="!form.passkey_configured"
                    class="mt-2 text-sm text-amber-600 dark:text-amber-400"
                  >
                    当前 WebAuthn 配置不完整，请先配置 RP ID 与允许来源。
                  </p>
                  <p
                    v-else
                    class="mt-2 text-xs text-gray-500 dark:text-gray-400"
                  >
                    RP ID：{{ form.passkey_rp_id }}；来源：{{
                      form.passkey_rp_origins.join(", ")
                    }}
                  </p>
                  <p v-if="!form.passkey_configured" class="mt-2">
                    {{ t("admin.settings.security.passkeyDeploymentHint") }}
                  </p>
                </div>
                <Toggle
                  v-model="form.passkey_enabled"
                  :disabled="!form.passkey_configured"
                />
              </div>
            </div>
          </div>

          <!-- API Key IP ACL Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.apiKeyAcl.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.apiKeyAcl.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">
                    {{ t("admin.settings.apiKeyAcl.trustForwardedIp") }}
                  </label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.apiKeyAcl.trustForwardedIpHint") }}
                  </p>
                </div>
                <Toggle v-model="form.api_key_acl_trust_forwarded_ip" />
              </div>

              <div
                v-if="form.api_key_acl_trust_forwarded_ip"
                class="border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <label
                  for="forwarded-client-ip-headers"
                  class="font-medium text-gray-900 dark:text-white"
                >
                  {{ t("admin.settings.apiKeyAcl.forwardedClientIpHeaders") }}
                </label>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.apiKeyAcl.forwardedClientIpHeadersHint") }}
                </p>
                <div
                  class="mt-3 rounded-lg border border-gray-300 bg-white p-2 dark:border-dark-500 dark:bg-dark-700"
                >
                  <div class="flex flex-wrap items-center gap-2">
                    <span
                      v-for="header in form.forwarded_client_ip_headers"
                      :key="header"
                      data-testid="forwarded-client-ip-header-tag"
                      class="inline-flex items-center gap-1 rounded bg-gray-100 px-2 py-1 text-xs font-mono text-gray-700 dark:bg-dark-600 dark:text-gray-200"
                    >
                      <span>{{ header }}</span>
                      <button
                        type="button"
                        class="rounded-full text-gray-500 hover:bg-gray-200 hover:text-gray-700 dark:text-gray-300 dark:hover:bg-dark-500 dark:hover:text-white"
                        :aria-label="t('admin.settings.apiKeyAcl.removeForwardedClientIpHeader', { header })"
                        @click="removeForwardedClientIpHeader(header)"
                      >
                        <Icon
                          name="x"
                          size="xs"
                          class="h-3.5 w-3.5"
                          :stroke-width="2"
                        />
                      </button>
                    </span>
                    <div
                      class="flex min-w-[220px] flex-1 items-center gap-1 rounded border border-transparent px-2 py-1 focus-within:border-primary-300 dark:focus-within:border-primary-700"
                    >
                      <input
                        id="forwarded-client-ip-headers"
                        v-model="forwardedClientIpHeaderDraft"
                        data-testid="forwarded-client-ip-headers-input"
                        type="text"
                        class="w-full bg-transparent text-sm font-mono text-gray-900 outline-none placeholder:text-gray-400 dark:text-white dark:placeholder:text-gray-500"
                        :placeholder="t('admin.settings.apiKeyAcl.forwardedClientIpHeadersPlaceholder')"
                        @keydown="handleForwardedClientIpHeaderKeydown"
                        @blur="commitForwardedClientIpHeaderDraft"
                        @paste="handleForwardedClientIpHeaderPaste"
                      />
                    </div>
                  </div>
                </div>
                <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.apiKeyAcl.forwardedClientIpHeadersRiskHint") }}
                </p>
              </div>
            </div>
          </div>

          <!-- Panel API Rate Limit Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <div class="flex items-center gap-2">
                <Icon
                  name="shield"
                  size="md"
                  class="text-primary-500"
                />
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.panelRateLimit.title") }}
                </h2>
              </div>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.panelRateLimit.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div
                v-if="panelRateLimitLoading"
                class="flex items-center gap-2 text-gray-500"
              >
                <div
                  class="h-4 w-4 animate-spin rounded-full border-b-2 border-primary-600"
                ></div>
                {{ t("common.loading") }}
              </div>

              <template v-else>
                <!-- 计数维度说明：按账号计数，反代部署无误伤 -->
                <div
                  class="rounded-lg border border-sky-200 bg-sky-50 p-4 dark:border-sky-800 dark:bg-sky-900/20"
                >
                  <div class="flex items-start">
                    <Icon
                      name="infoCircle"
                      size="md"
                      class="mt-0.5 flex-shrink-0 text-sky-500"
                    />
                    <p class="ml-3 text-sm text-sky-700 dark:text-sky-300">
                      {{ t("admin.settings.panelRateLimit.proxySafeNote") }}
                    </p>
                  </div>
                </div>

                <div class="flex items-center justify-between">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">{{
                      t("admin.settings.panelRateLimit.enabled")
                    }}</label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.panelRateLimit.enabledHint") }}
                    </p>
                  </div>
                  <Toggle v-model="panelRateLimitForm.enabled" />
                </div>

                <div
                  v-if="panelRateLimitForm.enabled"
                  class="space-y-5 border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
                    <div>
                      <label
                        class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                      >
                        {{ t("admin.settings.panelRateLimit.userRpm") }}
                      </label>
                      <div class="flex items-center gap-2">
                        <input
                          v-model.number="panelRateLimitForm.user_rpm"
                          data-testid="panel-rate-limit-user-rpm"
                          type="number"
                          min="0"
                          max="100000"
                          class="input w-32"
                        />
                        <span class="text-sm text-gray-500 dark:text-gray-400">
                          {{ t("admin.settings.panelRateLimit.perMinute") }}
                        </span>
                      </div>
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.panelRateLimit.userRpmHint") }}
                      </p>
                    </div>

                    <div>
                      <label
                        class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                      >
                        {{ t("admin.settings.panelRateLimit.heavyRpm") }}
                      </label>
                      <div class="flex items-center gap-2">
                        <input
                          v-model.number="panelRateLimitForm.heavy_rpm"
                          type="number"
                          min="0"
                          max="100000"
                          class="input w-32"
                        />
                        <span class="text-sm text-gray-500 dark:text-gray-400">
                          {{ t("admin.settings.panelRateLimit.perMinute") }}
                        </span>
                      </div>
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.panelRateLimit.heavyRpmHint") }}
                      </p>
                    </div>

                    <div>
                      <label
                        class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                      >
                        {{ t("admin.settings.panelRateLimit.publicIpRpm") }}
                      </label>
                      <div class="flex items-center gap-2">
                        <input
                          v-model.number="panelRateLimitForm.public_ip_rpm"
                          type="number"
                          min="0"
                          max="100000"
                          class="input w-32"
                        />
                        <span class="text-sm text-gray-500 dark:text-gray-400">
                          {{ t("admin.settings.panelRateLimit.perMinute") }}
                        </span>
                      </div>
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.panelRateLimit.publicIpRpmHint") }}
                      </p>
                    </div>
                  </div>

                  <div
                    class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
                  >
                    <div>
                      <label class="font-medium text-gray-900 dark:text-white">{{
                        t("admin.settings.panelRateLimit.exemptAdmin")
                      }}</label>
                      <p class="text-sm text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.panelRateLimit.exemptAdminHint") }}
                      </p>
                    </div>
                    <Toggle v-model="panelRateLimitForm.exempt_admin" />
                  </div>
                </div>

                <div
                  class="flex justify-end border-t border-gray-100 pt-4 dark:border-dark-700"
                >
                  <button
                    type="button"
                    data-testid="panel-rate-limit-save"
                    @click="savePanelRateLimitSettings"
                    :disabled="panelRateLimitSaving"
                    class="btn btn-primary btn-sm"
                  >
                    <svg
                      v-if="panelRateLimitSaving"
                      class="mr-1 h-4 w-4 animate-spin"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        class="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        stroke-width="4"
                      ></circle>
                      <path
                        class="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    {{
                      panelRateLimitSaving
                        ? t("common.saving")
                        : t("common.save")
                    }}
                  </button>
                </div>
              </template>
            </div>
          </div>

          <!-- 人机验证 Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.captcha.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.captcha.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <!-- Enable Captcha -->
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.captcha.enable")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.captcha.enableHint") }}
                  </p>
                </div>
                <Toggle
                  v-model="captchaMasterEnabled"
                  data-testid="captcha-enabled-toggle"
                />
              </div>

              <!-- Provider fields - Only show when enabled -->
              <div
                v-if="captchaMasterEnabled"
                class="border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <!-- Provider Selector -->
                <div class="mb-6">
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.captcha.provider") }}
                  </label>
                  <div
                    class="grid grid-cols-3 gap-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-700"
                  >
                    <button
                      type="button"
                      data-testid="captcha-provider-turnstile"
                      class="inline-flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition"
                      :class="
                        captchaProviderSelection === 'turnstile'
                          ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                          : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                      "
                      @click="selectCaptchaProvider('turnstile')"
                    >
                      {{ t("admin.settings.captcha.providerTurnstile") }}
                    </button>
                    <button
                      type="button"
                      data-testid="captcha-provider-tencent"
                      class="inline-flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition"
                      :class="
                        captchaProviderSelection === 'tencent'
                          ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                          : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                      "
                      @click="selectCaptchaProvider('tencent')"
                    >
                      {{ t("admin.settings.captcha.providerTencent") }}
                    </button>
                    <button
                      type="button"
                      data-testid="captcha-provider-aliyun"
                      class="inline-flex items-center justify-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition"
                      :class="
                        captchaProviderSelection === 'aliyun'
                          ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                          : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                      "
                      @click="selectCaptchaProvider('aliyun')"
                    >
                      {{ t("admin.settings.captcha.providerAliyun") }}
                    </button>
                  </div>
                </div>

                <!-- Cloudflare Turnstile fields -->
                <div
                  v-if="captchaProviderSelection === 'turnstile'"
                  class="grid grid-cols-1 gap-6"
                >
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.turnstile.siteKey") }}
                    </label>
                    <input
                      v-model="form.turnstile_site_key"
                      type="text"
                      class="input font-mono text-sm"
                      placeholder="0x4AAAAAAA..."
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.turnstile.siteKeyHint") }}
                      <a
                        href="https://dash.cloudflare.com/"
                        target="_blank"
                        class="text-primary-600 hover:text-primary-500"
                        >{{
                          t("admin.settings.turnstile.cloudflareDashboard")
                        }}</a
                      >
                    </p>
                  </div>
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.turnstile.secretKey") }}
                    </label>
                    <input
                      v-model="form.turnstile_secret_key"
                      type="password"
                      class="input font-mono text-sm"
                      placeholder="0x4AAAAAAA..."
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{
                        form.turnstile_secret_key_configured
                          ? t(
                              "admin.settings.turnstile.secretKeyConfiguredHint",
                            )
                          : t("admin.settings.turnstile.secretKeyHint")
                      }}
                    </p>
                  </div>
                </div>

                <!-- Tencent Captcha fields -->
                <div v-else-if="captchaProviderSelection === 'tencent'">
                  <div class="mb-6 max-w-sm">
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      {{ t("admin.settings.tencentCaptcha.region") }}
                    </label>
                    <div class="grid grid-cols-2 gap-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-700">
                      <button
                        type="button"
                        data-testid="tencent-captcha-region-cn"
                        class="inline-flex items-center justify-center rounded-md px-3 py-1.5 text-sm font-medium transition"
                        :class="
                          form.tencent_captcha_region !== 'intl'
                            ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                            : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                        "
                        @click="form.tencent_captcha_region = 'cn'"
                      >
                        {{ t("admin.settings.tencentCaptcha.regionCn") }}
                      </button>
                      <button
                        type="button"
                        data-testid="tencent-captcha-region-intl"
                        class="inline-flex items-center justify-center rounded-md px-3 py-1.5 text-sm font-medium transition"
                        :class="
                          form.tencent_captcha_region === 'intl'
                            ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                            : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                        "
                        @click="form.tencent_captcha_region = 'intl'"
                      >
                        {{ t("admin.settings.tencentCaptcha.regionIntl") }}
                      </button>
                    </div>
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.tencentCaptcha.regionHint") }}
                    </p>
                  </div>
                  <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
                    <div class="md:col-span-2">
                      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                        {{ t("admin.settings.tencentCaptcha.appCredentialsTitle") }}
                      </h3>
                      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.tencentCaptcha.appCredentialsHint") }}
                      </p>
                    </div>
                    <div>
                      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t("admin.settings.tencentCaptcha.appId") }}
                      </label>
                      <input
                        v-model="form.tencent_captcha_app_id"
                        type="text"
                        inputmode="numeric"
                        class="input font-mono text-sm"
                        placeholder="123456789"
                      />
                    </div>
                    <div>
                      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t("admin.settings.tencentCaptcha.appSecretKey") }}
                      </label>
                      <input
                        v-model="form.tencent_captcha_app_secret_key"
                        type="password"
                        autocomplete="new-password"
                        class="input font-mono text-sm"
                        :placeholder="t('admin.settings.tencentCaptcha.keepExisting')"
                      />
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ form.tencent_captcha_app_secret_key_configured ? t("admin.settings.tencentCaptcha.configured") : t("admin.settings.tencentCaptcha.required") }}
                      </p>
                    </div>
                    <div class="border-t border-gray-100 pt-5 md:col-span-2 dark:border-dark-700">
                      <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                        {{ t("admin.settings.tencentCaptcha.cloudCredentialsTitle") }}
                      </h3>
                      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.tencentCaptcha.cloudCredentialsHint") }}
                      </p>
                    </div>
                    <div>
                      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t("admin.settings.tencentCaptcha.cloudSecretId") }}
                      </label>
                      <input
                        v-model="form.tencent_captcha_cloud_secret_id"
                        type="password"
                        autocomplete="new-password"
                        class="input font-mono text-sm"
                        :placeholder="t('admin.settings.tencentCaptcha.keepExisting')"
                      />
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ form.tencent_captcha_cloud_secret_id_configured ? t("admin.settings.tencentCaptcha.configured") : t("admin.settings.tencentCaptcha.required") }}
                      </p>
                    </div>
                    <div>
                      <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                        {{ t("admin.settings.tencentCaptcha.cloudSecretKey") }}
                      </label>
                      <input
                        v-model="form.tencent_captcha_cloud_secret_key"
                        type="password"
                        autocomplete="new-password"
                        class="input font-mono text-sm"
                        :placeholder="t('admin.settings.tencentCaptcha.keepExisting')"
                      />
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ form.tencent_captcha_cloud_secret_key_configured ? t("admin.settings.tencentCaptcha.configured") : t("admin.settings.tencentCaptcha.required") }}
                      </p>
                    </div>
                  </div>
                  <p class="mt-5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.tencentCaptcha.camPermissionHint") }}
                  </p>
                  <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.tencentCaptcha.aidEncryptedHint") }}
                  </p>
                  <div class="mt-3 flex flex-wrap gap-x-4 gap-y-2 text-sm">
                    <a
                      :href="tencentCaptchaLinks.console"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="text-primary-600 hover:text-primary-500"
                    >
                      {{ t("admin.settings.tencentCaptcha.openCaptchaConsole") }}
                    </a>
                    <a
                      :href="tencentCaptchaLinks.cloudKeys"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="text-primary-600 hover:text-primary-500"
                    >
                      {{ t("admin.settings.tencentCaptcha.createCloudKeys") }}
                    </a>
                    <a
                      :href="tencentCaptchaLinks.webDocs"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="text-primary-600 hover:text-primary-500"
                    >
                      {{ t("admin.settings.tencentCaptcha.openWebDocs") }}
                    </a>
                  </div>
                </div>

                <!-- Aliyun Captcha 2.0 fields -->
                <div v-else class="grid grid-cols-1 gap-6">
                  <div class="grid grid-cols-1 gap-6 sm:grid-cols-2">
                    <div>
                      <label
                        class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                      >
                        {{ t("admin.settings.aliyunCaptcha.region") }}
                      </label>
                      <div
                        class="grid grid-cols-2 gap-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-700"
                      >
                        <button
                          type="button"
                          class="inline-flex items-center justify-center rounded-md px-3 py-1.5 text-sm font-medium transition"
                          :class="
                            form.aliyun_captcha_region !== 'sgp'
                              ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                              : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                          "
                          @click="form.aliyun_captcha_region = 'cn'"
                        >
                          {{ t("admin.settings.aliyunCaptcha.regionCn") }}
                        </button>
                        <button
                          type="button"
                          class="inline-flex items-center justify-center rounded-md px-3 py-1.5 text-sm font-medium transition"
                          :class="
                            form.aliyun_captcha_region === 'sgp'
                              ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                              : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                          "
                          @click="form.aliyun_captcha_region = 'sgp'"
                        >
                          {{ t("admin.settings.aliyunCaptcha.regionSgp") }}
                        </button>
                      </div>
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.aliyunCaptcha.regionHint") }}
                      </p>
                    </div>
                    <div>
                      <label
                        class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                      >
                        {{ t("admin.settings.aliyunCaptcha.prefix") }}
                      </label>
                      <input
                        v-model="form.aliyun_captcha_prefix"
                        type="text"
                        class="input font-mono text-sm"
                        placeholder="14xxxxx"
                      />
                      <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.aliyunCaptcha.prefixHint") }}
                      </p>
                    </div>
                  </div>
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.aliyunCaptcha.sceneId") }}
                    </label>
                    <input
                      v-model="form.aliyun_captcha_scene_id"
                      type="text"
                      class="input font-mono text-sm"
                      placeholder="1cxxxxxx"
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.aliyunCaptcha.sceneIdHint") }}
                    </p>
                  </div>
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.aliyunCaptcha.accessKeyId") }}
                    </label>
                    <input
                      v-model="form.aliyun_captcha_access_key_id"
                      type="text"
                      class="input font-mono text-sm"
                      placeholder="LTAI..."
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.aliyunCaptcha.accessKeyIdHint") }}
                    </p>
                  </div>
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.aliyunCaptcha.accessKeySecret") }}
                    </label>
                    <input
                      v-model="form.aliyun_captcha_access_key_secret"
                      type="password"
                      autocomplete="new-password"
                      class="input font-mono text-sm"
                      placeholder="••••••••"
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{
                        form.aliyun_captcha_access_key_secret_configured
                          ? t(
                              "admin.settings.aliyunCaptcha.accessKeySecretConfiguredHint",
                            )
                          : t("admin.settings.aliyunCaptcha.accessKeySecretHint")
                      }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- LinuxDo Connect OAuth 登录 -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.linuxdo.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.linuxdo.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.linuxdo.enable")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.linuxdo.enableHint") }}
                  </p>
                </div>
                <Toggle v-model="form.linuxdo_connect_enabled" />
              </div>

              <div
                v-if="form.linuxdo_connect_enabled"
                class="border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div class="grid grid-cols-1 gap-6">
                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.linuxdo.clientId") }}
                    </label>
                    <input
                      v-model="form.linuxdo_connect_client_id"
                      type="text"
                      class="input font-mono text-sm"
                      :placeholder="
                        t('admin.settings.linuxdo.clientIdPlaceholder')
                      "
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.linuxdo.clientIdHint") }}
                    </p>
                  </div>

                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.linuxdo.clientSecret") }}
                    </label>
                    <input
                      v-model="form.linuxdo_connect_client_secret"
                      type="password"
                      class="input font-mono text-sm"
                      :placeholder="
                        form.linuxdo_connect_client_secret_configured
                          ? t(
                              'admin.settings.linuxdo.clientSecretConfiguredPlaceholder',
                            )
                          : t('admin.settings.linuxdo.clientSecretPlaceholder')
                      "
                    />
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{
                        form.linuxdo_connect_client_secret_configured
                          ? t(
                              "admin.settings.linuxdo.clientSecretConfiguredHint",
                            )
                          : t("admin.settings.linuxdo.clientSecretHint")
                      }}
                    </p>
                  </div>

                  <div>
                    <label
                      class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                    >
                      {{ t("admin.settings.linuxdo.redirectUrl") }}
                    </label>
                    <input
                      v-model="form.linuxdo_connect_redirect_url"
                      type="url"
                      class="input font-mono text-sm"
                      :placeholder="
                        t('admin.settings.linuxdo.redirectUrlPlaceholder')
                      "
                    />
                    <div
                      class="mt-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:gap-3"
                    >
                      <LiquidButton
                        type="button"
                        class="w-fit"
                        @click="setAndCopyLinuxdoRedirectUrl"
                        variant="outline"
                        size="sm"
                      >
                        {{ t("admin.settings.linuxdo.quickSetCopy") }}
                      </LiquidButton>
                      <code
                        v-if="linuxdoRedirectUrlSuggestion"
                        class="select-all break-all rounded bg-gray-50 px-2 py-1 font-mono text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300"
                      >
                        {{ linuxdoRedirectUrlSuggestion }}
                      </code>
                    </div>
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.linuxdo.redirectUrlHint") }}
                    </p>
                  </div>
                </div>
              </div>
            </div>
          </div>

          <!-- WeChat Connect OAuth -->
          <div class="card" data-testid="wechat-connect-card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.wechatConnect.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.wechatConnect.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">
                    {{ t("admin.settings.wechatConnect.enabledLabel") }}
                  </label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.wechatConnect.enabledHint") }}
                  </p>
                </div>
                <Toggle
                  v-model="form.wechat_connect_enabled"
                  data-testid="wechat-connect-enabled"
                />
              </div>

              <div
                v-if="form.wechat_connect_enabled"
                class="space-y-5 border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div class="grid grid-cols-1 gap-4 md:grid-cols-3">
                  <div class="flex items-center justify-between gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
                    <div>
                      <label class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ t("admin.settings.wechatConnect.openModeLabel") }}
                      </label>
                      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.wechatConnect.openModeHint") }}
                      </p>
                    </div>
                    <Toggle
                      :model-value="form.wechat_connect_open_enabled"
                      data-testid="wechat-connect-open-enabled"
                      @update:model-value="handleWeChatOpenEnabledChange"
                    />
                  </div>
                  <div class="flex items-center justify-between gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
                    <div>
                      <label class="text-sm font-medium text-gray-900 dark:text-white">
                        {{ t("admin.settings.wechatConnect.mpModeLabel") }}
                      </label>
                      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        {{ t("admin.settings.wechatConnect.mpModeHint") }}
                      </p>
                    </div>
                    <Toggle
                      :model-value="form.wechat_connect_mp_enabled"
                      data-testid="wechat-connect-mp-enabled"
                      @update:model-value="handleWeChatMPEnabledChange"
                    />
                  </div>
                  <div class="flex items-center justify-between gap-3 rounded-lg border border-gray-200 p-3 dark:border-dark-600">
                    <div>
                      <label class="text-sm font-medium text-gray-900 dark:text-white">
                        移动应用
                      </label>
                      <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                        微信移动应用 OAuth。
                      </p>
                    </div>
                    <Toggle
                      :model-value="form.wechat_connect_mobile_enabled"
                      data-testid="wechat-connect-mobile-enabled"
                      @update:model-value="handleWeChatMobileEnabledChange"
                    />
                  </div>
                </div>

                <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
                  <div>
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      PC 应用 AppID
                    </label>
                    <input
                      v-model="form.wechat_connect_open_app_id"
                      type="text"
                      class="input font-mono text-sm"
                      data-testid="wechat-connect-open-app-id"
                      :placeholder="t('admin.settings.wechatConnect.appIdPlaceholder')"
                    />
                  </div>
                  <div>
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      PC 应用 AppSecret
                    </label>
                    <input
                      v-model="form.wechat_connect_open_app_secret"
                      type="password"
                      class="input font-mono text-sm"
                      data-testid="wechat-connect-open-app-secret"
                      :placeholder="
                        form.wechat_connect_open_app_secret_configured
                          ? t('admin.settings.wechatConnect.appSecretConfiguredPlaceholder')
                          : t('admin.settings.wechatConnect.appSecretPlaceholder')
                      "
                    />
                  </div>
                  <div>
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      公众号 AppID
                    </label>
                    <input
                      v-model="form.wechat_connect_mp_app_id"
                      type="text"
                      class="input font-mono text-sm"
                      data-testid="wechat-connect-mp-app-id"
                      :placeholder="t('admin.settings.wechatConnect.appIdPlaceholder')"
                    />
                  </div>
                  <div>
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      公众号 AppSecret
                    </label>
                    <input
                      v-model="form.wechat_connect_mp_app_secret"
                      type="password"
                      class="input font-mono text-sm"
                      data-testid="wechat-connect-mp-app-secret"
                      :placeholder="
                        form.wechat_connect_mp_app_secret_configured
                          ? t('admin.settings.wechatConnect.appSecretConfiguredPlaceholder')
                          : t('admin.settings.wechatConnect.appSecretPlaceholder')
                      "
                    />
                  </div>
                  <div>
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      移动应用 AppID
                    </label>
                    <input
                      v-model="form.wechat_connect_mobile_app_id"
                      type="text"
                      class="input font-mono text-sm"
                      data-testid="wechat-connect-mobile-app-id"
                      placeholder="移动应用 AppID"
                    />
                  </div>
                  <div>
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      移动应用 AppSecret
                    </label>
                    <input
                      v-model="form.wechat_connect_mobile_app_secret"
                      type="password"
                      class="input font-mono text-sm"
                      data-testid="wechat-connect-mobile-app-secret"
                      :placeholder="
                        form.wechat_connect_mobile_app_secret_configured
                          ? t('admin.settings.wechatConnect.appSecretConfiguredPlaceholder')
                          : t('admin.settings.wechatConnect.appSecretPlaceholder')
                      "
                    />
                  </div>
                </div>

                <div
                  v-if="form.wechat_connect_mode === 'open' || form.wechat_connect_mode === 'mobile'"
                >
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t("admin.settings.wechatConnect.scopesLabel") }}
                  </label>
                  <input
                    v-model="form.wechat_connect_scopes"
                    type="text"
                    class="input font-mono text-sm"
                    data-testid="wechat-connect-scopes"
                    placeholder="snsapi_login"
                  />
                </div>

                <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
                  <div>
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      {{ t("admin.settings.wechatConnect.redirectUrlLabel") }}
                    </label>
                    <input
                      v-model="form.wechat_connect_redirect_url"
                      type="url"
                      class="input font-mono text-sm"
                      data-testid="wechat-connect-redirect-url"
                      :placeholder="t('admin.settings.wechatConnect.redirectUrlPlaceholder')"
                    />
                    <div class="mt-2 flex flex-wrap items-center gap-2">
                      <LiquidButton
                        type="button"
                        variant="outline"
                        size="sm"
                        @click="setAndCopyWeChatRedirectUrl"
                      >
                        {{ t("admin.settings.wechatConnect.generateAndCopy") }}
                      </LiquidButton>
                      <code
                        v-if="wechatRedirectUrlSuggestion"
                        class="select-all break-all rounded bg-gray-50 px-2 py-1 font-mono text-xs text-gray-600 dark:bg-dark-800 dark:text-gray-300"
                      >
                        {{ wechatRedirectUrlSuggestion }}
                      </code>
                    </div>
                  </div>
                  <div>
                    <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                      {{ t("admin.settings.wechatConnect.frontendRedirectUrlLabel") }}
                    </label>
                    <input
                      v-model="form.wechat_connect_frontend_redirect_url"
                      type="text"
                      class="input font-mono text-sm"
                      data-testid="wechat-connect-frontend-redirect-url"
                      :placeholder="t('admin.settings.wechatConnect.frontendRedirectUrlPlaceholder')"
                    />
                  </div>
                </div>
              </div>
              <div class="border-t border-gray-100 pt-4 text-sm dark:border-dark-700">
                <a
                  href="https://github.com/settings/developers"
                  target="_blank"
                  rel="noopener noreferrer"
                  data-testid="github-oauth-apps-guide-link"
                  class="text-primary-600 hover:underline dark:text-primary-400"
                >
                  GitHub OAuth Apps
                </a>
              </div>
            </div>
          </div>

          <!-- Generic OIDC OAuth -->
          <div class="card" data-testid="oidc-connect-card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                OIDC Connect
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                配置兼容 OpenID Connect 的第三方登录提供商。
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">启用 OIDC 登录</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">仅在参数完整且提供商可用时启用。</p>
                </div>
                <Toggle v-model="form.oidc_connect_enabled" data-testid="oidc-connect-enabled" />
              </div>
              <div v-if="form.oidc_connect_enabled" class="grid grid-cols-1 gap-6 border-t border-gray-100 pt-4 md:grid-cols-2 dark:border-dark-700">
                <div v-for="field in oidcTextFields" :key="field[0]">
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ field[1] }}</label>
                  <input
                    :value="oidcTextFieldValue(field[0])"
                    type="text"
                    class="input font-mono text-sm"
                    :data-testid="field[0]"
                    @input="setOIDCTextFieldValue(field[0], $event)"
                  />
                </div>
                <!--
                  ['oidc_connect_provider_name', '提供商名称'],
                  ['oidc_connect_client_id', 'Client ID'],
                  ['oidc_connect_issuer_url', 'Issuer URL'],
                  ['oidc_connect_discovery_url', 'Discovery URL'],
                  ['oidc_connect_authorize_url', 'Authorize URL'],
                  ['oidc_connect_token_url', 'Token URL'],
                  ['oidc_connect_userinfo_url', 'UserInfo URL'],
                  ['oidc_connect_jwks_url', 'JWKS URL'],
                  ['oidc_connect_scopes', 'Scopes'],
                  ['oidc_connect_redirect_url', 'Redirect URL'],
                  ['oidc_connect_frontend_redirect_url', '前端回调路径'],
                ]" :key="field[0]">
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">{{ field[1] }}</label>
                  <input
                    :value="oidcTextFieldValue(field[0])"
                    type="text"
                    class="input font-mono text-sm"
                    :data-testid="field[0]"
                    @input="setOIDCTextFieldValue(field[0], $event)"
                  />
                </div>
                -->
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">Client Secret</label>
                  <input
                    v-model="form.oidc_connect_client_secret"
                    type="password"
                    class="input font-mono text-sm"
                    data-testid="oidc-connect-client-secret"
                    :placeholder="form.oidc_connect_client_secret_configured ? '密钥已配置，留空保持不变' : '请输入 Client Secret'"
                  />
                </div>
                <div class="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-dark-600">
                  <span class="text-sm text-gray-700 dark:text-gray-300">使用 PKCE</span>
                  <Toggle v-model="form.oidc_connect_use_pkce" data-testid="oidc-connect-use-pkce" />
                </div>
                <div class="flex items-center justify-between rounded-lg border border-gray-200 p-3 dark:border-dark-600">
                  <span class="text-sm text-gray-700 dark:text-gray-300">校验 ID Token</span>
                  <Toggle v-model="form.oidc_connect_validate_id_token" data-testid="oidc-connect-validate-id-token" />
                </div>
              </div>
            </div>
          </div>
        </div>
        <!-- /Tab: Security — Registration, Turnstile, LinuxDo -->

        <!-- Tab: Users -->
        <div v-show="activeTab === 'users'" class="space-y-6">
          <!-- Default Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.defaults.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.defaults.description") }}
              </p>
            </div>
            <div class="space-y-6 p-6">
              <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.defaults.defaultBalance") }}
                  </label>
                  <input
                    v-model.number="form.default_balance"
                    type="number"
                    step="0.01"
                    min="0"
                    class="input"
                    placeholder="0.00"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.defaults.defaultBalanceHint") }}
                  </p>
                </div>
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.defaults.defaultConcurrency") }}
                  </label>
                  <input
                    v-model.number="form.default_concurrency"
                    type="number"
                    min="1"
                    class="input"
                    placeholder="1"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.defaults.defaultConcurrencyHint") }}
                  </p>
                </div>
              </div>

              <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
                <div class="mb-3 flex items-center justify-between">
                  <div>
                    <label class="font-medium text-gray-900 dark:text-white">
                      {{ t("admin.settings.defaults.defaultSubscriptions") }}
                    </label>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{
                        t("admin.settings.defaults.defaultSubscriptionsHint")
                      }}
                    </p>
                  </div>
                  <LiquidButton
                    type="button"
                    @click="addDefaultSubscription"
                    :disabled="subscriptionGroups.length === 0"
                    variant="outline"
                    size="sm"
                  >
                    {{ t("admin.settings.defaults.addDefaultSubscription") }}
                  </LiquidButton>
                </div>

                <div
                  v-if="form.default_subscriptions.length === 0"
                  class="rounded border border-dashed border-gray-300 px-4 py-3 text-sm text-gray-500 dark:border-dark-600 dark:text-gray-400"
                >
                  {{ t("admin.settings.defaults.defaultSubscriptionsEmpty") }}
                </div>

                <div v-else class="space-y-3">
                  <div
                    v-for="(item, index) in form.default_subscriptions"
                    :key="`default-sub-${index}`"
                    class="grid grid-cols-1 gap-3 rounded border border-gray-200 p-3 md:grid-cols-[1fr_160px_auto] dark:border-dark-600"
                  >
                    <div>
                      <label
                        class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                      >
                        {{ t("admin.settings.defaults.subscriptionGroup") }}
                      </label>
                      <Select
                        v-model="item.group_id"
                        class="default-sub-group-select"
                        :options="defaultSubscriptionGroupOptions"
                        :placeholder="
                          t('admin.settings.defaults.subscriptionGroup')
                        "
                      >
                        <template #selected="{ option }">
                          <GroupBadge
                            v-if="option"
                            :name="
                              (
                                option as unknown as DefaultSubscriptionGroupOption
                              ).label
                            "
                            :platform="
                              (
                                option as unknown as DefaultSubscriptionGroupOption
                              ).platform
                            "
                            :subscription-type="
                              (
                                option as unknown as DefaultSubscriptionGroupOption
                              ).subscriptionType
                            "
                            :rate-multiplier="
                              (
                                option as unknown as DefaultSubscriptionGroupOption
                              ).rate
                            "
                          />
                          <span v-else class="text-gray-400">
                            {{ t("admin.settings.defaults.subscriptionGroup") }}
                          </span>
                        </template>
                        <template #option="{ option, selected }">
                          <GroupOptionItem
                            :name="
                              (
                                option as unknown as DefaultSubscriptionGroupOption
                              ).label
                            "
                            :platform="
                              (
                                option as unknown as DefaultSubscriptionGroupOption
                              ).platform
                            "
                            :subscription-type="
                              (
                                option as unknown as DefaultSubscriptionGroupOption
                              ).subscriptionType
                            "
                            :rate-multiplier="
                              (
                                option as unknown as DefaultSubscriptionGroupOption
                              ).rate
                            "
                            :description="
                              (
                                option as unknown as DefaultSubscriptionGroupOption
                              ).description
                            "
                            :selected="selected"
                          />
                        </template>
                      </Select>
                    </div>
                    <div>
                      <label
                        class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                      >
                        {{
                          t("admin.settings.defaults.subscriptionValidityDays")
                        }}
                      </label>
                      <input
                        v-model.number="item.validity_days"
                        type="number"
                        min="1"
                        max="36500"
                        class="input h-[42px]"
                      />
                    </div>
                    <div class="flex items-end">
                      <LiquidButton
                        type="button"
                        class="default-sub-delete-btn w-full text-red-600 hover:text-red-700 dark:text-red-400"
                        @click="removeDefaultSubscription(index)"
                        variant="outline"
                        size="sm"
                      >
                        {{ t("common.delete") }}
                      </LiquidButton>
                    </div>
                  </div>
                </div>
              </div>

              <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
                <div class="mb-3">
                  <label class="font-medium text-gray-900 dark:text-white">
                    平台默认额度
                  </label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    为新注册用户按平台设置每日、每周和每月的默认额度；留空表示不设置该周期额度。
                  </p>
                </div>
                <div class="overflow-x-auto">
                  <table class="w-full min-w-[560px] text-sm">
                    <thead class="text-left text-xs text-gray-500 dark:text-gray-400">
                      <tr>
                        <th class="pb-2 font-medium">平台</th>
                        <th class="pb-2 font-medium">每日</th>
                        <th class="pb-2 font-medium">每周</th>
                        <th class="pb-2 font-medium">每月</th>
                      </tr>
                    </thead>
                    <tbody>
                      <tr
                        v-for="platform in platformQuotaPlatforms"
                        :key="platform"
                        class="border-t border-gray-100 dark:border-dark-700"
                      >
                        <td class="py-2 font-mono text-xs text-gray-700 dark:text-gray-300">
                          {{ platform }}
                        </td>
                        <td v-for="window in ['daily', 'weekly', 'monthly']" :key="window" class="py-2 pr-2">
                          <input
                            v-model.number="form.default_platform_quotas![platform]![window as 'daily' | 'weekly' | 'monthly']"
                            type="number"
                            min="0"
                            step="0.01"
                            class="input w-28 text-sm"
                            placeholder="不限"
                          />
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              </div>
            </div>
          </div>

          <!-- Channel Monitor Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.features.channelMonitor.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.features.channelMonitor.description") }}
              </p>
              <p class="mt-1.5 text-xs">
                <router-link
                  to="/admin/channels/monitor"
                  class="inline-flex items-center gap-1 text-primary-600 hover:underline dark:text-primary-400"
                >
                  {{ t("admin.settings.features.channelMonitor.configureLink") }}
                  <span aria-hidden="true">→</span>
                </router-link>
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">
                    {{ t("admin.settings.features.channelMonitor.enabled") }}
                  </label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.features.channelMonitor.enabledHint") }}
                  </p>
                </div>
                <Toggle v-model="form.channel_monitor_enabled" />
              </div>

              <template v-if="form.channel_monitor_enabled">
                <div>
                  <label class="input-label">
                    {{ t("admin.settings.features.channelMonitor.mode") }}
                  </label>
                  <div
                    class="mt-1.5 inline-flex w-full max-w-md rounded-lg border border-gray-200 bg-gray-50 p-1 dark:border-dark-600 dark:bg-dark-900/40"
                  >
                    <button
                      v-for="mode in channelMonitorModes"
                      :key="mode"
                      type="button"
                      class="inline-flex flex-1 items-center justify-center rounded-md px-3 py-2 text-sm font-medium transition"
                      :class="
                        form.channel_monitor_mode === mode
                          ? 'bg-white text-primary-700 shadow-sm dark:bg-dark-800 dark:text-primary-300'
                          : 'text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white'
                      "
                      @click="form.channel_monitor_mode = mode"
                    >
                      {{
                        t(
                          mode === 'v2'
                            ? 'admin.settings.features.channelMonitor.modeV2'
                            : 'admin.settings.features.channelMonitor.modeV1',
                        )
                      }}
                    </button>
                  </div>
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      t(
                        form.channel_monitor_mode === 'v1'
                          ? 'admin.settings.features.channelMonitor.modeV1Hint'
                          : 'admin.settings.features.channelMonitor.modeV2Hint',
                      )
                    }}
                  </p>
                </div>

                <div v-if="form.channel_monitor_mode === 'v1'">
                  <label class="input-label">
                    {{ t("admin.settings.features.channelMonitor.defaultInterval") }}
                  </label>
                  <input
                    v-model.number="form.channel_monitor_default_interval_seconds"
                    type="number"
                    min="15"
                    max="3600"
                    class="input"
                  />
                  <p class="mt-1 text-xs text-gray-400">
                    {{ t("admin.settings.features.channelMonitor.defaultIntervalHint") }}
                  </p>
                </div>

                <div
                  v-if="form.channel_monitor_mode === 'v2'"
                  class="flex items-start justify-between gap-4"
                >
                  <div>
                    <p class="font-medium text-gray-900 dark:text-white">
                      {{ t("admin.settings.features.channelMonitor.hideThroughput") }}
                    </p>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.features.channelMonitor.hideThroughputHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.channel_monitor_hide_throughput" />
                </div>

                <div
                  v-if="form.channel_monitor_mode === 'v1'"
                  class="flex items-start justify-between gap-4"
                >
                  <div>
                    <p class="font-medium text-gray-900 dark:text-white">
                      {{ t("admin.settings.features.channelMonitor.showQuota") }}
                    </p>
                    <p class="text-sm text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.features.channelMonitor.showQuotaHint") }}
                    </p>
                  </div>
                  <Toggle v-model="form.channel_monitor_show_quota" />
                </div>
              </template>
            </div>
          </div>

          <!-- Affiliate Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                推广返利
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                控制用户推广入口和默认返利规则。实际返佣入账以后端账本为准。
              </p>
            </div>
            <div class="space-y-6 p-6">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white"
                    >启用推广返利</label
                  >
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    开启后，普通用户可在侧边栏看到推广返利入口。
                  </p>
                </div>
                <Toggle v-model="form.affiliate_enabled" />
              </div>

              <div
                v-if="form.affiliate_enabled"
                class="grid grid-cols-1 gap-6 border-t border-gray-100 pt-4 md:grid-cols-2 dark:border-dark-700"
              >
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    默认返利比例（%）
                  </label>
                  <input
                    v-model.number="form.affiliate_rebate_rate"
                    type="number"
                    min="0"
                    max="100"
                    step="0.01"
                    class="input"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    没有设置专属比例的推广用户使用这个默认比例。
                  </p>
                </div>

                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    返利冻结小时
                  </label>
                  <input
                    v-model.number="form.affiliate_rebate_freeze_hours"
                    type="number"
                    min="0"
                    step="1"
                    class="input"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    0 表示不冻结，具体结算仍以后端策略为准。
                  </p>
                </div>

                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    返利有效天数
                  </label>
                  <input
                    v-model.number="form.affiliate_rebate_duration_days"
                    type="number"
                    min="0"
                    step="1"
                    class="input"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    0 表示不限制邀请关系有效期。
                  </p>
                </div>

                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    单个被邀请人返利上限
                  </label>
                  <input
                    v-model.number="form.affiliate_rebate_per_invitee_cap"
                    type="number"
                    min="0"
                    step="0.01"
                    class="input"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    0 表示不设置单人上限。
                  </p>
                </div>
              </div>
            </div>
          </div>

          <!-- Model Plaza Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                模型广场
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                向用户展示可用分组、模型与公开价格信息。
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div class="flex items-center justify-between gap-4">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white"
                    >启用模型广场</label
                  >
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    开启后在用户页头显示模型广场入口。
                  </p>
                </div>
                <Toggle v-model="form.model_plaza_enabled" />
              </div>
              <div
                class="flex items-center justify-between gap-4 border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="font-medium text-gray-900 dark:text-white"
                    >要求登录</label
                  >
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    开启后，未登录访客不能查看模型广场。
                  </p>
                </div>
                <Toggle
                  v-model="form.model_plaza_require_auth"
                  :disabled="!form.model_plaza_enabled"
                />
              </div>
              <div class="border-t border-gray-100 pt-4 dark:border-dark-700">
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  顶部说明
                </label>
                <textarea
                  v-model="form.model_plaza_description"
                  rows="4"
                  class="input min-h-[96px]"
                  placeholder="支持 Markdown，可填写计费说明、汇率或活动信息"
                ></textarea>
              </div>
            </div>
          </div>

          <!-- Authentication source defaults -->
          <div class="card" data-testid="auth-source-defaults-card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.authSourceDefaults.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.authSourceDefaults.description") }}
              </p>
            </div>
            <div class="space-y-4 p-6">
              <div
                v-for="meta in authSourceDefaultsMeta"
                :key="meta.source"
                class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
              >
                <div class="flex items-center justify-between gap-4">
                  <div>
                    <h3 class="font-medium text-gray-900 dark:text-white">
                      {{ meta.title }}
                    </h3>
                    <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                      {{ meta.description }}
                    </p>
                  </div>
                  <Toggle
                    :model-value="authSourceDefaults[meta.source].grant_on_first_bind"
                    :data-testid="
                      meta.source === 'email'
                        ? 'auth-source-email-enabled'
                        : undefined
                    "
                    @update:model-value="
                      authSourceDefaults[meta.source].grant_on_first_bind = $event
                    "
                  />
                </div>
                <div
                  v-if="authSourceDefaults[meta.source].grant_on_first_bind"
                  class="mt-4 rounded-lg bg-gray-50 p-3 text-sm dark:bg-dark-800"
                  :data-testid="
                    meta.source === 'email'
                      ? 'auth-source-email-panel'
                      : undefined
                  "
                >
                  <p class="font-medium text-gray-800 dark:text-gray-200">
                    {{ t("admin.settings.authSourceDefaults.grantOnFirstBindLabel") }}
                  </p>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.authSourceDefaults.grantOnFirstBindHint") }}
                  </p>
                </div>
              </div>
            </div>
          </div>
        </div>
        <!-- /Tab: Users -->

        <!-- Tab: Payment -->
        <div v-show="activeTab === 'payment'" class="space-y-6">
          <div class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.payment.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.payment.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div class="flex items-start justify-between gap-6">
                <div>
                  <h3 class="text-sm font-semibold text-gray-900 dark:text-white">
                    {{ t("admin.settings.payment.enabled") }}
                  </h3>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.payment.enabledHint") }}
                  </p>
                </div>
                <Toggle v-model="form.payment_enabled" />
              </div>
              <div class="grid gap-5 md:grid-cols-2">
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t("admin.settings.payment.helpImage") }}
                  </label>
        <ImageUpload
          v-model="form.payment_help_image_url"
          mode="image"
          data-placeholder="admin.settings.payment.helpImagePlaceholder"
          :upload-label="t('admin.settings.site.uploadImage')"
          :remove-label="t('admin.settings.site.remove')"
        />
                </div>
                <div>
                  <label class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t("admin.settings.payment.helpText") }}
                  </label>
                  <textarea
                    v-model="form.payment_help_text"
                    rows="5"
                    class="input min-h-[128px]"
                    :placeholder="t('admin.settings.payment.helpTextPlaceholder')"
                  ></textarea>
                  <p class="mt-2 text-xs text-gray-500 dark:text-gray-400">
                    <a
                      href="https://github.com/Wei-Shaw/sub2api/blob/main/docs/PAYMENT_CN.md"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="text-primary-500 hover:text-primary-600 dark:text-primary-400"
                    >
                      {{ t("admin.settings.payment.configGuide") }}
                    </a>
                    <span class="mx-1 text-gray-400">·</span>
                    <a
                      href="https://github.com/Wei-Shaw/sub2api/blob/main/docs/PAYMENT_CN.md#支持的支付方式"
                      target="_blank"
                      rel="noopener noreferrer"
                      class="text-primary-500 hover:text-primary-600 dark:text-primary-400"
                    >
                      {{ t("admin.settings.payment.findProvider") }}
                    </a>
                  </p>
                </div>
              </div>
            </div>
          </div>
          <PaymentProviderList
            :providers="providers"
            :loading="providersLoading"
            :can-create="true"
            :enabled-payment-types="form.payment_enabled_types || []"
            :all-payment-types="allPaymentTypes"
            :redirect-label="t('admin.settings.payment.easypayRedirect')"
            @refresh="loadPaymentProviders"
            @create="openCreateProvider"
            @edit="openEditProvider"
            @delete="confirmDeleteProvider"
            @toggle-field="toggleProviderField"
            @toggle-type="toggleProviderType"
            @reorder="reorderProviders"
          />
          <PaymentProviderDialog
            ref="providerDialogRef"
            :show="showProviderDialog"
            :saving="savingProvider"
            :editing="editingProvider"
            :all-key-options="providerKeyOptions"
            :enabled-key-options="providerKeyOptions.filter((item) => (form.payment_enabled_types ?? []).includes(item.value))"
            :all-payment-types="allPaymentTypes"
            :redirect-label="t('admin.settings.payment.easypayRedirect')"
            @close="closeProviderDialog"
            @save="saveProvider"
          />
          <ConfirmDialog
            v-if="deletingProvider"
            :show="true"
            :title="t('admin.settings.payment.deleteProvider')"
            :message="t('admin.settings.payment.deleteProviderConfirm')"
            :danger="true"
            @confirm="deleteProvider"
            @cancel="cancelDeleteProvider"
          />
        </div>

        <!-- Tab: Gateway — Claude Code, Scheduling -->
        <div v-show="activeTab === 'gateway'" class="space-y-6">
          <div class="card">
            <div class="space-y-5 p-6">
              <div class="grid gap-5 border-b border-gray-100 pb-5 dark:border-dark-700 md:grid-cols-[minmax(0,1fr)_auto] md:items-end">
                <div>
                  <label for="grok-default-text-model" class="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t("admin.settings.gatewayForwarding.grokDefaultTextModel") }}
                  </label>
                  <input
                    id="grok-default-text-model"
                    v-model.trim="form.grok_default_text_model"
                    type="text"
                    class="input mt-2 w-full"
                    data-testid="grok-default-text-model"
                    placeholder="grok-4.5"
                  />
                </div>
                <div class="flex items-center justify-between gap-5 md:min-w-72">
                  <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                    {{ t("admin.settings.gatewayForwarding.grokCrossClientMap") }}
                  </label>
                  <Toggle v-model="form.grok_cross_client_model_map_enabled" data-testid="grok-cross-client-model-map-toggle" />
                </div>
              </div>
            </div>
          </div>
          <div data-testid="upstream-billing-probe-settings" class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.upstreamBillingProbe.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.upstreamBillingProbe.description") }}
              </p>
            </div>
            <div class="space-y-4 p-6">
              <div v-if="upstreamBillingProbeLoading" class="text-sm text-gray-500">
                {{ t("common.loading") }}
              </div>
              <template v-else>
                <div class="flex items-start justify-between gap-6">
                  <div>
                    <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {{ t("admin.settings.upstreamBillingProbe.enabled") }}
                    </label>
                    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.upstreamBillingProbe.enabledHint") }}
                    </p>
                  </div>
                  <Toggle
                    data-testid="upstream-billing-probe-enabled"
                    v-model="upstreamBillingProbeForm.enabled"
                  />
                </div>
                <div v-if="upstreamBillingProbeForm.enabled" class="max-w-xs">
                  <label class="mb-1 block text-sm text-gray-700 dark:text-gray-300">
                    {{ t("admin.settings.upstreamBillingProbe.intervalMinutes") }}
                  </label>
                  <input
                    v-model.number="upstreamBillingProbeForm.interval_minutes"
                    data-testid="upstream-billing-probe-interval"
                    type="number"
                    min="5"
                    max="1440"
                    class="input"
                  />
                </div>
                <button
                  type="button"
                  data-testid="upstream-billing-probe-save"
                  class="btn btn-primary btn-sm"
                  :disabled="upstreamBillingProbeSaving"
                  @click="saveUpstreamBillingProbeSettings"
                >
                  {{ t("common.save") }}
                </button>
              </template>
            </div>
          </div>

          <div data-testid="ollama-cloud-usage-global-settings" class="card">
            <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.ollamaCloudUsage.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.ollamaCloudUsage.description") }}
              </p>
            </div>
            <div class="space-y-4 p-6">
              <div v-if="ollamaCloudUsageLoading" class="text-sm text-gray-500">
                {{ t("common.loading") }}
              </div>
              <template v-else>
                <div class="flex items-start justify-between gap-6">
                  <div>
                    <label class="text-sm font-medium text-gray-700 dark:text-gray-300">
                      {{ t("admin.settings.ollamaCloudUsage.enabled") }}
                    </label>
                    <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                      {{ t("admin.settings.ollamaCloudUsage.enabledHint") }}
                    </p>
                  </div>
                  <Toggle
                    data-testid="ollama-cloud-usage-global-enabled"
                    v-model="ollamaCloudUsageForm.enabled"
                  />
                </div>
                <div v-if="ollamaCloudUsageForm.enabled" class="grid max-w-2xl gap-4 sm:grid-cols-2">
                  <div>
                    <label class="mb-1 block text-sm text-gray-700 dark:text-gray-300">
                      {{ t("admin.settings.ollamaCloudUsage.intervalMinutes") }}
                    </label>
                    <input
                      v-model.number="ollamaCloudUsageForm.interval_minutes"
                      data-testid="ollama-cloud-usage-global-interval"
                      type="number"
                      min="5"
                      max="1440"
                      class="input"
                    />
                  </div>
                  <div>
                    <label class="mb-1 block text-sm text-gray-700 dark:text-gray-300">
                      {{ t("admin.settings.ollamaCloudUsage.debounceMinutes") }}
                    </label>
                    <input
                      v-model.number="ollamaCloudUsageForm.debounce_minutes"
                      data-testid="ollama-cloud-usage-global-debounce"
                      type="number"
                      min="0"
                      max="60"
                      class="input"
                    />
                  </div>
                </div>
                <button
                  type="button"
                  data-testid="ollama-cloud-usage-global-save"
                  class="btn btn-primary btn-sm"
                  :disabled="ollamaCloudUsageSaving"
                  @click="saveOllamaCloudUsageSettings"
                >
                  {{ t("common.save") }}
                </button>
              </template>
            </div>
          </div>

          <!-- Claude Code Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.claudeCode.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.claudeCode.description") }}
              </p>
            </div>
            <div class="p-6">
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.claudeCode.minVersion") }}
                </label>
                <input
                  v-model="form.min_claude_code_version"
                  type="text"
                  class="input max-w-xs font-mono text-sm"
                  :placeholder="
                    t('admin.settings.claudeCode.minVersionPlaceholder')
                  "
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.claudeCode.minVersionHint") }}
                </p>
              </div>
              <div class="mt-4">
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.claudeCode.maxVersion") }}
                </label>
                <input
                  v-model="form.max_claude_code_version"
                  type="text"
                  class="input max-w-xs font-mono text-sm"
                  :placeholder="
                    t('admin.settings.claudeCode.maxVersionPlaceholder')
                  "
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.claudeCode.maxVersionHint") }}
                </p>
              </div>
            </div>
          </div>

          <!-- Gateway Scheduling Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.scheduling.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.scheduling.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <div class="flex items-center justify-between">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.scheduling.allowUngroupedKey") }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.scheduling.allowUngroupedKeyHint") }}
                  </p>
                </div>
                <Toggle v-model="form.allow_ungrouped_key_scheduling" />
              </div>

              <div class="border-t border-gray-100 pt-5 dark:border-dark-700">
                <div class="mb-3">
                  <label class="font-medium text-gray-900 dark:text-white">
                    {{
                      t(
                        "admin.settings.scheduling.accountSchedulingThresholdsTitle",
                      )
                    }}
                  </label>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{
                      t(
                        "admin.settings.scheduling.accountSchedulingThresholdsDescription",
                      )
                    }}
                  </p>
                  <p class="mt-0.5 text-xs text-amber-600 dark:text-amber-400">
                    {{
                      t(
                        "admin.settings.scheduling.accountSchedulingThresholdsDisabledHint",
                      )
                    }}
                  </p>
                </div>
                <div class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
                  <div
                    v-for="platform in schedulingThresholdPlatforms"
                    :key="platform"
                    class="rounded-lg border border-gray-200 p-4 dark:border-dark-700"
                  >
                    <div class="flex items-center justify-between gap-3">
                      <label class="font-mono text-sm font-medium text-gray-900 dark:text-white">
                        {{ platform }}
                      </label>
                      <span
                        class="rounded bg-gray-100 px-2 py-0.5 text-[11px] font-medium text-gray-600 dark:bg-dark-700 dark:text-gray-300"
                      >
                        %
                      </span>
                    </div>
                    <input
                      v-model.number="form.account_scheduling_thresholds[platform]"
                      type="number"
                      min="1"
                      max="100"
                      step="1"
                      class="input mt-3"
                      :data-testid="`account-scheduling-threshold-${platform}`"
                      placeholder="100"
                    />
                  </div>
                </div>
              </div>
            </div>
          </div>

          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.cleanup.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.cleanup.description") }}
              </p>
            </div>
            <div class="space-y-4 p-6">
              <div class="flex items-center justify-between">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.cleanup.autoDelete401Accounts") }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.cleanup.autoDelete401AccountsHint") }}
                  </p>
                </div>
                <Toggle v-model="form.auto_delete_401_accounts" />
              </div>

              <div class="flex items-center justify-between">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.cleanup.autoDelete429Accounts") }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.cleanup.autoDelete429AccountsHint") }}
                  </p>
                </div>
                <Toggle v-model="form.auto_delete_429_accounts" />
              </div>

              <div class="flex items-center justify-between">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.cleanup.autoDeleteUselessProxies") }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      t("admin.settings.cleanup.autoDeleteUselessProxiesHint")
                    }}
                  </p>
                </div>
                <Toggle
                  v-model="form.enable_claude_oauth_system_prompt_injection"
                />
              </div>

              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{
                    t(
                      "admin.settings.gatewayForwarding.claudeOAuthSystemPromptBlocks",
                    )
                  }}
                </label>
                <div class="space-y-3">
                  <div
                    v-for="(block, index) in claudeOAuthSystemPromptBlocks"
                    :key="block.id"
                    class="rounded-lg border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/60"
                  >
                    <div
                      :class="[
                        'flex flex-wrap items-center justify-between gap-3',
                        block.expanded && 'mb-3',
                      ]"
                    >
                      <div class="min-w-0">
                        <div
                          class="text-sm font-medium text-gray-900 dark:text-white"
                        >
                          {{
                            t(
                              "admin.settings.gatewayForwarding.systemBlockTitle",
                              { index: index + 1 },
                            )
                          }}
                        </div>
                        <div
                          class="mt-0.5 text-xs text-gray-500 dark:text-gray-400"
                        >
                          {{ getClaudeOAuthPresetLabel(block.preset) }}
                        </div>
                      </div>
                      <div class="flex items-center gap-2">
                        <button
                          type="button"
                          class="btn btn-secondary btn-sm px-2"
                          :title="
                            block.expanded
                              ? t(
                                  'admin.settings.gatewayForwarding.systemBlockHide',
                                )
                              : t(
                                  'admin.settings.gatewayForwarding.systemBlockShow',
                                )
                          "
                          :aria-label="
                            block.expanded
                              ? t(
                                  'admin.settings.gatewayForwarding.systemBlockHide',
                                )
                              : t(
                                  'admin.settings.gatewayForwarding.systemBlockShow',
                                )
                          "
                          @click="toggleClaudeOAuthSystemPromptBlock(index)"
                        >
                          <Icon
                            :name="block.expanded ? 'eyeOff' : 'eye'"
                            size="xs"
                          />
                        </button>
                        <button
                          type="button"
                          class="btn btn-secondary btn-sm px-2"
                          :disabled="index === 0"
                          @click="moveClaudeOAuthSystemPromptBlock(index, -1)"
                        >
                          <Icon name="arrowUp" size="xs" />
                        </button>
                        <button
                          type="button"
                          class="btn btn-secondary btn-sm px-2"
                          :disabled="
                            index === claudeOAuthSystemPromptBlocks.length - 1
                          "
                          @click="moveClaudeOAuthSystemPromptBlock(index, 1)"
                        >
                          <Icon name="arrowDown" size="xs" />
                        </button>
                        <Toggle v-model="block.enabled" />
                        <button
                          type="button"
                          class="btn btn-secondary btn-sm px-2 text-red-600 hover:text-red-700 dark:text-red-400"
                          @click="removeClaudeOAuthSystemPromptBlock(index)"
                        >
                          <Icon name="trash" size="xs" />
                        </button>
                      </div>
                    </div>

                    <div v-show="block.expanded">
                      <div class="grid gap-3 md:grid-cols-2">
                        <div>
                          <label
                            class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300"
                          >
                            {{
                              t(
                                "admin.settings.gatewayForwarding.systemBlockPreset",
                              )
                            }}
                          </label>
                          <Select
                            v-model="block.preset"
                            :options="claudeOAuthSystemPromptPresetOptions"
                            @change="
                              (value) =>
                                applyClaudeOAuthSystemPromptPreset(index, value)
                            "
                          />
                        </div>
                        <div>
                          <label
                            class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300"
                          >
                            {{
                              t(
                                "admin.settings.gatewayForwarding.systemBlockType",
                              )
                            }}
                          </label>
                          <Select
                            v-model="block.type"
                            :options="claudeOAuthSystemPromptBlockTypeOptions"
                          />
                        </div>
                      </div>

                      <div class="mt-3">
                        <label
                          class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300"
                        >
                          {{ t("admin.settings.gatewayForwarding.systemBlockText") }}
                        </label>
                        <textarea
                          v-model="block.text"
                          rows="6"
                          class="input w-full resize-y font-mono text-xs leading-5"
                          @input="markClaudeOAuthSystemPromptBlockCustom(block)"
                        />
                      </div>

                      <div
                        class="mt-3 grid gap-3 md:grid-cols-[minmax(0,1fr)_160px]"
                      >
                        <div class="flex items-center justify-between gap-4">
                          <div>
                            <label
                              class="text-xs font-medium text-gray-600 dark:text-gray-300"
                            >
                              {{
                                t(
                                  "admin.settings.gatewayForwarding.systemBlockCacheControl",
                                )
                              }}
                            </label>
                          </div>
                          <Toggle v-model="block.cacheControlEnabled" />
                        </div>
                        <div v-if="block.cacheControlEnabled">
                          <Select
                            v-model="block.cacheControlTTL"
                            :options="claudeOAuthSystemPromptCacheTTLOptions"
                          />
                        </div>
                      </div>
                    </div>
                  </div>
                </div>

                <div class="mt-3 flex flex-wrap gap-2">
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm"
                    @click="addClaudeOAuthSystemPromptBlock"
                  >
                    <Icon name="plus" size="xs" />
                    {{ t("admin.settings.gatewayForwarding.addSystemBlock") }}
                  </button>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm"
                    @click="resetClaudeOAuthSystemPromptBlocks"
                  >
                    <Icon name="refresh" size="xs" />
                    {{
                      t("admin.settings.gatewayForwarding.resetSystemBlocks")
                    }}
                  </button>
                </div>
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{
                    t(
                      "admin.settings.gatewayForwarding.claudeOAuthSystemPromptBlocksHint",
                    )
                  }}
                </p>
              </div>

              <!-- Anthropic Cache TTL 1h Injection -->
              <div class="flex items-center justify-between">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{
                      t(
                        "admin.settings.gatewayForwarding.anthropicCacheTTL1hInjection",
                      )
                    }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      t(
                        "admin.settings.gatewayForwarding.anthropicCacheTTL1hInjectionHint",
                      )
                    }}
                  </p>
                </div>
                <Toggle
                  v-model="form.enable_anthropic_cache_ttl_1h_injection"
                />
              </div>

              <!-- messages cache_control 改写 -->
              <div class="flex items-center justify-between">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{
                      t(
                        "admin.settings.gatewayForwarding.rewriteMessageCacheControl",
                      )
                    }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      t(
                        "admin.settings.gatewayForwarding.rewriteMessageCacheControlHint",
                      )
                    }}
                  </p>
                </div>
                <Toggle v-model="form.rewrite_message_cache_control" />
              </div>

              <!-- 客户端 dateline 归一化（仅 Anthropic OAuth/SetupToken） -->
              <div class="flex items-center justify-between">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{
                      t(
                        "admin.settings.gatewayForwarding.clientDatelineNormalization",
                      )
                    }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      t(
                        "admin.settings.gatewayForwarding.clientDatelineNormalizationHint",
                      )
                    }}
                  </p>
                </div>
                <Toggle
                  v-model="form.enable_client_dateline_normalization"
                />
              </div>

              <!-- Antigravity UA 版本 -->
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{
                    t(
                      "admin.settings.gatewayForwarding.antigravityUserAgentVersion",
                    )
                  }}
                </label>
                <input
                  v-model="form.antigravity_user_agent_version"
                  type="text"
                  class="input max-w-xs font-mono text-sm"
                  :placeholder="
                    t(
                      'admin.settings.gatewayForwarding.antigravityUserAgentVersionPlaceholder',
                    )
                  "
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{
                    t(
                      "admin.settings.gatewayForwarding.antigravityUserAgentVersionHint",
                    )
                  }}
                </p>
              </div>

              <!-- OpenAI Codex UA -->
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{
                    t(
                      "admin.settings.gatewayForwarding.openaiCodexUserAgent",
                    )
                  }}
                </label>
                <input
                  v-model="form.openai_codex_user_agent"
                  type="text"
                  class="input w-full font-mono text-sm"
                  :placeholder="
                    t(
                      'admin.settings.gatewayForwarding.openaiCodexUserAgentPlaceholder',
                    )
                  "
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{
                    t(
                      "admin.settings.gatewayForwarding.openaiCodexUserAgentHint",
                    )
                  }}
                </p>
              </div>

              <!-- Codex 客户端版本号 -->
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{
                    t(
                      "admin.settings.gatewayForwarding.openaiCodexClientVersion",
                    )
                  }}
                </label>
                <input
                  v-model="form.openai_codex_client_version"
                  type="text"
                  class="input w-full font-mono text-sm"
                  :placeholder="
                    t(
                      'admin.settings.gatewayForwarding.openaiCodexClientVersionPlaceholder',
                    )
                  "
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{
                    t(
                      "admin.settings.gatewayForwarding.openaiCodexClientVersionHint",
                    )
                  }}
                </p>
              </div>

              <!-- Codex 版本号自动同步 -->
              <div class="flex items-center justify-between">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{
                      t(
                        "admin.settings.gatewayForwarding.openaiCodexVersionAutoSync",
                      )
                    }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      t(
                        "admin.settings.gatewayForwarding.openaiCodexVersionAutoSyncHint",
                      )
                    }}
                  </p>
                  <p
                    v-if="codexSyncedVersionLabel"
                    class="mt-0.5 text-xs text-gray-500 dark:text-gray-400"
                  >
                    {{ codexSyncedVersionLabel }}
                  </p>
                </div>
                <Toggle v-model="form.openai_codex_version_auto_sync_enabled" />
              </div>

            </div>
          </div>

          <!-- Web Search Emulation -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.webSearchEmulation.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.webSearchEmulation.description") }}
              </p>
            </div>
            <div class="space-y-5 p-6">
              <!-- Global Toggle -->
              <div class="flex items-center justify-between">
                <div>
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.webSearchEmulation.enabled") }}
                  </label>
                  <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.webSearchEmulation.enabledHint") }}
                  </p>
                </div>
                <Toggle v-model="webSearchConfig.enabled" />
              </div>

              <!-- Providers -->
              <div v-if="webSearchConfig.enabled" class="space-y-4">
                <div class="flex items-center justify-between">
                  <label
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.webSearchEmulation.providers") }}
                  </label>
                  <button
                    type="button"
                    class="btn btn-secondary btn-sm"
                    @click="addWebSearchProvider"
                  >
                    {{ t("admin.settings.webSearchEmulation.addProvider") }}
                  </button>
                </div>

                <div
                  v-if="webSearchConfig.providers.length === 0"
                  class="rounded-lg border border-dashed border-gray-300 p-4 text-center text-sm text-gray-400 dark:border-dark-600"
                >
                  {{ t("admin.settings.webSearchEmulation.noProviders") }}
                </div>

                <div
                  v-for="(provider, pIdx) in webSearchConfig.providers"
                  :key="pIdx"
                  class="rounded-lg border border-gray-200 dark:border-dark-600"
                >
                  <!-- Collapsible header -->
                  <div
                    class="flex cursor-pointer items-center justify-between px-4 py-3"
                    @click="toggleProviderExpand(pIdx)"
                  >
                    <div class="flex items-center gap-3">
                      <svg
                        class="h-4 w-4 text-gray-400 transition-transform"
                        :class="{ 'rotate-90': expandedProviders[pIdx] }"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          stroke-width="2"
                          d="M9 5l7 7-7 7"
                        />
                      </svg>
                      <Select
                        v-model="provider.type"
                        :options="[
                          { value: 'brave', label: 'Brave Search' },
                          { value: 'tavily', label: 'Tavily' },
                        ]"
                        class="w-36"
                        @click.stop
                      />
                      <!-- Quota summary (always visible) -->
                      <span class="text-xs text-gray-400">
                        {{ provider.quota_used ?? 0 }} /
                        {{
                          provider.quota_limit != null &&
                          provider.quota_limit > 0
                            ? provider.quota_limit
                            : "∞"
                        }}
                      </span>
                      <span
                        v-if="
                          !expandedProviders[pIdx] &&
                          provider.api_key_configured
                        "
                        class="text-xs text-green-500"
                      >
                        {{
                          t(
                            "admin.settings.webSearchEmulation.apiKeyConfigured",
                          )
                        }}
                      </span>
                    </div>
                    <button
                      type="button"
                      class="text-red-500 hover:text-red-700 text-xs"
                      @click.stop="removeWebSearchProvider(pIdx)"
                    >
                      {{
                        t("admin.settings.webSearchEmulation.removeProvider")
                      }}
                    </button>
                  </div>

                  <!-- Expanded content -->
                  <div
                    v-if="expandedProviders[pIdx]"
                    class="space-y-3 border-t border-gray-100 px-4 pb-4 pt-3 dark:border-dark-700"
                  >
                    <!-- API Key with inline show/copy -->
                    <div>
                      <label class="text-xs text-gray-500">{{
                        t("admin.settings.webSearchEmulation.apiKey")
                      }}</label>
                      <div class="relative">
                        <input
                          v-model="provider.api_key"
                          :type="apiKeyVisible[pIdx] ? 'text' : 'password'"
                          class="input w-full text-sm"
                          :class="
                            provider.api_key || provider.api_key_configured
                              ? 'pr-16'
                              : ''
                          "
                          :placeholder="
                            provider.api_key_configured
                              ? '••••••••'
                              : t(
                                  'admin.settings.webSearchEmulation.apiKeyPlaceholder',
                                )
                          "
                        />
                        <div
                          v-if="provider.api_key || provider.api_key_configured"
                          class="absolute inset-y-0 right-0 flex items-center pr-1.5"
                        >
                          <button
                            type="button"
                            class="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            :title="
                              apiKeyVisible[pIdx]
                                ? t(
                                    'admin.settings.webSearchEmulation.hideApiKey',
                                  )
                                : t(
                                    'admin.settings.webSearchEmulation.showApiKey',
                                  )
                            "
                            @click="apiKeyVisible[pIdx] = !apiKeyVisible[pIdx]"
                          >
                            <svg
                              v-if="!apiKeyVisible[pIdx]"
                              class="h-4 w-4"
                              fill="none"
                              viewBox="0 0 24 24"
                              stroke="currentColor"
                            >
                              <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M15 12a3 3 0 11-6 0 3 3 0 016 0z"
                              />
                              <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z"
                              />
                            </svg>
                            <svg
                              v-else
                              class="h-4 w-4"
                              fill="none"
                              viewBox="0 0 24 24"
                              stroke="currentColor"
                            >
                              <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.878 9.878L3 3m6.878 6.878L21 21"
                              />
                            </svg>
                          </button>
                          <button
                            type="button"
                            class="rounded p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300"
                            :class="{
                              'opacity-30 cursor-not-allowed':
                                !provider.api_key,
                            }"
                            :title="
                              t('admin.settings.webSearchEmulation.copyApiKey')
                            "
                            :disabled="!provider.api_key"
                            @click="copyApiKey(pIdx)"
                          >
                            <svg
                              class="h-4 w-4"
                              fill="none"
                              viewBox="0 0 24 24"
                              stroke="currentColor"
                            >
                              <path
                                stroke-linecap="round"
                                stroke-linejoin="round"
                                stroke-width="2"
                                d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"
                              />
                            </svg>
                          </button>
                        </div>
                      </div>
                    </div>

                    <!-- Quota + Subscription in compact row -->
                    <div class="grid grid-cols-2 gap-3">
                      <div>
                        <label class="text-xs text-gray-500">{{
                          t("admin.settings.webSearchEmulation.quotaLimit")
                        }}</label>
                        <input
                          v-model="provider.quota_limit"
                          type="number"
                          min="1"
                          class="input text-sm"
                          :placeholder="'∞'"
                        />
                        <p class="mt-0.5 text-xs text-gray-400">
                          {{
                            t(
                              "admin.settings.webSearchEmulation.quotaLimitHint",
                            )
                          }}
                        </p>
                      </div>
                      <div>
                        <label class="text-xs text-gray-500">{{
                          t("admin.settings.webSearchEmulation.subscribedAt")
                        }}</label>
                        <input
                          :value="formatSubscribedAt(provider.subscribed_at)"
                          type="date"
                          class="input text-sm"
                          @input="
                            provider.subscribed_at = parseSubscribedAt(
                              ($event.target as HTMLInputElement).value,
                            )
                          "
                        />
                        <p class="mt-0.5 text-xs text-gray-400">
                          {{
                            t(
                              "admin.settings.webSearchEmulation.subscribedAtHint",
                            )
                          }}
                        </p>
                      </div>
                    </div>

                    <!-- Usage display -->
                    <div class="flex items-center gap-2">
                      <span class="text-xs text-gray-500"
                        >{{
                          t("admin.settings.webSearchEmulation.quotaUsage")
                        }}:</span
                      >
                      <div
                        v-if="
                          provider.quota_limit != null &&
                          provider.quota_limit > 0
                        "
                        class="flex-1 rounded-full bg-gray-200 dark:bg-dark-600"
                        style="height: 6px"
                      >
                        <div
                          class="h-full rounded-full transition-all"
                          :class="
                            quotaPercentage(provider) > 90
                              ? 'bg-red-500'
                              : quotaPercentage(provider) > 70
                                ? 'bg-yellow-500'
                                : 'bg-green-500'
                          "
                          :style="{
                            width:
                              Math.min(quotaPercentage(provider), 100) + '%',
                          }"
                        />
                      </div>
                      <div v-else class="flex-1" />
                      <span class="text-xs text-gray-500"
                        >{{ provider.quota_used ?? 0 }} /
                        {{
                          provider.quota_limit != null &&
                          provider.quota_limit > 0
                            ? provider.quota_limit
                            : "∞"
                        }}</span
                      >
                      <button
                        v-if="(provider.quota_used ?? 0) > 0"
                        type="button"
                        class="text-xs text-primary-600 hover:text-primary-700"
                        @click="resetWebSearchUsage(pIdx)"
                      >
                        {{ t("admin.settings.webSearchEmulation.resetUsage") }}
                      </button>
                    </div>

                    <!-- Proxy + Test on same row -->
                    <div class="flex items-end gap-3">
                      <div class="flex-1">
                        <label class="text-xs text-gray-500">{{
                          t("admin.settings.webSearchEmulation.proxy")
                        }}</label>
                        <ProxySelector
                          v-model="provider.proxy_id"
                          :proxies="webSearchProxies"
                        />
                      </div>
                      <button
                        type="button"
                        class="btn btn-secondary btn-sm whitespace-nowrap"
                        @click="openTestDialog()"
                      >
                        {{ t("admin.settings.webSearchEmulation.test") }}
                      </button>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
        <!-- /Tab: Gateway — Claude Code, Scheduling -->

        <!-- Tab: General -->
        <div v-show="activeTab === 'general'" class="space-y-6">
          <!-- Site Settings -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.site.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.site.description") }}
              </p>
            </div>
            <div class="space-y-6 p-6">
              <!-- Backend Mode -->
              <div
                class="flex items-center justify-between rounded-lg border border-amber-200 bg-amber-50 p-4 dark:border-amber-800 dark:bg-amber-900/20"
              >
                <div>
                  <h3 class="text-sm font-medium text-gray-900 dark:text-white">
                    {{ t("admin.settings.site.backendMode") }}
                  </h3>
                  <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.site.backendModeDescription") }}
                  </p>
                </div>
                <Toggle v-model="form.backend_mode_enabled" />
              </div>

              <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.site.siteName") }}
                  </label>
                  <input
                    v-model="form.site_name"
                    type="text"
                    class="input"
                    :placeholder="t('admin.settings.site.siteNamePlaceholder')"
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.site.siteNameHint") }}
                  </p>
                </div>
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.site.siteSubtitle") }}
                  </label>
                  <input
                    v-model="form.site_subtitle"
                    type="text"
                    class="input"
                    :placeholder="
                      t('admin.settings.site.siteSubtitlePlaceholder')
                    "
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.site.siteSubtitleHint") }}
                  </p>
                </div>
              </div>

              <!-- API Base URL -->
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.site.apiBaseUrl") }}
                </label>
                <input
                  v-model="form.api_base_url"
                  type="text"
                  class="input font-mono text-sm"
                  :placeholder="t('admin.settings.site.apiBaseUrlPlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.site.apiBaseUrlHint") }}
                </p>
              </div>

              <!-- Contact Info -->
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.site.contactInfo") }}
                </label>
                <input
                  v-model="form.contact_info"
                  type="text"
                  class="input"
                  data-testid="settings-contact-info-input"
                  :placeholder="t('admin.settings.site.contactInfoPlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.site.contactInfoHint") }}
                </p>
                <!-- Empty contact info leaves locked-out users without any appeal channel -->
                <p
                  v-if="contactInfoMissing"
                  data-testid="settings-contact-info-missing-hint"
                  class="mt-2 text-xs text-amber-600 dark:text-amber-400"
                >
                  {{ t("admin.settings.site.contactInfoMissingHint") }}
                </p>
              </div>

              <!-- Doc URL -->
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.site.docUrl") }}
                </label>
                <input
                  v-model="form.doc_url"
                  type="url"
                  class="input font-mono text-sm"
                  :placeholder="t('admin.settings.site.docUrlPlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.site.docUrlHint") }}
                </p>
              </div>

              <!-- Site Logo Upload -->
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.site.siteLogo") }}
                </label>
                <ImageUpload
                  v-model="form.site_logo"
                  mode="image"
                  :upload-label="t('admin.settings.site.uploadImage')"
                  :remove-label="t('admin.settings.site.remove')"
                  :hint="t('admin.settings.site.logoHint')"
                  :max-size="300 * 1024"
                />
              </div>

              <!-- Home Content -->
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.site.homeContent") }}
                </label>
                <textarea
                  v-model="form.home_content"
                  rows="6"
                  class="input font-mono text-sm"
                  :placeholder="t('admin.settings.site.homeContentPlaceholder')"
                ></textarea>
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.site.homeContentHint") }}
                </p>
                <!-- iframe CSP Warning -->
                <p class="mt-2 text-xs text-amber-600 dark:text-amber-400">
                  {{ t("admin.settings.site.homeContentIframeWarning") }}
                </p>
              </div>

              <!-- Compact Home Page -->
              <div class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.site.compactHome")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.site.compactHomeHint") }}
                  </p>
                </div>
                <Toggle v-model="form.compact_home_enabled" data-testid="compact-home-toggle" />
              </div>

              <!-- Hide CCS Import Button -->
              <div
                class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.site.hideCcsImportButton")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.site.hideCcsImportButtonHint") }}
                  </p>
                </div>
                <Toggle v-model="form.hide_ccs_import_button" />
              </div>
            </div>
          </div>

          <!-- Purchase Subscription Page -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.purchase.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.purchase.description") }}
              </p>
            </div>
            <div class="space-y-6 p-6">
              <!-- Enable Toggle -->
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.purchase.enabled")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.purchase.enabledHint") }}
                  </p>
                </div>
                <Toggle v-model="form.purchase_subscription_enabled" />
              </div>

              <!-- URL -->
              <div>
                <label
                  class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.purchase.url") }}
                </label>
                <input
                  v-model="form.purchase_subscription_url"
                  type="url"
                  class="input font-mono text-sm"
                  :placeholder="t('admin.settings.purchase.urlPlaceholder')"
                />
                <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.purchase.urlHint") }}
                </p>
                <p class="mt-2 text-xs text-amber-600 dark:text-amber-400">
                  {{ t("admin.settings.purchase.iframeWarning") }}
                </p>
              </div>

              <div class="grid gap-4 md:grid-cols-3">
                <label
                  class="block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.purchase.xianyu10") }}
                  <input
                    v-model="form.purchase_link_cny_10"
                    type="url"
                    class="input mt-2 font-mono text-sm"
                    :placeholder="
                      t('admin.settings.purchase.xianyuPlaceholder')
                    "
                  />
                </label>
                <label
                  class="block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.purchase.xianyu30") }}
                  <input
                    v-model="form.purchase_link_cny_30"
                    type="url"
                    class="input mt-2 font-mono text-sm"
                    :placeholder="
                      t('admin.settings.purchase.xianyuPlaceholder')
                    "
                  />
                </label>
                <label
                  class="block text-sm font-medium text-gray-700 dark:text-gray-300"
                >
                  {{ t("admin.settings.purchase.xianyu100") }}
                  <input
                    v-model="form.purchase_link_cny_100"
                    type="url"
                    class="input mt-2 font-mono text-sm"
                    :placeholder="
                      t('admin.settings.purchase.xianyuPlaceholder')
                    "
                  />
                </label>
              </div>

              <!-- Integration Docs -->
              <div class="flex items-center gap-2 text-sm">
                <svg
                  xmlns="http://www.w3.org/2000/svg"
                  class="h-4 w-4 shrink-0 text-gray-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    stroke-width="2"
                    d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
                  />
                </svg>
                <a
                  href="https://raw.githubusercontent.com/DR-lin-eng/sub2api/main/docs/ADMIN_PAYMENT_INTEGRATION_API.md"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="text-blue-600 hover:underline dark:text-blue-400"
                  download="ADMIN_PAYMENT_INTEGRATION_API.md"
                >
                  {{ t("admin.settings.purchase.integrationDoc") }}
                </a>
                <span class="text-gray-400 dark:text-gray-500">—</span>
                <span class="text-xs text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.purchase.integrationDocHint") }}
                </span>
              </div>
            </div>
          </div>

          <!-- Sora Client Toggle -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.soraClient.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.soraClient.description") }}
              </p>
            </div>
            <div class="space-y-6 p-6">
              <div class="flex items-center justify-between">
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.soraClient.enabled")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.soraClient.enabledHint") }}
                  </p>
                </div>
                <Toggle v-model="form.sora_client_enabled" />
              </div>
            </div>
          </div>

          <!-- Custom Menu Items -->
          <div class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.customMenu.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.customMenu.description") }}
              </p>
            </div>
            <div class="space-y-4 p-6">
              <!-- Existing menu items -->
              <div
                v-for="(item, index) in form.custom_menu_items"
                :key="item.id || index"
                class="rounded-lg border border-gray-200 p-4 dark:border-dark-600"
              >
                <div class="mb-3 flex items-center justify-between">
                  <span
                    class="text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{
                      t("admin.settings.customMenu.itemLabel", { n: index + 1 })
                    }}
                  </span>
                  <div class="flex items-center gap-2">
                    <!-- Move up -->
                    <LiquidButton
                      v-if="index > 0"
                      type="button"
                      class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
                      :title="t('admin.settings.customMenu.moveUp')"
                      @click="moveMenuItem(index, -1)"
                      variant="plain"
                      size="icon"
                    >
                      <svg
                        class="h-4 w-4"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          d="M5 15l7-7 7 7"
                        />
                      </svg>
                    </LiquidButton>
                    <!-- Move down -->
                    <LiquidButton
                      v-if="index < form.custom_menu_items.length - 1"
                      type="button"
                      class="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700"
                      :title="t('admin.settings.customMenu.moveDown')"
                      @click="moveMenuItem(index, 1)"
                      variant="plain"
                      size="icon"
                    >
                      <svg
                        class="h-4 w-4"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          d="M19 9l-7 7-7-7"
                        />
                      </svg>
                    </LiquidButton>
                    <!-- Delete -->
                    <LiquidButton
                      type="button"
                      class="rounded p-1 text-red-400 hover:bg-red-50 hover:text-red-600 dark:hover:bg-red-900/20"
                      :title="t('admin.settings.customMenu.remove')"
                      @click="removeMenuItem(index)"
                      variant="plain"
                      size="icon"
                    >
                      <svg
                        class="h-4 w-4"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        stroke-width="2"
                      >
                        <path
                          stroke-linecap="round"
                          stroke-linejoin="round"
                          d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"
                        />
                      </svg>
                    </LiquidButton>
                  </div>
                </div>

                <div class="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <!-- Label -->
                  <div>
                    <label
                      class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                    >
                      {{ t("admin.settings.customMenu.name") }}
                    </label>
                    <input
                      v-model="item.label"
                      type="text"
                      class="input text-sm"
                      :placeholder="
                        t('admin.settings.customMenu.namePlaceholder')
                      "
                    />
                  </div>

                  <!-- Visibility -->
                  <div>
                    <label
                      class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                    >
                      {{ t("admin.settings.customMenu.visibility") }}
                    </label>
                    <select v-model="item.visibility" class="input text-sm">
                      <option value="user">
                        {{ t("admin.settings.customMenu.visibilityUser") }}
                      </option>
                      <option value="admin">
                        {{ t("admin.settings.customMenu.visibilityAdmin") }}
                      </option>
                    </select>
                  </div>

                  <!-- URL (full width) -->
                  <div class="sm:col-span-2">
                    <label
                      class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                    >
                      {{ t("admin.settings.customMenu.url") }}
                    </label>
                    <input
                      v-model="item.url"
                      type="url"
                      class="input font-mono text-sm"
                      :placeholder="
                        t('admin.settings.customMenu.urlPlaceholder')
                      "
                    />
                  </div>

                  <!-- SVG Icon (full width) -->
                  <div class="sm:col-span-2">
                    <label
                      class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-400"
                    >
                      {{ t("admin.settings.customMenu.iconSvg") }}
                    </label>
                    <ImageUpload
                      :model-value="item.icon_svg"
                      mode="svg"
                      size="sm"
                      :upload-label="t('admin.settings.customMenu.uploadSvg')"
                      :remove-label="t('admin.settings.customMenu.removeSvg')"
                      @update:model-value="(v: string) => (item.icon_svg = v)"
                    />
                  </div>
                </div>
              </div>

              <!-- Add button -->
              <LiquidButton
                type="button"
                class="flex w-full items-center justify-center gap-2 rounded-lg border-2 border-dashed border-gray-300 py-3 text-sm text-gray-500 transition-colors hover:border-primary-400 hover:text-primary-600 dark:border-dark-600 dark:text-gray-400 dark:hover:border-primary-500 dark:hover:text-primary-400"
                @click="addMenuItem"
                variant="plain"
                size="sm"
              >
                <svg
                  class="h-4 w-4"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  stroke-width="2"
                >
                  <path
                    stroke-linecap="round"
                    stroke-linejoin="round"
                    d="M12 4v16m8-8H4"
                  />
                </svg>
                {{ t("admin.settings.customMenu.add") }}
              </LiquidButton>
            </div>
          </div>
        </div>
        <!-- /Tab: General -->

        <!-- Tab: Email -->
        <div v-show="activeTab === 'email'" class="space-y-6">
          <!-- Email disabled hint - show when email_verify_enabled is off -->
          <div v-if="!form.email_verify_enabled" class="card">
            <div class="p-6">
              <div class="flex items-start gap-3">
                <Icon
                  name="mail"
                  size="md"
                  class="mt-0.5 flex-shrink-0 text-gray-400 dark:text-gray-500"
                />
                <div>
                  <h3 class="font-medium text-gray-900 dark:text-white">
                    {{ t("admin.settings.emailTabDisabledTitle") }}
                  </h3>
                  <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.emailTabDisabledHint") }}
                  </p>
                </div>
              </div>
            </div>
          </div>

          <!-- SMTP Settings - Only show when email verification is enabled -->
          <div v-if="form.email_verify_enabled" class="card">
            <div
              class="flex items-center justify-between border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <div>
                <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                  {{ t("admin.settings.smtp.title") }}
                </h2>
                <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                  {{ t("admin.settings.smtp.description") }}
                </p>
              </div>
              <LiquidButton
                type="button"
                @click="testSmtpConnection"
                :disabled="testingSmtp"
                variant="outline"
                size="sm"
              >
                <svg
                  v-if="testingSmtp"
                  class="h-4 w-4 animate-spin"
                  fill="none"
                  viewBox="0 0 24 24"
                >
                  <circle
                    class="opacity-25"
                    cx="12"
                    cy="12"
                    r="10"
                    stroke="currentColor"
                    stroke-width="4"
                  ></circle>
                  <path
                    class="opacity-75"
                    fill="currentColor"
                    d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                  ></path>
                </svg>
                {{
                  testingSmtp
                    ? t("admin.settings.smtp.testing")
                    : t("admin.settings.smtp.testConnection")
                }}
              </LiquidButton>
            </div>
            <div class="space-y-6 p-6">
              <div class="grid grid-cols-1 gap-6 md:grid-cols-2">
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.smtp.host") }}
                  </label>
                  <input
                    v-model="form.smtp_host"
                    type="text"
                    class="input"
                    :placeholder="t('admin.settings.smtp.hostPlaceholder')"
                  />
                </div>
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.smtp.port") }}
                  </label>
                  <input
                    v-model.number="form.smtp_port"
                    type="number"
                    min="1"
                    max="65535"
                    class="input"
                    :placeholder="t('admin.settings.smtp.portPlaceholder')"
                  />
                </div>
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.smtp.username") }}
                  </label>
                  <input
                    v-model="form.smtp_username"
                    type="text"
                    class="input"
                    :placeholder="t('admin.settings.smtp.usernamePlaceholder')"
                  />
                </div>
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.smtp.password") }}
                  </label>
                  <input
                    v-model="form.smtp_password"
                    type="password"
                    class="input"
                    :placeholder="
                      form.smtp_password_configured
                        ? t('admin.settings.smtp.passwordConfiguredPlaceholder')
                        : t('admin.settings.smtp.passwordPlaceholder')
                    "
                  />
                  <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                    {{
                      form.smtp_password_configured
                        ? t("admin.settings.smtp.passwordConfiguredHint")
                        : t("admin.settings.smtp.passwordHint")
                    }}
                  </p>
                </div>
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.smtp.fromEmail") }}
                  </label>
                  <input
                    v-model="form.smtp_from_email"
                    type="email"
                    class="input"
                    :placeholder="t('admin.settings.smtp.fromEmailPlaceholder')"
                  />
                </div>
                <div>
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.smtp.fromName") }}
                  </label>
                  <input
                    v-model="form.smtp_from_name"
                    type="text"
                    class="input"
                    :placeholder="t('admin.settings.smtp.fromNamePlaceholder')"
                  />
                </div>
              </div>

              <!-- Use TLS Toggle -->
              <div
                class="flex items-center justify-between border-t border-gray-100 pt-4 dark:border-dark-700"
              >
                <div>
                  <label class="font-medium text-gray-900 dark:text-white">{{
                    t("admin.settings.smtp.useTls")
                  }}</label>
                  <p class="text-sm text-gray-500 dark:text-gray-400">
                    {{ t("admin.settings.smtp.useTlsHint") }}
                  </p>
                </div>
                <Toggle v-model="form.smtp_use_tls" />
              </div>
            </div>
          </div>

          <!-- Send Test Email - Only show when email verification is enabled -->
          <div v-if="form.email_verify_enabled" class="card">
            <div
              class="border-b border-gray-100 px-6 py-4 dark:border-dark-700"
            >
              <h2 class="text-lg font-semibold text-gray-900 dark:text-white">
                {{ t("admin.settings.testEmail.title") }}
              </h2>
              <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
                {{ t("admin.settings.testEmail.description") }}
              </p>
            </div>
            <div class="p-6">
              <div class="flex items-end gap-4">
                <div class="flex-1">
                  <label
                    class="mb-2 block text-sm font-medium text-gray-700 dark:text-gray-300"
                  >
                    {{ t("admin.settings.testEmail.recipientEmail") }}
                  </label>
                  <input
                    v-model="testEmailAddress"
                    type="email"
                    class="input"
                    :placeholder="
                      t('admin.settings.testEmail.recipientEmailPlaceholder')
                    "
                  />
                </div>
                <LiquidButton
                  type="button"
                  @click="sendTestEmail"
                  :disabled="sendingTestEmail || !testEmailAddress"
                  variant="outline"
                  size="sm"
                >
                  <svg
                    v-if="sendingTestEmail"
                    class="h-4 w-4 animate-spin"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      class="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      stroke-width="4"
                    ></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  {{
                    sendingTestEmail
                      ? t("admin.settings.testEmail.sending")
                      : t("admin.settings.testEmail.sendTestEmail")
                  }}
                </LiquidButton>
              </div>
            </div>
          </div>
        </div>
        <!-- /Tab: Email -->

        <!-- Tab: Backup -->
        <div v-show="activeTab === 'backup'">
          <BackupSettings />
        </div>

        <!-- Tab: Data Management -->
        <div v-show="activeTab === 'data'">
          <DataManagementSettings />
        </div>

        <!-- Save Button -->
        <div
          v-show="activeTab !== 'backup' && activeTab !== 'data'"
          class="flex justify-end"
        >
          <LiquidButton type="submit" :disabled="saving" size="default">
            <svg
              v-if="saving"
              class="h-4 w-4 animate-spin"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                class="opacity-25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              ></circle>
              <path
                class="opacity-75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              ></path>
            </svg>
            {{
              saving
                ? t("admin.settings.saving")
                : t("admin.settings.saveSettings")
            }}
          </LiquidButton>
        </div>
      </form>

      <div
        v-if="wsTestDialogOpen"
        class="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4"
        @click.self="wsTestDialogOpen = false"
      >
        <div class="w-full max-w-2xl rounded-xl bg-white p-5 shadow-xl dark:bg-dark-800">
          <div class="mb-4 flex items-center justify-between">
            <h3 class="text-lg font-semibold">{{ t("admin.settings.webSearchEmulation.test") }}</h3>
            <button type="button" class="text-gray-400 hover:text-gray-600" @click="wsTestDialogOpen = false">×</button>
          </div>
          <div class="flex gap-2">
            <input
              v-model="wsTestQuery"
              class="input flex-1"
              :placeholder="t('admin.settings.webSearchEmulation.testDefaultQuery')"
              @keyup.enter="testWebSearchProvider"
            />
            <LiquidButton type="button" :disabled="wsTestLoading" @click="testWebSearchProvider">
              {{ t("admin.settings.webSearchEmulation.test") }}
            </LiquidButton>
          </div>
          <div v-if="wsTestResult" class="mt-4 max-h-80 space-y-3 overflow-y-auto">
            <div
              v-for="result in wsTestResult.results"
              :key="result.url"
              class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"
            >
              <a :href="result.url" target="_blank" rel="noopener noreferrer" class="font-medium text-primary-600 hover:underline">
                {{ result.title }}
              </a>
              <p class="mt-1 text-sm text-gray-600 dark:text-gray-300">{{ result.snippet }}</p>
            </div>
          </div>
        </div>
      </div>
    </div>
    <TotpStepUpDialog :controller="stepUp" />
  </AppLayout>
</template>

<script setup lang="ts">
import LiquidButton from "@/components/common/LiquidButton.vue";
import { ref, reactive, computed, onMounted, nextTick } from "vue";
import { useI18n } from "vue-i18n";
import { adminAPI } from "@/api";
import {
  appendAuthSourceDefaultsToUpdateRequest,
  buildAuthSourceDefaultsState,
  defaultWeChatConnectScopesForMode,
  deriveWeChatConnectStoredMode,
  normalizeAccountSchedulingThresholdsMap,
  normalizePlatformQuotasMap,
  resolveWeChatConnectModeCapabilities,
  sanitizeAccountSchedulingThresholdsMap,
  sanitizePlatformQuotasMap,
  SCHEDULING_THRESHOLD_PLATFORMS,
} from "@/api/admin/settings";
import type {
  AuthSourceDefaultsState,
  AuthSourceType,
  SystemSettings,
  UpdateSettingsRequest,
  DefaultSubscriptionSetting,
  OpenAIFastPolicyRule,
  PlatformType,
  TLSFingerprintProfile,
  WebSearchEmulationConfig,
  WebSearchProviderConfig,
  WebSearchTestResult,
  WeChatConnectMode,
} from "@/api/admin/settings";
import type {
  AdminGroup,
  LoginAgreementDocument,
  NotifyEmailEntry,
  Proxy,
} from "@/types";
import AppLayout from "@/components/layout/AppLayout.vue";
import AdminPageHeader from "@/components/admin/AdminPageHeader.vue";
import Icon from "@/components/icons/Icon.vue";
import Select from "@/components/common/Select.vue";
import GroupBadge from "@/components/common/GroupBadge.vue";
import GroupOptionItem from "@/components/common/GroupOptionItem.vue";
import Toggle from "@/components/common/Toggle.vue";
import ImageUpload from "@/components/common/ImageUpload.vue";
import ConfirmDialog from "@/components/common/ConfirmDialog.vue";
import PaymentProviderDialog from "@/components/payment/PaymentProviderDialog.vue";
import PaymentProviderList from "@/components/payment/PaymentProviderList.vue";
import TotpStepUpDialog from "@/components/auth/TotpStepUpDialog.vue";
import BackupSettings from "@/views/admin/BackupView.vue";
import DataManagementSettings from "@/views/admin/DataManagementView.vue";
import OpenAIFastPolicyUserSelector from "@/views/admin/settings/OpenAIFastPolicyUserSelector.vue";
import type { ProviderInstance } from "@/types/payment";
import type { TypeOption } from "@/components/payment/providerConfig";
import { useClipboard } from "@/composables/useClipboard";
import { isStepUpCancelled, useStepUp } from "@/composables/useStepUp";
import { useAppStore } from "@/stores";
import { useAdminSettingsStore } from "@/stores/adminSettings";
import {
  isRegistrationEmailSuffixDomainValid,
  normalizeRegistrationEmailSuffixDomain,
  normalizeRegistrationEmailSuffixDomains,
  parseRegistrationEmailSuffixWhitelistInput,
} from "@/utils/registrationEmailPolicy";
import { DEFAULT_SITE_NAME, DEFAULT_SITE_SUBTITLE } from "@/utils/brand";
import {
  parseFingerprintSignalsToRows,
  serializeFingerprintRowsToJSON,
  defaultFingerprintSignalRows,
  type FingerprintSignalRow,
} from "./codexFingerprintSignals";

const { t, locale } = useI18n();
const appStore = useAppStore();
const stepUp = useStepUp();
const adminSettingsStore = useAdminSettingsStore();
const isZhLocale = computed(() => locale.value.startsWith("zh"));

function localText(zh: string, en: string): string {
  return isZhLocale.value ? zh : en;
}

type SettingsTab =
  | "general"
  | "security"
  | "users"
  | "payment"
  | "gateway"
  | "email"
  | "backup"
  | "data";
const activeTab = ref<SettingsTab>("general");
const channelMonitorModes = ["v2", "v1"] as const;
const settingsTabs = [
  { key: "general" as SettingsTab, icon: "home" as const },
  { key: "security" as SettingsTab, icon: "shield" as const },
  { key: "users" as SettingsTab, icon: "user" as const },
  { key: "payment" as SettingsTab, icon: "creditCard" as const },
  { key: "gateway" as SettingsTab, icon: "server" as const },
  { key: "email" as SettingsTab, icon: "mail" as const },
  { key: "backup" as SettingsTab, icon: "database" as const },
  { key: "data" as SettingsTab, icon: "cube" as const },
];
const { copyToClipboard } = useClipboard();

const loading = ref(true);
const saving = ref(false);
const testingSmtp = ref(false);
const sendingTestEmail = ref(false);
const testEmailAddress = ref("");
const registrationEmailSuffixWhitelistTags = ref<string[]>([]);
const registrationEmailSuffixWhitelistDraft = ref("");
const forwardedClientIpHeaderDraft = ref("");

// Admin API Key 状态
const adminApiKeyLoading = ref(true);
const adminApiKeyExists = ref(false);
const adminApiKeyMasked = ref("");
const adminApiKeyOperating = ref(false);
const newAdminApiKey = ref("");
const subscriptionGroups = ref<AdminGroup[]>([]);
const platformQuotaPlatforms: PlatformType[] = [
  "anthropic",
  "openai",
  "gemini",
  "antigravity",
  "grok",
];
const schedulingThresholdPlatforms = SCHEDULING_THRESHOLD_PLATFORMS;

type ProviderPayload = {
  provider_key: string;
  name: string;
  supported_types: string[];
  enabled: boolean;
  payment_mode: string;
  refund_enabled: boolean;
  allow_user_refund: boolean;
  config: Record<string, string>;
  limits: string;
};

type PaymentProviderDialogExpose = {
  reset: (defaultKey: string) => void;
  loadProvider: (provider: ProviderInstance) => void;
};

const paymentTypeOptions: TypeOption[] = [
  { value: "easypay", label: "易支付" },
  { value: "alipay", label: "支付宝" },
  { value: "wxpay", label: "微信支付" },
  { value: "stripe", label: "Stripe" },
];
const allPaymentTypes: TypeOption[] = [
  ...paymentTypeOptions,
  { value: "alipay_direct", label: "支付宝直连" },
  { value: "wxpay_direct", label: "微信支付直连" },
  { value: "card", label: "银行卡" },
  { value: "link", label: "支付链接" },
  { value: "airwallex", label: "Airwallex" },
];
const providerKeyOptions = paymentTypeOptions;
const providers = ref<ProviderInstance[]>([]);
const providersLoading = ref(false);
const savingProvider = ref(false);
const showProviderDialog = ref(false);
const editingProvider = ref<ProviderInstance | null>(null);
const deletingProvider = ref<ProviderInstance | null>(null);
const providerDialogRef = ref<PaymentProviderDialogExpose | null>(null);

const upstreamBillingProbeLoading = ref(true);
const upstreamBillingProbeSaving = ref(false);
const upstreamBillingProbeForm = reactive({
  enabled: false,
  interval_minutes: 60,
});
const ollamaCloudUsageLoading = ref(true);
const ollamaCloudUsageSaving = ref(false);
const ollamaCloudUsageForm = reactive({
  enabled: false,
  interval_minutes: 60,
  debounce_minutes: 1,
});

// Overload Cooldown (529) 状态
const overloadCooldownLoading = ref(true);
const overloadCooldownSaving = ref(false);
const overloadCooldownForm = reactive({
  enabled: true,
  cooldown_minutes: 10,
});

// Panel API Rate Limit 状态
const panelRateLimitLoading = ref(true);
const panelRateLimitSaving = ref(false);
const panelRateLimitForm = reactive({
  enabled: true,
  user_rpm: 240,
  heavy_rpm: 60,
  exempt_admin: true,
  public_ip_rpm: 300,
});

// Stream Timeout 状态
const streamTimeoutLoading = ref(true);
const streamTimeoutSaving = ref(false);
const streamTimeoutForm = reactive({
  enabled: true,
  action: "temp_unsched" as "temp_unsched" | "error" | "none",
  temp_unsched_minutes: 5,
  threshold_count: 3,
  threshold_window_minutes: 10,
});

// Rectifier 状态
const rectifierLoading = ref(true);
const rectifierSaving = ref(false);
const rectifierForm = reactive({
  enabled: true,
  thinking_signature_enabled: true,
  thinking_budget_enabled: true,
});

// Beta Policy 状态
const betaPolicyLoading = ref(true);
const betaPolicySaving = ref(false);
const betaPolicyForm = reactive({
  rules: [] as Array<{
    beta_token: string;
    action: "pass" | "filter" | "block";
    scope: "all" | "oauth" | "apikey" | "bedrock";
    error_message?: string;
  }>,
});

const openaiFastPolicyForm = reactive({
  rules: [] as OpenAIFastPolicyRule[],
});
// Only write this setting back after it was present in a successful response.
const openaiFastPolicyLoaded = ref(false);

function defaultLoginAgreementDocuments(): LoginAgreementDocument[] {
  return [
    { id: "terms", title: localText("服务条款", "Terms of Service"), content_md: "" },
    { id: "usage-policy", title: localText("使用政策", "Usage Policy"), content_md: "" },
    {
      id: "supported-regions",
      title: localText("支持的国家和地区", "Supported Countries and Regions"),
      content_md: "",
    },
    {
      id: "service-specific-terms",
      title: localText("服务特定条款", "Service-Specific Terms"),
      content_md: "",
    },
  ];
}

type ClaudeOAuthSystemPromptPreset = "billing" | "system" | "expansion" | "custom";

interface ClaudeOAuthSystemPromptBlock {
  id: string;
  enabled: boolean;
  expanded: boolean;
  type: "text";
  preset: ClaudeOAuthSystemPromptPreset;
  text: string;
  cacheControlEnabled: boolean;
  cacheControlTTL: string;
}

interface ClaudeOAuthSystemPromptRawBlock {
  enabled?: boolean;
  type?: string;
  text?: string;
  cache_control?: unknown;
}

let claudeOAuthSystemPromptBlockID = 0;

function createClaudeOAuthSystemPromptBlock(
  overrides: Partial<ClaudeOAuthSystemPromptBlock> = {},
): ClaudeOAuthSystemPromptBlock {
  const text = overrides.text ?? "";
  return {
    id: `claude-oauth-system-prompt-block-${++claudeOAuthSystemPromptBlockID}`,
    enabled: overrides.enabled ?? true,
    expanded: overrides.expanded ?? true,
    type: "text",
    preset: overrides.preset ?? detectClaudeOAuthSystemPromptPreset(text),
    text,
    cacheControlEnabled: overrides.cacheControlEnabled ?? false,
    cacheControlTTL: overrides.cacheControlTTL ?? "5m",
  };
}

function detectClaudeOAuthSystemPromptPreset(text: string): ClaudeOAuthSystemPromptPreset {
  const value = text.trim();
  if (value === "{billing_header}") return "billing";
  if (value === "{claude_code_system_prompt}") return "system";
  if (value === "{claude_code_expansion_prompt}") return "expansion";
  return "custom";
}

function createDefaultClaudeOAuthSystemPromptBlocks(
  expansionPrompt = "",
): ClaudeOAuthSystemPromptBlock[] {
  return [
    createClaudeOAuthSystemPromptBlock({ preset: "billing", text: "{billing_header}" }),
    createClaudeOAuthSystemPromptBlock({ preset: "system", text: "{claude_code_system_prompt}" }),
    createClaudeOAuthSystemPromptBlock({
      preset: expansionPrompt.trim() ? "custom" : "expansion",
      text: expansionPrompt.trim() || "{claude_code_expansion_prompt}",
      cacheControlEnabled: true,
    }),
  ];
}

function parseClaudeOAuthSystemPromptBlocks(
  raw: string,
  expansionPrompt = "",
): ClaudeOAuthSystemPromptBlock[] {
  if (!raw.trim()) return createDefaultClaudeOAuthSystemPromptBlocks(expansionPrompt);
  try {
    const parsed = JSON.parse(raw) as ClaudeOAuthSystemPromptRawBlock[] | { blocks?: ClaudeOAuthSystemPromptRawBlock[] };
    const blocks = Array.isArray(parsed) ? parsed : parsed.blocks;
    if (!Array.isArray(blocks) || blocks.length === 0) {
      return createDefaultClaudeOAuthSystemPromptBlocks(expansionPrompt);
    }
    return blocks.map((block) => {
      const text = typeof block.text === "string" ? block.text : "";
      const cache = block.cache_control;
      const ttl = cache && typeof cache === "object" && !Array.isArray(cache)
        ? String((cache as Record<string, unknown>).ttl || "5m")
        : "5m";
      return createClaudeOAuthSystemPromptBlock({
        enabled: block.enabled !== false,
        text,
        preset: detectClaudeOAuthSystemPromptPreset(text),
        cacheControlEnabled: cache === true || (!!cache && typeof cache === "object"),
        cacheControlTTL: ttl,
      });
    });
  } catch {
    return createDefaultClaudeOAuthSystemPromptBlocks(expansionPrompt);
  }
}

function serializeClaudeOAuthSystemPromptBlocksToJSON(
  blocks: ClaudeOAuthSystemPromptBlock[],
): string {
  return JSON.stringify(
    blocks.map((block) => ({
      enabled: block.enabled,
      type: "text",
      text: block.text,
      ...(block.cacheControlEnabled
        ? { cache_control: { type: "ephemeral", ttl: block.cacheControlTTL || "5m" } }
        : {}),
    })),
    null,
    2,
  );
}

const defaultClaudeOAuthSystemPromptBlocks =
  serializeClaudeOAuthSystemPromptBlocksToJSON(createDefaultClaudeOAuthSystemPromptBlocks());
const claudeOAuthSystemPromptBlocks = ref<ClaudeOAuthSystemPromptBlock[]>(
  createDefaultClaudeOAuthSystemPromptBlocks(),
);
const claudeOAuthSystemPromptPresetOptions = computed(() => [
  { value: "billing", label: t("admin.settings.gatewayForwarding.systemBlockPresetBilling") },
  { value: "system", label: t("admin.settings.gatewayForwarding.systemBlockPresetIdentity") },
  { value: "expansion", label: t("admin.settings.gatewayForwarding.systemBlockPresetExpansion") },
  { value: "custom", label: t("admin.settings.gatewayForwarding.systemBlockPresetCustom") },
]);
const claudeOAuthSystemPromptBlockTypeOptions = computed(() => [
  { value: "text", label: t("admin.settings.gatewayForwarding.systemBlockTypeText") },
]);
const claudeOAuthSystemPromptCacheTTLOptions = computed(() => [
  { value: "5m", label: t("admin.settings.gatewayForwarding.cacheTTL5m") },
  { value: "1h", label: t("admin.settings.gatewayForwarding.cacheTTL1h") },
]);

function getClaudeOAuthPresetLabel(preset: ClaudeOAuthSystemPromptPreset): string {
  return claudeOAuthSystemPromptPresetOptions.value.find((item) => item.value === preset)?.label
    || t("admin.settings.gatewayForwarding.systemBlockPresetCustom");
}

function syncClaudeOAuthSystemPromptBlocksFormField(): void {
  form.claude_oauth_system_prompt_blocks =
    serializeClaudeOAuthSystemPromptBlocksToJSON(claudeOAuthSystemPromptBlocks.value);
}

function addClaudeOAuthSystemPromptBlock(): void {
  claudeOAuthSystemPromptBlocks.value.push(
    createClaudeOAuthSystemPromptBlock({ preset: "custom", text: "" }),
  );
  syncClaudeOAuthSystemPromptBlocksFormField();
}

function toggleClaudeOAuthSystemPromptBlock(index: number): void {
  const block = claudeOAuthSystemPromptBlocks.value[index];
  if (block) block.expanded = !block.expanded;
}

function removeClaudeOAuthSystemPromptBlock(index: number): void {
  claudeOAuthSystemPromptBlocks.value.splice(index, 1);
  syncClaudeOAuthSystemPromptBlocksFormField();
}

function moveClaudeOAuthSystemPromptBlock(index: number, direction: -1 | 1): void {
  const target = index + direction;
  if (target < 0 || target >= claudeOAuthSystemPromptBlocks.value.length) return;
  const blocks = claudeOAuthSystemPromptBlocks.value;
  [blocks[index], blocks[target]] = [blocks[target], blocks[index]];
  syncClaudeOAuthSystemPromptBlocksFormField();
}

function applyClaudeOAuthSystemPromptPreset(
  index: number,
  value: string | number | boolean | null,
): void {
  const block = claudeOAuthSystemPromptBlocks.value[index];
  if (!block) return;
  const preset = String(value || "custom") as ClaudeOAuthSystemPromptPreset;
  block.preset = preset;
  if (preset === "billing") block.text = "{billing_header}";
  if (preset === "system") block.text = "{claude_code_system_prompt}";
  if (preset === "expansion") {
    block.text = form.claude_oauth_system_prompt?.trim() || "{claude_code_expansion_prompt}";
    block.cacheControlEnabled = true;
  }
  syncClaudeOAuthSystemPromptBlocksFormField();
}

function markClaudeOAuthSystemPromptBlockCustom(block: ClaudeOAuthSystemPromptBlock): void {
  block.preset = detectClaudeOAuthSystemPromptPreset(block.text);
  syncClaudeOAuthSystemPromptBlocksFormField();
}

function resetClaudeOAuthSystemPromptBlocks(): void {
  claudeOAuthSystemPromptBlocks.value = createDefaultClaudeOAuthSystemPromptBlocks(
    form.claude_oauth_system_prompt,
  );
  syncClaudeOAuthSystemPromptBlocksFormField();
}

const tlsFingerprintLoading = ref(true);
const tlsFingerprintSaving = ref(false);
const tlsFingerprintProfiles = ref<TLSFingerprintProfile[]>([]);
const tlsFingerprintGlobalEnabled = ref(true);
const showTLSFingerprintModal = ref(false);
const editingTLSFingerprintProfileID = ref("");
const tlsFingerprintForm = reactive({
  profile_id: "",
  name: "",
  enabled: true,
  enable_grease: false,
  cipher_suites_text: "",
  curves_text: "",
  point_formats_text: "",
});

interface DefaultSubscriptionGroupOption {
  value: number;
  label: string;
  description: string | null;
  platform: AdminGroup["platform"];
  subscriptionType: AdminGroup["subscription_type"];
  rate: number;
  [key: string]: unknown;
}

type SettingsForm = Omit<
  SystemSettings,
  | "wechat_connect_open_enabled"
  | "wechat_connect_mp_enabled"
  | "wechat_connect_mobile_enabled"
  | "channel_monitor_mode"
  | "channel_monitor_hide_throughput"
  | "channel_monitor_show_quota"
> & {
  channel_monitor_mode: "v1" | "v2";
  channel_monitor_hide_throughput: boolean;
  channel_monitor_show_quota: boolean;
  account_scheduling_thresholds: ReturnType<
    typeof normalizeAccountSchedulingThresholdsMap
  >;
  smtp_password: string;
  turnstile_secret_key: string;
  tencent_captcha_app_secret_key: string;
  tencent_captcha_cloud_secret_id: string;
  tencent_captcha_cloud_secret_key: string;
  aliyun_captcha_access_key_secret: string;
  linuxdo_connect_client_secret: string;
  affiliate_admin_recharge_enabled: boolean;
  // WeChat Connect OAuth
  wechat_connect_enabled: boolean;
  wechat_connect_app_id: string;
  wechat_connect_app_secret: string;
  wechat_connect_app_secret_configured: boolean;
  wechat_connect_open_app_id: string;
  wechat_connect_open_app_secret: string;
  wechat_connect_open_app_secret_configured: boolean;
  wechat_connect_mp_app_id: string;
  wechat_connect_mp_app_secret: string;
  wechat_connect_mp_app_secret_configured: boolean;
  wechat_connect_mobile_app_id: string;
  wechat_connect_mobile_app_secret: string;
  wechat_connect_mobile_app_secret_configured: boolean;
  wechat_connect_open_enabled: boolean;
  wechat_connect_mp_enabled: boolean;
  wechat_connect_mobile_enabled: boolean;
  wechat_connect_mode: WeChatConnectMode;
  wechat_connect_scopes: string;
  wechat_connect_redirect_url: string;
  wechat_connect_frontend_redirect_url: string;
  // Generic OIDC OAuth
  oidc_connect_enabled: boolean;
  oidc_connect_provider_name: string;
  oidc_connect_client_id: string;
  oidc_connect_client_secret: string;
  oidc_connect_client_secret_configured: boolean;
  oidc_connect_issuer_url: string;
  oidc_connect_discovery_url: string;
  oidc_connect_authorize_url: string;
  oidc_connect_token_url: string;
  oidc_connect_userinfo_url: string;
  oidc_connect_jwks_url: string;
  oidc_connect_scopes: string;
  oidc_connect_redirect_url: string;
  oidc_connect_frontend_redirect_url: string;
  oidc_connect_token_auth_method: string;
  oidc_connect_use_pkce: boolean;
  oidc_connect_validate_id_token: boolean;
  oidc_connect_allowed_signing_algs: string;
  oidc_connect_clock_skew_seconds: number;
  oidc_connect_require_email_verified: boolean;
  oidc_connect_userinfo_email_path: string;
  oidc_connect_userinfo_id_path: string;
  oidc_connect_userinfo_username_path: string;
}

type OIDCTextFieldKey =
  | "oidc_connect_provider_name"
  | "oidc_connect_client_id"
  | "oidc_connect_issuer_url"
  | "oidc_connect_discovery_url"
  | "oidc_connect_authorize_url"
  | "oidc_connect_token_url"
  | "oidc_connect_userinfo_url"
  | "oidc_connect_jwks_url"
  | "oidc_connect_scopes"
  | "oidc_connect_redirect_url"
  | "oidc_connect_frontend_redirect_url";

const oidcTextFields: ReadonlyArray<readonly [OIDCTextFieldKey, string]> = [
  ["oidc_connect_provider_name", "提供商名称"],
  ["oidc_connect_client_id", "Client ID"],
  ["oidc_connect_issuer_url", "Issuer URL"],
  ["oidc_connect_discovery_url", "Discovery URL"],
  ["oidc_connect_authorize_url", "Authorize URL"],
  ["oidc_connect_token_url", "Token URL"],
  ["oidc_connect_userinfo_url", "UserInfo URL"],
  ["oidc_connect_jwks_url", "JWKS URL"],
  ["oidc_connect_scopes", "Scopes"],
  ["oidc_connect_redirect_url", "Redirect URL"],
  ["oidc_connect_frontend_redirect_url", "前端回调路径"],
];

const form = reactive<SettingsForm>({
  registration_enabled: true,
  email_verify_enabled: false,
  registration_email_suffix_whitelist: [],
  promo_code_enabled: true,
  invitation_code_enabled: false,
  password_reset_enabled: false,
  totp_enabled: false,
  totp_encryption_key_configured: false,
  passkey_enabled: false,
  passkey_configured: false,
  passkey_rp_id: "",
  passkey_rp_origins: [],
  default_balance: 0,
  default_concurrency: 1,
  default_subscriptions: [],
  default_platform_quotas: normalizePlatformQuotasMap(),
  affiliate_enabled: false,
  affiliate_admin_recharge_enabled: false,
  affiliate_rebate_rate: 10,
  affiliate_rebate_freeze_hours: 0,
  affiliate_rebate_duration_days: 0,
  affiliate_rebate_per_invitee_cap: 0,
  site_name: DEFAULT_SITE_NAME,
  site_logo: "",
  site_subtitle: DEFAULT_SITE_SUBTITLE,
  api_base_url: "",
  contact_info: "",
  doc_url: "",
  home_content: "",
  compact_home_enabled: false,
  backend_mode_enabled: false,
  hide_ccs_import_button: false,
  purchase_subscription_enabled: false,
  purchase_subscription_url: "",
  purchase_link_cny_10: "",
  purchase_link_cny_30: "",
  purchase_link_cny_100: "",
  sora_client_enabled: false,
  custom_menu_items: [] as Array<{
    id: string;
    label: string;
    icon_svg: string;
    url: string;
    visibility: "user" | "admin";
    sort_order: number;
  }>,
  frontend_url: "",
  smtp_host: "",
  smtp_port: 587,
  smtp_username: "",
  smtp_password: "",
  smtp_password_configured: false,
  smtp_from_email: "",
  smtp_from_name: "",
  smtp_use_tls: true,
  // Cloudflare Turnstile
  turnstile_enabled: false,
  turnstile_site_key: "",
  turnstile_secret_key: "",
  turnstile_secret_key_configured: false,
  tencent_captcha_enabled: false,
  tencent_captcha_app_id: "",
  tencent_captcha_app_secret_key: "",
  tencent_captcha_app_secret_key_configured: false,
  tencent_captcha_cloud_secret_id: "",
  tencent_captcha_cloud_secret_id_configured: false,
  tencent_captcha_cloud_secret_key: "",
  tencent_captcha_cloud_secret_key_configured: false,
  tencent_captcha_region: "cn",
  aliyun_captcha_enabled: false,
  aliyun_captcha_access_key_id: "",
  aliyun_captcha_access_key_secret: "",
  aliyun_captcha_access_key_secret_configured: false,
  aliyun_captcha_scene_id: "",
  aliyun_captcha_prefix: "",
  aliyun_captcha_region: "cn",
  api_key_acl_trust_forwarded_ip: true,
  forwarded_client_ip_headers: [],
  // LinuxDo Connect OAuth 登录
  linuxdo_connect_enabled: false,
  linuxdo_connect_client_id: "",
  linuxdo_connect_client_secret: "",
  linuxdo_connect_client_secret_configured: false,
  linuxdo_connect_redirect_url: "",
  // WeChat Connect OAuth
  wechat_connect_enabled: false,
  wechat_connect_app_id: "",
  wechat_connect_app_secret: "",
  wechat_connect_app_secret_configured: false,
  wechat_connect_open_app_id: "",
  wechat_connect_open_app_secret: "",
  wechat_connect_open_app_secret_configured: false,
  wechat_connect_mp_app_id: "",
  wechat_connect_mp_app_secret: "",
  wechat_connect_mp_app_secret_configured: false,
  wechat_connect_mobile_app_id: "",
  wechat_connect_mobile_app_secret: "",
  wechat_connect_mobile_app_secret_configured: false,
  wechat_connect_open_enabled: false,
  wechat_connect_mp_enabled: false,
  wechat_connect_mobile_enabled: false,
  wechat_connect_mode: "open",
  wechat_connect_scopes: "snsapi_login",
  wechat_connect_redirect_url: "",
  wechat_connect_frontend_redirect_url: "/auth/wechat/callback",
  // Generic OIDC OAuth
  oidc_connect_enabled: false,
  oidc_connect_provider_name: "OIDC",
  oidc_connect_client_id: "",
  oidc_connect_client_secret: "",
  oidc_connect_client_secret_configured: false,
  oidc_connect_issuer_url: "",
  oidc_connect_discovery_url: "",
  oidc_connect_authorize_url: "",
  oidc_connect_token_url: "",
  oidc_connect_userinfo_url: "",
  oidc_connect_jwks_url: "",
  oidc_connect_scopes: "openid email profile",
  oidc_connect_redirect_url: "",
  oidc_connect_frontend_redirect_url: "/auth/oidc/callback",
  oidc_connect_token_auth_method: "client_secret_post",
  oidc_connect_use_pkce: true,
  oidc_connect_validate_id_token: true,
  oidc_connect_allowed_signing_algs: "RS256,ES256,PS256",
  oidc_connect_clock_skew_seconds: 120,
  oidc_connect_require_email_verified: false,
  oidc_connect_userinfo_email_path: "",
  oidc_connect_userinfo_id_path: "",
  oidc_connect_userinfo_username_path: "",
  // Model fallback
  enable_model_fallback: false,
  fallback_model_anthropic: "claude-3-5-sonnet-20241022",
  fallback_model_openai: "gpt-4o",
  fallback_model_gemini: "gemini-2.5-pro",
  fallback_model_antigravity: "gemini-2.5-pro",
  // Identity patch (Claude -> Gemini)
  enable_identity_patch: true,
  identity_patch_prompt: "",
  // Ops monitoring (vNext)
  ops_monitoring_enabled: true,
  ops_realtime_monitoring_enabled: true,
  ops_query_mode_default: "auto",
  ops_metrics_interval_seconds: 60,
  // Claude Code version check
  min_claude_code_version: "",
  max_claude_code_version: "",
  // 分组隔离
  allow_ungrouped_key_scheduling: false,
  openai_low_upstream_rate_priority_enabled: false,
  openai_oauth_scheduling_rate_multiplier: 1,
  openai_advanced_scheduler_enabled: false,
  openai_advanced_scheduler_sticky_weighted_enabled: false,
  openai_advanced_scheduler_subscription_priority_enabled: false,
  openai_advanced_scheduler_lb_top_k: "",
  openai_advanced_scheduler_weight_priority: "",
  openai_advanced_scheduler_weight_load: "",
  openai_advanced_scheduler_weight_queue: "",
  openai_advanced_scheduler_weight_error_rate: "",
  openai_advanced_scheduler_weight_ttft: "",
  openai_advanced_scheduler_weight_reset: "",
  openai_advanced_scheduler_weight_quota_headroom: "",
  openai_advanced_scheduler_weight_upstream_cost: "",
  openai_advanced_scheduler_weight_previous_response: "",
  openai_advanced_scheduler_weight_session_sticky: "",
  // Gateway forwarding behavior
  enable_fingerprint_unification: true,
  enable_metadata_passthrough: false,
  enable_cch_signing: false,
  enable_claude_oauth_system_prompt_injection: true,
  claude_oauth_system_prompt: "",
  claude_oauth_system_prompt_blocks: defaultClaudeOAuthSystemPromptBlocks,
  enable_anthropic_cache_ttl_1h_injection: false,
  rewrite_message_cache_control: false,
  enable_client_dateline_normalization: true,
  antigravity_user_agent_version: "",
  openai_codex_user_agent: "",
  openai_codex_client_version: "",
  // 只读展示：自动同步任务写入的官方最新稳定版，不参与提交（提交载荷按字段显式构造）
  openai_codex_client_version_synced: "",
  openai_codex_version_auto_sync_enabled: true,
  // codex_cli_only 加固
  min_codex_version: "",
  max_codex_version: "",
  codex_cli_only_blacklist: "",
  codex_cli_only_whitelist: "",
  codex_cli_only_allow_app_server_clients: false,
  codex_cli_only_engine_fingerprint_signals: "",
  // 余额、订阅到期与账号限额通知
  balance_low_notify_enabled: false,
  balance_low_notify_threshold: 0,
  balance_low_notify_recharge_url: "",
  subscription_expiry_notify_enabled: true,
  account_quota_notify_enabled: false,
  account_quota_notify_emails: [] as NotifyEmailEntry[],
  account_scheduling_thresholds: normalizeAccountSchedulingThresholdsMap(),
  // Channel Monitor feature switch
  channel_monitor_enabled: true,
  channel_monitor_mode: "v2",
  channel_monitor_default_interval_seconds: 60,
  channel_monitor_hide_throughput: false,
  channel_monitor_show_quota: false,
  // Available Channels feature switch
  available_channels_enabled: false,
  // Model Plaza feature switches + description
  model_plaza_enabled: false,
  model_plaza_require_auth: false,
  model_plaza_description: "",
  grok_default_text_model: "grok-4.5",
  grok_cross_client_model_map_enabled: false,
} as unknown as SettingsForm);

// 人机验证 UI 状态：单卡片「总开关 + 服务商单选」，落库仍是三个独立
// enabled 键（与上游一致），由下面的映射保证同一时间至多一家启用。
type CaptchaProviderSelection = "turnstile" | "tencent" | "aliyun";

const captchaProviderSelection = ref<CaptchaProviderSelection>("turnstile");

function applyCaptchaSelection(provider: CaptchaProviderSelection | null): void {
  form.turnstile_enabled = provider === "turnstile";
  form.tencent_captcha_enabled = provider === "tencent";
  form.aliyun_captcha_enabled = provider === "aliyun";
}

const captchaMasterEnabled = computed({
  get: () =>
    form.turnstile_enabled ||
    form.tencent_captcha_enabled ||
    form.aliyun_captcha_enabled,
  set: (enabled: boolean) =>
    applyCaptchaSelection(enabled ? captchaProviderSelection.value : null),
});

function selectCaptchaProvider(provider: CaptchaProviderSelection): void {
  captchaProviderSelection.value = provider;
  applyCaptchaSelection(provider);
}

// 天御中国站与国际站是两套独立账号体系，控制台与文档入口不通用，
// 按当前选择的站点给出对应链接，避免管理员在错误的控制台里找不到 CaptchaAppId。
const tencentCaptchaLinks = computed(() =>
  form.tencent_captcha_region === "intl"
    ? {
        console: "https://console.tencentcloud.com/captcha/graphical",
        cloudKeys: "https://console.tencentcloud.com/cam/capi",
        webDocs: "https://www.tencentcloud.com/document/product/1159/49680",
      }
    : {
        console: "https://console.cloud.tencent.com/captcha",
        cloudKeys: "https://console.cloud.tencent.com/cam/capi",
        webDocs: "https://cloud.tencent.com/document/product/1110/36841",
      },
);

function syncCaptchaProviderSelection(): void {
  if (form.tencent_captcha_enabled) {
    captchaProviderSelection.value = "tencent";
  } else if (form.aliyun_captcha_enabled) {
    captchaProviderSelection.value = "aliyun";
  } else if (form.turnstile_enabled) {
    captchaProviderSelection.value = "turnstile";
  }
}

type OpenAIAdvancedSchedulerOverrideKey =
  | "openai_advanced_scheduler_lb_top_k"
  | "openai_advanced_scheduler_weight_priority"
  | "openai_advanced_scheduler_weight_load"
  | "openai_advanced_scheduler_weight_queue"
  | "openai_advanced_scheduler_weight_error_rate"
  | "openai_advanced_scheduler_weight_ttft"
  | "openai_advanced_scheduler_weight_reset"
  | "openai_advanced_scheduler_weight_quota_headroom"
  | "openai_advanced_scheduler_weight_upstream_cost"
  | "openai_advanced_scheduler_weight_previous_response"
  | "openai_advanced_scheduler_weight_session_sticky";

type OpenAIAdvancedSchedulerEffectiveKey =
  | "openai_advanced_scheduler_effective_lb_top_k"
  | "openai_advanced_scheduler_effective_weight_priority"
  | "openai_advanced_scheduler_effective_weight_load"
  | "openai_advanced_scheduler_effective_weight_queue"
  | "openai_advanced_scheduler_effective_weight_error_rate"
  | "openai_advanced_scheduler_effective_weight_ttft"
  | "openai_advanced_scheduler_effective_weight_reset"
  | "openai_advanced_scheduler_effective_weight_quota_headroom"
  | "openai_advanced_scheduler_effective_weight_upstream_cost"
  | "openai_advanced_scheduler_effective_weight_previous_response"
  | "openai_advanced_scheduler_effective_weight_session_sticky";

const openAIAdvancedSchedulerWeightFields = computed<
  Array<{
    key: OpenAIAdvancedSchedulerOverrideKey;
    label: string;
    placeholder: string;
  }>
>(() => {
  const placeholder = (
    effectiveKey: OpenAIAdvancedSchedulerEffectiveKey,
    fallbackValue: string,
  ) => {
    const effectiveValue = String(
      (form as Record<string, unknown>)[effectiveKey] ?? "",
    ).trim();
    return t("admin.settings.openaiExperimentalScheduler.defaultPlaceholder", {
      value: effectiveValue || fallbackValue,
    });
  };

  return [
    {
      key: "openai_advanced_scheduler_lb_top_k",
      label: t("admin.settings.openaiExperimentalScheduler.topKLabel"),
      placeholder: placeholder("openai_advanced_scheduler_effective_lb_top_k", "7"),
    },
    {
      key: "openai_advanced_scheduler_weight_priority",
      label: t("admin.settings.openaiExperimentalScheduler.priorityWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_priority", "1"),
    },
    {
      key: "openai_advanced_scheduler_weight_load",
      label: t("admin.settings.openaiExperimentalScheduler.loadWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_load", "1"),
    },
    {
      key: "openai_advanced_scheduler_weight_queue",
      label: t("admin.settings.openaiExperimentalScheduler.queueWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_queue", "0.7"),
    },
    {
      key: "openai_advanced_scheduler_weight_error_rate",
      label: t("admin.settings.openaiExperimentalScheduler.errorRateWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_error_rate", "0.8"),
    },
    {
      key: "openai_advanced_scheduler_weight_ttft",
      label: t("admin.settings.openaiExperimentalScheduler.ttftWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_ttft", "0.5"),
    },
    {
      key: "openai_advanced_scheduler_weight_reset",
      label: t("admin.settings.openaiExperimentalScheduler.resetWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_reset", "0"),
    },
    {
      key: "openai_advanced_scheduler_weight_quota_headroom",
      label: t("admin.settings.openaiExperimentalScheduler.quotaHeadroomWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_quota_headroom", "0"),
    },
    {
      key: "openai_advanced_scheduler_weight_upstream_cost",
      label: t("admin.settings.openaiExperimentalScheduler.upstreamCostWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_upstream_cost", "0"),
    },
    {
      key: "openai_advanced_scheduler_weight_previous_response",
      label: t("admin.settings.openaiExperimentalScheduler.previousResponseWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_previous_response", "5"),
    },
    {
      key: "openai_advanced_scheduler_weight_session_sticky",
      label: t("admin.settings.openaiExperimentalScheduler.sessionStickyWeight"),
      placeholder: placeholder("openai_advanced_scheduler_effective_weight_session_sticky", "3"),
    },
  ];
});

const authSourceDefaults = reactive<AuthSourceDefaultsState>(
  buildAuthSourceDefaultsState({}),
);

const authSourceDefaultsMeta = computed(() => [
  {
    source: "email" as AuthSourceType,
    title: t("admin.settings.authSourceDefaults.sources.email.title"),
    description: t("admin.settings.authSourceDefaults.sources.email.description"),
  },
  {
    source: "linuxdo" as AuthSourceType,
    title: t("admin.settings.authSourceDefaults.sources.linuxdo.title"),
    description: t("admin.settings.authSourceDefaults.sources.linuxdo.description"),
  },
  {
    source: "oidc" as AuthSourceType,
    title: t("admin.settings.authSourceDefaults.sources.oidc.title"),
    description: t("admin.settings.authSourceDefaults.sources.oidc.description"),
  },
  {
    source: "wechat" as AuthSourceType,
    title: t("admin.settings.authSourceDefaults.sources.wechat.title"),
    description: t("admin.settings.authSourceDefaults.sources.wechat.description"),
  },
  {
    source: "github" as AuthSourceType,
    title: "GitHub",
    description: localText(
      "通过 GitHub 已验证邮箱首次注册或首次绑定时应用。",
      "Applied on first signup or first bind through a verified GitHub email.",
    ),
  },
  {
    source: "google" as AuthSourceType,
    title: "Google",
    description: localText(
      "通过 Google 已验证邮箱首次注册或首次绑定时应用。",
      "Applied on first signup or first bind through a verified Google email.",
    ),
  },
  {
    source: "dingtalk" as AuthSourceType,
    title: t("auth.dingtalkProviderName"),
    description: localText(
      "通过钉钉首次注册或首次绑定时应用。",
      "Applied on first signup or first bind through DingTalk.",
    ),
  },
]);

// Proxies for web search emulation ProxySelector
const webSearchProxies = ref<Proxy[]>([]);

// Web Search Emulation config (loaded/saved separately)
const DEFAULT_WEB_SEARCH_QUOTA_LIMIT = 1000;

const webSearchConfig = reactive<WebSearchEmulationConfig>({
  enabled: false,
  providers: [],
});

const expandedProviders = reactive<Record<number, boolean>>({});
const apiKeyVisible = reactive<Record<number, boolean>>({});
const wsTestQuery = ref("");
const wsTestLoading = ref(false);
const wsTestResult = ref<WebSearchTestResult | null>(null);
const wsTestDialogOpen = ref(false);

function openTestDialog() {
  wsTestResult.value = null;
  wsTestDialogOpen.value = true;
}

function toggleProviderExpand(idx: number) {
  expandedProviders[idx] = !expandedProviders[idx];
}

function removeWebSearchProvider(idx: number) {
  webSearchConfig.providers.splice(idx, 1);
  // Re-index expandedProviders and apiKeyVisible after removal
  const newExpanded: Record<number, boolean> = {};
  const newVisible: Record<number, boolean> = {};
  for (let i = 0; i < webSearchConfig.providers.length; i++) {
    const oldIdx = i >= idx ? i + 1 : i;
    newExpanded[i] = expandedProviders[oldIdx] ?? false;
    newVisible[i] = apiKeyVisible[oldIdx] ?? false;
  }
  Object.keys(expandedProviders).forEach(
    (k) => delete expandedProviders[Number(k)],
  );
  Object.keys(apiKeyVisible).forEach((k) => delete apiKeyVisible[Number(k)]);
  Object.assign(expandedProviders, newExpanded);
  Object.assign(apiKeyVisible, newVisible);
}

function addWebSearchProvider() {
  const idx = webSearchConfig.providers.length;
  webSearchConfig.providers.push({
    type: "brave",
    api_key: "",
    api_key_configured: false,
    quota_limit: DEFAULT_WEB_SEARCH_QUOTA_LIMIT,
    subscribed_at: null,
    proxy_id: null,
    expires_at: null,
  } as WebSearchProviderConfig);
  expandedProviders[idx] = true;
}

function formatSubscribedAt(ts: number | null): string {
  if (!ts) return "";
  // Use UTC to avoid timezone drift on repeated edits
  const d = new Date(ts * 1000);
  const y = d.getUTCFullYear();
  const m = String(d.getUTCMonth() + 1).padStart(2, "0");
  const day = String(d.getUTCDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

function parseSubscribedAt(dateStr: string): number | null {
  if (!dateStr) return null;
  // Parse as UTC to match formatSubscribedAt
  return Math.floor(new Date(dateStr + "T00:00:00Z").getTime() / 1000);
}

function quotaPercentage(provider: WebSearchProviderConfig): number {
  if (!provider.quota_limit || provider.quota_limit <= 0) return 0;
  return ((provider.quota_used ?? 0) / provider.quota_limit) * 100;
}

async function resetWebSearchUsage(idx: number) {
  const provider = webSearchConfig.providers[idx];
  if (!provider) return;
  if (!confirm(t("admin.settings.webSearchEmulation.resetUsageConfirm")))
    return;
  try {
    await adminAPI.settings.resetWebSearchUsage({ provider_type: provider.type });
    provider.quota_used = 0;
    appStore.showSuccess(t("admin.settings.webSearchEmulation.resetUsageSuccess"));
  } catch (error: unknown) {
    appStore.showError(error instanceof Error ? error.message : t("common.error"));
  }
}

async function copyApiKey(idx: number) {
  const key = webSearchConfig.providers[idx]?.api_key;
  if (!key) {
    appStore.showError(t("admin.settings.webSearchEmulation.apiKeyPlaceholder"));
    return;
  }
  await copyToClipboard(key);
  appStore.showSuccess(t("admin.settings.webSearchEmulation.copied"));
}

async function testWebSearchProvider() {
  wsTestLoading.value = true;
  wsTestResult.value = null;
  try {
    const query = wsTestQuery.value.trim() || t("admin.settings.webSearchEmulation.testDefaultQuery");
    wsTestResult.value = await adminAPI.settings.testWebSearchEmulation(query);
  } catch (error: unknown) {
    appStore.showError(error instanceof Error ? error.message : t("common.error"));
  } finally {
    wsTestLoading.value = false;
  }
}

async function loadWebSearchConfig() {
  try {
    const [config, proxies] = await Promise.all([
      adminAPI.settings.getWebSearchEmulationConfig(),
      adminAPI.proxies.list(),
    ]);
    webSearchConfig.enabled = config.enabled;
    webSearchConfig.providers = config.providers || [];
    webSearchProxies.value = Array.isArray(proxies) ? proxies : (proxies as { data?: Proxy[] }).data || [];
  } catch (error) {
    console.error("Failed to load web search emulation settings:", error);
  }
}

async function saveWebSearchConfig(): Promise<boolean> {
  try {
    await adminAPI.settings.updateWebSearchEmulationConfig({
      enabled: webSearchConfig.enabled,
      providers: webSearchConfig.providers,
    });
    return true;
  } catch (error: unknown) {
    appStore.showError(error instanceof Error ? error.message : t("common.error"));
    return false;
  }
}

function isValidHttpUrl(url: string | null | undefined): boolean {
  if (!url?.trim()) return false;
  try {
    const parsed = new URL(url.trim());
    return parsed.protocol === "http:" || parsed.protocol === "https:";
  } catch {
    return false;
  }
}

// DB 里 password_reset_enabled 的原始存储值（未与 email_verify_enabled 取与）。
// 后端返回的 form.password_reset_enabled 是取与后的生效值：邮箱验证关闭时恒为 false，
// 因此它**不能**用来判断「密码重置是否被配置过」。老后端不返回 stored 字段时回退到生效值。
const passwordResetEnabledStored = ref(false);

function syncPasswordResetStored(settings: {
  password_reset_enabled: boolean;
  password_reset_enabled_stored?: boolean;
}) {
  passwordResetEnabledStored.value =
    settings.password_reset_enabled_stored ?? settings.password_reset_enabled;
  // form 必须持有**原始意图**，而不是后端取与后的生效值。
  // 否则邮箱验证关闭时 form.password_reset_enabled 被种成 false，管理员一旦在页面上打开
  // 邮箱验证开关（这恰恰是状态 B 文案引导他做的事），判据会瞬间塌回 false：
  // 潜伏告警消失、状态 A 告警不出现、frontend_url 输入框消失，保存还会把 DB 里的 true 抹掉。
  form.password_reset_enabled = passwordResetEnabledStored.value;
}

// 管理员对「忘记密码」的真实意图。form 已在 syncPasswordResetStored 里被种成原始存储值，
// 因此这里直接取 form 即可：邮箱验证开启时它是开关的实时值，关闭时它保持 DB 原始值不变
// （开关不渲染，无人能改它），跨越 email_verify 开关切换时也不会失真。
const passwordResetIntended = computed(() => form.password_reset_enabled);

// 后端解析出的重置链接基址（service.ResolvePasswordResetBaseURL 的结果）。
// 空串表示「按当前配置根本解析不出链接、发信会被静默跳过」。
// 这是唯一可信的判据 —— 前端**不复刻**那条回落链路（DB → 配置文件 → Origin + CORS 白名单），
// 因为前端既拿不到请求 Origin 也拿不到 CORS 白名单，复刻必然漂移成第二个事实来源。
const passwordResetLinkBase = ref("");

function syncPasswordResetLinkBase(settings: {
  password_reset_link_base?: string;
}) {
  passwordResetLinkBase.value = (
    settings.password_reset_link_base ?? ""
  ).trim();
}

// 「重置链接解析不出来」的判断：
// - 表单里已经填了合法地址 → 保存后必然可解析，不告警（乐观，避免边填边报错）；
// - 表单为空 → 以后端解析结果为准。后端可能通过 config.yaml 回落或 Origin 兜底解析成功，
//   那种部署下 DB 的 frontend_url 为空是正常的，不该报「客户收不到邮件」。
const frontendUrlMissing = computed(() => {
  const typed = (form.frontend_url || "").trim();
  if (typed !== "") return false;
  return passwordResetLinkBase.value === "";
});

// 状态 A「正在静默失败」：邮箱验证开着、密码重置开着、frontend_url 为空。
// 此刻客户点重置就收不到邮件，页面还显示发送成功 —— 紧急措辞成立。
const passwordResetSilentlyFailing = computed(
  () =>
    passwordResetIntended.value &&
    form.email_verify_enabled &&
    frontendUrlMissing.value,
);

// 状态 B「潜伏」：密码重置存的是 true，但邮箱验证关着，重置功能因取与而未生效。
// 此刻**没有**在静默失败，不能套用状态 A 的文案；一旦开启邮箱验证才会立刻开始静默失败。
const passwordResetLatentlyMisconfigured = computed(
  () =>
    passwordResetIntended.value &&
    !form.email_verify_enabled &&
    frontendUrlMissing.value,
);

const frontendUrlFormatInvalid = computed(() => {
  const value = (form.frontend_url || "").trim();
  return value !== "" && !isValidHttpUrl(value);
});

const contactInfoMissing = computed(() => !(form.contact_info || "").trim());

const defaultSubscriptionGroupOptions = computed<
  DefaultSubscriptionGroupOption[]
>(() =>
  subscriptionGroups.value.map((group) => ({
    value: group.id,
    label: group.name,
    description: group.description,
    platform: group.platform,
    subscriptionType: group.subscription_type,
    rate: group.rate_multiplier,
  })),
);

const registrationEmailSuffixWhitelistSeparatorKeys = new Set([
  " ",
  ",",
  "，",
  "Enter",
  "Tab",
]);

function removeRegistrationEmailSuffixWhitelistTag(suffix: string) {
  registrationEmailSuffixWhitelistTags.value =
    registrationEmailSuffixWhitelistTags.value.filter(
      (item) => item !== suffix,
    );
}

function addRegistrationEmailSuffixWhitelistTag(raw: string) {
  const suffix = normalizeRegistrationEmailSuffixDomain(raw);
  if (
    !isRegistrationEmailSuffixDomainValid(suffix) ||
    registrationEmailSuffixWhitelistTags.value.includes(suffix)
  ) {
    return;
  }
  registrationEmailSuffixWhitelistTags.value = [
    ...registrationEmailSuffixWhitelistTags.value,
    suffix,
  ];
}

function commitRegistrationEmailSuffixWhitelistDraft() {
  if (!registrationEmailSuffixWhitelistDraft.value) {
    return;
  }
  addRegistrationEmailSuffixWhitelistTag(
    registrationEmailSuffixWhitelistDraft.value,
  );
  registrationEmailSuffixWhitelistDraft.value = "";
}

function handleRegistrationEmailSuffixWhitelistDraftInput() {
  registrationEmailSuffixWhitelistDraft.value =
    normalizeRegistrationEmailSuffixDomain(
      registrationEmailSuffixWhitelistDraft.value,
    );
}

function handleRegistrationEmailSuffixWhitelistDraftKeydown(
  event: KeyboardEvent,
) {
  if (event.isComposing) {
    return;
  }

  if (registrationEmailSuffixWhitelistSeparatorKeys.has(event.key)) {
    event.preventDefault();
    commitRegistrationEmailSuffixWhitelistDraft();
    return;
  }

  if (
    event.key === "Backspace" &&
    !registrationEmailSuffixWhitelistDraft.value &&
    registrationEmailSuffixWhitelistTags.value.length > 0
  ) {
    registrationEmailSuffixWhitelistTags.value.pop();
  }
}

function handleRegistrationEmailSuffixWhitelistPaste(event: ClipboardEvent) {
  const text = event.clipboardData?.getData("text") || "";
  if (!text.trim()) {
    return;
  }
  event.preventDefault();
  const tokens = parseRegistrationEmailSuffixWhitelistInput(text);
  for (const token of tokens) {
    addRegistrationEmailSuffixWhitelistTag(token);
  }
}

const forwardedClientIpHeaderSeparatorKeys = new Set([" ", ",", "，", "Enter", "Tab"]);
const forwardedClientIpHeaderTokenPattern = /^[!#$%&'*+\-.^_`|~0-9A-Za-z]+$/;
const maxForwardedClientIpHeaders = 16;
type ForwardedClientIpHeaderResult = "added" | "duplicate" | "invalid" | "full";

function normalizeForwardedClientIpHeader(raw: string): string {
  const header = raw.trim();
  if (!forwardedClientIpHeaderTokenPattern.test(header)) return "";
  return header.toLowerCase().split("-")
    .map((part) => `${part.charAt(0).toUpperCase()}${part.slice(1)}`)
    .join("-");
}

function normalizeForwardedClientIpHeaders(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  const headers: string[] = [];
  const seen = new Set<string>();
  for (const raw of value) {
    if (typeof raw !== "string") continue;
    const header = normalizeForwardedClientIpHeader(raw);
    const key = header.toLowerCase();
    if (!header || seen.has(key) || headers.length >= maxForwardedClientIpHeaders) continue;
    seen.add(key);
    headers.push(header);
  }
  return headers;
}

function removeForwardedClientIpHeader(header: string) {
  form.forwarded_client_ip_headers = (form.forwarded_client_ip_headers ?? []).filter(
    (item) => item !== header,
  );
}

function addForwardedClientIpHeader(raw: string): ForwardedClientIpHeaderResult {
  const header = normalizeForwardedClientIpHeader(raw);
  if (!header) return "invalid";
  const current = form.forwarded_client_ip_headers ?? [];
  if (current.some((item) => item.toLowerCase() === header.toLowerCase())) return "duplicate";
  if (current.length >= maxForwardedClientIpHeaders) return "full";
  form.forwarded_client_ip_headers = [...current, header];
  return "added";
}

function showForwardedClientIpHeaderError(result: ForwardedClientIpHeaderResult) {
  if (result === "invalid") {
    appStore.showError(t("admin.settings.apiKeyAcl.forwardedClientIpHeaderInvalid"));
  } else if (result === "full") {
    appStore.showError(t("admin.settings.apiKeyAcl.forwardedClientIpHeadersLimit", {
      max: maxForwardedClientIpHeaders,
    }));
  }
}

function commitForwardedClientIpHeaderDraft() {
  const draft = forwardedClientIpHeaderDraft.value;
  if (!draft) return;
  showForwardedClientIpHeaderError(addForwardedClientIpHeader(draft));
  forwardedClientIpHeaderDraft.value = "";
}

function handleForwardedClientIpHeaderKeydown(event: KeyboardEvent) {
  if (event.isComposing) return;
  if (forwardedClientIpHeaderSeparatorKeys.has(event.key)) {
    event.preventDefault();
    commitForwardedClientIpHeaderDraft();
    return;
  }
  if (event.key === "Backspace" && !forwardedClientIpHeaderDraft.value &&
      (form.forwarded_client_ip_headers?.length ?? 0) > 0) {
    form.forwarded_client_ip_headers?.pop();
  }
}

function handleForwardedClientIpHeaderPaste(event: ClipboardEvent) {
  const text = event.clipboardData?.getData("text") || "";
  if (!text.trim()) return;
  event.preventDefault();
  let error: ForwardedClientIpHeaderResult | undefined;
  for (const token of text.split(/[,，;\r\n]+/)) {
    if (!token.trim()) continue;
    const result = addForwardedClientIpHeader(token);
    if (result === "invalid" || result === "full") error = result;
  }
  if (error) showForwardedClientIpHeaderError(error);
}

// LinuxDo OAuth redirect URL suggestion
const linuxdoRedirectUrlSuggestion = computed(() => {
  if (typeof window === "undefined") return "";
  const origin =
    window.location.origin ||
    `${window.location.protocol}//${window.location.host}`;
  return `${origin}/api/v1/auth/oauth/linuxdo/callback`;
});

async function setAndCopyLinuxdoRedirectUrl() {
  const url = linuxdoRedirectUrlSuggestion.value;
  if (!url) return;

  form.linuxdo_connect_redirect_url = url;
  await copyToClipboard(
    url,
    t("admin.settings.linuxdo.redirectUrlSetAndCopied"),
  );
}

function syncWeChatConnectMode(preferredMode?: WeChatConnectMode) {
  if (form.wechat_connect_mp_enabled && form.wechat_connect_mobile_enabled) {
    if (preferredMode === "mobile") form.wechat_connect_mp_enabled = false;
    else form.wechat_connect_mobile_enabled = false;
  }
  const capabilities = resolveWeChatConnectModeCapabilities(
    form.wechat_connect_open_enabled,
    form.wechat_connect_mp_enabled,
    form.wechat_connect_mobile_enabled,
    form.wechat_connect_mode,
  );
  form.wechat_connect_open_enabled = capabilities.openEnabled;
  form.wechat_connect_mp_enabled = capabilities.mpEnabled;
  form.wechat_connect_mobile_enabled = capabilities.mobileEnabled;
  form.wechat_connect_mode = deriveWeChatConnectStoredMode(
    capabilities.openEnabled,
    capabilities.mpEnabled,
    capabilities.mobileEnabled,
    form.wechat_connect_mode,
  );
  form.wechat_connect_scopes = defaultWeChatConnectScopesForMode(
    form.wechat_connect_mode,
  );
}

function handleWeChatOpenEnabledChange(value: boolean) {
  form.wechat_connect_open_enabled = value;
  syncWeChatConnectMode(value ? "open" : undefined);
}

function handleWeChatMPEnabledChange(value: boolean) {
  form.wechat_connect_mp_enabled = value;
  if (value) form.wechat_connect_mobile_enabled = false;
  syncWeChatConnectMode(value ? "mp" : undefined);
}

function handleWeChatMobileEnabledChange(value: boolean) {
  form.wechat_connect_mobile_enabled = value;
  if (value) form.wechat_connect_mp_enabled = false;
  syncWeChatConnectMode(value ? "mobile" : undefined);
}

const wechatRedirectUrlSuggestion = computed(() => {
  if (typeof window === "undefined") return "";
  return `${window.location.origin}/api/v1/auth/oauth/wechat/callback`;
});

async function setAndCopyWeChatRedirectUrl() {
  const url = wechatRedirectUrlSuggestion.value;
  if (!url) return;
  form.wechat_connect_redirect_url = url;
  await copyToClipboard(url);
}

// Custom menu item management
function addMenuItem() {
  form.custom_menu_items.push({
    id: "",
    label: "",
    icon_svg: "",
    url: "",
    visibility: "user",
    sort_order: form.custom_menu_items.length,
  });
}

function removeMenuItem(index: number) {
  form.custom_menu_items.splice(index, 1);
  // Re-index sort_order
  form.custom_menu_items.forEach((item, i) => {
    item.sort_order = i;
  });
}

function moveMenuItem(index: number, direction: -1 | 1) {
  const targetIndex = index + direction;
  if (targetIndex < 0 || targetIndex >= form.custom_menu_items.length) return;
  const items = form.custom_menu_items;
  const temp = items[index];
  items[index] = items[targetIndex];
  items[targetIndex] = temp;
  // Re-index sort_order
  items.forEach((item, i) => {
    item.sort_order = i;
  });
}

function formatTLSNumberList(values: number[]): string {
  return values.length > 0 ? values.join(", ") : "";
}

function oidcTextFieldValue(key: OIDCTextFieldKey): string {
  return form[key];
}

function setOIDCTextFieldValue(key: OIDCTextFieldKey, event: Event): void {
  form[key] = (event.target as HTMLInputElement).value;
}

function parseTLSNumberList(
  raw: string,
  max: number,
  fieldKey: string,
): number[] {
  const trimmed = raw.trim();
  if (!trimmed) {
    return [];
  }
  const tokens = trimmed
    .split(/[,\s]+/)
    .map((item) => item.trim())
    .filter(Boolean);
  const unique: number[] = [];
  const seen = new Set<number>();
  for (const token of tokens) {
    if (!/^\d+$/.test(token)) {
      throw new Error(t(fieldKey, { value: token }));
    }
    const value = Number(token);
    if (!Number.isInteger(value) || value <= 0 || value > max) {
      throw new Error(t(fieldKey, { value: token }));
    }
    if (seen.has(value)) {
      continue;
    }
    seen.add(value);
    unique.push(value);
  }
  return unique;
}

function resetTLSFingerprintForm() {
  editingTLSFingerprintProfileID.value = "";
  tlsFingerprintForm.profile_id = "";
  tlsFingerprintForm.name = "";
  tlsFingerprintForm.enabled = true;
  tlsFingerprintForm.enable_grease = false;
  tlsFingerprintForm.cipher_suites_text = "";
  tlsFingerprintForm.curves_text = "";
  tlsFingerprintForm.point_formats_text = "";
}

function openCreateTLSFingerprintModal() {
  resetTLSFingerprintForm();
  showTLSFingerprintModal.value = true;
}

function openEditTLSFingerprintModal(profile: TLSFingerprintProfile) {
  editingTLSFingerprintProfileID.value = profile.profile_id;
  tlsFingerprintForm.profile_id = profile.profile_id;
  tlsFingerprintForm.name = profile.name;
  tlsFingerprintForm.enabled = profile.enabled;
  tlsFingerprintForm.enable_grease = profile.enable_grease;
  tlsFingerprintForm.cipher_suites_text = formatTLSNumberList(
    profile.cipher_suites,
  );
  tlsFingerprintForm.curves_text = formatTLSNumberList(profile.curves);
  tlsFingerprintForm.point_formats_text = formatTLSNumberList(
    profile.point_formats,
  );
  showTLSFingerprintModal.value = true;
}

function closeTLSFingerprintModal() {
  showTLSFingerprintModal.value = false;
  resetTLSFingerprintForm();
}

const codexSyncedVersionLabel = computed(() => {
  const synced = form.openai_codex_client_version_synced?.trim();
  if (!synced) return "";
  return t("admin.settings.gatewayForwarding.openaiCodexVersionSyncedValue", {
    version: synced,
  });
});

interface CodexClientRow {
  originator: string;
  uaContains: string;
  skipEngineFingerprint?: boolean;
}

const codexBlacklistRows = ref<CodexClientRow[]>([]);
const codexWhitelistRows = ref<CodexClientRow[]>([]);
const codexFingerprintRows = ref<FingerprintSignalRow[]>([]);

function parseCodexEntriesToRows(raw: string): CodexClientRow[] {
  if (!raw?.trim()) return [];
  try {
    const entries = JSON.parse(raw);
    if (!Array.isArray(entries)) return [];
    return entries.map((entry) => ({
      originator: typeof entry?.originator === "string" ? entry.originator : "",
      uaContains: Array.isArray(entry?.ua_contains)
        ? entry.ua_contains.filter((item: unknown) => typeof item === "string").join(", ")
        : "",
      skipEngineFingerprint: entry?.skip_engine_fingerprint === true,
    }));
  } catch {
    return [];
  }
}

function serializeCodexRowsToJSON(rows: CodexClientRow[]): string {
  const entries = rows.map((row) => {
    const entry: {
      originator: string;
      ua_contains: string[];
      skip_engine_fingerprint?: boolean;
    } = {
      originator: row.originator.trim(),
      ua_contains: row.uaContains.split(",").map((item) => item.trim()).filter(Boolean),
    };
    if (row.skipEngineFingerprint) entry.skip_engine_fingerprint = true;
    return entry;
  }).filter((entry) => entry.originator || entry.ua_contains.length > 0);
  return entries.length > 0 ? JSON.stringify(entries) : "";
}

const currentOrigin = typeof window !== "undefined" ? window.location.origin : "";

async function loadSettings() {
  loading.value = true;
  try {
    const settings = await adminAPI.settings.getSettings();
    const rawSettings = settings as SystemSettings & Record<string, unknown>;
    settings.payment_load_balance_strategy =
      settings.payment_load_balance_strategy || "round-robin";
    // Only assign non-null values from backend (null means unconfigured, keep defaults)
    for (const [key, value] of Object.entries(settings)) {
      if (value !== null && value !== undefined) {
        (form as Record<string, unknown>)[key] = value;
      }
    }
    syncCaptchaProviderSelection();
    if (!form.claude_oauth_system_prompt_blocks?.trim()) {
      form.claude_oauth_system_prompt_blocks =
        defaultClaudeOAuthSystemPromptBlocks;
    }
    claudeOAuthSystemPromptBlocks.value = parseClaudeOAuthSystemPromptBlocks(
      form.claude_oauth_system_prompt_blocks,
      form.claude_oauth_system_prompt,
    );
    syncClaudeOAuthSystemPromptBlocksFormField();
    codexBlacklistRows.value = parseCodexEntriesToRows(
      form.codex_cli_only_blacklist,
    );
    codexWhitelistRows.value = parseCodexEntriesToRows(
      form.codex_cli_only_whitelist,
    );
    codexFingerprintRows.value = form.codex_cli_only_engine_fingerprint_signals
      ? parseFingerprintSignalsToRows(form.codex_cli_only_engine_fingerprint_signals)
      : defaultFingerprintSignalRows();
    form.login_agreement_mode =
      settings.login_agreement_mode === "checkbox" ? "checkbox" : "modal";
    form.channel_monitor_mode =
      settings.channel_monitor_mode === "v2" ? "v2" : "v1";
    form.channel_monitor_hide_throughput = Boolean(
      settings.channel_monitor_hide_throughput
    );
    form.channel_monitor_show_quota = Boolean(
      settings.channel_monitor_show_quota
    );
    form.account_scheduling_thresholds = normalizeAccountSchedulingThresholdsMap(
      settings.account_scheduling_thresholds,
    );
    if (
      settings.openai_fast_policy_settings &&
      Array.isArray(settings.openai_fast_policy_settings.rules)
    ) {
      openaiFastPolicyForm.rules =
        settings.openai_fast_policy_settings.rules.map((rule) => ({
          ...rule,
          user_ids: rule.user_ids ? [...rule.user_ids] : [],
          model_whitelist: rule.model_whitelist
            ? [...rule.model_whitelist]
            : [],
        }));
      openaiFastPolicyLoaded.value = true;
    }
    form.login_agreement_updated_at =
      settings.login_agreement_updated_at || "2026-03-31";
    form.login_agreement_documents =
      Array.isArray(settings.login_agreement_documents) &&
      settings.login_agreement_documents.length > 0
        ? settings.login_agreement_documents.map((doc) => ({
            id: doc.id || "",
            title: doc.title || "",
            content_md: doc.content_md || "",
          }))
        : defaultLoginAgreementDocuments();
    Object.assign(authSourceDefaults, buildAuthSourceDefaultsState(settings));
    const wechatCapabilities = resolveWeChatConnectModeCapabilities(
      rawSettings.wechat_connect_open_enabled,
      rawSettings.wechat_connect_mp_enabled,
      rawSettings.wechat_connect_mobile_enabled,
      rawSettings.wechat_connect_mode,
    );
    form.wechat_connect_open_enabled = wechatCapabilities.openEnabled;
    form.wechat_connect_mp_enabled = wechatCapabilities.mpEnabled;
    form.wechat_connect_mobile_enabled = wechatCapabilities.mobileEnabled;
    form.wechat_connect_mode = deriveWeChatConnectStoredMode(
      wechatCapabilities.openEnabled,
      wechatCapabilities.mpEnabled,
      wechatCapabilities.mobileEnabled,
      rawSettings.wechat_connect_mode,
    );
    // Older installations stored one app id/secret. Keep it visible in the
    // selected mode until the administrator saves the split configuration.
    form.wechat_connect_open_app_id = String(
      rawSettings.wechat_connect_open_app_id ??
        rawSettings.wechat_connect_app_id ??
        "",
    );
    form.wechat_connect_mp_app_id = String(
      rawSettings.wechat_connect_mp_app_id ??
        rawSettings.wechat_connect_app_id ??
        "",
    );
    form.wechat_connect_mobile_app_id = String(
      rawSettings.wechat_connect_mobile_app_id ?? "",
    );
    form.wechat_connect_open_app_secret_configured =
      rawSettings.wechat_connect_open_app_secret_configured === true ||
      rawSettings.wechat_connect_app_secret_configured === true;
    form.wechat_connect_mp_app_secret_configured =
      rawSettings.wechat_connect_mp_app_secret_configured === true ||
      rawSettings.wechat_connect_app_secret_configured === true;
    form.wechat_connect_mobile_app_secret_configured =
      rawSettings.wechat_connect_mobile_app_secret_configured === true;
    form.wechat_connect_scopes = String(
      rawSettings.wechat_connect_scopes ??
        defaultWeChatConnectScopesForMode(form.wechat_connect_mode),
    );
    form.default_platform_quotas = normalizePlatformQuotasMap(settings.default_platform_quotas);
    form.backend_mode_enabled = settings.backend_mode_enabled;
    form.default_subscriptions = Array.isArray(settings.default_subscriptions)
      ? settings.default_subscriptions
          .filter((item) => item.group_id > 0 && item.validity_days > 0)
          .map((item) => ({
            group_id: item.group_id,
            validity_days: item.validity_days,
          }))
      : [];
    registrationEmailSuffixWhitelistTags.value =
      normalizeRegistrationEmailSuffixDomains(
        settings.registration_email_suffix_whitelist,
      );
    form.forwarded_client_ip_headers = normalizeForwardedClientIpHeaders(
      settings.forwarded_client_ip_headers,
    );
    forwardedClientIpHeaderDraft.value = "";
    registrationEmailSuffixWhitelistDraft.value = "";
    form.smtp_password = "";
    form.turnstile_secret_key = "";
    form.tencent_captcha_app_secret_key = "";
    form.tencent_captcha_cloud_secret_id = "";
    form.tencent_captcha_cloud_secret_key = "";
    form.aliyun_captcha_access_key_secret = "";
    form.linuxdo_connect_client_secret = "";
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.failedToLoad") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    loading.value = false;
  }
}

async function loadSubscriptionGroups() {
  try {
    const groups = await adminAPI.groups.getAll();
    subscriptionGroups.value = groups.filter(
      (group) =>
        group.subscription_type === "subscription" && group.status === "active",
    );
  } catch (error) {
    console.error("Failed to load subscription groups:", error);
    subscriptionGroups.value = [];
  }
}

function normalizePaymentProvider(provider: ProviderInstance): ProviderInstance {
  return {
    ...provider,
    supported_types: Array.isArray(provider.supported_types)
      ? provider.supported_types
      : [],
  };
}

async function loadPaymentProviders() {
  providersLoading.value = true;
  try {
    const response = await adminAPI.payment.getProviders();
    providers.value = (response.data ?? []).map(normalizePaymentProvider);
  } catch (error) {
    console.error("Failed to load payment providers:", error);
    providers.value = [];
  } finally {
    providersLoading.value = false;
  }
}

async function openCreateProvider() {
  editingProvider.value = null;
  showProviderDialog.value = true;
  await nextTick();
  providerDialogRef.value?.reset(providerKeyOptions[0]?.value ?? "");
}

async function openEditProvider(provider: ProviderInstance) {
  editingProvider.value = provider;
  showProviderDialog.value = true;
  await nextTick();
  providerDialogRef.value?.loadProvider(provider);
}

function closeProviderDialog() {
  showProviderDialog.value = false;
  editingProvider.value = null;
}

function confirmDeleteProvider(provider: ProviderInstance) {
  deletingProvider.value = provider;
}

function cancelDeleteProvider() {
  deletingProvider.value = null;
}

async function deleteProvider() {
  const provider = deletingProvider.value;
  if (!provider) return;
  try {
    await stepUp.run(() => adminAPI.payment.deleteProvider(provider.id));
    appStore.showSuccess(t("admin.settings.settingsSaved"));
    await loadPaymentProviders();
  } catch (error: any) {
    if (isStepUpCancelled(error)) return;
    appStore.showError(error?.message || t("common.unknownError"));
  } finally {
    deletingProvider.value = null;
  }
}

async function saveProvider(payload: ProviderPayload) {
  savingProvider.value = true;
  try {
    if (editingProvider.value) {
      await stepUp.run(() => adminAPI.payment.updateProvider(editingProvider.value!.id, payload));
    } else {
      await stepUp.run(() => adminAPI.payment.createProvider(payload));
    }
    appStore.showSuccess(t("admin.settings.settingsSaved"));
    closeProviderDialog();
    await loadPaymentProviders();
  } catch (error: any) {
    if (isStepUpCancelled(error)) return;
    appStore.showError(error?.message || t("common.unknownError"));
  } finally {
    savingProvider.value = false;
  }
}

async function toggleProviderField(
  provider: ProviderInstance,
  field: "enabled" | "refund_enabled" | "allow_user_refund",
) {
  try {
    await stepUp.run(() => adminAPI.payment.updateProvider(provider.id, {
      [field]: !provider[field],
    }));
    await loadPaymentProviders();
  } catch (error: any) {
    if (isStepUpCancelled(error)) return;
    appStore.showError(error?.message || t("common.unknownError"));
  }
}

async function toggleProviderType(provider: ProviderInstance, type: string) {
  const supportedTypes = Array.isArray(provider.supported_types)
    ? provider.supported_types
    : [];
  const nextTypes = supportedTypes.includes(type)
    ? supportedTypes.filter((value) => value !== type)
    : [...supportedTypes, type];
  try {
    await stepUp.run(() => adminAPI.payment.updateProvider(provider.id, {
      supported_types: nextTypes,
    }));
    await loadPaymentProviders();
  } catch (error: any) {
    if (isStepUpCancelled(error)) return;
    appStore.showError(error?.message || t("common.unknownError"));
  }
}

async function reorderProviders(items: { id: number; sort_order: number }[]) {
  try {
    await stepUp.run(() => Promise.all(
      items.map((item) =>
        adminAPI.payment.updateProvider(item.id, { sort_order: item.sort_order }),
      ),
    ));
    await loadPaymentProviders();
  } catch (error: any) {
    if (isStepUpCancelled(error)) return;
    appStore.showError(error?.message || t("common.unknownError"));
  }
}

async function loadUpstreamBillingProbeSettings() {
  upstreamBillingProbeLoading.value = true;
  try {
    Object.assign(
      upstreamBillingProbeForm,
      await adminAPI.accounts.getUpstreamBillingProbeSettings(),
    );
  } catch (error) {
    console.error("Failed to load upstream billing probe settings:", error);
  } finally {
    upstreamBillingProbeLoading.value = false;
  }
}

async function saveUpstreamBillingProbeSettings() {
  upstreamBillingProbeSaving.value = true;
  try {
    const updated = await adminAPI.accounts.updateUpstreamBillingProbeSettings({
      enabled: upstreamBillingProbeForm.enabled,
      interval_minutes: Number(upstreamBillingProbeForm.interval_minutes) || 60,
    });
    Object.assign(upstreamBillingProbeForm, updated);
    appStore.showSuccess(t("admin.settings.upstreamBillingProbe.saved"));
  } catch (error: any) {
    appStore.showError(
      error?.message || t("admin.settings.upstreamBillingProbe.saveFailed"),
    );
  } finally {
    upstreamBillingProbeSaving.value = false;
  }
}

async function loadOllamaCloudUsageSettings() {
  ollamaCloudUsageLoading.value = true;
  try {
    Object.assign(
      ollamaCloudUsageForm,
      await adminAPI.accounts.getOllamaCloudUsageSettings(),
    );
  } catch (error) {
    console.error("Failed to load Ollama Cloud usage settings:", error);
  } finally {
    ollamaCloudUsageLoading.value = false;
  }
}

async function saveOllamaCloudUsageSettings() {
  ollamaCloudUsageSaving.value = true;
  try {
    const updated = await adminAPI.accounts.updateOllamaCloudUsageSettings({
      enabled: ollamaCloudUsageForm.enabled,
      interval_minutes: Number(ollamaCloudUsageForm.interval_minutes) || 60,
      debounce_minutes: Number(ollamaCloudUsageForm.debounce_minutes) || 1,
    });
    Object.assign(ollamaCloudUsageForm, updated);
    appStore.showSuccess(t("admin.settings.ollamaCloudUsage.saved"));
  } catch (error: any) {
    appStore.showError(
      error?.message || t("admin.settings.ollamaCloudUsage.saveFailed"),
    );
  } finally {
    ollamaCloudUsageSaving.value = false;
  }
}

function schedulerFieldValue(key: OpenAIAdvancedSchedulerOverrideKey): string {
  return String((form as Record<string, unknown>)[key] ?? "");
}

function setSchedulerFieldFromEvent(
  key: OpenAIAdvancedSchedulerOverrideKey,
  event: Event,
) {
  const target = event.target as HTMLInputElement | null;
  (form as Record<string, unknown>)[key] = target?.value ?? "";
}

function addDefaultSubscription() {
  if (subscriptionGroups.value.length === 0) return;
  const existing = new Set(
    form.default_subscriptions.map((item) => item.group_id),
  );
  const candidate = subscriptionGroups.value.find(
    (group) => !existing.has(group.id),
  );
  if (!candidate) return;
  form.default_subscriptions.push({
    group_id: candidate.id,
    validity_days: 30,
  });
}

function removeDefaultSubscription(index: number) {
  form.default_subscriptions.splice(index, 1);
}

async function saveSettings() {
  saving.value = true;
  try {
    const normalizedDefaultSubscriptions = form.default_subscriptions
      .filter((item) => item.group_id > 0 && item.validity_days > 0)
      .map((item: DefaultSubscriptionSetting) => ({
        group_id: item.group_id,
        validity_days: Math.min(
          36500,
          Math.max(1, Math.floor(item.validity_days)),
        ),
      }));

    const seenGroupIDs = new Set<number>();
    const duplicateDefaultSubscription = normalizedDefaultSubscriptions.find(
      (item) => {
        if (seenGroupIDs.has(item.group_id)) {
          return true;
        }
        seenGroupIDs.add(item.group_id);
        return false;
      },
    );
    if (duplicateDefaultSubscription) {
      appStore.showError(
        t("admin.settings.defaults.defaultSubscriptionsDuplicate", {
          groupId: duplicateDefaultSubscription.group_id,
        }),
      );
      return;
    }

    // frontend_url drives password reset links. Never silently discard what the admin typed:
    // a cleared value looks saved but puts password reset right back into silent-failure mode.
    if (!isValidHttpUrl(form.frontend_url)) {
      appStore.showError(
        t("admin.settings.registration.frontendUrlInvalidError"),
      );
      return;
    }

    // Optional URL fields: auto-clear invalid values so they don't cause backend 400 errors
    if (!isValidHttpUrl(form.doc_url)) form.doc_url = "";
    if (!isValidHttpUrl(form.purchase_link_cny_10))
      form.purchase_link_cny_10 = "";
    if (!isValidHttpUrl(form.purchase_link_cny_30))
      form.purchase_link_cny_30 = "";
    if (!isValidHttpUrl(form.purchase_link_cny_100))
      form.purchase_link_cny_100 = "";
    // Purchase URL: required when enabled; auto-clear when disabled to avoid backend rejection
    if (form.purchase_subscription_enabled) {
      if (!form.purchase_subscription_url) {
        appStore.showError(
          t("admin.settings.purchase.url") +
            ": URL is required when purchase is enabled",
        );
        saving.value = false;
        return;
      }
      if (!isValidHttpUrl(form.purchase_subscription_url)) {
        appStore.showError(
          t("admin.settings.purchase.url") +
            ": must be an absolute http(s) URL (e.g. https://example.com)",
        );
        saving.value = false;
        return;
      }
    } else if (!isValidHttpUrl(form.purchase_subscription_url)) {
      form.purchase_subscription_url = "";
    }

    const payload: UpdateSettingsRequest = {
      registration_enabled: form.registration_enabled,
      email_verify_enabled: form.email_verify_enabled,
      registration_email_suffix_whitelist:
        registrationEmailSuffixWhitelistTags.value.map(
          (suffix) => `@${suffix}`,
        ),
      promo_code_enabled: form.promo_code_enabled,
      invitation_code_enabled: form.invitation_code_enabled,
      // form.password_reset_enabled 在 syncPasswordResetStored 里已被种成 DB 原始存储值，
      // 邮箱验证关闭时开关不渲染、无人能改它，因此原样回传不会把 DB 里的 true 抹掉。
      // 生效语义不受影响：后端仍与 email_verify_enabled 取与。
      password_reset_enabled: form.password_reset_enabled,
      totp_enabled: form.totp_enabled,
      passkey_enabled: form.passkey_enabled,
      default_balance: form.default_balance,
      default_concurrency: form.default_concurrency,
      default_subscriptions: normalizedDefaultSubscriptions,
      default_platform_quotas: sanitizePlatformQuotasMap(
        form.default_platform_quotas,
      ),
      affiliate_enabled: form.affiliate_enabled,
      affiliate_admin_recharge_enabled: form.affiliate_admin_recharge_enabled,
      affiliate_rebate_rate: form.affiliate_rebate_rate,
      affiliate_rebate_freeze_hours: form.affiliate_rebate_freeze_hours,
      affiliate_rebate_duration_days: form.affiliate_rebate_duration_days,
      affiliate_rebate_per_invitee_cap: form.affiliate_rebate_per_invitee_cap,
      site_name: form.site_name,
      site_logo: form.site_logo,
      site_subtitle: form.site_subtitle,
      api_base_url: form.api_base_url,
      contact_info: form.contact_info,
      doc_url: form.doc_url,
      home_content: form.home_content,
      compact_home_enabled: form.compact_home_enabled,
      backend_mode_enabled: form.backend_mode_enabled,
      hide_ccs_import_button: form.hide_ccs_import_button,
      purchase_subscription_enabled: form.purchase_subscription_enabled,
      purchase_subscription_url: form.purchase_subscription_url,
      purchase_link_cny_10: form.purchase_link_cny_10,
      purchase_link_cny_30: form.purchase_link_cny_30,
      purchase_link_cny_100: form.purchase_link_cny_100,
      sora_client_enabled: form.sora_client_enabled,
      custom_menu_items: form.custom_menu_items,
      frontend_url: form.frontend_url,
      smtp_host: form.smtp_host,
      smtp_port: form.smtp_port,
      smtp_username: form.smtp_username,
      smtp_password: form.smtp_password || undefined,
      smtp_from_email: form.smtp_from_email,
      smtp_from_name: form.smtp_from_name,
      smtp_use_tls: form.smtp_use_tls,
      turnstile_enabled: form.turnstile_enabled,
      turnstile_site_key: form.turnstile_site_key,
      turnstile_secret_key: form.turnstile_secret_key || undefined,
      tencent_captcha_enabled: form.tencent_captcha_enabled,
      tencent_captcha_app_id: form.tencent_captcha_app_id,
      tencent_captcha_app_secret_key:
        form.tencent_captcha_app_secret_key || undefined,
      tencent_captcha_cloud_secret_id:
        form.tencent_captcha_cloud_secret_id || undefined,
      tencent_captcha_cloud_secret_key:
        form.tencent_captcha_cloud_secret_key || undefined,
      tencent_captcha_region: form.tencent_captcha_region,
      aliyun_captcha_enabled: form.aliyun_captcha_enabled,
      aliyun_captcha_access_key_id: form.aliyun_captcha_access_key_id,
      aliyun_captcha_access_key_secret:
        form.aliyun_captcha_access_key_secret || undefined,
      aliyun_captcha_scene_id: form.aliyun_captcha_scene_id,
      aliyun_captcha_prefix: form.aliyun_captcha_prefix,
      aliyun_captcha_region: form.aliyun_captcha_region,
      api_key_acl_trust_forwarded_ip: form.api_key_acl_trust_forwarded_ip,
      forwarded_client_ip_headers: form.forwarded_client_ip_headers,
      linuxdo_connect_enabled: form.linuxdo_connect_enabled,
      linuxdo_connect_client_id: form.linuxdo_connect_client_id,
      linuxdo_connect_client_secret:
        form.linuxdo_connect_client_secret || undefined,
      linuxdo_connect_redirect_url: form.linuxdo_connect_redirect_url,
      wechat_connect_enabled: form.wechat_connect_enabled,
      wechat_connect_app_id:
        form.wechat_connect_mp_app_id ||
        form.wechat_connect_open_app_id ||
        form.wechat_connect_mobile_app_id,
      wechat_connect_app_secret: form.wechat_connect_app_secret || undefined,
      wechat_connect_open_app_id: form.wechat_connect_open_app_id,
      wechat_connect_open_app_secret:
        form.wechat_connect_open_app_secret || undefined,
      wechat_connect_mp_app_id: form.wechat_connect_mp_app_id,
      wechat_connect_mp_app_secret:
        form.wechat_connect_mp_app_secret || undefined,
      wechat_connect_mobile_app_id: form.wechat_connect_mobile_app_id,
      wechat_connect_mobile_app_secret:
        form.wechat_connect_mobile_app_secret || undefined,
      wechat_connect_open_enabled: form.wechat_connect_open_enabled,
      wechat_connect_mp_enabled: form.wechat_connect_mp_enabled,
      wechat_connect_mobile_enabled: form.wechat_connect_mobile_enabled,
      wechat_connect_mode: form.wechat_connect_mode,
      wechat_connect_scopes: form.wechat_connect_scopes,
      wechat_connect_redirect_url: form.wechat_connect_redirect_url,
      wechat_connect_frontend_redirect_url:
        form.wechat_connect_frontend_redirect_url,
      oidc_connect_enabled: form.oidc_connect_enabled,
      oidc_connect_provider_name: form.oidc_connect_provider_name,
      oidc_connect_client_id: form.oidc_connect_client_id,
      oidc_connect_client_secret: form.oidc_connect_client_secret || undefined,
      oidc_connect_issuer_url: form.oidc_connect_issuer_url,
      oidc_connect_discovery_url: form.oidc_connect_discovery_url,
      oidc_connect_authorize_url: form.oidc_connect_authorize_url,
      oidc_connect_token_url: form.oidc_connect_token_url,
      oidc_connect_userinfo_url: form.oidc_connect_userinfo_url,
      oidc_connect_jwks_url: form.oidc_connect_jwks_url,
      oidc_connect_scopes: form.oidc_connect_scopes,
      oidc_connect_redirect_url: form.oidc_connect_redirect_url,
      oidc_connect_frontend_redirect_url:
        form.oidc_connect_frontend_redirect_url,
      oidc_connect_token_auth_method: form.oidc_connect_token_auth_method,
      oidc_connect_use_pkce: form.oidc_connect_use_pkce,
      oidc_connect_validate_id_token: form.oidc_connect_validate_id_token,
      oidc_connect_allowed_signing_algs:
        form.oidc_connect_allowed_signing_algs,
      oidc_connect_clock_skew_seconds: form.oidc_connect_clock_skew_seconds,
      oidc_connect_require_email_verified:
        form.oidc_connect_require_email_verified,
      oidc_connect_userinfo_email_path: form.oidc_connect_userinfo_email_path,
      oidc_connect_userinfo_id_path: form.oidc_connect_userinfo_id_path,
      oidc_connect_userinfo_username_path:
        form.oidc_connect_userinfo_username_path,
      enable_model_fallback: form.enable_model_fallback,
      fallback_model_anthropic: form.fallback_model_anthropic,
      fallback_model_openai: form.fallback_model_openai,
        fallback_model_gemini: form.fallback_model_gemini,
        fallback_model_antigravity: form.fallback_model_antigravity,
        grok_default_text_model:
          form.grok_default_text_model?.trim() || "grok-4.5",
        grok_cross_client_model_map_enabled:
          form.grok_cross_client_model_map_enabled,
        enable_identity_patch: form.enable_identity_patch,
      identity_patch_prompt: form.identity_patch_prompt,
      min_claude_code_version: form.min_claude_code_version,
      max_claude_code_version: form.max_claude_code_version,
      allow_ungrouped_key_scheduling: form.allow_ungrouped_key_scheduling,
      enable_fingerprint_unification: form.enable_fingerprint_unification,
      enable_metadata_passthrough: form.enable_metadata_passthrough,
      enable_cch_signing: form.enable_cch_signing,
      enable_claude_oauth_system_prompt_injection:
        form.enable_claude_oauth_system_prompt_injection,
      claude_oauth_system_prompt: form.claude_oauth_system_prompt?.trim()
        ? form.claude_oauth_system_prompt
        : "",
      claude_oauth_system_prompt_blocks:
        serializeClaudeOAuthSystemPromptBlocksToJSON(claudeOAuthSystemPromptBlocks.value),
      enable_anthropic_cache_ttl_1h_injection:
        form.enable_anthropic_cache_ttl_1h_injection,
      rewrite_message_cache_control: form.rewrite_message_cache_control,
      enable_client_dateline_normalization:
        form.enable_client_dateline_normalization,
      antigravity_user_agent_version:
        form.antigravity_user_agent_version?.trim() || "",
      openai_codex_user_agent:
        form.openai_codex_user_agent?.trim() || "",
      openai_codex_client_version:
        form.openai_codex_client_version?.trim() || "",
      openai_codex_version_auto_sync_enabled:
        form.openai_codex_version_auto_sync_enabled,
      min_codex_version: form.min_codex_version?.trim() || "",
      max_codex_version: form.max_codex_version?.trim() || "",
      codex_cli_only_allow_app_server_clients:
        form.codex_cli_only_allow_app_server_clients,
      codex_cli_only_engine_fingerprint_signals: serializeFingerprintRowsToJSON(
        codexFingerprintRows.value,
      ),
      codex_cli_only_blacklist: serializeCodexRowsToJSON(
        codexBlacklistRows.value,
      ),
      codex_cli_only_whitelist: serializeCodexRowsToJSON(
        codexWhitelistRows.value,
      ),
      // Payment configuration
      payment_enabled: form.payment_enabled,
      risk_control_enabled: form.risk_control_enabled,
      cyber_session_block_enabled: form.cyber_session_block_enabled,
      cyber_session_block_ttl_seconds:
        Number(form.cyber_session_block_ttl_seconds) || 3600,
      payment_min_amount: Number(form.payment_min_amount) || 0,
      payment_max_amount: Number(form.payment_max_amount) || 0,
      payment_daily_limit: Number(form.payment_daily_limit) || 0,
      payment_max_pending_orders: Number(form.payment_max_pending_orders) || 0,
      payment_order_timeout_minutes:
        Number(form.payment_order_timeout_minutes) || 0,
      payment_balance_disabled: form.payment_balance_disabled,
      payment_balance_recharge_multiplier:
        Number(form.payment_balance_recharge_multiplier) || 1,
      payment_subscription_usd_to_cny_rate:
        Number(form.payment_subscription_usd_to_cny_rate) || 0,
      payment_recharge_fee_rate: Number(form.payment_recharge_fee_rate) || 0,
      payment_enabled_types: form.payment_enabled_types,
      payment_load_balance_strategy: form.payment_load_balance_strategy,
      payment_product_name_prefix: form.payment_product_name_prefix,
      payment_product_name_suffix: form.payment_product_name_suffix,
      payment_help_image_url: form.payment_help_image_url,
      payment_help_text: form.payment_help_text,
      payment_cancel_rate_limit_enabled: form.payment_cancel_rate_limit_enabled,
      payment_cancel_rate_limit_max:
        Number(form.payment_cancel_rate_limit_max) || 10,
      payment_cancel_rate_limit_window:
        Number(form.payment_cancel_rate_limit_window) || 1,
      payment_cancel_rate_limit_unit: form.payment_cancel_rate_limit_unit,
      payment_cancel_rate_limit_window_mode:
        form.payment_cancel_rate_limit_window_mode,
      payment_alipay_force_qrcode: form.payment_alipay_force_qrcode,
      payment_alipay_mobile_precreate_deep_link:
        form.payment_alipay_mobile_precreate_deep_link,
      openai_low_upstream_rate_priority_enabled:
        form.openai_low_upstream_rate_priority_enabled,
      openai_oauth_scheduling_rate_multiplier:
        form.openai_oauth_scheduling_rate_multiplier,
      openai_advanced_scheduler_enabled: form.openai_advanced_scheduler_enabled,
      openai_advanced_scheduler_sticky_weighted_enabled:
        form.openai_advanced_scheduler_sticky_weighted_enabled,
      openai_advanced_scheduler_subscription_priority_enabled:
        form.openai_advanced_scheduler_subscription_priority_enabled,
      openai_advanced_scheduler_lb_top_k:
        String(form.openai_advanced_scheduler_lb_top_k ?? "").trim(),
      openai_advanced_scheduler_weight_priority:
        String(form.openai_advanced_scheduler_weight_priority ?? "").trim(),
      openai_advanced_scheduler_weight_load:
        String(form.openai_advanced_scheduler_weight_load ?? "").trim(),
      openai_advanced_scheduler_weight_queue:
        String(form.openai_advanced_scheduler_weight_queue ?? "").trim(),
      openai_advanced_scheduler_weight_error_rate:
        String(form.openai_advanced_scheduler_weight_error_rate ?? "").trim(),
      openai_advanced_scheduler_weight_ttft:
        String(form.openai_advanced_scheduler_weight_ttft ?? "").trim(),
      openai_advanced_scheduler_weight_reset:
        String(form.openai_advanced_scheduler_weight_reset ?? "").trim(),
      openai_advanced_scheduler_weight_quota_headroom:
        String(form.openai_advanced_scheduler_weight_quota_headroom ?? "").trim(),
      openai_advanced_scheduler_weight_upstream_cost:
        String(form.openai_advanced_scheduler_weight_upstream_cost ?? "").trim(),
      openai_advanced_scheduler_weight_previous_response:
        String(form.openai_advanced_scheduler_weight_previous_response ?? "").trim(),
      openai_advanced_scheduler_weight_session_sticky:
        String(form.openai_advanced_scheduler_weight_session_sticky ?? "").trim(),
      // 余额、订阅到期与账号限额通知
      balance_low_notify_enabled: form.balance_low_notify_enabled,
      balance_low_notify_threshold:
        Number(form.balance_low_notify_threshold) || 0,
      balance_low_notify_recharge_url: (form.balance_low_notify_recharge_url =
        form.balance_low_notify_recharge_url || currentOrigin),
      subscription_expiry_notify_enabled:
        form.subscription_expiry_notify_enabled,
      account_quota_notify_enabled: form.account_quota_notify_enabled,
      account_quota_notify_emails: (
        form.account_quota_notify_emails || []
      ).filter((e) => e.email.trim() !== ""),
      // Channel Monitor feature switch
      channel_monitor_enabled: form.channel_monitor_enabled,
      channel_monitor_mode: form.channel_monitor_mode,
      channel_monitor_default_interval_seconds:
        Number(form.channel_monitor_default_interval_seconds) || 60,
      channel_monitor_hide_throughput: Boolean(form.channel_monitor_hide_throughput),
      channel_monitor_show_quota: Boolean(form.channel_monitor_show_quota),
      // Available Channels feature switch
      available_channels_enabled: form.available_channels_enabled,
      // Model Plaza feature switches + description
      model_plaza_enabled: form.model_plaza_enabled,
      model_plaza_require_auth: form.model_plaza_require_auth,
      model_plaza_description: form.model_plaza_description,
    };
    if (openaiFastPolicyLoaded.value) {
      payload.openai_fast_policy_settings = {
        rules: openaiFastPolicyForm.rules.map((rule) => {
          const modelWhitelist = (rule.model_whitelist || [])
            .map((pattern) => pattern.trim())
            .filter(Boolean);
          const hasModelWhitelist = modelWhitelist.length > 0;
          return {
            service_tier: rule.service_tier,
            action: rule.action,
            scope: rule.scope,
            user_ids:
              rule.user_ids && rule.user_ids.length > 0
                ? [...rule.user_ids]
                : undefined,
            error_message:
              rule.action === "block" ? rule.error_message : undefined,
            model_whitelist: hasModelWhitelist ? modelWhitelist : undefined,
            fallback_action: hasModelWhitelist
              ? rule.fallback_action || "pass"
              : undefined,
            fallback_error_message:
              hasModelWhitelist && rule.fallback_action === "block"
                ? rule.fallback_error_message
                : undefined,
          };
        }),
      };
    }
    payload.account_scheduling_thresholds = sanitizeAccountSchedulingThresholdsMap(
      form.account_scheduling_thresholds,
    );
    appendAuthSourceDefaultsToUpdateRequest(payload, authSourceDefaults);
    const updated = await stepUp.run(() => adminAPI.settings.updateSettings(payload));
    if (!(await saveWebSearchConfig())) return;
    Object.assign(form, updated);
    form.account_scheduling_thresholds = normalizeAccountSchedulingThresholdsMap(
      updated.account_scheduling_thresholds,
    );
    if (
      updated.openai_fast_policy_settings &&
      Array.isArray(updated.openai_fast_policy_settings.rules)
    ) {
      openaiFastPolicyForm.rules =
        updated.openai_fast_policy_settings.rules.map((rule) => ({
          ...rule,
          user_ids: rule.user_ids ? [...rule.user_ids] : [],
          model_whitelist: rule.model_whitelist
            ? [...rule.model_whitelist]
            : [],
        }));
      openaiFastPolicyLoaded.value = true;
    }
    // Object.assign 会把 form.password_reset_enabled 刷成取与后的值；
    // 告警判据必须跟着落库的原始值走，否则保存成功那一刻告警就凭空消失，而 DB 里隐患还在。
    syncPasswordResetStored(updated);
    syncPasswordResetLinkBase(updated);
    registrationEmailSuffixWhitelistTags.value =
      normalizeRegistrationEmailSuffixDomains(
        updated.registration_email_suffix_whitelist,
      );
    registrationEmailSuffixWhitelistDraft.value = "";
    form.smtp_password = "";
    form.turnstile_secret_key = "";
    form.aliyun_captcha_access_key_secret = "";
    form.linuxdo_connect_client_secret = "";
    form.wechat_connect_app_secret = "";
    form.wechat_connect_open_app_secret = "";
    form.wechat_connect_mp_app_secret = "";
    form.wechat_connect_mobile_app_secret = "";
    form.oidc_connect_client_secret = "";
    // Refresh cached settings so sidebar/header update immediately
    await appStore.fetchPublicSettings(true);
    await adminSettingsStore.fetch(true);
    appStore.showSuccess(t("admin.settings.settingsSaved"));
  } catch (error: any) {
    if (isStepUpCancelled(error)) return;
    appStore.showError(
      t("admin.settings.failedToSave") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    saving.value = false;
  }
}

async function testSmtpConnection() {
  testingSmtp.value = true;
  try {
    const result = await adminAPI.settings.testSmtpConnection({
      smtp_host: form.smtp_host,
      smtp_port: form.smtp_port,
      smtp_username: form.smtp_username,
      smtp_password: form.smtp_password,
      smtp_use_tls: form.smtp_use_tls,
    });
    // API returns { message: "..." } on success, errors are thrown as exceptions
    appStore.showSuccess(
      result.message || t("admin.settings.smtpConnectionSuccess"),
    );
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.failedToTestSmtp") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    testingSmtp.value = false;
  }
}

async function sendTestEmail() {
  if (!testEmailAddress.value) {
    appStore.showError(t("admin.settings.testEmail.enterRecipientHint"));
    return;
  }

  sendingTestEmail.value = true;
  try {
    const result = await adminAPI.settings.sendTestEmail({
      email: testEmailAddress.value,
      smtp_host: form.smtp_host,
      smtp_port: form.smtp_port,
      smtp_username: form.smtp_username,
      smtp_password: form.smtp_password,
      smtp_from_email: form.smtp_from_email,
      smtp_from_name: form.smtp_from_name,
      smtp_use_tls: form.smtp_use_tls,
    });
    // API returns { message: "..." } on success, errors are thrown as exceptions
    appStore.showSuccess(result.message || t("admin.settings.testEmailSent"));
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.failedToSendTestEmail") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    sendingTestEmail.value = false;
  }
}

// Admin API Key 方法
async function loadAdminApiKey() {
  adminApiKeyLoading.value = true;
  try {
    const status = await adminAPI.settings.getAdminApiKey();
    adminApiKeyExists.value = status.exists;
    adminApiKeyMasked.value = status.masked_key;
  } catch (error: any) {
    console.error("Failed to load admin API key status:", error);
  } finally {
    adminApiKeyLoading.value = false;
  }
}

async function createAdminApiKey() {
  adminApiKeyOperating.value = true;
  try {
    const result = await adminAPI.settings.regenerateAdminApiKey();
    newAdminApiKey.value = result.key;
    adminApiKeyExists.value = true;
    adminApiKeyMasked.value =
      result.key.substring(0, 10) + "..." + result.key.slice(-4);
    appStore.showSuccess(t("admin.settings.adminApiKey.keyGenerated"));
  } catch (error: any) {
    appStore.showError(error.message || t("common.error"));
  } finally {
    adminApiKeyOperating.value = false;
  }
}

async function regenerateAdminApiKey() {
  if (!confirm(t("admin.settings.adminApiKey.regenerateConfirm"))) return;
  await createAdminApiKey();
}

async function deleteAdminApiKey() {
  if (!confirm(t("admin.settings.adminApiKey.deleteConfirm"))) return;
  adminApiKeyOperating.value = true;
  try {
    await adminAPI.settings.deleteAdminApiKey();
    adminApiKeyExists.value = false;
    adminApiKeyMasked.value = "";
    newAdminApiKey.value = "";
    appStore.showSuccess(t("admin.settings.adminApiKey.keyDeleted"));
  } catch (error: any) {
    appStore.showError(error.message || t("common.error"));
  } finally {
    adminApiKeyOperating.value = false;
  }
}

function copyNewKey() {
  navigator.clipboard
    .writeText(newAdminApiKey.value)
    .then(() => {
      appStore.showSuccess(t("admin.settings.adminApiKey.keyCopied"));
    })
    .catch(() => {
      appStore.showError(t("common.copyFailed"));
    });
}

// Overload Cooldown 方法
async function loadOverloadCooldownSettings() {
  overloadCooldownLoading.value = true;
  try {
    const settings = await adminAPI.settings.getOverloadCooldownSettings();
    Object.assign(overloadCooldownForm, settings);
  } catch (error: any) {
    console.error("Failed to load overload cooldown settings:", error);
  } finally {
    overloadCooldownLoading.value = false;
  }
}

async function saveOverloadCooldownSettings() {
  overloadCooldownSaving.value = true;
  try {
    const updated = await adminAPI.settings.updateOverloadCooldownSettings({
      enabled: overloadCooldownForm.enabled,
      cooldown_minutes: overloadCooldownForm.cooldown_minutes,
    });
    Object.assign(overloadCooldownForm, updated);
    appStore.showSuccess(t("admin.settings.overloadCooldown.saved"));
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.overloadCooldown.saveFailed") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    overloadCooldownSaving.value = false;
  }
}

// Stream Timeout 方法
async function loadStreamTimeoutSettings() {
  streamTimeoutLoading.value = true;
  try {
    const settings = await adminAPI.settings.getStreamTimeoutSettings();
    Object.assign(streamTimeoutForm, settings);
  } catch (error: any) {
    console.error("Failed to load stream timeout settings:", error);
  } finally {
    streamTimeoutLoading.value = false;
  }
}

async function saveStreamTimeoutSettings() {
  streamTimeoutSaving.value = true;
  try {
    const updated = await adminAPI.settings.updateStreamTimeoutSettings({
      enabled: streamTimeoutForm.enabled,
      action: streamTimeoutForm.action,
      temp_unsched_minutes: streamTimeoutForm.temp_unsched_minutes,
      threshold_count: streamTimeoutForm.threshold_count,
      threshold_window_minutes: streamTimeoutForm.threshold_window_minutes,
    });
    Object.assign(streamTimeoutForm, updated);
    appStore.showSuccess(t("admin.settings.streamTimeout.saved"));
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.streamTimeout.saveFailed") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    streamTimeoutSaving.value = false;
  }
}

async function loadPanelRateLimitSettings() {
  panelRateLimitLoading.value = true;
  try {
    const settings = await adminAPI.settings.getPanelRateLimitSettings();
    Object.assign(panelRateLimitForm, settings);
  } catch (error: any) {
    console.error("Failed to load panel rate limit settings:", error);
  } finally {
    panelRateLimitLoading.value = false;
  }
}

async function savePanelRateLimitSettings() {
  panelRateLimitSaving.value = true;
  try {
    const updated = await adminAPI.settings.updatePanelRateLimitSettings({
      enabled: panelRateLimitForm.enabled,
      user_rpm: panelRateLimitForm.user_rpm,
      heavy_rpm: panelRateLimitForm.heavy_rpm,
      exempt_admin: panelRateLimitForm.exempt_admin,
      public_ip_rpm: panelRateLimitForm.public_ip_rpm,
    });
    Object.assign(panelRateLimitForm, updated);
    appStore.showSuccess(t("admin.settings.panelRateLimit.saved"));
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.panelRateLimit.saveFailed") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    panelRateLimitSaving.value = false;
  }
}

// Rectifier 方法
async function loadRectifierSettings() {
  rectifierLoading.value = true;
  try {
    const settings = await adminAPI.settings.getRectifierSettings();
    Object.assign(rectifierForm, settings);
  } catch (error: any) {
    console.error("Failed to load rectifier settings:", error);
  } finally {
    rectifierLoading.value = false;
  }
}

async function saveRectifierSettings() {
  rectifierSaving.value = true;
  try {
    const updated = await adminAPI.settings.updateRectifierSettings({
      enabled: rectifierForm.enabled,
      thinking_signature_enabled: rectifierForm.thinking_signature_enabled,
      thinking_budget_enabled: rectifierForm.thinking_budget_enabled,
    });
    Object.assign(rectifierForm, updated);
    appStore.showSuccess(t("admin.settings.rectifier.saved"));
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.rectifier.saveFailed") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    rectifierSaving.value = false;
  }
}

const betaPolicyActionOptions = computed(() => [
  { value: "pass", label: t("admin.settings.betaPolicy.actionPass") },
  { value: "filter", label: t("admin.settings.betaPolicy.actionFilter") },
  { value: "block", label: t("admin.settings.betaPolicy.actionBlock") },
]);

const betaPolicyScopeOptions = computed(() => [
  { value: "all", label: t("admin.settings.betaPolicy.scopeAll") },
  { value: "oauth", label: t("admin.settings.betaPolicy.scopeOAuth") },
  { value: "apikey", label: t("admin.settings.betaPolicy.scopeAPIKey") },
  { value: "bedrock", label: t("admin.settings.betaPolicy.scopeBedrock") },
]);

// Beta Policy 方法
const betaDisplayNames: Record<string, string> = {
  "fast-mode-2026-02-01": "Fast Mode",
  "context-1m-2025-08-07": "Context 1M",
};

function getBetaDisplayName(token: string): string {
  return betaDisplayNames[token] || token;
}

async function loadBetaPolicySettings() {
  betaPolicyLoading.value = true;
  try {
    const settings = await adminAPI.settings.getBetaPolicySettings();
    betaPolicyForm.rules = settings.rules;
  } catch (error: any) {
    console.error("Failed to load beta policy settings:", error);
  } finally {
    betaPolicyLoading.value = false;
  }
}

// ==================== OpenAI Fast/Flex Policy ====================

const openaiFastPolicyTierOptions = computed(() => [
  { value: "all", label: t("admin.settings.openaiFastPolicy.tierAll") },
  {
    value: "priority",
    label: t("admin.settings.openaiFastPolicy.tierPriority"),
  },
  { value: "flex", label: t("admin.settings.openaiFastPolicy.tierFlex") },
]);

const openaiFastPolicyActionOptions = computed(() => [
  { value: "pass", label: t("admin.settings.openaiFastPolicy.actionPass") },
  { value: "filter", label: t("admin.settings.openaiFastPolicy.actionFilter") },
  {
    value: "force_priority",
    label: t("admin.settings.openaiFastPolicy.actionForcePriority"),
  },
  { value: "block", label: t("admin.settings.openaiFastPolicy.actionBlock") },
]);

function openaiFastPolicyActionSummary(
  action: OpenAIFastPolicyRule["action"],
) {
  return t(`admin.settings.openaiFastPolicy.summaryAction.${action}`);
}

function hasOpenAIFastPolicyTargetModels(rule: OpenAIFastPolicyRule) {
  return Boolean(rule.model_whitelist?.some((pattern) => pattern.trim() !== ""));
}

const openaiFastPolicyScopeOptions = computed(() => [
  { value: "all", label: t("admin.settings.openaiFastPolicy.scopeAll") },
  { value: "oauth", label: t("admin.settings.openaiFastPolicy.scopeOAuth") },
  { value: "apikey", label: t("admin.settings.openaiFastPolicy.scopeAPIKey") },
  {
    value: "bedrock",
    label: t("admin.settings.openaiFastPolicy.scopeBedrock"),
  },
]);

function addOpenAIFastPolicyRule() {
  openaiFastPolicyForm.rules.push({
    service_tier: "priority",
    action: "filter",
    scope: "all",
    user_ids: [],
    error_message: "",
    model_whitelist: [],
    fallback_action: "pass",
    fallback_error_message: "",
  });
}

function removeOpenAIFastPolicyRule(index: number) {
  openaiFastPolicyForm.rules.splice(index, 1);
}

function addOpenAIFastPolicyModelPattern(rule: OpenAIFastPolicyRule) {
  if (!rule.model_whitelist) rule.model_whitelist = [];
  rule.model_whitelist.push("");
}

function removeOpenAIFastPolicyModelPattern(
  rule: OpenAIFastPolicyRule,
  idx: number,
) {
  rule.model_whitelist?.splice(idx, 1);
}

async function saveBetaPolicySettings() {
  betaPolicySaving.value = true;
  try {
    const updated = await adminAPI.settings.updateBetaPolicySettings({
      rules: betaPolicyForm.rules,
    });
    betaPolicyForm.rules = updated.rules;
    appStore.showSuccess(t("admin.settings.betaPolicy.saved"));
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.betaPolicy.saveFailed") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    betaPolicySaving.value = false;
  }
}

async function loadTLSFingerprintSettings() {
  tlsFingerprintLoading.value = true;
  try {
    const result = await adminAPI.settings.getTLSFingerprintSettings();
    tlsFingerprintGlobalEnabled.value = result.enabled;
    tlsFingerprintProfiles.value = result.items;
  } catch (error: any) {
    console.error("Failed to load TLS fingerprint settings:", error);
  } finally {
    tlsFingerprintLoading.value = false;
  }
}

async function saveTLSFingerprintGlobalSettings() {
  tlsFingerprintSaving.value = true;
  try {
    const updated = await adminAPI.settings.updateTLSFingerprintSettings({
      enabled: tlsFingerprintGlobalEnabled.value,
    });
    tlsFingerprintGlobalEnabled.value = updated.enabled;
    await loadTLSFingerprintSettings();
    appStore.showSuccess(t("admin.settings.tlsFingerprint.saved"));
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.tlsFingerprint.saveFailed") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    tlsFingerprintSaving.value = false;
  }
}

async function submitTLSFingerprintProfile() {
  const profileID = tlsFingerprintForm.profile_id.trim();
  const name = tlsFingerprintForm.name.trim();
  if (!editingTLSFingerprintProfileID.value && !profileID) {
    appStore.showError(t("admin.settings.tlsFingerprint.profileIDRequired"));
    return;
  }
  if (!name) {
    appStore.showError(t("admin.settings.tlsFingerprint.profileNameRequired"));
    return;
  }

  let cipherSuites: number[];
  let curves: number[];
  let pointFormats: number[];
  try {
    cipherSuites = parseTLSNumberList(
      tlsFingerprintForm.cipher_suites_text,
      65535,
      "admin.settings.tlsFingerprint.invalidUint16",
    );
    curves = parseTLSNumberList(
      tlsFingerprintForm.curves_text,
      65535,
      "admin.settings.tlsFingerprint.invalidUint16",
    );
    pointFormats = parseTLSNumberList(
      tlsFingerprintForm.point_formats_text,
      255,
      "admin.settings.tlsFingerprint.invalidUint8",
    );
  } catch (error: any) {
    appStore.showError(error.message || t("common.unknownError"));
    return;
  }

  tlsFingerprintSaving.value = true;
  try {
    const payload = {
      name,
      enabled: tlsFingerprintForm.enabled,
      enable_grease: tlsFingerprintForm.enable_grease,
      cipher_suites: cipherSuites,
      curves,
      point_formats: pointFormats,
    };
    if (editingTLSFingerprintProfileID.value) {
      await adminAPI.settings.updateTLSFingerprintProfile(
        editingTLSFingerprintProfileID.value,
        payload,
      );
      appStore.showSuccess(t("admin.settings.tlsFingerprint.profileSaved"));
    } else {
      await adminAPI.settings.createTLSFingerprintProfile({
        profile_id: profileID,
        ...payload,
      });
      appStore.showSuccess(t("admin.settings.tlsFingerprint.profileCreated"));
    }
    closeTLSFingerprintModal();
    await loadTLSFingerprintSettings();
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.tlsFingerprint.saveFailed") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    tlsFingerprintSaving.value = false;
  }
}

async function toggleTLSFingerprintProfile(profile: TLSFingerprintProfile) {
  tlsFingerprintSaving.value = true;
  try {
    await adminAPI.settings.updateTLSFingerprintProfile(profile.profile_id, {
      name: profile.name,
      enabled: !profile.enabled,
      enable_grease: profile.enable_grease,
      cipher_suites: profile.cipher_suites,
      curves: profile.curves,
      point_formats: profile.point_formats,
    });
    await loadTLSFingerprintSettings();
    appStore.showSuccess(t("admin.settings.tlsFingerprint.profileSaved"));
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.tlsFingerprint.saveFailed") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    tlsFingerprintSaving.value = false;
  }
}

async function deleteTLSFingerprintProfile(profile: TLSFingerprintProfile) {
  if (
    !confirm(
      t("admin.settings.tlsFingerprint.deleteConfirm", {
        profileID: profile.profile_id,
      }),
    )
  ) {
    return;
  }
  tlsFingerprintSaving.value = true;
  try {
    await adminAPI.settings.deleteTLSFingerprintProfile(profile.profile_id);
    await loadTLSFingerprintSettings();
    appStore.showSuccess(t("admin.settings.tlsFingerprint.profileDeleted"));
  } catch (error: any) {
    appStore.showError(
      t("admin.settings.tlsFingerprint.deleteFailed") +
        ": " +
        (error.message || t("common.unknownError")),
    );
  } finally {
    tlsFingerprintSaving.value = false;
  }
}

onMounted(() => {
  loadSettings();
  loadPaymentProviders();
  loadUpstreamBillingProbeSettings();
  loadOllamaCloudUsageSettings();
  loadSubscriptionGroups();
  loadAdminApiKey();
  loadOverloadCooldownSettings();
  loadPanelRateLimitSettings();
  loadStreamTimeoutSettings();
  loadRectifierSettings();
  loadBetaPolicySettings();
  loadTLSFingerprintSettings();
  loadWebSearchConfig();
});
</script>

<style scoped>
.default-sub-group-select :deep(.select-trigger) {
  @apply h-[42px];
}

.default-sub-delete-btn {
  @apply h-[42px];
}

/* ============ Settings Tab Navigation ============ */

/* Scroll container: thin scrollbar on PC, auto-hide on mobile */
.settings-tabs-scroll {
  scrollbar-width: thin;
  scrollbar-color: transparent transparent;
}
.settings-tabs-scroll:hover {
  scrollbar-color: rgb(0 0 0 / 0.15) transparent;
}
:root.dark .settings-tabs-scroll:hover {
  scrollbar-color: rgb(255 255 255 / 0.2) transparent;
}
.settings-tabs-scroll::-webkit-scrollbar {
  height: 3px;
}
.settings-tabs-scroll::-webkit-scrollbar-track {
  background: transparent;
}
.settings-tabs-scroll::-webkit-scrollbar-thumb {
  background: transparent;
  border-radius: 3px;
}
.settings-tabs-scroll:hover::-webkit-scrollbar-thumb {
  background: rgb(0 0 0 / 0.15);
}
:root.dark .settings-tabs-scroll:hover::-webkit-scrollbar-thumb {
  background: rgb(255 255 255 / 0.2);
}

.settings-tabs {
  @apply inline-flex min-w-full gap-0.5 rounded-2xl
         border border-gray-100 bg-white/80 p-1 backdrop-blur-sm
         dark:border-dark-700/50 dark:bg-dark-800/80;
  box-shadow:
    0 1px 3px rgb(0 0 0 / 0.04),
    0 1px 2px rgb(0 0 0 / 0.02);
}

@media (min-width: 640px) {
  .settings-tabs {
    @apply flex;
  }
}

.settings-tab {
  @apply relative flex flex-1 items-center justify-center gap-1.5
         whitespace-nowrap rounded-xl px-2.5 py-2
         text-sm font-medium
         text-gray-500 dark:text-dark-400
         transition-all duration-200 ease-out;
}

.settings-tab:hover:not(.settings-tab-active) {
  @apply text-gray-700 dark:text-gray-300;
  background: rgb(0 0 0 / 0.03);
}

:root.dark .settings-tab:hover:not(.settings-tab-active) {
  background: rgb(255 255 255 / 0.04);
}

.settings-tab-active {
  @apply text-primary-600 dark:text-primary-400;
  background: linear-gradient(
    135deg,
    rgba(20, 184, 166, 0.08),
    rgba(20, 184, 166, 0.03)
  );
  box-shadow: 0 1px 2px rgba(20, 184, 166, 0.1);
}

:root.dark .settings-tab-active {
  background: linear-gradient(
    135deg,
    rgba(45, 212, 191, 0.12),
    rgba(45, 212, 191, 0.05)
  );
  box-shadow: 0 1px 3px rgb(0 0 0 / 0.25);
}

.settings-tab-icon {
  @apply flex h-6 w-6 items-center justify-center rounded-lg
         transition-all duration-200;
}

.settings-tab-active .settings-tab-icon {
  @apply bg-primary-500/15 text-primary-600
         dark:bg-primary-400/15 dark:text-primary-400;
}
</style>

<style scoped>
.admin-b4-settings-scope :deep(.card) {
  background: transparent !important;
  border-color: var(--ssxz-border) !important;
  box-shadow: none !important;
}

.admin-b4-settings-scope :deep(thead),
.admin-b4-settings-scope :deep(.table-header) {
  background: var(--ssxz-surface-raised) !important;
}

.admin-b4-settings-scope :deep(tbody) {
  background: transparent !important;
}
</style>
