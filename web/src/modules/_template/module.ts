import type { ModuleManifest } from '@/shell/types'

// 这个 manifest 会被 Shell 自动发现。换掉字段后你的模块就出现在后台了。
// 注意：如果你保留 name='_template'，Shell 会从运行时过滤掉这个模块。
const manifest: ModuleManifest = {
  name: '_template',
  title: 'Template',
  description: '复制本模板作为新模块的起手点。',
  apiPrefix: '/api/_template',

  blankRoutes: [
    { path: 'login', component: () => import('./views/Login.vue') },
  ],

  adminRoutes: [
    { path: '', component: () => import('./views/Index.vue') },
  ],

  nav: [
    // { to: '/m/_template', label: '首页' },
  ],
}

export default manifest
