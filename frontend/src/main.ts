import '@fontsource/geist/400.css'
import '@fontsource/geist/500.css'
import '@fontsource/geist/600.css'
import '@fontsource/geist-mono/400.css'
import '@fontsource/geist-mono/500.css'
import '@fontsource/geist-mono/600.css'
import '@xterm/xterm/css/xterm.css'
import './themes/fleet-dark.css'
import './themes/flatwhite.css'
import './themes/catppuccin-mocha.css'
import './themes/catppuccin-latte.css'
import './themes/base.css'
import { createApp } from 'vue'
import App from './App.vue'
import { derby } from './lib/state'

void derby.init()
createApp(App).mount('#app')
