/**
 * 徽章内置样式列表（后台下拉框用）。
 *
 * key 必须和公开端 RoundNFC 仓库的 src/config/styles.ts 保持一致：
 * 后台这里选的 key 存进 badge.styleKey，公开端据此渲染对应底色。
 * 新增 / 修改样式时，两个仓库都要同步改。
 */
export interface BadgeStyleOption {
  key: string
  label: string
}

export const BADGE_STYLE_OPTIONS: BadgeStyleOption[] = [
  { key: 'sakura', label: '樱花粉' },
  { key: 'mint', label: '薄荷绿' },
  { key: 'sky', label: '天空蓝' },
  { key: 'lavender', label: '薰衣草紫' },
  { key: 'gold', label: '香槟金' },
  { key: 'night', label: '暗夜星' },
]
