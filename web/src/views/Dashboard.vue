<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { List, FolderOpened, CircleCheck, Clock } from '@element-plus/icons-vue'
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

onMounted(load)
</script>

<template>
  <div v-loading="loading" class="dashboard">
    <h2 class="page-title">待办中心</h2>
    <p class="desc">统一记录电商、发货、售后、门店等业务待办，支持图片笔记与手机扫码上传。</p>

    <div class="stat-row">
      <el-card shadow="never" class="stat"><div class="n">{{ stats.total }}</div><div class="l">全部</div></el-card>
      <el-card shadow="never" class="stat warn"><div class="n">{{ stats.pending }}</div><div class="l">待处理</div></el-card>
      <el-card shadow="never" class="stat"><div class="n">{{ stats.inProgress }}</div><div class="l">进行中</div></el-card>
      <el-card shadow="never" class="stat ok"><div class="n">{{ stats.done }}</div><div class="l">已完成</div></el-card>
    </div>

    <div class="card-grid">
      <el-card shadow="hover" class="action-card" @click="router.push('/todos')">
        <el-icon :size="32" color="#409eff"><List /></el-icon>
        <h3>待办列表</h3>
        <p>新建、筛选与跟进待办</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push('/todos?status=pending')">
        <el-icon :size="32" color="#e6a23c"><Clock /></el-icon>
        <h3>待处理</h3>
        <p>{{ stats.pending }} 条待跟进</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push('/todos?status=done')">
        <el-icon :size="32" color="#67c23a"><CircleCheck /></el-icon>
        <h3>已完成</h3>
        <p>{{ stats.done }} 条已完成</p>
      </el-card>
      <el-card shadow="hover" class="action-card" @click="router.push('/categories')">
        <el-icon :size="32" color="#909399"><FolderOpened /></el-icon>
        <h3>分类管理</h3>
        <p>电商 / 发货 / 售后 / 门店</p>
      </el-card>
    </div>

    <el-card shadow="never" class="by-cat">
      <template #header>分类待办数</template>
      <div class="cat-list">
        <div v-for="(cnt, code) in stats.byCategory" :key="code" class="cat-item">
          <span>{{ catNameByCode[code] || code }}</span>
          <strong>{{ cnt }}</strong>
        </div>
        <div v-if="!Object.keys(stats.byCategory || {}).length" class="muted">暂无数据</div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.page-title { margin: 0 0 8px; font-size: 20px; }
.desc { color: #909399; margin: 0 0 20px; }
.stat-row { display: flex; gap: 12px; flex-wrap: wrap; margin-bottom: 20px; }
.stat { width: 140px; text-align: center; }
.stat .n { font-size: 28px; font-weight: 600; color: #303133; }
.stat.warn .n { color: #e6a23c; }
.stat.ok .n { color: #67c23a; }
.stat .l { color: #909399; font-size: 13px; margin-top: 4px; }
.card-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 16px; margin-bottom: 20px; }
.action-card { cursor: pointer; text-align: center; padding: 8px 0; }
.action-card h3 { margin: 12px 0 6px; font-size: 16px; }
.action-card p { margin: 0; color: #909399; font-size: 13px; }
.by-cat { max-width: 480px; }
.cat-list { display: flex; flex-direction: column; gap: 8px; }
.cat-item { display: flex; justify-content: space-between; }
.muted { color: #909399; }
</style>
