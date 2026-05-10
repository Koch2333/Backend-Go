import type { ModuleManifest } from '@/shell/types'

const manifest: ModuleManifest = {
  name: 'roundnfc',
  title: 'RoundNFC',
  description: '漫展徽章 NFC 源 / 返图申请 / To 签申请',
  apiPrefix: '/api/roundnfc',

  blankRoutes: [{ path: 'login', component: () => import('./views/Login.vue') }],

  adminRoutes: [
    { path: '', redirect: '/m/roundnfc/badges' },
    { path: 'badges', component: () => import('./views/Badges.vue') },
    {
      path: 'badges/:id',
      component: () => import('./views/BadgeEdit.vue'),
      props: true,
    },
    { path: 'photo-requests', component: () => import('./views/PhotoRequests.vue') },
    {
      path: 'autograph-requests',
      component: () => import('./views/AutographRequests.vue'),
    },
    { path: 'security', component: () => import('./views/Security.vue') },
  ],

  nav: [
    { to: '/m/roundnfc/badges', label: '徽章', icon: 'medal-o' },
    { to: '/m/roundnfc/photo-requests', label: '返图申请', icon: 'photo-o' },
    { to: '/m/roundnfc/autograph-requests', label: 'To 签', icon: 'edit' },
    { to: '/m/roundnfc/security', label: '安全设置', icon: 'shield-o' },
  ],
}

export default manifest
