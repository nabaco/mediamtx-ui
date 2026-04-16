import { addMessages, init, getLocaleFromNavigator } from 'svelte-i18n'
import en from './en.json'
import he from './he.json'

addMessages('en', en)
addMessages('he', he)

init({
  fallbackLocale: 'en',
  initialLocale: localStorage.getItem('lang') ?? getLocaleFromNavigator() ?? 'en',
})

export { _ } from 'svelte-i18n'
