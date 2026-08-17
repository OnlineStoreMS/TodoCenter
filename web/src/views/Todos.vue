<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import MediaUploadField from '../components/MediaUploadField.vue'
import type { MediaItem } from '../api/upload'
import {
  createTodo,
  deleteTodo,
  listCategories,
  listTodos,
  updateTodo,
  updateTodoStatus,
  type Category,
  type Todo,
} from '../api/todo'

const route = useRoute()
const loading = ref(false)
const saving = ref(false)
const todos = ref<Todo[]>([])
const total = ref(0)
const categories = ref<Category[]>([])
const dialogVisible = ref(false)
const editingId = ref<number | null>(null)
const editingIsTemplate = ref(false)
const editingIsInstance = ref(false)

const filters = reactive({
  categoryId: undefined as number | undefined,
  status: '' as string,
  priority: '' as string,
  recurrence: '' as string,
  keyword: '',
  sortBy: '' as string,
  sortOrder: '' as '' | 'asc' | 'desc',
  page: 1,
  pageSize: 20,
})

const form = reactive({
  categoryId: 0,
  title: '',
  description: '',
  status: 'pending',
  priority: 'normal',
  recurrence: 'none',
  recurrenceDay: 1,
  dueAt: '',
  images: [] as MediaItem[],
})

const statusOptions = [
  { label: '全部状态', value: '' },
  { label: '待处理', value: 'pending' },
  { label: '进行中', value: 'in_progress' },
  { label: '已完成', value: 'done' },
  { label: '已取消', value: 'cancelled' },
]

const priorityFilterOptions = [
  { label: '全部优先级', value: '' },
  { label: '高', value: 'high' },
  { label: '普通', value: 'normal' },
  { label: '低', value: 'low' },
]

const priorityOptions = [
  { label: '低', value: 'low' },
  { label: '普通', value: 'normal' },
  { label: '高', value: 'high' },
]

const recurrenceFilterOptions = [
  { label: '全部待办', value: '' },
  { label: '普通待办', value: 'none' },
  { label: '月待办实例', value: 'instances' },
  { label: '固定模板', value: 'templates' },
]

const dayOptions = Array.from({ length: 28 }, (_, i) => ({ label: `每月 ${i + 1} 日`, value: i + 1 }))

const dialogTitle = computed(() => {
  if (!editingId.value) return '新建待办'
  if (editingIsTemplate.value) return '编辑固定月待办'
  if (editingIsInstance.value) return '编辑本月待办'
  return '编辑待办'
})

const isMonthlyForm = computed(() => form.recurrence === 'monthly')

async function loadCategories() {
  categories.value = await listCategories()
  if (!form.categoryId && categories.value.length) {
    form.categoryId = categories.value[0].id
  }
}

async function loadTodos() {
  loading.value = true
  try {
    const data = await listTodos({
      categoryId: filters.categoryId,
      status: filters.status || undefined,
      priority: filters.priority || undefined,
      keyword: filters.keyword || undefined,
      recurrence: filters.recurrence || undefined,
      sortBy: filters.sortBy || undefined,
      sortOrder: filters.sortOrder || undefined,
      page: filters.page,
      pageSize: filters.pageSize,
    })
    todos.value = data.list || []
    total.value = data.total || 0
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.value = false
  }
}

function resetForm() {
  editingId.value = null
  editingIsTemplate.value = false
  editingIsInstance.value = false
  form.categoryId = categories.value[0]?.id || 0
  form.title = ''
  form.description = ''
  form.status = 'pending'
  form.priority = 'normal'
  form.recurrence = 'none'
  form.recurrenceDay = 1
  form.dueAt = ''
  form.images = []
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: Todo) {
  editingId.value = row.id
  editingIsTemplate.value = !!row.isTemplate
  editingIsInstance.value = !!row.isMonthlyInstance
  form.categoryId = row.categoryId
  form.title = row.title
  form.description = row.description || ''
  form.status = row.status
  form.priority = row.priority
  form.recurrence = row.isTemplate || row.recurrence === 'monthly' ? 'monthly' : 'none'
  if (row.isMonthlyInstance) {
    // 实例不可改循环类型；展示为月待办只读语义
    form.recurrence = 'monthly'
  }
  form.recurrenceDay = row.recurrenceDay || 1
  form.dueAt = row.dueAt || ''
  form.images = [...(row.images || [])]
  dialogVisible.value = true
}

async function save() {
  if (!form.title.trim()) {
    ElMessage.warning('请填写标题')
    return
  }
  if (!form.categoryId) {
    ElMessage.warning('请选择分类')
    return
  }
  saving.value = true
  try {
    const payload: Record<string, unknown> = {
      categoryId: form.categoryId,
      title: form.title.trim(),
      description: form.description,
      status: form.status,
      priority: form.priority,
      images: form.images,
    }
    if (editingIsInstance.value) {
      // 月实例：只改本月内容，不改循环
      payload.dueAt = form.dueAt || undefined
      if (!form.dueAt) payload.clearDueAt = true
    } else {
      payload.recurrence = form.recurrence
      payload.recurrenceDay = form.recurrence === 'monthly' ? form.recurrenceDay : 1
      if (form.recurrence === 'none') {
        payload.dueAt = form.dueAt || undefined
        if (!form.dueAt) payload.clearDueAt = true
      }
    }
    if (editingId.value) {
      await updateTodo(editingId.value, payload)
      ElMessage.success('已保存')
    } else {
      await createTodo(payload as any)
      ElMessage.success(form.recurrence === 'monthly' ? '已创建月待办（本月实例已生成）' : '已创建')
    }
    dialogVisible.value = false
    await loadTodos()
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    saving.value = false
  }
}

async function onStatusChange(row: Todo, status: string) {
  if (row.isTemplate) {
    ElMessage.warning('请对本月实例改状态；固定模板请在「固定模板」中管理')
    return
  }
  try {
    await updateTodoStatus(row.id, status)
    ElMessage.success('状态已更新')
    await loadTodos()
  } catch (e) {
    ElMessage.error((e as Error).message || '更新失败')
  }
}

async function onDelete(row: Todo) {
  const tip = row.isTemplate
    ? `删除固定模板「${row.title}」后将不再每月生成；已有月份实例仍会保留。确定删除？`
    : `确定删除「${row.title}」？`
  try {
    await ElMessageBox.confirm(tip, '删除待办', { type: 'warning' })
  } catch {
    return
  }
  try {
    await deleteTodo(row.id)
    ElMessage.success('已删除')
    await loadTodos()
  } catch (e) {
    ElMessage.error((e as Error).message || '删除失败')
  }
}

function onFilterChange() {
  filters.page = 1
  void loadTodos()
}

function onSortChange(payload: { prop: string; order: string | null }) {
  const propMap: Record<string, string> = {
    categoryName: 'category',
    priority: 'priority',
    status: 'status',
    dueAt: 'dueAt',
    updatedAt: 'updatedAt',
    title: 'title',
  }
  const sortBy = propMap[payload.prop] || ''
  if (!sortBy || !payload.order) {
    filters.sortBy = ''
    filters.sortOrder = ''
  } else {
    filters.sortBy = sortBy
    filters.sortOrder = payload.order === 'ascending' ? 'asc' : 'desc'
  }
  filters.page = 1
  void loadTodos()
}

function formatDateTime(v?: string) {
  if (!v) return '-'
  return v.replace('T', ' ').replace(/\.\d+Z?$/, '').slice(0, 19)
}

function recurrenceTag(row: Todo) {
  if (row.isTemplate) return { text: '固定模板', type: 'warning' as const }
  if (row.isMonthlyInstance) return { text: row.periodKey ? `月待办 ${row.periodKey}` : '月待办', type: '' as const }
  return null
}

watch(
  () => [route.query.status, route.query.categoryId, route.query.recurrence, route.query.priority] as const,
  ([status, categoryId, recurrence, priority]) => {
    filters.status = typeof status === 'string' ? status : ''
    filters.recurrence = typeof recurrence === 'string' ? recurrence : ''
    filters.priority = typeof priority === 'string' ? priority : ''
    if (typeof categoryId === 'string' && categoryId) {
      const n = Number(categoryId)
      filters.categoryId = Number.isFinite(n) && n > 0 ? n : undefined
    } else {
      filters.categoryId = undefined
    }
    filters.page = 1
    void loadTodos()
  },
  { immediate: true },
)

onMounted(async () => {
  await loadCategories()
})
</script>

<template>
  <div class="todo-page">
    <el-card shadow="never">
      <template #header>
        <div class="row-between">
          <div class="card-title">
            待办列表 <span class="count">({{ total }})</span>
            <span class="hint">月待办会在每月自动生成一条实例</span>
          </div>
          <el-button type="primary" @click="openCreate">新建待办</el-button>
        </div>
      </template>

      <div class="filters">
        <el-select v-model="filters.categoryId" clearable placeholder="全部分类" style="width: 140px" @change="onFilterChange">
          <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
        </el-select>
        <el-select v-model="filters.priority" style="width: 130px" @change="onFilterChange">
          <el-option v-for="o in priorityFilterOptions" :key="o.value || 'all'" :label="o.label" :value="o.value" />
        </el-select>
        <el-select v-model="filters.status" style="width: 130px" @change="onFilterChange">
          <el-option v-for="o in statusOptions" :key="o.value || 'all'" :label="o.label" :value="o.value" />
        </el-select>
        <el-select v-model="filters.recurrence" style="width: 140px" @change="onFilterChange">
          <el-option v-for="o in recurrenceFilterOptions" :key="o.value || 'all'" :label="o.label" :value="o.value" />
        </el-select>
        <el-input
          v-model="filters.keyword"
          clearable
          placeholder="搜索标题/内容"
          style="width: 200px"
          @keyup.enter="onFilterChange"
          @clear="onFilterChange"
        />
        <el-button type="primary" :loading="loading" @click="loadTodos">查询</el-button>
      </div>

      <el-table
        :data="todos"
        v-loading="loading"
        stripe
        border
        empty-text="暂无待办"
        @sort-change="onSortChange"
      >
        <el-table-column prop="title" label="标题" min-width="200" show-overflow-tooltip sortable="custom">
          <template #default="{ row }">
            <span>{{ row.title }}</span>
            <el-tag v-if="recurrenceTag(row)" :type="recurrenceTag(row)!.type" size="small" class="ml">
              {{ recurrenceTag(row)!.text }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="categoryName" label="分类" width="100" sortable="custom" />
        <el-table-column prop="priority" label="优先级" width="100" sortable="custom">
          <template #default="{ row }">
            <el-tag v-if="row.priority === 'high'" type="danger" size="small">高</el-tag>
            <el-tag v-else-if="row.priority === 'low'" type="info" size="small">低</el-tag>
            <span v-else>普通</span>
          </template>
        </el-table-column>
        <el-table-column prop="status" label="状态" width="140" class-name="col-status" sortable="custom">
          <template #default="{ row }">
            <el-select
              v-if="!row.isTemplate"
              :model-value="row.status"
              size="small"
              class="status-select"
              @change="(v: string) => onStatusChange(row, v)"
            >
              <el-option v-for="o in statusOptions.filter((x) => x.value)" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
            <span v-else class="muted">模板</span>
          </template>
        </el-table-column>
        <el-table-column label="图片" width="100">
          <template #default="{ row }">
            <div v-if="row.images?.length" class="thumbs">
              <el-image
                v-for="(img, i) in row.images.slice(0, 3)"
                :key="i"
                :src="img.url"
                fit="cover"
                class="thumb"
                :preview-src-list="row.images.map((x: MediaItem) => x.url)"
                preview-teleported
              />
            </div>
            <span v-else class="muted">-</span>
          </template>
        </el-table-column>
        <el-table-column prop="dueAt" label="截止" width="170" show-overflow-tooltip sortable="custom">
          <template #default="{ row }">
            {{ formatDateTime(row.dueAt) }}
          </template>
        </el-table-column>
        <el-table-column prop="updatedAt" label="更新" width="170" show-overflow-tooltip sortable="custom">
          <template #default="{ row }">
            {{ formatDateTime(row.updatedAt) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEdit(row)">编辑</el-button>
            <el-button link type="danger" @click="onDelete(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>

      <div class="pager" v-if="total > filters.pageSize">
        <el-pagination
          background
          layout="total, prev, pager, next"
          :total="total"
          :page-size="filters.pageSize"
          :current-page="filters.page"
          @current-change="(p: number) => { filters.page = p; loadTodos() }"
        />
      </div>
    </el-card>

    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="640px" destroy-on-close>
      <el-form label-width="96px">
        <el-form-item label="分类" required>
          <el-select v-model="form.categoryId" style="width: 100%">
            <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="标题" required>
          <el-input v-model="form.title" maxlength="200" show-word-limit />
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="form.description" type="textarea" :rows="3" />
        </el-form-item>

        <el-form-item v-if="!editingIsInstance" label="循环">
          <el-radio-group v-model="form.recurrence" :disabled="editingIsInstance">
            <el-radio-button label="none">普通待办</el-radio-button>
            <el-radio-button label="monthly">固定月待办</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="isMonthlyForm && !editingIsInstance" label="每月几号">
          <el-select v-model="form.recurrenceDay" style="width: 200px">
            <el-option v-for="d in dayOptions" :key="d.value" :label="d.label" :value="d.value" />
          </el-select>
          <span class="form-tip">每月自动生成一条待处理实例</span>
        </el-form-item>
        <el-alert
          v-if="editingIsInstance"
          type="info"
          :closable="false"
          show-icon
          title="这是本月生成的月待办实例，改状态/内容只影响本月；固定规则请到「固定模板」筛选中编辑。"
          style="margin-bottom: 12px"
        />

        <el-form-item v-if="!isMonthlyForm || editingIsInstance" label="状态">
          <el-radio-group v-model="form.status">
            <el-radio-button v-for="o in statusOptions.filter((x) => x.value)" :key="o.value" :label="o.value">
              {{ o.label }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="优先级">
          <el-radio-group v-model="form.priority">
            <el-radio-button v-for="o in priorityOptions" :key="o.value" :label="o.value">{{ o.label }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="!isMonthlyForm || editingIsInstance" label="截止时间">
          <el-date-picker
            v-model="form.dueAt"
            type="datetime"
            value-format="YYYY-MM-DD HH:mm:ss"
            placeholder="可选"
            style="width: 100%"
          />
        </el-form-item>
        <el-form-item label="图片笔记">
          <MediaUploadField v-model="form.images" subdir="todos" :max-count="12" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<style scoped>
.row-between { display: flex; justify-content: space-between; align-items: center; gap: 12px; }
.card-title { font-weight: 600; }
.count { color: #909399; font-weight: 400; }
.hint { margin-left: 12px; color: #909399; font-size: 12px; font-weight: 400; }
.filters { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 16px; }
.pager { margin-top: 16px; display: flex; justify-content: flex-end; }
.thumbs { display: flex; gap: 4px; }
.thumb { width: 36px; height: 36px; border-radius: 4px; }
.muted { color: #909399; }
.ml { margin-left: 8px; }
.form-tip { margin-left: 12px; color: #909399; font-size: 12px; }
.status-select { width: 100%; }
:deep(.col-status .cell) {
  overflow: visible;
  text-overflow: clip;
}
</style>
