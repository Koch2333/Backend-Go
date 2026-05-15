import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './shell/router'
import { consumeApiQueryParam } from './shell/backend'
import './style.css'
import './shell/m3'

consumeApiQueryParam()

createApp(App).use(createPinia()).use(router).mount('#app')
