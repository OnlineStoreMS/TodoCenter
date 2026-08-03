import client, { unwrap } from './client'

export interface NotifyLevel {
  key: 'warning' | 'critical' | 'imminent' | string
  label: string
  enabled: boolean
  beforeMinutes: number
  repeatHours?: number
}

export interface NotifyConfig {
  enabled: boolean
  webhookUrl: string
  secretSet: boolean
  pollIntervalMinutes: number
  levels: NotifyLevel[]
}

export interface NotifyState {
  lastRunAt?: string
  lastRunOk: boolean
  lastError?: string
  lastSentCount: number
}

export async function getNotification() {
  return unwrap<{ config: NotifyConfig; state: NotifyState }>(await client.get('/notifications'))
}

export async function saveNotification(body: Partial<NotifyConfig> & { secret?: string }) {
  return unwrap<{ config: NotifyConfig; state: NotifyState }>(await client.put('/notifications', body))
}

export async function testNotification(text?: string) {
  return unwrap<{ ok: boolean }>(await client.post('/notifications/test', { text }))
}

export async function runNotification() {
  return unwrap<{ sent: number }>(await client.post('/notifications/run'))
}

export async function resetNotificationState() {
  return unwrap<{ config: NotifyConfig; state: NotifyState }>(await client.post('/notifications/reset-state'))
}
