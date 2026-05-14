import { useEffect, useMemo, useState } from "react"
import { authedJson } from "../lib/auth"

type ComponentKind =
  | "section"
  | "static_text"
  | "text"
  | "textarea"
  | "number"
  | "date"
  | "single_select"
  | "multi_select"
  | "rating"
  | "likert"
  | "matrix"
  | "table"
  | "remote_options"
  | "consent"

type BindingKind = "none" | "static" | "http" | "grpc" | "mysql" | "postgres" | "hl7" | "dicom" | "custom"

interface OptionItem {
  label: string
  value: string
}

interface DataBinding {
  kind: BindingKind
  dataSourceId?: string
  operation?: string
  params?: string
  valuePath?: string
  labelPath?: string
}

interface FormComponent {
  id: string
  type: ComponentKind
  label: string
  required: boolean
  category?: "原子组件" | "公共组件"
  helpText?: string
  placeholder?: string
  options?: OptionItem[]
  rows?: string[]
  columns?: string[]
  scale?: number
  binding?: DataBinding
  children?: FormComponent[]
}

interface Preset {
  id: string
  label: string
  hint: string
  scenario?: "调查" | "随访" | "术后" | "慢病" | "体检"
  components: FormComponent[]
}

interface FormLibraryResponse {
  templates: Preset[]
  commonComponents: Preset[]
  atomicComponents: Preset[]
}

const satisfactionOptions: OptionItem[] = [
  { label: "很不满意", value: "1" },
  { label: "不满意", value: "2" },
  { label: "一般", value: "3" },
  { label: "满意", value: "4" },
  { label: "非常满意", value: "5" },
]

const dataSources = [
  { id: "patients-api", name: "患者主索引 API", kind: "http" },
  { id: "survey-dict", name: "满意度字典库", kind: "mysql" },
  { id: "dept-grpc", name: "科室 gRPC 服务", kind: "grpc" },
  { id: "hl7-adt", name: "HL7 ADT 入院登记", kind: "hl7" },
  { id: "dicom-pacs", name: "DICOM/PACS 检查影像", kind: "dicom" },
]

const atomicComponents: Preset[] = [
  { id: "atom-section", label: "分组标题", hint: "用于组织题目区域", components: [{ id: "section", type: "section", label: "分组标题", required: false, category: "原子组件" }] },
  { id: "atom-text", label: "单行文本", hint: "姓名、编号、短文本", components: [{ id: "text", type: "text", label: "单行文本", required: false, category: "原子组件" }] },
  { id: "atom-textarea", label: "多行文本", hint: "主诉、意见建议", components: [{ id: "textarea", type: "textarea", label: "多行文本", required: false, category: "原子组件" }] },
  { id: "atom-number", label: "数字", hint: "年龄、评分、次数", components: [{ id: "number", type: "number", label: "数字", required: false, category: "原子组件" }] },
  { id: "atom-date", label: "日期", hint: "就诊、随访、手术日期", components: [{ id: "date", type: "date", label: "日期", required: false, category: "原子组件" }] },
  { id: "atom-select", label: "单选", hint: "本地或远程选项", components: [{ id: "single_select", type: "single_select", label: "单选", required: false, category: "原子组件", options: [{ label: "选项 A", value: "a" }, { label: "选项 B", value: "b" }] }] },
  { id: "atom-multi", label: "多选", hint: "多项原因或症状", components: [{ id: "multi_select", type: "multi_select", label: "多选", required: false, category: "原子组件", options: [{ label: "选项 A", value: "a" }, { label: "选项 B", value: "b" }] }] },
  { id: "atom-remote", label: "远程选项", hint: "来自数据库/API/gRPC/HL7/DICOM", components: [{ id: "remote_options", type: "remote_options", label: "远程选项", required: false, category: "原子组件", binding: { kind: "http", dataSourceId: "patients-api", operation: "GET /options", labelPath: "$.label", valuePath: "$.value" } }] },
  { id: "atom-rating", label: "评分", hint: "星级、NPS、疼痛评分", components: [{ id: "rating", type: "rating", label: "评分", required: false, category: "原子组件", scale: 5 }] },
  { id: "atom-matrix", label: "矩阵", hint: "多维度量表", components: [{ id: "matrix", type: "matrix", label: "矩阵评分", required: false, category: "原子组件", rows: ["评价项 1", "评价项 2"], columns: satisfactionOptions.map((item) => item.label) }] },
]

const commonComponents: Preset[] = [
  {
    id: "patient-basic",
    label: "患者基础信息",
    hint: "姓名、性别、年龄、手机号，可从主索引/API/HL7 ADT 回填",
    components: [
      { id: "patient_section", type: "section", label: "患者基础信息", required: false, category: "公共组件" },
      { id: "patient_name", type: "text", label: "患者姓名", required: true, category: "公共组件", binding: { kind: "http", dataSourceId: "patients-api", operation: "GET /patients/:patientId", valuePath: "$.name" } },
      { id: "patient_gender", type: "single_select", label: "性别", required: false, category: "公共组件", options: [{ label: "男", value: "male" }, { label: "女", value: "female" }, { label: "其他", value: "other" }], binding: { kind: "hl7", dataSourceId: "hl7-adt", operation: "PID-8", valuePath: "PID.8" } },
      { id: "patient_age", type: "number", label: "年龄", required: false, category: "公共组件", binding: { kind: "http", dataSourceId: "patients-api", operation: "GET /patients/:patientId", valuePath: "$.age" } },
      { id: "patient_phone", type: "text", label: "联系电话", required: false, category: "公共组件", binding: { kind: "http", dataSourceId: "patients-api", operation: "GET /patients/:patientId", valuePath: "$.phone" } },
    ],
  },
  {
    id: "visit-info",
    label: "就诊信息",
    hint: "科室、医生、就诊日期、诊断，支持 HIS/API/gRPC/HL7",
    components: [
      { id: "visit_section", type: "section", label: "就诊信息", required: false, category: "公共组件" },
      { id: "visit_date", type: "date", label: "就诊日期", required: true, category: "公共组件", binding: { kind: "hl7", dataSourceId: "hl7-adt", operation: "PV1-44", valuePath: "PV1.44" } },
      { id: "department", type: "remote_options", label: "就诊科室", required: true, category: "公共组件", binding: { kind: "grpc", dataSourceId: "dept-grpc", operation: "DepartmentService/ListDepartments", labelPath: "$.name", valuePath: "$.code" } },
      { id: "doctor_name", type: "text", label: "接诊医生", required: false, category: "公共组件", binding: { kind: "http", dataSourceId: "patients-api", operation: "GET /visits/:visitId", valuePath: "$.doctorName" } },
      { id: "diagnosis", type: "remote_options", label: "诊断", required: false, category: "公共组件", binding: { kind: "mysql", dataSourceId: "survey-dict", operation: "select label, value from diagnosis_dict where keyword like :keyword", labelPath: "$.label", valuePath: "$.value" } },
    ],
  },
  {
    id: "follow-up",
    label: "随访",
    hint: "随访方式、时间、症状、用药依从性",
    components: [
      { id: "follow_section", type: "section", label: "随访记录", required: false, category: "公共组件" },
      { id: "follow_date", type: "date", label: "随访日期", required: true, category: "公共组件" },
      { id: "follow_method", type: "single_select", label: "随访方式", required: true, category: "公共组件", options: [{ label: "电话", value: "phone" }, { label: "门诊", value: "clinic" }, { label: "线上", value: "online" }, { label: "上门", value: "home" }] },
      { id: "symptoms", type: "multi_select", label: "当前症状", required: false, category: "公共组件", binding: { kind: "mysql", dataSourceId: "survey-dict", operation: "select label, value from symptom_dict where disease_code = :diseaseCode", labelPath: "$.label", valuePath: "$.value" } },
      { id: "medication_adherence", type: "likert", label: "用药依从性", required: false, category: "公共组件", options: [{ label: "很差", value: "1" }, { label: "较差", value: "2" }, { label: "一般", value: "3" }, { label: "较好", value: "4" }, { label: "很好", value: "5" }] },
    ],
  },
  {
    id: "post-op",
    label: "术后跟踪",
    hint: "手术信息、切口恢复、疼痛评分、影像检查",
    components: [
      { id: "post_op_section", type: "section", label: "术后跟踪", required: false, category: "公共组件" },
      { id: "surgery_date", type: "date", label: "手术日期", required: true, category: "公共组件", binding: { kind: "hl7", dataSourceId: "hl7-adt", operation: "PR1-5", valuePath: "PR1.5" } },
      { id: "procedure_name", type: "text", label: "手术名称", required: true, category: "公共组件", binding: { kind: "hl7", dataSourceId: "hl7-adt", operation: "PR1-3", valuePath: "PR1.3" } },
      { id: "pain_score", type: "rating", label: "疼痛评分", required: true, category: "公共组件", scale: 10, helpText: "0 表示无痛，10 表示最剧烈疼痛。" },
      { id: "wound_status", type: "single_select", label: "切口恢复", required: false, category: "公共组件", options: [{ label: "良好", value: "good" }, { label: "红肿", value: "redness" }, { label: "渗液", value: "exudate" }, { label: "其他", value: "other" }] },
      { id: "image_followup", type: "remote_options", label: "相关影像检查", required: false, category: "公共组件", binding: { kind: "dicom", dataSourceId: "dicom-pacs", operation: "QIDO-RS /studies?PatientID=:patientId", labelPath: "$.StudyDescription", valuePath: "$.StudyInstanceUID" } },
    ],
  },
  {
    id: "satisfaction",
    label: "满意度",
    hint: "总体满意、分项矩阵、推荐意愿、原因和建议",
    components: [
      { id: "satisfaction_section", type: "section", label: "满意度评价", required: false, category: "公共组件" },
      { id: "overall_satisfaction", type: "likert", label: "总体满意度", required: true, category: "公共组件", options: satisfactionOptions, binding: { kind: "mysql", dataSourceId: "survey-dict", operation: "select label, value from survey_options where group_code = 'satisfaction'", labelPath: "$.label", valuePath: "$.value" } },
      { id: "service_matrix", type: "matrix", label: "分项满意度", required: true, category: "公共组件", rows: ["挂号缴费流程", "候诊时间", "医生沟通", "护士服务", "检查检验指引", "院内环境"], columns: satisfactionOptions.map((item) => item.label) },
      { id: "recommend_score", type: "rating", label: "推荐意愿", required: true, category: "公共组件", scale: 10, helpText: "0 表示完全不推荐，10 表示非常愿意推荐。" },
      { id: "problem_reasons", type: "multi_select", label: "不满意原因", required: false, category: "公共组件", binding: { kind: "mysql", dataSourceId: "survey-dict", operation: "select label, value from survey_options where group_code = 'dissatisfied_reason'", labelPath: "$.label", valuePath: "$.value" } },
      { id: "feedback", type: "textarea", label: "意见与建议", required: false, category: "公共组件", placeholder: "请填写您希望医院改进的地方" },
    ],
  },
]

const templates: Preset[] = [
  {
    id: "outpatient-satisfaction",
    label: "患者就诊满意度调查",
    hint: "由患者基础信息、就诊信息、满意度公共组件组合而成",
    scenario: "调查",
    components: [...commonComponents[0].components, ...commonComponents[1].components, ...commonComponents[4].components],
  },
  {
    id: "discharge-follow-up",
    label: "出院后随访问卷",
    hint: "出院患者基础信息、随访方式、症状、用药依从性和复诊提醒",
    scenario: "随访",
    components: [
      ...commonComponents[0].components,
      { id: "discharge_section", type: "section", label: "出院信息", required: false, category: "公共组件" },
      { id: "discharge_date", type: "date", label: "出院日期", required: true, category: "公共组件", binding: { kind: "hl7", dataSourceId: "hl7-adt", operation: "PV1-45", valuePath: "PV1.45" } },
      { id: "discharge_diagnosis", type: "remote_options", label: "出院诊断", required: false, category: "公共组件", binding: { kind: "http", dataSourceId: "patients-api", operation: "GET /discharges/:visitId", valuePath: "$.diagnosis" } },
      ...commonComponents[2].components,
      { id: "next_visit_date", type: "date", label: "建议复诊日期", required: false, category: "公共组件" },
      { id: "follow_note", type: "textarea", label: "随访备注", required: false, category: "公共组件" },
    ],
  },
  {
    id: "post-op-follow-up",
    label: "术后随访问卷",
    hint: "由患者基础信息、术后跟踪、随访公共组件组合而成",
    scenario: "术后",
    components: [...commonComponents[0].components, ...commonComponents[3].components, ...commonComponents[2].components],
  },
  {
    id: "hypertension-follow-up",
    label: "高血压慢病随访",
    hint: "血压、用药、症状、生活方式和复诊计划",
    scenario: "慢病",
    components: [
      ...commonComponents[0].components,
      ...commonComponents[2].components,
      { id: "bp_section", type: "section", label: "血压与生活方式", required: false, category: "公共组件" },
      { id: "systolic_bp", type: "number", label: "收缩压 mmHg", required: true, category: "公共组件" },
      { id: "diastolic_bp", type: "number", label: "舒张压 mmHg", required: true, category: "公共组件" },
      { id: "bp_control", type: "likert", label: "血压控制情况", required: false, category: "公共组件", options: [{ label: "很差", value: "1" }, { label: "偏差", value: "2" }, { label: "一般", value: "3" }, { label: "较好", value: "4" }, { label: "很好", value: "5" }] },
      { id: "lifestyle", type: "multi_select", label: "生活方式干预", required: false, category: "公共组件", options: [{ label: "限盐", value: "salt" }, { label: "规律运动", value: "exercise" }, { label: "控制体重", value: "weight" }, { label: "戒烟限酒", value: "smoke_alcohol" }] },
      { id: "adverse_reaction", type: "textarea", label: "药物不良反应", required: false, category: "公共组件" },
    ],
  },
  {
    id: "diabetes-management",
    label: "糖尿病管理随访",
    hint: "血糖、低血糖事件、饮食运动、足部和用药依从性",
    scenario: "慢病",
    components: [
      ...commonComponents[0].components,
      ...commonComponents[2].components,
      { id: "glucose_section", type: "section", label: "血糖管理", required: false, category: "公共组件" },
      { id: "fasting_glucose", type: "number", label: "空腹血糖 mmol/L", required: true, category: "公共组件" },
      { id: "postprandial_glucose", type: "number", label: "餐后 2 小时血糖 mmol/L", required: false, category: "公共组件" },
      { id: "hypoglycemia", type: "single_select", label: "近期低血糖事件", required: true, category: "公共组件", options: [{ label: "无", value: "none" }, { label: "1 次", value: "once" }, { label: "2 次及以上", value: "multiple" }] },
      { id: "diet_exercise", type: "matrix", label: "饮食与运动执行情况", required: false, category: "公共组件", rows: ["控制主食", "规律运动", "监测血糖", "足部护理"], columns: ["未执行", "偶尔", "基本做到", "完全做到"] },
      { id: "foot_problem", type: "textarea", label: "足部异常或其他问题", required: false, category: "公共组件" },
    ],
  },
  {
    id: "physical-exam-review",
    label: "体检异常复查登记",
    hint: "体检异常项、影像/检验关联、复查建议和结果跟踪",
    scenario: "体检",
    components: [
      ...commonComponents[0].components,
      { id: "exam_section", type: "section", label: "体检异常信息", required: false, category: "公共组件" },
      { id: "exam_date", type: "date", label: "体检日期", required: true, category: "公共组件" },
      { id: "abnormal_items", type: "multi_select", label: "异常项目", required: true, category: "公共组件", binding: { kind: "http", dataSourceId: "patients-api", operation: "GET /exam/:examId/abnormal-items", labelPath: "$.name", valuePath: "$.code" } },
      { id: "related_image", type: "remote_options", label: "相关影像", required: false, category: "公共组件", binding: { kind: "dicom", dataSourceId: "dicom-pacs", operation: "QIDO-RS /studies?PatientID=:patientId", labelPath: "$.StudyDescription", valuePath: "$.StudyInstanceUID" } },
      { id: "review_advice", type: "textarea", label: "复查建议", required: true, category: "公共组件" },
      { id: "review_date", type: "date", label: "计划复查日期", required: false, category: "公共组件" },
      { id: "review_result", type: "textarea", label: "复查结果", required: false, category: "公共组件" },
    ],
  },
]

const templateScenarios: Array<NonNullable<Preset["scenario"]>> = ["调查", "随访", "术后", "慢病", "体检"]
type LibraryTab = "templates" | "common" | "atoms"

const libraryTabs: Array<{ id: LibraryTab; label: string }> = [
  { id: "templates", label: "模板" },
  { id: "common", label: "公共" },
  { id: "atoms", label: "原子" },
]

function cloneComponents(components: FormComponent[], suffix = Date.now().toString(36)) {
  return components.map((item, index) => ({
    ...item,
    id: `${item.id}_${suffix}_${index + 1}`,
    options: item.options ? [...item.options] : undefined,
    rows: item.rows ? [...item.rows] : undefined,
    columns: item.columns ? [...item.columns] : undefined,
    binding: item.binding ? { ...item.binding } : { kind: "none" as BindingKind },
  }))
}

export function FormDesigner() {
  const [formName, setFormName] = useState("未命名表单")
  const [components, setComponents] = useState<FormComponent[]>([])
  const [selected, setSelected] = useState<string>("")
  const [activeTab, setActiveTab] = useState<LibraryTab>("templates")
  const [library, setLibrary] = useState<FormLibraryResponse>({ templates, commonComponents, atomicComponents })
  const [libraryMessage, setLibraryMessage] = useState("")
  const current = useMemo(() => components.find((item) => item.id === selected), [components, selected])
  const visibleTemplateGroups = useMemo(() => templateScenarios
    .map((scenario) => ({ scenario, templates: library.templates.filter((template) => template.scenario === scenario) }))
    .filter((group) => group.templates.length > 0), [library.templates])

  useEffect(() => {
    authedJson<FormLibraryResponse>("/api/v1/form-library")
      .then((data) => {
        setLibrary({
          templates: data.templates?.length ? data.templates : templates,
          commonComponents: data.commonComponents?.length ? data.commonComponents : commonComponents,
          atomicComponents: data.atomicComponents?.length ? data.atomicComponents : atomicComponents,
        })
        setLibraryMessage("")
      })
      .catch((error) => setLibraryMessage(`组件库使用本地兜底：${error instanceof Error ? error.message : "接口不可用"}`))
  }, [])

  function appendPreset(preset: Preset) {
    const next = cloneComponents(preset.components)
    setComponents([...components, ...next])
    setSelected(next[0]?.id || "")
  }

  function applyTemplate(template: Preset) {
    const next = cloneComponents(template.components, template.id)
    setFormName(template.label)
    setComponents(next)
    setSelected(next[0]?.id || "")
  }

  function updateCurrent(patch: Partial<FormComponent>) {
    setComponents(components.map((item) => (item.id === selected ? { ...item, ...patch } : item)))
  }

  function updateBinding(patch: Partial<DataBinding>) {
    updateCurrent({ binding: { kind: current?.binding?.kind || "none", ...current?.binding, ...patch } })
  }

  function updateList(key: "options" | "rows" | "columns", value: string) {
    const parts = value.split("\n").map((item) => item.trim()).filter(Boolean)
    if (key === "options") {
      updateCurrent({ options: parts.map((item, index) => ({ label: item, value: String(index + 1) })) })
      return
    }
    updateCurrent({ [key]: parts } as Partial<FormComponent>)
  }

  return (
    <div className="grid min-h-[760px] grid-cols-[300px_minmax(0,1fr)_360px] overflow-hidden rounded-lg border border-line bg-surface">
      <aside className="border-r border-line p-4">
        <h2 className="text-sm font-semibold">组件库</h2>
        {libraryMessage && <div className="mt-3 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">{libraryMessage}</div>}
        <div className="mt-4 grid grid-cols-3 gap-1 rounded-lg border border-line bg-gray-50 p-1">
          {libraryTabs.map((tab) => (
            <button
              key={tab.id}
              className={`rounded-md px-2 py-1.5 text-sm font-medium ${activeTab === tab.id ? "bg-white text-ink shadow-sm" : "text-muted hover:text-ink"}`}
              onClick={() => setActiveTab(tab.id)}
            >
              {tab.label}
            </button>
          ))}
        </div>
        <div className="mt-4 max-h-[650px] overflow-y-auto pr-1">
          {activeTab === "templates" && (
            <section>
            <h3 className="mb-2 text-xs font-semibold text-muted">业务模板</h3>
            <div className="grid gap-3">
              {visibleTemplateGroups.map((group) => (
                <div key={group.scenario}>
                  <div className="mb-1 text-[11px] font-medium text-muted">{group.scenario}</div>
                  <div className="grid gap-2">
                    {group.templates.map((template) => (
                      <button key={template.id} className="rounded-md border border-line px-3 py-2 text-left text-sm hover:border-primary hover:bg-blue-50" onClick={() => applyTemplate(template)}>
                        <span className="block font-medium">{template.label}</span>
                        <span className="mt-1 block text-xs text-muted">{template.hint}</span>
                      </button>
                    ))}
                  </div>
                </div>
              ))}
            </div>
          </section>
          )}

          {activeTab === "common" && (
            <section>
            <h3 className="mb-2 text-xs font-semibold text-muted">医疗公共组件</h3>
            <div className="grid gap-2">
              {library.commonComponents.map((preset) => (
                <button key={preset.id} className="rounded-md border border-line px-3 py-2 text-left text-sm hover:border-primary hover:bg-blue-50" onClick={() => appendPreset(preset)}>
                  <span className="block font-medium">{preset.label}</span>
                  <span className="mt-1 block text-xs text-muted">{preset.hint}</span>
                </button>
              ))}
            </div>
          </section>
          )}

          {activeTab === "atoms" && (
            <section>
            <h3 className="mb-2 text-xs font-semibold text-muted">原子组件</h3>
            <div className="grid gap-2">
              {library.atomicComponents.map((preset) => (
                <button key={preset.id} className="rounded-md border border-line px-3 py-2 text-left text-sm hover:border-primary hover:bg-blue-50" onClick={() => appendPreset(preset)}>
                  <span className="block font-medium">{preset.label}</span>
                  <span className="mt-1 block text-xs text-muted">{preset.hint}</span>
                </button>
              ))}
            </div>
          </section>
          )}
        </div>
      </aside>

      <section className="p-5">
        <div className="mb-4 flex items-center justify-between gap-4">
          <label className="grid gap-1">
            <span className="text-xs font-semibold text-muted">表单名称</span>
            <input className="w-[360px] rounded-md border border-line px-3 py-2 text-sm" value={formName} onChange={(event) => setFormName(event.target.value)} />
          </label>
          <button className="rounded-md bg-primary px-3 py-2 text-sm font-medium text-white">保存版本</button>
        </div>
        {components.length === 0 ? (
          <div className="flex min-h-[560px] items-center justify-center rounded-lg border-2 border-dashed border-line text-sm text-muted">
            从左侧选择医疗公共组件、原子组件，或套用业务模板。
          </div>
        ) : (
          <div className="grid gap-3">
            {components.map((item, index) => (
              <button key={`${item.id}-${index}`} className={`rounded-lg border px-4 py-3 text-left ${selected === item.id ? "border-primary bg-blue-50" : "border-line"}`} onClick={() => setSelected(item.id)}>
                <span className="flex items-center justify-between gap-3">
                  <span className="text-sm font-medium">{item.label}</span>
                  <span className="flex items-center gap-2">
                    {item.category && <span className="rounded-sm bg-gray-100 px-2 py-0.5 text-xs text-muted">{item.category}</span>}
                    {item.required && <span className="rounded-sm bg-red-50 px-2 py-0.5 text-xs text-danger">必填</span>}
                  </span>
                </span>
                <span className="mt-1 block text-xs text-muted">
                  {item.type}
                  {item.binding?.kind && item.binding.kind !== "none" ? ` · ${item.binding.kind} · ${item.binding.dataSourceId || "未选数据源"}` : " · 无绑定"}
                </span>
                {item.helpText && <span className="mt-2 block text-xs text-muted">{item.helpText}</span>}
              </button>
            ))}
          </div>
        )}
      </section>

      <aside className="border-l border-line p-4">
        <h2 className="text-sm font-semibold">属性与数据绑定</h2>
        {current ? (
          <div className="mt-4 grid gap-3 text-sm">
            <label className="grid gap-1">
              <span className="text-muted">字段名</span>
              <input className="rounded-md border border-line px-3 py-2" value={current.id} onChange={(event) => updateCurrent({ id: event.target.value })} />
            </label>
            <label className="grid gap-1">
              <span className="text-muted">题目/标签</span>
              <input className="rounded-md border border-line px-3 py-2" value={current.label} onChange={(event) => updateCurrent({ label: event.target.value })} />
            </label>
            <label className="grid gap-1">
              <span className="text-muted">说明</span>
              <textarea className="min-h-16 rounded-md border border-line px-3 py-2" value={current.helpText || ""} onChange={(event) => updateCurrent({ helpText: event.target.value })} />
            </label>
            {["text", "textarea", "number"].includes(current.type) && (
              <label className="grid gap-1">
                <span className="text-muted">占位提示</span>
                <input className="rounded-md border border-line px-3 py-2" value={current.placeholder || ""} onChange={(event) => updateCurrent({ placeholder: event.target.value })} />
              </label>
            )}
            {!["section", "static_text"].includes(current.type) && (
              <label className="flex items-center gap-2">
                <input type="checkbox" checked={current.required} onChange={(event) => updateCurrent({ required: event.target.checked })} />
                必填
              </label>
            )}
            {["single_select", "multi_select", "likert"].includes(current.type) && (
              <label className="grid gap-1">
                <span className="text-muted">静态选项，每行一个</span>
                <textarea className="min-h-28 rounded-md border border-line px-3 py-2" value={(current.options || []).map((item) => item.label).join("\n")} onChange={(event) => updateList("options", event.target.value)} />
              </label>
            )}
            {current.type === "rating" && (
              <label className="grid gap-1">
                <span className="text-muted">评分上限</span>
                <input type="number" min={3} max={10} className="rounded-md border border-line px-3 py-2" value={current.scale || 5} onChange={(event) => updateCurrent({ scale: Number(event.target.value) })} />
              </label>
            )}
            {current.type === "matrix" && (
              <>
                <label className="grid gap-1">
                  <span className="text-muted">矩阵行，每行一个评价项</span>
                  <textarea className="min-h-28 rounded-md border border-line px-3 py-2" value={(current.rows || []).join("\n")} onChange={(event) => updateList("rows", event.target.value)} />
                </label>
                <label className="grid gap-1">
                  <span className="text-muted">矩阵列，每行一个等级</span>
                  <textarea className="min-h-28 rounded-md border border-line px-3 py-2" value={(current.columns || []).join("\n")} onChange={(event) => updateList("columns", event.target.value)} />
                </label>
              </>
            )}

            <div className="grid gap-3 rounded-md border border-line bg-gray-50 p-3">
              <h3 className="text-xs font-semibold text-muted">数据绑定</h3>
              <label className="grid gap-1">
                <span className="text-muted">绑定类型</span>
                <select className="rounded-md border border-line bg-white px-3 py-2" value={current.binding?.kind || "none"} onChange={(event) => updateBinding({ kind: event.target.value as BindingKind })}>
                  <option value="none">不绑定</option>
                  <option value="static">静态值</option>
                  <option value="http">HTTP API</option>
                  <option value="grpc">gRPC</option>
                  <option value="mysql">MySQL</option>
                  <option value="postgres">PostgreSQL</option>
                  <option value="hl7">HL7</option>
                  <option value="dicom">DICOM</option>
                  <option value="custom">其他接口</option>
                </select>
              </label>
              <label className="grid gap-1">
                <span className="text-muted">数据源</span>
                <select className="rounded-md border border-line bg-white px-3 py-2" value={current.binding?.dataSourceId || ""} onChange={(event) => updateBinding({ dataSourceId: event.target.value })}>
                  <option value="">请选择</option>
                  {dataSources.map((source) => <option key={source.id} value={source.id}>{source.name}</option>)}
                </select>
              </label>
              <label className="grid gap-1">
                <span className="text-muted">操作/查询/消息路径</span>
                <input className="rounded-md border border-line bg-white px-3 py-2" value={current.binding?.operation || ""} onChange={(event) => updateBinding({ operation: event.target.value })} />
              </label>
              <label className="grid gap-1">
                <span className="text-muted">参数 JSON</span>
                <textarea className="min-h-16 rounded-md border border-line bg-white px-3 py-2" value={current.binding?.params || ""} onChange={(event) => updateBinding({ params: event.target.value })} placeholder='{"patientId":"{{context.patientId}}"}' />
              </label>
              <div className="grid grid-cols-2 gap-2">
                <label className="grid gap-1">
                  <span className="text-muted">显示路径</span>
                  <input className="rounded-md border border-line bg-white px-3 py-2" value={current.binding?.labelPath || ""} onChange={(event) => updateBinding({ labelPath: event.target.value })} placeholder="$.label / PID.5" />
                </label>
                <label className="grid gap-1">
                  <span className="text-muted">值路径</span>
                  <input className="rounded-md border border-line bg-white px-3 py-2" value={current.binding?.valuePath || ""} onChange={(event) => updateBinding({ valuePath: event.target.value })} placeholder="$.value / StudyUID" />
                </label>
              </div>
            </div>
          </div>
        ) : (
          <p className="mt-4 text-sm text-muted">请选择画布中的组件。</p>
        )}
      </aside>
    </div>
  )
}
