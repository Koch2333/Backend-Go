import type { ListResult } from '@/shell/types'
import type { RegBeginOptions, LoginBeginOptions } from '@/shell/webauthn'
import { getApiBase } from '@/shell/backend'
import { ROUNDNFC } from './core'
import type { AutographRequest, Badge, PhotoRequest, RequestStatus } from './types'

const M = ROUNDNFC

export interface LoginResult {
  token?: string
  expiresAt?: string
  username?: string
  needsTOTP?: boolean
}

export async function login(username: string, password: string, totpCode?: string) {
  const resp = await M.http().post('/admin/login', {
    username,
    password,
    totpCode: totpCode ?? '',
  })
  return M.unwrap<LoginResult>(resp)
}

export interface PasskeyInfo {
  id: string
  name: string
  createdAt: string
}

// ----- TOTP -----

export async function getTOTPStatus() {
  const resp = await M.http().get('/admin/totp/status')
  return M.unwrap<{ enabled: boolean }>(resp)
}

export async function setupTOTP() {
  const resp = await M.http().post('/admin/totp/setup')
  return M.unwrap<{ uri: string; secret: string }>(resp)
}

export async function enableTOTP(code: string) {
  const resp = await M.http().post('/admin/totp/enable', { code })
  return M.unwrap<{ ok: boolean }>(resp)
}

export async function disableTOTP() {
  await M.http().delete('/admin/totp')
}

// ----- Passkeys -----

export async function beginPasskeyRegister() {
  const resp = await M.http().post('/admin/webauthn/register/begin')
  return M.unwrap<RegBeginOptions>(resp)
}

export async function finishPasskeyRegister(sessionId: string, name: string, credential: object) {
  const resp = await M.http().post('/admin/webauthn/register/finish', {
    sessionId,
    name,
    credential,
  })
  return M.unwrap<{ ok: boolean; id: string }>(resp)
}

export async function listPasskeys() {
  const resp = await M.http().get('/admin/webauthn/credentials')
  return M.unwrap<{ items: PasskeyInfo[] }>(resp)
}

export async function deletePasskey(id: string) {
  await M.http().delete(`/admin/webauthn/credentials/${encodeURIComponent(id)}`)
}

export async function beginPasskeyLogin(username: string) {
  const resp = await M.http().post('/admin/webauthn/login/begin', { username })
  return M.unwrap<LoginBeginOptions>(resp)
}

export async function finishPasskeyLogin(sessionId: string, credential: object) {
  const resp = await M.http().post('/admin/webauthn/login/finish', { sessionId, credential })
  return M.unwrap<LoginResult>(resp)
}

// ----- Badges -----

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

export interface BadgeStyleTemplate {
  key: string
  label: string
  description?: string
  imageUrl?: string
  imageOriginalUrl?: string
  imagePreviewUrl?: string
  payload?: unknown
  enabled?: boolean
}

export async function listBadgeStyles() {
  const resp = await M.http().get('/admin/style-templates')
  return M.unwrap<{ items: BadgeStyleTemplate[] }>(resp)
}

export async function saveBadgeStyleTemplate(t: BadgeStyleTemplate) {
  const body = {
    key: t.key,
    label: t.label,
    description: t.description ?? '',
    imageUrl: t.imageUrl ?? '',
    payload: t.payload ?? {},
    enabled: t.enabled ?? true,
  }
  const resp = await M.http().put(`/admin/style-templates/${encodeURIComponent(t.key)}`, body)
  return M.unwrap<BadgeStyleTemplate>(resp)
}

export async function createBadgeStyleTemplate(t: BadgeStyleTemplate) {
  const body = {
    key: t.key,
    label: t.label,
    description: t.description ?? '',
    imageUrl: t.imageUrl ?? '',
    payload: t.payload ?? {},
    enabled: t.enabled ?? true,
  }
  const resp = await M.http().post('/admin/style-templates', body)
  return M.unwrap<BadgeStyleTemplate>(resp)
}

export async function deleteBadgeStyleTemplate(key: string) {
  await M.http().delete(`/admin/style-templates/${encodeURIComponent(key)}`)
}

export async function uploadBadgeStyleTemplateImage(key: string, file: File) {
  const fd = new FormData()
  fd.append('file', file)
  const resp = await M.http().post(`/admin/style-templates/${encodeURIComponent(key)}/image`, fd, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
  return M.unwrap<{ key: string; item: BadgeStyleTemplate }>(resp)
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

// ----- Android App Pairing -----

export interface AppToken {
  id: string
  name: string
  tokenPrefix: string
  enabled: boolean
  lastUsedAt?: string
  createdAt: string
  updatedAt: string
}

export interface AppPairingConfig {
  protocol: 'roundnfc-writer'
  version: number
  name: string
  apiBase: string
  apiPrefix: string
  tokenHeader: string
  token: string
  endpoints: Record<string, string>
  createdAt: string
}

export async function listAppTokens() {
  const resp = await M.http().get('/admin/app-tokens')
  return M.unwrap<{ items: AppToken[] }>(resp)
}

export async function createAppToken(name: string) {
  const resp = await M.http().post('/admin/app-tokens', {
    name,
    apiBase: getApiBase('roundnfc') || window.location.origin,
  })
  return M.unwrap<{ item: AppToken; token: string; pairing: AppPairingConfig }>(resp)
}

export async function setAppTokenEnabled(id: string, enabled: boolean) {
  await M.http().patch(`/admin/app-tokens/${encodeURIComponent(id)}`, { enabled })
}

export async function deleteAppToken(id: string) {
  await M.http().delete(`/admin/app-tokens/${encodeURIComponent(id)}`)
}
