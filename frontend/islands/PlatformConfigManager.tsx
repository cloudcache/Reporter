import { useEffect, useState } from "react"
import { apiBase, requireSession } from "../lib/auth"

type Section = "sip" | "storage" | "recording" | "models"

interface Props {
  section: Section
}

interface SipEndpoint {
  id: string
  name: string
  wssUrl: string
  domain: string
  proxy: string
  config?: Record<string, unknown>
}

interface StorageConfig {
  id: string
  name: string
  kind: string
  endpoint?: string
  bucket?: string
  basePath?: string
  baseUri?: string
  credentialRef?: string
  config?: Record<string, unknown>
}

interface RecordingConfig {
  id: string
  name: string
  mode: string
  storageConfigId: string
  format: string
  retentionDays: number
  autoStart: boolean
  autoStop: boolean
  config?: Record<string, unknown>
}

interface ModelProvider {
  id: string
  name: string
  kind: string
  mode: string
  endpoint: string
  model: string
  credentialRef?: string
  config?: Record<string, unknown>
}

type ConfigItem = SipEndpoint | StorageConfig | RecordingConfig | ModelProvider

const meta = {
  sip: { path: "/api/v1/call-center/sip-endpoints", title: "SIP 网关配置", create: "新增网关" },
  storage: { path: "/api/v1/call-center/storage-configs", title: "存储配置", create: "新增存储" },
  recording: { path: "/api/v1/call-center/recording-configs", title: "录音策略配置", create: "新增策略" },
  models: { path: "/api/v1/call-center/model-providers", title: "大模型配置", create: "新增模型接口" },
}

const empty = {
  sip: { id: "", name: "", wssUrl: "", domain: "", proxy: "", config: { enabled: false, transport: "udp", trunkUri: "sip:{phone}@carrier.example.local" } },
  storage: { id: "", name: "", kind: "local", basePath: "data/recordings", baseUri: "", config: { pathStrategy: "yyyy/mm/dd" } },
  recording: { id: "", name: "", mode: "server", storageConfigId: "STOR001", format: "wav", retentionDays: 365, autoStart: true, autoStop: true, config: { source: "pbx_or_diago" } },
  models: { id: "", name: "", kind: "openai-compatible", mode: "offline", endpoint: "", model: "", credentialRef: "", config: { audio_analysis: true, json_schema: true } },
} satisfies Record<Section, ConfigItem>

const optionLabels: Record<string, string> = {
  local: "本地存储",
  s3: "S3 对象存储",
  minio: "MinIO 对象存储",
  server: "服务端录音",
  browser: "浏览器录音",
  siprec: "SIPREC 录音",
  diago: "Diago 网关录音",
  realtime: "实时识别",
  offline: "离线分析",
  both: "实时和离线",
  "openai-compatible": "OpenAI 兼容接口",
  "azure-openai": "Azure OpenAI",
  custom: "自定义接口",
}

function optionLabel(value: string) {
  return optionLabels[value] || value
}

export function PlatformConfigManager({ section }: Props) {
  const [token, setToken] = useState("")
  const [items, setItems] = useState<ConfigItem[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [draft, setDraft] = useState<ConfigItem>(empty[section])
  const [message, setMessage] = useState("正在连接配置 API...")
  const current = meta[section]

  async function authed<T>(path: string, accessToken = token, init?: RequestInit): Promise<T> {
    requireSession()
    const response = await fetch(`${apiBase}${path}`, {
      ...init,
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        ...(init?.headers || {}),
      },
    })
    if (!response.ok) throw new Error(await response.text())
    return response.json()
  }

  async function load() {
    try {
      requireSession()
      setItems(await authed<ConfigItem[]>(current.path))
      setMessage("")
    } catch (error) {
      setMessage(`配置 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function save() {
    try {
      const method = selectedId ? "PUT" : "POST"
      const path = selectedId ? `${current.path}/${selectedId}` : current.path
      const saved = await authed<ConfigItem>(path, token, { method, body: JSON.stringify(draft) })
      setItems(selectedId ? items.map((item) => item.id === selectedId ? saved : item) : [saved, ...items])
      setSelectedId(saved.id)
      setDraft(saved)
      setMessage("配置已保存")
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function remove(id: string) {
    const item = items.find((entry) => entry.id === id)
    if (!item || !window.confirm(`删除「${item.name}」？`)) return
    try {
      await authed<ConfigItem>(`${current.path}/${id}`, token, { method: "DELETE" })
      setItems(items.filter((entry) => entry.id !== id))
      if (selectedId === id) {
        setSelectedId("")
        setDraft(empty[section])
      }
      setMessage("配置已删除")
    } catch (error) {
      setMessage(`删除失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  useEffect(() => {
    setDraft(empty[section])
    load()
  }, [section])

  return (
    <section className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_420px]">
      <article className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between border-b border-line p-4">
          <h2 className="text-sm font-semibold">{current.title}</h2>
          <button className="rounded-lg border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => { setSelectedId(""); setDraft(empty[section]); }}>{current.create}</button>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid gap-3 p-4">
          {items.map((item) => (
            <div key={item.id} className={`cursor-pointer rounded-lg border p-3 text-sm ${item.id === selectedId ? "border-primary bg-blue-50" : "border-line"}`} onClick={() => { setSelectedId(item.id); setDraft(item); }}>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="font-medium">{item.name}</div>
                  <div className="mt-1 text-muted">{summary(section, item)}</div>
                  <div className="mt-1 text-xs text-muted">{item.id}</div>
                </div>
                <button className="rounded-lg border border-red-200 px-3 py-1.5 text-xs font-medium text-red-600 hover:bg-red-50" onClick={(event) => { event.stopPropagation(); remove(item.id); }}>删除</button>
              </div>
            </div>
          ))}
        </div>
      </article>

      <aside className="rounded-lg border border-line bg-surface p-4">
        <h2 className="text-sm font-semibold">{selectedId ? "编辑配置" : current.create}</h2>
        <div className="mt-4 grid gap-3 text-sm">
          {renderFields(section, draft, setDraft)}
          <label className="grid gap-1">
            <span className="text-muted">扩展配置 JSON</span>
            <textarea className="min-h-28 rounded-lg border border-line px-3 py-2 font-mono text-xs" value={JSON.stringify((draft as any).config || {}, null, 2)} onChange={(event) => {
              try {
                setDraft({ ...draft, config: JSON.parse(event.target.value) } as ConfigItem)
              } catch {
                setDraft(draft)
              }
            }} />
          </label>
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={save}>保存配置</button>
        </div>
      </aside>
    </section>
  )
}

function summary(section: Section, item: ConfigItem) {
  if (section === "sip") {
    const entry = item as SipEndpoint
    return `${entry.domain || "-"} · ${entry.wssUrl || "-"}`
  }
  if (section === "storage") {
    const entry = item as StorageConfig
    return `${optionLabel(entry.kind)} · ${entry.bucket || entry.basePath || entry.endpoint || "-"}`
  }
  if (section === "recording") {
    const entry = item as RecordingConfig
    return `${optionLabel(entry.mode)} · ${entry.format} · 保留 ${entry.retentionDays} 天`
  }
  const entry = item as ModelProvider
  return `${optionLabel(entry.mode)} · ${optionLabel(entry.kind)} · ${entry.model || "-"}`
}

function renderFields(section: Section, draft: ConfigItem, setDraft: (item: ConfigItem) => void) {
  if (section === "sip") {
    const item = draft as SipEndpoint
    return <>
      <Text label="名称" value={item.name} onChange={(value) => setDraft({ ...item, name: value })} />
      <Text label="WSS 地址" value={item.wssUrl} onChange={(value) => setDraft({ ...item, wssUrl: value })} />
      <Text label="Domain" value={item.domain} onChange={(value) => setDraft({ ...item, domain: value })} />
      <Text label="Proxy" value={item.proxy} onChange={(value) => setDraft({ ...item, proxy: value })} />
    </>
  }
  if (section === "storage") {
    const item = draft as StorageConfig
    return <>
      <Text label="名称" value={item.name} onChange={(value) => setDraft({ ...item, name: value })} />
      <Select label="类型" value={item.kind} options={["local", "s3", "minio"]} onChange={(value) => setDraft({ ...item, kind: value })} />
      <Text label="Endpoint" value={item.endpoint || ""} onChange={(value) => setDraft({ ...item, endpoint: value })} />
      <Text label="Bucket" value={item.bucket || ""} onChange={(value) => setDraft({ ...item, bucket: value })} />
      <Text label="本地路径" value={item.basePath || ""} onChange={(value) => setDraft({ ...item, basePath: value })} />
      <Text label="访问 Base URI" value={item.baseUri || ""} onChange={(value) => setDraft({ ...item, baseUri: value })} />
      <Text label="凭据引用" value={item.credentialRef || ""} onChange={(value) => setDraft({ ...item, credentialRef: value })} />
    </>
  }
  if (section === "recording") {
    const item = draft as RecordingConfig
    return <>
      <Text label="名称" value={item.name} onChange={(value) => setDraft({ ...item, name: value })} />
      <Select label="录音方式" value={item.mode} options={["server", "browser", "siprec", "diago"]} onChange={(value) => setDraft({ ...item, mode: value })} />
      <Text label="存储配置 ID" value={item.storageConfigId} onChange={(value) => setDraft({ ...item, storageConfigId: value })} />
      <Text label="格式" value={item.format} onChange={(value) => setDraft({ ...item, format: value })} />
      <NumberField label="保留天数" value={item.retentionDays} onChange={(value) => setDraft({ ...item, retentionDays: value })} />
      <label className="flex items-center gap-2"><input type="checkbox" checked={item.autoStart} onChange={(event) => setDraft({ ...item, autoStart: event.target.checked })} />接通自动录音</label>
      <label className="flex items-center gap-2"><input type="checkbox" checked={item.autoStop} onChange={(event) => setDraft({ ...item, autoStop: event.target.checked })} />挂断自动停止</label>
    </>
  }
  const item = draft as ModelProvider
  return <>
    <Text label="名称" value={item.name} onChange={(value) => setDraft({ ...item, name: value })} />
    <Select label="用途" value={item.mode} options={["realtime", "offline", "both"]} onChange={(value) => setDraft({ ...item, mode: value })} />
    <Select label="类型" value={item.kind} options={["openai-compatible", "azure-openai", "local", "custom"]} onChange={(value) => setDraft({ ...item, kind: value })} />
    <Text label="Endpoint" value={item.endpoint} onChange={(value) => setDraft({ ...item, endpoint: value })} />
    <Text label="模型" value={item.model} onChange={(value) => setDraft({ ...item, model: value })} />
    <Text label="凭据引用" value={item.credentialRef || ""} onChange={(value) => setDraft({ ...item, credentialRef: value })} />
  </>
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input className="rounded-lg border border-line px-3 py-2" value={value} onChange={(event) => onChange(event.target.value)} /></label>
}

function NumberField({ label, value, onChange }: { label: string; value: number; onChange: (value: number) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><input type="number" className="rounded-lg border border-line px-3 py-2" value={value} onChange={(event) => onChange(Number(event.target.value))} /></label>
}

function Select({ label, value, options, onChange }: { label: string; value: string; options: string[]; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-muted">{label}</span><select className="rounded-lg border border-line px-3 py-2" value={value} onChange={(event) => onChange(event.target.value)}>{options.map((option) => <option key={option} value={option}>{optionLabel(option)}</option>)}</select></label>
}
