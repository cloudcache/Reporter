import { useEffect, useState } from "react"
import { currentUser, logout, type CurrentUser } from "../lib/auth"

const roleLabel: Record<string, string> = {
  admin: "系统管理员",
  analyst: "数据分析员",
  agent: "随访坐席",
}

export function AccountMenu() {
  const [user, setUser] = useState<CurrentUser | null>(null)
  const [open, setOpen] = useState(false)

  useEffect(() => {
    currentUser()
      .then(setUser)
      .catch(() => setUser(null))
  }, [])

  const name = user?.displayName || "未登录"
  const role = user?.roles?.map((item) => roleLabel[item] || item).join("、") || "请登录"

  return (
    <div className="relative">
      <button className="flex items-center gap-3 rounded-lg px-2 py-1.5 hover:bg-gray-50" type="button" onClick={() => setOpen(!open)}>
        <span className="hidden text-right sm:block">
          <span className="block text-sm font-medium text-ink">{name}</span>
          <span className="block text-xs text-muted">{role}</span>
        </span>
        <span className="grid h-9 w-9 place-items-center rounded-full bg-gray-100 text-sm font-semibold text-ink">{name.slice(0, 1)}</span>
      </button>
      {open && (
        <div className="absolute right-0 mt-2 w-44 rounded-lg border border-line bg-white p-1 text-sm shadow-lg">
          <a className="block rounded-md px-3 py-2 hover:bg-gray-50" href="/account">个人中心</a>
          <button className="block w-full rounded-md px-3 py-2 text-left text-red-600 hover:bg-red-50" type="button" onClick={logout}>退出登录</button>
        </div>
      )}
    </div>
  )
}
