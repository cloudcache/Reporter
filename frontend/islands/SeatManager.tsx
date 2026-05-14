import { useEffect, useState } from "react"
import { authedJson } from "../lib/auth"

interface Seat {
  id: string
  userId?: string
  username?: string
  userDisplay?: string
  name: string
  extension: string
  sipUri: string
  status: string
  skills: string[]
}

interface User {
  id: string
  username: string
  displayName: string
  roles: string[]
}

const emptySeat: Seat = {
  id: "",
  userId: "",
  name: "",
  extension: "",
  sipUri: "",
  status: "offline",
  skills: [],
}

const statusOptions = [
  { value: "available", label: "空闲" },
  { value: "busy", label: "忙碌" },
  { value: "offline", label: "离线" },
  { value: "wrap_up", label: "整理中" },
]

function statusLabel(status: string) {
  return statusOptions.find((option) => option.value === status)?.label || status
}

export function SeatManager() {
  const [message, setMessage] = useState("正在连接坐席 API...")
  const [seats, setSeats] = useState<Seat[]>([])
  const [users, setUsers] = useState<User[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [draft, setDraft] = useState<Seat>(emptySeat)
  const [view, setView] = useState<"list" | "form">("list")

  async function authed<T>(path: string, init?: RequestInit): Promise<T> {
    return authedJson<T>(path, init)
  }

  async function load() {
    try {
      const [seatData, userData] = await Promise.all([
        authed<Seat[]>("/api/v1/call-center/seats"),
        authed<User[]>("/api/v1/users"),
      ])
      setSeats(seatData)
      setUsers(userData.filter((user) => user.roles?.includes("agent") || user.roles?.includes("admin")))
      setMessage("")
    } catch (error) {
      setMessage(`坐席 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function save() {
    try {
      const method = selectedId ? "PUT" : "POST"
      const path = selectedId ? `/api/v1/call-center/seats/${selectedId}` : "/api/v1/call-center/seats"
      const saved = await authed<Seat>(path, { method, body: JSON.stringify(draft) })
      setSeats(selectedId ? seats.map((seat) => seat.id === selectedId ? saved : seat) : [saved, ...seats])
      setSelectedId(saved.id)
      setDraft(saved)
      setView("list")
      setMessage("坐席分机已保存")
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function remove(id: string) {
    const seat = seats.find((item) => item.id === id)
    if (!seat || !window.confirm(`删除分机「${seat.name}」？`)) return
    try {
      await authed<Seat>(`/api/v1/call-center/seats/${id}`, { method: "DELETE" })
      setSeats(seats.filter((item) => item.id !== id))
      if (selectedId === id) {
        setSelectedId("")
        setDraft(emptySeat)
      }
      setMessage("坐席分机已删除")
    } catch (error) {
      setMessage(`删除失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  function newSeat() {
    setSelectedId("")
    setDraft(emptySeat)
    setView("form")
    setMessage("")
  }

  function editSeat(seat: Seat) {
    setSelectedId(seat.id)
    setDraft(seat)
    setView("form")
    setMessage("")
  }

  function backToList() {
    setSelectedId("")
    setDraft(emptySeat)
    setView("list")
  }

  useEffect(() => {
    load()
  }, [])

  return (
    <section className="grid gap-5">
      {view === "list" && (
      <article className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between border-b border-line p-4">
          <h2 className="text-sm font-semibold">坐席分机</h2>
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={newSeat}>新增分机</button>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="overflow-x-auto">
          <table className="w-full text-sm">
            <thead className="bg-gray-50 text-xs uppercase text-muted">
              <tr>
                <th className="px-4 py-3 text-left">分机名称</th>
                <th className="px-4 py-3 text-left">绑定坐席用户</th>
                <th className="px-4 py-3 text-left">分机号</th>
                <th className="px-4 py-3 text-left">技能</th>
                <th className="px-4 py-3 text-left">状态</th>
                <th className="px-4 py-3 text-right">操作</th>
              </tr>
            </thead>
            <tbody>
              {seats.map((seat) => (
                <tr key={seat.id} className={`cursor-pointer border-t border-line hover:bg-gray-50 ${seat.id === selectedId ? "bg-blue-50" : ""}`} onClick={() => editSeat(seat)}>
                  <td className="px-4 py-3">
                    <div className="font-medium">{seat.name}</div>
                    <div className="mt-1 text-xs text-muted">{seat.sipUri}</div>
                  </td>
                  <td className="px-4 py-3">
                    <div>{seat.userDisplay || "-"}</div>
                    <div className="mt-1 text-xs text-muted">{seat.username || ""}</div>
                  </td>
                  <td className="px-4 py-3">{seat.extension}</td>
                  <td className="px-4 py-3">{seat.skills?.join("、")}</td>
                  <td className="px-4 py-3"><span className="rounded-full bg-gray-100 px-2 py-1 text-xs">{statusLabel(seat.status)}</span></td>
                  <td className="px-4 py-3 text-right">
                    <button className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50" onClick={(event) => { event.stopPropagation(); remove(seat.id); }}>删除</button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </article>
      )}

      {view === "form" && (
      <article className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between gap-3 border-b border-line p-4">
          <div>
            <h2 className="text-base font-semibold">{selectedId ? "编辑分机" : "新增分机"}</h2>
            <p className="mt-1 text-sm text-muted">为已有用户绑定坐席分机、SIP URI、状态和技能。</p>
          </div>
          <div className="flex gap-2">
            <button className="rounded-lg border border-line px-4 py-2 text-sm hover:border-primary" onClick={backToList}>返回列表</button>
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={save}>保存</button>
          </div>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid max-w-5xl gap-5 p-4 text-sm md:grid-cols-2 xl:grid-cols-3">
          <label className="grid gap-1">
            <span className="text-muted">绑定坐席用户</span>
            <select className="rounded-lg border border-line px-3 py-2" value={draft.userId || ""} onChange={(event) => {
              const user = users.find((item) => item.id === event.target.value)
              setDraft({ ...draft, userId: event.target.value, username: user?.username, userDisplay: user?.displayName })
            }}>
              <option value="">未绑定</option>
              {users.map((user) => <option key={user.id} value={user.id}>{user.displayName}（{user.username}）</option>)}
            </select>
          </label>
          <label className="grid gap-1">
            <span className="text-muted">分机名称</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.name} onChange={(event) => setDraft({ ...draft, name: event.target.value })} />
          </label>
          <div className="grid grid-cols-2 gap-2">
            <label className="grid gap-1">
              <span className="text-muted">分机号</span>
              <input className="rounded-lg border border-line px-3 py-2" value={draft.extension} onChange={(event) => setDraft({ ...draft, extension: event.target.value })} />
            </label>
            <label className="grid gap-1">
              <span className="text-muted">状态</span>
              <select className="rounded-lg border border-line px-3 py-2" value={draft.status} onChange={(event) => setDraft({ ...draft, status: event.target.value })}>
                {statusOptions.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
          </div>
          <label className="grid gap-1">
            <span className="text-muted">SIP URI</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.sipUri} onChange={(event) => setDraft({ ...draft, sipUri: event.target.value })} />
          </label>
          <label className="grid gap-1">
            <span className="text-muted">技能，逗号分隔</span>
            <input className="rounded-lg border border-line px-3 py-2" value={draft.skills?.join(",") || ""} onChange={(event) => setDraft({ ...draft, skills: event.target.value.split(",").map((item) => item.trim()).filter(Boolean) })} />
          </label>
        </div>
      </article>
      )}
    </section>
  )
}
