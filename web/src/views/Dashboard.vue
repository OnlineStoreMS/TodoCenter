<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { fetchDashboardStats, listCategories, type DashboardStats, type Category } from '../api/todo'

const router = useRouter()
const loading = ref(false)
const categories = ref<Category[]>([])
const stats = ref<DashboardStats>({
  total: 0,
  pending: 0,
  inProgress: 0,
  done: 0,
  cancelled: 0,
  byCategory: {},
})

const catNameByCode = computed(() => {
  const m: Record<string, string> = {}
  for (const c of categories.value) m[c.code] = c.name
  return m
})

const catIdByCode = computed(() => {
  const m: Record<string, number> = {}
  for (const c of categories.value) m[c.code] = c.id
  return m
})

const statusCards = computed(() => [
  {
    key: 'total',
    label: '全部',
    tip: '普通待办 + 月实例',
    value: stats.value.total,
    color: '#1677ff',
    go: () => goTodos(),
  },
  {
    key: 'pending',
    label: '待处理',
    tip: '需跟进',
    value: stats.value.pending,
    color: '#e6a23c',
    go: () => goTodos({ status: 'pending' }),
  },
  {
    key: 'in_progress',
    label: '进行中',
    tip: '处理中',
    value: stats.value.inProgress,
    color: '#409eff',
    go: () => goTodos({ status: 'in_progress' }),
  },
  {
    key: 'done',
    label: '已完成',
    tip: '已收工',
    value: stats.value.done,
    color: '#67c23a',
    go: () => goTodos({ status: 'done' }),
  },
  {
    key: 'cancelled',
    label: '已取消',
    tip: '不再处理',
    value: stats.value.cancelled,
    color: '#909399',
    go: () => goTodos({ status: 'cancelled' }),
  },
])

const entryCards = computed(() => [
  {
    key: 'list',
    label: '待办列表',
    tip: '新建、筛选与跟进',
    value: stats.value.total,
    color: '#1677ff',
    go: () => goTodos(),
  },
  {
    key: 'pending',
    label: '待处理',
    tip: '优先跟进',
    value: stats.value.pending,
    color: '#e6a23c',
    go: () => goTodos({ status: 'pending' }),
  },
  {
    key: 'monthly',
    label: '固定月待办',
    tip: '管理每月循环模板',
    value: '模板',
    color: '#f56c6c',
    go: () => goTodos({ recurrence: 'templates' }),
  },
  {
    key: 'done',
    label: '已完成',
    tip: '历史完成项',
    value: stats.value.done,
    color: '#0f766e',
    go: () => goTodos({ status: 'done' }),
  },
  {
    key: 'categories',
    label: '分类管理',
    tip: '电商 / 发货 / 售后 / 门店',
    value: categories.value.length,
    color: '#595959',
    go: () => router.push('/categories'),
  },
  {
    key: 'notifications',
    label: '通知管理',
    tip: '飞书到期提醒',
    value: '推送',
    color: '#722ed1',
    go: () => router.push('/notifications'),
  },
])

const categoryChips = computed(() => {
  const seed = ['ecommerce', 'shipping', 'aftersales', 'store']
  const codes = new Set([...seed, ...Object.keys(stats.value.byCategory || {})])
  return [...codes].map((code) => ({
    code,
    name: catNameByCode.value[code] || code,
    count: stats.value.byCategory?.[code] || 0,
  }))
})

async function load() {
  loading.value = true
  try {
    const [s, cats] = await Promise.all([fetchDashboardStats(), listCategories()])
    stats.value = s
    categories.value = cats
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

function goTodos(query: Record<string, string> = {}) {
  router.push({ path: '/todos', query })
}

function goCategory(code: string) {
  const id = catIdByCode.value[code]
  if (!id) {
    goTodos()
    return
  }
  goTodos({ categoryId: String(id) })
}

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="page">
    <div class="section-head">状态概览</div>
    <div class="work-cards">
      <button
        v-for="c in statusCards"
        :key="c.key"
        type="button"
        class="work-card"
        :style="{ '--accent': c.color }"
        @click="c.go()"
      >
        <div class="work-label">{{ c.label }}</div>
        <div class="work-value">{{ c.value }}</div>
        <div class="work-tip">{{ c.tip }}</div>
      </button>
    </div>

    <div class="section-head">快捷入口</div>
    <div class="work-cards">
      <button
        v-for="c in entryCards"
        :key="c.key"
        type="button"
        class="work-card"
        :style="{ '--accent': c.color }"
        @click="c.go()"
      >
        <div class="work-label">{{ c.label }}</div>
        <div class="work-value entry-value">{{ c.value }}</div>
        <div class="work-tip">{{ c.tip }}</div>
      </button>
    </div>

    <section class="status-panel">
      <h3>业务分类</h3>
      <div class="chips">
        <div
          v-for="c in categoryChips"
          :key="c.code"
          class="chip"
          @click="goCategory(c.code)"
        >
          <span>{{ c.name }}</span>
          <strong>{{ c.count }}</strong>
        </div>
        <div v-if="!categoryChips.length" class="empty">暂无分类数据</div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.page {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: 100%;
}

.section-head {
  font-size: 13px;
  font-weight: 600;
  color: #64748b;
  margin-top: 2px;
}

.work-cards {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 12px;
  width: 100%;
}

.work-card {
  text-align: left;
  border: 1px solid #e8edf3;
  background: #fff;
  border-radius: 10px;
  padding: 14px 16px;
  cursor: pointer;
  border-top: 3px solid var(--accent, #1677ff);
  transition: box-shadow 0.15s, border-color 0.15s;
  font: inherit;
  color: inherit;
}
.work-card:hover {
  box-shadow: 0 4px 14px rgba(15, 39, 68, 0.08);
}

.work-label {
  font-size: 13px;
  color: #64748b;
}
.work-value {
  margin-top: 6px;
  font-size: 28px;
  font-weight: 700;
  color: #0f172a;
  line-height: 1.1;
}
.entry-value {
  font-size: 24px;
}
.work-tip {
  margin-top: 6px;
  font-size: 12px;
  color: #94a3b8;
}

.status-panel {
  background: #fff;
  border-radius: 10px;
  padding: 16px 18px;
  border: 1px solid #eef0f3;
  width: 100%;
}
.status-panel h3 {
  margin: 0 0 10px;
  font-size: 15px;
  color: #334155;
}

.chips {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}
.chip {
  padding: 12px 14px;
  border-radius: 8px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  cursor: pointer;
  display: flex;
  justify-content: space-between;
  gap: 12px;
}
.chip:hover {
  border-color: #93c5fd;
  background: #eff6ff;
}
.chip strong {
  color: #0f172a;
}
.empty {
  color: #94a3b8;
  font-size: 13px;
}

@media (max-width: 1100px) {
  .work-cards {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
  .chips {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (max-width: 700px) {
  .work-cards,
  .chips {
    grid-template-columns: 1fr 1fr;
  }
}
</style>
