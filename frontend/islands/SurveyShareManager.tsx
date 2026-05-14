import { useEffect, useMemo, useState } from "react"
import QRCode from "qrcode"
import { authedJson } from "../lib/auth"

interface Template { id: string; label: string; scenario?: string }
interface Library { templates: Template[] }
interface Project { id: string; name: string; targetType: string; formTemplateId: string; startDate?: string; endDate?: string; targetSampleSize: number; actualSampleSize: number; anonymous: boolean; requiresVerification: boolean; status: string }
interface Share { id: string; projectId?: string; formTemplateId: string; title: string; channel: string; token: string; url: string }
interface AnswerItem { questionId: string; questionLabel: string; questionType: string; answer: unknown; score?: number }
interface Submission { id: string; projectId?: string; channel: string; patientId?: string; visitId?: string; qualityStatus: string; qualityReason?: string; durationSeconds: number; submittedAt: string; answers?: Record<string, unknown>; answerItems?: AnswerItem[] }
interface Stats { total: number; pending: number; suspicious: number; valid: number; scoreAverage: number; byChannel: Record<string, number>; departmentRanking: Array<{ name: string; score: number; count: number }>; indicatorScores: Array<{ name: string; score: number; count: number }>; lowReasons: Record<string, number> }

const origin = "http://127.0.0.1:4321"
const emptyProject: Project = { id: "", name: "", targetType: "outpatient", formTemplateId: "", startDate: "", endDate: "", targetSampleSize: 0, actualSampleSize: 0, anonymous: true, requiresVerification: false, status: "draft" }
const targetLabels: Record<string, string> = { outpatient: "门诊", emergency: "急诊", inpatient: "住院", discharge: "出院", physical: "体检", staff: "员工" }
const channelLabels: Record<string, string> = { web: "Web 链接", wechat: "微信", qq: "QQ", sms: "短信", qr: "二维码", tablet: "平板" }
const qualityLabels: Record<string, string> = { pending: "待审核", suspicious: "可疑", valid: "有效", invalid: "无效" }

export function SurveyShareManager() {
  const [templates, setTemplates] = useState<Template[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [shares, setShares] = useState<Share[]>([])
  const [submissions, setSubmissions] = useState<Submission[]>([])
  const [stats, setStats] = useState<Stats | null>(null)
  const [draft, setDraft] = useState<Project>(emptyProject)
  const [selectedProjectId, setSelectedProjectId] = useState("")
  const [channelDraft, setChannelDraft] = useState({ channel: "web", title: "" })
  const [activeTab, setActiveTab] = useState<"config" | "channels" | "answers" | "analysis">("config")
  const [detail, setDetail] = useState<Submission | null>(null)
  const [message, setMessage] = useState("正在加载满意度项目...")

  const selectedProject = useMemo(() => projects.find((item) => item.id === selectedProjectId), [projects, selectedProjectId])
  const projectShares = shares.filter((share) => !selectedProjectId || share.projectId === selectedProjectId)

  async function load(nextProjectId = selectedProjectId) {
    try {
      const [library, nextProjects, nextShares] = await Promise.all([
        authedJson<Library>("/api/v1/form-library"),
        authedJson<Project[]>("/api/v1/satisfaction/projects"),
        authedJson<Share[]>("/api/v1/survey-share-links"),
      ])
      setTemplates(library.templates || [])
      setProjects(nextProjects)
      setShares(nextShares)
      const activeId = nextProjectId || selectedProjectId || nextProjects[0]?.id || ""
      setSelectedProjectId(activeId)
      if (!draft.formTemplateId && library.templates?.[0]) setDraft({ ...emptyProject, formTemplateId: library.templates[0].id, name: library.templates[0].label })
      await loadProjectData(activeId)
      setMessage("")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "加载失败")
    }
  }

  async function loadProjectData(projectId: string) {
    const suffix = projectId ? `?projectId=${projectId}` : ""
    const [nextSubmissions, nextStats] = await Promise.all([
      authedJson<Submission[]>(`/api/v1/satisfaction/submissions${suffix}`),
      authedJson<Stats>(`/api/v1/satisfaction/stats${suffix}`),
    ])
    setSubmissions(nextSubmissions)
    setStats(nextStats)
  }

  async function saveProject() {
    const saved = await authedJson<Project>(draft.id ? `/api/v1/satisfaction/projects/${draft.id}` : "/api/v1/satisfaction/projects", { method: draft.id ? "PUT" : "POST", body: JSON.stringify(draft) })
    setDraft(saved)
    setMessage("满意度项目已保存")
    await load(saved.id)
  }

  async function createShare() {
    if (!selectedProject) return setMessage("请先选择或创建满意度项目")
    const created = await authedJson<Share>("/api/v1/survey-share-links", {
      method: "POST",
      body: JSON.stringify({
        projectId: selectedProject.id,
        formTemplateId: selectedProject.formTemplateId,
        title: channelDraft.title || selectedProject.name,
        channel: channelDraft.channel,
        config: { allowAnonymous: selectedProject.anonymous, requiresVerification: selectedProject.requiresVerification },
      }),
    })
    setShares([created, ...shares])
    setMessage("项目采集渠道已生成")
  }

  async function openDetail(id: string) {
    setDetail(await authedJson<Submission>(`/api/v1/satisfaction/submissions/${id}`))
  }

  async function audit(id: string, qualityStatus: string, qualityReason = "") {
    const updated = await authedJson<Submission>(`/api/v1/satisfaction/submissions/${id}/quality`, { method: "PUT", body: JSON.stringify({ qualityStatus, qualityReason }) })
    setSubmissions(submissions.map((item) => item.id === id ? { ...item, ...updated } : item))
    setDetail(detail?.id === id ? updated : detail)
    await loadProjectData(selectedProjectId)
  }

  function editProject(project: Project) {
    setDraft(project)
    setSelectedProjectId(project.id)
    loadProjectData(project.id)
  }

  useEffect(() => { load("") }, [])

  return (
    <div className="grid gap-5 xl:grid-cols-[340px_minmax(0,1fr)]">
      <aside className="rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between gap-3 border-b border-line p-4">
          <div><h2 className="text-base font-semibold">满意度项目</h2><p className="mt-1 text-sm text-muted">{projects.length} 个项目</p></div>
          <button className="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white" onClick={() => { setDraft({ ...emptyProject, formTemplateId: templates[0]?.id || "" }); setActiveTab("config") }}>新增</button>
        </div>
        {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        <div className="grid gap-2 p-3">
          {projects.map((project) => (
            <button key={project.id} className={`rounded-lg border p-3 text-left ${selectedProjectId === project.id ? "border-primary bg-blue-50" : "border-line hover:border-primary"}`} onClick={() => editProject(project)}>
              <div className="font-medium">{project.name}</div>
              <div className="mt-1 text-xs text-muted">{targetLabels[project.targetType] || project.targetType} · {statusText(project.status)} · {project.actualSampleSize}/{project.targetSampleSize || "-"}</div>
              <div className="mt-2 text-xs text-muted">{templates.find((item) => item.id === project.formTemplateId)?.label || project.formTemplateId}</div>
            </button>
          ))}
        </div>
      </aside>

      <main className="min-w-0 rounded-lg border border-line bg-surface">
        <div className="flex flex-wrap items-center justify-between gap-3 border-b border-line p-4">
          <div><h2 className="text-base font-semibold">{selectedProject?.name || draft.name || "新建满意度项目"}</h2><p className="mt-1 text-sm text-muted">配置、渠道、答卷、分析按项目闭环管理。</p></div>
          <div className="flex rounded-lg border border-line bg-gray-50 p-1 text-sm">
            {(["config", "channels", "answers", "analysis"] as const).map((tab) => <button key={tab} className={`rounded-md px-3 py-1.5 ${activeTab === tab ? "bg-white text-primary shadow-sm" : "text-muted hover:text-ink"}`} onClick={() => setActiveTab(tab)}>{tabLabel(tab)}</button>)}
          </div>
        </div>

        {activeTab === "config" && <ConfigTab draft={draft} templates={templates} setDraft={setDraft} saveProject={saveProject} />}
        {activeTab === "channels" && <ChannelsTab shares={projectShares} selectedProject={selectedProject} channelDraft={channelDraft} setChannelDraft={setChannelDraft} createShare={createShare} />}
        {activeTab === "answers" && <AnswersTab submissions={submissions} openDetail={openDetail} audit={audit} />}
        {activeTab === "analysis" && <AnalysisTab stats={stats} />}
      </main>

      {detail && <SubmissionDrawer detail={detail} audit={audit} onClose={() => setDetail(null)} />}
    </div>
  )
}

function ConfigTab({ draft, templates, setDraft, saveProject }: { draft: Project; templates: Template[]; setDraft: (project: Project) => void; saveProject: () => void }) {
  return <section className="p-4">
    <div className="flex justify-end"><button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={saveProject}>保存项目</button></div>
    <div className="mt-4 grid gap-3 md:grid-cols-3">
      <Text label="项目名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
      <Select label="调查对象" value={draft.targetType} options={Object.keys(targetLabels)} labels={targetLabels} onChange={(v) => setDraft({ ...draft, targetType: v })} />
      <Select label="项目状态" value={draft.status} options={["draft", "active", "paused", "closed", "archived"]} labels={{ draft: "草稿", active: "发布中", paused: "暂停", closed: "结束", archived: "归档" }} onChange={(v) => setDraft({ ...draft, status: v })} />
      <label className="grid gap-1"><span className="text-sm font-medium text-muted">问卷模板</span><select className="h-10 rounded-lg border border-line px-3 text-sm" value={draft.formTemplateId} onChange={(e) => setDraft({ ...draft, formTemplateId: e.target.value })}>{templates.map((template) => <option key={template.id} value={template.id}>{template.label}</option>)}</select></label>
      <Text label="开始日期" type="date" value={draft.startDate || ""} onChange={(v) => setDraft({ ...draft, startDate: v })} />
      <Text label="结束日期" type="date" value={draft.endDate || ""} onChange={(v) => setDraft({ ...draft, endDate: v })} />
      <Text label="目标样本数" type="number" value={String(draft.targetSampleSize || "")} onChange={(v) => setDraft({ ...draft, targetSampleSize: Number(v) })} />
      <Toggle label="允许匿名" checked={draft.anonymous} onChange={(v) => setDraft({ ...draft, anonymous: v, requiresVerification: !v || draft.requiresVerification })} />
      <Toggle label="患者验证" checked={draft.requiresVerification} onChange={(v) => setDraft({ ...draft, requiresVerification: v, anonymous: v ? false : draft.anonymous })} />
    </div>
  </section>
}

function ChannelsTab({ shares, selectedProject, channelDraft, setChannelDraft, createShare }: { shares: Share[]; selectedProject?: Project; channelDraft: { channel: string; title: string }; setChannelDraft: (value: { channel: string; title: string }) => void; createShare: () => void }) {
  return <section className="grid gap-4 p-4">
    <div className="grid gap-3 md:grid-cols-[1fr_1fr_auto]">
      <input className="h-10 rounded-lg border border-line px-3 text-sm" placeholder="渠道标题，默认使用项目名称" value={channelDraft.title} onChange={(e) => setChannelDraft({ ...channelDraft, title: e.target.value })} />
      <select className="h-10 rounded-lg border border-line px-3 text-sm" value={channelDraft.channel} onChange={(e) => setChannelDraft({ ...channelDraft, channel: e.target.value })}>{Object.entries(channelLabels).map(([id, label]) => <option key={id} value={id}>{label}</option>)}</select>
      <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white disabled:bg-gray-300" disabled={!selectedProject} onClick={createShare}>生成渠道</button>
    </div>
    <div className="grid gap-3 xl:grid-cols-2">
      {shares.map((share) => <ChannelCard key={share.id} share={share} />)}
    </div>
  </section>
}

function ChannelCard({ share }: { share: Share }) {
  const url = `${origin}${share.url}`
  const [svg, setSvg] = useState("")
  useEffect(() => {
    QRCode.toString(url, { type: "svg", width: 160, margin: 1, errorCorrectionLevel: "M" }).then(setSvg)
  }, [url])
  const dataUrl = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
  return <div className="rounded-lg border border-line p-4">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div><div className="font-medium">{share.title}</div><div className="mt-1 text-sm text-muted">{channelLabels[share.channel] || share.channel}</div></div>
      <div className="flex gap-2"><a className="rounded-lg border border-line px-3 py-2 text-sm text-primary" href={url} target="_blank">打开</a><a className="rounded-lg border border-line px-3 py-2 text-sm text-primary" href={dataUrl} download={`${share.title || "survey"}-qr.svg`}>下载二维码</a></div>
    </div>
    <div className="mt-3 grid gap-3 sm:grid-cols-[120px_minmax(0,1fr)]">
      <div className="rounded-lg border border-line bg-white p-2" dangerouslySetInnerHTML={{ __html: svg || "<svg viewBox='0 0 160 160'></svg>" }} />
      <div className="break-all rounded-lg bg-gray-50 px-3 py-2 font-mono text-xs text-muted">{url}</div>
    </div>
  </div>
}

function AnswersTab({ submissions, openDetail, audit }: { submissions: Submission[]; openDetail: (id: string) => void; audit: (id: string, status: string, reason?: string) => void }) {
  return <section className="overflow-x-auto">
    <table className="w-full text-sm">
      <thead className="bg-gray-50 text-muted"><tr><th className="px-4 py-3 text-left">提交时间</th><th className="px-4 py-3 text-left">渠道</th><th className="px-4 py-3 text-left">质量状态</th><th className="px-4 py-3 text-left">清洗原因</th><th className="px-4 py-3 text-left">时长</th><th className="px-4 py-3 text-left">总体满意</th><th className="px-4 py-3 text-right">操作</th></tr></thead>
      <tbody>{submissions.map((item) => <tr key={item.id} className="border-t border-line"><td className="px-4 py-3">{new Date(item.submittedAt).toLocaleString()}</td><td className="px-4 py-3">{channelLabels[item.channel] || item.channel}</td><td className="px-4 py-3">{qualityLabels[item.qualityStatus] || item.qualityStatus}</td><td className="px-4 py-3">{item.qualityReason || "-"}</td><td className="px-4 py-3">{item.durationSeconds}s</td><td className="px-4 py-3">{String(item.answers?.overall_satisfaction || "-")}</td><td className="px-4 py-3 text-right"><button className="mr-3 text-primary" onClick={() => openDetail(item.id)}>详情</button><button className="mr-3 text-primary" onClick={() => audit(item.id, "valid")}>有效</button><button className="text-red-600" onClick={() => audit(item.id, "invalid", "人工判定无效")}>无效</button></td></tr>)}</tbody>
    </table>
  </section>
}

function AnalysisTab({ stats }: { stats: Stats | null }) {
  const channels = Object.entries(stats?.byChannel || {})
  const reasons = Object.entries(stats?.lowReasons || {}).sort((a, b) => b[1] - a[1]).slice(0, 8)
  return <section className="grid gap-4 p-4">
    <div className="grid gap-3 md:grid-cols-5"><MetricBox label="答卷数" value={String(stats?.total || 0)} /><MetricBox label="总分均值" value={stats?.scoreAverage ? stats.scoreAverage.toFixed(1) : "-"} /><MetricBox label="有效" value={String(stats?.valid || 0)} /><MetricBox label="待审核" value={String(stats?.pending || 0)} /><MetricBox label="可疑" value={String(stats?.suspicious || 0)} /></div>
    <div className="grid gap-4 xl:grid-cols-2">
      <Rank title="科室排名" items={stats?.departmentRanking || []} />
      <Rank title="指标得分" items={stats?.indicatorScores || []} />
      <BarList title="渠道分布" items={channels.map(([name, count]) => ({ name: channelLabels[name] || name, value: count }))} />
      <BarList title="低分原因" items={reasons.map(([name, count]) => ({ name, value: count }))} />
    </div>
  </section>
}

function SubmissionDrawer({ detail, audit, onClose }: { detail: Submission; audit: (id: string, status: string, reason?: string) => void; onClose: () => void }) {
  const items = detail.answerItems?.length ? detail.answerItems : Object.entries(detail.answers || {}).map(([questionId, answer]) => ({ questionId, questionLabel: questionId, questionType: "", answer }))
  return <div className="fixed inset-0 z-50 grid justify-items-end bg-gray-900/40">
    <aside className="h-full w-full max-w-2xl overflow-y-auto bg-white shadow-xl">
      <div className="flex items-center justify-between border-b border-line p-4"><div><h2 className="font-semibold">答卷详情</h2><p className="mt-1 text-sm text-muted">{qualityLabels[detail.qualityStatus] || detail.qualityStatus} · {detail.qualityReason || "无清洗原因"}</p></div><button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={onClose}>关闭</button></div>
      <div className="grid gap-3 p-4 text-sm">
        <div className="grid gap-2 rounded-lg bg-gray-50 p-3 md:grid-cols-2"><Info label="提交时间" value={new Date(detail.submittedAt).toLocaleString()} /><Info label="渠道" value={channelLabels[detail.channel] || detail.channel} /><Info label="患者" value={detail.patientId || "-"} /><Info label="就诊" value={detail.visitId || "-"} /><Info label="答题时长" value={`${detail.durationSeconds}s`} /><Info label="状态" value={qualityLabels[detail.qualityStatus] || detail.qualityStatus} /></div>
        <div className="flex gap-2"><button className="rounded-lg bg-primary px-3 py-2 text-white" onClick={() => audit(detail.id, "valid")}>标记有效</button><button className="rounded-lg border border-red-200 px-3 py-2 text-red-600" onClick={() => audit(detail.id, "invalid", "人工判定无效")}>标记无效</button><button className="rounded-lg border border-line px-3 py-2" onClick={() => audit(detail.id, "suspicious", "人工复核可疑")}>标记可疑</button></div>
        <div className="grid gap-2">{items.map((item) => <div key={item.questionId} className="rounded-lg border border-line p-3"><div className="text-muted">{item.questionLabel}</div><div className="mt-1 font-medium">{formatAnswer(item.answer)}</div>{item.score !== undefined && <div className="mt-1 text-xs text-muted">得分：{item.score}</div>}</div>)}</div>
      </div>
    </aside>
  </div>
}

function Text({ label, value, type = "text", onChange }: { label: string; value: string; type?: string; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-sm font-medium text-muted">{label}</span><input type={type} className="h-10 rounded-lg border border-line px-3 text-sm" value={value} onChange={(e) => onChange(e.target.value)} /></label>
}
function Select({ label, value, options, labels, onChange }: { label: string; value: string; options: string[]; labels: Record<string, string>; onChange: (value: string) => void }) {
  return <label className="grid gap-1"><span className="text-sm font-medium text-muted">{label}</span><select className="h-10 rounded-lg border border-line px-3 text-sm" value={value} onChange={(e) => onChange(e.target.value)}>{options.map((option) => <option key={option} value={option}>{labels[option] || option}</option>)}</select></label>
}
function Toggle({ label, checked, onChange }: { label: string; checked: boolean; onChange: (value: boolean) => void }) {
  return <label className="flex h-10 items-center gap-2 rounded-lg border border-line px-3 text-sm"><input type="checkbox" checked={checked} onChange={(e) => onChange(e.target.checked)} />{label}</label>
}
function MetricBox({ label, value }: { label: string; value: string }) { return <div className="rounded-lg border border-line p-4"><div className="text-sm text-muted">{label}</div><div className="mt-2 text-2xl font-semibold">{value}</div></div> }
function Info({ label, value }: { label: string; value: string }) { return <div><span className="text-muted">{label}</span><div className="font-medium">{value}</div></div> }
function Rank({ title, items }: { title: string; items: Array<{ name: string; score: number; count: number }> }) { return <div className="rounded-lg border border-line p-4"><h3 className="font-semibold">{title}</h3><div className="mt-3 grid gap-2">{items.sort((a, b) => b.score - a.score).map((item) => <div key={item.name} className="flex items-center justify-between gap-3 text-sm"><span>{item.name}</span><span className="font-semibold">{item.score.toFixed(1)} <span className="text-xs text-muted">({item.count})</span></span></div>)}</div></div> }
function BarList({ title, items }: { title: string; items: Array<{ name: string; value: number }> }) { const max = Math.max(1, ...items.map((item) => item.value)); return <div className="rounded-lg border border-line p-4"><h3 className="font-semibold">{title}</h3><div className="mt-3 grid gap-3">{items.map((item) => <div key={item.name} className="text-sm"><div className="mb-1 flex justify-between"><span>{item.name}</span><span>{item.value}</span></div><div className="h-2 rounded-full bg-gray-100"><div className="h-full rounded-full bg-primary" style={{ width: `${(item.value / max) * 100}%` }} /></div></div>)}</div></div> }
function Metric({ label, value }: { label: string; value: string }) { return <div className="flex items-center justify-between border-b border-line pb-2 last:border-0"><span className="text-muted">{label}</span><span className="font-semibold">{value}</span></div> }
function tabLabel(tab: string) { return ({ config: "配置", channels: "渠道", answers: "答卷", analysis: "分析" } as Record<string, string>)[tab] || tab }
function statusText(status: string) { return ({ draft: "草稿", active: "发布中", paused: "暂停", closed: "结束", archived: "归档" } as Record<string, string>)[status] || status }
function formatAnswer(value: unknown): string { return typeof value === "object" && value ? JSON.stringify(value) : String(value ?? "-") }
