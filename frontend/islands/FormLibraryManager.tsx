import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

interface Item { id: string; kind: string; label: string; hint: string; scenario?: string; components: Array<Record<string, unknown>>; sortOrder: number; enabled?: boolean }
interface Library { templates: Item[]; commonComponents: Item[]; atomicComponents: Item[] }

const empty: Item = { id: "", kind: "common", label: "", hint: "", scenario: "", components: [], sortOrder: 100, enabled: true }
const kindLabels: Record<string, string> = { template: "表单模板", common: "组件模板", atom: "字段组件" }
const kindHints: Record<string, string> = {
  template: "可直接绑定到项目，用于调查、随访、登记等采集。",
  common: "可复用的业务片段，例如患者基础信息、随访记录、满意度题组。",
  atom: "最小字段控件，例如文本、日期、单选、下拉、评分。",
}
const kindOrder = ["template", "common", "atom"]
const templateFlow = [
  { title: "字段组件", text: "沉淀文本、日期、单选、下拉、评分等最小控件。" },
  { title: "组件模板", text: "组合患者基础、就诊、用药、随访记录等业务片段。" },
  { title: "表单模板", text: "形成调查、随访、投诉、登记等可发布采集表单。" },
  { title: "项目绑定", text: "项目选择已发布模板版本，渠道和答卷按版本闭环。" },
]

export function FormLibraryManager({ initialKind = "all", lockedKind = false }: { initialKind?: string; lockedKind?: boolean }) {
  const [items, setItems] = useState<Item[]>([])
  const [draft, setDraft] = useState<Item>(empty)
  const [json, setJson] = useState("[]")
  const [activeKind, setActiveKind] = useState(initialKind)
  const [message, setMessage] = useState("正在加载组件库...")
  const filteredItems = useMemo(() => activeKind === "all" ? items : items.filter((item) => item.kind === activeKind), [activeKind, items])

  async function load() {
    const data = await authedJson<Library>("/api/v1/form-library")
    setItems([...(data.templates || []), ...(data.commonComponents || []), ...(data.atomicComponents || [])])
    setMessage("")
  }

  async function save() {
    let components: Array<Record<string, unknown>>
    try {
      components = JSON.parse(json || "[]")
    } catch {
      setMessage("高级结构不是合法 JSON，请检查后再保存")
      return
    }
    const body = { ...draft, components }
    const saved = await authedJson<Item>(draft.id ? `/api/v1/form-library/${draft.id}` : "/api/v1/form-library", { method: draft.id ? "PUT" : "POST", body: JSON.stringify(body) })
    setDraft(saved)
    setJson(JSON.stringify(saved.components, null, 2))
    setMessage("组件库条目已保存")
    await load()
  }

  function edit(item: Item) {
    setDraft(item)
    setJson(JSON.stringify(item.components || [], null, 2))
  }

  useEffect(() => { load().catch((error) => setMessage(error instanceof Error ? error.message : "加载失败")) }, [])

  return (
    <div className="grid gap-5">
      <section className="rounded-lg border border-line bg-surface p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">表单 / 组件模板闭环</h2>
            <p className="mt-1 text-sm text-muted">模板库不是零散素材，按字段、组件、表单、项目绑定四层复用。</p>
          </div>
          <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={() => { setDraft({ ...empty, kind: initialKind === "all" ? "common" : initialKind }); setJson("[]") }}>{initialKind === "template" ? "新增表单模板" : initialKind === "common" ? "新增组件模板" : "新增模板"}</button>
        </div>
        <div className="mt-4 grid gap-3 md:grid-cols-4">
          {templateFlow.map((step, index) => <FlowStep key={step.title} index={index + 1} title={step.title} text={step.text} />)}
        </div>
      </section>
      <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_460px]">
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
          <div>
            <h2 className="text-base font-semibold">模板库</h2>
            <p className="mt-1 text-sm text-muted">先选层级，再维护条目；表单模板用于项目，组件模板用于复用。</p>
          </div>
          <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => { setDraft({ ...empty, kind: initialKind === "all" ? "common" : initialKind }); setJson("[]") }}>新增</button>
        </div>
        {!lockedKind && <div className="flex flex-wrap gap-2 border-b border-line px-4 py-3">
          {[{ key: "all", label: "全部" }, ...kindOrder.map((kind) => ({ key: kind, label: kindLabels[kind] }))].map((item) => (
            <button key={item.key} className={`rounded-lg px-3 py-1.5 text-sm ${activeKind === item.key ? "bg-blue-50 text-primary" : "text-muted hover:bg-gray-50"}`} onClick={() => setActiveKind(item.key)}>{item.label}</button>
          ))}
        </div>}
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid gap-2 p-4">{filteredItems.map((item) => <button key={item.id} className="rounded-lg border border-line px-3 py-2 text-left hover:border-primary" onClick={() => edit(item)}>
          <span className="font-medium">{item.label}</span>
          <span className="ml-2 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">{kindLabels[item.kind] || item.kind}</span>
          {item.scenario && <span className="ml-2 text-xs text-muted">{item.scenario}</span>}
          <span className="mt-1 block text-xs text-muted">{item.hint || kindHints[item.kind]}</span>
        </button>)}
        {!filteredItems.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-6 text-center text-sm text-muted">当前层级还没有模板。</div>}
        </div>
      </section>
      <aside className="rounded-lg border border-line bg-surface p-4">
        <h2 className="text-base font-semibold">{draft.id ? "编辑条目" : "新增条目"}</h2>
        <p className="mt-1 text-sm text-muted">基础信息在这里维护，高级结构由表单设计器生成，日常不需要手写。</p>
        <div className="mt-4 grid gap-3 text-sm">
          <Text label="ID" value={draft.id} onChange={(v) => setDraft({ ...draft, id: v })} />
          <Text label="名称" value={draft.label} onChange={(v) => setDraft({ ...draft, label: v })} />
          <Text label="说明" value={draft.hint} onChange={(v) => setDraft({ ...draft, hint: v })} />
          <label className="grid gap-1"><span className="text-muted">类型</span><select className="rounded-lg border border-line px-3 py-2" value={draft.kind} disabled={lockedKind} onChange={(e) => setDraft({ ...draft, kind: e.target.value })}><option value="template">表单模板</option><option value="common">组件模板</option><option value="atom">字段组件</option></select></label>
          <div className="rounded-lg bg-blue-50 px-3 py-2 text-xs leading-5 text-primary">{kindHints[draft.kind] || "选择模板类型后用于不同层级复用。"}</div>
          <Text label="适用场景" value={draft.scenario || ""} onChange={(v) => setDraft({ ...draft, scenario: v })} />
          <label className="grid gap-1"><span className="text-muted">高级结构 JSON</span><textarea className="min-h-52 rounded-lg border border-line px-3 py-2 font-mono text-xs" value={json} onChange={(e) => setJson(e.target.value)} /></label>
          <div className="rounded-lg border border-line bg-gray-50 px-3 py-2 text-xs leading-5 text-muted">闭环去向：字段组件进入组件模板，组件模板进入表单模板，表单模板在项目“表单数据”中绑定并随渠道发布。</div>
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={save}>保存</button>
        </div>
      </aside>
      </div>
    </div>
  )
}

function FlowStep({ index, title, text }: { index: number; title: string; text: string }) {
  return <div className="rounded-lg border border-line bg-white p-3">
    <div className="flex items-center gap-2"><span className="grid h-6 w-6 place-items-center rounded-full bg-blue-50 text-xs font-semibold text-primary">{index}</span><span className="font-medium">{title}</span></div>
    <div className="mt-2 text-xs leading-5 text-muted">{text}</div>
  </div>
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input className="rounded-lg border border-line px-3 py-2" value={value} onChange={(e) => onChange(e.target.value)} /></label>
}
