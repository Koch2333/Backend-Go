import type { ListResult } from '@/shell/types'
import { ROUNDNFC } from './core'
import type { AutographRequest, Badge, PhotoRequest, RequestStatus } from './types'

const M = ROUNDNFC

export interface LoginResult {
  token: string
  expiresAt: string
  username: string
}

export async function login(username: string, password: string) {
  const resp = await M.http().post('/admin/login', { username, password })
  return M.unwrap<LoginResult>(resp)
}

export async function listBadges(params: { q?: string; limit?: number; offset?: number } = {}) {
  const resp = await M.http().get('/admin/badges', { params })
  return M.unwrap<ListResult<Badge>>(resp)
}

export async function getBadge(id: string) {
  const resp = await M.http().get(`/admin/badges/${encodeURIComponent(id)}`)
  return M.unwrap<Badge>(resp)
}

export async function createBadge(b: Partial<Badge> & { id: string }) {
  const resp = await M.http().post('/admin/badges', b)
  return M.unwrap<Badge>(resp)
}

export async function upsertBadge(b: Partial<Badge> & { id: string }) {
  const resp = await M.http().put(`/admin/badges/${encodeURIComponent(b.id)}`, b)
  return M.unwrap<Badge>(resp)
}

export async function deleteBadge(id: string) {
  await M.http().delete(`/admin/badges/${encodeURIComponent(id)}`)
}

export async function uploadBadgeImage(id: string, file: File) {
  const fd = new FormData()
  fd.append('file', file)
  const resp = await M.http().post(`/admin/badges/${encodeURIComponent(id)}/image`, fd, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return M.unwrap<{ key: string }>(resp)
}

export interface RequestListParams {
  badgeId?: string
  status?: RequestStatus
  limit?: number
  offset?: number
}

export async function listPhotoRequests(params: RequestListParams = {}) {
  const resp = await M.http().get('/admin/photo-requests', { params })
  return M.unwrap<ListResult<PhotoRequest>>(resp)
}
export async function setPhotoStatus(id: string, status: RequestStatus) {
  await M.http().patch(`/admin/photo-requests/${encodeURIComponent(id)}`, { status })
}

export async function listAutographRequests(params: RequestListParams = {}) {
  const resp = await M.http().get('/admin/autograph-requests', { params })
  return M.unwrap<ListResult<AutographRequest>>(resp)
}
export async function setAutographStatus(id: string, status: RequestStatus) {
  await M.http().patch(`/admin/autograph-requests/${encodeURIComponent(id)}`, { status })
}
