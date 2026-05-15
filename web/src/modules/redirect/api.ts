import type { ListResult } from '@/shell/types'
import type { RegBeginOptions, LoginBeginOptions } from '@/shell/webauthn'
import { REDIRECT } from './core'
import type { NFCCard, RedirectRule } from './types'

const M = REDIRECT

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

// ----- Rules -----

export async function listRules(params: { q?: string; limit?: number; offset?: number } = {}) {
  const resp = await M.http().get('/admin/rules', { params })
  return M.unwrap<ListResult<RedirectRule>>(resp)
}

export async function upsertRule(r: { name: string; targetUrl: string; enabled: boolean }) {
  const resp = await M.http().put(`/admin/rules/${encodeURIComponent(r.name)}`, r)
  return M.unwrap<{ ok: boolean }>(resp)
}

export async function deleteRule(name: string) {
  await M.http().delete(`/admin/rules/${encodeURIComponent(name)}`)
}

// ----- NFC Cards -----

export async function listCards(params: { q?: string; limit?: number; offset?: number } = {}) {
  const resp = await M.http().get('/admin/cards', { params })
  return M.unwrap<ListResult<NFCCard>>(resp)
}

export async function upsertCard(c: { hwid: string; isRegistered: boolean; userId: string }) {
  const resp = await M.http().put(`/admin/cards/${encodeURIComponent(c.hwid)}`, c)
  return M.unwrap<{ ok: boolean }>(resp)
}

export async function deleteCard(hwid: string) {
  await M.http().delete(`/admin/cards/${encodeURIComponent(hwid)}`)
}
