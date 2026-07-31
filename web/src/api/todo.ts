import client, { unwrap, type PageData } from './client'
import type { MediaItem } from './upload'

export interface Category {
  id: number
  name: string
  code: string
  sort: number
  enabled: number
}

export interface Todo {
  id: number
  categoryId: number
  categoryName?: string
  categoryCode?: string
  title: string
  description: string
  status: string
  priority: string
  dueAt?: string
  completedAt?: string
  images: MediaItem[]
  assigneeUserId: number
  createdBy: number
  createdAt: string
  updatedAt: string
}

export interface DashboardStats {
  total: number
  pending: number
  inProgress: number
  done: number
  cancelled: number
  byCategory: Record<string, number>
}

export async function fetchDashboardStats() {
  return unwrap<DashboardStats>(await client.get('/dashboard/stats'))
}

export async function listCategories() {
  return unwrap<Category[]>(await client.get('/categories'))
}

export async function createCategory(body: { name: string; code?: string; sort?: number }) {
  return unwrap<Category>(await client.post('/categories', body))
}

export async function updateCategory(id: number, body: Partial<{ name: string; sort: number; enabled: number }>) {
  return unwrap<Category>(await client.put(`/categories/${id}`, body))
}

export async function deleteCategory(id: number) {
  return unwrap<{ ok: boolean }>(await client.delete(`/categories/${id}`))
}

export async function listTodos(params: {
  categoryId?: number
  status?: string
  priority?: string
  keyword?: string
  page?: number
  pageSize?: number
}) {
  const res = await client.get('/todos', { params })
  return unwrap<PageData<Todo> & { list: Todo[] }>(res)
}

export async function getTodo(id: number) {
  return unwrap<Todo>(await client.get(`/todos/${id}`))
}

export async function createTodo(body: Partial<Todo> & { title: string; categoryId: number }) {
  return unwrap<Todo>(await client.post('/todos', body))
}

export async function updateTodo(id: number, body: Record<string, unknown>) {
  return unwrap<Todo>(await client.put(`/todos/${id}`, body))
}

export async function updateTodoStatus(id: number, status: string) {
  return unwrap<Todo>(await client.patch(`/todos/${id}/status`, { status }))
}

export async function deleteTodo(id: number) {
  return unwrap<{ ok: boolean }>(await client.delete(`/todos/${id}`))
}
