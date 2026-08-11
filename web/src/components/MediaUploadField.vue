<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import { ElMessage, type UploadRequestOptions } from 'element-plus'
import { Iphone, Plus, Delete, VideoCamera } from '@element-plus/icons-vue'
import QRCode from 'qrcode'
import {
  createPhotoUploadSession,
  getPhotoUploadSession,
  uploadMedia,
  type MediaItem,
} from '../api/upload'

const props = withDefaults(
  defineProps<{
    modelValue?: MediaItem[]
    subdir?: string
    maxCount?: number
  }>(),
  {
    modelValue: () => [],
    subdir: 'todos',
    maxCount: 12,
  },
)

const emit = defineEmits<{
  (e: 'update:modelValue', v: MediaItem[]): void
}>()

const uploading = ref(false)
const scanVisible = ref(false)
const scanLoading = ref(false)
const qrDataUrl = ref('')
const scanToken = ref('')
const scanStatus = ref<'idle' | 'waiting' | 'done' | 'expired'>('idle')
let pollTimer: ReturnType<typeof setInterval> | null = null

function stopPoll() {
  if (pollTimer) {
    clearInterval(pollTimer)
    pollTimer = null
  }
}

function pushMedia(item: MediaItem) {
  const list = [...(props.modelValue || [])]
  if (list.some((m) => m.url === item.url)) return
  if (list.length >= props.maxCount) {
    ElMessage.warning(`最多上传 ${props.maxCount} 个文件`)
    return
  }
  list.push(item)
  emit('update:modelValue', list)
}

function removeAt(index: number) {
  const list = [...(props.modelValue || [])]
  list.splice(index, 1)
  emit('update:modelValue', list)
}

async function doUpload(opt: UploadRequestOptions) {
  uploading.value = true
  try {
    const file = opt.file as File
    const item = await uploadMedia(file, props.subdir)
    pushMedia(item)
    ElMessage.success(item.mediaType === 'video' ? '视频已上传' : '图片已上传')
  } catch (e) {
    ElMessage.error((e as Error).message || '上传失败')
  } finally {
    uploading.value = false
  }
}

async function openScan() {
  if ((props.modelValue?.length || 0) >= props.maxCount) {
    ElMessage.warning(`最多上传 ${props.maxCount} 个文件`)
    return
  }
  scanVisible.value = true
  scanLoading.value = true
  scanStatus.value = 'idle'
  qrDataUrl.value = ''
  scanToken.value = ''
  stopPoll()
  try {
    const session = await createPhotoUploadSession(props.subdir, 'media')
    scanToken.value = session.token
    const pageUrl = `${window.location.origin}${import.meta.env.BASE_URL || '/'}m/photo-upload?token=${encodeURIComponent(session.token)}&mode=media`
    qrDataUrl.value = await QRCode.toDataURL(pageUrl, {
      width: 220,
      margin: 2,
      errorCorrectionLevel: 'M',
    })
    scanStatus.value = 'waiting'
    pollTimer = setInterval(async () => {
      try {
        const s = await getPhotoUploadSession(scanToken.value)
        if (s.status === 'done' && s.url) {
          pushMedia({
            url: s.url,
            mediaType: s.mediaType === 'video' ? 'video' : 'image',
          })
          scanStatus.value = 'done'
          stopPoll()
          ElMessage.success('手机端上传成功')
          setTimeout(() => {
            scanVisible.value = false
          }, 400)
        }
      } catch {
        scanStatus.value = 'expired'
        stopPoll()
      }
    }, 2000)
  } catch (e) {
    ElMessage.error((e as Error).message || '创建扫码会话失败')
    scanVisible.value = false
  } finally {
    scanLoading.value = false
  }
}

function closeScan() {
  scanVisible.value = false
  stopPoll()
}

onUnmounted(stopPoll)
</script>

<template>
  <div class="media-field">
    <div class="media-row">
      <div v-for="(m, i) in modelValue" :key="m.url + i" class="media-item">
        <el-image v-if="m.mediaType !== 'video'" :src="m.url" fit="cover" class="thumb" :preview-src-list="[m.url]" />
        <a v-else :href="m.url" target="_blank" class="thumb video">
          <el-icon :size="28"><VideoCamera /></el-icon>
          <span>视频</span>
        </a>
        <el-button class="rm" circle size="small" type="danger" :icon="Delete" @click.stop="removeAt(i)" />
      </div>
      <el-upload
        v-if="(modelValue?.length || 0) < maxCount"
        class="media-upload"
        :show-file-list="false"
        accept="image/*,video/*"
        :disabled="uploading"
        :http-request="doUpload"
      >
        <div class="media-item add">
          <el-icon><Plus /></el-icon>
          <span>{{ uploading ? '上传中…' : '本机上传' }}</span>
        </div>
      </el-upload>
    </div>
    <div class="actions">
      <el-button type="primary" plain :icon="Iphone" :disabled="uploading" @click="openScan">
        手机扫码上传
      </el-button>
      <span class="hint">支持图片与视频；扫码上传后自动回填</span>
    </div>
  </div>

  <el-dialog
    v-model="scanVisible"
    title="手机扫码上传图片/视频"
    width="360px"
    append-to-body
    destroy-on-close
    @closed="closeScan"
  >
    <div class="scan-body" v-loading="scanLoading">
      <img v-if="qrDataUrl" :src="qrDataUrl" alt="扫码上传" class="qr" />
      <p v-if="scanStatus === 'waiting'" class="scan-hint">请用手机扫描二维码，拍照或从相册选择图片/视频</p>
      <p v-else-if="scanStatus === 'done'" class="scan-hint ok">上传成功</p>
      <p v-else-if="scanStatus === 'expired'" class="scan-hint err">会话已过期，请关闭后重试</p>
      <p v-else class="scan-hint">正在生成二维码…</p>
    </div>
    <template #footer>
      <el-button @click="closeScan">关闭</el-button>
      <el-button type="primary" :disabled="scanLoading" @click="openScan">刷新二维码</el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.media-field { width: 100%; max-width: 100%; }
.media-row {
  display: flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  gap: 10px;
  overflow-x: auto;
  padding-bottom: 4px;
  width: 100%;
}
.media-item {
  position: relative;
  flex: 0 0 auto;
  width: 96px;
  height: 96px;
  border-radius: 8px;
  border: 1px dashed #dcdfe6;
  overflow: hidden;
  background: #fafafa;
}
.media-upload {
  display: inline-flex;
  flex: 0 0 auto;
  width: 96px;
  height: 96px;
}
.media-upload :deep(.el-upload) {
  display: block;
  width: 96px;
  height: 96px;
}
.media-item .thumb {
  width: 96px !important;
  height: 96px !important;
  display: block;
}
.media-item .thumb :deep(img) {
  width: 96px;
  height: 96px;
  object-fit: cover;
}
a.thumb.video {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: #409eff;
  text-decoration: none;
  font-size: 12px;
  width: 96px;
  height: 96px;
}
.media-item.add {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: #909399;
  font-size: 12px;
  cursor: pointer;
  width: 96px;
  height: 96px;
  box-sizing: border-box;
}
.rm { position: absolute; top: 4px; right: 4px; z-index: 1; }
.actions { margin-top: 10px; display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.hint { font-size: 12px; color: #909399; }
.scan-body {
  display: flex; flex-direction: column; align-items: center; min-height: 260px; justify-content: center;
}
.qr { width: 220px; height: 220px; border: 1px solid #ebeef5; border-radius: 8px; }
.scan-hint { margin: 12px 0 0; font-size: 13px; color: #606266; text-align: center; line-height: 1.5; }
.scan-hint.ok { color: #67c23a; }
.scan-hint.err { color: #f56c6c; }
</style>
