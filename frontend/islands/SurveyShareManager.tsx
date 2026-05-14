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
interface Indicator { id: string; projectId?: string; targetType: string; level: number; parentId?: string; name: string; serviceStage?: string; serviceNode?: string; questionId?: string; weight: number; includeTotalScore: boolean; nationalDimension?: string; includeNational: boolean; enabled: boolean }
interface IndicatorQuestion { id: string; projectId?: string; indicatorId: string; formTemplateId: string; questionId: string; questionLabel?: string; scoreDirection: string; weight: number }
interface CleaningRule { id: string; projectId?: string; name: string; ruleType: string; enabled: boolean; config?: Record<string, unknown>; action: string }
interface Issue { id: string; projectId?: string; submissionId?: string; indicatorId?: string; title: string; source: string; responsibleDepartment?: string; responsiblePerson?: string; severity: string; suggestion?: string; measure?: string; materialUrls?: string[]; verificationResult?: string; status: string; dueDate?: string; closedAt?: string }
interface IssueEvent { id: string; issueId: string; action: string; fromStatus?: string; toStatus?: string; content?: string; attachments?: string[]; actorId?: string; createdAt: string }

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
  const [indicators, setIndicators] = useState<Indicator[]>([])
  const [indicatorQuestions, setIndicatorQuestions] = useState<IndicatorQuestion[]>([])
  const [cleaningRules, setCleaningRules] = useState<CleaningRule[]>([])
  const [issues, setIssues] = useState<Issue[]>([])
  const [issueEvents, setIssueEvents] = useState<IssueEvent[]>([])
  const [draft, setDraft] = useState<Project>(emptyProject)
  const [indicatorDraft, setIndicatorDraft] = useState<Indicator>({ id: "", targetType: "outpatient", level: 1, name: "", serviceStage: "", serviceNode: "", questionId: "", weight: 1, includeTotalScore: true, nationalDimension: "", includeNational: false, enabled: true })
  const [bindingDraft, setBindingDraft] = useState<IndicatorQuestion>({ id: "", indicatorId: "", formTemplateId: "", questionId: "", questionLabel: "", scoreDirection: "positive", weight: 1 })
  const [ruleDraft, setRuleDraft] = useState<CleaningRule>({ id: "", name: "", ruleType: "duration", enabled: true, config: { minSeconds: 20 }, action: "mark_suspicious" })
  const [issueDraft, setIssueDraft] = useState<Issue>({ id: "", title: "", source: "manual", severity: "medium", status: "open", responsibleDepartment: "", responsiblePerson: "", suggestion: "" })
  const [eventDraft, setEventDraft] = useState({ action: "assign", toStatus: "assigned", content: "", attachments: "" })
  const [selectedProjectId, setSelectedProjectId] = useState("")
  const [channelDraft, setChannelDraft] = useState({ channel: "web", title: "" })
  const [activeTab, setActiveTab] = useState<"config" | "channels" | "answers">("config")
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
    const [nextIndicators, nextIssues] = await Promise.all([
      authedJson<Indicator[]>(`/api/v1/satisfaction/indicators${suffix}`),
      authedJson<Issue[]>(`/api/v1/satisfaction/issues${suffix}`),
    ])
    const [nextBindings, nextRules] = await Promise.all([
      authedJson<IndicatorQuestion[]>(`/api/v1/satisfaction/indicator-questions${suffix}`),
      authedJson<CleaningRule[]>(`/api/v1/satisfaction/cleaning-rules${suffix}`),
    ])
    setSubmissions(nextSubmissions)
    setStats(nextStats)
    setIndicators(nextIndicators)
    setIndicatorQuestions(nextBindings)
    setCleaningRules(nextRules)
    setIssues(nextIssues)
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

  async function saveIndicator() {
    const saved = await authedJson<Indicator>(indicatorDraft.id ? `/api/v1/satisfaction/indicators/${indicatorDraft.id}` : "/api/v1/satisfaction/indicators", { method: indicatorDraft.id ? "PUT" : "POST", body: JSON.stringify({ ...indicatorDraft, projectId: selectedProjectId }) })
    setIndicatorDraft({ id: "", targetType: selectedProject?.targetType || "outpatient", level: 1, name: "", serviceStage: "", serviceNode: "", questionId: "", weight: 1, includeTotalScore: true, nationalDimension: "", includeNational: false, enabled: true })
    setIndicators(indicators.some((item) => item.id === saved.id) ? indicators.map((item) => item.id === saved.id ? saved : item) : [...indicators, saved])
  }

  async function saveBinding() {
    const saved = await authedJson<IndicatorQuestion>(bindingDraft.id ? `/api/v1/satisfaction/indicator-questions/${bindingDraft.id}` : "/api/v1/satisfaction/indicator-questions", { method: bindingDraft.id ? "PUT" : "POST", body: JSON.stringify({ ...bindingDraft, projectId: selectedProjectId, formTemplateId: bindingDraft.formTemplateId || selectedProject?.formTemplateId || draft.formTemplateId }) })
    setBindingDraft({ id: "", indicatorId: "", formTemplateId: selectedProject?.formTemplateId || draft.formTemplateId || "", questionId: "", questionLabel: "", scoreDirection: "positive", weight: 1 })
    setIndicatorQuestions(indicatorQuestions.some((item) => item.id === saved.id) ? indicatorQuestions.map((item) => item.id === saved.id ? saved : item) : [...indicatorQuestions, saved])
  }

  async function saveRule() {
    const saved = await authedJson<CleaningRule>(ruleDraft.id ? `/api/v1/satisfaction/cleaning-rules/${ruleDraft.id}` : "/api/v1/satisfaction/cleaning-rules", { method: ruleDraft.id ? "PUT" : "POST", body: JSON.stringify({ ...ruleDraft, projectId: selectedProjectId }) })
    setRuleDraft({ id: "", name: "", ruleType: "duration", enabled: true, config: { minSeconds: 20 }, action: "mark_suspicious" })
    setCleaningRules(cleaningRules.some((item) => item.id === saved.id) ? cleaningRules.map((item) => item.id === saved.id ? saved : item) : [...cleaningRules, saved])
  }

  async function saveIssue() {
    const saved = await authedJson<Issue>(issueDraft.id ? `/api/v1/satisfaction/issues/${issueDraft.id}` : "/api/v1/satisfaction/issues", { method: issueDraft.id ? "PUT" : "POST", body: JSON.stringify({ ...issueDraft, projectId: selectedProjectId }) })
    setIssueDraft({ id: "", title: "", source: "manual", severity: "medium", status: "open", responsibleDepartment: "", responsiblePerson: "", suggestion: "" })
    setIssues(issues.some((item) => item.id === saved.id) ? issues.map((item) => item.id === saved.id ? saved : item) : [saved, ...issues])
  }

  async function generateIssues() {
    const created = await authedJson<Issue[]>(`/api/v1/satisfaction/issues/generate${selectedProjectId ? `?projectId=${selectedProjectId}` : ""}`, { method: "POST" })
    setIssues([...created, ...issues])
  }

  async function openIssue(issue: Issue) {
    setIssueDraft(issue)
    setIssueEvents(await authedJson<IssueEvent[]>(`/api/v1/satisfaction/issues/${issue.id}/events`))
  }

  async function addIssueEvent() {
    if (!issueDraft.id) return setMessage("请先选择问题")
    const saved = await authedJson<IssueEvent>(`/api/v1/satisfaction/issues/${issueDraft.id}/events`, { method: "POST", body: JSON.stringify({ ...eventDraft, attachments: eventDraft.attachments.split(",").map((item) => item.trim()).filter(Boolean) }) })
    setIssueEvents([saved, ...issueEvents])
    setIssueDraft({ ...issueDraft, status: eventDraft.toStatus || issueDraft.status })
    setEventDraft({ action: "assign", toStatus: "assigned", content: "", attachments: "" })
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
          {!projects.length && !message && <div className="rounded-lg border border-dashed border-line bg-gray-50 p-4 text-sm text-muted">暂无满意度项目。先绑定问卷或访谈表单，再生成采集渠道；最终分析报告在报表模块统一查看。</div>}
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
          <div><h2 className="text-base font-semibold">{selectedProject?.name || draft.name || "新建满意度项目"}</h2><p className="mt-1 text-sm text-muted">项目负责绑定采集表单、生成渠道和答卷审核；分析报告统一进入报表模块。</p></div>
          <div className="flex rounded-lg border border-line bg-gray-50 p-1 text-sm">
            {(["config", "channels", "answers"] as const).map((tab) => <button key={tab} className={`rounded-md px-3 py-1.5 ${activeTab === tab ? "bg-white text-primary shadow-sm" : "text-muted hover:text-ink"}`} onClick={() => setActiveTab(tab)}>{tabLabel(tab)}</button>)}
          </div>
        </div>

        {activeTab === "config" && <ConfigTab draft={draft} templates={templates} setDraft={setDraft} saveProject={saveProject} />}
        {activeTab === "channels" && <ChannelsTab shares={projectShares} selectedProject={selectedProject} channelDraft={channelDraft} setChannelDraft={setChannelDraft} createShare={createShare} />}
        {activeTab === "answers" && <AnswersTab submissions={submissions} openDetail={openDetail} audit={audit} />}
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
      <label className="grid gap-1"><span className="text-sm font-medium text-muted">问卷 / 访谈采集模板</span><select className="h-10 rounded-lg border border-line px-3 text-sm" value={draft.formTemplateId} onChange={(e) => setDraft({ ...draft, formTemplateId: e.target.value })}>{templates.map((template) => <option key={template.id} value={template.id}>{template.label}</option>)}</select></label>
      <Text label="开始日期" type="date" value={draft.startDate || ""} onChange={(v) => setDraft({ ...draft, startDate: v })} />
      <Text label="结束日期" type="date" value={draft.endDate || ""} onChange={(v) => setDraft({ ...draft, endDate: v })} />
      <Text label="目标样本数" type="number" value={String(draft.targetSampleSize || "")} onChange={(v) => setDraft({ ...draft, targetSampleSize: Number(v) })} />
      <Toggle label="允许匿名" checked={draft.anonymous} onChange={(v) => setDraft({ ...draft, anonymous: v, requiresVerification: !v || draft.requiresVerification })} />
      <Toggle label="患者验证" checked={draft.requiresVerification} onChange={(v) => setDraft({ ...draft, requiresVerification: v, anonymous: v ? false : draft.anonymous })} />
    </div>
    <div className="mt-4 rounded-lg border border-line bg-gray-50 p-4 text-sm text-muted">
      指标绑定、满意度分析和整改报告不在项目页内散落展示；项目答卷入库后，统一到 <a className="font-medium text-primary" href="/reports">分析报表</a> 中和评价投诉报表并列查看。
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

function AnalysisTab({ stats, indicators, indicatorQuestions, cleaningRules, issues, issueEvents, indicatorDraft, bindingDraft, ruleDraft, issueDraft, eventDraft, setIndicatorDraft, setBindingDraft, setRuleDraft, setIssueDraft, setEventDraft, saveIndicator, saveBinding, saveRule, saveIssue, openIssue, addIssueEvent, generateIssues, project, submissions }: { stats: Stats | null; indicators: Indicator[]; indicatorQuestions: IndicatorQuestion[]; cleaningRules: CleaningRule[]; issues: Issue[]; issueEvents: IssueEvent[]; indicatorDraft: Indicator; bindingDraft: IndicatorQuestion; ruleDraft: CleaningRule; issueDraft: Issue; eventDraft: { action: string; toStatus: string; content: string; attachments: string }; setIndicatorDraft: (value: Indicator) => void; setBindingDraft: (value: IndicatorQuestion) => void; setRuleDraft: (value: CleaningRule) => void; setIssueDraft: (value: Issue) => void; setEventDraft: (value: { action: string; toStatus: string; content: string; attachments: string }) => void; saveIndicator: () => void; saveBinding: () => void; saveRule: () => void; saveIssue: () => void; openIssue: (issue: Issue) => void; addIssueEvent: () => void; generateIssues: () => void; project?: Project; submissions: Submission[] }) {
  const channels = Object.entries(stats?.byChannel || {})
  const reasons = Object.entries(stats?.lowReasons || {}).sort((a, b) => b[1] - a[1]).slice(0, 8)
  return <section className="grid gap-4 p-4">
    <div className="rounded-lg border border-primary/20 bg-blue-50 p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="text-base font-semibold text-ink">二期分析报告工作区</div>
          <div className="mt-1 text-sm text-muted">指标体系、满意度看板、数据清洗、问题台账和报表预览合并在同一页，按当前满意度项目联动。</div>
        </div>
        <div className="rounded-full bg-white px-3 py-1 text-xs font-medium text-primary">指标 / 看板 / 报表 / 整改</div>
      </div>
    </div>
    {!project && <div className="rounded-lg border border-dashed border-line p-4 text-sm text-muted">
      当前还没有选中项目。二期能力已经在这里聚合展示：保存项目并收到答卷后，会自动填充总分、科室排名、指标得分、低分原因、渠道分布，并可从低分答卷生成整改问题。
    </div>}
    <div className="grid gap-3 md:grid-cols-4">
      <PhaseCard title="指标体系" text={`指标 ${indicators.length} 个，题目绑定 ${indicatorQuestions.length} 条，支持树级、权重和国考维度。`} />
      <PhaseCard title="数据清洗" text={`清洗规则 ${cleaningRules.length} 条，答卷 ${submissions.length} 份，支持可疑、有效、无效审核。`} />
      <PhaseCard title="问题台账" text={`问题 ${issues.length} 条，未关闭 ${issues.filter((item) => item.status !== "closed").length} 条，可分派、整改、验证。`} />
      <PhaseCard title="整改闭环" text="问题事件记录分派、整改措施、材料、验证结果和关闭动作。" />
    </div>
    <div className="grid gap-3 md:grid-cols-5"><MetricBox label="答卷数" value={String(stats?.total || 0)} /><MetricBox label="总分均值" value={stats?.scoreAverage ? stats.scoreAverage.toFixed(1) : "-"} /><MetricBox label="有效" value={String(stats?.valid || 0)} /><MetricBox label="待审核" value={String(stats?.pending || 0)} /><MetricBox label="可疑" value={String(stats?.suspicious || 0)} /></div>
    <div className="grid gap-4 xl:grid-cols-2">
      <Rank title="科室排名" items={stats?.departmentRanking || []} />
      <Rank title="指标得分" items={stats?.indicatorScores || []} />
      <BarList title="渠道分布" items={channels.map(([name, count]) => ({ name: channelLabels[name] || name, value: count }))} />
      <BarList title="低分原因" items={reasons.map(([name, count]) => ({ name, value: count }))} />
    </div>
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">指标体系与题目绑定</h3>
        <div className="mt-3 grid gap-2">{indicators.map((item) => <button key={item.id} className="rounded-lg border border-line p-3 text-left hover:border-primary" onClick={() => setIndicatorDraft(item)}><div className="font-medium">{item.name}</div><div className="mt-1 text-xs text-muted">L{item.level} · 题目 {item.questionId || "-"} · 权重 {item.weight} · {item.nationalDimension || "未映射国考维度"}</div></button>)}</div>
      </div>
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">{indicatorDraft.id ? "编辑指标" : "新增指标"}</h3>
        <div className="mt-3 grid gap-3 text-sm">
          <Text label="指标名称" value={indicatorDraft.name} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, name: v })} />
          <div className="grid grid-cols-2 gap-2"><Select label="场景" value={indicatorDraft.targetType} options={Object.keys(targetLabels)} labels={targetLabels} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, targetType: v })} /><Text label="服务环节" value={indicatorDraft.serviceStage || ""} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, serviceStage: v })} /></div>
          <Text label="服务节点" value={indicatorDraft.serviceNode || ""} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, serviceNode: v })} />
          <Text label="绑定题目 ID" value={indicatorDraft.questionId || ""} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, questionId: v })} />
          <div className="grid grid-cols-2 gap-2"><Text label="层级" type="number" value={String(indicatorDraft.level)} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, level: Number(v) })} /><Text label="权重" type="number" value={String(indicatorDraft.weight)} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, weight: Number(v) })} /></div>
          <Text label="国考维度" value={indicatorDraft.nationalDimension || ""} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, nationalDimension: v })} />
          <div className="grid grid-cols-2 gap-2"><Toggle label="纳入总分" checked={indicatorDraft.includeTotalScore} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, includeTotalScore: v })} /><Toggle label="纳入国考" checked={indicatorDraft.includeNational} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, includeNational: v })} /></div>
          <Toggle label="启用指标" checked={indicatorDraft.enabled} onChange={(v) => setIndicatorDraft({ ...indicatorDraft, enabled: v })} />
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={saveIndicator}>保存指标</button>
        </div>
      </div>
    </div>
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">问卷题目绑定指标</h3>
        <div className="mt-3 grid gap-2">{indicatorQuestions.map((item) => <button key={item.id} className="rounded-lg border border-line p-3 text-left hover:border-primary" onClick={() => setBindingDraft(item)}><div className="font-medium">{item.questionLabel || item.questionId}</div><div className="mt-1 text-xs text-muted">{item.formTemplateId} · {indicators.find((indicator) => indicator.id === item.indicatorId)?.name || item.indicatorId} · 权重 {item.weight}</div></button>)}</div>
      </div>
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">{bindingDraft.id ? "编辑绑定" : "新增绑定"}</h3>
        <div className="mt-3 grid gap-3 text-sm">
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">满意度指标</span><select className="h-10 rounded-lg border border-line px-3 text-sm" value={bindingDraft.indicatorId} onChange={(e) => setBindingDraft({ ...bindingDraft, indicatorId: e.target.value })}><option value="">请选择</option>{indicators.map((item) => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>
          <Text label="表单模板 ID" value={bindingDraft.formTemplateId || project?.formTemplateId || ""} onChange={(v) => setBindingDraft({ ...bindingDraft, formTemplateId: v })} />
          <Text label="题目 ID" value={bindingDraft.questionId} onChange={(v) => setBindingDraft({ ...bindingDraft, questionId: v })} />
          <Text label="题目名称" value={bindingDraft.questionLabel || ""} onChange={(v) => setBindingDraft({ ...bindingDraft, questionLabel: v })} />
          <div className="grid grid-cols-2 gap-2"><Select label="计分方向" value={bindingDraft.scoreDirection} options={["positive", "negative"]} labels={{ positive: "正向", negative: "反向" }} onChange={(v) => setBindingDraft({ ...bindingDraft, scoreDirection: v })} /><Text label="权重" type="number" value={String(bindingDraft.weight)} onChange={(v) => setBindingDraft({ ...bindingDraft, weight: Number(v) })} /></div>
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={saveBinding}>保存绑定</button>
        </div>
      </div>
    </div>
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">清洗规则配置</h3>
        <div className="mt-3 grid gap-2">{cleaningRules.map((item) => <button key={item.id} className="rounded-lg border border-line p-3 text-left hover:border-primary" onClick={() => setRuleDraft(item)}><div className="font-medium">{item.name}</div><div className="mt-1 text-xs text-muted">{item.enabled ? "启用" : "停用"} · {item.ruleType} · {item.action} · {JSON.stringify(item.config || {})}</div></button>)}</div>
      </div>
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">{ruleDraft.id ? "编辑规则" : "新增规则"}</h3>
        <div className="mt-3 grid gap-3 text-sm">
          <Text label="规则名称" value={ruleDraft.name} onChange={(v) => setRuleDraft({ ...ruleDraft, name: v })} />
          <Select label="规则类型" value={ruleDraft.ruleType} options={["duration", "duplicate_project", "same_device", "same_option"]} labels={{ duration: "时长阈值", duplicate_project: "同项目重复", same_device: "同 IP/设备", same_option: "全同选项" }} onChange={(v) => setRuleDraft({ ...ruleDraft, ruleType: v })} />
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">规则配置 JSON</span><textarea className="min-h-20 rounded-lg border border-line px-3 py-2 font-mono text-xs" value={JSON.stringify(ruleDraft.config || {}, null, 2)} onChange={(e) => setRuleDraft({ ...ruleDraft, config: safeJSON(e.target.value, ruleDraft.config || {}) as Record<string, unknown> })} /></label>
          <div className="grid grid-cols-2 gap-2"><Select label="处理动作" value={ruleDraft.action} options={["mark_suspicious", "mark_invalid", "manual_review"]} labels={{ mark_suspicious: "标记可疑", mark_invalid: "标记无效", manual_review: "人工审核" }} onChange={(v) => setRuleDraft({ ...ruleDraft, action: v })} /><Toggle label="启用规则" checked={ruleDraft.enabled} onChange={(v) => setRuleDraft({ ...ruleDraft, enabled: v })} /></div>
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={saveRule}>保存规则</button>
        </div>
      </div>
    </div>
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="rounded-lg border border-line p-4">
        <div className="flex items-center justify-between gap-3"><h3 className="font-semibold">问题台账</h3><button className="rounded-lg border border-line px-3 py-2 text-sm text-primary" onClick={generateIssues}>从低分生成</button></div>
        <div className="mt-3 grid gap-2">{issues.map((item) => <button key={item.id} className="rounded-lg border border-line p-3 text-left hover:border-primary" onClick={() => openIssue(item)}><div className="font-medium">{item.title}</div><div className="mt-1 text-xs text-muted">{item.source} · {item.responsibleDepartment || "待分派"} · {item.severity} · {item.status}</div>{item.suggestion && <div className="mt-2 text-xs text-muted">{item.suggestion}</div>}</button>)}</div>
      </div>
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">{issueDraft.id ? "编辑问题" : "新增问题"}</h3>
        <div className="mt-3 grid gap-3 text-sm">
          <Text label="问题标题" value={issueDraft.title} onChange={(v) => setIssueDraft({ ...issueDraft, title: v })} />
          <Text label="责任科室" value={issueDraft.responsibleDepartment || ""} onChange={(v) => setIssueDraft({ ...issueDraft, responsibleDepartment: v })} />
          <Text label="责任人" value={issueDraft.responsiblePerson || ""} onChange={(v) => setIssueDraft({ ...issueDraft, responsiblePerson: v })} />
          <div className="grid grid-cols-2 gap-2"><Select label="严重程度" value={issueDraft.severity} options={["low", "medium", "high"]} labels={{ low: "低", medium: "中", high: "高" }} onChange={(v) => setIssueDraft({ ...issueDraft, severity: v })} /><Select label="状态" value={issueDraft.status} options={["open", "assigned", "improving", "verified", "closed"]} labels={{ open: "待分派", assigned: "已分派", improving: "整改中", verified: "待验证", closed: "已关闭" }} onChange={(v) => setIssueDraft({ ...issueDraft, status: v })} /></div>
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">整改建议</span><textarea className="min-h-24 rounded-lg border border-line px-3 py-2" value={issueDraft.suggestion || ""} onChange={(e) => setIssueDraft({ ...issueDraft, suggestion: e.target.value })} /></label>
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">整改措施</span><textarea className="min-h-20 rounded-lg border border-line px-3 py-2" value={issueDraft.measure || ""} onChange={(e) => setIssueDraft({ ...issueDraft, measure: e.target.value })} /></label>
          <Text label="验证结果" value={issueDraft.verificationResult || ""} onChange={(v) => setIssueDraft({ ...issueDraft, verificationResult: v })} />
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={saveIssue}>保存问题</button>
        </div>
      </div>
    </div>
    {issueDraft.id && <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">整改事件流</h3>
        <div className="mt-3 grid gap-2">{issueEvents.map((item) => <div key={item.id} className="rounded-lg border border-line p-3"><div className="text-sm font-medium">{item.action} · {item.fromStatus || "-"} → {item.toStatus || "-"}</div><div className="mt-1 text-xs text-muted">{item.content || "无说明"} · {new Date(item.createdAt).toLocaleString()}</div></div>)}</div>
      </div>
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">新增流转记录</h3>
        <div className="mt-3 grid gap-3 text-sm">
          <Select label="动作" value={eventDraft.action} options={["assign", "improve", "upload_material", "verify", "close", "reopen", "note"]} labels={{ assign: "分派", improve: "整改", upload_material: "上传材料", verify: "验证", close: "关闭", reopen: "重开", note: "备注" }} onChange={(v) => setEventDraft({ ...eventDraft, action: v })} />
          <Select label="流转到" value={eventDraft.toStatus} options={["open", "assigned", "improving", "verified", "closed"]} labels={{ open: "待分派", assigned: "已分派", improving: "整改中", verified: "待验证", closed: "已关闭" }} onChange={(v) => setEventDraft({ ...eventDraft, toStatus: v })} />
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">说明</span><textarea className="min-h-20 rounded-lg border border-line px-3 py-2" value={eventDraft.content} onChange={(e) => setEventDraft({ ...eventDraft, content: e.target.value })} /></label>
          <Text label="材料链接，逗号分隔" value={eventDraft.attachments} onChange={(v) => setEventDraft({ ...eventDraft, attachments: v })} />
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={addIssueEvent}>保存流转</button>
        </div>
      </div>
    </div>}
    <div className="rounded-lg border border-line p-4">
      <h3 className="font-semibold">报告预览</h3>
      <div className="mt-3 grid gap-3 text-sm leading-6 text-ink">
        <p>{project?.name || "当前项目"} 共回收 {stats?.total || 0} 份答卷，当前总分均值为 {stats?.scoreAverage ? stats.scoreAverage.toFixed(1) : "-"}，有效样本 {stats?.valid || 0} 份，可疑样本 {stats?.suspicious || 0} 份。</p>
        <p>主要短板集中在：{reasons.length ? reasons.map(([name]) => name).join("、") : "暂无明显低分原因"}。建议优先处理高频低分原因并分派至责任科室闭环整改。</p>
        <p>当前问题台账 {issues.length} 条，其中未关闭 {issues.filter((item) => item.status !== "closed").length} 条；后续可按整改完成情况复评对应指标变化。</p>
      </div>
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
function PhaseCard({ title, text }: { title: string; text: string }) { return <div className="rounded-lg border border-line bg-white p-4"><div className="font-semibold">{title}</div><div className="mt-2 text-sm leading-6 text-muted">{text}</div></div> }
function tabLabel(tab: string) { return ({ config: "配置", channels: "渠道", answers: "答卷" } as Record<string, string>)[tab] || tab }
function statusText(status: string) { return ({ draft: "草稿", active: "发布中", paused: "暂停", closed: "结束", archived: "归档" } as Record<string, string>)[status] || status }
function formatAnswer(value: unknown): string { return typeof value === "object" && value ? JSON.stringify(value) : String(value ?? "-") }
function safeJSON(value: string, fallback: unknown) { try { return JSON.parse(value) } catch { return fallback } }
