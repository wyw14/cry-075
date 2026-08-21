import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import { router } from './router'
import './styles.css'
import './environment.css'

const app = createApp(App)
const state = createPinia()

app.use(state)
app.use(router)

router.isReady().then(() => app.mount('#app'))
