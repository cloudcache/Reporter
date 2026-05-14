import { useEffect, useMemo, useRef, useState } from "react"
import { publicApiUrl, publicFetch, publicJson } from "../lib/auth"

interface OptionItem { label: string; value: string }
interface ComponentItem {
  id: string
  type: string
  label: string
  required?: boolean
  helpText?: string
  placeholder?: string
  options?: OptionItem[]
  rows?: string[]
  columns?: string[]
  scale?: number
}
interface SurveyPayload {
  share: { id: string; title: string; token: string; channel?: string; config?: Record<string, unknown> }
  template: { id: string; label: string; hint?: string; scenario?: string; components: ComponentItem[] }
  requiresVerification?: boolean
}
interface VerificationPayload { verified: boolean; patient?: { id: string; name: string; patientNo: string; phone: string }; visit?: { id: string; visitNo: string; departmentName?: string; diagnosisName?: string }; values?: Record<string, unknown> }

export function PublicSurveyChat() {
  const [survey, setSurvey] = useState<SurveyPayload | null>(null)
  const [components, setComponents] = useState<ComponentItem[]>([])
  const [answers, setAnswers] = useState<Record<string, unknown>>({})
  const [verified, setVerified] = useState(false)
  const [verification, setVerification] = useState({ identifier: "", phone: "" })
  const [verifiedPatient, setVerifiedPatient] = useState<VerificationPayload | null>(null)
  const [message, setMessage] = useState("正在加载调查表单...")
  const [submitted, setSubmitted] = useState(false)
  const [displayMode, setDisplayMode] = useState({ tablet: false, kiosk: false, point: "" })
  const [startedAt] = useState(() => new Date().toISOString())
  const sourceRef = useRef<EventSource | null>(null)

  const questions = useMemo(() => components.filter((item) => item.type !== "section"), [components])
  const sections = useMemo(() => buildSections(components), [components])
  const answeredCount = questions.filter((item) => hasAnswer(answers[item.id])).length
  const progress = questions.length ? Math.round((answeredCount / questions.length) * 100) : 0

  useEffect(() => {
    const params = new URLSearchParams(window.location.search)
    const token = params.get("token") || ""
    setDisplayMode({ tablet: params.get("mode") === "tablet", kiosk: params.get("kiosk") === "1", point: params.get("point") || "" })
    if (!token) {
      setMessage("缺少调查链接参数，请确认链接是否完整")
      return
    }

    publicJson<SurveyPayload>(`/api/v1/public/survey/${token}`)
      .then((data: SurveyPayload) => {
        setSurvey(data)
        setDisplayMode((current) => ({
          tablet: current.tablet || data.share.channel === "tablet" || data.share.config?.tabletMode === true,
          kiosk: current.kiosk || data.share.config?.kioskMode === true,
          point: current.point || String(data.share.config?.pointCode || data.share.config?.pointName || ""),
        }))
        if (data.requiresVerification) {
          setMessage("")
          return
        }
        setVerified(true)
        startSurvey(token, data)
      })
      .catch((error) => setMessage(error instanceof Error ? error.message : "加载失败"))

    return () => sourceRef.current?.close()
  }, [])

  function startSurvey(token: string, data = survey, patientId = "") {
    if (!data) return
    if (typeof EventSource === "undefined") {
      setComponents(data.template.components || [])
      setMessage("")
      return
    }
    publicFetch(`/api/v1/public/survey/${token}/interviews`, {
      method: "POST",
      body: JSON.stringify({ patientId }),
    })
      .then((res) => {
        if (!res.ok) throw new Error("访谈会话创建失败")
        sourceRef.current?.close()
        sourceRef.current = new EventSource(publicApiUrl(`/api/v1/public/survey/${token}/events`))
        sourceRef.current.addEventListener("form_component", (event) => {
          const component = JSON.parse((event as MessageEvent).data) as ComponentItem
          setMessage("")
          setComponents((items) => items.some((item) => item.id === component.id) ? items : [...items, component])
        })
        sourceRef.current.addEventListener("done", () => {
          setMessage("")
          sourceRef.current?.close()
        })
        sourceRef.current.onerror = () => {
          setMessage((current) => current || "实时连接已断开，已加载的问题仍可继续填写")
          sourceRef.current?.close()
        }
      })
      .catch((error) => setMessage(error instanceof Error ? error.message : "加载失败"))
  }

  function answer(component: ComponentItem, value: unknown) {
    if (survey?.requiresVerification && isAutoFilledField(component.id)) return
    setAnswers((current) => ({ ...current, [component.id]: value }))
  }

  async function verifyPatient() {
    const token = new URLSearchParams(window.location.search).get("token") || ""
    if (!verification.identifier || !verification.phone) {
      setMessage("请填写就诊号/患者号和手机号")
      return
    }
    try {
      setMessage("正在核验身份并拉取就诊信息...")
      const data = await publicJson<VerificationPayload>(`/api/v1/public/survey/${token}/verify`, {
        method: "POST",
        body: JSON.stringify(verification),
      })
      setVerified(true)
      setVerifiedPatient(data)
      setAnswers((current) => ({ ...(data.values || {}), ...current }))
      setMessage("")
      startSurvey(token, survey, data.patient?.id || "")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "验证失败")
    }
  }

  async function submit() {
    const missing = questions.find((item) => item.required && !hasAnswer(answers[item.id]))
    if (missing) {
      setMessage(`请先填写：${missing.label}`)
      const node = document.getElementById(`field-${missing.id}`)
      node?.scrollIntoView({ behavior: "smooth", block: "center" })
      return
    }
    const token = new URLSearchParams(window.location.search).get("token") || ""
    try {
      setMessage("正在提交调查结果...")
      await publicFetch(`/api/v1/public/survey/${token}/submissions`, {
        method: "POST",
        body: JSON.stringify({
          patientId: verifiedPatient?.patient?.id || "",
          visitId: verifiedPatient?.visit?.id || "",
          startedAt,
          durationSeconds: Math.max(1, Math.round((Date.now() - new Date(startedAt).getTime()) / 1000)),
          answers,
        }),
      }).then((response) => {
        if (!response.ok) throw new Error("提交失败，请稍后重试")
      })
      setSubmitted(true)
      setMessage("")
    } catch (error) {
      setMessage(error instanceof Error ? error.message : "提交失败")
    }
  }

  return (
    <main className={`min-h-screen bg-[#f4f7fb] text-ink ${displayMode.tablet ? "text-[18px]" : ""}`}>
      <div className="survey-hero">
        <div className={`mx-auto grid w-full gap-6 px-4 py-8 sm:px-6 lg:grid-cols-[minmax(0,1fr)_300px] lg:px-8 ${displayMode.tablet ? "max-w-7xl" : "max-w-6xl"}`}>
          <section className="min-w-0">
            <div className="mb-5">
              <div className="mb-3 inline-flex rounded-full bg-white/90 px-3 py-1 text-xs font-medium text-primary shadow-sm">医院随访调查</div>
              <h1 className="text-2xl font-semibold leading-tight text-white sm:text-3xl">{survey?.share.title || "在线调查表单"}</h1>
              <p className="mt-2 max-w-2xl text-sm leading-6 text-blue-50">{survey?.template.hint || "请根据实际情况填写，系统会自动保存已展示的调查项目，支持微信、QQ 和网页链接访问。"}</p>
            </div>
            <div className="rounded-lg border border-white/40 bg-white/95 p-4 shadow-sm">
              <div className="flex flex-wrap items-center justify-between gap-3">
                <div>
                  <div className="text-sm font-semibold">填写进度</div>
                  <div className="mt-1 text-xs text-muted">已填写 {answeredCount} / {questions.length || 0} 项</div>
                </div>
                <div className="text-2xl font-semibold text-primary">{progress}%</div>
              </div>
              <div className="mt-3 h-2 overflow-hidden rounded-full bg-gray-100">
                <div className="h-full rounded-full bg-primary transition-all" style={{ width: `${progress}%` }} />
              </div>
            </div>
          </section>

          <aside className="rounded-lg border border-white/40 bg-white/95 p-4 shadow-sm">
            <div className="text-sm font-semibold">表单信息</div>
            <div className="mt-3 grid gap-2 text-sm">
              <Info label="模板" value={survey?.template.label || "-"} />
              <Info label="类型" value={survey?.template.scenario || "调查访谈"} />
              <Info label="方式" value={channelLabel(survey?.share.channel)} />
              {displayMode.point && <Info label="点位" value={displayMode.point} />}
              {displayMode.tablet && <Info label="模式" value={displayMode.kiosk ? "平板自助全屏" : "平板调查"} />}
            </div>
          </aside>
        </div>
      </div>

      <div className={`mx-auto grid w-full gap-5 px-4 pb-24 pt-5 sm:px-6 lg:grid-cols-[minmax(0,1fr)_300px] lg:px-8 ${displayMode.tablet ? "max-w-7xl" : "max-w-6xl"}`}>
        <section className="min-w-0 rounded-lg border border-line bg-white shadow-sm">
          <div className="border-b border-line px-4 py-4 sm:px-5">
            <h2 className="text-base font-semibold">调查内容</h2>
            <p className="mt-1 text-sm text-muted">问题按表单模板推送，选项和输入框都支持手机触控填写。</p>
          </div>
          {message && <div className="border-b border-line bg-blue-50 px-4 py-3 text-sm text-primary sm:px-5">{message}</div>}
          {survey?.requiresVerification && !verified && (
            <div className="grid gap-4 p-4 sm:p-5">
              <div className="rounded-lg border border-amber-200 bg-amber-50 p-4 text-sm leading-6 text-amber-800">本次调查需要先核验患者身份。验证通过后，系统会自动带入患者基础信息和就诊信息。</div>
              <div className="grid gap-4 md:grid-cols-2">
                <label className="grid gap-1">
                  <span className="text-sm font-medium text-muted">微信身份 / 就诊号 / 患者号 / 病历号</span>
                  <input className="h-11 rounded-lg border border-line px-3 text-base outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" value={verification.identifier} onChange={(e) => setVerification({ ...verification, identifier: e.target.value })} placeholder="例如 MZ20260501001" />
                </label>
                <label className="grid gap-1">
                  <span className="text-sm font-medium text-muted">预留手机号</span>
                  <input className="h-11 rounded-lg border border-line px-3 text-base outline-none focus:border-primary focus:ring-2 focus:ring-blue-100" value={verification.phone} onChange={(e) => setVerification({ ...verification, phone: e.target.value })} placeholder="请输入手机号" />
                </label>
              </div>
              <button className="h-11 rounded-lg bg-primary px-4 text-sm font-semibold text-white md:w-40" onClick={verifyPatient}>验证并填写</button>
            </div>
          )}
          {verifiedPatient?.patient && (
            <div className="border-b border-line bg-green-50 px-4 py-3 text-sm text-green-800 sm:px-5">已验证：{verifiedPatient.patient.name} · {verifiedPatient.patient.patientNo} · 就诊信息已自动带入</div>
          )}
          {(!survey?.requiresVerification || verified) && <div className="grid gap-4 p-4 sm:p-5">
            {!components.length && !message && <div className="rounded-lg border border-dashed border-line p-8 text-center text-sm text-muted">正在等待问题推送...</div>}
            {sections.map((section) => (
              <div key={section.title} className="grid gap-3">
                {section.title && <div className="rounded-lg bg-gray-50 px-3 py-2 text-sm font-semibold">{section.title}</div>}
                {section.items.map((component, index) => (
                  <article id={`field-${component.id}`} key={component.id} className={`rounded-lg border border-line bg-white transition focus-within:border-primary focus-within:ring-2 focus-within:ring-blue-100 ${displayMode.tablet ? "p-5" : "p-4"}`}>
                    <div className="mb-3 flex items-start gap-3">
                      <span className="grid h-7 w-7 shrink-0 place-items-center rounded-full bg-blue-50 text-xs font-semibold text-primary">{index + 1}</span>
                      <div className="min-w-0">
                        <h3 className="text-base font-semibold leading-6">{component.label}{component.required ? <span className="ml-1 text-danger">*</span> : null}</h3>
                        {component.helpText && <p className="mt-1 text-sm leading-5 text-muted">{component.helpText}</p>}
                      </div>
                    </div>
                    <AnswerControl component={component} value={answers[component.id]} readOnly={Boolean(survey?.requiresVerification && isAutoFilledField(component.id))} onChange={(value) => answer(component, value)} />
                  </article>
                ))}
              </div>
            ))}
          </div>}
        </section>

        <aside className="hidden content-start gap-4 lg:grid">
          <div className="rounded-lg border border-line bg-white p-4 shadow-sm">
            <div className="text-sm font-semibold">填写提示</div>
            <ul className="mt-3 grid gap-2 text-sm leading-6 text-muted">
              <li>必填项带红色星号。</li>
              <li>如需修改，直接返回对应问题编辑。</li>
              <li>提交前可检查全部已填答案。</li>
            </ul>
          </div>
          <button className="h-11 rounded-lg bg-primary text-sm font-semibold text-white shadow-sm hover:bg-blue-700 disabled:bg-gray-300" disabled={!questions.length || submitted || Boolean(survey?.requiresVerification && !verified)} onClick={submit}>{submitted ? "已提交" : "提交调查"}</button>
        </aside>
      </div>

      <div className="fixed inset-x-0 bottom-0 border-t border-line bg-white/95 px-4 py-3 shadow-lg lg:hidden">
        <div className="mx-auto flex max-w-6xl items-center gap-3">
          <div className="min-w-0 flex-1">
            <div className="text-xs text-muted">填写进度</div>
            <div className="mt-1 h-2 overflow-hidden rounded-full bg-gray-100"><div className="h-full bg-primary" style={{ width: `${progress}%` }} /></div>
          </div>
          <button className="h-11 rounded-lg bg-primary px-5 text-sm font-semibold text-white disabled:bg-gray-300" disabled={!questions.length || submitted || Boolean(survey?.requiresVerification && !verified)} onClick={submit}>{submitted ? "已提交" : "提交"}</button>
        </div>
      </div>

      {submitted && (
        <div className="fixed inset-0 z-50 grid place-items-center bg-gray-900/45 p-4">
          <div className="w-full max-w-md rounded-lg bg-white p-6 text-center shadow-xl">
            <div className="mx-auto grid h-12 w-12 place-items-center rounded-full bg-green-50 text-xl font-semibold text-success">✓</div>
            <h2 className="mt-4 text-lg font-semibold">提交成功</h2>
            <p className="mt-2 text-sm leading-6 text-muted">感谢您的配合，调查结果已记录。您可以关闭当前页面。</p>
            <button className="mt-5 h-10 rounded-lg border border-line px-4 text-sm font-medium hover:border-primary" onClick={() => setSubmitted(false)}>返回查看</button>
          </div>
        </div>
      )}
    </main>
  )
}

function AnswerControl({ component, value, readOnly, onChange }: { component: ComponentItem; value: unknown; readOnly?: boolean; onChange: (value: unknown) => void }) {
  if (shouldUseSelect(component)) {
    const options = optionsFor(component)
    return <select className="h-11 w-full rounded-lg border border-line bg-white px-3 text-base outline-none focus:border-primary focus:ring-2 focus:ring-blue-100 disabled:bg-gray-50 disabled:text-muted" disabled={readOnly} value={String(value || "")} onChange={(e) => onChange(e.target.value)}><option value="">请选择</option>{options.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}</select>
  }
  if (component.type === "textarea") {
    return <textarea className="min-h-28 w-full rounded-lg border border-line px-3 py-2 text-base leading-6 outline-none focus:border-primary focus:ring-2 focus:ring-blue-100 disabled:bg-gray-50 disabled:text-muted" disabled={readOnly} placeholder={component.placeholder || "请输入"} value={String(value || "")} onChange={(e) => onChange(e.target.value)} />
  }
  if (component.type === "multi_select") {
    const selected = Array.isArray(value) ? value.map(String) : []
    return <div className="grid gap-2 sm:grid-cols-2">{(component.options || []).map((option) => <ChoiceButton key={option.value} disabled={readOnly} active={selected.includes(option.value)} label={option.label} onClick={() => onChange(selected.includes(option.value) ? selected.filter((item) => item !== option.value) : [...selected, option.value])} />)}</div>
  }
  if (component.type === "single_select" || component.type === "rating" || component.type === "likert" || component.options?.length) {
    const options = component.options?.length ? component.options : Array.from({ length: component.scale || 5 }, (_, index) => ({ label: String(index + 1), value: String(index + 1) }))
    return <div className="grid gap-2 sm:grid-cols-2">{options.map((option) => <ChoiceButton key={option.value} disabled={readOnly} active={String(value || "") === option.value} label={option.label} onClick={() => onChange(option.value)} />)}</div>
  }
  if (component.type === "matrix") {
    const matrixValue = typeof value === "object" && value ? value as Record<string, string> : {}
    return <div className="overflow-x-auto rounded-lg border border-line"><table className="w-full min-w-[520px] text-sm"><tbody>{(component.rows || []).map((row) => <tr key={row} className="border-t border-line first:border-0"><td className="w-36 bg-gray-50 px-3 py-3 font-medium">{row}</td>{(component.columns || []).map((column) => <td key={column} className="px-3 py-3"><label className="flex items-center gap-2"><input type="radio" disabled={readOnly} name={`${component.id}-${row}`} checked={matrixValue[row] === column} onChange={() => onChange({ ...matrixValue, [row]: column })} />{column}</label></td>)}</tr>)}</tbody></table></div>
  }
  const inputType = component.type === "number" ? "number" : component.type === "date" ? "date" : "text"
  return <input type={inputType} className="h-11 w-full rounded-lg border border-line px-3 text-base outline-none focus:border-primary focus:ring-2 focus:ring-blue-100 disabled:bg-gray-50 disabled:text-muted" disabled={readOnly} placeholder={readOnly ? "验证后自动带入" : component.placeholder || "请输入"} value={String(value || "")} onChange={(e) => onChange(inputType === "number" ? Number(e.target.value) : e.target.value)} />
}

function ChoiceButton({ active, disabled, label, onClick }: { active: boolean; disabled?: boolean; label: string; onClick: () => void }) {
  return <button type="button" disabled={disabled} className={`min-h-11 rounded-lg border px-3 py-2 text-left text-sm font-medium transition disabled:bg-gray-50 disabled:text-muted ${active ? "border-primary bg-blue-50 text-primary shadow-sm" : "border-line bg-white text-ink hover:border-primary"}`} onClick={onClick}>{label}</button>
}

function shouldUseSelect(component: ComponentItem) {
  return component.id === "overall_satisfaction" || component.id === "recommend_score" || component.label === "总体满意度" || component.label === "推荐意愿"
}

function optionsFor(component: ComponentItem) {
  if (component.options?.length) return component.options
  return Array.from({ length: component.scale || 10 }, (_, index) => ({ label: String(index + 1), value: String(index + 1) }))
}

function buildSections(components: ComponentItem[]) {
  const sections: Array<{ title: string; items: ComponentItem[] }> = []
  for (const component of components) {
    if (component.type === "section") {
      sections.push({ title: component.label, items: [] })
    } else {
      if (!sections.length) sections.push({ title: "", items: [] })
      sections[sections.length - 1].items.push(component)
    }
  }
  return sections.filter((section) => section.title || section.items.length)
}

function hasAnswer(value: unknown) {
  if (Array.isArray(value)) return value.length > 0
  if (typeof value === "object" && value) return Object.keys(value).length > 0
  return value !== undefined && value !== null && String(value) !== ""
}

function isAutoFilledField(id: string) {
  return new Set(["patient_id", "patient_no", "patient_name", "patient_gender", "patient_age", "patient_phone", "blood_type", "visit_id", "visit_no", "visit_date", "discharge_date", "department", "doctor_name", "diagnosis", "discharge_diagnosis"]).has(id)
}

function channelLabel(value?: string) {
  const labels: Record<string, string> = { web: "网页链接", qr: "院内二维码", tablet: "平板调查", wechat: "微信公众号", wework: "企业微信", mini_program: "微信小程序", qq: "QQ 链接", sms: "短信链接", phone: "电话随访" }
  return labels[value || ""] || "公开链接"
}

function Info({ label, value }: { label: string; value: string }) {
  return <div className="flex items-start justify-between gap-3 border-b border-line/70 pb-2 last:border-0"><span className="text-muted">{label}</span><span className="text-right font-medium">{value}</span></div>
}
