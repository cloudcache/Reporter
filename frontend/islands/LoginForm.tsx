import { useEffect, useState } from "react"
import { accessTokenKey, apiBase } from "../lib/auth"

export function LoginForm() {
  const [username, setUsername] = useState("admin")
  const [password, setPassword] = useState("")
  const [captchaId, setCaptchaId] = useState("")
  const [captchaQuestion, setCaptchaQuestion] = useState("")
  const [captchaAnswer, setCaptchaAnswer] = useState("")
  const [message, setMessage] = useState("")
  const [loading, setLoading] = useState(false)
  const [showInstallLink, setShowInstallLink] = useState(false)

  useEffect(() => {
    fetch(`${apiBase}/api/v1/install/status`)
      .then((response) => response.ok ? response.json() : null)
      .then((status) => setShowInstallLink(status ? !status.installed : false))
      .catch(() => setShowInstallLink(false))
  }, [])

  async function login() {
    if (loading) return
    setLoading(true)
    try {
      const response = await fetch(`${apiBase}/api/v1/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        body: JSON.stringify({ username, password, captchaId, captchaAnswer }),
      })
      const text = await response.text()
      const data = text ? safeParse(text) : {}
      if (!response.ok) {
        if (data.captchaRequired) await loadCaptcha()
        throw new Error(data.message || text || "登录失败")
      }
      if (data.accessToken) {
        localStorage.setItem(accessTokenKey, data.accessToken)
      }
      const me = await fetch(`${apiBase}/api/v1/auth/me`, {
        credentials: "include",
        headers: data.accessToken ? { Authorization: `Bearer ${data.accessToken}` } : {},
      })
      if (!me.ok) {
        throw new Error("登录已成功但会话校验失败，请刷新后重试")
      }
      const params = new URLSearchParams(window.location.search)
      window.location.replace(params.get("next") || "/")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "登录失败")
    } finally {
      setLoading(false)
    }
  }

  async function loadCaptcha() {
    const response = await fetch(`${apiBase}/api/v1/auth/captcha?username=${encodeURIComponent(username)}`, {
      credentials: "include",
    })
    if (!response.ok) return
    const data = await response.json()
    if (data.required) {
      setCaptchaId(data.captchaId)
      setCaptchaQuestion(data.question)
      setCaptchaAnswer("")
    }
  }

  return (
    <section className="mx-auto mt-24 w-full max-w-md rounded-lg border border-line bg-surface p-6 shadow-sm">
      <h1 className="text-2xl font-bold">登录 Reporter</h1>
      <p className="mt-2 text-sm text-muted">请输入安装时创建的管理员账户。</p>
      {message && <div className="mt-4 rounded-lg border border-red-100 bg-red-50 px-4 py-3 text-sm text-red-600">{message}</div>}
      <div className="mt-5 grid gap-4">
        <label className="grid gap-1 text-sm">
          <span className="text-muted">登录名</span>
          <input className="rounded-lg border border-line px-3 py-2" value={username} onChange={(event) => setUsername(event.target.value)} />
        </label>
        <label className="grid gap-1 text-sm">
          <span className="text-muted">密码</span>
          <input type="password" className="rounded-lg border border-line px-3 py-2" value={password} onChange={(event) => setPassword(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") login() }} />
        </label>
        {captchaQuestion && (
          <label className="grid gap-1 text-sm">
            <span className="text-muted">验证码：{captchaQuestion}</span>
            <input className="rounded-lg border border-line px-3 py-2" value={captchaAnswer} onChange={(event) => setCaptchaAnswer(event.target.value)} onKeyDown={(event) => { if (event.key === "Enter") login() }} />
          </label>
        )}
        <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white disabled:bg-gray-300" disabled={loading} onClick={login}>{loading ? "登录中..." : "登录"}</button>
        {showInstallLink && <a className="text-center text-sm text-primary hover:underline" href="/install">进入安装向导</a>}
      </div>
    </section>
  )
}

function safeParse(text: string) {
  try {
    return JSON.parse(text)
  } catch {
    return {}
  }
}
