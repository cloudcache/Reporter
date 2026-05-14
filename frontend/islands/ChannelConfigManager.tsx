import { useEffect, useState } from "react"
import { authedJson } from "../lib/auth"

interface Channel {
  id: string
  kind: string
  name: string
  endpoint?: string
  appId?: string
  credentialRef?: string
  enabled: boolean
  config?: Record<string, unknown>
}

const empty: Channel = { id: "", kind: "sms", name: "", endpoint: "", appId: "", credentialRef: "", enabled: true, config: {} }
const kindLabel: Record<string, string> = { sms: "短信", wechat: "微信", qq: "QQ", web: "Web 链接" }

export function ChannelConfigManager() {
  const [items, setItems] = useState<Channel[]>([])
  const [draft, setDraft] = useState<Channel>(empty)
  const [configText, setConfigText] = useState("{}")
  const [message, setMessage] = useState("正在加载接口配置...")

  async function load() {
    try {
      const data = await authedJson<Channel[]>("/api/v1/integration-channels")
      setItems(data)
      setMessage("")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "加载失败")
    }
  }

  async function save() {
    const saved = await authedJson<Channel>("/api/v1/integration-channels", { method: "POST", body: JSON.stringify({ ...draft, config: JSON.parse(configText || "{}") }) })
    setDraft(saved)
    setConfigText(JSON.stringify(saved.config || {}, null, 2))
    setMessage("接口配置已保存")
    await load()
  }

  function edit(item: Channel) {
    setDraft(item)
    setConfigText(JSON.stringify(item.config || {}, null, 2))
  }

  useEffect(() => { load() }, [])

  return (
    <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_440px]">
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between border-b border-line p-4"><h2 className="text-base font-semibold">统一接口配置</h2><button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => { setDraft(empty); setConfigText("{}") }}>新增接口</button></div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid gap-3 p-4 md:grid-cols-2">
          {items.map((item) => <button key={item.id} className="rounded-lg border border-line p-4 text-left hover:border-primary" onClick={() => edit(item)}>
            <div className="flex items-center justify-between"><span className="font-medium">{item.name}</span><span className={`rounded-full px-2 py-1 text-xs ${item.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-muted"}`}>{item.enabled ? "启用" : "停用"}</span></div>
            <div className="mt-2 text-sm text-muted">{kindLabel[item.kind] || item.kind}</div>
            <div className="mt-1 truncate text-xs text-muted">{item.endpoint || "-"}</div>
          </button>)}
        </div>
      </section>
      <aside className="rounded-lg border border-line bg-surface p-4">
        <h2 className="text-base font-semibold">{draft.id ? "编辑接口" : "新增接口"}</h2>
        <div className="mt-4 grid gap-3 text-sm">
          <label className="grid gap-1"><span className="text-muted">接口类型</span><select className="rounded-lg border border-line px-3 py-2" value={draft.kind} onChange={(e) => setDraft({ ...draft, kind: e.target.value })}>{["sms", "wechat", "qq", "web"].map((kind) => <option key={kind} value={kind}>{kindLabel[kind]}</option>)}</select></label>
          <Text label="名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
          <Text label="Endpoint" value={draft.endpoint || ""} onChange={(v) => setDraft({ ...draft, endpoint: v })} />
          <Text label="AppID" value={draft.appId || ""} onChange={(v) => setDraft({ ...draft, appId: v })} />
          <Text label="凭据引用" value={draft.credentialRef || ""} onChange={(v) => setDraft({ ...draft, credentialRef: v })} />
          <label className="flex items-center gap-2"><input type="checkbox" checked={draft.enabled} onChange={(e) => setDraft({ ...draft, enabled: e.target.checked })} />启用</label>
          <label className="grid gap-1"><span className="text-muted">扩展配置 JSON</span><textarea className="min-h-40 rounded-lg border border-line px-3 py-2 font-mono text-xs" value={configText} onChange={(e) => setConfigText(e.target.value)} /></label>
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={save}>保存接口</button>
        </div>
      </aside>
    </div>
  )
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input className="rounded-lg border border-line px-3 py-2" value={value} onChange={(e) => onChange(e.target.value)} /></label>
}
