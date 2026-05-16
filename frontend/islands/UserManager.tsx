import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

interface User {
  id: string
  username: string
  displayName: string
  roles: string[]
  departmentId?: string
  departmentName?: string
  departmentIds?: string[]
  managedDepartmentIds?: string[]
  createdAt?: string
  updatedAt?: string
}

interface Role {
  id: string
  name: string
  description: string
  permissions: string[]
}

interface Department {
  id: string
  code: string
  name: string
  kind: string
  status: string
}

const emptyUser: User & { password?: string } = {
  id: "",
  username: "",
  displayName: "",
  roles: ["agent"],
  departmentId: "",
  departmentIds: [],
  managedDepartmentIds: [],
  password: "",
}

const roleLabel: Record<string, string> = {
  admin: "系统管理员",
  analyst: "数据分析员",
  agent: "随访坐席",
}

function displayRole(roleId: string) {
  return roleLabel[roleId] || roleId
}

export function UserManager() {
  const [users, setUsers] = useState<User[]>([])
  const [roles, setRoles] = useState<Role[]>([])
  const [departments, setDepartments] = useState<Department[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [draft, setDraft] = useState<User & { password?: string }>(emptyUser)
  const [view, setView] = useState<"list" | "form">("list")
  const [message, setMessage] = useState("正在连接用户 API...")
  const selected = useMemo(() => users.find((user) => user.id === selectedId), [users, selectedId])

  async function authed<T>(path: string, init?: RequestInit): Promise<T> {
    return authedJson<T>(path, init)
  }

  async function load() {
    try {
      const [userData, roleData, departmentData] = await Promise.all([
        authed<User[]>("/api/v1/users"),
        authed<Role[]>("/api/v1/roles"),
        authed<Department[]>("/api/v1/departments"),
      ])
      setUsers(userData)
      setRoles(roleData)
      setDepartments(departmentData)
      setMessage("")
    } catch (error) {
      setMessage(`用户 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function saveUser() {
    try {
      const method = selectedId ? "PUT" : "POST"
      const path = selectedId ? `/api/v1/users/${selectedId}` : "/api/v1/users"
      const saved = await authed<User>(path, {
        method,
        body: JSON.stringify({
          username: draft.username,
          displayName: draft.displayName,
          password: draft.password,
          roles: draft.roles,
          departmentId: draft.departmentId,
          departmentIds: draft.departmentIds || [],
          managedDepartmentIds: draft.managedDepartmentIds || [],
        }),
      })
      setUsers(selectedId ? users.map((user) => user.id === selectedId ? saved : user) : [saved, ...users])
      setSelectedId(saved.id)
      setDraft({ ...saved, password: "" })
      setView("list")
      setMessage("用户已保存")
    } catch (error) {
      setMessage(`保存用户失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function deleteUser(id: string) {
    const user = users.find((item) => item.id === id)
    if (!user || !window.confirm(`删除用户「${user.displayName || user.username}」？`)) return
    try {
      await authed<User>(`/api/v1/users/${id}`, { method: "DELETE" })
      setUsers(users.filter((item) => item.id !== id))
      if (selectedId === id) {
        setSelectedId("")
        setDraft(emptyUser)
      }
      setMessage("用户已删除，关联坐席分机会自动解除绑定")
    } catch (error) {
      setMessage(`删除用户失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  function toggleRole(roleId: string) {
    const roles = draft.roles.includes(roleId) ? draft.roles.filter((item) => item !== roleId) : [...draft.roles, roleId]
    setDraft({ ...draft, roles })
  }

  function toggleDepartment(id: string, field: "departmentIds" | "managedDepartmentIds") {
    const current = draft[field] || []
    const next = current.includes(id) ? current.filter((item) => item !== id) : [...current, id]
    setDraft({ ...draft, [field]: next })
  }

  function newUser() {
    setSelectedId("")
    setDraft(emptyUser)
    setView("form")
    setMessage("")
  }

  function editUser(user: User) {
    setSelectedId(user.id)
    setDraft({ ...user, password: "" })
    setView("form")
    setMessage("")
  }

  function backToList() {
    setSelectedId("")
    setDraft(emptyUser)
    setView("list")
  }

  useEffect(() => {
    load()
  }, [])

  return (
    <div className="grid gap-5">
      {view === "list" && (
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between border-b border-line p-4">
          <h2 className="text-sm font-semibold">用户管理</h2>
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={newUser}>新增用户</button>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-xs uppercase text-muted">
              <tr>
                <th className="px-4 py-3 text-left">用户</th>
                <th className="px-4 py-3 text-left">登录名</th>
                <th className="px-4 py-3 text-left">角色</th>
                <th className="px-4 py-3 text-left">所属科室</th>
                <th className="px-4 py-3 text-left">创建时间</th>
                <th className="px-4 py-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              {users.map((user) => (
                <tr key={user.id} className={`cursor-pointer border-t border-line hover:bg-gray-50 ${user.id === selectedId ? "bg-blue-50" : ""}`} onClick={() => editUser(user)}>
                  <td className="px-4 py-3 font-medium">{user.displayName}</td>
                  <td className="px-4 py-3">{user.username}</td>
                  <td className="px-4 py-3">{user.roles?.map(displayRole).join("、")}</td>
                  <td className="px-4 py-3">{user.departmentName || departmentNames(user.departmentIds, departments) || "-"}</td>
                  <td className="px-4 py-3">{user.createdAt?.slice(0, 10)}</td>
                  <td className="px-4 py-3 text-right">
                    <button className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50" onClick={(event) => { event.stopPropagation(); deleteUser(user.id); }}>删除</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
      )}

      {view === "form" && (
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between gap-3 border-b border-line p-4">
          <div>
            <h2 className="text-base font-semibold">{selected ? "编辑用户" : "新增用户"}</h2>
            <p className="mt-1 text-sm text-muted">维护后台账号、角色和坐席权限。</p>
          </div>
          <div className="flex gap-2">
            <button className="rounded-lg border border-line px-4 py-2 text-sm hover:border-primary" onClick={backToList}>返回列表</button>
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={saveUser}>保存</button>
          </div>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid max-w-5xl gap-5 p-4 text-sm md:grid-cols-2 xl:grid-cols-3">
          <label className="grid gap-1">
            <span className="text-muted">登录名</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.username} onChange={(event) => setDraft({ ...draft, username: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">姓名</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.displayName} onChange={(event) => setDraft({ ...draft, displayName: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">{selected ? "重置密码" : "初始密码"}</span>
            <input type="password" className="rounded-lg border border-line px-3 py-2" value={draft.password || ""} onChange={(event) => setDraft({ ...draft, password: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">主属科室</span>
            <select className="rounded-lg border border-line px-3 py-2" value={draft.departmentId || ""} onChange={(event) => setDraft({ ...draft, departmentId: event.target.value, departmentIds: event.target.value ? [event.target.value, ...(draft.departmentIds || [])] : draft.departmentIds })}>
              <option value="">请选择</option>
              {departments.map((department) => <option key={department.id} value={department.id}>{department.name}</option>)}
            </select>
          </label>
          <div className="grid gap-2 md:col-span-2 xl:col-span-3">
            <span className="text-muted">角色</span>
            {roles.map((role) => (
              <label key={role.id} className="flex items-start gap-2 rounded-lg border border-line p-3">
                <input type="checkbox" className="mt-1" checked={draft.roles.includes(role.id)} onChange={() => toggleRole(role.id)} />
                <span>
                  <span className="block font-medium">{role.name || displayRole(role.id)}</span>
                  <span className="text-xs text-muted">{role.description}</span>
                </span>
              </label>
            ))}
          </div>
          <div className="grid gap-2 md:col-span-2 xl:col-span-3">
            <span className="text-muted">所属科室</span>
            <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              {departments.map((department) => (
                <label key={department.id} className="flex items-center gap-2 rounded-lg border border-line p-3">
                  <input type="checkbox" checked={(draft.departmentIds || []).includes(department.id) || draft.departmentId === department.id} onChange={() => toggleDepartment(department.id, "departmentIds")} />
                  <span>
                    <span className="block font-medium">{department.name}</span>
                    <span className="text-xs text-muted">{department.code} · {department.kind}</span>
                  </span>
                </label>
              ))}
            </div>
          </div>
          <div className="grid gap-2 md:col-span-2 xl:col-span-3">
            <span className="text-muted">管理范围</span>
            <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
              {departments.map((department) => (
                <label key={department.id} className="flex items-center gap-2 rounded-lg border border-line p-3">
                  <input type="checkbox" checked={(draft.managedDepartmentIds || []).includes(department.id)} onChange={() => toggleDepartment(department.id, "managedDepartmentIds")} />
                  <span>
                    <span className="block font-medium">{department.name}</span>
                    <span className="text-xs text-muted">可查看和管理该科室范围数据</span>
                  </span>
                </label>
              ))}
            </div>
          </div>
        </div>
      </section>
      )}
    </div>
  )
}

function departmentNames(ids: string[] | undefined, departments: Department[]) {
  if (!ids?.length) return ""
  const names = ids.map((id) => departments.find((department) => department.id === id)?.name).filter(Boolean)
  return names.join("、")
}
