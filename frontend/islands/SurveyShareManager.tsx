import { useEffect, useMemo, useState } from "react"
import type { ReactNode } from "react"
import QRCode from "qrcode"
import { authedJson } from "../lib/auth"

interface Template { id: string; label: string; scenario?: string }
interface Library { templates: Template[] }
interface FormVersion { id: string; formId: string; version: number; published: boolean; createdAt: string }
interface ManagedForm { id: string; name: string; description?: string; status: string; currentVersionId?: string; versions?: FormVersion[] }
type ProjectCoreTab = "overview" | "lifecycle" | "people" | "tasks" | "data" | "modules" | "phases"
type ProjectModule = "channels" | "answers" | "indicators" | "cleaning" | "analysis" | "issues"
type ProjectTab = ProjectCoreTab | ProjectModule
interface Project { id: string; name: string; targetType: string; formTemplateId: string; startDate?: string; endDate?: string; targetSampleSize: number; actualSampleSize: number; anonymous: boolean; requiresVerification: boolean; status: string; config?: Record<string, unknown> }
interface Share { id: string; projectId?: string; formTemplateId: string; title: string; channel: string; token: string; url: string; config?: Record<string, unknown> }
interface Delivery { id: string; projectId?: string; shareId: string; channel: string; recipient: string; recipientName?: string; status: string; message?: string; error?: string; providerRef?: string; sentAt?: string; createdAt: string }
interface ChannelConfig { id: string; kind: string; name: string; endpoint?: string; appId?: string; credentialRef?: string; enabled: boolean; config?: Record<string, unknown> }
interface SipEndpoint { id: string; name: string; config?: Record<string, unknown> }
interface ChannelRecipient { patientId: string; patientNo?: string; name: string; channel: string; recipient: string; source: string; available: boolean; unavailable?: string }
interface AnswerItem { questionId: string; questionLabel: string; questionType: string; answer: unknown; score?: number }
interface Submission { id: string; projectId?: string; channel: string; patientId?: string; visitId?: string; qualityStatus: string; qualityReason?: string; durationSeconds: number; submittedAt: string; answers?: Record<string, unknown>; answerItems?: AnswerItem[] }
interface Stats { total: number; pending: number; suspicious: number; valid: number; invalid?: number; scoreAverage: number; byChannel: Record<string, number>; departmentRanking: Array<{ name: string; score: number; count: number }>; indicatorScores: Array<{ name: string; score: number; count: number }>; trend?: Array<{ name: string; score: number; count: number }>; periodCompare?: Array<{ name: string; score: number; count: number; mom?: number | null; yoy?: number | null }>; crossAnalysis?: Record<string, Array<{ name: string; score: number; count: number }>>; dimensionRankings?: Record<string, Array<{ name: string; score: number; count: number }>>; jobAnalysis?: Array<{ name: string; score: number; count: number }>; importanceMatrix?: Array<{ name: string; score: number; impact: number; count: number }>; shortBoards?: Array<{ dimension: string; name: string; score: number; count: number; reason: string }>; varianceAnalysis?: Array<{ dimension: string; variance: number; stddev: number; minName: string; minScore: number; maxName: string; maxScore: number; gap: number }>; correlation?: Array<{ name: string; coefficient: number; count: number }>; graphql?: boolean; aiInsights?: string[]; lowReasons: Record<string, number> }
interface Indicator { id: string; projectId?: string; targetType: string; level: number; parentId?: string; name: string; serviceStage?: string; serviceNode?: string; questionId?: string; weight: number; includeTotalScore: boolean; nationalDimension?: string; includeNational: boolean; enabled: boolean }
interface IndicatorQuestion { id: string; projectId?: string; indicatorId: string; formTemplateId: string; questionId: string; questionLabel?: string; scoreDirection: string; weight: number }
interface CleaningRule { id: string; projectId?: string; name: string; ruleType: string; enabled: boolean; config?: Record<string, unknown>; action: string }
interface Issue { id: string; projectId?: string; submissionId?: string; indicatorId?: string; title: string; source: string; responsibleDepartment?: string; responsiblePerson?: string; severity: string; suggestion?: string; measure?: string; materialUrls?: string[]; verificationResult?: string; status: string; dueDate?: string; closedAt?: string }
interface IssueEvent { id: string; issueId: string; action: string; fromStatus?: string; toStatus?: string; content?: string; attachments?: string[]; actorId?: string; createdAt: string }
type IssueEventDraft = { action: string; toStatus: string; content: string; attachments: string }

const origin = "http://127.0.0.1:4321"
const defaultModules: Record<ProjectModule, boolean> = { channels: true, answers: true, indicators: true, cleaning: true, analysis: true, issues: true }
const emptyProject: Project = { id: "", name: "", targetType: "outpatient", formTemplateId: "", startDate: "", endDate: "", targetSampleSize: 0, actualSampleSize: 0, anonymous: true, requiresVerification: false, status: "draft", config: { projectType: "satisfaction", phases: [], modules: defaultModules } }
const targetLabels: Record<string, string> = { outpatient: "门诊", emergency: "急诊", inpatient: "住院", discharge: "出院", physical: "体检", staff: "员工" }
const projectTypeLabels: Record<string, string> = { satisfaction: "满意度调查", followup: "随访访谈", complaint: "投诉评价", research: "科研采集", screening: "筛查登记", custom: "自定义项目" }
const moduleLabels: Record<ProjectModule, string> = { channels: "渠道发布", answers: "答卷入库", indicators: "指标体系", cleaning: "数据清洗", analysis: "分析看板", issues: "问题整改" }
const moduleDescriptions: Record<ProjectModule, string> = {
  channels: "生成 Web、二维码、短信、微信、电话等触达入口。",
  answers: "查看答卷详情、人工审核和入库状态。",
  indicators: "配置项目指标、题目绑定、权重和国考映射。",
  cleaning: "配置重复、时长、同设备、异常答案等质控规则。",
  analysis: "按项目维度查看统计、趋势、科室和指标分析。",
  issues: "从低分、投诉和负面意见形成整改闭环。",
}
const channelLabels: Record<string, string> = { web: "Web 链接", wechat: "微信公众号", wework: "企业微信", mini_program: "微信小程序", qq: "QQ", sms: "短信", phone: "电话", qr: "二维码", tablet: "平板" }
const qualityLabels: Record<string, string> = { pending: "待初审", suspicious: "可疑", level1_review: "一级复核", level2_review: "二级抽检", valid: "有效入库", invalid: "剔除无效" }
const cleaningRuleLabels: Record<string, string> = { duration: "时长阈值", duplicate_project: "同项目重复", same_device: "同 IP/设备", same_option: "全同选项", identity_required: "身份缺失", answer_completion: "答题完整度", investigator_required: "调查员缺失", sample_authenticity: "样本真实性", quota_control: "样本配额" }
const cleaningActionLabels: Record<string, string> = { mark_suspicious: "标记可疑", mark_invalid: "剔除无效", manual_review: "进入复核" }
const issueStatusLabels: Record<string, string> = { open: "待分派", assigned: "已分派", improving: "整改中", verified: "待复评", closed: "已关闭" }
const issueSeverityLabels: Record<string, string> = { low: "低", medium: "中", high: "高" }
const issueActionLabels: Record<string, string> = { assign: "分派", remind: "催办", improve: "整改", upload_material: "上传材料", verify: "复评", close: "关闭", reopen: "重开", note: "备注" }

type ProjectView = "list" | "new" | "edit" | "dashboard" | "module"

export function SurveyShareManager({ initialTab = "overview", view = "module" }: { initialTab?: ProjectTab; view?: ProjectView }) {
  const [templates, setTemplates] = useState<Template[]>([])
  const [forms, setForms] = useState<ManagedForm[]>([])
  const [projects, setProjects] = useState<Project[]>([])
  const [shares, setShares] = useState<Share[]>([])
  const [deliveries, setDeliveries] = useState<Delivery[]>([])
  const [channelConfigs, setChannelConfigs] = useState<ChannelConfig[]>([])
  const [sipEndpoints, setSipEndpoints] = useState<SipEndpoint[]>([])
  const [systemRecipients, setSystemRecipients] = useState<ChannelRecipient[]>([])
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
  const [channelDraft, setChannelDraft] = useState({ channel: "web", title: "", pointCode: "", pointName: "", location: "", scene: "general", tabletMode: false, kioskMode: false, dailyTarget: "" })
  const [deliveryDraft, setDeliveryDraft] = useState({ shareId: "", recipients: "", message: "" })
  const [recipientKeyword, setRecipientKeyword] = useState("")
  const [recipientMode, setRecipientMode] = useState<"system" | "manual">("system")
  const [activeTab, setActiveTab] = useState<ProjectTab>(initialTab)
  const [detail, setDetail] = useState<Submission | null>(null)
  const [message, setMessage] = useState("正在加载项目...")

  const selectedProject = useMemo(() => projects.find((item) => item.id === selectedProjectId), [projects, selectedProjectId])
  const projectShares = shares.filter((share) => !selectedProjectId || share.projectId === selectedProjectId)
  const selectedFormVersion = useMemo(() => publishedFormVersion(forms.find((form) => form.id === selectedProject?.formTemplateId)), [forms, selectedProject?.formTemplateId])
  const channelOptions = useMemo(() => {
    const enabled = new Set(channelConfigs.filter((item) => item.enabled).map((item) => item.kind))
    const options = ["web", "qr", "tablet", ...Array.from(enabled).filter((kind) => ["sms", "wechat", "wework", "mini_program", "qq"].includes(kind))]
    if (sipEndpoints.some((item) => item.config?.enabled === true)) options.push("phone")
    return Array.from(new Set(options))
  }, [channelConfigs, sipEndpoints])

  async function load(nextProjectId = selectedProjectId) {
    try {
      if (view === "list") {
        const [library, managedForms, nextProjects] = await Promise.all([
          authedJson<Library>("/api/v1/form-library"),
          authedJson<ManagedForm[]>("/api/v1/forms").catch(() => []),
          loadProjects(),
        ])
        setTemplates(library.templates || [])
        setForms(managedForms)
        setProjects(nextProjects)
        setMessage("")
        return
      }
      const [library, managedForms, nextProjects, nextShares, nextChannels, nextSipEndpoints] = await Promise.all([
        authedJson<Library>("/api/v1/form-library"),
        authedJson<ManagedForm[]>("/api/v1/forms").catch(() => []),
        loadProjects(),
        authedJson<Share[]>("/api/v1/survey-share-links"),
        authedJson<ChannelConfig[]>("/api/v1/integration-channels").catch(() => []),
        authedJson<SipEndpoint[]>("/api/v1/call-center/sip-endpoints").catch(() => []),
      ])
      setTemplates(library.templates || [])
      setForms(managedForms)
      setProjects(nextProjects)
      setShares(nextShares)
      setChannelConfigs(nextChannels)
      setSipEndpoints(nextSipEndpoints)
      const focusId = typeof window !== "undefined" ? new URLSearchParams(window.location.search).get("focus") || "" : ""
      const activeId = nextProjectId || selectedProjectId || focusId || nextProjects[0]?.id || ""
      setSelectedProjectId(activeId)
      if (!draft.formTemplateId) {
        const firstForm = managedForms.find((form) => publishedFormVersion(form))
        if (firstForm) setDraft({ ...emptyProject, formTemplateId: firstForm.id, name: firstForm.name })
        else if (library.templates?.[0]) setDraft({ ...emptyProject, formTemplateId: library.templates[0].id, name: library.templates[0].label })
      }
      if (view !== "list") await loadProjectData(activeId)
      setMessage("")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "加载失败")
    }
  }

  async function loadProjectData(projectId: string) {
    const suffix = projectId ? `?projectId=${projectId}` : ""
    const [nextSubmissions, nextStats] = await Promise.all([
      authedJson<Submission[]>(`/api/v1/satisfaction/submissions${suffix}`),
      loadSatisfactionAnalysis(projectId).catch(() => authedJson<Stats>(`/api/v1/satisfaction/stats${suffix}`)),
    ])
    const [nextIndicators, nextIssues] = await Promise.all([
      authedJson<Indicator[]>(`/api/v1/satisfaction/indicators${suffix}`),
      authedJson<Issue[]>(`/api/v1/satisfaction/issues${suffix}`),
    ])
    const [nextBindings, nextRules] = await Promise.all([
      authedJson<IndicatorQuestion[]>(`/api/v1/satisfaction/indicator-questions${suffix}`),
      authedJson<CleaningRule[]>(`/api/v1/satisfaction/cleaning-rules${suffix}`),
    ])
    const nextDeliveries = await authedJson<Delivery[]>(`/api/v1/survey-channel-deliveries${suffix}`)
    setSubmissions(nextSubmissions)
    setStats(nextStats)
    setIndicators(nextIndicators)
    setIndicatorQuestions(nextBindings)
    setCleaningRules(nextRules)
    setIssues(nextIssues)
    setDeliveries(nextDeliveries)
  }

  async function saveProject(nextProject = draft) {
    const normalized = normalizeProject(nextProject)
    const init = { method: normalized.id ? "PUT" : "POST", body: JSON.stringify(normalized) }
    const saved = await authedJson<Project>(normalized.id ? `/api/v1/projects/${normalized.id}` : "/api/v1/projects", init)
      .catch(() => authedJson<Project>(normalized.id ? `/api/v1/satisfaction/projects/${normalized.id}` : "/api/v1/satisfaction/projects", init))
    setDraft(saved)
    setMessage("项目配置已保存")
    await load(saved.id)
  }

  async function deleteProject(id: string) {
    if (!window.confirm("确认删除这个项目？删除后不再出现在项目列表。")) return
    await authedJson(`/api/v1/projects/${id}`, { method: "DELETE" }).catch(() => authedJson(`/api/v1/satisfaction/projects/${id}`, { method: "DELETE" }))
    setProjects(projects.filter((item) => item.id !== id))
    setMessage("项目已删除")
  }

  async function createShare() {
    if (!selectedProject) return setMessage("请先选择或创建项目")
    if (forms.some((form) => form.id === selectedProject.formTemplateId) && !selectedFormVersion) return setMessage("当前表单还没有发布版本，请先在表单设计器发布后再生成公开渠道")
    const created = await authedJson<Share>("/api/v1/survey-share-links", {
      method: "POST",
      body: JSON.stringify({
        projectId: selectedProject.id,
        formTemplateId: selectedProject.formTemplateId,
        title: channelDraft.title || selectedProject.name,
        channel: channelDraft.channel,
        config: {
          allowAnonymous: selectedProject.anonymous,
          requiresVerification: selectedProject.requiresVerification,
          formId: selectedFormVersion?.formId,
          formVersionId: selectedFormVersion?.id,
          formVersion: selectedFormVersion?.version,
          pointCode: channelDraft.pointCode,
          pointName: channelDraft.pointName,
          location: channelDraft.location,
          scene: channelDraft.scene,
          tabletMode: channelDraft.tabletMode || channelDraft.channel === "tablet",
          kioskMode: channelDraft.kioskMode,
          dailyTarget: Number(channelDraft.dailyTarget || 0),
        },
      }),
    })
    setShares([created, ...shares])
    setChannelDraft({ ...channelDraft, pointCode: "", pointName: "", location: "", dailyTarget: "" })
    setMessage("项目采集渠道已生成")
  }

  async function createDeliveries() {
    const share = shares.find((item) => item.id === deliveryDraft.shareId) || projectShares[0]
    if (!selectedProject || !share) return setMessage("请先选择项目和采集渠道")
    const url = `${origin}${share.url}`
    const recipientValues = deliveryDraft.recipients.split(/[\n,，;]/).map((item) => item.trim()).filter(Boolean)
    const recipients = recipientMode === "system" ? systemRecipients.filter((item) => item.available && item.channel === share.channel).map((item) => ({ patientId: item.patientId, name: item.name, recipient: item.recipient, source: item.source })) : []
    if (!recipientValues.length && !recipients.length) return setMessage("请先从患者库拉取收件人，或手工补充收件人")
    const created = await authedJson<Delivery[]>("/api/v1/survey-channel-deliveries", {
      method: "POST",
      body: JSON.stringify({
        projectId: selectedProject.id,
        shareId: share.id,
        channel: share.channel,
        url,
        recipients,
        recipientValues,
        message: deliveryDraft.message || `${selectedProject.name}：请点击链接完成调查 ${url}`,
      }),
    })
    setDeliveries([...created, ...deliveries])
    setDeliveryDraft({ shareId: share.id, recipients: "", message: "" })
    setSystemRecipients([])
    setMessage(`已生成 ${created.length} 条${channelLabels[share.channel] || share.channel}触达任务`)
  }

  async function loadSystemRecipients() {
    const share = shares.find((item) => item.id === deliveryDraft.shareId) || projectShares[0]
    if (!share) return setMessage("请先生成采集渠道")
    const query = new URLSearchParams({ channel: share.channel, limit: "500" })
    if (recipientKeyword.trim()) query.set("keyword", recipientKeyword.trim())
    const recipients = await authedJson<ChannelRecipient[]>(`/api/v1/survey-channel-recipients?${query.toString()}`)
    setSystemRecipients(recipients)
    setRecipientMode("system")
    setMessage(`已从患者库拉取 ${recipients.filter((item) => item.available).length}/${recipients.length} 个可触达收件人`)
  }

  async function sendDelivery(id: string) {
    const updated = await authedJson<Delivery>(`/api/v1/survey-channel-deliveries/${id}/send`, { method: "POST" })
    setDeliveries(deliveries.map((item) => item.id === id ? updated : item))
    setMessage(updated.status === "sent" ? "触达任务已发送" : `触达失败：${updated.error || "请检查接口配置"}`)
  }

  async function sendQueuedDeliveries() {
    const updated = await authedJson<Delivery[]>(`/api/v1/survey-channel-deliveries/send${selectedProjectId ? `?projectId=${selectedProjectId}` : ""}`, { method: "POST" })
    const byId = new Map(updated.map((item) => [item.id, item]))
    setDeliveries(deliveries.map((item) => byId.get(item.id) || item))
    setMessage(`已处理 ${updated.length} 条触达任务`)
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

  async function reapplyCleaningRules() {
    const updated = await authedJson<Submission[]>(`/api/v1/satisfaction/cleaning-rules/reapply${selectedProjectId ? `?projectId=${selectedProjectId}` : ""}`, { method: "POST" })
    setSubmissions(updated)
    await loadProjectData(selectedProjectId)
    setMessage(`已按当前规则重新清洗 ${updated.length} 份答卷`)
  }

  async function saveIssue() {
    const editing = Boolean(issueDraft.id)
    const saved = await authedJson<Issue>(issueDraft.id ? `/api/v1/satisfaction/issues/${issueDraft.id}` : "/api/v1/satisfaction/issues", { method: issueDraft.id ? "PUT" : "POST", body: JSON.stringify({ ...issueDraft, projectId: selectedProjectId }) })
    setIssueDraft(editing ? saved : { id: "", title: "", source: "manual", severity: "medium", status: "open", responsibleDepartment: "", responsiblePerson: "", suggestion: "" })
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

  async function addIssueEvent(nextEvent: IssueEventDraft = eventDraft) {
    if (!issueDraft.id) return setMessage("请先选择问题")
    const attachments = nextEvent.attachments.split(",").map((item) => item.trim()).filter(Boolean)
    const saved = await authedJson<IssueEvent>(`/api/v1/satisfaction/issues/${issueDraft.id}/events`, { method: "POST", body: JSON.stringify({ ...nextEvent, fromStatus: issueDraft.status, attachments }) })
    setIssueEvents([saved, ...issueEvents])
    const nextIssue = { ...issueDraft, status: nextEvent.toStatus || issueDraft.status, materialUrls: attachments.length ? Array.from(new Set([...(issueDraft.materialUrls || []), ...attachments])) : issueDraft.materialUrls }
    setIssueDraft(nextIssue)
    setIssues(issues.map((item) => item.id === nextIssue.id ? nextIssue : item))
    setEventDraft({ action: "assign", toStatus: "assigned", content: "", attachments: "" })
  }

  function editProject(project: Project) {
    setDraft(project)
    setSelectedProjectId(project.id)
    loadProjectData(project.id)
  }

  useEffect(() => { load("") }, [])

  const flowProject = selectedProject || draft
  const operationFlow = activeTab === "overview" ? null : <ProjectFlowPanel project={flowProject} stats={stats} submissions={submissions} shares={projectShares} indicators={indicators} cleaningRules={cleaningRules} issues={issues} />

  const content = <>
    {operationFlow}
    {activeTab === "overview" && <ProjectOverviewTab draft={draft} templates={templates} forms={forms} stats={stats} submissions={submissions} issues={issues} saveProject={saveProject} />}
    {activeTab === "lifecycle" && <ProjectLifecycleTab draft={draft} stats={stats} submissions={submissions} shares={projectShares} indicators={indicators} cleaningRules={cleaningRules} issues={issues} setDraft={setDraft} saveProject={saveProject} setActiveTab={setActiveTab} />}
    {activeTab === "people" && <ProjectPeopleTab draft={draft} setDraft={setDraft} saveProject={saveProject} />}
    {activeTab === "tasks" && <ProjectTasksTab draft={draft} setDraft={setDraft} saveProject={saveProject} />}
    {activeTab === "data" && <ProjectDataTab draft={draft} templates={templates} forms={forms} setDraft={setDraft} saveProject={saveProject} />}
    {activeTab === "modules" && <ProjectModulesTab draft={draft} setDraft={setDraft} saveProject={saveProject} />}
    {activeTab === "phases" && <ProjectPhasesTab draft={draft} setDraft={setDraft} saveProject={saveProject} />}
    {activeTab === "channels" && moduleEnabled(selectedProject || draft, "channels") && <ChannelsTab shares={projectShares} deliveries={deliveries} selectedProject={selectedProject} channelOptions={channelOptions} channelConfigs={channelConfigs} sipEndpoints={sipEndpoints} channelDraft={channelDraft} deliveryDraft={deliveryDraft} recipientKeyword={recipientKeyword} recipientMode={recipientMode} systemRecipients={systemRecipients} setChannelDraft={setChannelDraft} setDeliveryDraft={setDeliveryDraft} setRecipientKeyword={setRecipientKeyword} setRecipientMode={setRecipientMode} createShare={createShare} createDeliveries={createDeliveries} loadSystemRecipients={loadSystemRecipients} sendDelivery={sendDelivery} sendQueuedDeliveries={sendQueuedDeliveries} />}
    {activeTab === "answers" && moduleEnabled(selectedProject || draft, "answers") && <AnswersTab submissions={submissions} openDetail={openDetail} audit={audit} />}
    {(["indicators", "cleaning", "analysis", "issues"] as const).includes(activeTab as "indicators" | "cleaning" | "analysis" | "issues") && moduleEnabled(selectedProject || draft, activeTab as ProjectModule) && (
      <AnalysisTab
        mode={activeTab as "indicators" | "cleaning" | "analysis" | "issues"}
        stats={stats}
        indicators={indicators}
        indicatorQuestions={indicatorQuestions}
        cleaningRules={cleaningRules}
        issues={issues}
        issueEvents={issueEvents}
        indicatorDraft={indicatorDraft}
        bindingDraft={bindingDraft}
        ruleDraft={ruleDraft}
        issueDraft={issueDraft}
        eventDraft={eventDraft}
        setIndicatorDraft={setIndicatorDraft}
        setBindingDraft={setBindingDraft}
        setRuleDraft={setRuleDraft}
        setIssueDraft={setIssueDraft}
        setEventDraft={setEventDraft}
        saveIndicator={saveIndicator}
        saveBinding={saveBinding}
        saveRule={saveRule}
        reapplyCleaningRules={reapplyCleaningRules}
        saveIssue={saveIssue}
        openIssue={openIssue}
        addIssueEvent={addIssueEvent}
        generateIssues={generateIssues}
        project={selectedProject || draft}
        setProject={setDraft}
        saveProject={saveProject}
        submissions={submissions}
      />
    )}
  </>

  if (view === "list") {
    return <ProjectListView projects={projects} templates={templates} forms={forms} message={message} deleteProject={deleteProject} />
  }

  const pageTitle = view === "new" ? "新建项目" : view === "edit" ? "编辑项目属性" : selectedProject?.name || draft.name || "项目看板"
  const pageDesc = view === "new" ? "创建项目基础信息、项目属性、表单数据和能力模块。" : view === "edit" ? "维护项目属性、角色、任务模板、表单数据和期次。" : "查看项目进度、发布前检查、数据质量和整改闭环。"

  if (view === "new") {
    return <ProjectPageFrame title={pageTitle} desc={pageDesc} status={draft.status} activeTab={activeTab} projectId={selectedProjectId || draft.id} saveProject={saveProject}>{content}</ProjectPageFrame>
  }

  if (view === "edit") {
    return <ProjectPageFrame title={pageTitle} desc={pageDesc} status={draft.status} activeTab={activeTab} projectId={selectedProjectId || draft.id} saveProject={saveProject}>{content}</ProjectPageFrame>
  }

  if (view === "dashboard") {
    return <ProjectPageFrame title={pageTitle} desc={pageDesc} status={draft.status} activeTab="overview" projectId={selectedProjectId || draft.id}>{content}</ProjectPageFrame>
  }

  return (
    <div>
      <ProjectPageFrame title={pageTitle} desc={tabHint(activeTab)} status={draft.status} activeTab={activeTab} projectId={selectedProjectId || draft.id} saveProject={saveProject}>{content}</ProjectPageFrame>
      {detail && <SubmissionDrawer detail={detail} audit={audit} onClose={() => setDetail(null)} />}
    </div>
  )
}

function ProjectOverviewTab({ draft, templates, forms, stats, submissions, issues, saveProject }: { draft: Project; templates: Template[]; forms: ManagedForm[]; stats: Stats | null; submissions: Submission[]; issues: Issue[]; saveProject: (project?: Project) => void }) {
  const activeForm = forms.find((form) => form.id === draft.formTemplateId)
  const activeVersion = publishedFormVersion(activeForm)
  const modules = projectModules(draft)
  const checklist = lifecycleChecks(draft, { shares: [], submissions, indicators: [], cleaningRules: [], issues })
  const projectId = draft.id
  const openIssues = issues.filter((item) => item.status !== "closed").length
  const validSamples = stats?.valid || 0
  const workflow = [
    { title: "完善项目配置", status: checklist.slice(0, 4).every((item) => item.ok) ? "已完成" : "待完善", href: `/projects/edit?focus=${encodeURIComponent(projectId)}` },
    { title: "发布采集渠道", status: checklist.find((item) => item.key === "channels")?.ok ? "已生成" : "待发布", href: projectHref("channels", projectId) },
    { title: "审核答卷质量", status: validSamples > 0 ? `${validSamples} 份有效` : "待入库", href: projectHref("answers", projectId) },
    { title: "查看分析看板", status: (stats?.total || 0) > 0 ? "可分析" : "待样本", href: projectHref("analysis", projectId) },
    { title: "跟踪整改闭环", status: openIssues ? `${openIssues} 个未关闭` : "无未关闭", href: projectHref("issues", projectId) },
  ]
  const todos = checklist.filter((item) => !item.ok).slice(0, 4)
  return <section className="grid gap-4 p-4">
    <div className="rounded-lg border border-line bg-surface p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold">项目流程</h3>
          <p className="mt-1 text-sm text-muted">按顺序完成配置、发布、审核、分析和整改。</p>
        </div>
        <a className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" href={todos[0] ? projectHref(todos[0].tab, projectId) : projectHref("channels", projectId)}>{todos[0] ? `去处理：${todos[0].label}` : "继续发布采集"}</a>
      </div>
      <div className="mt-4 grid gap-2 lg:grid-cols-5">
        {workflow.map((item, index) => <a key={item.title} className="rounded-lg border border-line bg-white p-3 hover:border-primary hover:bg-blue-50" href={item.href}>
          <div className="flex items-center justify-between gap-2">
            <span className="grid h-6 w-6 place-items-center rounded-full bg-blue-50 text-xs font-semibold text-primary">{index + 1}</span>
            <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">{item.status}</span>
          </div>
          <div className="mt-3 text-sm font-semibold">{item.title}</div>
        </a>)}
      </div>
    </div>
    <div className="grid gap-4 xl:grid-cols-[1fr_360px]">
      <div className="rounded-lg border border-line p-4">
        <div className="flex flex-wrap items-start justify-between gap-3">
          <div>
            <h3 className="text-lg font-semibold">{draft.name || "未命名项目"}</h3>
            <p className="mt-1 text-sm text-muted">{projectTypeLabels[projectType(draft)] || projectType(draft)} · {targetLabels[draft.targetType] || draft.targetType} · {statusText(draft.status)}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <a className="rounded-lg border border-line px-3 py-2 text-sm text-primary" href={`/projects/edit?focus=${encodeURIComponent(projectId)}`}>编辑项目</a>
            <a className="rounded-lg border border-line px-3 py-2 text-sm text-primary" href={projectHref("lifecycle", projectId)}>生命周期</a>
          </div>
        </div>
        <div className="mt-4 grid gap-3 md:grid-cols-4">
          <MetricBox label="目标样本" value={String(draft.targetSampleSize || 0)} />
          <MetricBox label="实际答卷" value={String(stats?.total || submissions.length)} />
          <MetricBox label="有效答卷" value={String(validSamples)} />
          <MetricBox label="未关闭问题" value={String(openIssues)} />
        </div>
        <div className="mt-4 grid gap-2 md:grid-cols-2">
          <Info label="周期" value={`${draft.startDate || "-"} 至 ${draft.endDate || "-"}`} />
          <Info label="表单" value={activeForm ? `${activeForm.name} v${activeVersion?.version || "-"}` : templateLabel(draft.formTemplateId, templates, forms)} />
          <Info label="采集方式" value={draft.requiresVerification ? "患者验证后采集" : draft.anonymous ? "匿名采集" : "实名采集"} />
          <Info label="启用能力" value={(Object.keys(moduleLabels) as ProjectModule[]).filter((item) => modules[item]).map((item) => moduleLabels[item]).join("、") || "-"} />
        </div>
      </div>
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">待办</h3>
        <div className="mt-3 grid gap-2">
          {!todos.length && <div className="rounded-lg border border-green-100 bg-green-50 px-3 py-3 text-sm text-green-700">当前发布前检查已完成，可以继续采集或查看分析。</div>}
          {todos.map((item) => <a key={item.key} className="flex items-start gap-2 rounded-lg border border-amber-100 bg-amber-50 px-3 py-2 text-left text-sm hover:border-primary" href={projectHref(item.tab, draft.id)}>
            <span className="mt-0.5 grid h-5 w-5 place-items-center rounded-full bg-white text-xs text-amber-700">!</span>
            <span><span className="block font-medium">{item.label}</span><span className="text-xs text-muted">{item.hint}</span></span>
          </a>)}
          <a className="rounded-lg border border-line px-3 py-2 text-center text-sm text-primary" href={projectHref("lifecycle", projectId)}>查看全部检查项</a>
        </div>
      </div>
    </div>
  </section>
}

function ProjectFlowPanel({ project, stats, submissions, shares, indicators, cleaningRules, issues }: { project: Project; stats: Stats | null; submissions: Submission[]; shares: Share[]; indicators: Indicator[]; cleaningRules: CleaningRule[]; issues: Issue[] }) {
  const checklist = lifecycleChecks(project, { shares, submissions, indicators, cleaningRules, issues })
  const openIssues = issues.filter((item) => item.status !== "closed").length
  const validSamples = stats?.valid || 0
  const projectId = project.id
  const workflow = [
    { title: "完善项目配置", status: checklist.slice(0, 4).every((item) => item.ok) ? "已完成" : "待完善", href: `/projects/edit${projectId ? `?focus=${encodeURIComponent(projectId)}` : ""}` },
    { title: "发布采集渠道", status: checklist.find((item) => item.key === "channels")?.ok ? "已生成" : "待发布", href: projectHref("channels", projectId) },
    { title: "审核答卷质量", status: validSamples > 0 ? `${validSamples} 份有效` : "待入库", href: projectHref("answers", projectId) },
    { title: "查看分析看板", status: (stats?.total || 0) > 0 ? "可分析" : "待样本", href: projectHref("analysis", projectId) },
    { title: "跟踪整改闭环", status: openIssues ? `${openIssues} 个未关闭` : "无未关闭", href: projectHref("issues", projectId) },
  ]
  const todos = checklist.filter((item) => !item.ok)
  return <section className="p-4 pb-0">
    <div className="rounded-lg border border-line bg-surface p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold">项目流程</h3>
          <p className="mt-1 text-sm text-muted">按顺序完成配置、发布、审核、分析和整改；进入任何操作页都保留这条主线。</p>
        </div>
        <a className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" href={todos[0] ? projectHref(todos[0].tab, projectId) : projectHref("channels", projectId)}>{todos[0] ? `去处理：${todos[0].label}` : "继续发布采集"}</a>
      </div>
      <div className="mt-4 grid gap-2 lg:grid-cols-5">
        {workflow.map((item, index) => <a key={item.title} className="rounded-lg border border-line bg-white p-3 hover:border-primary hover:bg-blue-50" href={item.href}>
          <div className="flex items-center justify-between gap-2">
            <span className="grid h-6 w-6 place-items-center rounded-full bg-blue-50 text-xs font-semibold text-primary">{index + 1}</span>
            <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">{item.status}</span>
          </div>
          <div className="mt-3 text-sm font-semibold">{item.title}</div>
        </a>)}
      </div>
    </div>
  </section>
}

function ProjectListView({ projects, templates, forms, message, deleteProject }: { projects: Project[]; templates: Template[]; forms: ManagedForm[]; message: string; deleteProject: (id: string) => void }) {
  return <section className="grid gap-4">
    <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-line bg-surface p-4">
      <div>
        <h2 className="text-base font-semibold">项目列表</h2>
        <p className="mt-1 text-sm text-muted">项目是业务根节点；列表只负责查看和进入 CRUD，不再把配置堆在同一页。</p>
      </div>
      <a className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" href="/projects/new">新建项目</a>
    </div>
    <div className="grid gap-3 md:grid-cols-4">
      <PhaseCard title="1. 列表检索" text="只展示项目摘要、状态、表单和主要操作。" />
      <PhaseCard title="2. 新建项目" text="创建基础属性、对象分类和默认表单。" />
      <PhaseCard title="3. 属性编辑" text="配置人员、任务模板、期次、表单和能力模块。" />
      <PhaseCard title="4. 项目看板" text="查看生命周期、待办、渠道、质控、分析和整改。" />
    </div>
    {message && <div className="rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
    <div className="overflow-x-auto rounded-lg border border-line bg-surface">
      <table className="w-full text-sm">
        <thead className="bg-gray-50 text-muted">
          <tr><th className="px-4 py-3 text-left">项目名称</th><th className="px-4 py-3 text-left">类型</th><th className="px-4 py-3 text-left">对象</th><th className="px-4 py-3 text-left">状态</th><th className="px-4 py-3 text-left">表单</th><th className="px-4 py-3 text-right">操作</th></tr>
        </thead>
        <tbody>
          {!projects.length && <tr><td className="px-4 py-8 text-center text-muted" colSpan={6}>暂无项目</td></tr>}
          {projects.map((project) => <tr key={project.id} className="border-t border-line">
            <td className="px-4 py-3 font-medium">{project.name}</td>
            <td className="px-4 py-3">{projectTypeLabels[projectType(project)] || projectType(project)}</td>
            <td className="px-4 py-3">{targetLabels[project.targetType] || project.targetType}</td>
            <td className="px-4 py-3">{statusText(project.status)}</td>
            <td className="px-4 py-3 text-muted">{templateLabel(project.formTemplateId, templates, forms)}</td>
            <td className="px-4 py-3">
              <div className="flex flex-wrap justify-end gap-2">
                <a className="rounded-md border border-line px-3 py-1.5 text-xs text-primary" href={`/projects/dashboard?focus=${encodeURIComponent(project.id)}`}>看板</a>
                <a className="rounded-md border border-line px-3 py-1.5 text-xs text-primary" href={`/projects/edit?focus=${encodeURIComponent(project.id)}`}>编辑属性</a>
                <a className="rounded-md border border-line px-3 py-1.5 text-xs text-primary" href={projectHref("data", project.id)}>表单数据</a>
                <a className="rounded-md border border-line px-3 py-1.5 text-xs text-primary" href={projectHref("channels", project.id)}>发布渠道</a>
                <button className="rounded-md border border-red-200 px-3 py-1.5 text-xs text-red-600" onClick={() => deleteProject(project.id)}>删除</button>
              </div>
            </td>
          </tr>)}
        </tbody>
      </table>
    </div>
  </section>
}

function ProjectPageFrame({ title, desc, status, activeTab, projectId, saveProject, children }: { title: string; desc: string; status: string; activeTab: ProjectTab; projectId?: string; saveProject?: () => void; children: ReactNode }) {
  return <main className="min-w-0 rounded-lg border border-line bg-surface">
    <div className="border-b border-line p-4">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <h2 className="truncate text-base font-semibold">{title}</h2>
          <p className="mt-1 text-sm text-muted">{desc}</p>
        </div>
        <span className="shrink-0 rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-primary">{statusText(status)}</span>
      </div>
    </div>
    <div className="sticky top-16 z-10 flex flex-wrap items-center justify-between gap-2 border-b border-line bg-surface/95 px-4 py-3 backdrop-blur">
      <div className="min-w-0">
        <div className="truncate text-sm font-semibold">{tabLabel(activeTab)}</div>
        <div className="text-xs text-muted">{tabHint(activeTab)}</div>
      </div>
      <div className="flex shrink-0 flex-wrap gap-2">
        <a className="rounded-lg border border-line px-3 py-2 text-sm" href="/projects">项目列表</a>
        {projectId && <a className="rounded-lg border border-line px-3 py-2 text-sm" href={`/projects/edit?focus=${encodeURIComponent(projectId)}`}>编辑项目</a>}
        {activeTab !== "overview" && <a className="rounded-lg border border-line px-3 py-2 text-sm" href={projectId ? `/projects/dashboard?focus=${encodeURIComponent(projectId)}` : "/projects/dashboard"}>返回看板</a>}
        {projectId && activeTab === "overview" && <a className="rounded-lg border border-line px-3 py-2 text-sm" href={projectHref("channels", projectId)}>发布渠道</a>}
        {saveProject && <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={saveProject}>保存项目</button>}
      </div>
    </div>
    {children}
  </main>
}

function ProjectLifecycleTab({ draft, stats, submissions, shares, indicators, cleaningRules, issues, setDraft, saveProject, setActiveTab }: { draft: Project; stats: Stats | null; submissions: Submission[]; shares: Share[]; indicators: Indicator[]; cleaningRules: CleaningRule[]; issues: Issue[]; setDraft: (project: Project) => void; saveProject: (project?: Project) => void; setActiveTab: (tab: ProjectTab) => void }) {
  const checks = lifecycleChecks(draft, { shares, submissions, indicators, cleaningRules, issues })
  const completed = checks.filter((item) => item.ok).length
  const progress = Math.round((completed / Math.max(1, checks.length)) * 100)
  function transition(status: string) {
    const next = { ...draft, status }
    setDraft(next)
    saveProject(next)
  }
  return <section className="grid gap-4 p-4">
    <div className="rounded-lg border border-line p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="text-base font-semibold">项目生命周期</h3>
          <p className="mt-1 text-sm text-muted">草稿配置、发布执行、质控审核、分析整改、结束归档在项目内闭环。</p>
        </div>
        <div className="rounded-full bg-blue-50 px-3 py-1 text-sm font-medium text-primary">当前：{statusText(draft.status)}</div>
      </div>
      <div className="mt-4 h-2 rounded-full bg-gray-100"><div className="h-full rounded-full bg-primary" style={{ width: `${progress}%` }} /></div>
      <div className="mt-2 text-sm text-muted">闭环完成度 {progress}% · {completed}/{checks.length}</div>
      <div className="mt-4 flex flex-wrap gap-2">
        <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => transition("draft")}>保存草稿</button>
        <button className="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white disabled:bg-gray-300" disabled={!checks.filter((item) => item.stage === "publish").every((item) => item.ok)} onClick={() => transition("active")}>发布项目</button>
        <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => transition("paused")}>暂停</button>
        <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => transition("closed")}>结束</button>
        <button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={() => transition("archived")}>归档</button>
      </div>
    </div>
    <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
      {checks.map((item) => <button key={item.key} className={`rounded-lg border p-4 text-left ${item.ok ? "border-green-100 bg-green-50" : "border-amber-100 bg-amber-50"}`} onClick={() => setActiveTab(item.tab)}>
        <div className="flex items-center justify-between gap-3"><span className="font-semibold">{item.label}</span><span className={item.ok ? "text-green-700" : "text-amber-700"}>{item.ok ? "已完成" : "待完善"}</span></div>
        <p className="mt-2 text-sm leading-6 text-muted">{item.hint}</p>
      </button>)}
    </div>
    <div className="grid gap-3 md:grid-cols-4">
      <MetricBox label="渠道" value={String(shares.length)} />
      <MetricBox label="答卷" value={String(stats?.total || submissions.length)} />
      <MetricBox label="清洗规则" value={String(cleaningRules.length)} />
      <MetricBox label="问题" value={String(issues.length)} />
    </div>
  </section>
}

function ProjectPeopleTab({ draft, setDraft, saveProject }: { draft: Project; setDraft: (project: Project) => void; saveProject: (project?: Project) => void }) {
  return <section className="grid gap-4 p-4">
    <SectionHeader title="角色 / 人员" desc="项目负责人、执行人、审核人、数据分析和整改责任人独立维护，后续任务按角色分派。" save={() => saveProject()} />
    <ProjectPeopleEditor project={draft} setProject={setDraft} />
  </section>
}

function ProjectTasksTab({ draft, setDraft, saveProject }: { draft: Project; setDraft: (project: Project) => void; saveProject: (project?: Project) => void }) {
  return <section className="grid gap-4 p-4">
    <SectionHeader title="任务 / 任务模板" desc="把发布、患者触达、电话访谈、答卷审核、分析报告、问题整改和验证拆成可执行任务模板。" save={() => saveProject()} />
    <ProjectTaskTemplateEditor project={draft} setProject={setDraft} />
  </section>
}

function ProjectDataTab({ draft, templates, forms, setDraft, saveProject }: { draft: Project; templates: Template[]; forms: ManagedForm[]; setDraft: (project: Project) => void; saveProject: (project?: Project) => void }) {
  const activeForm = forms.find((form) => form.id === draft.formTemplateId)
  const activeVersion = publishedFormVersion(activeForm)
  return <section className="grid gap-4 p-4">
    <SectionHeader title="表单与数据" desc="绑定版本化表单，配置患者验证、自动带入字段和项目数据范围。" save={() => saveProject()} />
    <div className="grid gap-3 md:grid-cols-4">
      <PhaseCard title="1. 项目属性" text="先确定项目类型、对象、周期和目标样本。" />
      <PhaseCard title="2. 表单模板" text="选择已发布版本，公开问卷严格绑定该版本。" />
      <PhaseCard title="3. 患者验证" text="决定匿名、实名或患者验证后自动带入资料。" />
      <PhaseCard title="4. 数据范围" text="选择患者、就诊、用药、检查、随访等自动拉取域。" />
    </div>
    <ConfigStep index={1} title="项目基础属性" desc="这些字段决定项目列表、生命周期和报表筛选口径。">
        <div className="grid gap-3 md:grid-cols-3">
          <Text label="项目名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
          <Select label="项目类型" value={projectType(draft)} options={Object.keys(projectTypeLabels)} labels={projectTypeLabels} onChange={(v) => setProjectConfig(setDraft, draft, { projectType: v })} />
          <Select label="调查对象" value={draft.targetType} options={Object.keys(targetLabels)} labels={targetLabels} onChange={(v) => setDraft({ ...draft, targetType: v })} />
          <Select label="项目状态" value={draft.status} options={["draft", "active", "paused", "closed", "archived"]} labels={{ draft: "草稿", active: "发布中", paused: "暂停", closed: "结束", archived: "归档" }} onChange={(v) => setDraft({ ...draft, status: v })} />
          <Text label="开始日期" type="date" value={draft.startDate || ""} onChange={(v) => setDraft({ ...draft, startDate: v })} />
          <Text label="结束日期" type="date" value={draft.endDate || ""} onChange={(v) => setDraft({ ...draft, endDate: v })} />
          <Text label="目标样本数" type="number" value={String(draft.targetSampleSize || "")} onChange={(v) => setDraft({ ...draft, targetSampleSize: Number(v) })} />
        </div>
    </ConfigStep>
    <ConfigStep index={2} title="表单模板与数据策略" desc="表单模板、患者验证和数据范围在这里一次性闭环。">
        <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_320px]">
          <label className="grid gap-1">
            <span className="text-sm font-medium text-muted">问卷 / 访谈采集模板</span>
            <select className="h-10 rounded-lg border border-line px-3 text-sm" value={draft.formTemplateId} onChange={(e) => {
              const form = forms.find((item) => item.id === e.target.value)
              setDraft({ ...draft, formTemplateId: e.target.value, name: draft.name || form?.name || templates.find((item) => item.id === e.target.value)?.label || "" })
            }}>
              {forms.length > 0 && <optgroup label="已版本化表单">
                {forms.map((form) => {
                  const version = publishedFormVersion(form)
                  return <option key={form.id} value={form.id} disabled={!version}>{form.name} · {version ? `v${version.version}` : "未发布"}</option>
                })}
              </optgroup>}
              <optgroup label="组件库模板">
                {templates.map((template) => <option key={template.id} value={template.id}>{template.label}</option>)}
              </optgroup>
            </select>
          </label>
          <div className="rounded-lg border border-line bg-gray-50 p-3 text-sm">
            <div className="font-medium text-ink">{activeForm ? "版本化表单" : "组件库模板"}</div>
            <p className="mt-1 leading-6 text-muted">{activeForm ? `当前绑定 ${activeForm.name} v${activeVersion?.version || "-"}，生成渠道后公开问卷严格使用该版本。` : "保持历史模板兼容；正式项目建议先在表单设计器发布版本化表单。"}</p>
          </div>
        </div>
        <div className="mt-3 grid gap-3 md:grid-cols-3">
          <Toggle label="允许匿名" checked={draft.anonymous} onChange={(v) => setDraft({ ...draft, anonymous: v, requiresVerification: !v || draft.requiresVerification })} />
          <Toggle label="患者验证" checked={draft.requiresVerification} onChange={(v) => setDraft({ ...draft, requiresVerification: v, anonymous: v ? false : draft.anonymous })} />
          <div className="flex h-10 items-center rounded-lg border border-line px-3 text-sm text-muted">{draft.requiresVerification ? "验证后自动带入患者、就诊和联系方式" : "匿名渠道不绑定患者档案"}</div>
        </div>
        <ProjectPropertyEditor project={draft} setProject={setDraft} />
    </ConfigStep>
  </section>
}

function ProjectModulesTab({ draft, setDraft, saveProject }: { draft: Project; setDraft: (project: Project) => void; saveProject: (project?: Project) => void }) {
  const modules = projectModules(draft)
  return <section className="grid gap-4 p-4">
    <SectionHeader title="能力模块" desc="渠道、答卷、指标、清洗、分析、整改都作为项目能力开关，启用后成为项目下的子菜单。" save={() => saveProject()} />
        <div className="grid gap-2 md:grid-cols-2 xl:grid-cols-3">
          {(Object.keys(moduleLabels) as ProjectModule[]).map((module) => (
            <label key={module} className={`grid cursor-pointer gap-2 rounded-lg border p-3 ${modules[module] ? "border-primary bg-blue-50" : "border-line bg-white"}`}>
              <span className="flex items-center gap-2 font-medium">
                <input type="checkbox" checked={modules[module]} onChange={(event) => setProjectConfig(setDraft, draft, { modules: { ...modules, [module]: event.target.checked } })} />
                {moduleLabels[module]}
              </span>
              <span className="text-xs leading-5 text-muted">{moduleDescriptions[module]}</span>
            </label>
          ))}
        </div>
  </section>
}

function ProjectPhasesTab({ draft, setDraft, saveProject }: { draft: Project; setDraft: (project: Project) => void; saveProject: (project?: Project) => void }) {
  return <section className="grid gap-4 p-4">
    <SectionHeader title="期次管理" desc="按月度、季度或专项活动拆分目标样本，后续报表按期次追踪。" save={() => saveProject()} />
    <ProjectPhaseEditor project={draft} setProject={setDraft} embedded />
  </section>
}

function ConfigTab({ draft, templates, forms, setDraft, saveProject }: { draft: Project; templates: Template[]; forms: ManagedForm[]; setDraft: (project: Project) => void; saveProject: (project?: Project) => void }) {
  return <ProjectDataTab draft={draft} templates={templates} forms={forms} setDraft={setDraft} saveProject={saveProject} />
}

type ChannelDraft = { channel: string; title: string; pointCode: string; pointName: string; location: string; scene: string; tabletMode: boolean; kioskMode: boolean; dailyTarget: string }

function ChannelsTab({ shares, deliveries, selectedProject, channelOptions, channelConfigs, sipEndpoints, channelDraft, deliveryDraft, recipientKeyword, recipientMode, systemRecipients, setChannelDraft, setDeliveryDraft, setRecipientKeyword, setRecipientMode, createShare, createDeliveries, loadSystemRecipients, sendDelivery, sendQueuedDeliveries }: { shares: Share[]; deliveries: Delivery[]; selectedProject?: Project; channelOptions: string[]; channelConfigs: ChannelConfig[]; sipEndpoints: SipEndpoint[]; channelDraft: ChannelDraft; deliveryDraft: { shareId: string; recipients: string; message: string }; recipientKeyword: string; recipientMode: "system" | "manual"; systemRecipients: ChannelRecipient[]; setChannelDraft: (value: ChannelDraft) => void; setDeliveryDraft: (value: { shareId: string; recipients: string; message: string }) => void; setRecipientKeyword: (value: string) => void; setRecipientMode: (value: "system" | "manual") => void; createShare: () => void; createDeliveries: () => void; loadSystemRecipients: () => void; sendDelivery: (id: string) => void; sendQueuedDeliveries: () => void }) {
  const selectedShare = shares.find((item) => item.id === deliveryDraft.shareId) || shares[0]
  const availableRecipients = systemRecipients.filter((item) => item.available && item.channel === selectedShare?.channel)
  const enabledMessageChannels = channelConfigs.filter((item) => item.enabled).map((item) => `${channelLabels[item.kind] || item.kind}${typeof item.config?.provider === "string" ? `(${item.config.provider})` : ""}`).join("、") || "未启用"
  const phoneReady = sipEndpoints.some((item) => item.config?.enabled === true)
  const deliveryStats = ["queued", "sent", "failed", "delivered"].map((status) => ({ status, count: deliveries.filter((item) => item.status === status).length }))
  const sent = deliveries.filter((item) => item.status === "sent" || item.status === "delivered").length
  const failed = deliveries.filter((item) => item.status === "failed").length
  const successRate = sent + failed > 0 ? `${Math.round((sent / (sent + failed)) * 100)}%` : "-"
  const flow = [
    { title: "1. 接口准备", text: `短信/微信：${enabledMessageChannels}；电话：${phoneReady ? "SIP 可拨打" : "未启用"}` },
    { title: "2. 发布入口", text: "Web、二维码、院内点位和平板入口都绑定项目、表单版本和渠道场景。" },
    { title: "3. 触达收件人", text: "从患者档案自动拉取手机号、OpenID、QQ 标识，手工补充只做临时兜底。" },
    { title: "4. 回执重试", text: "触达任务记录发送、失败、服务商回执、重试和统计结果。" },
  ]
  return <section className="grid gap-4 p-4">
    <div className="rounded-lg border border-blue-200 bg-blue-50 px-4 py-3 text-sm text-primary">
      当前接口：消息渠道 {enabledMessageChannels}；电话接口 {phoneReady ? "已启用" : "未启用"}。触达收件人优先从患者档案读取手机号、微信 OpenID 或 QQ 标识。
    </div>
    <div className="grid gap-3 md:grid-cols-4">
      {flow.map((item) => <div key={item.title} className="rounded-lg border border-line bg-white p-3">
        <div className="font-medium">{item.title}</div>
        <div className="mt-2 text-xs leading-5 text-muted">{item.text}</div>
      </div>)}
    </div>
    <div className="grid gap-3 rounded-lg border border-line bg-white p-4">
      <div>
        <h3 className="font-semibold">渠道发布</h3>
        <p className="mt-1 text-sm text-muted">Web/短信/微信/电话用于触达，二维码和平板用于院内点位采集；生成后都进入同一条答卷入库和统计链路。</p>
      </div>
      <div className="grid gap-3 md:grid-cols-[1fr_220px_auto]">
        <input className="h-10 rounded-lg border border-line px-3 text-sm" placeholder="渠道标题，默认使用项目名称" value={channelDraft.title} onChange={(e) => setChannelDraft({ ...channelDraft, title: e.target.value })} />
        <select className="h-10 rounded-lg border border-line px-3 text-sm" value={channelDraft.channel} onChange={(e) => setChannelDraft({ ...channelDraft, channel: e.target.value })}>{channelOptions.map((id) => <option key={id} value={id}>{channelLabels[id] || id}</option>)}</select>
        <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white disabled:bg-gray-300" disabled={!selectedProject} onClick={createShare}>生成渠道</button>
      </div>
      {(channelDraft.channel === "qr" || channelDraft.channel === "tablet") && <div className="grid gap-3 md:grid-cols-4">
        <input className="h-10 rounded-lg border border-line px-3 text-sm" placeholder="点位编码，如 OPD-1F-001" value={channelDraft.pointCode} onChange={(e) => setChannelDraft({ ...channelDraft, pointCode: e.target.value })} />
        <input className="h-10 rounded-lg border border-line px-3 text-sm" placeholder="点位名称，如 门诊一楼导诊台" value={channelDraft.pointName} onChange={(e) => setChannelDraft({ ...channelDraft, pointName: e.target.value })} />
        <input className="h-10 rounded-lg border border-line px-3 text-sm" placeholder="院区/楼层/科室位置" value={channelDraft.location} onChange={(e) => setChannelDraft({ ...channelDraft, location: e.target.value })} />
        <input className="h-10 rounded-lg border border-line px-3 text-sm" placeholder="每日目标样本" value={channelDraft.dailyTarget} onChange={(e) => setChannelDraft({ ...channelDraft, dailyTarget: e.target.value })} />
        <select className="h-10 rounded-lg border border-line px-3 text-sm" value={channelDraft.scene} onChange={(e) => setChannelDraft({ ...channelDraft, scene: e.target.value })}>
          <option value="general">通用点位</option>
          <option value="outpatient">门诊点位</option>
          <option value="inpatient">住院病区</option>
          <option value="discharge">出院结算</option>
          <option value="physical">体检中心</option>
        </select>
        <label className="flex h-10 items-center gap-2 rounded-lg border border-line px-3 text-sm"><input type="checkbox" checked={channelDraft.tabletMode || channelDraft.channel === "tablet"} onChange={(e) => setChannelDraft({ ...channelDraft, tabletMode: e.target.checked })} />平板模式</label>
        <label className="flex h-10 items-center gap-2 rounded-lg border border-line px-3 text-sm"><input type="checkbox" checked={channelDraft.kioskMode} onChange={(e) => setChannelDraft({ ...channelDraft, kioskMode: e.target.checked })} />全屏自助</label>
      </div>}
    </div>
    <div className="grid gap-3 md:grid-cols-4">
      {deliveryStats.map((item) => <MetricBox key={item.status} label={deliveryStatus(item.status)} value={String(item.count)} />)}
    </div>
    <div className="grid gap-3 md:grid-cols-3">
      <MetricBox label="触达总数" value={String(deliveries.length)} />
      <MetricBox label="发送成功率" value={successRate} />
      <MetricBox label="待回执" value={String(deliveries.filter((item) => item.status === "sent").length)} />
    </div>
    <div className="grid gap-3 rounded-lg border border-line bg-gray-50 p-4">
      <div>
        <h3 className="font-semibold">收件人触达</h3>
        <p className="mt-1 text-sm text-muted">选择已生成渠道后，从患者管理档案自动读取联系方式；手工输入只作为临时补充。</p>
      </div>
      <div className="grid gap-3 lg:grid-cols-[220px_minmax(0,1fr)]">
        <select className="h-10 rounded-lg border border-line px-3 text-sm" value={deliveryDraft.shareId || shares[0]?.id || ""} onChange={(e) => setDeliveryDraft({ ...deliveryDraft, shareId: e.target.value })}>
          {shares.map((share) => <option key={share.id} value={share.id}>{channelLabels[share.channel] || share.channel} · {share.title}</option>)}
        </select>
        <input className="h-10 rounded-lg border border-line px-3 text-sm" placeholder="发送文案，可留空自动使用调查链接" value={deliveryDraft.message} onChange={(e) => setDeliveryDraft({ ...deliveryDraft, message: e.target.value })} />
      </div>
      <div className="grid gap-3 lg:grid-cols-[1fr_auto_auto]">
        <input className="h-10 rounded-lg border border-line px-3 text-sm" placeholder="按姓名、手机号、诊断搜索患者，可留空拉取全部" value={recipientKeyword} onChange={(e) => setRecipientKeyword(e.target.value)} />
        <button className="rounded-lg border border-line bg-white px-4 py-2 text-sm" disabled={!shares.length} onClick={loadSystemRecipients}>从患者库拉取</button>
        <select className="h-10 rounded-lg border border-line px-3 text-sm" value={recipientMode} onChange={(e) => setRecipientMode(e.target.value as "system" | "manual")}>
          <option value="system">系统收件人</option>
          <option value="manual">仅手工补充</option>
        </select>
      </div>
      {!!systemRecipients.length && <div className="rounded-lg border border-line bg-white p-3 text-sm">
        <div className="mb-2 font-medium">患者收件人 {availableRecipients.length}/{systemRecipients.length}</div>
        <div className="grid max-h-44 gap-2 overflow-auto md:grid-cols-2 xl:grid-cols-3">
          {systemRecipients.map((item) => <div key={`${item.patientId}-${item.channel}`} className={`rounded-lg border px-3 py-2 ${item.available ? "border-line" : "border-red-100 bg-red-50 text-red-600"}`}>
            <div className="font-medium">{item.name} <span className="text-xs text-muted">{item.patientNo}</span></div>
            <div className="mt-1 text-xs">{item.available ? `${item.recipient} · ${item.source}` : item.unavailable}</div>
          </div>)}
        </div>
      </div>}
      <textarea className="min-h-20 rounded-lg border border-line px-3 py-2 text-sm" placeholder="手工补充收件人，每行一个。短信/电话填手机号，微信填 OpenID。" value={deliveryDraft.recipients} onChange={(e) => setDeliveryDraft({ ...deliveryDraft, recipients: e.target.value })} />
      <div className="flex flex-wrap justify-end gap-2">
        <button className="rounded-lg border border-line bg-white px-4 py-2 text-sm disabled:text-muted" disabled={!deliveries.some((item) => item.status === "queued" || item.status === "failed")} onClick={sendQueuedDeliveries}>发送队列</button>
        <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white disabled:bg-gray-300" disabled={!shares.length || (recipientMode === "system" && availableRecipients.length === 0 && !deliveryDraft.recipients.trim()) || (recipientMode === "manual" && !deliveryDraft.recipients.trim())} onClick={createDeliveries}>生成触达任务</button>
      </div>
    </div>
    <div className="grid gap-3 xl:grid-cols-2">
      {shares.map((share) => <ChannelCard key={share.id} share={share} />)}
    </div>
    <div className="overflow-x-auto rounded-lg border border-line">
      <table className="w-full text-sm">
        <thead className="bg-gray-50 text-muted"><tr><th className="px-3 py-2 text-left">时间</th><th className="px-3 py-2 text-left">渠道</th><th className="px-3 py-2 text-left">接收人</th><th className="px-3 py-2 text-left">状态</th><th className="px-3 py-2 text-left">消息/错误</th><th className="px-3 py-2 text-left">操作</th></tr></thead>
        <tbody>{deliveries.map((item) => <tr key={item.id} className="border-t border-line"><td className="px-3 py-2">{new Date(item.createdAt).toLocaleString()}</td><td className="px-3 py-2">{channelLabels[item.channel] || item.channel}</td><td className="px-3 py-2"><div>{item.recipientName || item.recipient}</div><div className="text-xs text-muted">{item.providerRef || item.sentAt || ""}</div></td><td className="px-3 py-2">{deliveryStatus(item.status)}</td><td className="max-w-[320px] truncate px-3 py-2" title={item.error || item.message || ""}>{item.error || item.message || "-"}</td><td className="px-3 py-2"><button className="rounded-md border border-line px-2 py-1 text-xs text-primary disabled:text-muted" disabled={item.status === "sent" || item.status === "sending"} onClick={() => sendDelivery(item.id)}>{item.status === "failed" ? "重试" : "发送"}</button></td></tr>)}</tbody>
      </table>
    </div>
  </section>
}

function ChannelCard({ share }: { share: Share }) {
  const isTablet = share.channel === "tablet" || share.config?.tabletMode === true
  const isPoint = share.channel === "qr" || share.channel === "tablet"
  const params = new URLSearchParams()
  if (isTablet) params.set("mode", "tablet")
  if (typeof share.config?.pointCode === "string" && share.config.pointCode) params.set("point", share.config.pointCode)
  const url = `${origin}${share.url}${params.toString() ? `&${params.toString()}` : ""}`
  const [svg, setSvg] = useState("")
  useEffect(() => {
    QRCode.toString(url, { type: "svg", width: 160, margin: 1, errorCorrectionLevel: "M" }).then(setSvg)
  }, [url])
  const dataUrl = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`
  return <div className="rounded-lg border border-line p-4">
    <div className="flex flex-wrap items-start justify-between gap-3">
      <div><div className="font-medium">{share.title}</div><div className="mt-1 text-sm text-muted">{channelLabels[share.channel] || share.channel}{isPoint ? ` · ${String(share.config?.pointName || share.config?.pointCode || "院内点位")}` : ""}</div></div>
      <div className="flex gap-2"><a className="rounded-lg border border-line px-3 py-2 text-sm text-primary" href={url} target="_blank">打开</a><a className="rounded-lg border border-line px-3 py-2 text-sm text-primary" href={dataUrl} download={`${share.title || "survey"}-qr.svg`}>下载二维码</a></div>
    </div>
    {isPoint && <div className="mt-3 grid gap-2 rounded-lg bg-blue-50 px-3 py-2 text-xs text-primary md:grid-cols-3">
      <span>点位：{String(share.config?.pointCode || "-")}</span>
      <span>位置：{String(share.config?.location || "-")}</span>
      <span>目标：{String(share.config?.dailyTarget || 0)} / 日</span>
    </div>}
    <div className="mt-3 grid gap-3 sm:grid-cols-[120px_minmax(0,1fr)]">
      <div className="rounded-lg border border-line bg-white p-2" dangerouslySetInnerHTML={{ __html: svg || "<svg viewBox='0 0 160 160'></svg>" }} />
      <div className="break-all rounded-lg bg-gray-50 px-3 py-2 font-mono text-xs text-muted">{url}</div>
    </div>
  </div>
}

function AnswersTab({ submissions, openDetail, audit }: { submissions: Submission[]; openDetail: (id: string) => void; audit: (id: string, status: string, reason?: string) => void }) {
  const qualityCounts = qualitySummary(submissions)
  return <section className="overflow-x-auto">
    <div className="grid gap-3 p-4 md:grid-cols-5">
      <MetricBox label="待初审" value={String(qualityCounts.pending)} />
      <MetricBox label="复核/可疑" value={String(qualityCounts.review)} />
      <MetricBox label="有效入库" value={String(qualityCounts.valid)} />
      <MetricBox label="剔除无效" value={String(qualityCounts.invalid)} />
      <MetricBox label="有效率" value={qualityCounts.total ? `${Math.round((qualityCounts.valid / qualityCounts.total) * 100)}%` : "-"} />
    </div>
    <table className="w-full text-sm">
      <thead className="bg-gray-50 text-muted"><tr><th className="px-4 py-3 text-left">提交时间</th><th className="px-4 py-3 text-left">渠道</th><th className="px-4 py-3 text-left">质量状态</th><th className="px-4 py-3 text-left">清洗原因</th><th className="px-4 py-3 text-left">时长</th><th className="px-4 py-3 text-left">总体满意</th><th className="px-4 py-3 text-right">操作</th></tr></thead>
      <tbody>{submissions.map((item) => <tr key={item.id} className="border-t border-line"><td className="px-4 py-3">{new Date(item.submittedAt).toLocaleString()}</td><td className="px-4 py-3">{channelLabels[item.channel] || item.channel}</td><td className="px-4 py-3">{qualityLabels[item.qualityStatus] || item.qualityStatus}</td><td className="px-4 py-3">{item.qualityReason || "-"}</td><td className="px-4 py-3">{item.durationSeconds}s</td><td className="px-4 py-3">{String(item.answers?.overall_satisfaction || "-")}</td><td className="px-4 py-3 text-right"><button className="mr-3 text-primary" onClick={() => openDetail(item.id)}>详情</button><button className="mr-3 text-primary" onClick={() => audit(item.id, "level1_review", "进入一级复核")}>复核</button><button className="mr-3 text-primary" onClick={() => audit(item.id, "valid", "复核通过")}>有效</button><button className="text-red-600" onClick={() => audit(item.id, "invalid", "人工判定无效并剔除")}>剔除</button></td></tr>)}</tbody>
    </table>
  </section>
}

function AnalysisTab({ mode, stats, indicators, indicatorQuestions, cleaningRules, issues, issueEvents, indicatorDraft, bindingDraft, ruleDraft, issueDraft, eventDraft, setIndicatorDraft, setBindingDraft, setRuleDraft, setIssueDraft, setEventDraft, saveIndicator, saveBinding, saveRule, reapplyCleaningRules, saveIssue, openIssue, addIssueEvent, generateIssues, project, setProject, saveProject, submissions }: { mode: "indicators" | "cleaning" | "analysis" | "issues"; stats: Stats | null; indicators: Indicator[]; indicatorQuestions: IndicatorQuestion[]; cleaningRules: CleaningRule[]; issues: Issue[]; issueEvents: IssueEvent[]; indicatorDraft: Indicator; bindingDraft: IndicatorQuestion; ruleDraft: CleaningRule; issueDraft: Issue; eventDraft: IssueEventDraft; setIndicatorDraft: (value: Indicator) => void; setBindingDraft: (value: IndicatorQuestion) => void; setRuleDraft: (value: CleaningRule) => void; setIssueDraft: (value: Issue) => void; setEventDraft: (value: IssueEventDraft) => void; saveIndicator: () => void; saveBinding: () => void; saveRule: () => void; reapplyCleaningRules: () => void; saveIssue: () => void; openIssue: (issue: Issue) => void; addIssueEvent: (nextEvent?: IssueEventDraft) => void; generateIssues: () => void; project?: Project; setProject?: (project: Project) => void; saveProject?: (project?: Project) => void; submissions: Submission[] }) {
  const channels = Object.entries(stats?.byChannel || {})
  const reasons = Object.entries(stats?.lowReasons || {}).sort((a, b) => b[1] - a[1]).slice(0, 8)
  const showIndicators = mode === "indicators"
  const showCleaning = mode === "cleaning"
  const showAnalysis = mode === "analysis"
  const showIssues = mode === "issues"
  const modeTitle = ({ indicators: "指标体系", cleaning: "数据清洗", analysis: "分析看板", issues: "问题整改" } as Record<typeof mode, string>)[mode]
  return <section className="grid gap-4 p-4">
    <div className="rounded-lg border border-primary/20 bg-blue-50 p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div className="text-base font-semibold text-ink">{modeTitle}</div>
          <div className="mt-1 text-sm text-muted">当前项目内统一管理指标、清洗、分析和整改，所有数据来自项目答卷入库结果。</div>
        </div>
        <div className="rounded-full bg-white px-3 py-1 text-xs font-medium text-primary">满意度二期</div>
      </div>
    </div>
    {!project && <div className="rounded-lg border border-dashed border-line p-4 text-sm text-muted">
      当前还没有选中项目。二期能力已经在这里聚合展示：保存项目并收到答卷后，会自动填充总分、科室排名、指标得分、低分原因、渠道分布，并可从低分答卷生成整改问题。
    </div>}
    {showAnalysis && <div className="grid gap-3 md:grid-cols-4">
      <PhaseCard title="指标体系" text={`指标 ${indicators.length} 个，题目绑定 ${indicatorQuestions.length} 条，支持树级、权重和国考维度。`} />
      <PhaseCard title="数据清洗" text={`清洗规则 ${cleaningRules.length} 条，答卷 ${submissions.length} 份，支持可疑、有效、无效审核。`} />
      <PhaseCard title="问题台账" text={`问题 ${issues.length} 条，未关闭 ${issues.filter((item) => item.status !== "closed").length} 条，可分派、整改、验证。`} />
      <PhaseCard title="整改闭环" text="问题事件记录分派、整改措施、材料、验证结果和关闭动作。" />
    </div>}
    {showAnalysis && <div className="grid gap-3 md:grid-cols-6"><MetricBox label="答卷数" value={String(stats?.total || 0)} /><MetricBox label="总分均值" value={stats?.scoreAverage ? stats.scoreAverage.toFixed(1) : "-"} /><MetricBox label="有效" value={String(stats?.valid || 0)} /><MetricBox label="待审核" value={String(stats?.pending || 0)} /><MetricBox label="复核/可疑" value={String(stats?.suspicious || 0)} /><MetricBox label="剔除" value={String(stats?.invalid || 0)} /></div>}
    {showAnalysis && <div className="rounded-lg border border-line bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div><h3 className="font-semibold">GraphQL 分析查询层</h3><p className="mt-1 text-sm text-muted">当前分析由 GraphQL 聚合查询返回，可组合项目、答卷、指标、患者就诊、科室、医生、病种和渠道维度。</p></div>
        <span className={`rounded-full px-3 py-1 text-xs ${stats?.graphql ? "bg-green-50 text-green-700" : "bg-gray-100 text-muted"}`}>{stats?.graphql ? "GraphQL 已启用" : "REST 兼容模式"}</span>
      </div>
    </div>}
    {showAnalysis && <div className="grid gap-4 xl:grid-cols-2">
      <Rank title="科室排名" items={stats?.departmentRanking || []} />
      <Rank title="指标得分" items={stats?.indicatorScores || []} />
      <BarList title="渠道分布" items={channels.map(([name, count]) => ({ name: channelLabels[name] || name, value: count }))} />
      <BarList title="低分原因" items={reasons.map(([name, count]) => ({ name, value: count }))} />
      <Rank title="时间趋势" items={stats?.trend || []} />
      <BarList title="重要性矩阵" items={(stats?.importanceMatrix || []).map((item) => ({ name: item.name, value: Math.round((5 - item.score) * (item.impact || 0) * 100) }))} />
    </div>}
    {showAnalysis && <div className="grid gap-4 xl:grid-cols-2">
      <PeriodCompareTable items={stats?.periodCompare || []} />
      <Rank title="岗位分析" items={stats?.jobAnalysis || []} />
      <ShortBoardList items={stats?.shortBoards || []} />
      <VarianceTable items={stats?.varianceAnalysis || []} />
      <CorrelationTable items={stats?.correlation || []} />
    </div>}
    {showAnalysis && <div className="grid gap-4 xl:grid-cols-2">
      {Object.entries(stats?.dimensionRankings || {}).map(([dimension, rows]) => <Rank key={dimension} title={`${dimension}维度排名`} items={rows} />)}
    </div>}
    {showAnalysis && <div className="grid gap-4 xl:grid-cols-2">
      {Object.entries(stats?.crossAnalysis || {}).map(([dimension, rows]) => <Rank key={dimension} title={`${dimension}交叉分析`} items={rows} />)}
    </div>}
    {showAnalysis && <div className="rounded-lg border border-line p-4">
      <h3 className="font-semibold">AI 洞察</h3>
      <div className="mt-3 grid gap-2 text-sm leading-6 text-muted">{(stats?.aiInsights || ["样本量不足，暂未生成洞察。"]).map((item) => <p key={item}>{item}</p>)}</div>
    </div>}
    {showIndicators && <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
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
    </div>}
    {showIndicators && <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
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
    </div>}
    {showCleaning && project && setProject && saveProject && <QualitySystemPanel project={project} setProject={setProject} saveProject={saveProject} submissions={submissions} cleaningRules={cleaningRules} reapplyCleaningRules={reapplyCleaningRules} />}
    {showCleaning && <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="rounded-lg border border-line p-4">
        <div className="flex items-center justify-between gap-3"><h3 className="font-semibold">清洗规则配置</h3><button className="rounded-lg border border-line px-3 py-2 text-sm text-primary" onClick={reapplyCleaningRules}>重新清洗</button></div>
        <div className="mt-3 grid gap-2">{cleaningRules.map((item) => <button key={item.id} className="rounded-lg border border-line p-3 text-left hover:border-primary" onClick={() => setRuleDraft(item)}><div className="font-medium">{item.name}</div><div className="mt-1 text-xs text-muted">{item.enabled ? "启用" : "停用"} · {cleaningRuleLabels[item.ruleType] || item.ruleType} · {cleaningActionLabels[item.action] || item.action}</div><div className="mt-2 break-all rounded bg-gray-50 px-2 py-1 font-mono text-xs text-muted">{JSON.stringify(item.config || {})}</div></button>)}</div>
      </div>
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">{ruleDraft.id ? "编辑规则" : "新增规则"}</h3>
        <div className="mt-3 grid gap-3 text-sm">
          <Text label="规则名称" value={ruleDraft.name} onChange={(v) => setRuleDraft({ ...ruleDraft, name: v })} />
          <Select label="规则类型" value={ruleDraft.ruleType} options={Object.keys(cleaningRuleLabels)} labels={cleaningRuleLabels} onChange={(v) => setRuleDraft({ ...ruleDraft, ruleType: v, config: defaultCleaningConfig(v) })} />
          <CleaningConfigFields rule={ruleDraft} setRule={setRuleDraft} />
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">规则配置 JSON</span><textarea className="min-h-20 rounded-lg border border-line px-3 py-2 font-mono text-xs" value={JSON.stringify(ruleDraft.config || {}, null, 2)} onChange={(e) => setRuleDraft({ ...ruleDraft, config: safeJSON(e.target.value, ruleDraft.config || {}) as Record<string, unknown> })} /></label>
          <div className="grid grid-cols-2 gap-2"><Select label="处理动作" value={ruleDraft.action} options={Object.keys(cleaningActionLabels)} labels={cleaningActionLabels} onChange={(v) => setRuleDraft({ ...ruleDraft, action: v })} /><Toggle label="启用规则" checked={ruleDraft.enabled} onChange={(v) => setRuleDraft({ ...ruleDraft, enabled: v })} /></div>
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={saveRule}>保存规则</button>
        </div>
      </div>
    </div>}
    {showIssues && <IssueWorkflowPanel issues={issues} activeIssue={issueDraft} openIssue={openIssue} generateIssues={generateIssues} />}
    {showIssues && <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_420px]">
      <div className="rounded-lg border border-line p-4">
        <div className="flex items-center justify-between gap-3"><h3 className="font-semibold">整改工单池</h3><button className="rounded-lg border border-line px-3 py-2 text-sm text-primary" onClick={generateIssues}>从低分/投诉生成</button></div>
        <div className="mt-3 grid gap-2">{issues.map((item) => <button key={item.id} className={`rounded-lg border p-3 text-left hover:border-primary ${issueDraft.id === item.id ? "border-primary bg-blue-50" : "border-line"}`} onClick={() => openIssue(item)}>
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <div className="truncate font-medium">{item.title}</div>
              <div className="mt-1 text-xs text-muted">{item.source} · {item.responsibleDepartment || "待分派科室"} · {item.responsiblePerson || "待定责任人"}</div>
            </div>
            <IssueStatusBadge issue={item} />
          </div>
          <div className="mt-2 flex flex-wrap gap-2 text-xs text-muted">
            <span>严重程度：{issueSeverityLabels[item.severity] || item.severity}</span>
            <span>截止：{item.dueDate || "未设置"}</span>
            {isIssueOverdue(item) && <span className="text-red-600">已超期</span>}
          </div>
          {item.suggestion && <div className="mt-2 line-clamp-2 text-xs text-muted">{item.suggestion}</div>}
        </button>)}</div>
        {!issues.length && <div className="mt-3 rounded-lg border border-dashed border-line bg-gray-50 px-3 py-6 text-sm text-muted">还没有整改工单。可以从低分答卷、投诉记录或负面开放题生成。</div>}
      </div>
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">{issueDraft.id ? "工单详情" : "新建整改工单"}</h3>
        <div className="mt-3 grid gap-3 text-sm">
          <Text label="问题标题" value={issueDraft.title} onChange={(v) => setIssueDraft({ ...issueDraft, title: v })} />
          <div className="grid grid-cols-2 gap-2"><Text label="责任科室" value={issueDraft.responsibleDepartment || ""} onChange={(v) => setIssueDraft({ ...issueDraft, responsibleDepartment: v })} /><Text label="责任人" value={issueDraft.responsiblePerson || ""} onChange={(v) => setIssueDraft({ ...issueDraft, responsiblePerson: v })} /></div>
          <div className="grid grid-cols-2 gap-2"><Select label="严重程度" value={issueDraft.severity} options={["low", "medium", "high"]} labels={{ low: "低", medium: "中", high: "高" }} onChange={(v) => setIssueDraft({ ...issueDraft, severity: v })} /><Select label="状态" value={issueDraft.status} options={["open", "assigned", "improving", "verified", "closed"]} labels={{ open: "待分派", assigned: "已分派", improving: "整改中", verified: "待验证", closed: "已关闭" }} onChange={(v) => setIssueDraft({ ...issueDraft, status: v })} /></div>
          <Text label="截止日期" type="date" value={issueDraft.dueDate || ""} onChange={(v) => setIssueDraft({ ...issueDraft, dueDate: v })} />
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">整改建议</span><textarea className="min-h-24 rounded-lg border border-line px-3 py-2" value={issueDraft.suggestion || ""} onChange={(e) => setIssueDraft({ ...issueDraft, suggestion: e.target.value })} /></label>
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">整改措施</span><textarea className="min-h-20 rounded-lg border border-line px-3 py-2" value={issueDraft.measure || ""} onChange={(e) => setIssueDraft({ ...issueDraft, measure: e.target.value })} /></label>
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">整改材料链接，每行一个</span><textarea className="min-h-20 rounded-lg border border-line px-3 py-2" value={(issueDraft.materialUrls || []).join("\n")} onChange={(e) => setIssueDraft({ ...issueDraft, materialUrls: e.target.value.split("\n").map((item) => item.trim()).filter(Boolean) })} /></label>
          <Text label="验证结果" value={issueDraft.verificationResult || ""} onChange={(v) => setIssueDraft({ ...issueDraft, verificationResult: v })} />
          <IssueCompareCard issue={issueDraft} stats={stats} indicators={indicators} />
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={saveIssue}>保存工单</button>
        </div>
      </div>
    </div>}
    {showIssues && issueDraft.id && <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">工单流转记录</h3>
        <div className="mt-3 grid gap-2">{issueEvents.map((item) => <div key={item.id} className="rounded-lg border border-line p-3">
          <div className="flex items-center justify-between gap-3 text-sm font-medium"><span>{issueActionLabels[item.action] || item.action} · {issueStatusLabels[item.fromStatus || ""] || item.fromStatus || "-"} → {issueStatusLabels[item.toStatus || ""] || item.toStatus || "-"}</span><span className="text-xs text-muted">{new Date(item.createdAt).toLocaleString()}</span></div>
          <div className="mt-1 text-xs text-muted">{item.content || "无说明"}</div>
          {!!item.attachments?.length && <div className="mt-2 grid gap-1 text-xs">{item.attachments.map((url) => <a key={url} className="break-all text-primary" href={url} target="_blank" rel="noreferrer">{url}</a>)}</div>}
        </div>)}</div>
        {!issueEvents.length && <div className="mt-3 rounded-lg border border-dashed border-line bg-gray-50 px-3 py-6 text-sm text-muted">还没有流转记录。分派、催办、上传材料、复评和关闭都会留痕。</div>}
      </div>
      <div className="rounded-lg border border-line p-4">
        <h3 className="font-semibold">工单操作</h3>
        <div className="mt-3 grid grid-cols-2 gap-2 text-sm">
          <button className="rounded-lg border border-line px-3 py-2 text-primary" onClick={() => addIssueEvent({ action: "assign", toStatus: "assigned", content: `已分派给 ${issueDraft.responsibleDepartment || "责任科室"} ${issueDraft.responsiblePerson || ""}`.trim(), attachments: "" })}>分派</button>
          <button className="rounded-lg border border-line px-3 py-2 text-primary" onClick={() => addIssueEvent({ action: "remind", toStatus: issueDraft.status || "assigned", content: `整改催办：请在 ${issueDraft.dueDate || "截止日期前"} 完成措施和材料上传。`, attachments: "" })}>催办</button>
          <button className="rounded-lg border border-line px-3 py-2 text-primary" onClick={() => addIssueEvent({ action: "improve", toStatus: "improving", content: issueDraft.measure || "责任科室已开始整改。", attachments: "" })}>整改中</button>
          <button className="rounded-lg border border-line px-3 py-2 text-primary" onClick={() => addIssueEvent({ action: "verify", toStatus: "verified", content: issueDraft.verificationResult || "整改材料已提交，进入复评。", attachments: (issueDraft.materialUrls || []).join(",") })}>提交复评</button>
          <button className="rounded-lg border border-line px-3 py-2 text-primary" onClick={() => addIssueEvent({ action: "upload_material", toStatus: "improving", content: "补充整改材料。", attachments: (issueDraft.materialUrls || []).join(",") })}>上传材料留痕</button>
          <button className="rounded-lg border border-red-200 px-3 py-2 text-red-600" onClick={() => addIssueEvent({ action: "close", toStatus: "closed", content: issueDraft.verificationResult || "复评通过，关闭归档。", attachments: (issueDraft.materialUrls || []).join(",") })}>关闭归档</button>
        </div>
        <h4 className="mt-4 font-semibold">手工补充流转</h4>
        <div className="mt-3 grid gap-3 text-sm">
          <Select label="动作" value={eventDraft.action} options={["assign", "remind", "improve", "upload_material", "verify", "close", "reopen", "note"]} labels={issueActionLabels} onChange={(v) => setEventDraft({ ...eventDraft, action: v })} />
          <Select label="流转到" value={eventDraft.toStatus} options={["open", "assigned", "improving", "verified", "closed"]} labels={{ open: "待分派", assigned: "已分派", improving: "整改中", verified: "待验证", closed: "已关闭" }} onChange={(v) => setEventDraft({ ...eventDraft, toStatus: v })} />
          <label className="grid gap-1"><span className="text-sm font-medium text-muted">说明</span><textarea className="min-h-20 rounded-lg border border-line px-3 py-2" value={eventDraft.content} onChange={(e) => setEventDraft({ ...eventDraft, content: e.target.value })} /></label>
          <Text label="材料链接，逗号分隔" value={eventDraft.attachments} onChange={(v) => setEventDraft({ ...eventDraft, attachments: v })} />
          <button className="rounded-lg bg-primary px-4 py-2 font-medium text-white" onClick={addIssueEvent}>保存流转</button>
        </div>
      </div>
    </div>}
    {showAnalysis && <div className="rounded-lg border border-line p-4">
      <h3 className="font-semibold">报告预览</h3>
      <div className="mt-3 grid gap-3 text-sm leading-6 text-ink">
        <p>{project?.name || "当前项目"} 共回收 {stats?.total || 0} 份答卷，当前总分均值为 {stats?.scoreAverage ? stats.scoreAverage.toFixed(1) : "-"}，有效样本 {stats?.valid || 0} 份，可疑样本 {stats?.suspicious || 0} 份。</p>
        <p>主要短板集中在：{reasons.length ? reasons.map(([name]) => name).join("、") : "暂无明显低分原因"}。建议优先处理高频低分原因并分派至责任科室闭环整改。</p>
        <p>当前问题台账 {issues.length} 条，其中未关闭 {issues.filter((item) => item.status !== "closed").length} 条；后续可按整改完成情况复评对应指标变化。</p>
      </div>
    </div>}
  </section>
}

function SubmissionDrawer({ detail, audit, onClose }: { detail: Submission; audit: (id: string, status: string, reason?: string) => void; onClose: () => void }) {
  const items = detail.answerItems?.length ? detail.answerItems : Object.entries(detail.answers || {}).map(([questionId, answer]) => ({ questionId, questionLabel: questionId, questionType: "", answer }))
  return <div className="fixed inset-0 z-50 grid justify-items-end bg-gray-900/40">
    <aside className="h-full w-full max-w-2xl overflow-y-auto bg-white shadow-xl">
      <div className="flex items-center justify-between border-b border-line p-4"><div><h2 className="font-semibold">答卷详情</h2><p className="mt-1 text-sm text-muted">{qualityLabels[detail.qualityStatus] || detail.qualityStatus} · {detail.qualityReason || "无清洗原因"}</p></div><button className="rounded-lg border border-line px-3 py-2 text-sm" onClick={onClose}>关闭</button></div>
      <div className="grid gap-3 p-4 text-sm">
        <div className="grid gap-2 rounded-lg bg-gray-50 p-3 md:grid-cols-2"><Info label="提交时间" value={new Date(detail.submittedAt).toLocaleString()} /><Info label="渠道" value={channelLabels[detail.channel] || detail.channel} /><Info label="患者" value={detail.patientId || "-"} /><Info label="就诊" value={detail.visitId || "-"} /><Info label="答题时长" value={`${detail.durationSeconds}s`} /><Info label="状态" value={qualityLabels[detail.qualityStatus] || detail.qualityStatus} /></div>
        <div className="flex flex-wrap gap-2"><button className="rounded-lg border border-line px-3 py-2" onClick={() => audit(detail.id, "level1_review", "一级复核中")}>一级复核</button><button className="rounded-lg border border-line px-3 py-2" onClick={() => audit(detail.id, "level2_review", "二级抽检中")}>二级抽检</button><button className="rounded-lg bg-primary px-3 py-2 text-white" onClick={() => audit(detail.id, "valid", "复核通过")}>有效入库</button><button className="rounded-lg border border-red-200 px-3 py-2 text-red-600" onClick={() => audit(detail.id, "invalid", "人工判定无效并剔除")}>剔除无效</button><button className="rounded-lg border border-line px-3 py-2" onClick={() => audit(detail.id, "suspicious", "人工复核可疑")}>标记可疑</button></div>
        <div className="grid gap-2">{items.map((item) => <div key={item.questionId} className="rounded-lg border border-line p-3"><div className="text-muted">{item.questionLabel}</div><div className="mt-1 font-medium">{formatAnswer(item.answer)}</div>{item.score !== undefined && <div className="mt-1 text-xs text-muted">得分：{item.score}</div>}</div>)}</div>
      </div>
    </aside>
  </div>
}

function QualitySystemPanel({ project, setProject, saveProject, submissions, cleaningRules, reapplyCleaningRules }: { project: Project; setProject: (project: Project) => void; saveProject: (project?: Project) => void; submissions: Submission[]; cleaningRules: CleaningRule[]; reapplyCleaningRules: () => void }) {
  const qualityPlan = projectQualityPlan(project)
  const levels = projectQualityLevels(project)
  const people = projectConfigArray(project, "people")
  const counts = qualitySummary(submissions)
  const investigators = people.filter((item) => ["执行人", "调查员", "随访员"].includes(String(item.role || "")))
  const reviewers = people.filter((item) => ["审核人", "质控员", "负责人"].includes(String(item.role || "")))
  function updatePlan(patch: Record<string, unknown>) {
    const next = { ...project, config: { ...(project.config || {}), qualityPlan: { ...qualityPlan, ...patch } } }
    setProject(next)
  }
  function savePlan() {
    saveProject({ ...project, config: { ...(project.config || {}), qualityPlan, qualityLevels: levels } })
  }
  function updateLevel(index: number, patch: Record<string, unknown>) {
    const nextLevels = levels.map((item, i) => i === index ? { ...item, ...patch } : item)
    const next = { ...project, config: { ...(project.config || {}), qualityLevels: nextLevels } }
    setProject(next)
  }
  return <div className="grid gap-4">
    <div className="rounded-lg border border-line bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="font-semibold">项目数据质量体系</h3>
          <p className="mt-1 text-sm text-muted">按项目树形结构组织：调查员采集、自动规则、一级复核、二级抽检、终审剔除和审计留痕。</p>
        </div>
        <div className="flex flex-wrap gap-2">
          <button className="rounded-lg border border-line px-3 py-2 text-sm text-primary" onClick={reapplyCleaningRules}>执行自动质控</button>
          <button className="rounded-lg bg-primary px-3 py-2 text-sm font-medium text-white" onClick={savePlan}>保存质控方案</button>
        </div>
      </div>
      <div className="mt-4 grid gap-3 md:grid-cols-6">
        <MetricBox label="总样本" value={String(counts.total)} />
        <MetricBox label="待初审" value={String(counts.pending)} />
        <MetricBox label="复核/可疑" value={String(counts.review)} />
        <MetricBox label="有效入库" value={String(counts.valid)} />
        <MetricBox label="剔除无效" value={String(counts.invalid)} />
        <MetricBox label="有效率" value={counts.total ? `${Math.round((counts.valid / counts.total) * 100)}%` : "-"} />
      </div>
    </div>
    <div className="grid gap-4 xl:grid-cols-[minmax(0,1fr)_380px]">
      <div className="rounded-lg border border-line bg-white p-4">
        <h3 className="font-semibold">多级质控流程</h3>
        <div className="mt-3 grid gap-2">
          {levels.map((level, index) => <div key={`${level.name}-${index}`} className="grid gap-3 rounded-lg border border-line p-3 md:grid-cols-[80px_1fr_1fr_1fr] md:items-end">
            <Text label="层级" value={String(level.level || index + 1)} onChange={(v) => updateLevel(index, { level: Number(v) })} />
            <Text label="节点名称" value={String(level.name || "")} onChange={(v) => updateLevel(index, { name: v })} />
            <Text label="负责角色" value={String(level.role || "")} onChange={(v) => updateLevel(index, { role: v })} />
            <Text label="通过后状态" value={String(level.toStatus || "")} onChange={(v) => updateLevel(index, { toStatus: v })} />
          </div>)}
        </div>
      </div>
      <div className="rounded-lg border border-line bg-white p-4">
        <h3 className="font-semibold">质控策略</h3>
        <div className="mt-3 grid gap-3 text-sm">
          <Text label="抽检比例 %" type="number" value={String(qualityPlan.samplingRate)} onChange={(v) => updatePlan({ samplingRate: Number(v) })} />
          <Text label="最低有效率 %" type="number" value={String(qualityPlan.minValidRate)} onChange={(v) => updatePlan({ minValidRate: Number(v) })} />
          <Select label="无效样本策略" value={String(qualityPlan.invalidPolicy)} options={["exclude_from_report", "keep_for_trace", "manual_decision"]} labels={{ exclude_from_report: "报表剔除但留痕", keep_for_trace: "保留但标记", manual_decision: "人工终审决定" }} onChange={(v) => updatePlan({ invalidPolicy: v })} />
          <Toggle label="实名项目必须校验患者/就诊身份" checked={qualityPlan.identityRequired !== false} onChange={(v) => updatePlan({ identityRequired: v })} />
          <Toggle label="启用调查员采集留痕" checked={qualityPlan.investigatorTrace !== false} onChange={(v) => updatePlan({ investigatorTrace: v })} />
          <Toggle label="自动规则命中后必须人工复核" checked={qualityPlan.manualReviewRequired !== false} onChange={(v) => updatePlan({ manualReviewRequired: v })} />
        </div>
      </div>
    </div>
    <div className="grid gap-4 xl:grid-cols-3">
      <QualityRoster title="调查员 / 执行人" people={investigators} empty="请在项目角色人员里配置执行人或调查员。" />
      <QualityRoster title="审核员 / 质控员" people={reviewers} empty="请在项目角色人员里配置审核人、质控员或负责人。" />
      <div className="rounded-lg border border-line bg-white p-4">
        <h3 className="font-semibold">规则覆盖</h3>
        <div className="mt-3 grid gap-2 text-sm">
          {cleaningRules.map((rule) => <div key={rule.id} className="flex items-center justify-between gap-3 rounded-lg border border-line px-3 py-2">
            <span>{rule.name}</span>
            <span className={`rounded-full px-2 py-0.5 text-xs ${rule.enabled ? "bg-green-50 text-green-700" : "bg-gray-100 text-muted"}`}>{rule.enabled ? "启用" : "停用"}</span>
          </div>)}
          {!cleaningRules.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-4 text-sm text-muted">还没有自动质控规则。</div>}
        </div>
      </div>
    </div>
  </div>
}

function QualityRoster({ title, people, empty }: { title: string; people: Array<Record<string, unknown>>; empty: string }) {
  return <div className="rounded-lg border border-line bg-white p-4">
    <h3 className="font-semibold">{title}</h3>
    <div className="mt-3 grid gap-2">
      {!people.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-4 text-sm text-muted">{empty}</div>}
      {people.map((item, index) => <div key={`${String(item.name)}-${index}`} className="rounded-lg border border-line px-3 py-2 text-sm">
        <div className="font-medium">{String(item.name || "-")} <span className="text-xs text-muted">{String(item.role || "")}</span></div>
        <div className="mt-1 text-xs text-muted">{String(item.department || "-")} · {String(item.scope || "全项目")}</div>
      </div>)}
    </div>
  </div>
}

function IssueWorkflowPanel({ issues, activeIssue, openIssue, generateIssues }: { issues: Issue[]; activeIssue: Issue; openIssue: (issue: Issue) => void; generateIssues: () => void }) {
  const counts = issueCounts(issues)
  const steps = [
    { key: "open", title: "识别问题", text: "低分、投诉、负面建议或人工创建", count: counts.open },
    { key: "assigned", title: "分派责任", text: "明确责任科室、责任人和截止日期", count: counts.assigned },
    { key: "improving", title: "整改催办", text: "记录措施、催办事件和整改材料", count: counts.improving },
    { key: "verified", title: "复评验证", text: "核对整改结果和指标变化", count: counts.verified },
    { key: "closed", title: "关闭归档", text: "通过复评后归档留痕", count: counts.closed },
  ]
  const overdue = issues.filter(isIssueOverdue)
  return <div className="grid gap-4">
    <div className="grid gap-3 md:grid-cols-6">
      <MetricBox label="待分派" value={String(counts.open)} />
      <MetricBox label="整改中" value={String(counts.improving + counts.assigned)} />
      <MetricBox label="待复评" value={String(counts.verified)} />
      <MetricBox label="已超期" value={String(overdue.length)} />
      <MetricBox label="已关闭" value={String(counts.closed)} />
      <MetricBox label="闭环率" value={issues.length ? `${Math.round((counts.closed / issues.length) * 100)}%` : "-"} />
    </div>
    <div className="rounded-lg border border-line bg-white p-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h3 className="font-semibold">整改闭环工单流</h3>
          <p className="mt-1 text-sm text-muted">按“识别、分派、整改、复评、关闭”推进，每一步都有事件留痕和超期提醒。</p>
        </div>
        <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={generateIssues}>生成待整改工单</button>
      </div>
      <div className="mt-4 grid gap-3 lg:grid-cols-5">
        {steps.map((step, index) => <button key={step.key} className={`rounded-lg border p-3 text-left ${activeIssue.status === step.key ? "border-primary bg-blue-50" : "border-line bg-white"}`} onClick={() => issues.find((item) => item.status === step.key) && openIssue(issues.find((item) => item.status === step.key) as Issue)}>
          <div className="flex items-center justify-between gap-3">
            <span className="flex h-7 w-7 items-center justify-center rounded-full bg-blue-50 text-sm font-semibold text-primary">{index + 1}</span>
            <span className="rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">{step.count} 条</span>
          </div>
          <div className="mt-3 font-semibold">{step.title}</div>
          <div className="mt-1 text-xs leading-5 text-muted">{step.text}</div>
        </button>)}
      </div>
      {!!overdue.length && <div className="mt-4 rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">超期提醒：{overdue.length} 条工单已经超过截止日期，需要催办或调整责任人。</div>}
    </div>
  </div>
}

function IssueStatusBadge({ issue }: { issue: Issue }) {
  const overdue = isIssueOverdue(issue)
  if (overdue) return <span className="shrink-0 rounded-full bg-red-50 px-2 py-0.5 text-xs text-red-700">已超期</span>
  const tone = issue.status === "closed" ? "bg-green-50 text-green-700" : issue.status === "verified" ? "bg-amber-50 text-amber-700" : "bg-blue-50 text-primary"
  return <span className={`shrink-0 rounded-full px-2 py-0.5 text-xs ${tone}`}>{issueStatusLabels[issue.status] || issue.status}</span>
}

function IssueCompareCard({ issue, stats, indicators }: { issue: Issue; stats: Stats | null; indicators: Indicator[] }) {
  const indicator = indicators.find((item) => item.id === issue.indicatorId)
  const scoreRow = stats?.indicatorScores?.find((item) => item.name === indicator?.name || item.name === issue.indicatorId) || stats?.indicatorScores?.[0]
  const current = scoreRow?.score || stats?.scoreAverage || 0
  const target = 4.5
  const gap = current ? Math.max(0, target - current) : 0
  return <div className="rounded-lg border border-line bg-gray-50 p-3">
    <div className="font-medium">整改前后指标对比</div>
    <div className="mt-2 grid gap-2 text-xs text-muted">
      <div className="flex items-center justify-between gap-3"><span>关联指标</span><span className="font-medium text-ink">{indicator?.name || scoreRow?.name || "按项目总体满意度"}</span></div>
      <div className="flex items-center justify-between gap-3"><span>当前得分</span><span className="font-medium text-ink">{current ? current.toFixed(2) : "-"}</span></div>
      <div className="flex items-center justify-between gap-3"><span>整改目标</span><span className="font-medium text-ink">{target.toFixed(2)}</span></div>
      <div className="flex items-center justify-between gap-3"><span>提升缺口</span><span className="font-medium text-ink">{current ? gap.toFixed(2) : "待样本"}</span></div>
    </div>
    <p className="mt-2 text-xs leading-5 text-muted">复评后继续回收样本，分析看板会按指标、科室、医生和病种维度刷新，对比整改前后的短板变化。</p>
  </div>
}

function issueCounts(issues: Issue[]) {
  return issues.reduce((acc, item) => {
    const key = item.status || "open"
    acc[key] = (acc[key] || 0) + 1
    return acc
  }, { open: 0, assigned: 0, improving: 0, verified: 0, closed: 0 } as Record<string, number>)
}

function isIssueOverdue(issue: Issue) {
  if (!issue.dueDate || issue.status === "closed") return false
  const due = new Date(`${issue.dueDate}T23:59:59`)
  return !Number.isNaN(due.getTime()) && due.getTime() < Date.now()
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
function ConfigStep({ index, title, desc, children }: { index: number; title: string; desc: string; children: ReactNode }) {
  return <div className="grid gap-4 rounded-lg border border-line bg-white p-4 lg:grid-cols-[220px_minmax(0,1fr)]">
    <div>
      <div className="flex items-center gap-2">
        <span className="flex h-7 w-7 items-center justify-center rounded-full bg-blue-50 text-sm font-semibold text-primary">{index}</span>
        <h4 className="font-semibold">{title}</h4>
      </div>
      <p className="mt-2 text-sm leading-6 text-muted">{desc}</p>
    </div>
    <div className="min-w-0">{children}</div>
  </div>
}
function SectionHeader({ title, desc, save: _save }: { title: string; desc: string; save: () => void }) {
  return <div className="rounded-lg border border-line bg-white p-4">
    <div>
      <h3 className="text-base font-semibold">{title}</h3>
      <p className="mt-1 text-sm text-muted">{desc}</p>
    </div>
  </div>
}
function QuickNav({ title, text, href }: { title: string; text: string; href: string }) {
  return <a className="rounded-lg border border-line bg-white p-4 text-left hover:border-primary hover:bg-blue-50" href={href}>
    <div className="font-semibold">{title}</div>
    <div className="mt-2 text-sm leading-6 text-muted">{text}</div>
  </a>
}
function WorkflowCard({ title, status, text, href }: { title: string; status: string; text: string; href: string }) {
  return <a className="grid gap-3 rounded-lg border border-line bg-white p-4 hover:border-primary hover:bg-blue-50" href={href}>
    <div className="flex items-start justify-between gap-3">
      <div className="font-semibold">{title}</div>
      <span className="shrink-0 rounded-full bg-gray-100 px-2 py-0.5 text-xs text-muted">{status}</span>
    </div>
    <div className="text-sm leading-6 text-muted">{text}</div>
    <div className="text-sm font-medium text-primary">去处理</div>
  </a>
}
function ProjectPeopleEditor({ project, setProject }: { project: Project; setProject: (project: Project) => void }) {
  const people = projectConfigArray(project, "people")
  const [draft, setDraft] = useState({ role: "负责人", name: "", userId: "", department: "", scope: "全项目" })
  function add() {
    if (!draft.name.trim()) return
    setProjectConfig(setProject, project, { people: [...people, draft] })
    setDraft({ role: "负责人", name: "", userId: "", department: "", scope: "全项目" })
  }
  return <div className="grid gap-3">
    <div className="grid gap-3 md:grid-cols-[130px_1fr_1fr_1fr_120px]">
      <Select label="项目角色" value={draft.role} options={["负责人", "执行人", "审核人", "数据分析", "整改责任人"]} labels={{ 负责人: "负责人", 执行人: "执行人", 审核人: "审核人", 数据分析: "数据分析", 整改责任人: "整改责任人" }} onChange={(v) => setDraft({ ...draft, role: v })} />
      <Text label="人员姓名" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
      <Text label="账号 / 工号" value={draft.userId} onChange={(v) => setDraft({ ...draft, userId: v })} />
      <Text label="科室 / 团队" value={draft.department} onChange={(v) => setDraft({ ...draft, department: v })} />
      <button className="mt-6 rounded-lg border border-line px-3 py-2 text-sm text-primary" onClick={add}>添加</button>
    </div>
    <div className="grid gap-2">
      {!people.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-4 text-sm text-muted">还没有项目成员。至少建议配置负责人、执行人和审核人。</div>}
      {people.map((item, index) => <div key={`${item.name}-${index}`} className="grid gap-2 rounded-lg border border-line px-3 py-2 text-sm md:grid-cols-[120px_1fr_1fr_1fr_auto] md:items-center">
        <span className="font-medium">{String(item.role || "-")}</span>
        <span>{String(item.name || "-")}</span>
        <span className="text-muted">{String(item.userId || "-")}</span>
        <span className="text-muted">{String(item.department || "-")}</span>
        <button className="text-left text-red-600 md:text-right" onClick={() => setProjectConfig(setProject, project, { people: people.filter((_, i) => i !== index) })}>删除</button>
      </div>)}
    </div>
  </div>
}

function ProjectTaskTemplateEditor({ project, setProject }: { project: Project; setProject: (project: Project) => void }) {
  const tasks = projectConfigArray(project, "taskTemplates")
  const [draft, setDraft] = useState({ name: "", category: "采集", assigneeRole: "执行人", dueOffsetDays: "0", trigger: "项目启动" })
  function add() {
    if (!draft.name.trim()) return
    setProjectConfig(setProject, project, { taskTemplates: [...tasks, { ...draft, dueOffsetDays: Number(draft.dueOffsetDays || 0) }] })
    setDraft({ name: "", category: "采集", assigneeRole: "执行人", dueOffsetDays: "0", trigger: "项目启动" })
  }
  return <div className="grid gap-3">
    <div className="grid gap-3 md:grid-cols-[1fr_120px_140px_120px_120px_auto]">
      <Text label="任务模板名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
      <Select label="类型" value={draft.category} options={["采集", "触达", "审核", "分析", "整改", "验证"]} labels={{ 采集: "采集", 触达: "触达", 审核: "审核", 分析: "分析", 整改: "整改", 验证: "验证" }} onChange={(v) => setDraft({ ...draft, category: v })} />
      <Text label="分派角色" value={draft.assigneeRole} onChange={(v) => setDraft({ ...draft, assigneeRole: v })} />
      <Text label="触发点" value={draft.trigger} onChange={(v) => setDraft({ ...draft, trigger: v })} />
      <Text label="期限偏移天" type="number" value={draft.dueOffsetDays} onChange={(v) => setDraft({ ...draft, dueOffsetDays: v })} />
      <button className="mt-6 rounded-lg border border-line px-3 py-2 text-sm text-primary" onClick={add}>添加</button>
    </div>
    <div className="grid gap-2">
      {!tasks.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-4 text-sm text-muted">还没有任务模板。项目发布、答卷审核、分析报告、问题整改都可以作为模板管理。</div>}
      {tasks.map((item, index) => <div key={`${item.name}-${index}`} className="grid gap-2 rounded-lg border border-line px-3 py-2 text-sm md:grid-cols-[1fr_100px_120px_120px_100px_auto] md:items-center">
        <span className="font-medium">{String(item.name || "-")}</span>
        <span>{String(item.category || "-")}</span>
        <span className="text-muted">{String(item.assigneeRole || "-")}</span>
        <span className="text-muted">{String(item.trigger || "-")}</span>
        <span>{String(item.dueOffsetDays || 0)} 天</span>
        <button className="text-left text-red-600 md:text-right" onClick={() => setProjectConfig(setProject, project, { taskTemplates: tasks.filter((_, i) => i !== index) })}>删除</button>
      </div>)}
    </div>
  </div>
}

function ProjectPropertyEditor({ project, setProject }: { project: Project; setProject: (project: Project) => void }) {
  const properties = projectConfigArray(project, "propertyFields")
  const dataScope = projectDataScope(project)
  const [draft, setDraft] = useState({ name: "", code: "", type: "文本", defaultValue: "", required: false })
  const dataOptions = [
    { key: "patient", label: "患者主索引", desc: "姓名、性别、年龄、联系方式、患者号" },
    { key: "visit", label: "就诊记录", desc: "门诊/住院、科室、医生、诊断、就诊时间" },
    { key: "medication", label: "用药记录", desc: "药品、剂量、频次、用药依从性" },
    { key: "history", label: "既往史", desc: "既往病史、个人史、过敏史、家族史" },
    { key: "lab", label: "检验结果", desc: "检验项目、结果值、异常标识" },
    { key: "exam", label: "检查记录", desc: "检查项目、影像、报告结论" },
    { key: "complaint", label: "评价投诉", desc: "投诉、表扬、建议、处理状态" },
    { key: "followup", label: "随访记录", desc: "历史随访、电话记录、问卷结果" },
  ]
  function addProperty() {
    if (!draft.name.trim()) return
    const code = draft.code.trim() || draft.name.trim()
    setProjectConfig(setProject, project, { propertyFields: [...properties, { ...draft, code }] })
    setDraft({ name: "", code: "", type: "文本", defaultValue: "", required: false })
  }
  function toggleScope(key: string, checked: boolean) {
    const next = checked ? [...dataScope, key] : dataScope.filter((item) => item !== key)
    setProjectConfig(setProject, project, { dataScope: next })
  }
  return <div className="mt-3 grid gap-4">
    <section className="rounded-lg border border-line p-4">
      <div>
        <h3 className="font-semibold">项目属性</h3>
        <p className="mt-1 text-sm text-muted">用结构化字段维护项目属性，不需要手写“属性名=默认值”。</p>
      </div>
      <div className="mt-3 grid gap-3 lg:grid-cols-[1fr_1fr_120px_1fr_100px_auto]">
        <Text label="属性名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
        <Text label="字段编码" value={draft.code} onChange={(v) => setDraft({ ...draft, code: v })} />
        <Select label="类型" value={draft.type} options={["文本", "数字", "日期", "选项", "开关"]} labels={{ 文本: "文本", 数字: "数字", 日期: "日期", 选项: "选项", 开关: "开关" }} onChange={(v) => setDraft({ ...draft, type: v })} />
        <Text label="默认值" value={draft.defaultValue} onChange={(v) => setDraft({ ...draft, defaultValue: v })} />
        <Toggle label="必填" checked={draft.required} onChange={(v) => setDraft({ ...draft, required: v })} />
        <button className="mt-6 rounded-lg border border-line px-3 py-2 text-sm text-primary" onClick={addProperty}>添加</button>
      </div>
      <div className="mt-3 grid gap-2">
        {!properties.length && <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-4 text-sm text-muted">还没有自定义项目属性。常见属性如样本来源、低分阈值、调查批次、责任科室。</div>}
        {properties.map((item, index) => <div key={`${item.code}-${index}`} className="grid gap-2 rounded-lg border border-line px-3 py-2 text-sm md:grid-cols-[1fr_1fr_100px_1fr_80px_auto] md:items-center">
          <span className="font-medium">{String(item.name || "-")}</span>
          <span className="text-muted">{String(item.code || "-")}</span>
          <span>{String(item.type || "文本")}</span>
          <span className="text-muted">{String(item.defaultValue || "-")}</span>
          <span>{item.required ? "必填" : "选填"}</span>
          <button className="text-left text-red-600 md:text-right" onClick={() => setProjectConfig(setProject, project, { propertyFields: properties.filter((_, i) => i !== index) })}>删除</button>
        </div>)}
      </div>
    </section>
    <section className="rounded-lg border border-line p-4">
      <div>
        <h3 className="font-semibold">数据自动拉取范围</h3>
        <p className="mt-1 text-sm text-muted">选择项目需要自动汇聚的数据域，后续表单带入、分析看板和患者 360 都按这里取数。</p>
      </div>
      <div className="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-4">
        {dataOptions.map((item) => <label key={item.key} className={`grid cursor-pointer gap-2 rounded-lg border p-3 ${dataScope.includes(item.key) ? "border-primary bg-blue-50" : "border-line bg-white"}`}>
          <span className="flex items-center gap-2 font-medium"><input type="checkbox" checked={dataScope.includes(item.key)} onChange={(event) => toggleScope(item.key, event.target.checked)} />{item.label}</span>
          <span className="text-xs leading-5 text-muted">{item.desc}</span>
        </label>)}
      </div>
    </section>
  </div>
}

function ProjectPhaseEditor({ project, setProject, embedded = false }: { project: Project; setProject: (project: Project) => void; embedded?: boolean }) {
  const phases = Array.isArray(project.config?.phases) ? project.config?.phases as Array<Record<string, string | number>> : []
  const [draft, setDraft] = useState({ name: "", startDate: "", endDate: "", targetSampleSize: "", status: "draft" })
  function addPhase() {
    if (!draft.name.trim()) return
    setProject({ ...project, config: { ...(project.config || {}), phases: [...phases, { ...draft, targetSampleSize: Number(draft.targetSampleSize || 0) }] } })
    setDraft({ name: "", startDate: "", endDate: "", targetSampleSize: "", status: "draft" })
  }
  function removePhase(index: number) {
    setProject({ ...project, config: { ...(project.config || {}), phases: phases.filter((_, i) => i !== index) } })
  }
  return <div className={embedded ? "grid gap-3" : "mt-4 rounded-lg border border-line p-4"}>
    {!embedded && <h3 className="font-semibold">项目期次管理</h3>}
    <div className="grid gap-3 md:grid-cols-[1fr_140px_140px_120px_120px_auto]">
      <Text label="期次名称" value={draft.name} onChange={(v) => setDraft({ ...draft, name: v })} />
      <Text label="开始" type="date" value={draft.startDate} onChange={(v) => setDraft({ ...draft, startDate: v })} />
      <Text label="结束" type="date" value={draft.endDate} onChange={(v) => setDraft({ ...draft, endDate: v })} />
      <Text label="目标样本" type="number" value={draft.targetSampleSize} onChange={(v) => setDraft({ ...draft, targetSampleSize: v })} />
      <Select label="状态" value={draft.status} options={["draft", "active", "closed"]} labels={{ draft: "草稿", active: "执行中", closed: "已结束" }} onChange={(v) => setDraft({ ...draft, status: v })} />
      <button className="mt-6 rounded-lg border border-line px-3 py-2 text-sm text-primary" onClick={addPhase}>添加</button>
    </div>
    <div className="mt-3 grid gap-2">
      {phases.length === 0 && <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-4 text-sm text-muted">还没有期次。可以先不拆期次，项目会按整体周期统计。</div>}
      {phases.map((phase, index) => <div key={`${phase.name}-${index}`} className="grid gap-2 rounded-lg border border-line px-3 py-2 text-sm md:grid-cols-[1fr_120px_120px_110px_90px_auto] md:items-center">
        <span className="font-medium">{String(phase.name)}</span>
        <span className="text-muted">{String(phase.startDate || "-")}</span>
        <span className="text-muted">{String(phase.endDate || "-")}</span>
        <span>目标 {String(phase.targetSampleSize || 0)}</span>
        <span>{String(phase.status || "draft")}</span>
        <button className="text-left text-red-600 md:text-right" onClick={() => removePhase(index)}>删除</button>
      </div>)}
    </div>
  </div>
}
function CleaningConfigFields({ rule, setRule }: { rule: CleaningRule; setRule: (value: CleaningRule) => void }) {
  const config = rule.config || {}
  const update = (key: string, value: unknown) => setRule({ ...rule, config: { ...config, [key]: value } })
  if (rule.ruleType === "duration") return <Text label="最短答题时长（秒）" type="number" value={String(config.minSeconds || 20)} onChange={(v) => update("minSeconds", Number(v))} />
  if (rule.ruleType === "duplicate_project") return <div className="grid grid-cols-2 gap-2"><Text label="重复窗口（小时）" type="number" value={String(config.windowHours || 24)} onChange={(v) => update("windowHours", Number(v))} /><Text label="处理策略" value={String(config.strategy || "keep_latest")} onChange={(v) => update("strategy", v)} /></div>
  if (rule.ruleType === "same_device") return <div className="grid grid-cols-2 gap-2"><Text label="窗口（小时）" type="number" value={String(config.windowHours || 1)} onChange={(v) => update("windowHours", Number(v))} /><Text label="最大提交数" type="number" value={String(config.maxCount || 5)} onChange={(v) => update("maxCount", Number(v))} /></div>
  if (rule.ruleType === "same_option") return <Text label="最少题目数" type="number" value={String(config.minQuestionCount || 5)} onChange={(v) => update("minQuestionCount", Number(v))} />
  if (rule.ruleType === "identity_required") return <Toggle label="允许手机号作为身份兜底" checked={config.allowPhoneFallback !== false} onChange={(v) => update("allowPhoneFallback", v)} />
  if (rule.ruleType === "answer_completion") return <div className="grid gap-2"><Text label="最少有效答题数" type="number" value={String(config.minAnswered || 3)} onChange={(v) => update("minAnswered", Number(v))} /><Text label="必填字段 ID（逗号分隔）" value={Array.isArray(config.requiredFields) ? config.requiredFields.join(",") : String(config.requiredFields || "")} onChange={(v) => update("requiredFields", v.split(",").map((item) => item.trim()).filter(Boolean))} /></div>
  if (rule.ruleType === "investigator_required") return <div className="grid gap-2"><Toggle label="要求调查员/平板点位留痕" checked={config.requireInvestigatorId !== false} onChange={(v) => update("requireInvestigatorId", v)} /><Text label="适用渠道" value={String(config.fallbackChannel || "tablet,qr")} onChange={(v) => update("fallbackChannel", v)} /></div>
  if (rule.ruleType === "sample_authenticity") return <div className="grid gap-2"><Toggle label="要求患者或就诊身份" checked={config.requireVisitOrPatient !== false} onChange={(v) => update("requireVisitOrPatient", v)} /><Toggle label="匿名重复样本进入复核" checked={config.blockAnonymousDuplicate !== false} onChange={(v) => update("blockAnonymousDuplicate", v)} /></div>
  if (rule.ruleType === "quota_control") return <Text label="允许超配额比例 %" type="number" value={String(config.maxOverQuotaPercent || 10)} onChange={(v) => update("maxOverQuotaPercent", Number(v))} />
  return null
}
function PeriodCompareTable({ items }: { items: Array<{ name: string; score: number; count: number; mom?: number | null; yoy?: number | null }> }) {
  return <div className="rounded-lg border border-line p-4"><h3 className="font-semibold">趋势 / 同比环比</h3><div className="mt-3 overflow-x-auto"><table className="w-full text-sm"><thead className="text-muted"><tr><th className="py-2 text-left">周期</th><th className="py-2 text-left">均分</th><th className="py-2 text-left">样本</th><th className="py-2 text-left">环比</th><th className="py-2 text-left">同比</th></tr></thead><tbody>{items.map((item) => <tr key={item.name} className="border-t border-line"><td className="py-2">{item.name}</td><td className="py-2">{item.score?.toFixed?.(1) || "-"}</td><td className="py-2">{item.count}</td><td className="py-2">{formatPercent(item.mom)}</td><td className="py-2">{formatPercent(item.yoy)}</td></tr>)}</tbody></table></div></div>
}
function ShortBoardList({ items }: { items: Array<{ dimension: string; name: string; score: number; count: number; reason: string }> }) {
  return <div className="rounded-lg border border-line p-4"><h3 className="font-semibold">短板定位</h3><div className="mt-3 grid gap-2">{items.length ? items.map((item) => <div key={`${item.dimension}-${item.name}`} className="rounded-lg border border-line px-3 py-2 text-sm"><div className="font-medium">{item.dimension} · {item.name}</div><div className="mt-1 text-xs text-muted">{item.reason} · 分值 {item.score ? item.score.toFixed(1) : "-"} · 样本 {item.count}</div></div>) : <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-4 text-sm text-muted">暂无短板，等待更多有效样本。</div>}</div></div>
}
function VarianceTable({ items }: { items: Array<{ dimension: string; variance: number; stddev: number; minName: string; minScore: number; maxName: string; maxScore: number; gap: number }> }) {
  return <div className="rounded-lg border border-line p-4"><h3 className="font-semibold">方差 / 差距分析</h3><div className="mt-3 grid gap-2">{items.length ? items.slice(0, 8).map((item) => <div key={item.dimension} className="rounded-lg border border-line px-3 py-2 text-sm"><div className="flex justify-between gap-3"><span className="font-medium">{item.dimension}</span><span>差距 {item.gap.toFixed(1)}</span></div><div className="mt-1 text-xs text-muted">最低 {item.minName} {item.minScore.toFixed(1)}；最高 {item.maxName} {item.maxScore.toFixed(1)}；标准差 {item.stddev.toFixed(2)}</div></div>) : <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-4 text-sm text-muted">暂无可比较维度。</div>}</div></div>
}
function CorrelationTable({ items }: { items: Array<{ name: string; coefficient: number; count: number }> }) {
  return <div className="rounded-lg border border-line p-4"><h3 className="font-semibold">相关性分析</h3><div className="mt-3 grid gap-2">{items.length ? items.map((item) => <div key={item.name} className="rounded-lg border border-line px-3 py-2 text-sm"><div className="flex justify-between gap-3"><span className="font-medium">{item.name}</span><span>{item.coefficient.toFixed(2)}</span></div><div className="mt-2 h-2 overflow-hidden rounded-full bg-gray-100"><div className="h-full rounded-full bg-primary" style={{ width: `${Math.min(100, Math.abs(item.coefficient) * 100)}%` }} /></div><div className="mt-1 text-xs text-muted">样本 {item.count}，越接近 1 表示和总体满意度越同步。</div></div>) : <div className="rounded-lg border border-dashed border-line bg-gray-50 px-3 py-4 text-sm text-muted">样本不足，暂无法计算相关性。</div>}</div></div>
}
function MetricBox({ label, value }: { label: string; value: string }) { return <div className="rounded-lg border border-line p-4"><div className="text-sm text-muted">{label}</div><div className="mt-2 text-2xl font-semibold">{value}</div></div> }
function Info({ label, value }: { label: string; value: string }) { return <div><span className="text-muted">{label}</span><div className="font-medium">{value}</div></div> }
function Rank({ title, items }: { title: string; items: Array<{ name: string; score: number; count: number }> }) { return <div className="rounded-lg border border-line p-4"><h3 className="font-semibold">{title}</h3><div className="mt-3 grid gap-2">{items.sort((a, b) => b.score - a.score).map((item) => <div key={item.name} className="flex items-center justify-between gap-3 text-sm"><span>{item.name}</span><span className="font-semibold">{item.score.toFixed(1)} <span className="text-xs text-muted">({item.count})</span></span></div>)}</div></div> }
function BarList({ title, items }: { title: string; items: Array<{ name: string; value: number }> }) { const max = Math.max(1, ...items.map((item) => item.value)); return <div className="rounded-lg border border-line p-4"><h3 className="font-semibold">{title}</h3><div className="mt-3 grid gap-3">{items.map((item) => <div key={item.name} className="text-sm"><div className="mb-1 flex justify-between"><span>{item.name}</span><span>{item.value}</span></div><div className="h-2 rounded-full bg-gray-100"><div className="h-full rounded-full bg-primary" style={{ width: `${(item.value / max) * 100}%` }} /></div></div>)}</div></div> }
function Metric({ label, value }: { label: string; value: string }) { return <div className="flex items-center justify-between border-b border-line pb-2 last:border-0"><span className="text-muted">{label}</span><span className="font-semibold">{value}</span></div> }
function PhaseCard({ title, text }: { title: string; text: string }) { return <div className="rounded-lg border border-line bg-white p-4"><div className="font-semibold">{title}</div><div className="mt-2 text-sm leading-6 text-muted">{text}</div></div> }
function tabLabel(tab: string) { return ({ overview: "项目概况", lifecycle: "生命周期", people: "角色人员", tasks: "任务模板", data: "表单数据", modules: "能力模块", phases: "期次管理", channels: "渠道发布", answers: "答卷入库", indicators: "指标体系", cleaning: "数据清洗", analysis: "分析看板", issues: "问题整改" } as Record<string, string>)[tab] || tab }
function tabHint(tab: string) { return ({ overview: "查看项目摘要和发布前检查", lifecycle: "管理草稿、发布、审核、整改、归档", people: "维护项目角色和参与人员", tasks: "配置项目任务模板和触发规则", data: "绑定表单、验证和自动拉取数据范围", modules: "启用或关闭项目能力模块", phases: "维护项目期次和目标样本", channels: "生成渠道并触达收件人", answers: "查看答卷详情并人工审核", indicators: "维护指标树和题目绑定", cleaning: "配置并执行清洗审核规则", analysis: "查看项目分析看板", issues: "跟踪问题台账和整改闭环" } as Record<string, string>)[tab] || "" }
function projectHref(tab: ProjectTab, projectId?: string) {
  const path = tab === "overview" ? "/projects" : `/projects/${tab}`
  return projectId ? `${path}?focus=${encodeURIComponent(projectId)}` : path
}
function statusText(status: string) { return ({ draft: "草稿", active: "发布中", paused: "暂停", closed: "结束", archived: "归档" } as Record<string, string>)[status] || status }
function deliveryStatus(status: string) { return ({ queued: "待发送", sent: "已发送", failed: "失败", delivered: "已回执" } as Record<string, string>)[status] || status }
function formatAnswer(value: unknown): string { return typeof value === "object" && value ? JSON.stringify(value) : String(value ?? "-") }
function formatPercent(value?: number | null) { return typeof value === "number" && Number.isFinite(value) ? `${value > 0 ? "+" : ""}${Math.round(value * 100)}%` : "-" }
function safeJSON(value: string, fallback: unknown) { try { return JSON.parse(value) } catch { return fallback } }
function projectType(project: Project) {
  return typeof project.config?.projectType === "string" ? project.config.projectType : "satisfaction"
}
function projectModules(project: Project): Record<ProjectModule, boolean> {
  const value = project.config?.modules
  if (!value || typeof value !== "object" || Array.isArray(value)) return { ...defaultModules }
  return { ...defaultModules, ...(value as Partial<Record<ProjectModule, boolean>>) }
}
function moduleEnabled(project: Project, module: ProjectModule) {
  return projectModules(project)[module] !== false
}
function normalizeProject(project: Project): Project {
  return { ...project, config: { ...(project.config || {}), projectType: projectType(project), modules: projectModules(project) } }
}
function setProjectConfig(setProject: (project: Project) => void, project: Project, patch: Record<string, unknown>) {
  setProject({ ...project, config: { ...(project.config || {}), ...patch } })
}
function projectConfigArray(project: Project, key: string): Array<Record<string, unknown>> {
  const value = project.config?.[key]
  return Array.isArray(value) ? value as Array<Record<string, unknown>> : []
}
function configText(project: Project, key: string, fallback: string) {
  const value = project.config?.[key]
  return typeof value === "string" ? value : fallback
}
function projectDataScope(project: Project): string[] {
  const value = project.config?.dataScope
  if (Array.isArray(value)) return value.map((item) => String(item))
  if (typeof value === "string") return value.split(/[,\n，、]/).map((item) => item.trim()).filter(Boolean)
  return []
}
function publishedFormVersion(form?: ManagedForm) {
  if (!form) return undefined
  return form.versions?.find((version) => version.id === form.currentVersionId) || form.versions?.find((version) => version.published)
}
function templateLabel(id: string, templates: Template[], forms: ManagedForm[]) {
  const form = forms.find((item) => item.id === id)
  if (form) {
    const version = publishedFormVersion(form)
    return `${form.name}${version ? ` · v${version.version}` : " · 未发布"}`
  }
  return templates.find((item) => item.id === id)?.label || id
}
function defaultCleaningConfig(ruleType: string): Record<string, unknown> {
  return ({
    duration: { minSeconds: 20 },
    duplicate_project: { windowHours: 24, strategy: "keep_latest" },
    same_device: { windowHours: 1, maxCount: 5 },
    same_option: { minQuestionCount: 5 },
    identity_required: { allowPhoneFallback: true },
    answer_completion: { minAnswered: 3, requiredFields: [] },
    investigator_required: { requireInvestigatorId: true, fallbackChannel: "tablet" },
    sample_authenticity: { requireVisitOrPatient: true, blockAnonymousDuplicate: true },
    quota_control: { maxOverQuotaPercent: 10 },
  } as Record<string, Record<string, unknown>>)[ruleType] || {}
}

function projectQualityPlan(project: Project): Record<string, unknown> {
  const value = project.config?.qualityPlan
  return {
    samplingRate: 20,
    minValidRate: 85,
    invalidPolicy: "exclude_from_report",
    identityRequired: true,
    investigatorTrace: true,
    manualReviewRequired: true,
    ...(value && typeof value === "object" && !Array.isArray(value) ? value as Record<string, unknown> : {}),
  }
}
function projectQualityLevels(project: Project): Array<Record<string, unknown>> {
  const value = project.config?.qualityLevels
  if (Array.isArray(value) && value.length) return value as Array<Record<string, unknown>>
  return [
    { level: 1, name: "自动规则初筛", role: "系统", fromStatus: "submitted", toStatus: "pending" },
    { level: 2, name: "一级人工复核", role: "审核人", fromStatus: "pending/suspicious", toStatus: "level1_review" },
    { level: 3, name: "二级抽检", role: "质控员", fromStatus: "level1_review", toStatus: "level2_review" },
    { level: 4, name: "终审入库/剔除", role: "负责人", fromStatus: "level2_review", toStatus: "valid/invalid" },
  ]
}
function qualitySummary(submissions: Submission[]) {
  return submissions.reduce((acc, item) => {
    acc.total += 1
    if (item.qualityStatus === "valid") acc.valid += 1
    else if (item.qualityStatus === "invalid") acc.invalid += 1
    else if (["suspicious", "level1_review", "level2_review"].includes(item.qualityStatus)) acc.review += 1
    else acc.pending += 1
    return acc
  }, { total: 0, pending: 0, review: 0, valid: 0, invalid: 0 })
}

function loadProjects() {
  return authedJson<Project[]>("/api/v1/projects").catch(() => authedJson<Project[]>("/api/v1/satisfaction/projects"))
}

async function loadSatisfactionAnalysis(projectId: string) {
  const query = `query SatisfactionAnalysis($projectId: ID) {
    satisfactionAnalysis(projectId: $projectId) {
      total valid pending suspicious invalid scoreAverage
      byChannel
      departmentRanking { name score count }
      indicatorScores { name score count }
      trend { name score count }
      periodCompare { name score count mom yoy }
      dimensionRankings
      jobAnalysis { name score count }
      importanceMatrix { name score impact count }
      crossAnalysis
      shortBoards { dimension name score count reason }
      varianceAnalysis { dimension variance stddev minName minScore maxName maxScore gap }
      correlation { name coefficient count }
      lowReasons
      aiInsights
      graphql
    }
  }`
  const response = await authedJson<{ data?: { satisfactionAnalysis?: Stats } }>("/api/v1/graphql", { method: "POST", body: JSON.stringify({ query, variables: { projectId } }) })
  if (!response.data?.satisfactionAnalysis) throw new Error("GraphQL analysis missing")
  return response.data.satisfactionAnalysis
}

function lifecycleChecks(project: Project, data: { shares: Share[]; submissions: Submission[]; indicators: Indicator[]; cleaningRules: CleaningRule[]; issues: Issue[] }) {
  const people = projectConfigArray(project, "people")
  const tasks = projectConfigArray(project, "taskTemplates")
  const phases = projectConfigArray(project, "phases")
  const modules = projectModules(project)
  return [
    { key: "base", stage: "publish", tab: "data" as ProjectTab, ok: Boolean(project.name && project.targetType && project.formTemplateId), label: "项目基础信息", hint: "需要项目名称、对象分类和绑定表单。" },
    { key: "people", stage: "publish", tab: "people" as ProjectTab, ok: people.length > 0, label: "角色人员", hint: "至少配置负责人、执行人或审核人，任务才能分派。" },
    { key: "tasks", stage: "publish", tab: "tasks" as ProjectTab, ok: tasks.length > 0, label: "任务模板", hint: "发布、触达、审核、分析、整改建议沉淀为模板。" },
    { key: "phases", stage: "publish", tab: "phases" as ProjectTab, ok: phases.length > 0 || Boolean(project.startDate && project.endDate), label: "周期 / 期次", hint: "配置整体周期或拆分期次，报表才能按周期追踪。" },
    { key: "channels", stage: "execute", tab: "channels" as ProjectTab, ok: !modules.channels || data.shares.length > 0, label: "采集渠道", hint: "生成 Web、二维码、短信、微信、电话或平板入口。" },
    { key: "answers", stage: "execute", tab: "answers" as ProjectTab, ok: !modules.answers || data.submissions.length > 0, label: "答卷入库", hint: "项目发布后需要有真实提交和答卷明细。" },
    { key: "indicators", stage: "quality", tab: "indicators" as ProjectTab, ok: !modules.indicators || data.indicators.length > 0, label: "指标体系", hint: "题目需要映射到指标、权重、服务环节和国考维度。" },
    { key: "cleaning", stage: "quality", tab: "cleaning" as ProjectTab, ok: !modules.cleaning || data.cleaningRules.length > 0, label: "数据清洗", hint: "配置重复、时长、同设备、全同选项等质控规则。" },
    { key: "analysis", stage: "analysis", tab: "analysis" as ProjectTab, ok: !modules.analysis || data.submissions.length > 0, label: "分析报表", hint: "有有效样本后生成指标、科室、医生、病种、就诊类型聚合。" },
    { key: "issues", stage: "closure", tab: "issues" as ProjectTab, ok: !modules.issues || data.issues.length === 0 || data.issues.every((item) => item.status === "closed"), label: "整改闭环", hint: "低分和投诉进入问题台账，分派、整改、验证后关闭。" },
  ]
}
