import { useEffect, useState } from "react"
import { authedJson, logout, type CurrentUser } from "../lib/auth"

const roleLabel: Record<string, string> = {
  admin: "系统管理员",
  analyst: "数据分析员",
  agent: "随访坐席",
}

export function AccountCenter() {
  const [user, setUser] = useState<CurrentUser | null>(null)
  const [displayName, setDisplayName] = useState("")
  const [oldPassword, setOldPassword] = useState("")
  const [newPassword, setNewPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [message, setMessage] = useState("正在读取账户信息...")
  const [savingProfile, setSavingProfile] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)

  useEffect(() => {
    authedJson<CurrentUser>("/api/v1/auth/me")
      .then((data) => {
        setUser(data)
        setDisplayName(data.displayName)
        setMessage("")
      })
      .catch((error) => setMessage(error instanceof Error ? error.message : "账户信息读取失败"))
  }, [])

  async function saveProfile() {
    setSavingProfile(true)
    try {
      const updated = await authedJson<CurrentUser>("/api/v1/auth/me", {
        method: "PUT",
        body: JSON.stringify({ displayName }),
      })
      setUser(updated)
      setMessage("个人资料已保存")
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "未知错误"}`)
    } finally {
      setSavingProfile(false)
    }
  }

  async function savePassword() {
    if (newPassword !== confirmPassword) {
      setMessage("两次输入的新密码不一致")
      return
    }
    setSavingPassword(true)
    try {
      await authedJson<{ status: string }>("/api/v1/auth/password", {
        method: "PUT",
        body: JSON.stringify({ oldPassword, newPassword }),
      })
      setOldPassword("")
      setNewPassword("")
      setConfirmPassword("")
      setMessage("密码已修改，请使用新密码登录")
    } catch (error) {
      setMessage(`修改失败：${error instanceof Error ? error.message : "未知错误"}`)
    } finally {
      setSavingPassword(false)
    }
  }

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
      <section className="rounded-lg border border-line bg-surface">
        <div className="border-b border-line p-4">
          <h2 className="text-base font-semibold">个人资料</h2>
          <p className="mt-1 text-sm text-muted">查看当前登录账户并维护后台显示名称。</p>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid max-w-3xl gap-4 p-4 text-sm md:grid-cols-2">
          <Read label="用户 ID" value={user?.id || "-"} />
          <Read label="登录名" value={user?.username || "-"} />
          <Read label="角色" value={user?.roles.map((role) => roleLabel[role] || role).join("、") || "-"} />
          <label className="grid gap-1">
            <span className="text-muted">显示名称</span>
            <input className="rounded-lg border border-line px-3 py-2" value={displayName} onChange={(event) => setDisplayName(event.target.value)} />
          </label>
        </div>
        <div className="border-t border-line p-4">
          <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white disabled:bg-gray-300" type="button" disabled={savingProfile} onClick={saveProfile}>
            {savingProfile ? "保存中..." : "保存资料"}
          </button>
        </div>
      </section>

      <section className="rounded-lg border border-line bg-surface">
        <div className="border-b border-line p-4">
          <h2 className="text-base font-semibold">修改密码</h2>
          <p className="mt-1 text-sm text-muted">新密码至少 8 位，修改后可继续当前会话。</p>
        </div>
        <div className="grid gap-4 p-4 text-sm">
          <Password label="原密码" value={oldPassword} onChange={setOldPassword} />
          <Password label="新密码" value={newPassword} onChange={setNewPassword} />
          <Password label="确认新密码" value={confirmPassword} onChange={setConfirmPassword} />
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white disabled:bg-gray-300" type="button" disabled={savingPassword} onClick={savePassword}>
            {savingPassword ? "提交中..." : "修改密码"}
          </button>
          <button className="rounded-lg border border-line px-4 py-2 font-medium text-red-600 hover:bg-red-50" type="button" onClick={logout}>退出登录</button>
        </div>
      </section>
    </div>
  )
}

function Read({ label, value }: { label: string; value: string }) {
  return <div className="grid gap-1"><span className="text-muted">{label}</span><span className="rounded-lg border border-line bg-gray-50 px-3 py-2">{value}</span></div>
}

function Password({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input type="password" className="rounded-lg border border-line px-3 py-2" value={value} onChange={(event) => onChange(event.target.value)} /></label>
}
