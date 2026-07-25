<template>
  <AppLayout>
    <AdminPageHeader title="用户管理" description="查看全站用户账号与余额" />

    <TablePageLayout class="admin-b2-outline-scope">
      <!-- Single Row: Search, Filters, and Actions -->
      <template #filters>
        <div class="flex flex-wrap items-center gap-3">
          <!-- Left: Search + Active Filters -->
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <!-- Search Box -->
            <div class="relative w-full md:w-64">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.users.searchUsers')"
                class="input pl-10"
                @input="handleSearch"
              />
            </div>

            <!-- Role Filter (visible when enabled) -->
            <div v-if="visibleFilters.has('role')" class="w-full sm:w-32">
              <Select
                v-model="filters.role"
                :options="[
                  { value: '', label: t('admin.users.allRoles') },
                  { value: 'admin', label: t('admin.users.admin') },
                  { value: 'user', label: t('admin.users.user') }
                ]"
                @change="applyFilter"
              />
            </div>

            <!-- Status Filter (visible when enabled) -->
            <div v-if="visibleFilters.has('status')" class="w-full sm:w-32">
              <Select
                v-model="filters.status"
                :options="[
                  { value: '', label: t('admin.users.allStatus') },
                  { value: 'active', label: t('common.active') },
                  { value: 'disabled', label: t('admin.users.disabled') }
                ]"
                @change="applyFilter"
              />
            </div>

            <!-- Group Filter (visible when enabled) -->
            <div v-if="visibleFilters.has('group')" class="w-full sm:w-44">
              <Select
                v-model="filters.group"
                :options="groupFilterOptions"
                searchable
                creatable
                :creatable-prefix="t('admin.users.fuzzySearch')"
                :search-placeholder="t('admin.users.searchGroups')"
                @change="applyFilter"
              />
            </div>

            <!-- Dynamic Attribute Filters -->
            <template v-for="(value, attrId) in activeAttributeFilters" :key="attrId">
              <div
                v-if="visibleFilters.has(`attr_${attrId}`)"
                class="relative w-full sm:w-36"
              >
                <!-- Text/Email/URL/Textarea/Date type: styled input -->
                <input
                  v-if="['text', 'textarea', 'email', 'url', 'date'].includes(getAttributeDefinition(Number(attrId))?.type || 'text')"
                  :value="value"
                  @input="(e) => updateAttributeFilter(Number(attrId), (e.target as HTMLInputElement).value)"
                  @keyup.enter="applyFilter"
                  :placeholder="getAttributeDefinitionName(Number(attrId))"
                  class="input w-full"
                />
                <!-- Number type: number input -->
                <input
                  v-else-if="getAttributeDefinition(Number(attrId))?.type === 'number'"
                  :value="value"
                  type="number"
                  @input="(e) => updateAttributeFilter(Number(attrId), (e.target as HTMLInputElement).value)"
                  @keyup.enter="applyFilter"
                  :placeholder="getAttributeDefinitionName(Number(attrId))"
                  class="input w-full"
                />
                <!-- Select/Multi-select type -->
                <template v-else-if="['select', 'multi_select'].includes(getAttributeDefinition(Number(attrId))?.type || '')">
                  <div class="w-full">
                    <Select
                      :model-value="value"
                      :options="[
                        { value: '', label: getAttributeDefinitionName(Number(attrId)) },
                        ...(getAttributeDefinition(Number(attrId))?.options || [])
                      ]"
                      @update:model-value="(val) => { updateAttributeFilter(Number(attrId), String(val ?? '')); applyFilter() }"
                    />
                  </div>
                </template>
                <!-- Fallback -->
                <input
                  v-else
                  :value="value"
                  @input="(e) => updateAttributeFilter(Number(attrId), (e.target as HTMLInputElement).value)"
                  @keyup.enter="applyFilter"
                  :placeholder="getAttributeDefinitionName(Number(attrId))"
                  class="input w-full"
                />
              </div>
            </template>
          </div>

          <!-- Right: Actions and Settings -->
          <div class="flex flex-wrap items-center justify-end gap-2">
            <!-- Mobile: Secondary buttons (icon only) -->
            <div class="flex items-center gap-2 md:contents">
              <!-- Refresh Button -->
              <button
                @click="loadUsers"
                :disabled="loading"
                class="btn btn-secondary px-2 md:px-3"
                :title="t('common.refresh')"
              >
                <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              </button>
              <!-- Filter Settings Dropdown -->
              <div class="relative" ref="filterDropdownRef">
                <button
                  @click="showFilterDropdown = !showFilterDropdown"
                  class="btn btn-secondary px-2 md:px-3"
                  :title="t('admin.users.filterSettings')"
                >
                  <Icon name="filter" size="sm" class="md:mr-1.5" />
                  <span class="hidden md:inline">{{ t('admin.users.filterSettings') }}</span>
                </button>
                <!-- Dropdown menu -->
                <div
                  v-if="showFilterDropdown"
                  class="absolute right-0 top-full z-50 mt-1 w-48 rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
                >
                  <!-- Built-in filters -->
                  <button
                    v-for="filter in builtInFilters"
                    :key="filter.key"
                    @click="toggleBuiltInFilter(filter.key)"
                    class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
                  >
                    <span>{{ filter.name }}</span>
                    <Icon
                      v-if="visibleFilters.has(filter.key)"
                      name="check"
                      size="sm"
                      class="text-primary-500"
                      :stroke-width="2"
                    />
                  </button>
                  <!-- Divider if custom attributes exist -->
                  <div
                    v-if="filterableAttributes.length > 0"
                    class="my-1 border-t border-gray-100 dark:border-dark-700"
                  ></div>
                  <!-- Custom attribute filters -->
                  <button
                    v-for="attr in filterableAttributes"
                    :key="attr.id"
                    @click="toggleAttributeFilter(attr)"
                    class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
                  >
                    <span>{{ attr.name }}</span>
                    <Icon
                      v-if="visibleFilters.has(`attr_${attr.id}`)"
                      name="check"
                      size="sm"
                      class="text-primary-500"
                      :stroke-width="2"
                    />
                  </button>
                </div>
              </div>
              <!-- Column Settings Dropdown -->
              <div class="relative" ref="columnDropdownRef">
                <button
                  @click="showColumnDropdown = !showColumnDropdown"
                  class="btn btn-secondary px-2 md:px-3"
                  :title="t('admin.users.columnSettings')"
                >
                  <svg class="h-4 w-4 md:mr-1.5" fill="none" stroke="currentColor" viewBox="0 0 24 24" stroke-width="1.5">
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 4.5v15m6-15v15m-10.875 0h15.75c.621 0 1.125-.504 1.125-1.125V5.625c0-.621-.504-1.125-1.125-1.125H4.125C3.504 4.5 3 5.004 3 5.625v12.75c0 .621.504 1.125 1.125 1.125z" />
                  </svg>
                  <span class="hidden md:inline">{{ t('admin.users.columnSettings') }}</span>
                </button>
                <!-- Dropdown menu -->
                <div
                  v-if="showColumnDropdown"
                  class="absolute right-0 top-full z-50 mt-1 max-h-80 w-48 overflow-y-auto rounded-lg border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-600 dark:bg-dark-800"
                >
                  <button
                    v-for="col in toggleableColumns"
                    :key="col.key"
                    @click="toggleColumn(col.key)"
                    class="flex w-full items-center justify-between px-4 py-2 text-left text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
                  >
                    <span>{{ col.label }}</span>
                    <Icon
                      v-if="isColumnVisible(col.key)"
                      name="check"
                      size="sm"
                      class="text-primary-500"
                      :stroke-width="2"
                    />
                  </button>
                </div>
              </div>
              <!-- Attributes Config Button -->
              <button
                @click="showAttributesModal = true"
                class="btn btn-secondary px-2 md:px-3"
                :title="t('admin.users.attributes.configButton')"
              >
                <Icon name="cog" size="sm" class="md:mr-1.5" />
                <span class="hidden md:inline">{{ t('admin.users.attributes.configButton') }}</span>
              </button>
            </div>

            <!-- Create User Button (full width on mobile, auto width on desktop) -->
            <button @click="showCreateModal = true" class="btn btn-primary flex-1 md:flex-initial">
              <Icon name="plus" size="md" class="mr-2" />
              {{ t('admin.users.createUser') }}
            </button>
          </div>
        </div>
      </template>

      <!-- Users Table -->
      <template #table>
        <DataTable :columns="columns" :data="users" :loading="loading" :actions-count="7">
          <template #cell-email="{ value }">
            <div class="flex items-center gap-2">
              <div
                class="flex h-8 w-8 items-center justify-center rounded-full bg-primary-100 dark:bg-primary-900/30"
              >
                <span class="text-sm font-medium text-primary-700 dark:text-primary-300">
                  {{ value.charAt(0).toUpperCase() }}
                </span>
              </div>
              <span class="font-medium text-gray-900 dark:text-white">{{ value }}</span>
            </div>
          </template>

          <template #cell-username="{ value }">
            <span class="text-sm text-gray-700 dark:text-gray-300">{{ value || '-' }}</span>
          </template>

          <template #cell-notes="{ value }">
            <div class="max-w-xs">
              <span
                v-if="value"
                :title="value.length > 30 ? value : undefined"
                class="block truncate text-sm text-gray-600 dark:text-gray-400"
              >
                {{ value.length > 30 ? value.substring(0, 25) + '...' : value }}
              </span>
              <span v-else class="text-sm text-gray-400">-</span>
            </div>
          </template>

          <!-- Dynamic attribute columns -->
          <template
            v-for="def in attributeDefinitions.filter(d => d.enabled)"
            :key="def.id"
            #[`cell-attr_${def.id}`]="{ row }"
          >
            <div class="max-w-xs">
              <span
                class="block truncate text-sm text-gray-700 dark:text-gray-300"
                :title="getAttributeValue(row.id, def.id)"
              >
                {{ getAttributeValue(row.id, def.id) }}
              </span>
            </div>
          </template>

          <template #cell-role="{ value }">
            <span :class="['badge', value === 'admin' ? 'badge-purple' : 'badge-gray']">
              {{ t('admin.users.roles.' + value) }}
            </span>
          </template>

          <template #cell-groups="{ row }">
            <div v-if="allGroups.length > 0" class="flex flex-col gap-1">
              <!-- 专属分组行 -->
              <span
                v-if="getUserGroups(row).exclusive.length > 0"
                class="group/ex relative inline-flex cursor-pointer items-center gap-1 whitespace-nowrap text-xs"
                @click.stop="toggleExpandedGroup(row.id)"
              >
                <Icon name="shield" size="xs" class="h-3.5 w-3.5 text-purple-500 dark:text-purple-400" />
                <span class="font-medium text-purple-600 dark:text-purple-400">{{ getUserGroups(row).exclusive.length }}</span>
                <span class="text-gray-500 dark:text-dark-400">{{ t('admin.users.exclusiveLabel') }}</span>
                <!-- Hover tooltip（操作菜单未打开时显示） -->
                <div
                  v-if="expandedGroupUserId !== row.id"
                  class="pointer-events-none absolute left-0 top-full z-50 mt-1.5 rounded bg-gray-900 px-2.5 py-1.5 text-xs text-white opacity-0 shadow-lg transition-opacity duration-75 group-hover/ex:opacity-100 dark:bg-dark-600"
                >
                  <div class="absolute left-4 bottom-full border-4 border-transparent border-b-gray-900 dark:border-b-dark-600"></div>
                  <div class="flex flex-col gap-0.5 whitespace-nowrap">
                    <span v-for="g in getUserGroups(row).exclusive" :key="g.id">{{ g.name }}</span>
                  </div>
                </div>
                <!-- 点击展开分组操作菜单 -->
                <div
                  v-if="expandedGroupUserId === row.id"
                  class="absolute left-0 top-full z-50 mt-1.5 min-w-[160px] overflow-hidden rounded-lg border border-gray-200 bg-white py-1 text-xs shadow-xl dark:border-dark-600 dark:bg-dark-700"
                >
                  <div class="border-b border-gray-100 px-3 py-1.5 text-[10px] font-medium uppercase tracking-wider text-gray-400 dark:border-dark-600 dark:text-dark-400">
                    {{ t('admin.users.clickToReplace') }}
                  </div>
                  <div
                    v-for="g in getUserGroups(row).exclusive"
                    :key="g.id"
                    class="flex cursor-pointer items-center gap-2 px-3 py-2 text-gray-700 transition-colors hover:bg-primary-50 hover:text-primary-600 dark:text-dark-200 dark:hover:bg-primary-900/30 dark:hover:text-primary-400"
                    @click.stop="openGroupReplace(row, g)"
                  >
                    <Icon name="swap" size="xs" class="h-3.5 w-3.5 flex-shrink-0 opacity-50" />
                    <span class="flex-1">{{ g.name }}</span>
                  </div>
                </div>
              </span>
              <!-- 公开分组行 -->
              <span
                v-if="getUserGroups(row).publicGroups.length > 0"
                class="group/pub relative inline-flex cursor-default items-center gap-1 whitespace-nowrap text-xs"
              >
                <Icon name="globe" size="xs" class="h-3.5 w-3.5 text-gray-400 dark:text-dark-500" />
                <span class="font-medium text-gray-600 dark:text-dark-300">{{ getUserGroups(row).publicGroups.length }}</span>
                <span class="text-gray-400 dark:text-dark-500">{{ t('admin.users.publicLabel') }}</span>
                <!-- Tooltip: 向下弹出 -->
                <div class="pointer-events-none absolute left-0 top-full z-50 mt-1.5 rounded bg-gray-900 px-2.5 py-1.5 text-xs text-white opacity-0 shadow-lg transition-opacity duration-75 group-hover/pub:opacity-100 dark:bg-dark-600">
                  <div class="absolute left-4 bottom-full border-4 border-transparent border-b-gray-900 dark:border-b-dark-600"></div>
                  <div class="flex flex-col gap-0.5 whitespace-nowrap">
                    <span v-for="g in getUserGroups(row).publicGroups" :key="g.id">{{ g.name }}</span>
                  </div>
                </div>
              </span>
              <!-- 都没有 -->
              <span
                v-if="getUserGroups(row).exclusive.length === 0 && getUserGroups(row).publicGroups.length === 0"
                class="text-xs text-gray-400 dark:text-dark-500"
              >-</span>
            </div>
            <span v-else class="text-xs text-gray-400 dark:text-dark-500">-</span>
          </template>

          <template #cell-subscriptions="{ row }">
            <div
              v-if="row.subscriptions && row.subscriptions.length > 0"
              class="flex flex-wrap gap-1.5"
            >
              <GroupBadge
                v-for="sub in row.subscriptions"
                :key="sub.id"
                :name="sub.group?.name || ''"
                :platform="sub.group?.platform"
                :subscription-type="sub.group?.subscription_type"
                :rate-multiplier="sub.group?.rate_multiplier"
                :days-remaining="sub.expires_at ? getDaysRemaining(sub.expires_at) : null"
                :title="sub.expires_at ? formatDateTime(sub.expires_at) : ''"
              />
            </div>
            <span
              v-else
              class="inline-flex items-center gap-1.5 rounded-md bg-gray-50 px-2 py-1 text-xs text-gray-400 dark:bg-dark-700/50 dark:text-dark-500"
            >
              <Icon name="ban" size="xs" class="h-3.5 w-3.5" />
              <span>{{ t('admin.users.noSubscription') }}</span>
            </span>
          </template>

          <template #cell-balance="{ value, row }">
            <div class="flex items-center gap-2">
              <div class="group relative">
                <button
                  class="font-medium text-gray-900 underline decoration-dashed decoration-gray-300 underline-offset-4 transition-colors hover:text-primary-600 dark:text-white dark:decoration-dark-500 dark:hover:text-primary-400"
                  @click="handleBalanceHistory(row)"
                >
                  ${{ value.toFixed(2) }}
                </button>
                <!-- Instant tooltip -->
                <div class="pointer-events-none absolute bottom-full left-1/2 z-50 mb-1.5 -translate-x-1/2 whitespace-nowrap rounded bg-gray-900 px-2 py-1 text-xs text-white opacity-0 shadow-lg transition-opacity duration-75 group-hover:opacity-100 dark:bg-dark-600">
                  {{ t('admin.users.balanceHistoryTip') }}
                  <div class="absolute left-1/2 top-full -translate-x-1/2 border-4 border-transparent border-t-gray-900 dark:border-t-dark-600"></div>
                </div>
              </div>
              <button
                @click.stop="handleDeposit(row)"
                class="rounded px-2 py-0.5 text-xs font-medium text-emerald-600 transition-colors hover:bg-emerald-50 dark:text-emerald-400 dark:hover:bg-emerald-900/20"
                :title="t('admin.users.deposit')"
              >
                {{ t('admin.users.deposit') }}
              </button>
            </div>
          </template>

          <template #cell-usage="{ row }">
            <div class="text-sm">
              <div class="flex items-center gap-1.5">
                <span class="text-gray-500 dark:text-gray-400">{{ t('admin.users.today') }}:</span>
                <span
                  class="font-medium text-gray-900 dark:text-white"
                  :title="formatCurrencyTitle(usageStats[row.id]?.today_actual_cost ?? 0)"
                >
                  {{ formatCurrency(usageStats[row.id]?.today_actual_cost ?? 0) }}
                </span>
              </div>
              <div class="mt-0.5 flex items-center gap-1.5">
                <span class="text-gray-500 dark:text-gray-400">{{ t('admin.users.total') }}:</span>
                <span
                  class="font-medium text-gray-900 dark:text-white"
                  :title="formatCurrencyTitle(usageStats[row.id]?.total_actual_cost ?? 0)"
                >
                  {{ formatCurrency(usageStats[row.id]?.total_actual_cost ?? 0) }}
                </span>
              </div>
            </div>
          </template>

          <template #cell-concurrency="{ row }">
            <UserConcurrencyCell
              :current="row.current_concurrency ?? 0"
              :max="row.concurrency"
              :unlimited="row.unlimited_concurrency === true"
            />
          </template>

          <template #cell-status="{ value }">
            <div class="flex items-center gap-1.5">
              <span
                :class="[
                  'inline-block h-2 w-2 rounded-full',
                  value === 'active' ? 'bg-green-500' : 'bg-red-500'
                ]"
              ></span>
              <span class="text-sm text-gray-700 dark:text-gray-300">
                {{ value === 'active' ? t('common.active') : t('admin.users.disabled') }}
              </span>
            </div>
          </template>

          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <!-- Edit Button -->
              <button
                @click="handleEdit(row)"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
              >
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>

              <!-- Toggle Status Button (not for admin) -->
              <button
                v-if="row.role !== 'admin'"
                @click="handleToggleStatus(row)"
                :class="[
                  'flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors',
                  row.status === 'active'
                    ? 'hover:bg-orange-50 hover:text-orange-600 dark:hover:bg-orange-900/20 dark:hover:text-orange-400'
                    : 'hover:bg-green-50 hover:text-green-600 dark:hover:bg-green-900/20 dark:hover:text-green-400'
                ]"
              >
                <Icon v-if="row.status === 'active'" name="ban" size="sm" />
                <Icon v-else name="checkCircle" size="sm" />
                <span class="text-xs">{{ row.status === 'active' ? t('admin.users.disable') : t('admin.users.enable') }}</span>
              </button>

              <!-- More Actions Menu Trigger -->
              <button
                @click="openActionMenu(row, $event)"
                class="action-menu-trigger flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-gray-900 dark:hover:bg-dark-700 dark:hover:text-white"
                :class="{ 'bg-gray-100 text-gray-900 dark:bg-dark-700 dark:text-white': activeMenuId === row.id }"
              >
                <Icon name="more" size="sm" />
                <span class="text-xs">{{ t('common.more') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="t('admin.users.noUsersYet')"
              :description="t('admin.users.createFirstUser')"
              :action-text="t('admin.users.createUser')"
              @action="showCreateModal = true"
            />
          </template>
        </DataTable>
      </template>

      <!-- Pagination -->
      <template #pagination>
      <Pagination
        v-if="pagination.total > 0"
        :page="pagination.page"
        :total="pagination.total"
        :page-size="pagination.page_size"
        @update:page="handlePageChange"
        @update:pageSize="handlePageSizeChange"
      />
      </template>
    </TablePageLayout>

    <!-- Action Menu (Teleported) -->
    <Teleport to="body">
      <div
        v-if="activeMenuId !== null && menuPosition"
        class="action-menu-content fixed z-[9999] w-48 overflow-hidden rounded-xl bg-white shadow-lg ring-1 ring-black/5 dark:bg-dark-800 dark:ring-white/10"
        :style="{ top: menuPosition.top + 'px', left: menuPosition.left + 'px' }"
      >
        <div class="py-1">
          <template v-for="user in users" :key="user.id">
            <template v-if="user.id === activeMenuId">
              <!-- View API Keys -->
              <button
                @click="handleViewApiKeys(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <Icon name="key" size="sm" class="text-gray-400" :stroke-width="2" />
                {{ t('admin.users.apiKeys') }}
              </button>

              <!-- Customer Handoff Checklist -->
              <button
                data-testid="customer-handoff-open"
                @click="handleCustomerHandoff(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <Icon name="clipboard" size="sm" class="text-gray-400" :stroke-width="2" />
                客户交付核对
              </button>

              <!-- View Usage -->
              <button
                @click="handleViewUsage(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <Icon name="eye" size="sm" class="text-gray-400" :stroke-width="2" />
                {{ t('common.view') }} {{ t('admin.usage.title') }}
              </button>

              <!-- View Orders -->
              <button
                @click="handleViewOrders(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <Icon name="eye" size="sm" class="text-gray-400" :stroke-width="2" />
                {{ t('payment.result.viewOrders') }}
              </button>

              <!-- View Affiliate -->
              <button
                @click="handleViewAffiliate(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <Icon name="link" size="sm" class="text-gray-400" :stroke-width="2" />
                推广返利
              </button>

              <!-- View Redeem Codes -->
              <button
                @click="handleViewRedeemCodes(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <Icon name="gift" size="sm" class="text-gray-400" :stroke-width="2" />
                兑换记录
              </button>

              <!-- Allowed Groups -->
              <button
                @click="handleAllowedGroups(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <Icon name="users" size="sm" class="text-gray-400" :stroke-width="2" />
                {{ t('admin.users.groups') }}
              </button>

              <div class="my-1 border-t border-gray-100 dark:border-dark-700"></div>

              <!-- Deposit -->
              <button
                @click="handleDeposit(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <Icon name="plus" size="sm" class="text-emerald-500" :stroke-width="2" />
                {{ t('admin.users.deposit') }}
              </button>

              <!-- Withdraw -->
              <button
                @click="handleWithdraw(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <svg class="h-4 w-4 text-amber-500" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                  <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M20 12H4" />
                </svg>
                {{ t('admin.users.withdraw') }}
              </button>

              <!-- Balance History -->
              <button
                @click="handleBalanceHistory(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100 dark:text-gray-300 dark:hover:bg-dark-700"
              >
                <Icon name="dollar" size="sm" class="text-gray-400" :stroke-width="2" />
                {{ t('admin.users.balanceHistory') }}
              </button>

              <div class="my-1 border-t border-gray-100 dark:border-dark-700"></div>

              <!-- Delete (not for admin) -->
              <button
                v-if="user.role !== 'admin'"
                @click="handleDelete(user); closeActionMenu()"
                class="flex w-full items-center gap-2 px-4 py-2 text-sm text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
              >
                <Icon name="trash" size="sm" :stroke-width="2" />
                {{ t('common.delete') }}
              </button>
            </template>
          </template>
        </div>
      </div>
    </Teleport>

    <ConfirmDialog :show="showDeleteDialog" :title="t('admin.users.deleteUser')" :message="t('admin.users.deleteConfirm', { email: deletingUser?.email })" :danger="true" @confirm="confirmDelete" @cancel="showDeleteDialog = false" />
    <BaseDialog
      :show="showCustomerHandoffModal"
      title="客户交付核对"
      width="wide"
      @close="closeCustomerHandoff"
    >
      <div
        v-if="customerHandoffUser"
        data-testid="customer-handoff-checklist"
        class="space-y-5 text-sm text-gray-700 dark:text-gray-300"
      >
        <section class="rounded-xl border border-gray-200 bg-gray-50 p-4 dark:border-dark-700 dark:bg-dark-800/70">
          <div class="flex flex-wrap items-start justify-between gap-4">
            <div>
              <p class="text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-gray-400">客户账号</p>
              <h3 class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">
                {{ customerHandoffUser.email || customerHandoffUser.username || customerHandoffUser.id }}
              </h3>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                ID {{ customerHandoffUser.id }} · {{ customerHandoffUser.status === 'active' ? '启用' : '禁用' }}
              </p>
            </div>
            <div class="grid min-w-[260px] grid-cols-2 gap-2 text-xs">
              <div class="rounded-lg bg-white p-3 dark:bg-dark-900">
                <span class="text-gray-500 dark:text-gray-400">余额</span>
                <strong class="mt-1 block text-base text-gray-900 dark:text-white">
                  ${{ customerHandoffUser.balance.toFixed(2) }}
                </strong>
              </div>
              <div class="rounded-lg bg-white p-3 dark:bg-dark-900">
                <span class="text-gray-500 dark:text-gray-400">近期用量</span>
                <strong class="mt-1 block text-base text-gray-900 dark:text-white">
                  {{ formatCustomerHandoffUsage(customerHandoffUser) }}
                </strong>
              </div>
            </div>
          </div>
          <p class="mt-3 text-xs leading-5 text-gray-500 dark:text-gray-400">
            这个面板只给运营核对使用。给客户前先看 Key、余额、分组、通道和最近用量；客户侧只需要拿到可用 Key 和简单接入方式。
          </p>
        </section>

        <section
          data-testid="customer-handoff-key-readiness"
          class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900"
        >
          <div class="flex flex-wrap items-start justify-between gap-3">
            <div>
              <p class="font-semibold text-gray-900 dark:text-white">API Key 可用性</p>
              <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                只给运营判断交付状态；客户侧只需要拿到可用 Key 和接入方式。
              </p>
            </div>
            <div v-if="customerHandoffKeyReadiness.loading" class="text-xs text-gray-500 dark:text-gray-400">
              加载中
            </div>
            <div v-else class="flex flex-wrap gap-2 text-xs">
              <span class="rounded-full bg-gray-100 px-2 py-1 text-gray-700 dark:bg-dark-800 dark:text-dark-100">
                总数 {{ customerHandoffKeyReadiness.total }}
              </span>
              <span class="rounded-full bg-green-100 px-2 py-1 text-green-700 dark:bg-green-900/30 dark:text-green-300">
                可交付 {{ customerHandoffKeyReadiness.ready }}
              </span>
              <span class="rounded-full bg-red-100 px-2 py-1 text-red-700 dark:bg-red-900/30 dark:text-red-300">
                需处理 {{ customerHandoffKeyReadiness.blocked }}
              </span>
              <span class="rounded-full bg-yellow-100 px-2 py-1 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-300">
                需留意 {{ customerHandoffKeyReadiness.warning }}
              </span>
            </div>
          </div>
          <p v-if="!customerHandoffKeyReadiness.loading && customerHandoffKeyReadiness.total === 0" class="mt-3 text-sm text-red-600 dark:text-red-300">
            暂无 API Key，先创建低额度测试 Key。
          </p>
          <ul v-else-if="customerHandoffKeyReadiness.notes.length > 0" class="mt-3 space-y-1 text-xs text-gray-600 dark:text-gray-300">
            <li v-for="note in customerHandoffKeyReadiness.notes" :key="note">• {{ note }}</li>
          </ul>
        </section>

        <section class="grid gap-3 lg:grid-cols-2">
          <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
            <p class="font-semibold text-gray-900 dark:text-white">交付前按这个顺序看</p>
            <ol class="mt-3 space-y-2 text-sm leading-6">
              <li><strong>1.</strong> 看账号是否启用，余额是否足够本次测试。</li>
              <li><strong>2.</strong> 看 API Key 是否启用、是否有额度或时间限制。</li>
              <li><strong>3.</strong> 看分组/套餐是否覆盖客户要测的模型。</li>
              <li><strong>4.</strong> 客户发起测试后，看用量定位码和通道状态。</li>
            </ol>
          </div>
          <div class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
            <p class="font-semibold text-gray-900 dark:text-white">错误快速归类</p>
            <ul class="mt-3 space-y-2 text-sm leading-6">
              <li><strong>401</strong>：优先查 Key 是否完整、启用、填错位置。</li>
              <li><strong>403</strong>：优先查余额、Key 额度、分组和模型权限。</li>
              <li><strong>503</strong>：优先查当前线路、上游账号或模型临时状态。</li>
              <li><strong>慢</strong>：优先换低推理/轻量模型；深度检索和高推理本来会慢。</li>
            </ul>
          </div>
        </section>

        <section class="rounded-xl border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-900">
          <p class="font-semibold text-gray-900 dark:text-white">常用排查入口</p>
          <div class="mt-3 grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
            <button data-testid="customer-handoff-api-keys" type="button" class="btn btn-secondary justify-center" @click="openCustomerHandoffApiKeys">
              API Key
            </button>
            <button data-testid="customer-handoff-usage" type="button" class="btn btn-secondary justify-center" @click="openCustomerHandoffUsage">
              用量记录
            </button>
            <button type="button" class="btn btn-secondary justify-center" @click="openCustomerHandoffOrders">
              订单
            </button>
            <button type="button" class="btn btn-secondary justify-center" @click="openCustomerHandoffRedeem">
              兑换码
            </button>
            <button type="button" class="btn btn-secondary justify-center" @click="openCustomerHandoffAffiliate">
              推广记录
            </button>
            <button type="button" class="btn btn-secondary justify-center" @click="openCustomerHandoffGroups">
              分组权限
            </button>
            <button type="button" class="btn btn-secondary justify-center" @click="openCustomerHandoffBalanceHistory">
              余额历史
            </button>
            <button data-testid="customer-handoff-channel-status" type="button" class="btn btn-secondary justify-center" @click="openCustomerHandoffChannelStatus">
              通道监控
            </button>
            <button data-testid="customer-handoff-request-details" type="button" class="btn btn-secondary justify-center" @click="openCustomerHandoffRequestDetails">
              最近请求排查
            </button>
          </div>
        </section>
      </div>
    </BaseDialog>
    <UserCreateModal :show="showCreateModal" @close="showCreateModal = false" @success="loadUsers" />
    <UserEditModal :show="showEditModal" :user="editingUser" @close="closeEditModal" @success="loadUsers" />
    <UserApiKeysModal :show="showApiKeysModal" :user="viewingUser" @close="closeApiKeysModal" />
    <UserAllowedGroupsModal :show="showAllowedGroupsModal" :user="allowedGroupsUser" @close="closeAllowedGroupsModal" @success="loadUsers" />
    <UserBalanceModal :show="showBalanceModal" :user="balanceUser" :operation="balanceOperation" @close="closeBalanceModal" @success="loadUsers" />
    <UserBalanceHistoryModal :show="showBalanceHistoryModal" :user="balanceHistoryUser" @close="closeBalanceHistoryModal" @deposit="handleDepositFromHistory" @withdraw="handleWithdrawFromHistory" />
    <GroupReplaceModal :show="showGroupReplaceModal" :user="groupReplaceUser" :old-group="groupReplaceOldGroup" :all-groups="allGroups" @close="closeGroupReplaceModal" @success="loadUsers" />
    <UserAttributesConfigModal :show="showAttributesModal" @close="handleAttributesModalClose" />
  </AppLayout>
</template>

<script setup lang="ts">
import { ref, reactive, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores/app'
import { getPersistedPageSize } from '@/composables/usePersistedPageSize'
import { formatCurrency, formatCurrencyTitle, formatDateTime } from '@/utils/format'
import Icon from '@/components/icons/Icon.vue'

const { t } = useI18n()
const router = useRouter()
import { adminAPI } from '@/api/admin'
import type { AdminUser, AdminGroup, ApiKey, UserAttributeDefinition } from '@/types'
import type { BatchUserUsageStats } from '@/api/admin/dashboard'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import AdminPageHeader from '@/components/admin/AdminPageHeader.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import Select from '@/components/common/Select.vue'
import UserAttributesConfigModal from '@/components/user/UserAttributesConfigModal.vue'
import UserConcurrencyCell from '@/components/user/UserConcurrencyCell.vue'
import UserCreateModal from '@/components/admin/user/UserCreateModal.vue'
import UserEditModal from '@/components/admin/user/UserEditModal.vue'
import UserApiKeysModal from '@/components/admin/user/UserApiKeysModal.vue'
import UserAllowedGroupsModal from '@/components/admin/user/UserAllowedGroupsModal.vue'
import UserBalanceModal from '@/components/admin/user/UserBalanceModal.vue'
import UserBalanceHistoryModal from '@/components/admin/user/UserBalanceHistoryModal.vue'
import GroupReplaceModal from '@/components/admin/user/GroupReplaceModal.vue'

const appStore = useAppStore()

// Generate dynamic attribute columns from enabled definitions
const attributeColumns = computed<Column[]>(() =>
  attributeDefinitions.value
    .filter(def => def.enabled)
    .map(def => ({
      key: `attr_${def.id}`,
      label: def.name,
      sortable: false
    }))
)

// Get formatted attribute value for display in table
const getAttributeValue = (userId: number, attrId: number): string => {
  const userAttrs = userAttributeValues.value[userId]
  if (!userAttrs) return '-'
  const value = userAttrs[attrId]
  if (!value) return '-'

  // Find definition for this attribute
  const def = attributeDefinitions.value.find(d => d.id === attrId)
  if (!def) return value

  // Format based on type
  if (def.type === 'multi_select' && value) {
    try {
      const arr = JSON.parse(value)
      if (Array.isArray(arr)) {
        // Map values to labels
        return arr.map(v => {
          const opt = def.options?.find(o => o.value === v)
          return opt?.label || v
        }).join(', ')
      }
    } catch {
      return value
    }
  }

  if (def.type === 'select' && value && def.options) {
    const opt = def.options.find(o => o.value === value)
    return opt?.label || value
  }

  return value
}

// All possible columns (for column settings)
const allColumns = computed<Column[]>(() => [
  { key: 'email', label: t('admin.users.columns.user'), sortable: true },
  { key: 'id', label: 'ID', sortable: true },
  { key: 'username', label: t('admin.users.columns.username'), sortable: true },
  { key: 'notes', label: t('admin.users.columns.notes'), sortable: false },
  // Dynamic attribute columns
  ...attributeColumns.value,
  { key: 'role', label: t('admin.users.columns.role'), sortable: true },
  { key: 'groups', label: t('admin.users.columns.groups'), sortable: false },
  { key: 'subscriptions', label: t('admin.users.columns.subscriptions'), sortable: false },
  { key: 'balance', label: t('admin.users.columns.balance'), sortable: true },
  { key: 'usage', label: t('admin.users.columns.usage'), sortable: false },
  { key: 'concurrency', label: t('admin.users.columns.concurrency'), sortable: true },
  { key: 'status', label: t('admin.users.columns.status'), sortable: true },
  { key: 'created_at', label: t('admin.users.columns.created'), sortable: true },
  { key: 'actions', label: t('admin.users.columns.actions'), sortable: false }
])

// Columns that can be toggled (exclude email and actions which are always visible)
const toggleableColumns = computed(() =>
  allColumns.value.filter(col => col.key !== 'email' && col.key !== 'actions')
)

// Hidden columns (stored in Set - columns NOT in this set are visible)
// This way, new columns are visible by default
const hiddenColumns = reactive<Set<string>>(new Set())

// Default hidden columns (columns hidden by default on first load)
const DEFAULT_HIDDEN_COLUMNS = ['notes', 'groups', 'subscriptions', 'usage', 'concurrency']

// localStorage key for column settings
const HIDDEN_COLUMNS_KEY = 'user-hidden-columns'

// Load saved column settings
const loadSavedColumns = () => {
  try {
    const saved = localStorage.getItem(HIDDEN_COLUMNS_KEY)
    if (saved) {
      const parsed = JSON.parse(saved) as string[]
      parsed.forEach(key => hiddenColumns.add(key))
    } else {
      // Use default hidden columns on first load
      DEFAULT_HIDDEN_COLUMNS.forEach(key => hiddenColumns.add(key))
    }
  } catch (e) {
    console.error('Failed to load saved columns:', e)
    DEFAULT_HIDDEN_COLUMNS.forEach(key => hiddenColumns.add(key))
  }
}

// Save column settings to localStorage
const saveColumnsToStorage = () => {
  try {
    localStorage.setItem(HIDDEN_COLUMNS_KEY, JSON.stringify([...hiddenColumns]))
  } catch (e) {
    console.error('Failed to save columns:', e)
  }
}

// Toggle column visibility
const toggleColumn = (key: string) => {
  const wasHidden = hiddenColumns.has(key)
  if (hiddenColumns.has(key)) {
    hiddenColumns.delete(key)
  } else {
    hiddenColumns.add(key)
  }
  saveColumnsToStorage()
  if (wasHidden && (key === 'usage' || key.startsWith('attr_'))) {
    refreshCurrentPageSecondaryData()
  }
  if (key === 'subscriptions') {
    loadUsers()
  }
  if (wasHidden && key === 'groups') {
    loadAllGroups()
  }
}

// Check if column is visible (not in hidden set)
const isColumnVisible = (key: string) => !hiddenColumns.has(key)
const hasVisibleUsageColumn = computed(() => !hiddenColumns.has('usage'))
const hasVisibleSubscriptionsColumn = computed(() => !hiddenColumns.has('subscriptions'))
const hasVisibleGroupsColumn = computed(() => !hiddenColumns.has('groups'))
const hasVisibleAttributeColumns = computed(() =>
  attributeDefinitions.value.some((def) => def.enabled && !hiddenColumns.has(`attr_${def.id}`))
)

// Filtered columns based on visibility
const columns = computed<Column[]>(() =>
  allColumns.value.filter(col =>
    col.key === 'email' || col.key === 'actions' || !hiddenColumns.has(col.key)
  )
)

const users = ref<AdminUser[]>([])
const loading = ref(false)
const searchQuery = ref('')

// Groups data for the groups column
const allGroups = ref<AdminGroup[]>([])
const loadAllGroups = async () => {
  if (allGroups.value.length > 0) return
  try {
    allGroups.value = await adminAPI.groups.getAll()
  } catch (e) {
    console.error('Failed to load groups:', e)
  }
}
// Resolve user's accessible groups: exclusive groups first, then public groups
const getUserGroups = (user: AdminUser) => {
  const exclusive: AdminGroup[] = []
  const publicGroups: AdminGroup[] = []
  for (const g of allGroups.value) {
    if (g.status !== 'active' || g.subscription_type !== 'standard') continue
    if (g.is_exclusive) {
      if (user.allowed_groups?.includes(g.id)) {
        exclusive.push(g)
      }
    } else {
      publicGroups.push(g)
    }
  }
  return { exclusive, publicGroups }
}

// Group filter options: "All Groups" + active exclusive groups (value = group name for fuzzy match)
const groupFilterOptions = computed(() => {
  const options: { value: string; label: string }[] = [
    { value: '', label: t('admin.users.allGroups') }
  ]
  for (const g of allGroups.value) {
    if (g.status !== 'active' || !g.is_exclusive || g.subscription_type !== 'standard') continue
    options.push({ value: g.name, label: g.name })
  }
  return options
})

// Filter values (role, status, and custom attributes)
const filters = reactive({
  role: '',
  status: '',
  group: ''  // group name for fuzzy match, '' = all
})
const activeAttributeFilters = reactive<Record<number, string>>({})

// Visible filters tracking (which filters are shown in the UI)
// Keys: 'role', 'status', 'attr_${id}'
const visibleFilters = reactive<Set<string>>(new Set())

// Dropdown states
const showFilterDropdown = ref(false)
const showColumnDropdown = ref(false)

// Dropdown refs for click outside detection
const filterDropdownRef = ref<HTMLElement | null>(null)
const columnDropdownRef = ref<HTMLElement | null>(null)

// localStorage keys
const FILTER_VALUES_KEY = 'user-filter-values'
const VISIBLE_FILTERS_KEY = 'user-visible-filters'

// All filterable attribute definitions (enabled attributes)
const filterableAttributes = computed(() =>
  attributeDefinitions.value.filter(def => def.enabled)
)

// Built-in filter definitions
const builtInFilters = computed(() => [
  { key: 'role', name: t('admin.users.columns.role'), type: 'select' as const },
  { key: 'status', name: t('admin.users.columns.status'), type: 'select' as const },
  { key: 'group', name: t('admin.users.columns.groups'), type: 'select' as const }
])

// Load saved filters from localStorage
const loadSavedFilters = () => {
  try {
    // Load visible filters
    const savedVisible = localStorage.getItem(VISIBLE_FILTERS_KEY)
    if (savedVisible) {
      const parsed = JSON.parse(savedVisible) as string[]
      parsed.forEach(key => visibleFilters.add(key))
    }
    // Load filter values
    const savedValues = localStorage.getItem(FILTER_VALUES_KEY)
    if (savedValues) {
      const parsed = JSON.parse(savedValues)
      if (parsed.role) filters.role = parsed.role
      if (parsed.status) filters.status = parsed.status
      if (parsed.group) filters.group = parsed.group
      if (parsed.attributes) {
        Object.assign(activeAttributeFilters, parsed.attributes)
      }
    }
  } catch (e) {
    console.error('Failed to load saved filters:', e)
  }
}

// Save filters to localStorage
const saveFiltersToStorage = () => {
  try {
    // Save visible filters
    localStorage.setItem(VISIBLE_FILTERS_KEY, JSON.stringify([...visibleFilters]))
    // Save filter values
    const values = {
      role: filters.role,
      status: filters.status,
      group: filters.group,
      attributes: activeAttributeFilters
    }
    localStorage.setItem(FILTER_VALUES_KEY, JSON.stringify(values))
  } catch (e) {
    console.error('Failed to save filters:', e)
  }
}

// Get attribute definition by ID
const getAttributeDefinition = (attrId: number): UserAttributeDefinition | undefined => {
  return attributeDefinitions.value.find(d => d.id === attrId)
}
const usageStats = ref<Record<string, BatchUserUsageStats>>({})
type CustomerHandoffKeyReadiness = {
  loading: boolean
  total: number
  ready: number
  warning: number
  blocked: number
  notes: string[]
}

const createEmptyCustomerHandoffKeyReadiness = (): CustomerHandoffKeyReadiness => ({
  loading: false,
  total: 0,
  ready: 0,
  warning: 0,
  blocked: 0,
  notes: []
})

const customerHandoffKeyReadiness = ref<CustomerHandoffKeyReadiness>(
  createEmptyCustomerHandoffKeyReadiness()
)
// User attribute definitions and values
const attributeDefinitions = ref<UserAttributeDefinition[]>([])
const userAttributeValues = ref<Record<number, Record<number, string>>>({})
const pagination = reactive({
  page: 1,
  page_size: getPersistedPageSize(),
  total: 0,
  pages: 0
})

const showCreateModal = ref(false)
const showEditModal = ref(false)
const showDeleteDialog = ref(false)
const showApiKeysModal = ref(false)
const showAttributesModal = ref(false)
const showCustomerHandoffModal = ref(false)
const editingUser = ref<AdminUser | null>(null)
const deletingUser = ref<AdminUser | null>(null)
const viewingUser = ref<AdminUser | null>(null)
const customerHandoffUser = ref<AdminUser | null>(null)
let abortController: AbortController | null = null
let secondaryDataSeq = 0

const loadUsersSecondaryData = async (
  userIds: number[],
  signal?: AbortSignal,
  expectedSeq?: number
) => {
  if (userIds.length === 0) return

  const tasks: Promise<void>[] = []

  if (hasVisibleUsageColumn.value) {
    tasks.push(
      (async () => {
        try {
          const usageResponse = await adminAPI.dashboard.getBatchUsersUsage(userIds)
          if (signal?.aborted) return
          if (typeof expectedSeq === 'number' && expectedSeq !== secondaryDataSeq) return
          usageStats.value = usageResponse.stats
        } catch (e) {
          if (signal?.aborted) return
          console.error('Failed to load usage stats:', e)
        }
      })()
    )
  }

  if (attributeDefinitions.value.length > 0 && hasVisibleAttributeColumns.value) {
    tasks.push(
      (async () => {
        try {
          const attrResponse = await adminAPI.userAttributes.getBatchUserAttributes(userIds)
          if (signal?.aborted) return
          if (typeof expectedSeq === 'number' && expectedSeq !== secondaryDataSeq) return
          userAttributeValues.value = attrResponse.attributes
        } catch (e) {
          if (signal?.aborted) return
          console.error('Failed to load user attribute values:', e)
        }
      })()
    )
  }

  if (tasks.length > 0) {
    await Promise.allSettled(tasks)
  }
}

const refreshCurrentPageSecondaryData = () => {
  const userIds = users.value.map((u) => u.id)
  if (userIds.length === 0) return
  const seq = ++secondaryDataSeq
  void loadUsersSecondaryData(userIds, undefined, seq)
}

// Action Menu State
const activeMenuId = ref<number | null>(null)
const menuPosition = ref<{ top: number; left: number } | null>(null)

const openActionMenu = (user: AdminUser, e: MouseEvent) => {
  if (activeMenuId.value === user.id) {
    closeActionMenu()
  } else {
    const target = e.currentTarget as HTMLElement
    if (!target) {
      closeActionMenu()
      return
    }

    const rect = target.getBoundingClientRect()
    const menuWidth = 200
    const menuHeight = 240
    const padding = 8
    const viewportWidth = window.innerWidth
    const viewportHeight = window.innerHeight

    let left, top

    if (viewportWidth < 768) {
      // 居中显示,水平位置
      left = Math.max(padding, Math.min(
        rect.left + rect.width / 2 - menuWidth / 2,
        viewportWidth - menuWidth - padding
      ))

      // 优先显示在按钮下方
      top = rect.bottom + 4

      // 如果下方空间不够,显示在上方
      if (top + menuHeight > viewportHeight - padding) {
        top = rect.top - menuHeight - 4
        // 如果上方也不够,就贴在视口顶部
        if (top < padding) {
          top = padding
        }
      }
    } else {
      left = Math.max(padding, Math.min(
        e.clientX - menuWidth,
        viewportWidth - menuWidth - padding
      ))
      top = e.clientY
      if (top + menuHeight > viewportHeight - padding) {
        top = viewportHeight - menuHeight - padding
      }
    }

    menuPosition.value = { top, left }
    activeMenuId.value = user.id
  }
}

const closeActionMenu = () => {
  activeMenuId.value = null
  menuPosition.value = null
}

// Close menu when clicking outside
const handleClickOutside = (event: MouseEvent) => {
  const target = event.target as HTMLElement
  if (!target.closest('.action-menu-trigger') && !target.closest('.action-menu-content')) {
    closeActionMenu()
  }
  // Close filter dropdown when clicking outside
  if (filterDropdownRef.value && !filterDropdownRef.value.contains(target)) {
    showFilterDropdown.value = false
  }
  // Close column dropdown when clicking outside
  if (columnDropdownRef.value && !columnDropdownRef.value.contains(target)) {
    showColumnDropdown.value = false
  }
  // Close expanded group dropdown when clicking outside
  if (expandedGroupUserId.value !== null) {
    expandedGroupUserId.value = null
  }
}

// Allowed groups modal state
const showAllowedGroupsModal = ref(false)
const allowedGroupsUser = ref<AdminUser | null>(null)

// Expanded group dropdown state (click to show exclusive groups list)
const expandedGroupUserId = ref<number | null>(null)
const toggleExpandedGroup = (userId: number) => {
  expandedGroupUserId.value = expandedGroupUserId.value === userId ? null : userId
}

// Group replace modal state
const showGroupReplaceModal = ref(false)
const groupReplaceUser = ref<AdminUser | null>(null)
const groupReplaceOldGroup = ref<{ id: number; name: string } | null>(null)

// Balance (Deposit/Withdraw) modal state
const showBalanceModal = ref(false)
const balanceUser = ref<AdminUser | null>(null)
const balanceOperation = ref<'add' | 'subtract'>('add')

// Balance History modal state
const showBalanceHistoryModal = ref(false)
const balanceHistoryUser = ref<AdminUser | null>(null)

// 计算剩余天数
const getDaysRemaining = (expiresAt: string): number => {
  const now = new Date()
  const expires = new Date(expiresAt)
  const diffMs = expires.getTime() - now.getTime()
  return Math.ceil(diffMs / (1000 * 60 * 60 * 24))
}

const loadAttributeDefinitions = async () => {
  try {
    attributeDefinitions.value = await adminAPI.userAttributes.listEnabledDefinitions()
  } catch (e) {
    console.error('Failed to load attribute definitions:', e)
  }
}

// Handle attributes modal close - reload definitions and users
const handleAttributesModalClose = async () => {
  showAttributesModal.value = false
  await loadAttributeDefinitions()
  loadUsers()
}

const loadUsers = async () => {
  abortController?.abort()
  const currentAbortController = new AbortController()
  abortController = currentAbortController
  const { signal } = currentAbortController
  loading.value = true
  try {
    // Build attribute filters from active filters
    const attrFilters: Record<number, string> = {}
    for (const [attrId, value] of Object.entries(activeAttributeFilters)) {
      if (value) {
        attrFilters[Number(attrId)] = value
      }
    }

    const response = await adminAPI.users.list(
      pagination.page,
      pagination.page_size,
      {
        role: filters.role as any,
        status: filters.status as any,
        search: searchQuery.value || undefined,
        group_name: filters.group || undefined,
        attributes: Object.keys(attrFilters).length > 0 ? attrFilters : undefined,
        include_subscriptions: hasVisibleSubscriptionsColumn.value
      },
      { signal }
    )
    if (signal.aborted) {
      return
    }
    users.value = response.items
    pagination.total = response.total
    pagination.pages = response.pages
    usageStats.value = {}
    userAttributeValues.value = {}

    // Defer heavy secondary data so table can render first.
    if (response.items.length > 0) {
      const userIds = response.items.map((u) => u.id)
      const seq = ++secondaryDataSeq
      window.setTimeout(() => {
        if (signal.aborted || seq !== secondaryDataSeq) return
        void loadUsersSecondaryData(userIds, signal, seq)
      }, 50)
    }
  } catch (error: any) {
    const errorInfo = error as { name?: string; code?: string }
    if (errorInfo?.name === 'AbortError' || errorInfo?.name === 'CanceledError' || errorInfo?.code === 'ERR_CANCELED') {
      return
    }
    const message = error.response?.data?.detail || error.message || t('admin.users.failedToLoad')
    appStore.showError(message)
    console.error('Error loading users:', error)
  } finally {
    if (abortController === currentAbortController) {
      loading.value = false
    }
  }
}

let searchTimeout: ReturnType<typeof setTimeout>
const handleSearch = () => {
  clearTimeout(searchTimeout)
  searchTimeout = setTimeout(() => {
    pagination.page = 1
    loadUsers()
  }, 300)
}

const handlePageChange = (page: number) => {
  // 确保页码在有效范围内
  const validPage = Math.max(1, Math.min(page, pagination.pages || 1))
  pagination.page = validPage
  loadUsers()
}

const handlePageSizeChange = (pageSize: number) => {
  pagination.page_size = pageSize
  pagination.page = 1
  loadUsers()
}

// Filter helpers
const getAttributeDefinitionName = (attrId: number): string => {
  const def = attributeDefinitions.value.find(d => d.id === attrId)
  return def?.name || String(attrId)
}

// Toggle a built-in filter (role/status)
const toggleBuiltInFilter = (key: string) => {
  if (visibleFilters.has(key)) {
    visibleFilters.delete(key)
    if (key === 'role') filters.role = ''
    if (key === 'status') filters.status = ''
    if (key === 'group') filters.group = ''
  } else {
    visibleFilters.add(key)
    if (key === 'group') loadAllGroups()
  }
  saveFiltersToStorage()
  pagination.page = 1
  loadUsers()
}

// Toggle a custom attribute filter
const toggleAttributeFilter = (attr: UserAttributeDefinition) => {
  const key = `attr_${attr.id}`
  if (visibleFilters.has(key)) {
    visibleFilters.delete(key)
    delete activeAttributeFilters[attr.id]
  } else {
    visibleFilters.add(key)
    activeAttributeFilters[attr.id] = ''
  }
  saveFiltersToStorage()
  pagination.page = 1
  loadUsers()
}

const updateAttributeFilter = (attrId: number, value: string) => {
  activeAttributeFilters[attrId] = value
}

// Apply filter and save to localStorage
const applyFilter = () => {
  saveFiltersToStorage()
  loadUsers()
}

const handleEdit = (user: AdminUser) => {
  editingUser.value = user
  showEditModal.value = true
}

const closeEditModal = () => {
  showEditModal.value = false
  editingUser.value = null
}

const handleToggleStatus = async (user: AdminUser) => {
  const newStatus = user.status === 'active' ? 'disabled' : 'active'
  try {
    await adminAPI.users.toggleStatus(user.id, newStatus)
    appStore.showSuccess(
      newStatus === 'active' ? t('admin.users.userEnabled') : t('admin.users.userDisabled')
    )
    loadUsers()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.users.failedToToggle'))
    console.error('Error toggling user status:', error)
  }
}

const handleViewApiKeys = (user: AdminUser) => {
  viewingUser.value = user
  showApiKeysModal.value = true
}

const handleViewUsage = (user: AdminUser) => {
  void router.push({
    path: '/admin/usage',
    query: { user_id: String(user.id) }
  })
}

const handleViewOrders = (user: AdminUser) => {
  void router.push({
    path: '/admin/orders',
    query: { user_id: String(user.id) }
  })
}

const investigationKeywordForUser = (user: AdminUser) => user.email || user.username || String(user.id)

const handleViewAffiliate = (user: AdminUser) => {
  void router.push({
    path: '/admin/affiliates',
    query: { search: investigationKeywordForUser(user) }
  })
}

const handleViewRedeemCodes = (user: AdminUser) => {
  void router.push({
    path: '/admin/redeem',
    query: { search: investigationKeywordForUser(user) }
  })
}

const loadCustomerHandoffUsage = async (user: AdminUser) => {
  try {
    const usageResponse = await adminAPI.dashboard.getBatchUsersUsage([user.id])
    usageStats.value = {
      ...usageStats.value,
      ...usageResponse.stats
    }
  } catch (e) {
    console.error('Failed to load customer handoff usage stats:', e)
  }
}

const hasCustomerHandoffKeyGroup = (key: ApiKey) =>
  Boolean(
    key.group_id ||
    (key.group_ids && key.group_ids.length > 0) ||
    (key.groups && key.groups.length > 0)
  )

const isCustomerHandoffKeyExpired = (value: string | null) =>
  Boolean(value && new Date(value).getTime() <= Date.now())

const isCustomerHandoffKeyExpiringSoon = (value: string | null) => {
  if (!value) return false
  const expiresAt = new Date(value).getTime()
  const sevenDaysMs = 7 * 24 * 60 * 60 * 1000
  return expiresAt > Date.now() && expiresAt - Date.now() <= sevenDaysMs
}

const customerHandoffRateWindows = (key: ApiKey) => [
  { label: '5小时', limit: key.rate_limit_5h, usage: key.usage_5h },
  { label: '1天', limit: key.rate_limit_1d, usage: key.usage_1d },
  { label: '7天', limit: key.rate_limit_7d, usage: key.usage_7d }
]

const classifyCustomerHandoffKey = (user: AdminUser, key: ApiKey) => {
  const blockers: string[] = []
  const warnings: string[] = []

  if (user.balance <= 0) blockers.push('账户余额不足')
  if (key.status !== 'active') blockers.push(`Key 状态为 ${key.status}`)
  if (!hasCustomerHandoffKeyGroup(key)) blockers.push('Key 未绑定分组')
  if (isCustomerHandoffKeyExpired(key.expires_at)) blockers.push('Key 已过期')
  if (key.quota > 0 && key.quota_used >= key.quota) blockers.push('Key 额度已用完')

  const exhaustedWindows = customerHandoffRateWindows(key).filter((window) => window.limit > 0 && window.usage >= window.limit)
  if (exhaustedWindows.length > 0) {
    blockers.push(`${exhaustedWindows.map((window) => window.label).join('、')}限额已用完`)
  }

  if (isCustomerHandoffKeyExpiringSoon(key.expires_at)) warnings.push('Key 即将过期')
  if (key.quota > 0 && key.quota_used >= key.quota * 0.8 && key.quota_used < key.quota) warnings.push('Key 额度接近上限')

  const nearWindows = customerHandoffRateWindows(key).filter((window) => window.limit > 0 && window.usage >= window.limit * 0.8 && window.usage < window.limit)
  if (nearWindows.length > 0) {
    warnings.push(`${nearWindows.map((window) => window.label).join('、')}限额接近上限`)
  }
  if (key.allowed_models?.length) warnings.push('Key 有模型限制')
  if ((key.ip_whitelist?.length || 0) > 0 || (key.ip_blacklist?.length || 0) > 0) warnings.push('Key 有 IP 限制')
  if (!key.last_used_at) warnings.push('暂无调用记录')

  if (blockers.length > 0) return { level: 'blocked' as const, notes: [...blockers, ...warnings] }
  if (warnings.length > 0) return { level: 'warning' as const, notes: warnings }
  return { level: 'ready' as const, notes: ['状态、余额、分组、额度看起来可交付'] }
}

const loadCustomerHandoffKeyReadiness = async (user: AdminUser) => {
  customerHandoffKeyReadiness.value = {
    ...createEmptyCustomerHandoffKeyReadiness(),
    loading: true
  }
  try {
    const response = await adminAPI.users.getUserApiKeys(user.id)
    const keys = response.items || []
    const summary = createEmptyCustomerHandoffKeyReadiness()
    summary.total = keys.length

    const notes = new Set<string>()
    for (const key of keys) {
      const result = classifyCustomerHandoffKey(user, key)
      summary[result.level] += 1
      for (const note of result.notes.slice(0, 2)) {
        notes.add(note)
      }
    }

    summary.notes = [...notes].slice(0, 5)
    customerHandoffKeyReadiness.value = summary
  } catch (e) {
    console.error('Failed to load customer handoff key readiness:', e)
    customerHandoffKeyReadiness.value = {
      ...createEmptyCustomerHandoffKeyReadiness(),
      notes: ['Key 摘要加载失败，点 API Key 入口查看']
    }
  }
}

const handleCustomerHandoff = (user: AdminUser) => {
  customerHandoffUser.value = user
  showCustomerHandoffModal.value = true
  void loadAllGroups()
  void loadCustomerHandoffUsage(user)
  void loadCustomerHandoffKeyReadiness(user)
}

const closeCustomerHandoff = () => {
  showCustomerHandoffModal.value = false
  customerHandoffUser.value = null
  customerHandoffKeyReadiness.value = createEmptyCustomerHandoffKeyReadiness()
}

const withCustomerHandoffUser = (action: (user: AdminUser) => void) => {
  const user = customerHandoffUser.value
  if (!user) return
  closeCustomerHandoff()
  action(user)
}

const openCustomerHandoffApiKeys = () => withCustomerHandoffUser(handleViewApiKeys)
const openCustomerHandoffUsage = () => withCustomerHandoffUser(handleViewUsage)
const openCustomerHandoffOrders = () => withCustomerHandoffUser(handleViewOrders)
const openCustomerHandoffAffiliate = () => withCustomerHandoffUser(handleViewAffiliate)
const openCustomerHandoffRedeem = () => withCustomerHandoffUser(handleViewRedeemCodes)
const openCustomerHandoffGroups = () => withCustomerHandoffUser(handleAllowedGroups)
const openCustomerHandoffBalanceHistory = () => withCustomerHandoffUser(handleBalanceHistory)
const openCustomerHandoffChannelStatus = () => {
  closeCustomerHandoff()
  void router.push({ path: '/admin/channels/monitor' })
}
const openCustomerHandoffRequestDetails = () => withCustomerHandoffUser((user) => {
  void router.push({
    path: '/admin/ops',
    query: {
      tr: '24h',
      open_request_details: '1',
      user_id: String(user.id)
    }
  })
})

const formatCustomerHandoffUsage = (user: AdminUser) => {
  const stats = usageStats.value[user.id]
  if (!stats) return '加载中'
  return `${formatCurrency(stats.today_actual_cost ?? 0)} / ${formatCurrency(stats.total_actual_cost ?? 0)}`
}

const closeApiKeysModal = () => {
  showApiKeysModal.value = false
  viewingUser.value = null
}

const handleAllowedGroups = (user: AdminUser) => {
  allowedGroupsUser.value = user
  showAllowedGroupsModal.value = true
}

const closeAllowedGroupsModal = () => {
  showAllowedGroupsModal.value = false
  allowedGroupsUser.value = null
}

const openGroupReplace = (user: AdminUser, group: { id: number; name: string }) => {
  expandedGroupUserId.value = null
  groupReplaceUser.value = user
  groupReplaceOldGroup.value = group
  showGroupReplaceModal.value = true
}

const closeGroupReplaceModal = () => {
  showGroupReplaceModal.value = false
  groupReplaceUser.value = null
  groupReplaceOldGroup.value = null
}

const handleDelete = (user: AdminUser) => {
  deletingUser.value = user
  showDeleteDialog.value = true
}

const confirmDelete = async () => {
  if (!deletingUser.value) return
  try {
    await adminAPI.users.delete(deletingUser.value.id)
    appStore.showSuccess(t('common.success'))
    showDeleteDialog.value = false
    deletingUser.value = null
    loadUsers()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.users.failedToDelete'))
    console.error('Error deleting user:', error)
  }
}

const handleDeposit = (user: AdminUser) => {
  balanceUser.value = user
  balanceOperation.value = 'add'
  showBalanceModal.value = true
}

const handleWithdraw = (user: AdminUser) => {
  balanceUser.value = user
  balanceOperation.value = 'subtract'
  showBalanceModal.value = true
}

const closeBalanceModal = () => {
  showBalanceModal.value = false
  balanceUser.value = null
}

const handleBalanceHistory = (user: AdminUser) => {
  balanceHistoryUser.value = user
  showBalanceHistoryModal.value = true
}

const closeBalanceHistoryModal = () => {
  showBalanceHistoryModal.value = false
  balanceHistoryUser.value = null
}

// Handle deposit from balance history modal
const handleDepositFromHistory = () => {
  if (balanceHistoryUser.value) {
    handleDeposit(balanceHistoryUser.value)
  }
}

// Handle withdraw from balance history modal
const handleWithdrawFromHistory = () => {
  if (balanceHistoryUser.value) {
    handleWithdraw(balanceHistoryUser.value)
  }
}

// 滚动时关闭菜单
const handleScroll = () => {
  closeActionMenu()
}

onMounted(async () => {
  await loadAttributeDefinitions()
  loadSavedFilters()
  loadSavedColumns()
  loadUsers()
  if (hasVisibleGroupsColumn.value || visibleFilters.has('group')) {
    loadAllGroups()
  }
  document.addEventListener('click', handleClickOutside)
  window.addEventListener('scroll', handleScroll, true)
})

onUnmounted(() => {
  document.removeEventListener('click', handleClickOutside)
  window.removeEventListener('scroll', handleScroll, true)
  clearTimeout(searchTimeout)
  abortController?.abort()
})
</script>

<style scoped>
.admin-b2-outline-scope :deep(.table-scroll-container),
.admin-b2-outline-scope :deep(.table-wrapper),
.admin-b2-outline-scope :deep(.table-wrapper table),
.admin-b2-outline-scope :deep(.table-wrapper tbody) {
  background: transparent !important;
  border-color: var(--ssxz-border) !important;
  box-shadow: none !important;
}

.admin-b2-outline-scope :deep(thead),
.admin-b2-outline-scope :deep(.table-header) {
  background: var(--ssxz-surface-raised) !important;
}
</style>
