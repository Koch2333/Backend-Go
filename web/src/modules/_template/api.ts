import { M } from './core'

export interface LoginResult {
  token: string
  expiresAt: string
  username: string
}

export async function login(username: string, password: string) {
  const resp = await M.http().post('/admin/login', { username, password })
  return M.unwrap<LoginResult>(resp)
}

// 按需加接口，例如：
// export async function listItems() {
//   const resp = await M.http().get('/admin/items')
//   return M.unwrap<ListResult<Item>>(resp)
// }
