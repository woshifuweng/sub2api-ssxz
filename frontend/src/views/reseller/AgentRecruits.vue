<template>
  <ResellerPageLayout
    :title="t('reseller.pages.recruits.title')"
    :description="t('reseller.pages.recruits.description')"
  >
    <section class="card overflow-hidden">
      <header class="recruits-header">
        <div>
          <h2>&#x62DB;&#x52DF;&#x5217;&#x8868;</h2>
          <p>&#x4EC5;&#x5C55;&#x793A;&#x63A8;&#x5E7F;&#x5173;&#x7CFB;&#x6458;&#x8981;&#xFF0C;&#x4E0D;&#x663E;&#x793A;&#x7528;&#x6237;&#x7684;&#x654F;&#x611F;&#x4FE1;&#x606F;&#x3002;</p>
        </div>
        <LiquidButton type="button" variant="outline" size="sm" :disabled="loading" @click="loadPage()">
          <Icon name="refresh" size="sm" />
          <span>&#x5237;&#x65B0;</span>
        </LiquidButton>
      </header>

      <div v-if="loading" class="recruits-empty">&#x6B63;&#x5728;&#x52A0;&#x8F7D;&#x62DB;&#x52DF;&#x6570;&#x636E;...</div>
      <div v-else-if="loadError" class="recruits-empty">{{ loadError }}</div>
      <div v-else class="overflow-x-auto">
        <table class="recruits-table min-w-[720px]">
          <thead>
            <tr>
              <th>&#x7528;&#x6237;</th>
              <th>&#x52A0;&#x5165;&#x65F6;&#x95F4;</th>
              <th>&#x7D2F;&#x8BA1;&#x989D;&#x5EA6;</th>
              <th>&#x72B6;&#x6001;</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="item in recruits.items"
              :key="item.user_id"
              class="recruit-row"
              tabindex="0"
              @click="openRecruit(item)"
              @keydown.enter="openRecruit(item)"
            >
              <td>
                <strong>{{ item.username || '\u672A\u8BBE\u7F6E\u6635\u79F0' }}</strong>
                <small>{{ maskEmail(item.email) }}</small>
              </td>
              <td>{{ formatRelativeTime(item.joined_at) }}</td>
              <td class="font-medium text-gray-900 dark:text-white">
                {{ formatNumber(item.total_rebate) }} &#x989D;&#x5EA6;
              </td>
              <td>
                <span :class="item.is_active ? 'status-active' : 'status-muted'">
                  {{ item.is_active ? '\u6B63\u5E38' : '\u5DF2\u505C\u7528' }}
                </span>
              </td>
            </tr>
            <tr v-if="recruits.items.length === 0">
              <td colspan="4" class="py-10 text-center text-gray-500 dark:text-gray-400">
                &#x6682;&#x65E0;&#x62DB;&#x52DF;&#x7528;&#x6237;
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <Pagination
        v-if="recruits.total > 0"
        :page="recruits.page"
        :page-size="recruits.page_size"
        :total="recruits.total"
        :show-page-size-selector="false"
        @update:page="changePage"
      />
    </section>

    <RecruitDetailDrawer
      :show="drawerOpen"
      :user-id="selectedUserId"
      :recruit="selectedRecruit"
      @close="drawerOpen = false"
    />
  </ResellerPageLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import ResellerPageLayout from '@/components/reseller/ResellerPageLayout.vue'
import RecruitDetailDrawer from '@/components/reseller/RecruitDetailDrawer.vue'
import Icon from '@/components/icons/Icon.vue'
import LiquidButton from '@/components/common/LiquidButton.vue'
import Pagination from '@/components/common/Pagination.vue'
import resellerAPI, { type RecruitRecord } from '@/api/reseller'
import type { PaginatedResponse } from '@/types'
import { formatNumber, formatRelativeTime } from '@/utils/format'
import { extractApiErrorMessage } from '@/utils/apiError'

const loading = ref(true)
const { t } = useI18n()
const loadError = ref('')
const recruits = ref<PaginatedResponse<RecruitRecord>>(emptyPage())
const drawerOpen = ref(false)
const selectedUserId = ref<number | null>(null)
const selectedRecruit = ref<RecruitRecord | null>(null)

function emptyPage<T>(): PaginatedResponse<T> {
  return { items: [], total: 0, page: 1, page_size: 20, pages: 0 }
}

function maskEmail(email: string | undefined): string {
  if (!email) return '--'
  const [local, domain] = email.split('@')
  if (!domain) return '***'
  if (local.length <= 2) return `${local[0] || '*'}***@${domain}`
  return `${local.slice(0, 2)}***@${domain}`
}

async function loadPage(page = recruits.value.page || 1): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    recruits.value = await resellerAPI.listRecruits(page, recruits.value.page_size)
  } catch (error) {
    loadError.value = extractApiErrorMessage(error, '\u62DB\u52DF\u6570\u636E\u52A0\u8F7D\u5931\u8D25')
  } finally {
    loading.value = false
  }
}

function changePage(page: number): void {
  void loadPage(page)
}

function openRecruit(recruit: RecruitRecord): void {
  selectedUserId.value = recruit.user_id
  selectedRecruit.value = recruit
  drawerOpen.value = true
}

onMounted(() => void loadPage(1))
</script>

<style scoped>
.recruits-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
  border-bottom: 1px solid light-dark(#e5e7eb, #374151);
  padding: 1.25rem 1.5rem;
}

.recruits-header h2 {
  margin: 0;
  color: light-dark(#111827, #f9fafb);
  font-size: 1rem;
}

.recruits-header p,
.recruits-empty {
  color: light-dark(#6b7280, #9ca3af);
  font-size: 0.8rem;
}

.recruits-header p {
  margin: 0.35rem 0 0;
}

.recruits-empty {
  padding: 3rem 1.5rem;
  text-align: center;
}

.recruits-table {
  width: 100%;
  border-collapse: collapse;
}

.recruits-table th,
.recruits-table td {
  border-bottom: 1px solid light-dark(#e5e7eb, #374151);
  padding: 0.85rem 1.5rem;
  text-align: left;
  white-space: nowrap;
}

.recruits-table th {
  color: light-dark(#6b7280, #9ca3af);
  font-size: 0.75rem;
  font-weight: 500;
}

.recruits-table td {
  color: light-dark(#4b5563, #d1d5db);
  font-size: 0.85rem;
}

.recruits-table td strong,
.recruits-table td small {
  display: block;
}

.recruit-row {
  cursor: pointer;
}

.recruit-row:hover,
.recruit-row:focus-visible {
  background: light-dark(#ffffff, #1f2937);
  outline: none;
}

.recruits-table td strong {
  color: light-dark(#111827, #f9fafb);
  font-weight: 600;
}

.recruits-table td small {
  margin-top: 0.25rem;
  color: light-dark(#6b7280, #9ca3af);
}

.status-active,
.status-muted {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  padding: 0.2rem 0.55rem;
  font-size: 0.75rem;
}

.status-active {
  background: color-mix(in srgb, #22c55e 12%, transparent);
  color: #22c55e;
}

.status-muted {
  background: color-mix(in srgb, light-dark(#6b7280, #9ca3af) 12%, transparent);
  color: light-dark(#6b7280, #9ca3af);
}
</style>
