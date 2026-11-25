import assert from 'node:assert/strict'
import { execFileSync, spawn } from 'node:child_process'
import { once } from 'node:events'
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises'
import { createServer } from 'node:net'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import { setTimeout as delay } from 'node:timers/promises'
import test from 'node:test'
import { fileURLToPath } from 'node:url'
import { chromium } from 'playwright-core'

const root = fileURLToPath(new URL('../../', import.meta.url))

async function availablePort() {
  const listener = createServer()
  listener.listen(0, '127.0.0.1')
  await once(listener, 'listening')
  const { port } = listener.address()
  await new Promise(resolve => listener.close(resolve))
  return port
}

test('folder selection, project creation, branch settings and persisted tasks', async t => {
  const dir = await mkdtemp(join(tmpdir(), 'aifactory-ui-'))
  let browser, server, exited
  try {
    const port = await availablePort()
    const url = 'http://127.0.0.1:' + port
    await mkdir(join(dir, '.ai-factory'))
    await writeFile(join(dir, '.ai-factory/config.yaml'), 'listen_address: 127.0.0.1:' + port + '\n')
    const picker = process.platform === 'darwin' ? 'osascript' : 'zenity'
    const mode = join(dir, 'picker-mode')
    await writeFile(join(dir, picker), [
      '#!/bin/sh',
      'case "$(cat "$PICKER_MODE")" in',
      '  cancel) echo "Cancelado (-128)" >&2; exit 1;;',
      '  fail) exit 69;;',
      '  select) echo "$PICKER_PATH";;',
      'esac',
    ].join('\n'), { mode: 0o755 })
    const binary = join(dir, 'aifactory')
    execFileSync('go', ['build', '-o', binary, '.'], { cwd: root })
    server = spawn(binary, ['serve', '--no-browser'], {
      env: { ...process.env, HOME: dir, PATH: dir + ':' + process.env.PATH, PICKER_MODE: mode, PICKER_PATH: dir },
      stdio: ['ignore','ignore','inherit'],
    })
    exited = once(server, 'exit')
    for (let i = 0; i < 100; i++) {
      if (server.exitCode !== null) throw Error('Server failed to start')
      if ((await fetch(url + '/api/health').catch(() => null))?.ok) break
      await delay(50)
    }
    await t.test('HTTP API with the native picker executable simulated', async () => {
      async function request(path, payload) {
        return fetch(url + path, payload && {
          method: 'POST', headers: {'Content-Type':'application/json'}, body:JSON.stringify(payload),
        })
      }
      for (const [state, status, expected] of [
        ['select', 200, {path:dir}], ['cancel', 200, {cancelled:true}], ['fail', 503, null],
      ]) {
        await writeFile(mode, state)
        const response = await request('/api/folder-selection', {operation:'open'})
        assert.equal(response.status, status)
        const result = await response.json()
        if (expected) assert.deepEqual(result, expected)
        else if (process.platform === 'darwin') assert.match(result.error, /macOS/)
      }
      const result = await request('/api/projects', {operation:'create',name:'api-demo',parent_path:dir})
      assert.equal(result.status, 201)
      const project = await result.json()
      execFileSync('git', ['-C',project.path,'-c','user.name=Test','-c','user.email=test@example.org','commit','--allow-empty','-m','initial'])
      const tasksPath = '/api/projects/' + project.id + '/tasks'
      const created = await request(tasksPath, {title:'API test'})
      assert.equal(created.status, 201)
      const task = await created.json()
      assert.equal(task.base_branch, project.base_branch)
      assert.deepEqual(await (await request(tasksPath)).json(), [task])
    })
    browser = await chromium.launch({ executablePath: process.env.CHROME_BIN })
    const page = await browser.newPage()
    page.setDefaultTimeout(8000)
    const errors = []
    page.on('pageerror', error => errors.push(error.message))
    await page.goto(url)
    const button = name => page.getByRole('button', { name, exact: true })
    const existing = page.getByPlaceholder('Selecciona o pega la ruta de proyecto')
    await existing.fill('/previa')
    await writeFile(mode, 'cancel')
    const cancelled = page.waitForResponse(response => response.url().endsWith('/api/folder-selection'))
    await button('Seleccionar…').click()
    assert.deepEqual(await (await cancelled).json(), { cancelled: true })
    assert.equal(await existing.inputValue(), '/previa')
    await writeFile(mode, 'fail')
    await button('Seleccionar…').click()
    await page.getByRole('alert').filter({hasText:'Puedes pegar la ruta'}).waitFor()
    await button('＋ Nuevo proyecto').click()
    await writeFile(mode, 'select')
    await page.locator('.form-card').getByRole('button', {name:'Seleccionar…'}).click()
    await page.getByLabel('Nombre del proyecto').fill('demo')
    await page.getByText(join(dir, 'demo'), {exact:true}).waitFor()
    await button('Crear proyecto').click()
    await page.getByRole('heading', {name:'demo', exact:true}).waitFor()
    const repo = join(dir, 'demo')
    const git = (...args) => execFileSync('git', ['-C', repo, ...args])
    git('symbolic-ref', 'HEAD', 'refs/heads/main')
    git('-c','user.name=Test','-c','user.email=test@example.org','commit','--allow-empty','-m','initial')
    git('branch','release')
    await button('← Todos los proyectos').click()
    await existing.fill(repo)
    await button('Abrir').click()
    await page.getByLabel('Rama base predeterminada').selectOption('release')
    await button('Guardar').click()
    await page.getByText('Rama base del proyecto actualizada.').waitFor()
    await button('＋ Nueva tarea').click()
    assert.equal(await page.getByLabel('Rama base de esta tarea').inputValue(), 'release')
    await page.getByLabel('Rama base de esta tarea').selectOption('main')
    await page.getByLabel('Título').fill('Primera tarea')
    await page.getByLabel('Descripción').fill('Prueba de integración')
    await button('Crear tarea').click()
    await page.getByText('Primera tarea', {exact:true}).waitFor()
    await page.reload()
    await page.locator('.project-card').filter({has:page.getByText('demo', {exact:true})}).click()
    const task = page.locator('.task-card').filter({hasText:'Primera tarea'})
    await task.waitFor()
    for (const expected of ['main','backlog','Prueba de integración']) assert.ok((await task.innerText()).includes(expected))
    assert.equal(await page.getByLabel('Rama base predeterminada').inputValue(), 'release')
    assert.deepEqual(errors, [])
  } finally {
    await browser?.close()
    if (server && server.exitCode === null && server.signalCode === null) server.kill()
    if (exited) await exited
    await rm(dir, { recursive:true, force:true })
  }
})
