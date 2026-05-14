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
const kindLabel: Record<string, string> = { sms: "短信", wechat: "微信公众号", wework: "企业微信", mini_program: "微信小程序", qq: "QQ", web: "Web 链接" }
const kindHint: Record<string, string> = {
  sms: "使用阿里云短信 SDK 配置：AccessKey 凭据、签名、模板 Code 和模板变量。",
  wechat: "用于微信公众号模板消息，优先匹配患者 OpenID。",
  wework: "用于企业微信应用消息，适合院内员工、随访专员和患者企业微信场景。",
  mini_program: "用于微信小程序订阅消息或小程序入口跳转。",
  qq: "用于 QQ 触达入口，适合作为补充渠道。",
  web: "用于公开链接、二维码和平板调查入口。",
}
const channelFlow = [
  { title: "接口配置", text: "配置短信、微信、QQ、Web 或电话服务商参数。" },
  { title: "启用校验", text: "启用后才会进入项目可选渠道和患者触达动作。" },
  { title: "项目渠道", text: "项目生成渠道链接、二维码、短信或微信触达任务。" },
  { title: "发送回执", text: "记录发送、失败、回执和重试结果，形成闭环。" },
]
const providerOptions: Record<string, Array<{ value: string; label: string }>> = {
  sms: [{ value: "aliyun_sms", label: "阿里云短信 SDK" }],
  wechat: [{ value: "wechat_official", label: "微信公众号模板消息" }],
  wework: [{ value: "wework", label: "企业微信应用消息" }],
  mini_program: [{ value: "wechat_mini_program", label: "微信小程序订阅消息" }],
  qq: [{ value: "qq", label: "QQ 触达" }],
  web: [{ value: "web_link", label: "Web 链接" }],
}

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
    let config: Record<string, unknown>
    try {
      config = JSON.parse(configText || "{}")
    } catch {
      setMessage("高级参数不是合法 JSON，请检查后再保存")
      return
    }
    const saved = await authedJson<Channel>("/api/v1/integration-channels", { method: "POST", body: JSON.stringify({ ...draft, config }) })
    setDraft(saved)
    setConfigText(JSON.stringify(saved.config || {}, null, 2))
    setMessage("接口配置已保存")
    await load()
  }

  function edit(item: Channel) {
    setDraft(item)
    setConfigText(JSON.stringify(item.config || {}, null, 2))
  }

  function draftConfig() {
    try {
      const parsed = JSON.parse(configText || "{}")
      return typeof parsed === "object" && parsed ? parsed as Record<string, unknown> : {}
    } catch {
      return {}
    }
  }

  function updateConfig(key: string, value: string | boolean) {
    const next = { ...draftConfig(), [key]: value }
    setConfigText(JSON.stringify(next, null, 2))
  }

  function configValue(key: string) {
    const value = draftConfig()[key]
    return typeof value === "string" ? value : ""
  }

  function providerValue() {
    return configValue("provider") || providerOptions[draft.kind]?.[0]?.value || ""
  }

  useEffect(() => { load() }, [])

  return (
    <div className="grid gap-5">
      <section className="rounded-lg border border-line bg-surface p-4">
        <div>
          <h2 className="text-base font-semibold">通道配置闭环</h2>
          <p className="mt-1 text-sm text-muted">这里维护全局接口；项目只引用已启用通道，不在项目页重复配置服务商参数。</p>
        </div>
        <div className="mt-4 grid gap-3 md:grid-cols-4">
          {channelFlow.map((step, index) => <FlowStep key={step.title} index={index + 1} title={step.title} text={step.text} />)}
        </div>
      </section>
      <div className="grid gap-5 xl:grid-cols-[minmax(0,1fr)_440px]">
      <section className="rounded-lg border border-line bg-surface">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
          <div>
            <h2 className="text-base font-semibold">统一接口配置</h2>
            <p className="mt-1 text-sm text-muted">短信、微信、QQ 和 Web 统一在这里维护，项目渠道只做引用。</p>
          </div>
          <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => { setDraft(empty); setConfigText("{}") }}>新增接口</button>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid gap-3 p-4 md:grid-cols-2">
          {items.map((item) => <button key={item.id} className="rounded-lg border border-line p-4 text-left hover:border-primary" onClick={() => edit(item)}>
            <div className="flex items-center justify-between"><span className="font-medium">{item.name}</span><span className={`rounded-full px-2 py-1 text-xs ${item.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-muted"}`}>{item.enabled ? "启用" : "停用"}</span></div>
            <div className="mt-2 text-sm text-muted">{kindLabel[item.kind] || item.kind}</div>
            <div className="mt-1 text-xs leading-5 text-muted">{kindHint[item.kind] || "用于项目触达或采集入口。"}</div>
            <div className="mt-1 truncate text-xs text-muted">{item.endpoint || "-"}</div>
          </button>)}
          {!items.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-6 text-center text-sm text-muted md:col-span-2">还没有接口。先配置并启用后，项目渠道才能发送短信、微信或 QQ。</div>}
        </div>
      </section>
      <aside className="rounded-lg border border-line bg-surface p-4">
        <h2 className="text-base font-semibold">{draft.id ? "编辑接口" : "新增接口"}</h2>
        <p className="mt-1 text-sm text-muted">基础连接信息放在主表单，签名、模板号、回调等高级参数放到 JSON。</p>
        <div className="mt-4 grid gap-3 text-sm">
          <label className="grid gap-1"><span className="text-muted">接口类型</span><select className="rounded-lg border border-line px-3 py-2" value={draft.kind} onChange={(e) => {
            const kind = e.target.value
            const provider = providerOptions[kind]?.[0]?.value || kind
            setDraft({ ...draft, kind })
            setConfigText(JSON.stringify({ ...draftConfig(), provider }, null, 2))
          }}>{["sms", "wechat", "wework", "mini_program", "qq", "web"].map((kind) => <option key={kind} value={kind}>{kindLabel[kind]}</option>)}</select></label>
          <div className="rounded-lg bg-blue-50 px-3 py-2 text-xs leading-5 text-primary">{kindHint[draft.kind] || "选择接口类型后，项目渠道会按能力启用。"}</div>
          <label className="grid gap-1"><span className="text-muted">服务商 / SDK</span><select className="rounded-lg border border-line px-3 py-2" value={providerValue()} onChange={(e) => updateConfig("provider", e.target.value)}>{(providerOptions[draft.kind] || []).map((item) => <option key={item.value} value={item.value}>{item.label}</option>)}</select></label>
          <Text label="名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
          <Text label="Endpoint" value={draft.endpoint || ""} onChange={(v) => setDraft({ ...draft, endpoint: v })} />
          <Text label="AppID" value={draft.appId || ""} onChange={(v) => setDraft({ ...draft, appId: v })} />
          <Text label="凭据引用" value={draft.credentialRef || ""} onChange={(v) => setDraft({ ...draft, credentialRef: v })} />
          {draft.kind === "sms" && <div className="grid gap-3 rounded-lg border border-line bg-gray-50 p-3">
            <Text label="阿里云 Region" value={configValue("regionId") || "cn-hangzhou"} onChange={(v) => updateConfig("regionId", v)} />
            <Text label="短信签名 SignName" value={configValue("signName")} onChange={(v) => updateConfig("signName", v)} />
            <Text label="模板 TemplateCode" value={configValue("templateCode")} onChange={(v) => updateConfig("templateCode", v)} />
          </div>}
          {(draft.kind === "wechat" || draft.kind === "wework" || draft.kind === "mini_program") && <div className="grid gap-3 rounded-lg border border-line bg-gray-50 p-3">
            <Text label="模板 ID" value={configValue("templateId")} onChange={(v) => updateConfig("templateId", v)} />
            {draft.kind === "wework" && <Text label="企业微信 AgentID" value={configValue("agentId")} onChange={(v) => updateConfig("agentId", v)} />}
            {draft.kind !== "wework" && <Text label="小程序/公众号页面路径" value={configValue("pagePath")} onChange={(v) => updateConfig("pagePath", v)} />}
          </div>}
          {draft.kind === "sms" && <ProviderPreset title="阿里云短信 SDK 参数" text="高级参数建议包含 provider=aliyun_sms、regionId、signName、templateCode、templateParamKeys。credentialRef 指向 AccessKeyId/AccessKeySecret。" />}
          {draft.kind === "wechat" && <ProviderPreset title="微信公众号参数" text="高级参数建议包含 provider=wechat_official、templateId、urlField、dataFields。AppID/凭据引用对应公众号应用。" />}
          {draft.kind === "wework" && <ProviderPreset title="企业微信参数" text="高级参数建议包含 provider=wework、agentId、messageType。AppID/凭据引用对应 CorpID/Secret。" />}
          {draft.kind === "mini_program" && <ProviderPreset title="小程序参数" text="高级参数建议包含 provider=wechat_mini_program、templateId、pagePath、envVersion。" />}
          <label className="flex items-center gap-2"><input type="checkbox" checked={draft.enabled} onChange={(e) => setDraft({ ...draft, enabled: e.target.checked })} />启用</label>
          <label className="grid gap-1"><span className="text-muted">高级参数 JSON</span><textarea className="min-h-40 rounded-lg border border-line px-3 py-2 font-mono text-xs" value={configText} onChange={(e) => setConfigText(e.target.value)} /></label>
          <div className="rounded-lg border border-line bg-gray-50 px-3 py-2 text-xs leading-5 text-muted">保存并启用后，项目“渠道发布”会自动出现对应入口；发送结果进入触达任务和回执记录。</div>
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={save}>保存接口</button>
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

function ProviderPreset({ title, text }: { title: string; text: string }) {
  return <div className="rounded-lg border border-blue-100 bg-blue-50 px-3 py-2 text-xs leading-5 text-primary">
    <div className="font-medium">{title}</div>
    <div className="mt-1">{text}</div>
  </div>
}
