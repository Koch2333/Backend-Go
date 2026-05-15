import type { ModuleManifest } from '@/shell/types'

const manifest: ModuleManifest = {
  name: 'redirect',
  title: 'Redirect',
  description: '短链规则 / NFC 卡片注册映射',
  apiPrefix: '/api/redirect',

  blankRoutes: [{ path: 'login', component: () => import('./views/Login.vue') }],

  adminRoutes: [
    { path: '', redirect: '/m/redirect/rules' },
    { path: 'rules', component: () => import('./views/Rules.vue') },
    { path: 'cards', component: () => import('./views/Cards.vue') },
    { path: 'security', component: () => import('./views/Security.vue') },
  ],

  nav: [
    { to: '/m/redirect/rules', label: '短链规则', icon: 'link-o' },
    { to: '/m/redirect/cards', label: 'NFC 卡片', icon: 'credit-pay' },
    { to: '/m/redirect/security', label: '安全设置', icon: 'shield-o' },
  ],
}

export default manifest
