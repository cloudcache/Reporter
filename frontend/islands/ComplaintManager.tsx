import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

interface Complaint {
  id: string
  source: string
  kind: string
  patientName?: string
  patientPhone?: string
  title: string
  content: string
  rating?: number
  category?: string
  authenticity: string
  status: string
  responsibleDepartment?: string
  responsiblePerson?: string
  auditOpinion?: string
  handlingOpinion?: string
  rectificationMeasures?: string
  trackingOpinion?: string
  createdAt?: string
}

type StatItem = { name: string; count: number }
type Stats = Record<"kind" | "status" | "source" | "category", StatItem[]>

const empty: Complaint = {
  id: "",
  source: "manual",
  kind: "complaint",
  patientName: "",
  patientPhone: "",
  title: "",
  content: "",
  rating: 0,
  category: "",
  authenticity: "unconfirmed",
  status: "new",
  responsibleDepartment: "",
  responsiblePerson: "",
  auditOpinion: "",
  handlingOpinion: "",
  rectificationMeasures: "",
  trackingOpinion: "",
}

const labels: Record<string, string> = {
  manual: "手工录入",
  phone_followup: "电话随访",
  questionnaire: "问卷调查",
  sms: "短信回复",
  wechat: "微信回复",
  api: "接口采集",
  complaint: "投诉",
  praise: "表扬",
  suggestion: "建议",
  opinion: "意见",
  new: "待确认",
  confirmed: "已确认",
  audit_pending: "待审核",
  processing: "处理中",
  tracking: "跟踪中",
  archived: "已归档",
  deleted: "已删除",
  unconfirmed: "未确认",
  true: "真实",
  false: "不真实",
  duplicate: "重复",
}

export function ComplaintManager() {
  const [items, setItems] = useState<Complaint[]>([])
  const [stats, setStats] = useState<Stats>({ kind: [], status: [], source: [], category: [] })
  const [draft, setDraft] = useState<Complaint>(empty)
  const [active, setActive] = useState("all")
  const [message, setMessage] = useState("正在加载评价投诉数据...")
  const filtered = useMemo(() => active === "all" ? items : items.filter((item) => item.status === active || item.kind === active), [active, items])

  async function load() {
    try {
      const [nextItems, nextStats] = await Promise.all([
        authedJson<Complaint[]>("/api/v1/evaluation-complaints"),
        authedJson<Stats>("/api/v1/evaluation-complaints/stats"),
      ])
      setItems(nextItems)
      setStats(nextStats)
      setMessage("")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "加载失败")
    }
  }

  async function save() {
    const path = draft.id ? `/api/v1/evaluation-complaints/${draft.id}` : "/api/v1/evaluation-complaints"
    const saved = await authedJson<Complaint>(path, { method: draft.id ? "PUT" : "POST", body: JSON.stringify(draft) })
    setDraft(saved)
    setMessage("已保存评价投诉")
    await load()
  }

  async function transition(status: string, patch: Partial<Complaint> = {}) {
    if (!draft.id) return
    const saved = await authedJson<Complaint>(`/api/v1/evaluation-complaints/${draft.id}`, { method: "PUT", body: JSON.stringify({ ...patch, status }) })
    setDraft(saved)
    await load()
  }

  async function remove(id: string) {
    await authedJson(`/api/v1/evaluation-complaints/${id}`, { method: "DELETE" })
    if (draft.id === id) setDraft(empty)
    await load()
  }

  useEffect(() => { load() }, [])

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_430px]">
      <section className="grid gap-5">
        <div className="grid gap-3 md:grid-cols-4">
          <StatCard title="评价投诉" items={stats.kind} />
          <StatCard title="处理状态" items={stats.status} />
          <StatCard title="来源渠道" items={stats.source} />
          <StatCard title="问题分类" items={stats.category} />
        </div>
        <div className="rounded-lg border border-line bg-surface">
          <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
            <h2 className="text-base font-semibold">评价投诉管理</h2>
            <div className="flex flex-wrap gap-2">
              {["all", "complaint", "praise", "suggestion", "new", "processing", "tracking", "archived"].map((tab) => (
                <button key={tab} className={`rounded-lg border px-3 py-1.5 text-sm ${active === tab ? "border-primary bg-blue-50 text-primary" : "border-line text-muted"}`} onClick={() => setActive(tab)}>
                  {tab === "all" ? "全部" : labels[tab] || tab}
                </button>
              ))}
              <button className="rounded-lg bg-primary px-3 py-1.5 text-sm font-medium text-white" onClick={() => setDraft(empty)}>手工录入</button>
            </div>
          </div>
          {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="bg-gray-50 text-muted">
                <tr><th className="px-4 py-3 text-left">标题</th><th className="px-4 py-3 text-left">患者</th><th className="px-4 py-3 text-left">类型</th><th className="px-4 py-3 text-left">来源</th><th className="px-4 py-3 text-left">状态</th><th className="px-4 py-3 text-right">操作</th></tr>
              </thead>
              <tbody>{filtered.map((item) => (
                <tr key={item.id} className="border-t border-line">
                  <td className="px-4 py-3"><button className="font-medium text-ink hover:text-primary" onClick={() => setDraft(item)}>{item.title}</button><div className="mt-1 max-w-md truncate text-xs text-muted">{item.content}</div></td>
                  <td className="px-4 py-3">{item.patientName || "-"}<div className="text-xs text-muted">{item.patientPhone}</div></td>
                  <td className="px-4 py-3">{labels[item.kind] || item.kind}</td>
                  <td className="px-4 py-3">{labels[item.source] || item.source}</td>
                  <td className="px-4 py-3">{labels[item.status] || item.status}</td>
                  <td className="px-4 py-3 text-right"><button className="text-primary" onClick={() => setDraft(item)}>处理</button><button className="ml-3 text-red-600" onClick={() => remove(item.id)}>删除</button></td>
                </tr>
              ))}</tbody>
            </table>
          </div>
        </div>
      </section>

      <aside className="rounded-lg border border-line bg-surface p-4">
        <h2 className="text-base font-semibold">{draft.id ? "处理评价投诉" : "评价录入"}</h2>
        <div className="mt-4 grid gap-3 text-sm">
          <Select label="来源" value={draft.source} options={["manual", "phone_followup", "questionnaire", "sms", "wechat", "api"]} onChange={(v) => setDraft({ ...draft, source: v })} />
          <Select label="类型" value={draft.kind} options={["complaint", "praise", "suggestion", "opinion"]} onChange={(v) => setDraft({ ...draft, kind: v })} />
          <div className="grid gap-3 md:grid-cols-2"><Text label="患者姓名" value={draft.patientName || ""} onChange={(v) => setDraft({ ...draft, patientName: v })} /><Text label="联系电话" value={draft.patientPhone || ""} onChange={(v) => setDraft({ ...draft, patientPhone: v })} /></div>
          <Text label="标题" value={draft.title} onChange={(v) => setDraft({ ...draft, title: v })} />
          <label className="grid gap-1"><span className="text-muted">内容</span><textarea className="min-h-24 rounded-lg border border-line px-3 py-2" value={draft.content} onChange={(e) => setDraft({ ...draft, content: e.target.value })} /></label>
          <div className="grid gap-3 md:grid-cols-2"><Text label="分类" value={draft.category || ""} onChange={(v) => setDraft({ ...draft, category: v })} /><NumberField label="评分" value={draft.rating || 0} onChange={(v) => setDraft({ ...draft, rating: v })} /></div>
          <Select label="真实性" value={draft.authenticity} options={["unconfirmed", "true", "false", "duplicate"]} onChange={(v) => setDraft({ ...draft, authenticity: v })} />
          <div className="grid gap-3 md:grid-cols-2"><Text label="处理科室" value={draft.responsibleDepartment || ""} onChange={(v) => setDraft({ ...draft, responsibleDepartment: v })} /><Text label="责任人" value={draft.responsiblePerson || ""} onChange={(v) => setDraft({ ...draft, responsiblePerson: v })} /></div>
          <Text label="审核意见" value={draft.auditOpinion || ""} onChange={(v) => setDraft({ ...draft, auditOpinion: v })} />
          <Text label="处理意见" value={draft.handlingOpinion || ""} onChange={(v) => setDraft({ ...draft, handlingOpinion: v })} />
          <Text label="整改措施" value={draft.rectificationMeasures || ""} onChange={(v) => setDraft({ ...draft, rectificationMeasures: v })} />
          <Text label="跟踪意见" value={draft.trackingOpinion || ""} onChange={(v) => setDraft({ ...draft, trackingOpinion: v })} />
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={save}>保存</button>
          {draft.id && <div className="grid grid-cols-2 gap-2">
            <button className="rounded-lg border border-line px-3 py-2" onClick={() => transition("confirmed", { authenticity: draft.authenticity === "unconfirmed" ? "true" : draft.authenticity })}>确认分类</button>
            <button className="rounded-lg border border-line px-3 py-2" onClick={() => transition("processing")}>审核通过</button>
            <button className="rounded-lg border border-line px-3 py-2" onClick={() => transition("tracking")}>进入跟踪</button>
            <button className="rounded-lg border border-line px-3 py-2" onClick={() => transition("archived")}>投诉归档</button>
          </div>}
        </div>
      </aside>
    </div>
  )
}

function StatCard({ title, items }: { title: string; items: StatItem[] }) {
  const total = items.reduce((sum, item) => sum + item.count, 0)
  return <div className="rounded-lg border border-line bg-surface p-4"><div className="text-xs text-muted">{title}</div><div className="mt-2 text-2xl font-semibold">{total}</div><div className="mt-3 grid gap-1 text-xs text-muted">{items.slice(0, 4).map((item) => <div key={item.name} className="flex justify-between"><span>{labels[item.name] || item.name}</span><span>{item.count}</span></div>)}</div></div>
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input className="rounded-lg border border-line px-3 py-2" value={value} onChange={(e) => onChange(e.target.value)} /></label>
}

function NumberField({ label, value, onChange }: { label: string; value: number; onChange: (value: number) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input type="number" className="rounded-lg border border-line px-3 py-2" value={value} onChange={(e) => onChange(Number(e.target.value))} /></label>
}

function Select({ label, value, options, onChange }: { label: string; value: string; options: string[]; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><select className="rounded-lg border border-line px-3 py-2" value={value} onChange={(e) => onChange(e.target.value)}>{options.map((item) => <option key={item} value={item}>{labels[item] || item}</option>)}</select></label>
}
