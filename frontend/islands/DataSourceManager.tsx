import { useEffect, useMemo, useState } from "react"
import { apiBase, requireSession } from "../lib/auth"

type Protocol = "mysql" | "postgres" | "http" | "soap" | "xml" | "grpc" | "hl7" | "dicom" | "custom"
type Tab = "base" | "mapping" | "dictionary" | "payload" | "result"

interface DictionaryEntry {
  key: string
  label: string
  value: string
}

interface DictionaryMapping {
  name: string
  keyField?: string
  labelField?: string
  valueField?: string
  entries?: DictionaryEntry[]
}

interface FieldMapping {
  source: string
  target: string
  entity?: string
  dictionary?: string
  required?: boolean
  type?: string
  default?: unknown
}

interface DataSource {
  id: string
  name: string
  protocol: Protocol
  endpoint: string
  config?: Record<string, unknown>
  dictionaries?: DictionaryMapping[]
  fieldMapping?: FieldMapping[]
}

interface SyncPreview {
  columns: string[]
  rows: Array<Record<string, unknown>>
}

interface SyncResult {
  created: number
  updated: number
  patients: unknown[]
  visits: unknown[]
  medicalRecords: unknown[]
}

const emptySource: DataSource = {
  id: "",
  name: "",
  protocol: "http",
  endpoint: "",
  config: { method: "GET", timeoutMs: 3000 },
  dictionaries: [],
  fieldMapping: [
    { source: "$.id", target: "patient.patientNo", required: true },
    { source: "$.name", target: "patient.name", required: true },
  ],
}

const samplePayload: Record<Protocol, string> = {
  http: JSON.stringify({ id: "P9001", name: "赵六", gender: "M", phone: "13900009999", age: 46, visit: { visitNo: "V9001", departmentName: "心内科", diagnosisName: "高血压" } }, null, 2),
  soap: `<soap:Envelope>
  <soap:Body>
    <Patient><PatientNo>P9001</PatientNo><Name>赵六</Name><Gender>M</Gender><Phone>13900009999</Phone></Patient>
    <Visit><VisitNo>V9001</VisitNo><DepartmentName>心内科</DepartmentName></Visit>
  </soap:Body>
</soap:Envelope>`,
  xml: `<Patient><PatientNo>P9001</PatientNo><Name>赵六</Name><Gender>M</Gender><Phone>13900009999</Phone></Patient>`,
  grpc: JSON.stringify({ patient: { id: "P9001", name: "赵六" }, visit: { visitNo: "V9001" } }, null, 2),
  mysql: JSON.stringify({ patient_no: "P9001", name: "赵六", gender: "M", phone: "13900009999" }, null, 2),
  postgres: JSON.stringify({ patient_no: "P9001", name: "赵六", gender: "M", phone: "13900009999" }, null, 2),
  hl7: "MSH|^~\\&|HIS|HOSP|REPORTER|HOSP|202605141200||ADT^A01|MSG001|P|2.5.1\rPID|1||P9001||赵六||19800101|M|||北京市朝阳区||13900009999\rPV1|1|O|CARD^心内科^1||||1001^王医生|||||||||||V9001|||||||||||||||||||||||||202605141130",
  dicom: JSON.stringify({ "0010,0020": "P9001", "0010,0010": "赵六", "0008,0050": "ACC001", "0008,1030": "胸部 CT", "0020,000D": "1.2.840.113619.2" }, null, 2),
  custom: JSON.stringify({ id: "P9001", name: "赵六" }, null, 2),
}

const targetOptions = [
  "patient.patientNo",
  "patient.medicalRecordNo",
  "patient.name",
  "patient.gender",
  "patient.birthDate",
  "patient.age",
  "patient.idCardNo",
  "patient.phone",
  "patient.address",
  "patient.insuranceType",
  "patient.bloodType",
  "patient.allergies",
  "patient.diagnosis",
  "visit.visitNo",
  "visit.visitType",
  "visit.departmentCode",
  "visit.departmentName",
  "visit.attendingDoctor",
  "visit.visitAt",
  "visit.diagnosisCode",
  "visit.diagnosisName",
  "record.recordNo",
  "record.recordType",
  "record.title",
  "record.chiefComplaint",
  "record.presentIllness",
  "record.diagnosisCode",
  "record.diagnosisName",
  "record.procedureName",
  "record.studyUid",
  "record.studyDesc",
  "record.recordedAt",
]

const tabs: Array<{ id: Tab; label: string }> = [
  { id: "base", label: "基础信息" },
  { id: "mapping", label: "字段映射" },
  { id: "dictionary", label: "字典" },
  { id: "payload", label: "样例报文" },
  { id: "result", label: "映射结果" },
]

export function DataSourceManager() {
  const [token, setToken] = useState("")
  const [sources, setSources] = useState<DataSource[]>([])
  const [selectedId, setSelectedId] = useState("")
  const [draft, setDraft] = useState<DataSource>(emptySource)
  const [payload, setPayload] = useState(samplePayload.http)
  const [preview, setPreview] = useState<SyncPreview | null>(null)
  const [result, setResult] = useState<SyncResult | null>(null)
  const [message, setMessage] = useState("正在连接数据源 API...")
  const [activeTab, setActiveTab] = useState<Tab>("base")
  const selected = useMemo(() => sources.find((source) => source.id === selectedId), [sources, selectedId])

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
      const list = await authed<DataSource[]>("/api/v1/data-sources")
      setSources(list)
      if (list[0]) choose(list[0])
      setMessage("")
    } catch (error) {
      setMessage(`数据源 API 未连接：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  function choose(source: DataSource) {
    setSelectedId(source.id)
    setDraft(normalizeSource(source))
    setPayload(samplePayload[source.protocol] || samplePayload.http)
    setPreview(null)
    setResult(null)
    setActiveTab("base")
  }

  async function save() {
    try {
      const body = JSON.stringify(normalizeSource(draft))
      const saved = selectedId
        ? await authed<DataSource>(`/api/v1/data-sources/${selectedId}`, token, { method: "PUT", body })
        : await authed<DataSource>("/api/v1/data-sources", token, { method: "POST", body })
      setSources(selectedId ? sources.map((item) => item.id === selectedId ? saved : item) : [saved, ...sources])
      setSelectedId(saved.id)
      setDraft(normalizeSource(saved))
      setMessage("数据源配置已保存")
    } catch (error) {
      setMessage(`保存失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function remove() {
    if (!selectedId || !selected || !window.confirm(`删除「${selected.name}」？`)) return
    try {
      await authed(`/api/v1/data-sources/${selectedId}`, token, { method: "DELETE" })
      const next = sources.filter((source) => source.id !== selectedId)
      setSources(next)
      if (next[0]) choose(next[0])
      else {
        setSelectedId("")
        setDraft(emptySource)
      }
      setMessage("数据源已删除")
    } catch (error) {
      setMessage(`删除失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function previewMapping() {
    try {
      const data = await authed<SyncPreview>(`/api/v1/data-sources/${selectedId}/preview`, token, { method: "POST", body: JSON.stringify({ payload: parsePayload(payload) }) })
      setPreview(data)
      setResult(null)
      setActiveTab("result")
      setMessage("映射预览已生成")
    } catch (error) {
      setMessage(`预览失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  async function sync(dryRun = false) {
    try {
      const data = await authed<SyncResult>(`/api/v1/data-sources/${selectedId}/sync`, token, { method: "POST", body: JSON.stringify({ payload: parsePayload(payload), dryRun }) })
      setResult(data)
      setActiveTab("result")
      setMessage(dryRun ? "已完成试同步" : `同步完成：新增 ${data.created}，更新 ${data.updated}`)
    } catch (error) {
      setMessage(`同步失败：${error instanceof Error ? error.message : "未知错误"}`)
    }
  }

  function updateMapping(index: number, patch: Partial<FieldMapping>) {
    const mapping = [...(draft.fieldMapping || [])]
    mapping[index] = { ...mapping[index], ...patch }
    setDraft({ ...draft, fieldMapping: mapping })
  }

  function removeMapping(index: number) {
    setDraft({ ...draft, fieldMapping: (draft.fieldMapping || []).filter((_, itemIndex) => itemIndex !== index) })
  }

  function updateDictionary(index: number, patch: Partial<DictionaryMapping>) {
    const dictionaries = [...(draft.dictionaries || [])]
    dictionaries[index] = { ...dictionaries[index], ...patch }
    setDraft({ ...draft, dictionaries })
  }

  function removeDictionary(index: number) {
    setDraft({ ...draft, dictionaries: (draft.dictionaries || []).filter((_, itemIndex) => itemIndex !== index) })
  }

  useEffect(() => {
    load()
  }, [])

  const previewColumns = preview?.columns?.length ? preview.columns : ["patient.patientNo", "patient.name", "visit.visitNo", "record.recordNo"]

  return (
    <div className="grid gap-4 lg:grid-cols-[280px_minmax(0,1fr)]">
      <aside className="min-w-0 rounded-lg border border-line bg-surface">
        <div className="flex items-center justify-between border-b border-line p-4">
          <h2 className="text-sm font-semibold">数据源</h2>
          <button className="rounded-md border border-line px-2.5 py-1.5 text-xs hover:border-primary" onClick={() => { setSelectedId(""); setDraft(emptySource); setPayload(samplePayload.http); setPreview(null); setResult(null); setActiveTab("base") }}>新增</button>
        </div>
        <div className="max-h-[calc(100vh-190px)] overflow-y-auto p-3">
          <div className="grid gap-2">
            {sources.map((source) => (
              <button key={source.id} className={`rounded-lg border px-3 py-3 text-left ${source.id === selectedId ? "border-primary bg-blue-50" : "border-line hover:bg-gray-50"}`} onClick={() => choose(source)}>
                <span className="flex items-center justify-between gap-2">
                  <span className="min-w-0 truncate text-sm font-medium">{source.name}</span>
                  <span className="shrink-0 rounded bg-gray-100 px-2 py-0.5 text-xs text-muted">{source.protocol}</span>
                </span>
                <span className="mt-1 block truncate text-xs text-muted">{source.endpoint || "未配置连接地址"}</span>
              </button>
            ))}
          </div>
        </div>
      </aside>

      <main className="min-w-0 rounded-lg border border-line bg-surface">
        <div className="border-b border-line p-4">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div className="min-w-0">
              <h2 className="text-base font-semibold">{selectedId ? "编辑数据源" : "新增数据源"}</h2>
              <p className="mt-1 text-sm text-muted">配置连接、字典和字段映射，并用样例报文验证患者、就诊、病历同步结果。</p>
            </div>
            <div className="flex flex-wrap gap-2">
              <button className="rounded-md border border-line px-3 py-2 text-sm hover:border-primary disabled:text-muted" disabled={!selectedId} onClick={previewMapping}>预览映射</button>
              <button className="rounded-md border border-line px-3 py-2 text-sm hover:border-primary disabled:text-muted" disabled={!selectedId} onClick={() => sync(true)}>试同步</button>
              <button className="rounded-md border border-line px-3 py-2 text-sm hover:border-primary disabled:text-muted" disabled={!selectedId} onClick={() => sync(false)}>同步入库</button>
              {selectedId && <button className="rounded-md border border-red-200 px-3 py-2 text-sm text-red-600 hover:bg-red-50" onClick={remove}>删除</button>}
              <button className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-white" onClick={save}>保存</button>
            </div>
          </div>
          {message && <div className="mt-3 rounded-lg border border-blue-100 bg-blue-50 px-4 py-3 text-sm text-primary">{message}</div>}
        </div>

        <div className="border-b border-line px-4">
          <div className="flex gap-1 overflow-x-auto py-2">
            {tabs.map((tab) => (
              <button key={tab.id} className={`shrink-0 rounded-md px-3 py-2 text-sm ${activeTab === tab.id ? "bg-blue-50 font-medium text-primary" : "text-muted hover:bg-gray-50 hover:text-ink"}`} onClick={() => setActiveTab(tab.id)}>
                {tab.label}
              </button>
            ))}
          </div>
        </div>

        <div className="min-w-0 p-4">
          {activeTab === "base" && (
            <section className="grid max-w-4xl gap-4 md:grid-cols-2">
              <Text label="名称" value={draft.name} onChange={(value) => setDraft({ ...draft, name: value })} />
              <Text label="Endpoint" value={draft.endpoint} onChange={(value) => setDraft({ ...draft, endpoint: value })} />
              <label className="grid gap-1 text-sm">
                <span className="text-muted">协议</span>
                <select className="h-10 rounded-md border border-line px-3 text-sm" value={draft.protocol} onChange={(event) => {
                  const protocol = event.target.value as Protocol
                  setDraft({ ...draft, protocol, config: protocolDefaults(protocol) })
                  setPayload(samplePayload[protocol])
                }}>
                  <option value="http">REST JSON</option>
                  <option value="soap">SOAP Web Service</option>
                  <option value="xml">XML</option>
                  <option value="grpc">gRPC</option>
                  <option value="mysql">MySQL</option>
                  <option value="postgres">PostgreSQL</option>
                  <option value="hl7">HL7 v2</option>
                  <option value="dicom">DICOM</option>
                  <option value="custom">自定义</option>
                </select>
              </label>
              <label className="grid gap-1 text-sm md:col-span-2">
                <span className="text-muted">配置 JSON</span>
                <textarea className="min-h-32 resize-y rounded-md border border-line px-3 py-2 font-mono text-xs leading-5" value={JSON.stringify(draft.config || {}, null, 2)} onChange={(event) => setDraft({ ...draft, config: safeJSON(event.target.value, draft.config || {}) })} />
              </label>
            </section>
          )}

          {activeTab === "mapping" && (
            <section className="grid gap-3">
              <div className="flex items-center justify-between gap-3">
                <h3 className="text-sm font-semibold">字段映射</h3>
                <button className="rounded-md border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => setDraft({ ...draft, fieldMapping: [...(draft.fieldMapping || []), { source: "", target: "patient.name" }] })}>新增映射</button>
              </div>
              <div className="overflow-x-auto rounded-lg border border-line">
                <table className="w-full min-w-[880px] text-sm">
                  <thead className="bg-gray-50 text-xs text-muted">
                    <tr>
                      <th className="px-3 py-2 text-left">来源路径</th>
                      <th className="px-3 py-2 text-left">系统字段</th>
                      <th className="px-3 py-2 text-left">字典</th>
                      <th className="px-3 py-2 text-left">类型</th>
                      <th className="px-3 py-2 text-left">必填</th>
                      <th className="px-3 py-2 text-right">操作</th>
                    </tr>
                  </thead>
                  <tbody>
                    {(draft.fieldMapping || []).map((mapping, index) => (
                      <tr key={index} className="border-t border-line">
                        <td className="px-3 py-2"><input className="h-10 w-full rounded-md border border-line px-3 font-mono text-sm" value={mapping.source} onChange={(event) => updateMapping(index, { source: event.target.value })} /></td>
                        <td className="px-3 py-2">
                          <input className="h-10 w-full rounded-md border border-line px-3 text-sm" list="data-source-target-fields" value={mapping.target} onChange={(event) => updateMapping(index, { target: event.target.value })} />
                        </td>
                        <td className="px-3 py-2"><input className="h-10 w-full rounded-md border border-line px-3 text-sm" value={mapping.dictionary || ""} onChange={(event) => updateMapping(index, { dictionary: event.target.value })} /></td>
                        <td className="px-3 py-2"><input className="h-10 w-full rounded-md border border-line px-3 text-sm" value={mapping.type || ""} onChange={(event) => updateMapping(index, { type: event.target.value })} /></td>
                        <td className="px-3 py-2"><input type="checkbox" checked={!!mapping.required} onChange={(event) => updateMapping(index, { required: event.target.checked })} /></td>
                        <td className="px-3 py-2 text-right"><button className="rounded-md border border-red-200 px-2.5 py-1.5 text-xs text-red-600 hover:bg-red-50" onClick={() => removeMapping(index)}>删除</button></td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <datalist id="data-source-target-fields">{targetOptions.map((target) => <option key={target} value={target} />)}</datalist>
            </section>
          )}

          {activeTab === "dictionary" && (
            <section className="grid max-w-5xl gap-3">
              <div className="flex items-center justify-between gap-3">
                <h3 className="text-sm font-semibold">字典映射</h3>
                <button className="rounded-md border border-line px-3 py-2 text-sm hover:border-primary" onClick={() => setDraft({ ...draft, dictionaries: [...(draft.dictionaries || []), { name: "新字典", entries: [] }] })}>新增字典</button>
              </div>
              {(draft.dictionaries || []).map((dictionary, index) => (
                <div key={index} className="grid gap-3 rounded-lg border border-line p-4">
                  <div className="flex items-center gap-3">
                    <input className="h-10 min-w-0 flex-1 rounded-md border border-line px-3 text-sm" value={dictionary.name} onChange={(event) => updateDictionary(index, { name: event.target.value })} />
                    <button className="rounded-md border border-red-200 px-3 py-2 text-sm text-red-600 hover:bg-red-50" onClick={() => removeDictionary(index)}>删除</button>
                  </div>
                  <textarea className="min-h-36 resize-y rounded-md border border-line px-3 py-2 font-mono text-xs leading-5" value={JSON.stringify(dictionary.entries || [], null, 2)} onChange={(event) => updateDictionary(index, { entries: safeJSON(event.target.value, dictionary.entries || []) as DictionaryEntry[] })} />
                </div>
              ))}
            </section>
          )}

          {activeTab === "payload" && (
            <section className="grid gap-3">
              <h3 className="text-sm font-semibold">样例报文</h3>
              <textarea className="min-h-[420px] resize-y rounded-md border border-line px-3 py-2 font-mono text-xs leading-5" value={payload} onChange={(event) => setPayload(event.target.value)} />
            </section>
          )}

          {activeTab === "result" && (
            <section className="grid gap-4">
              <h3 className="text-sm font-semibold">映射结果</h3>
              <div className="overflow-x-auto rounded-lg border border-line">
                <table className="w-full min-w-[720px] text-sm">
                  <thead className="bg-gray-50 text-xs text-muted">
                    <tr>{previewColumns.map((column) => <th key={column} className="px-3 py-2 text-left">{column}</th>)}</tr>
                  </thead>
                  <tbody>
                    {(preview?.rows?.length ? preview.rows : [{}]).map((row, rowIndex) => (
                      <tr key={rowIndex} className="border-t border-line">
                        {previewColumns.map((column) => <td key={column} className="px-3 py-2">{String(row[column] ?? "-")}</td>)}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              {result && <div className="rounded-lg border border-green-100 bg-green-50 px-4 py-3 text-sm text-green-800">患者 {result.patients.length}，就诊 {result.visits.length}，病历 {result.medicalRecords.length}；新增 {result.created}，更新 {result.updated}</div>}
            </section>
          )}
        </div>
      </main>
    </div>
  )
}

function normalizeSource(source: DataSource): DataSource {
  return {
    ...source,
    config: source.config || protocolDefaults(source.protocol),
    dictionaries: source.dictionaries || [],
    fieldMapping: normalizeMappings(source.fieldMapping || []),
  }
}

function normalizeMappings(mappings: FieldMapping[]): FieldMapping[] {
  return mappings.map((mapping) => ({ ...mapping, target: legacyTargetMap[mapping.target] || mapping.target }))
}

const legacyTargetMap: Record<string, string> = {
  patient_id: "patient.patientNo",
  patient_name: "patient.name",
  gender: "patient.gender",
  age: "patient.age",
  phone: "patient.phone",
  visit_id: "visit.visitNo",
  department_code: "visit.departmentCode",
  department_name: "visit.departmentName",
  diagnosis_code: "visit.diagnosisCode",
  diagnosis_name: "visit.diagnosisName",
  study_description: "record.studyDesc",
  study_uid: "record.studyUid",
}

function protocolDefaults(protocol: Protocol): Record<string, unknown> {
  if (protocol === "mysql" || protocol === "postgres") return { objectType: "table", schema: "", table: "", selectedFields: [], whereTemplate: "" }
  if (protocol === "grpc") return { proto: "", packageName: "", service: "", method: "", requestMessage: "", responseMessage: "" }
  if (protocol === "hl7") return { version: "2.5.1", messageTypes: ["ADT^A01"], segments: ["PID", "PV1", "OBR", "OBX"] }
  if (protocol === "dicom") return { service: "qido", aeTitle: "REPORTER", tags: ["0010,0020", "0010,0010", "0008,1030", "0020,000D"] }
  return { method: "GET", timeoutMs: 3000 }
}

function parsePayload(value: string) {
  const trimmed = value.trim()
  if (!trimmed) return null
  if (trimmed.startsWith("{") || trimmed.startsWith("[")) return JSON.parse(trimmed)
  return value
}

function safeJSON(value: string, fallback: unknown) {
  try {
    return JSON.parse(value)
  } catch {
    return fallback
  }
}

function Text({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="grid gap-1 text-sm">
      <span className="text-muted">{label}</span>
      <input className="h-10 rounded-md border border-line px-3 text-sm" value={value} onChange={(event) => onChange(event.target.value)} />
    </label>
  )
}
