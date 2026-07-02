import base from './playwright.config'

// Local visual run: point at the dev server started manually on :3100 (port
// 3000 is occupied by another app), and do not let Playwright manage a server.
export default {
  ...base,
  webServer: undefined,
  use: {
    ...base.use,
    baseURL: 'http://localhost:3100',
    headless: false,
    launchOptions: { slowMo: 700 },
  },
}
