import axios from 'axios'
import client, { unwrap } from './client'

export type MediaType = 'image' | 'video'

export interface MediaItem {
  url: string
  mediaType: MediaType
}

function guessMediaType(file: File, url?: string): MediaType {
  if (file.type.startsWith('video/')) return 'video'
  if (file.type.startsWith('image/')) return 'image'
  const name = (file.name || url || '').toLowerCase()
  if (/\.(mp4|mov|webm|m4v|avi|mkv)(\?|$)/i.test(name)) return 'video'
  return 'image'
}

export async function uploadImage(file: File, subdir = 'stores'): Promise<string> {
  const item = await uploadMedia(file, subdir)
  return item.url
}

export async function uploadMedia(file: File, subdir = 'stores'): Promise<MediaItem> {
  const form = new FormData()
  form.append('file', file)
  form.append('subdir', subdir)
  const res = await client.post('/upload', form, {
    headers: { 'Content-Type': 'multipart/form-data' },
    timeout: 120000,
  })
  const data = unwrap<{ url: string; mediaType?: string }>(res)
  return {
    url: data.url,
    mediaType: data.mediaType === 'video' ? 'video' : guessMediaType(file, data.url),
  }
}

export interface PhotoUploadSession {
  token: string
  status: 'pending' | 'done'
  url?: string
  mediaType?: MediaType
  accept?: 'image' | 'media'
  expireAt: string
}

export async function createPhotoUploadSession(
  subdir = 'payments/service',
  accept: 'image' | 'media' = 'image',
): Promise<PhotoUploadSession> {
  const res = await client.post('/photo-upload-sessions', { subdir, accept })
  return unwrap<PhotoUploadSession>(res)
}

export async function getPhotoUploadSession(token: string): Promise<PhotoUploadSession> {
  const res = await client.get(`/photo-upload-sessions/${token}`)
  return unwrap<PhotoUploadSession>(res)
}

/** 手机端免登录查询/上传 */
const mobileClient = axios.create({
  baseURL: '/api/v1/mobile',
  timeout: 120000,
})

function unwrapMobile<T>(res: { data: { code: number; message: string; data?: T } }): T {
  if (res.data.code !== 200) {
    throw new Error(res.data.message || '请求失败')
  }
  return res.data.data as T
}

export async function mobileGetPhotoSession(token: string): Promise<PhotoUploadSession> {
  const res = await mobileClient.get(`/photo-upload/${token}`)
  return unwrapMobile<PhotoUploadSession>(res)
}

export async function mobileUploadPhoto(
  token: string,
  file: File,
): Promise<{ url: string; status: string; mediaType?: string }> {
  const form = new FormData()
  form.append('file', file)
  const res = await mobileClient.post(`/photo-upload/${token}`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return unwrapMobile(res)
}
