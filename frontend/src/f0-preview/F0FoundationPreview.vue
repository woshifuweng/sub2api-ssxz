<template>
  <FoundationProvider :theme="theme">
    <div class="f0-preview-layout">
      <FoundationSidebar
        title="SSXZ UI"
        subtitle="Foundation 0 / isolated"
        :sections="sidebarSections"
        :active-id="activeSection"
        :mobile-open="mobileNavOpen"
        @select="activeSection = $event"
        @close="mobileNavOpen = false"
      >
        <template #footer>
          <div class="f0-preview-sidebar-user">
            <span class="f0-preview-avatar">OP</span>
            <div class="f0-preview-sidebar-user-copy">
              <strong>运营管理员</strong>
              <span>preview@ssxz.local</span>
            </div>
            <MoreHorizontal :size="16" aria-hidden="true" />
          </div>
        </template>
      </FoundationSidebar>

      <main class="f0-preview-main">
        <header class="f0-preview-topbar">
          <div class="f0-preview-topbar-left">
            <FoundationButton
              class="f0-preview-mobile-menu"
              variant="ghost"
              size="icon"
              aria-label="打开导航"
              title="打开导航"
              @click="mobileNavOpen = true"
            >
              <Menu aria-hidden="true" />
            </FoundationButton>
            <div class="f0-preview-breadcrumb">
              <span>设计系统</span>
              <ChevronRight :size="14" aria-hidden="true" />
              <strong>组件样板</strong>
            </div>
          </div>
          <div class="f0-preview-topbar-actions">
            <FoundationButton
              class="f0-preview-desktop-action"
              variant="outline"
              size="sm"
              disabled
            >
              仅供预览
            </FoundationButton>
            <FoundationButton
              variant="ghost"
              size="icon"
              :aria-label="theme === 'light' ? '切换到深色模式' : '切换到浅色模式'"
              :title="theme === 'light' ? '切换到深色模式' : '切换到浅色模式'"
              data-testid="theme-toggle"
              @click="toggleTheme"
            >
              <Moon v-if="theme === 'light'" aria-hidden="true" />
              <Sun v-else aria-hidden="true" />
            </FoundationButton>
            <FoundationButton variant="outline" size="icon" aria-label="通知" title="通知">
              <Bell aria-hidden="true" />
            </FoundationButton>
          </div>
        </header>

        <div class="f0-preview-content">
          <div class="f0-preview-page-header">
            <div>
              <p class="f0-preview-eyebrow">Foundation 0</p>
              <h1 class="f0-preview-title">组件基线</h1>
              <p class="f0-preview-subtitle">
                只验证结构、颜色、密度和交互状态。该入口不接生产路由，任何正式页面接入前都需单独审批。
              </p>
            </div>
            <FoundationButton @click="dialogOpen = true">
              <template #leading><Plus aria-hidden="true" /></template>
              打开对话框
            </FoundationButton>
          </div>

          <section class="f0-preview-kpi-grid" aria-label="统计卡片样板">
            <FoundationCard v-for="metric in metrics" :key="metric.label">
              <div class="f0-preview-kpi">
                <div class="f0-preview-kpi-label">{{ metric.label }}</div>
                <div class="f0-preview-kpi-value">{{ metric.value }}</div>
                <div class="f0-preview-kpi-meta">{{ metric.meta }}</div>
              </div>
            </FoundationCard>
          </section>

          <div class="f0-preview-grid">
            <FoundationCard
              title="颜色与操作"
              description="shadcn token 语义；单一主色，状态色只表达状态。"
            >
              <div class="f0-preview-palette" aria-label="颜色样板">
                <div v-for="swatch in swatches" :key="swatch.name" class="f0-preview-swatch">
                  <div
                    class="f0-preview-swatch-color"
                    :class="`f0-preview-swatch--${swatch.className}`"
                  />
                  <div class="f0-preview-swatch-name">{{ swatch.name }}</div>
                </div>
              </div>

              <div class="f0-preview-section-label">Button</div>
              <div class="f0-preview-action-rail" aria-label="按钮层级样板">
                <div class="f0-preview-button-row" data-action-tier="primary">
                  <FoundationButton>
                    <template #leading><Save aria-hidden="true" /></template>
                    保存设置
                  </FoundationButton>
                  <FoundationButton variant="secondary">
                    <template #leading><Eye aria-hidden="true" /></template>
                    预览变更
                  </FoundationButton>
                  <FoundationButton variant="outline">
                    <template #leading><Download aria-hidden="true" /></template>
                    导出记录
                  </FoundationButton>
                </div>
                <div class="f0-preview-button-row" data-action-tier="utility">
                  <FoundationButton variant="ghost">更多操作</FoundationButton>
                  <FoundationButton variant="destructive">
                    <template #leading><Trash2 aria-hidden="true" /></template>
                    删除条目
                  </FoundationButton>
                  <FoundationButton disabled>不可用</FoundationButton>
                </div>
              </div>

              <div class="f0-preview-section-label">Badge</div>
              <div class="f0-preview-badge-row">
                <FoundationBadge>主标签</FoundationBadge>
                <FoundationBadge variant="secondary">0.8x 倍率</FoundationBadge>
                <FoundationBadge variant="outline">OpenAI</FoundationBadge>
                <FoundationBadge variant="success">运行中</FoundationBadge>
                <FoundationBadge variant="warning">观察中</FoundationBadge>
                <FoundationBadge variant="destructive">已禁用</FoundationBadge>
              </div>
            </FoundationCard>

            <div class="f0-preview-stack">
              <FoundationCard title="图标规范" description="动作图标统一线性风格；供应商图标使用 ModelIcon。">
                <div class="f0-preview-icon-set" data-testid="f0-icon-set" aria-label="图标规范样板">
                  <div class="f0-preview-icon-family">
                    <span class="f0-preview-icon-family-label">动作</span>
                    <div class="f0-preview-icon-row">
                      <Search :size="20" aria-hidden="true" />
                      <Bell :size="20" aria-hidden="true" />
                      <Settings :size="20" aria-hidden="true" />
                      <ShieldCheck :size="20" aria-hidden="true" />
                    </div>
                  </div>
                  <div class="f0-preview-icon-family">
                    <span class="f0-preview-icon-family-label">供应商</span>
                    <div class="f0-preview-icon-row f0-preview-provider-icons">
                      <ModelIcon model="gpt-5.5" size="22px" />
                      <ModelIcon model="claude-opus-4-8" size="22px" />
                      <ModelIcon model="gemini-2.5-pro" size="22px" />
                      <ModelIcon model="grok-4" size="22px" />
                    </div>
                  </div>
                </div>
              </FoundationCard>

              <FoundationCard title="输入与校验" description="标签、帮助文本和错误信息保持固定层级。">
                <div class="f0-preview-form-grid">
                  <FoundationInput
                    v-model="formName"
                    label="显示名称"
                    placeholder="例如：客户测试 Key"
                    help="用于后台识别，不会成为凭据。"
                  />
                  <FoundationInput
                    v-model="formEmail"
                    label="通知邮箱"
                    placeholder="name@example.com"
                    error="示例错误：请输入有效邮箱。"
                  />
                </div>
              </FoundationCard>

              <FoundationCard title="Dialog" description="使用浏览器原生 modal 能力管理焦点与 Escape。">
                <p class="f0-preview-dialog-note">
                  危险操作使用 AlertDialog 语义；普通表单使用 Dialog。确认层不展示密钥等敏感值。
                </p>
                <template #footer>
                  <FoundationButton variant="outline" @click="dialogOpen = true">
                    查看交互
                  </FoundationButton>
                </template>
              </FoundationCard>
            </div>
          </div>

          <FoundationCard title="数据表格" description="主信息正常权重；ID、时间和说明降为次要信息。">
            <template #action>
              <div class="f0-preview-card-action">
                <FoundationBadge variant="outline">4 条示例</FoundationBadge>
                <FoundationButton variant="outline" size="sm">
                  <template #leading><Search aria-hidden="true" /></template>
                  筛选
                </FoundationButton>
              </div>
            </template>

            <FoundationTable>
              <template #header>
                <tr>
                  <th>客户</th>
                  <th>分组</th>
                  <th>近 30 天消费</th>
                  <th>状态</th>
                  <th>更新时间</th>
                  <th aria-label="操作" />
                </tr>
              </template>
              <tr v-for="row in tableRows" :key="row.id">
                <td>
                  <div class="f0-preview-table-primary">
                    <span class="f0-preview-avatar">{{ row.initials }}</span>
                    <div class="f0-preview-table-user">
                      <strong>{{ row.name }}</strong>
                      <span>#{{ row.id }} · {{ row.email }}</span>
                    </div>
                  </div>
                </td>
                <td>
                  <strong>{{ row.group }}</strong>
                  <div class="f0-preview-muted">{{ row.provider }}</div>
                </td>
                <td>
                  <strong>{{ row.cost }}</strong>
                  <div class="f0-preview-muted">{{ row.requests }} 次请求</div>
                </td>
                <td><FoundationBadge :variant="row.statusVariant">{{ row.status }}</FoundationBadge></td>
                <td>
                  <span>{{ row.updatedAt }}</span>
                  <div class="f0-preview-muted">UTC+8</div>
                </td>
                <td>
                  <FoundationButton variant="ghost" size="icon" aria-label="更多操作" title="更多操作">
                    <MoreHorizontal aria-hidden="true" />
                  </FoundationButton>
                </td>
              </tr>
            </FoundationTable>
          </FoundationCard>
        </div>
      </main>
    </div>

    <FoundationDialog
      v-model:open="dialogOpen"
      title="创建测试条目"
      description="标准表单弹窗示例，不连接任何接口。"
    >
      <FoundationInput v-model="dialogName" label="名称" placeholder="输入名称" />
      <FoundationInput
        model-value=""
        label="API Key"
        type="password"
        placeholder="仅演示输入结构"
        help="敏感值不会出现在确认摘要或截图中。"
      />
      <template #footer>
        <FoundationButton variant="outline" @click="dialogOpen = false">取消</FoundationButton>
        <FoundationButton @click="dialogOpen = false">确认创建</FoundationButton>
      </template>
    </FoundationDialog>
  </FoundationProvider>
</template>

<script setup lang="ts">
import {
  Activity,
  Bell,
  ChevronRight,
  Code2,
  Database,
  Download,
  Eye,
  LayoutDashboard,
  Menu,
  Moon,
  MoreHorizontal,
  Plus,
  Save,
  Search,
  Settings,
  ShieldCheck,
  Sun,
  Trash2,
  Users
} from '@lucide/vue'
import { ref } from 'vue'
import ModelIcon from '@/components/common/ModelIcon.vue'
import {
  FoundationBadge,
  FoundationButton,
  FoundationCard,
  FoundationDialog,
  FoundationInput,
  FoundationProvider,
  FoundationSidebar,
  FoundationTable
} from '@/components/foundation'
import type { FoundationSidebarSection } from '@/components/foundation'

type BadgeVariant = 'success' | 'warning' | 'destructive'

const theme = ref<'light' | 'dark'>('light')
const mobileNavOpen = ref(false)
const dialogOpen = ref(false)
const activeSection = ref('foundation')
const formName = ref('首批客户')
const formEmail = ref('invalid-address')
const dialogName = ref('')

const toggleTheme = () => {
  theme.value = theme.value === 'light' ? 'dark' : 'light'
}

const sidebarSections: FoundationSidebarSection[] = [
  {
    label: '工作区',
    items: [
      { id: 'overview', label: '运营概览', icon: LayoutDashboard },
      { id: 'customers', label: '客户管理', icon: Users },
      { id: 'usage', label: '用量监控', icon: Activity }
    ]
  },
  {
    label: '设计基线',
    items: [
      { id: 'foundation', label: '组件样板', icon: Code2, badge: 'F0' },
      { id: 'data', label: '数据结构', icon: Database },
      { id: 'security', label: '安全状态', icon: ShieldCheck }
    ]
  },
  {
    label: '系统',
    items: [{ id: 'settings', label: '站点设置', icon: Settings }]
  }
]

const metrics = [
  { label: '活跃客户', value: '128', meta: '近 30 天 +12%' },
  { label: '有效 Key', value: '246', meta: '7 个本周新建' },
  { label: '客户消费', value: '$842.60', meta: '按 actual_cost 汇总' },
  { label: '系统状态', value: '正常', meta: '无未处理告警' }
]

const swatches = [
  { name: 'Background', className: 'background' },
  { name: 'Card', className: 'card' },
  { name: 'Primary', className: 'primary' },
  { name: 'Steel accent', className: 'brand' },
  { name: 'Success', className: 'success' },
  { name: 'Destructive', className: 'destructive' }
]

const tableRows: Array<{
  id: number
  initials: string
  name: string
  email: string
  group: string
  provider: string
  cost: string
  requests: number
  status: string
  statusVariant: BadgeVariant
  updatedAt: string
}> = [
  {
    id: 1042,
    initials: 'LX',
    name: '林晓',
    email: 'lin@example.test',
    group: 'GPT Pro池',
    provider: 'OpenAI · 1.0x',
    cost: '$126.40',
    requests: 184,
    status: '启用',
    statusVariant: 'success',
    updatedAt: '07-15 09:42'
  },
  {
    id: 1039,
    initials: 'CY',
    name: '陈宇',
    email: 'chen@example.test',
    group: 'Claude 满血池',
    provider: 'Anthropic · 1.2x',
    cost: '$98.16',
    requests: 76,
    status: '启用',
    statusVariant: 'success',
    updatedAt: '07-15 09:18'
  },
  {
    id: 1028,
    initials: 'WM',
    name: '王敏',
    email: 'wang@example.test',
    group: 'Codex池',
    provider: 'OpenAI · 0.8x',
    cost: '$42.08',
    requests: 52,
    status: '观察',
    statusVariant: 'warning',
    updatedAt: '07-14 22:06'
  },
  {
    id: 1016,
    initials: 'ZZ',
    name: '周舟',
    email: 'zhou@example.test',
    group: 'Kiro高缓池',
    provider: 'Anthropic · 0.8x',
    cost: '$0.00',
    requests: 0,
    status: '暂停',
    statusVariant: 'destructive',
    updatedAt: '07-13 18:30'
  }
]
</script>
