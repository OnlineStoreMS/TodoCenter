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

const filters = reactive({
  categoryId: undefined as number | undefined,
  status: '' as string,
  keyword: '',
  page: 1,
  pageSize: 20,
})

const form = reactive({
  categoryId: 0,
  title: '',
  description: '',
  status: 'pending',
  priority: 'normal',
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

const priorityOptions = [
  { label: '低', value: 'low' },
  { label: '普通', value: 'normal' },
  { label: '高', value: 'high' },
]

const dialogTitle = computed(() => (editingId.value ? '编辑待办' : '新建待办'))

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
      keyword: filters.keyword || undefined,
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
  form.categoryId = categories.value[0]?.id || 0
  form.title = ''
  form.description = ''
  form.status = 'pending'
  form.priority = 'normal'
  form.dueAt = ''
  form.images = []
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

function openEdit(row: Todo) {
  editingId.value = row.id
  form.categoryId = row.categoryId
  form.title = row.title
  form.description = row.description || ''
  form.status = row.status
  form.priority = row.priority
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
    const payload = {
      categoryId: form.categoryId,
      title: form.title.trim(),
      description: form.description,
      status: form.status,
      priority: form.priority,
      dueAt: form.dueAt || undefined,
      images: form.images,
    }
    if (editingId.value) {
      await updateTodo(editingId.value, payload)
      ElMessage.success('已保存')
    } else {
      await createTodo(payload)
      ElMessage.success('已创建')
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
  try {
    await updateTodoStatus(row.id, status)
    ElMessage.success('状态已更新')
    await loadTodos()
  } catch (e) {
    ElMessage.error((e as Error).message || '更新失败')
  }
}

async function onDelete(row: Todo) {
  try {
    await ElMessageBox.confirm(`确定删除「${row.title}」？`, '删除待办', { type: 'warning' })
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

watch(
  () => route.query.status,
  (v) => {
    if (typeof v === 'string') filters.status = v
  },
  { immediate: true },
)

onMounted(async () => {
  await loadCategories()
  await loadTodos()
})
</script>

<template>
  <div class="todo-page">
    <el-card shadow="never">
      <template #header>
        <div class="row-between">
          <div class="card-title">待办列表 <span class="count">({{ total }})</span></div>
          <el-button type="primary" @click="openCreate">新建待办</el-button>
        </div>
      </template>

      <div class="filters">
        <el-select v-model="filters.categoryId" clearable placeholder="全部分类" style="width: 140px" @change="onFilterChange">
          <el-option v-for="c in categories" :key="c.id" :label="c.name" :value="c.id" />
        </el-select>
        <el-select v-model="filters.status" style="width: 130px" @change="onFilterChange">
          <el-option v-for="o in statusOptions" :key="o.value || 'all'" :label="o.label" :value="o.value" />
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

      <el-table :data="todos" v-loading="loading" stripe border empty-text="暂无待办">
        <el-table-column prop="title" label="标题" min-width="180" show-overflow-tooltip />
        <el-table-column prop="categoryName" label="分类" width="90" />
        <el-table-column label="优先级" width="80">
          <template #default="{ row }">
            <el-tag v-if="row.priority === 'high'" type="danger" size="small">高</el-tag>
            <el-tag v-else-if="row.priority === 'low'" type="info" size="small">低</el-tag>
            <span v-else>普通</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template #default="{ row }">
            <el-select :model-value="row.status" size="small" style="width: 100px" @change="(v: string) => onStatusChange(row, v)">
              <el-option v-for="o in statusOptions.filter((x) => x.value)" :key="o.value" :label="o.label" :value="o.value" />
            </el-select>
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
        <el-table-column prop="dueAt" label="截止" width="160" show-overflow-tooltip />
        <el-table-column prop="updatedAt" label="更新" width="160" show-overflow-tooltip />
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
      <el-form label-width="88px">
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
        <el-form-item label="状态">
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
        <el-form-item label="截止时间">
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
.row-between { display: flex; justify-content: space-between; align-items: center; }
.card-title { font-weight: 600; }
.count { color: #909399; font-weight: 400; }
.filters { display: flex; gap: 8px; flex-wrap: wrap; margin-bottom: 16px; }
.pager { margin-top: 16px; display: flex; justify-content: flex-end; }
.thumbs { display: flex; gap: 4px; }
.thumb { width: 36px; height: 36px; border-radius: 4px; }
.muted { color: #909399; }
</style>
