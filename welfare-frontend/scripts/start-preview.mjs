import { spawn } from 'node:child_process'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

// Zeabur/容器环境通常通过 PORT 注入监听端口；需要绑定 0.0.0.0 才能被外部访问。
const port = String(process.env.PORT || 4173)
const host = String(process.env.HOST || '0.0.0.0')

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const projectRoot = path.resolve(__dirname, '..')
const viteBin = path.join(projectRoot, 'node_modules', 'vite', 'bin', 'vite.js')

const child = spawn(
  process.execPath,
  [viteBin, 'preview', '--host', host, '--port', port, '--strictPort'],
  { cwd: projectRoot, stdio: 'inherit' }
)

child.on('exit', (code, signal) => {
  if (typeof code === 'number') {
    process.exit(code)
  }
  if (signal) {
    // 将退出信号原样透传，便于平台判断是否为正常停止。
    process.kill(process.pid, signal)
    return
  }
  process.exit(0)
})

