// Material 3 Web Components — registers <md-*> custom elements.
import '@material/web/button/filled-button.js'
import '@material/web/button/outlined-button.js'
import '@material/web/button/text-button.js'
import '@material/web/iconbutton/icon-button.js'
import '@material/web/iconbutton/filled-icon-button.js'
import '@material/web/iconbutton/filled-tonal-icon-button.js'
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
import '@material/web/fab/fab.js'
import '@material/web/ripple/ripple.js'

// 用 Roboto 当 M3 typography 字体（index.html 已经加载 Google Fonts CDN）。
import { styles as typescaleStyles } from '@material/web/typography/md-typescale-styles.js'

document.adoptedStyleSheets = [
  ...(document.adoptedStyleSheets || []),
  typescaleStyles.styleSheet!,
]
