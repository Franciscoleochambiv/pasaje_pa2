import { chromium } from 'playwright'
import fs from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath, pathToFileURL } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const configPath = process.argv[2]
  ? path.resolve(process.argv[2])
  : path.join(__dirname, 'targets.json')

const config = JSON.parse(await fs.readFile(configPath, 'utf8'))
const outDir = path.resolve(__dirname, config.outputDir || '../../evidencias_anexos/capturas_web')
await fs.mkdir(outDir, { recursive: true })

function stampText(title) {
  const now = new Date()
  const date = new Intl.DateTimeFormat('es-PE', {
    timeZone: 'America/Lima',
    dateStyle: 'full',
    timeStyle: 'medium'
  }).format(now)
  return `${title} | ${date} | America/Lima`
}

function resolveUrl(rawUrl) {
  if (/^https?:\/\//i.test(rawUrl)) return rawUrl
  return pathToFileURL(path.resolve(__dirname, rawUrl)).href
}

async function injectEvidenceBanner(page, title) {
  const text = stampText(title)
  await page.addStyleTag({
    content: `
      html { scroll-padding-top: 96px !important; }
      body { padding-top: 96px !important; }
      #codex-evidence-banner {
        position: fixed !important;
        top: 0 !important;
        left: 0 !important;
        right: 0 !important;
        z-index: 2147483647 !important;
        min-height: 76px !important;
        box-sizing: border-box !important;
        padding: 14px 22px !important;
        background: #fff200 !important;
        color: #111 !important;
        border-bottom: 6px solid #111 !important;
        font-family: Arial, Helvetica, sans-serif !important;
        font-size: 28px !important;
        line-height: 1.18 !important;
        font-weight: 800 !important;
        letter-spacing: 0 !important;
        text-align: left !important;
        box-shadow: 0 4px 16px rgba(0,0,0,.35) !important;
        white-space: normal !important;
      }
      @media (max-width: 700px) {
        #codex-evidence-banner { font-size: 18px !important; min-height: 68px !important; }
        body { padding-top: 84px !important; }
      }
    `
  })
  await page.evaluate((bannerText) => {
    const old = document.getElementById('codex-evidence-banner')
    if (old) old.remove()
    const div = document.createElement('div')
    div.id = 'codex-evidence-banner'
    div.textContent = bannerText
    document.documentElement.appendChild(div)
  }, text)
}

async function runActions(page, actions = []) {
  for (const action of actions) {
    if (action.type === 'fill') {
      await page.locator(action.selector).fill(action.value ?? '')
    } else if (action.type === 'click') {
      await page.locator(action.selector).click()
    } else if (action.type === 'wait') {
      await page.waitForTimeout(action.ms ?? 1000)
    } else if (action.type === 'press') {
      await page.locator(action.selector).press(action.key)
    } else {
      throw new Error(`Accion no soportada: ${JSON.stringify(action)}`)
    }
  }
}

const browser = await chromium.launch({ headless: true })
const context = await browser.newContext({
  viewport: config.viewport || { width: 1440, height: 950 },
  ignoreHTTPSErrors: false
})

const results = []

for (const target of config.targets) {
  const page = await context.newPage()
  const url = resolveUrl(target.url)
  const started = new Date()
  let status = 'ok'
  let detail = ''

  try {
    const response = await page.goto(url, { waitUntil: 'domcontentloaded', timeout: target.timeoutMs || 45000 })
    await page.waitForTimeout(target.waitMs || 2500)
    await runActions(page, target.actions)
    await injectEvidenceBanner(page, target.title || target.name)
    await page.waitForTimeout(500)

    const fileName = `${target.name}.png`
    const fullPath = path.join(outDir, fileName)
    await page.screenshot({ path: fullPath, fullPage: target.fullPage ?? false })
    status = response ? `${response.status()} ${response.statusText()}` : 'sin respuesta HTTP'
    detail = fullPath
  } catch (error) {
    status = 'error'
    detail = error?.message || String(error)

    await page.setContent(`
      <!doctype html>
      <meta charset="utf-8">
      <title>Error de captura</title>
      <body style="font-family: Arial; padding: 120px 32px 32px;">
        <h1>No se pudo capturar: ${target.name}</h1>
        <p><strong>URL:</strong> ${url}</p>
        <pre style="white-space: pre-wrap; background:#111; color:#fff; padding:16px;">${detail}</pre>
      </body>
    `)
    await injectEvidenceBanner(page, `ERROR - ${target.title || target.name}`)
    const fullPath = path.join(outDir, `${target.name}_ERROR.png`)
    await page.screenshot({ path: fullPath, fullPage: false })
    detail = `${detail} | captura_error=${fullPath}`
  } finally {
    await page.close()
  }

  results.push({
    name: target.name,
    title: target.title || target.name,
    url,
    status,
    detail,
    started_at: started.toISOString()
  })
}

await browser.close()

const reportPath = path.join(outDir, 'capturas_web_manifest.json')
await fs.writeFile(reportPath, JSON.stringify(results, null, 2), 'utf8')

console.log(`Capturas guardadas en: ${outDir}`)
for (const result of results) {
  console.log(`${result.name}: ${result.status} -> ${result.detail}`)
}
console.log(`Manifest: ${reportPath}`)
