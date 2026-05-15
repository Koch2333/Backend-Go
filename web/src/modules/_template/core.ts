import { defineModule } from '@/shell/defineModule'

// 换成你的后端模块名 / API 前缀。整个模块中的运行时从这里走。
export const M = defineModule({
  name: '_template',
  apiPrefix: '/api/_template',
})
