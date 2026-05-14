import { useEffect, useState } from "react"

interface InstallStatus {
  installed: boolean
  lockPath: string
  databaseDsn: boolean
  lockCreatedAt?: string
}

const apiBase = "http://127.0.0.1:8080"

export function InstallWizard() {
  const [status, setStatus] = useState<InstallStatus | null>(null)
  const [message, setMessage] = useState("正在检查安装状态...")
  const [testing, setTesting] = useState(false)
  const [installing, setInstalling] = useState(false)
  const [database, setDatabase] = useState({
    driver: "mysql",
    host: "127.0.0.1",
    port: 3306,
    database: "reporter",
    username: "reporter",
    password: "",
    charset: "utf8mb4",
    loc: "Local",
    dsn: "",
  })
  const [admin, setAdmin] = useState({
    username: "admin",
    displayName: "系统管理员",
    password: "",
    confirmPassword: "",
  })

  async function loadStatus() {
    try {
      const response = await fetch(`${apiBase}/api/v1/install/status`)
      if (!response.ok) throw new Error(await response.text())
      const data = await response.json() as InstallStatus
      setStatus(data)
      if (data.installed) {
        setMessage("系统已安装，正在跳转登录页。")
        window.location.replace("/login")
        return
      }
      setMessage("系统尚未安装，请完成数据库和管理员配置。")
    } catch (error) {
      setMessage(`无法连接安装接口：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function testDB() {
    setTesting(true)
    try {
      const response = await fetch(`${apiBase}/api/v1/install/test-db`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(database),
      })
      const data = await response.json()
      if (!response.ok) throw new Error(data.message || "数据库连接失败")
      setMessage("数据库连接成功")
    } catch (error) {
      setMessage(`数据库连接失败：${error instanceof Error ? error.message : "未知错误"}`)
    } finally {
      setTesting(false)
    }
  }

  async function install() {
    if (admin.password !== admin.confirmPassword) {
      setMessage("两次输入的管理员密码不一致")
      return
    }
    setInstalling(true)
    try {
      const response = await fetch(`${apiBase}/api/v1/install`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ database, admin: { username: admin.username, displayName: admin.displayName, password: admin.password } }),
      })
      const data = await response.json()
      if (!response.ok) throw new Error(data.message || "安装失败")
      setMessage("安装完成，正在进入登录页...")
      window.location.href = "/login"
    } catch (error) {
      setMessage(`安装失败：${error instanceof Error ? error.message : "未知错误"}`)
    } finally {
      setInstalling(false)
    }
  }

  useEffect(() => {
    loadStatus()
  }, [])

  return (
    <div className="mx-auto grid max-w-5xl gap-5">
      <section className="rounded-lg border border-line bg-surface p-5">
        <div className="flex items-start justify-between gap-4">
          <div>
            <h1 className="text-2xl font-bold">Reporter 安装向导</h1>
            <p className="mt-2 text-sm text-muted">首次部署时创建数据库表、初始化角色、创建管理员账户，并生成安装锁。</p>
          </div>
          {status?.installed && <a className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" href="/login">去登录</a>}
        </div>
        {message && <div className="mt-4 rounded-lg border border-blue-100 bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
      </section>

      <fieldset disabled={!!status?.installed || installing} className="grid gap-5 disabled:opacity-60">
        <section className="rounded-lg border border-line bg-surface">
          <div className="border-b border-line p-4">
            <h2 className="text-base font-semibold">数据库连接</h2>
            <p className="mt-1 text-sm text-muted">需要提前创建空数据库，并授予建表、写入权限。</p>
          </div>
          <div className="grid gap-4 p-4 md:grid-cols-2 xl:grid-cols-3">
            <Text label="数据库主机" value={database.host} onChange={(value) => setDatabase({ ...database, host: value })} />
            <NumberField label="端口" value={database.port} onChange={(value) => setDatabase({ ...database, port: value })} />
            <Text label="数据库名" value={database.database} onChange={(value) => setDatabase({ ...database, database: value })} />
            <Text label="用户名" value={database.username} onChange={(value) => setDatabase({ ...database, username: value })} />
            <Password label="密码" value={database.password} onChange={(value) => setDatabase({ ...database, password: value })} />
            <Text label="字符集" value={database.charset} onChange={(value) => setDatabase({ ...database, charset: value })} />
            <label className="grid gap-1 text-sm md:col-span-2 xl:col-span-3">
              <span className="text-muted">DSN，可选；填写后优先使用</span>
              <input className="rounded-lg border border-line px-3 py-2 font-mono text-xs" value={database.dsn} onChange={(event) => setDatabase({ ...database, dsn: event.target.value })} />
            </label>
          </div>
          <div className="border-t border-line p-4">
            <button className="rounded-lg border border-line px-4 py-2 text-sm hover:border-primary disabled:text-muted" type="button" disabled={testing} onClick={testDB}>
              {testing ? "测试中..." : "测试数据库连接"}
            </button>
          </div>
        </section>

        <section className="rounded-lg border border-line bg-surface">
          <div className="border-b border-line p-4">
            <h2 className="text-base font-semibold">管理员账户</h2>
            <p className="mt-1 text-sm text-muted">安装完成后使用该账号登录后台。</p>
          </div>
          <div className="grid gap-4 p-4 md:grid-cols-2">
            <Text label="登录名" value={admin.username} onChange={(value) => setAdmin({ ...admin, username: value })} />
            <Text label="姓名" value={admin.displayName} onChange={(value) => setAdmin({ ...admin, displayName: value })} />
            <Password label="密码" value={admin.password} onChange={(value) => setAdmin({ ...admin, password: value })} />
            <Password label="确认密码" value={admin.confirmPassword} onChange={(value) => setAdmin({ ...admin, confirmPassword: value })} />
          </div>
          <div className="border-t border-line p-4">
            <button className="rounded-lg bg-primary px-5 py-2 text-sm font-medium text-white disabled:bg-gray-300" type="button" disabled={installing} onClick={install}>
              {installing ? "正在安装..." : "开始安装"}
            </button>
          </div>
        </section>
      </fieldset>
    </div>
  )
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1 text-sm"><span className="text-muted">{label}</span><input className="rounded-lg border border-line px-3 py-2" value={value} onChange={(event) => onChange(event.target.value)} /></label>
}

function Password({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1 text-sm"><span className="text-muted">{label}</span><input type="password" className="rounded-lg border border-line px-3 py-2" value={value} onChange={(event) => onChange(event.target.value)} /></label>
}

function NumberField({ label, value, onChange }: { label: string; value: number; onChange: (value: number) => void }) {
  return <label className="grid gap-1 text-sm"><span className="text-muted">{label}</span><input type="number" className="rounded-lg border border-line px-3 py-2" value={value} onChange={(event) => onChange(Number(event.target.value))} /></label>
}
