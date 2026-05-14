import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

interface FollowupTask {
  id: string
  patientId: string
  patientName?: string
  patientPhone?: string
  formTemplateId?: string
  assigneeName?: string
  role?: string
  channel: string
  status: string
  priority: string
  dueAt: string
  lastEvent?: string
}

interface ScheduleEvent {
  id: string
  type: "followup"
  title: string
  date: string
  time?: string
  patientName?: string
  patientPhone?: string
  owner?: string
  status: string
  priority?: string
  description?: string
}

const statusLabels: Record<string, string> = { pending: "待随访", assigned: "已分配", in_progress: "进行中", completed: "已完成", failed: "失败" }
const channelLabels: Record<string, string> = { phone: "电话", sms: "短信", wechat: "微信", qq: "QQ", web: "Web" }

export function ScheduleManager() {
  const [tasks, setTasks] = useState<FollowupTask[]>([])
  const [message, setMessage] = useState("正在加载日程...")
  const [month, setMonth] = useState(() => new Date().toISOString().slice(0, 7))
  const [selectedDate, setSelectedDate] = useState(() => new Date().toISOString().slice(0, 10))
  const [statusFilter, setStatusFilter] = useState("")

  async function load() {
    try {
      const data = await authedJson<FollowupTask[]>("/api/v1/followup/tasks")
      setTasks(data)
      setMessage("")
    } catch (error) {
      setMessage(`日程 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  useEffect(() => { load() }, [])

  const events = useMemo<ScheduleEvent[]>(() => tasks.map((task) => {
    const due = normalizeDateTime(task.dueAt)
    return {
      id: task.id,
      type: "followup",
      title: `${task.patientName || task.patientId}随访`,
      date: due.date,
      time: due.time,
      patientName: task.patientName,
      patientPhone: task.patientPhone,
      owner: task.assigneeName || task.role,
      status: task.status,
      priority: task.priority,
      description: `${channelLabels[task.channel] || task.channel || "随访"} · ${task.lastEvent || "待处理"}`,
    }
  }).filter((event) => event.date), [tasks])

  const visibleEvents = events.filter((event) => !statusFilter || event.status === statusFilter)
  const monthDays = buildMonthDays(month)
  const eventsByDate = groupByDate(visibleEvents)
  const selectedEvents = visibleEvents.filter((event) => event.date === selectedDate).sort(compareEvent)
  const today = new Date().toISOString().slice(0, 10)
  const overdueCount = visibleEvents.filter((event) => event.date < today && !["completed", "failed"].includes(event.status)).length
  const todayCount = visibleEvents.filter((event) => event.date === today && !["completed", "failed"].includes(event.status)).length

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_360px]">
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
          <div>
            <h2 className="text-base font-semibold">全局日历</h2>
            <p className="mt-1 text-sm text-muted">随访任务按到期时间自动汇入，后续可继续接投诉整改、复诊预约和自定义提醒。</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <input type="month" className="rounded-lg border border-line px-3 py-2 text-sm" value={month} onChange={(event) => setMonth(event.target.value)} />
            <select className="rounded-lg border border-line px-3 py-2 text-sm" value={statusFilter} onChange={(event) => setStatusFilter(event.target.value)}>
              <option value="">全部状态</option>
              <option value="pending">待随访</option>
              <option value="assigned">已分配</option>
              <option value="in_progress">进行中</option>
              <option value="completed">已完成</option>
              <option value="failed">失败</option>
            </select>
            <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={load}>刷新</button>
          </div>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid grid-cols-7 border-b border-line bg-gray-50 text-center text-xs font-semibold text-muted">
          {["一", "二", "三", "四", "五", "六", "日"].map((day) => <div key={day} className="px-2 py-2">{day}</div>)}
        </div>
        <div className="grid grid-cols-7">
          {monthDays.map((day) => {
            const dayEvents = eventsByDate[day.date] || []
            const active = day.date === selectedDate
            return (
              <button key={day.date} className={`min-h-28 border-b border-r border-line p-2 text-left last:border-r-0 ${day.inMonth ? "bg-white" : "bg-gray-50 text-muted"} ${active ? "ring-2 ring-inset ring-primary" : ""}`} onClick={() => setSelectedDate(day.date)}>
                <div className="flex items-center justify-between">
                  <span className={`grid h-7 w-7 place-items-center rounded-full text-sm ${day.date === today ? "bg-primary text-white" : ""}`}>{Number(day.date.slice(-2))}</span>
                  {dayEvents.length > 0 && <span className="rounded-full bg-blue-50 px-2 py-0.5 text-xs text-primary">{dayEvents.length}</span>}
                </div>
                <div className="mt-2 grid gap-1">
                  {dayEvents.slice(0, 3).map((event) => (
                    <span key={event.id} className={`truncate rounded px-2 py-1 text-xs ${event.status === "completed" ? "bg-green-50 text-green-700" : event.status === "failed" ? "bg-red-50 text-red-700" : "bg-amber-50 text-amber-700"}`}>{event.title}</span>
                  ))}
                  {dayEvents.length > 3 && <span className="text-xs text-muted">+{dayEvents.length - 3} 更多</span>}
                </div>
              </button>
            )
          })}
        </div>
      </section>

      <aside className="grid content-start gap-4">
        <div className="grid grid-cols-2 gap-3">
          <Metric label="今日待办" value={todayCount} />
          <Metric label="逾期提醒" value={overdueCount} tone={overdueCount > 0 ? "danger" : "normal"} />
        </div>
        <section className="rounded-lg border border-line bg-surface">
          <div className="border-b border-line p-4">
            <h2 className="text-base font-semibold">{selectedDate} 日程</h2>
            <p className="mt-1 text-sm text-muted">{selectedEvents.length} 项提醒</p>
          </div>
          <div className="grid gap-3 p-4">
            {selectedEvents.length === 0 && <div className="rounded-lg border border-dashed border-line p-4 text-sm text-muted">当天暂无日程。</div>}
            {selectedEvents.map((event) => (
              <a key={event.id} href="/followups/tasks" className="rounded-lg border border-line p-3 hover:border-primary">
                <div className="flex items-start justify-between gap-2">
                  <div className="font-medium text-ink">{event.title}</div>
                  <span className="shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">{statusLabels[event.status] || event.status}</span>
                </div>
                <div className="mt-2 text-sm text-muted">{event.time || "全天"} · {event.patientPhone || "-"} · {event.owner || "未分配"}</div>
                <div className="mt-1 text-xs text-muted">{event.description}</div>
              </a>
            ))}
          </div>
        </section>
      </aside>
    </div>
  )
}

export function ScheduleOverview() {
  const [tasks, setTasks] = useState<FollowupTask[]>([])
  const [message, setMessage] = useState("正在加载日程...")

  async function load() {
    try {
      const data = await authedJson<FollowupTask[]>("/api/v1/followup/tasks")
      setTasks(data)
      setMessage("")
    } catch (error) {
      setMessage(`日程 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  useEffect(() => { load() }, [])

  const today = new Date().toISOString().slice(0, 10)
  const events = useMemo<ScheduleEvent[]>(() => tasks.map((task) => {
    const due = normalizeDateTime(task.dueAt)
    return {
      id: task.id,
      type: "followup",
      title: `${task.patientName || task.patientId}随访`,
      date: due.date,
      time: due.time,
      patientName: task.patientName,
      patientPhone: task.patientPhone,
      owner: task.assigneeName || task.role,
      status: task.status,
      priority: task.priority,
      description: `${channelLabels[task.channel] || task.channel || "随访"} · ${task.lastEvent || "待处理"}`,
    }
  }).filter((event) => event.date), [tasks])
  const activeEvents = events.filter((event) => !["completed", "failed"].includes(event.status))
  const todayCount = activeEvents.filter((event) => event.date === today).length
  const overdueCount = activeEvents.filter((event) => event.date < today).length
  const upcoming = activeEvents.filter((event) => event.date >= today).sort(compareEvent).slice(0, 5)
  const weekDays = Array.from({ length: 7 }, (_, index) => {
    const date = new Date()
    date.setDate(date.getDate() + index)
    return date.toISOString().slice(0, 10)
  })
  const eventsByDate = groupByDate(activeEvents)

  return <article className="rounded-lg border border-line bg-surface p-5">
    <div className="flex items-start justify-between gap-3">
      <div>
        <h2 className="text-base font-semibold">日程日历</h2>
        <p className="mt-1 text-sm text-muted">随访任务和提醒汇总</p>
      </div>
      <a className="rounded-lg border border-line px-3 py-1.5 text-sm text-primary hover:border-primary" href="/schedule">进入日历</a>
    </div>
    {message && <div className="mt-4 rounded-lg bg-blue-50 px-3 py-2 text-sm text-primary">{message}</div>}
    <div className="mt-4 grid grid-cols-2 gap-3">
      <Metric label="今日待办" value={todayCount} />
      <Metric label="逾期提醒" value={overdueCount} tone={overdueCount > 0 ? "danger" : "normal"} />
    </div>
    <div className="mt-4 grid grid-cols-7 gap-1">
      {weekDays.map((date) => {
        const count = eventsByDate[date]?.length || 0
        return <a key={date} href="/schedule" className={`rounded-lg border px-2 py-2 text-center ${date === today ? "border-primary bg-blue-50 text-primary" : "border-line hover:border-primary"}`}>
          <div className="text-xs text-muted">{["日", "一", "二", "三", "四", "五", "六"][new Date(date).getDay()]}</div>
          <div className="mt-1 text-sm font-semibold">{Number(date.slice(-2))}</div>
          <div className={`mx-auto mt-1 h-1.5 w-1.5 rounded-full ${count > 0 ? "bg-primary" : "bg-gray-200"}`} />
        </a>
      })}
    </div>
    <div className="mt-4 grid gap-2">
      {upcoming.length === 0 && !message && <div className="rounded-lg border border-dashed border-line p-4 text-sm text-muted">近期暂无待办日程。</div>}
      {upcoming.map((event) => <a key={event.id} href="/followups/tasks" className="rounded-lg border border-line px-3 py-2 hover:border-primary">
        <div className="flex items-center justify-between gap-2">
          <span className="truncate text-sm font-medium">{event.title}</span>
          <span className="shrink-0 text-xs text-muted">{event.date.slice(5)} {event.time}</span>
        </div>
        <div className="mt-1 truncate text-xs text-muted">{event.description}</div>
      </a>)}
    </div>
  </article>
}

function normalizeDateTime(value: string) {
  if (!value) return { date: "", time: "" }
  const normalized = value.replace(" ", "T")
  const date = normalized.slice(0, 10)
  const time = normalized.length > 10 ? normalized.slice(11, 16) : ""
  return { date, time }
}

function buildMonthDays(month: string) {
  const [year, monthIndex] = month.split("-").map(Number)
  const first = new Date(year, monthIndex - 1, 1)
  const startOffset = (first.getDay() + 6) % 7
  const start = new Date(year, monthIndex - 1, 1 - startOffset)
  return Array.from({ length: 42 }, (_, index) => {
    const date = new Date(start)
    date.setDate(start.getDate() + index)
    const iso = date.toISOString().slice(0, 10)
    return { date: iso, inMonth: iso.slice(0, 7) === month }
  })
}

function groupByDate(events: ScheduleEvent[]) {
  return events.reduce<Record<string, ScheduleEvent[]>>((acc, event) => {
    acc[event.date] = [...(acc[event.date] || []), event].sort(compareEvent)
    return acc
  }, {})
}

function compareEvent(a: ScheduleEvent, b: ScheduleEvent) {
  return `${a.date} ${a.time || ""}`.localeCompare(`${b.date} ${b.time || ""}`)
}

function Metric({ label, value, tone = "normal" }: { label: string; value: number; tone?: "normal" | "danger" }) {
  return <div className={`rounded-lg border p-4 ${tone === "danger" ? "border-red-200 bg-red-50 text-red-700" : "border-line bg-surface text-ink"}`}>
    <div className="text-sm text-muted">{label}</div>
    <div className="mt-2 text-2xl font-semibold">{value}</div>
  </div>
}
