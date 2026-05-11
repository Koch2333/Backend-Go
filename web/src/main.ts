import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './shell/router'
import './style.css'
import './shell/m3'

createApp(App).use(createPinia()).use(router).mount('#app')
