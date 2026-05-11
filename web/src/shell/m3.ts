// 仅 redirect 模块用到的 Material 3 Web Components。
// 引入即注册自定义元素；Vue 模板里直接写 <md-filled-button> 等即可。
// roundnfc 仍走 Vant，不受影响。
import '@material/web/button/filled-button.js'
import '@material/web/button/outlined-button.js'
import '@material/web/button/text-button.js'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/textfield/outlined-text-field.js'
import '@material/web/textfield/filled-text-field.js'
import '@material/web/switch/switch.js'
import '@material/web/dialog/dialog.js'
import '@material/web/divider/divider.js'
import '@material/web/icon/icon.js'
import '@material/web/list/list.js'
import '@material/web/list/list-item.js'
import '@material/web/progress/circular-progress.js'
import '@material/web/chips/chip-set.js'
import '@material/web/chips/assist-chip.js'

// 用 Roboto 当 M3 typography 字体（index.html 已经加载 Google Fonts CDN）。
import { styles as typescaleStyles } from '@material/web/typography/md-typescale-styles.js'

document.adoptedStyleSheets = [
  ...(document.adoptedStyleSheets || []),
  typescaleStyles.styleSheet!,
]
