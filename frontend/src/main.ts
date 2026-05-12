import './app.css'
import App from './App.svelte'
import { mount } from 'svelte'

// Hostname-pinned landing page: on the feedback subdomain the page is
// always /ch14-live regardless of the URL visited. Set the hash before
// mount so the router boots straight into the target route (no Dashboard
// flash). Also pre-seed loginReturn so the auth flow lands here too.
const FEEDBACK_HOST = 'feedback.tec14.co.il'
const FEEDBACK_ROUTE = '/ch14-live'
if (window.location.hostname === FEEDBACK_HOST) {
  const hash = window.location.hash.replace(/^#/, '')
  if (hash !== FEEDBACK_ROUTE && hash !== '/login') {
    window.location.hash = '#' + FEEDBACK_ROUTE
  }
  sessionStorage.setItem('loginReturn', FEEDBACK_ROUTE)
}

const app = mount(App, { target: document.getElementById('app')! })

export default app
