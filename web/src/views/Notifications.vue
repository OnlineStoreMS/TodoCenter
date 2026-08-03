<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getNotification,
  resetNotificationState,
  runNotification,
  saveNotification,
  testNotification,
  type NotifyLevel,
  type NotifyState,
} from '../api/notify'

const loading = reactive({ load: false, save: false, test: false, run: false, reset: false })
const secretInput = ref('')
const state = ref<NotifyState>({ lastRunOk: false, lastSentCount: 0 })

const form = reactive({
  enabled: false,
  webhookUrl: '',
  secretSet: false,
  pollIntervalMinutes: 5,
  levels: [] as NotifyLevel[],
})

const levelMeta: Record<string, { tip: string; color: string }> = {
  warning: { tip: '距截止 ≤ 该时间进入 Warning', color: '#e6a23c' },
  critical: { tip: '距截止 ≤ 该时间进入 Critical（应 ≤ Warning）', color: '#f56c6c' },
  imminent: { tip: '距截止 ≤ 该时间进入 Imminent（应 ≤ Critical）', color: '#c45656' },
}

function formatMinutes(m: number) {
  if (m <= 0) return '—'
  if (m < 60) return `${m} 分钟`
  if (m % 60 === 0) return `${m / 60} 小时`
  const h = Math.floor(m / 60)
  return `${h} 小时 ${m % 60} 分钟`
}

function ensureLevels(levels?: NotifyLevel[]) {
  const defaults: NotifyLevel[] = [
    { key: 'warning', label: 'Warning 预警', enabled: true, beforeMinutes: 1440, repeatHours: 0 },
    { key: 'critical', label: 'Critical 紧急', enabled: true, beforeMinutes: 240, repeatHours: 0 },
    { key: 'imminent', label: 'Imminent 临期', enabled: true, beforeMinutes: 30, repeatHours: 0 },
  ]
  if (!levels?.length) return defaults
  return defaults.map((d) => {
    const hit = levels.find((x) => x.key === d.key)
    return hit ? { ...d, ...hit, label: d.label, key: d.key } : d
  })
}

async function load() {
  loading.load = true
  try {
    const data = await getNotification()
    form.enabled = data.config.enabled
    form.webhookUrl = data.config.webhookUrl
    form.secretSet = data.config.secretSet
    form.pollIntervalMinutes = data.config.pollIntervalMinutes
    form.levels = ensureLevels(data.config.levels)
    state.value = data.state || { lastRunOk: false, lastSentCount: 0 }
    secretInput.value = ''
  } catch (e) {
    ElMessage.error((e as Error).message || '加载失败')
  } finally {
    loading.load = false
  }
}

async function onSave() {
  loading.save = true
  try {
    const data = await saveNotification({
      enabled: form.enabled,
      webhookUrl: form.webhookUrl,
      pollIntervalMinutes: form.pollIntervalMinutes,
      levels: form.levels,
      secret: secretInput.value || undefined,
    })
    form.enabled = data.config.enabled
    form.webhookUrl = data.config.webhookUrl
    form.secretSet = data.config.secretSet
    form.pollIntervalMinutes = data.config.pollIntervalMinutes
    form.levels = ensureLevels(data.config.levels)
    state.value = data.state || state.value
    secretInput.value = ''
    ElMessage.success('已保存')
  } catch (e) {
    ElMessage.error((e as Error).message || '保存失败')
  } finally {
    loading.save = false
  }
}

async function onTest() {
  loading.test = true
  try {
    await testNotification('【待办中心】飞书通知测试消息')
    ElMessage.success('测试消息已发送')
  } catch (e) {
    ElMessage.error((e as Error).message || '测试失败')
  } finally {
    loading.test = false
  }
}

async function onRun() {
  loading.run = true
  try {
    const data = await runNotification()
    ElMessage.success(`扫描完成，本次推送 ${data.sent || 0} 条`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message || '扫描失败')
  } finally {
    loading.run = false
  }
}

async function onReset() {
  try {
    await ElMessageBox.confirm('清空已推送去重记录后，各等级符合条件的待办会再次推送。确定？', '重置状态', {
      type: 'warning',
    })
  } catch {
    return
  }
  loading.reset = true
  try {
    const data = await resetNotificationState()
    form.levels = ensureLevels(data.config.levels)
    state.value = data.state || state.value
    ElMessage.success('已重置')
  } catch (e) {
    ElMessage.error((e as Error).message || '重置失败')
  } finally {
    loading.reset = false
  }
}

onMounted(load)
</script>

<template>
  <div v-loading="loading.load" class="page">
    <el-card shadow="never">
      <template #header>
        <div class="row-between">
          <div>
            <div class="title">通知管理</div>
            <div class="sub">等级升级：Warning → Critical → Imminent，时间可配；仅截止前提醒，不过期催促</div>
          </div>
          <div class="actions">
            <el-button :loading="loading.test" @click="onTest">发送测试</el-button>
            <el-button :loading="loading.run" @click="onRun">立即扫描</el-button>
            <el-button :loading="loading.reset" @click="onReset">重置去重</el-button>
            <el-button type="primary" :loading="loading.save" @click="onSave">保存</el-button>
          </div>
        </div>
      </template>

      <el-form label-width="140px" style="max-width: 920px">
        <el-form-item label="启用推送">
          <el-switch v-model="form.enabled" />
        </el-form-item>
        <el-form-item label="飞书 Webhook" required>
          <el-input v-model="form.webhookUrl" placeholder="https://open.feishu.cn/open-apis/bot/v2/hook/..." />
        </el-form-item>
        <el-form-item :label="form.secretSet ? '签名密钥(已设置)' : '签名密钥'">
          <el-input v-model="secretInput" type="password" show-password placeholder="可选；留空则不修改" />
        </el-form-item>
        <el-form-item label="扫描间隔(分钟)">
          <el-input-number v-model="form.pollIntervalMinutes" :min="1" :max="1440" />
          <span class="tip">建议 ≤ 最小等级窗口（如 Imminent 30 分钟时用 5～10 分钟）</span>
        </el-form-item>
      </el-form>

      <div class="levels-head">
        <div class="levels-title">重复提醒等级</div>
        <div class="levels-sub">距截止越近等级越高；每个等级在同一截止时刻各推送一次</div>
      </div>

      <el-table :data="form.levels" border stripe class="levels-table">
        <el-table-column label="等级" width="150">
          <template #default="{ row }">
            <span class="level-name" :style="{ color: levelMeta[row.key]?.color }">{{ row.label }}</span>
          </template>
        </el-table-column>
        <el-table-column label="启用" width="80" align="center">
          <template #default="{ row }">
            <el-switch v-model="row.enabled" />
          </template>
        </el-table-column>
        <el-table-column label="时间配置" width="380">
          <template #default="{ row }">
            <div class="cfg-row">
              <span class="cfg-label">截止前</span>
              <el-input-number v-model="row.beforeMinutes" :min="1" :max="10080" :step="5" controls-position="right" />
              <span class="unit">分钟</span>
              <span class="cfg-fmt">（{{ formatMinutes(row.beforeMinutes) }}）</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="说明" min-width="220">
          <template #default="{ row }">
            <span class="tip-inline">{{ levelMeta[row.key]?.tip }}</span>
          </template>
        </el-table-column>
      </el-table>

      <div class="flow">
        示例：Warning 24h → Critical 4h → Imminent 30m
      </div>

      <el-divider />

      <div class="state">
        <div class="state-title">运行状态</div>
        <div class="state-grid">
          <div><span class="k">上次运行</span>{{ state.lastRunAt || '-' }}</div>
          <div><span class="k">结果</span>{{ state.lastRunAt ? (state.lastRunOk ? '成功' : '失败') : '-' }}</div>
          <div><span class="k">上次推送条数</span>{{ state.lastSentCount }}</div>
          <div class="err" v-if="state.lastError"><span class="k">错误</span>{{ state.lastError }}</div>
        </div>
      </div>
    </el-card>
  </div>
</template>

<style scoped>
.page { width: 100%; }
.row-between { display: flex; justify-content: space-between; align-items: flex-start; gap: 12px; flex-wrap: wrap; }
.title { font-weight: 600; font-size: 16px; }
.sub { margin-top: 4px; font-size: 12px; color: #94a3b8; }
.actions { display: flex; gap: 8px; flex-wrap: wrap; }
.tip { margin-left: 12px; font-size: 12px; color: #94a3b8; }
.tip-inline { font-size: 12px; color: #94a3b8; }
.levels-head { margin: 8px 0 12px; }
.levels-title { font-weight: 600; color: #334155; }
.levels-sub { margin-top: 4px; font-size: 12px; color: #94a3b8; }
.levels-table { width: 100%; max-width: 1100px; }
.level-name { font-weight: 600; }
.cfg-row { display: flex; align-items: center; gap: 8px; flex-wrap: nowrap; white-space: nowrap; }
.cfg-label { color: #64748b; font-size: 13px; flex-shrink: 0; }
.cfg-fmt { color: #94a3b8; font-size: 12px; flex-shrink: 0; }
.unit { color: #64748b; font-size: 13px; flex-shrink: 0; }
.flow { margin-top: 10px; font-size: 12px; color: #64748b; }
.state-title { font-weight: 600; margin-bottom: 10px; color: #334155; }
.state-grid { display: grid; gap: 8px; font-size: 13px; color: #0f172a; }
.state-grid .k { display: inline-block; min-width: 96px; color: #64748b; }
.err { color: #b91c1c; }
</style>
