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
  | "computed"
  | "attachment"
  | "remote_options"
  | "consent"

type BindingKind = "none" | "static" | "http" | "grpc" | "mysql" | "postgres" | "hl7" | "dicom" | "custom"
type LogicRule = Record<string, unknown>

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
  config?: Record<string, unknown>
  binding?: DataBinding
  children?: FormComponent[]
  visibilityRules?: LogicRule
  jumpRules?: LogicRule
  validationRules?: LogicRule
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
interface FormVersion { id: string; formId: string; version: number; schema: FormComponent[]; published: boolean; createdAt: string }
interface ManagedForm { id: string; name: string; description: string; status: string; currentVersionId?: string; versions?: FormVersion[] }
type SurveyRuntime = {
  Model: typeof import("survey-core").Model
  Survey: typeof import("survey-react-ui").Survey
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
  { id: "atom-table", label: "多维表格", hint: "明细、指标、规格、用药等表格采集", components: [{ id: "sheet_table", type: "table", label: "多维表格", required: false, category: "原子组件", rows: ["记录 1"], columns: ["项目", "数值", "单位"], config: { addRows: true, addColumns: false } }] },
  { id: "atom-computed", label: "计算字段", hint: "按表达式自动计算分值、费用、风险", components: [{ id: "computed_score", type: "computed", label: "计算字段", required: false, category: "原子组件", config: { expression: "", precision: 2, readonly: true } }] },
  { id: "atom-attachment", label: "附件上传", hint: "文件、图片、视频、录音上传到对象存储", components: [{ id: "attachments", type: "attachment", label: "附件上传", required: false, category: "原子组件", config: { accept: "image/*,video/*,audio/*,application/pdf", maxSizeMb: 200, multiple: true } }] },
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
    id: "surveyjs-outpatient-satisfaction",
    label: "SurveyJS 门诊满意度模板",
    hint: "面向公开链接、微信和短信渠道的标准调查结构，支持矩阵、NPS、条件题和附件扩展",
    scenario: "调查",
    components: [
      ...commonComponents[0].components,
      ...commonComponents[1].components,
      ...commonComponents[4].components,
      { id: "surveyjs_attachment", type: "attachment", label: "补充材料", required: false, category: "公共组件", helpText: "可上传图片、录音或说明材料。", config: { accept: "image/*,audio/*,application/pdf", maxSizeMb: 50, multiple: true } },
    ],
  },
  {
    id: "surveyjs-nps",
    label: "SurveyJS NPS 推荐度调查",
    hint: "推荐意愿、原因追问、开放建议，适合快速满意度或体验净推荐值采集",
    scenario: "调查",
    components: [
      { id: "nps_section", type: "section", label: "推荐意愿", required: false, category: "公共组件" },
      { id: "recommend_score", type: "rating", label: "您愿意向亲友推荐本院服务吗？", required: true, category: "公共组件", scale: 10, helpText: "0 表示完全不推荐，10 表示非常愿意推荐。" },
      { id: "low_score_reason", type: "multi_select", label: "影响您推荐的主要原因", required: false, category: "公共组件", options: [{ label: "等待时间", value: "wait_time" }, { label: "沟通解释", value: "communication" }, { label: "流程指引", value: "guidance" }, { label: "费用体验", value: "billing" }, { label: "环境设施", value: "environment" }], visibilityRules: { when: { questionId: "recommend_score", operator: "less_than", value: "7" } } },
      { id: "nps_feedback", type: "textarea", label: "还有哪些改进建议？", required: false, category: "公共组件" },
    ],
  },
  {
    id: "surveyjs-registration-table",
    label: "SurveyJS 多维登记表",
    hint: "包含动态明细表、计算字段和附件，适合预约、登记、会务和宣传报名",
    scenario: "调查",
    components: [
      { id: "register_section", type: "section", label: "登记信息", required: false, category: "公共组件" },
      { id: "contact_name", type: "text", label: "联系人", required: true, category: "公共组件" },
      { id: "contact_phone", type: "text", label: "联系电话", required: true, category: "公共组件", validationRules: { regex: "^1\\d{10}$", message: "请输入 11 位手机号" } },
      { id: "items_table", type: "table", label: "报名/预约明细", required: false, category: "公共组件", rows: ["记录 1"], columns: ["项目", "人数", "备注"], config: { addRows: true, addColumns: false } },
      { id: "estimated_total", type: "computed", label: "预计人数", required: false, category: "公共组件", config: { expression: "sum(items_table.人数)", precision: 0, readonly: true } },
    ],
  },
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
type LibraryTab = "templates" | "common" | "atoms" | "bank"
type DesignerView = "structure" | "surveyPreview" | "surveyJson"
type ConditionOperator = "equals" | "not_equals" | "contains" | "empty" | "not_empty" | "greater_than" | "less_than"
interface SimpleCondition { questionId: string; operator: ConditionOperator; value: string }
interface JumpRuleVisual { when: SimpleCondition; goto: string }
interface ValidationVisual { min: string; max: string; regex: string; message: string }

const libraryTabs: Array<{ id: LibraryTab; label: string }> = [
  { id: "templates", label: "模板" },
  { id: "common", label: "公共" },
  { id: "atoms", label: "原子" },
  { id: "bank", label: "题库" },
]
const designFlow = [
  { title: "题库复用", text: "导入或选择标准题目，沉淀为可复用题库。" },
  { title: "问卷结构", text: "组合公共组件、原子组件和矩阵量表。" },
  { title: "逻辑规则", text: "配置跳转、关联显示、必填和校验规则。" },
  { title: "版本发布", text: "保存新版本，发布后线上问卷锁定版本。" },
  { title: "项目绑定", text: "项目表单数据页选择发布版本并生成渠道。" },
]
const conditionOperators: Array<{ id: ConditionOperator; label: string }> = [
  { id: "equals", label: "等于" },
  { id: "not_equals", label: "不等于" },
  { id: "contains", label: "包含" },
  { id: "empty", label: "为空" },
  { id: "not_empty", label: "不为空" },
  { id: "greater_than", label: "大于" },
  { id: "less_than", label: "小于" },
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
  const [forms, setForms] = useState<ManagedForm[]>([])
  const [formId, setFormId] = useState("")
  const [formStatus, setFormStatus] = useState("draft")
  const [formName, setFormName] = useState("未命名表单")
  const [formDescription, setFormDescription] = useState("")
  const [components, setComponents] = useState<FormComponent[]>([])
  const [selected, setSelected] = useState<string>("")
  const [activeTab, setActiveTab] = useState<LibraryTab>("templates")
  const [designerView, setDesignerView] = useState<DesignerView>("structure")
  const [surveyRuntime, setSurveyRuntime] = useState<SurveyRuntime | null>(null)
  const [library, setLibrary] = useState<FormLibraryResponse>({ templates, commonComponents, atomicComponents })
  const [libraryMessage, setLibraryMessage] = useState("")
  const [importText, setImportText] = useState("")
  const [surveyImportText, setSurveyImportText] = useState("")
  const current = useMemo(() => components.find((item) => item.id === selected), [components, selected])
  const surveyJson = useMemo(() => toSurveyJson(formName, formDescription, components), [formName, formDescription, components])
  const surveyModel = useMemo(() => {
    if (!surveyRuntime) return null
    const model = new surveyRuntime.Model(surveyJson)
    model.locale = "zh-cn"
    return model
  }, [surveyRuntime, surveyJson])
  const SurveyComponent = surveyRuntime?.Survey
  const visibleTemplateGroups = useMemo(() => templateScenarios
    .map((scenario) => ({ scenario, templates: library.templates.filter((template) => template.scenario === scenario) }))
    .filter((group) => group.templates.length > 0), [library.templates])

  useEffect(() => {
    Promise.all([
      authedJson<FormLibraryResponse>("/api/v1/form-library"),
      authedJson<ManagedForm[]>("/api/v1/forms").catch(() => []),
    ])
      .then(([data, formData]) => {
        setForms(formData)
        if (formData[0] && !formId) {
          setFormId(formData[0].id)
          setFormName(formData[0].name)
          setFormDescription(formData[0].description || "")
          setFormStatus(formData[0].status)
          const version = formData[0].versions?.find((item) => item.id === formData[0].currentVersionId) || formData[0].versions?.at(-1)
          if (version?.schema?.length) setComponents(version.schema)
        }
        return data
      })
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

  useEffect(() => {
    if (designerView !== "surveyPreview" || surveyRuntime) return
    let mounted = true
    Promise.all([
      import("survey-core"),
      import("survey-core/i18n/simplified-chinese"),
      import("survey-react-ui"),
      import("survey-core/survey-core.min.css"),
    ])
      .then(([core, , ui]) => {
        if (mounted) setSurveyRuntime({ Model: core.Model, Survey: ui.Survey })
      })
      .catch((error) => setLibraryMessage(`SurveyJS 预览加载失败：${error instanceof Error ? error.message : "依赖不可用"}`))
    return () => { mounted = false }
  }, [designerView, surveyRuntime])

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

  async function saveVersion() {
    let activeFormId = formId
    if (!activeFormId) {
      const form = await authedJson<ManagedForm>("/api/v1/forms", { method: "POST", body: JSON.stringify({ name: formName, description: formDescription }) })
      activeFormId = form.id
      setFormId(form.id)
      setForms([form, ...forms])
    }
    const version = await authedJson<FormVersion>(`/api/v1/forms/${activeFormId}/versions`, { method: "POST", body: JSON.stringify({ schema: components }) })
    await reloadForms(activeFormId)
    setFormStatus("draft")
    setLibraryMessage(`已保存第 ${version.version} 版，发布后该版本锁定；继续修改会生成新版本。`)
  }

  async function publishCurrent() {
    if (!formId) return
    const form = await authedJson<ManagedForm>(`/api/v1/forms/${formId}/publish`, { method: "POST" })
    setFormStatus(form.status)
    setForms(forms.map((item) => item.id === form.id ? form : item))
    setLibraryMessage("已发布当前版本，线上问卷将使用该版本。")
  }

  async function reloadForms(activeFormId = formId) {
    const nextForms = await authedJson<ManagedForm[]>("/api/v1/forms").catch(() => forms)
    setForms(nextForms)
    const active = nextForms.find((item) => item.id === activeFormId)
    if (active) {
      setFormStatus(active.status)
      const version = active.versions?.find((item) => item.id === active.currentVersionId) || active.versions?.at(-1)
      if (version?.schema?.length) setComponents(version.schema)
    }
  }

  function loadVersion(version: FormVersion) {
    setComponents(version.schema || [])
    setSelected(version.schema?.[0]?.id || "")
    setLibraryMessage(`已载入 v${version.version} ${version.published ? "发布版本" : "历史版本"}。修改后请保存为新版本，原版本不被覆盖。`)
  }

  async function importQuestionBank() {
    const imported = parseQuestionImport(importText)
    if (!imported.length) return setLibraryMessage("没有识别到可导入题目")
    const item = await authedJson<Preset>("/api/v1/form-library", {
      method: "POST",
      body: JSON.stringify({ id: `question-bank-${Date.now().toString(36)}`, kind: "common", label: "导入题库", hint: `导入 ${imported.length} 题`, scenario: "调查", components: imported, sortOrder: 90, enabled: true }),
    })
    setLibrary({ ...library, commonComponents: [item, ...library.commonComponents] })
    setImportText("")
    setLibraryMessage(`已导入 ${imported.length} 道题，可在公共组件和题库中复用。`)
  }

  function importSurveyJson() {
    const imported = parseSurveyJsonImport(surveyImportText)
    if (!imported.components.length) {
      setLibraryMessage("没有识别到可导入的 SurveyJS 题目")
      return
    }
    setFormName(imported.title || "导入的 SurveyJS 表单")
    setFormDescription(imported.description || "由 SurveyJS JSON 导入，已转换为平台表单 schema。")
    setComponents(imported.components)
    setSelected(imported.components[0]?.id || "")
    setSurveyImportText("")
    setDesignerView("structure")
    setLibraryMessage(`已导入 SurveyJS JSON：${imported.components.filter((item) => item.type !== "section").length} 个题目。请保存为新版本后再发布。`)
  }

  function updateList(key: "options" | "rows" | "columns", value: string) {
    const parts = value.split("\n").map((item) => item.trim()).filter(Boolean)
    if (key === "options") {
      updateCurrent({ options: parts.map((item, index) => ({ label: item, value: String(index + 1) })) })
      return
    }
    updateCurrent({ [key]: parts } as Partial<FormComponent>)
  }

  const currentForm = forms.find((form) => form.id === formId)
  const currentVersion = currentForm?.versions?.find((item) => item.id === currentForm.currentVersionId) || currentForm?.versions?.at(-1)

  return (
    <div className="grid gap-5">
      <section className="rounded-lg border border-line bg-surface p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 className="text-base font-semibold">问卷 / 表单设计闭环</h2>
            <p className="mt-1 text-sm text-muted">按题库、结构、逻辑、版本、项目绑定推进；发布后的公开问卷始终绑定指定版本。</p>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <span className="rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-primary">{formStatus === "published" ? `线上版本 v${currentVersion?.version || "-"}` : "草稿编辑中"}</span>
            <button className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-white" onClick={saveVersion}>保存新版本</button>
            <button className="rounded-lg border border-line bg-white px-4 py-2 text-sm" disabled={!formId} onClick={publishCurrent}>发布版本</button>
          </div>
        </div>
        <div className="mt-4 grid gap-3 lg:grid-cols-5">
          {designFlow.map((step, index) => <FlowStep key={step.title} index={index + 1} title={step.title} text={step.text} />)}
        </div>
      </section>
    <div className="grid min-h-[760px] grid-cols-[300px_minmax(0,1fr)_360px] overflow-hidden rounded-lg border border-line bg-surface">
      <aside className="border-r border-line p-4">
        <h2 className="text-sm font-semibold">组件库</h2>
        {libraryMessage && <div className="mt-3 rounded-md border border-amber-200 bg-amber-50 px-3 py-2 text-xs text-amber-700">{libraryMessage}</div>}
        <div className="mt-4 grid grid-cols-4 gap-1 rounded-lg border border-line bg-gray-50 p-1">
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
          {activeTab === "bank" && (
            <section>
              <h3 className="mb-2 text-xs font-semibold text-muted">题库复用与导入</h3>
              <textarea className="min-h-28 w-full rounded-md border border-line px-3 py-2 text-xs" placeholder="每行一题：字段ID,题目名称,题型,是否必填&#10;doctor_service,医生沟通,single_select,true" value={importText} onChange={(event) => setImportText(event.target.value)} />
              <button className="mt-2 w-full rounded-md bg-primary px-3 py-2 text-sm font-medium text-white" onClick={importQuestionBank}>导入题库</button>
              <div className="mt-4 rounded-lg border border-line bg-white p-3">
                <div className="text-xs font-semibold text-ink">导入 SurveyJS JSON</div>
                <div className="mt-1 text-xs leading-5 text-muted">支持从 SurveyJS 示例、历史问卷或外部模板导入；题目会转换为平台 schema，并继续走版本治理和项目发布。</div>
                <textarea
                  className="mt-3 min-h-32 w-full rounded-md border border-line px-3 py-2 font-mono text-xs"
                  placeholder='{"title":"Patient Satisfaction Survey","pages":[{"elements":[{"type":"rating","name":"recommend","title":"How likely are you to recommend us?"}]}]}'
                  value={surveyImportText}
                  onChange={(event) => setSurveyImportText(event.target.value)}
                />
                <button className="mt-2 w-full rounded-md border border-line bg-white px-3 py-2 text-sm font-medium text-primary" onClick={importSurveyJson}>转换为平台表单</button>
              </div>
              <div className="mt-3 grid gap-2">
                {[...library.commonComponents, ...library.atomicComponents].flatMap((preset) => preset.components.map((component) => ({ preset, component }))).filter(({ component }) => component.type !== "section").map(({ preset, component }) => (
                  <button key={`${preset.id}-${component.id}`} className="rounded-md border border-line px-3 py-2 text-left text-sm hover:border-primary hover:bg-blue-50" onClick={() => appendPreset({ id: component.id, label: component.label, hint: preset.label, components: [component] })}>
                    <span className="block font-medium">{component.label}</span>
                    <span className="mt-1 block text-xs text-muted">{component.type} · {preset.label}</span>
                  </button>
                ))}
              </div>
            </section>
          )}
        </div>
      </aside>

      <section className="p-5">
        <div className="mb-4 grid gap-3">
          <div className="grid gap-3 lg:grid-cols-[minmax(220px,1fr)_minmax(220px,1fr)]">
            <select className="h-11 rounded-md border border-line px-3 text-sm" value={formId} onChange={(event) => {
              const form = forms.find((item) => item.id === event.target.value)
              setFormId(event.target.value)
              if (form) {
                setFormName(form.name)
                setFormDescription(form.description || "")
                setFormStatus(form.status)
                const version = form.versions?.find((item) => item.id === form.currentVersionId) || form.versions?.at(-1)
                setComponents(version?.schema || [])
              }
            }}><option value="">新建表单</option>{forms.map((form) => <option key={form.id} value={form.id}>{form.name} · {form.status}</option>)}</select>
            <input className="h-11 rounded-md border border-line px-3 text-sm" value={formName} onChange={(event) => setFormName(event.target.value)} />
          </div>
          <textarea className="min-h-14 rounded-md border border-line px-3 py-2 text-sm" placeholder="版本说明 / 适用项目 / 调查期次" value={formDescription} onChange={(event) => setFormDescription(event.target.value)} />
        </div>
        <div className="mb-3 grid gap-2 rounded-md bg-gray-50 px-3 py-2 text-xs text-muted md:grid-cols-[1fr_auto]">
          <span>状态：{formStatus === "published" ? "已发布，线上版本已锁定；继续保存会生成新版本" : "草稿，可继续编辑"} · 当前题目 {components.filter((item) => item.type !== "section").length} 个 · 矩阵 {components.filter((item) => item.type === "matrix").length} 个</span>
          <span>{currentForm?.versions?.length ? `历史版本 ${currentForm.versions.length} 个` : "尚未保存版本"}</span>
        </div>
        <div className="mb-4 flex flex-wrap items-center justify-between gap-3 rounded-lg border border-line bg-white p-3">
          <div>
            <div className="text-sm font-semibold">SurveyJS 兼容预览</div>
            <div className="mt-1 text-xs text-muted">当前表单 schema 会实时转换为 SurveyJS JSON，便于预览复杂问卷、矩阵、文件和条件显示。</div>
          </div>
          <div className="grid grid-cols-3 gap-1 rounded-lg border border-line bg-gray-50 p-1 text-sm">
            {[
              { id: "structure", label: "结构" },
              { id: "surveyPreview", label: "预览" },
              { id: "surveyJson", label: "JSON" },
            ].map((item) => (
              <button
                key={item.id}
                className={`rounded-md px-3 py-1.5 ${designerView === item.id ? "bg-white text-primary shadow-sm" : "text-muted hover:text-ink"}`}
                onClick={() => setDesignerView(item.id as DesignerView)}
              >
                {item.label}
              </button>
            ))}
          </div>
        </div>
        {!!currentForm?.versions?.length && <div className="mb-4 rounded-lg border border-line bg-white p-3">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div className="text-sm font-semibold">版本治理</div>
              <div className="mt-1 text-xs text-muted">发布版本会锁定给公开问卷和项目渠道使用；历史版本只能载入复制，不能原地覆盖。</div>
            </div>
            <button className="rounded-lg border border-line px-3 py-1.5 text-xs text-primary" onClick={() => reloadForms()}>刷新版本</button>
          </div>
          <div className="mt-3 grid gap-2 md:grid-cols-2 xl:grid-cols-3">
            {currentForm.versions.map((version) => {
              const isCurrent = version.id === currentForm.currentVersionId
              return <button key={version.id} className={`rounded-lg border p-3 text-left text-xs ${isCurrent ? "border-primary bg-blue-50 text-primary" : "border-line text-muted"}`} onClick={() => loadVersion(version)}>
                <span className="flex items-center justify-between gap-2">
                  <span className="font-semibold text-ink">v{version.version}</span>
                  <span className={`rounded-full px-2 py-0.5 ${version.published ? "bg-green-50 text-green-700" : "bg-gray-100 text-muted"}`}>{version.published ? "已发布 / 已锁定" : "草稿快照"}</span>
                </span>
                <span className="mt-2 block text-muted">{new Date(version.createdAt).toLocaleString()}</span>
                <span className="mt-1 block text-muted">{version.schema?.filter((item) => item.type !== "section").length || 0} 个题目 · 点击载入后另存新版本</span>
              </button>
            })}
          </div>
        </div>}
        {designerView === "surveyPreview" ? (
          <div className="min-h-[560px] rounded-lg border border-line bg-white p-3">
            {SurveyComponent && surveyModel ? <SurveyComponent model={surveyModel} /> : <div className="flex min-h-[520px] items-center justify-center text-sm text-muted">正在加载 SurveyJS 预览...</div>}
          </div>
        ) : designerView === "surveyJson" ? (
          <pre className="max-h-[620px] overflow-auto rounded-lg border border-line bg-slate-950 p-4 text-xs leading-5 text-slate-100">{JSON.stringify(surveyJson, null, 2)}</pre>
        ) : components.length === 0 ? (
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
            {(current.type === "matrix" || current.type === "table") && (
              <>
                <label className="grid gap-1">
                  <span className="text-muted">{current.type === "table" ? "表格默认行，每行一个" : "矩阵行，每行一个评价项"}</span>
                  <textarea className="min-h-28 rounded-md border border-line px-3 py-2" value={(current.rows || []).join("\n")} onChange={(event) => updateList("rows", event.target.value)} />
                </label>
                <label className="grid gap-1">
                  <span className="text-muted">{current.type === "table" ? "表格列，每行一个字段" : "矩阵列，每行一个等级"}</span>
                  <textarea className="min-h-28 rounded-md border border-line px-3 py-2" value={(current.columns || []).join("\n")} onChange={(event) => updateList("columns", event.target.value)} />
                </label>
              </>
            )}
            {current.type === "computed" && (
              <div className="grid gap-2 rounded-md border border-line bg-gray-50 p-3">
                <h3 className="text-xs font-semibold text-muted">计算字段</h3>
                <label className="grid gap-1">
                  <span className="text-muted">表达式</span>
                  <input className="rounded-md border border-line bg-white px-3 py-2" value={String(current.config?.expression || "")} placeholder="overall_satisfaction * 20" onChange={(event) => updateCurrent({ config: { ...(current.config || {}), expression: event.target.value } })} />
                </label>
                <label className="grid gap-1">
                  <span className="text-muted">小数位</span>
                  <input type="number" min={0} max={6} className="rounded-md border border-line bg-white px-3 py-2" value={Number(current.config?.precision ?? 2)} onChange={(event) => updateCurrent({ config: { ...(current.config || {}), precision: Number(event.target.value) } })} />
                </label>
              </div>
            )}
            {current.type === "attachment" && (
              <div className="grid gap-2 rounded-md border border-line bg-gray-50 p-3">
                <h3 className="text-xs font-semibold text-muted">多模态附件</h3>
                <label className="grid gap-1">
                  <span className="text-muted">允许类型</span>
                  <input className="rounded-md border border-line bg-white px-3 py-2" value={String(current.config?.accept || "")} placeholder="image/*,video/*,audio/*,application/pdf" onChange={(event) => updateCurrent({ config: { ...(current.config || {}), accept: event.target.value } })} />
                </label>
                <label className="grid gap-1">
                  <span className="text-muted">最大大小 MB</span>
                  <input type="number" min={1} max={500} className="rounded-md border border-line bg-white px-3 py-2" value={Number(current.config?.maxSizeMb ?? 200)} onChange={(event) => updateCurrent({ config: { ...(current.config || {}), maxSizeMb: Number(event.target.value) } })} />
                </label>
                <label className="flex items-center gap-2"><input type="checkbox" checked={Boolean(current.config?.multiple ?? true)} onChange={(event) => updateCurrent({ config: { ...(current.config || {}), multiple: event.target.checked } })} />允许多文件</label>
              </div>
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
            <div className="grid gap-3 rounded-md border border-line bg-gray-50 p-3">
              <h3 className="text-xs font-semibold text-muted">跳转 / 关联 / 校验规则</h3>
              <ConditionEditor
                title="关联显示条件"
                hint="满足条件时显示当前题；不配置则始终显示。"
                components={components}
                currentId={current.id}
                value={conditionFromVisibility(current.visibilityRules)}
                onChange={(value) => updateCurrent({ visibilityRules: visibilityToRule(value) })}
              />
              <JumpRuleEditor
                components={components}
                currentId={current.id}
                value={jumpFromRule(current.jumpRules)}
                onChange={(value) => updateCurrent({ jumpRules: jumpToRule(value) })}
              />
              <ValidationEditor value={validationFromRule(current.validationRules)} onChange={(value) => updateCurrent({ validationRules: validationToRule(value) })} />
              <details className="rounded-md border border-line bg-white p-3">
                <summary className="cursor-pointer text-xs font-medium text-muted">高级 JSON</summary>
                <div className="mt-3 grid gap-2">
                  <label className="grid gap-1"><span className="text-muted">显示条件 JSON</span><textarea className="min-h-16 rounded-md border border-line bg-white px-3 py-2 font-mono text-xs" value={JSON.stringify(current.visibilityRules || {}, null, 2)} onChange={(event) => updateCurrent({ visibilityRules: safeJSON(event.target.value, current.visibilityRules || {}) as LogicRule })} /></label>
                  <label className="grid gap-1"><span className="text-muted">跳转逻辑 JSON</span><textarea className="min-h-16 rounded-md border border-line bg-white px-3 py-2 font-mono text-xs" value={JSON.stringify(current.jumpRules || {}, null, 2)} onChange={(event) => updateCurrent({ jumpRules: safeJSON(event.target.value, current.jumpRules || {}) as LogicRule })} /></label>
                  <label className="grid gap-1"><span className="text-muted">复杂校验 JSON</span><textarea className="min-h-16 rounded-md border border-line bg-white px-3 py-2 font-mono text-xs" value={JSON.stringify(current.validationRules || {}, null, 2)} onChange={(event) => updateCurrent({ validationRules: safeJSON(event.target.value, current.validationRules || {}) as LogicRule })} /></label>
                </div>
              </details>
            </div>
          </div>
        ) : (
          <p className="mt-4 text-sm text-muted">请选择画布中的组件。</p>
        )}
      </aside>
    </div>
    </div>
  )
}

function toSurveyJson(title: string, description: string, components: FormComponent[]) {
  const page = { name: "page1", title: title || "未命名表单", description: description || "", elements: [] as Array<Record<string, unknown>> }
  let currentPanel: Record<string, unknown> | null = null

  components.forEach((component) => {
    if (component.type === "section") {
      currentPanel = { type: "panel", name: surveyName(component.id), title: component.label, elements: [] as Array<Record<string, unknown>> }
      page.elements.push(currentPanel)
      return
    }
    const element = toSurveyElement(component)
    if (!element) return
    if (currentPanel && Array.isArray(currentPanel.elements)) {
      currentPanel.elements.push(element)
    } else {
      page.elements.push(element)
    }
  })

  return {
    title: title || "未命名表单",
    description: description || "",
    locale: "zh-cn",
    showQuestionNumbers: "on",
    questionErrorLocation: "bottom",
    requiredText: "*",
    pagePrevText: "上一页",
    pageNextText: "下一页",
    completeText: "提交",
    previewText: "预览",
    editText: "编辑",
    clearInvisibleValues: "onHidden",
    textUpdateMode: "onTyping",
    pages: [page],
  }
}

function toSurveyElement(component: FormComponent): Record<string, unknown> | null {
  const base: Record<string, unknown> = {
    name: surveyName(component.id),
    title: component.label,
    description: component.helpText || undefined,
    isRequired: component.required || undefined,
    placeholder: component.placeholder || undefined,
    visibleIf: surveyVisibleIf(component.visibilityRules),
    reporterComponentType: component.type,
    reporterBinding: component.binding && component.binding.kind !== "none" ? component.binding : undefined,
  }
  const validators = surveyValidators(component)
  if (validators.length) base.validators = validators

  switch (component.type) {
    case "static_text":
      return { type: "html", name: surveyName(component.id), html: component.helpText || component.label }
    case "text":
      return { ...base, type: "text" }
    case "textarea":
      return { ...base, type: "comment", rows: 4 }
    case "number":
      return { ...base, type: "text", inputType: "number" }
    case "date":
      return { ...base, type: "text", inputType: "date" }
    case "single_select":
    case "likert":
    case "remote_options":
      return { ...base, type: "dropdown", choices: surveyChoices(component.options) }
    case "multi_select":
      return { ...base, type: "checkbox", choices: surveyChoices(component.options) }
    case "rating":
      return { ...base, type: "rating", rateMin: 1, rateMax: component.scale || 5, minRateDescription: "低", maxRateDescription: "高" }
    case "matrix":
      return {
        ...base,
        type: "matrix",
        rows: (component.rows?.length ? component.rows : ["评价项"]).map((row, index) => ({ value: surveyValue(row, index), text: row })),
        columns: (component.columns?.length ? component.columns : satisfactionOptions.map((item) => item.label)).map((column, index) => ({ value: String(index + 1), text: column })),
      }
    case "table":
      return {
        ...base,
        type: "matrixdynamic",
        rowCount: Math.max(1, component.rows?.length || 1),
        addRowText: "添加一行",
        removeRowText: "删除",
        noRowsText: "暂无明细，请添加一行",
        columns: (component.columns?.length ? component.columns : ["项目", "数值"]).map((column) => ({ name: surveyName(column), title: column, cellType: "text" })),
      }
    case "computed":
      return { ...base, type: "expression", expression: String(component.config?.expression || "") }
    case "attachment":
      return {
        ...base,
        type: "file",
        waitForUpload: true,
        storeDataAsText: false,
        titleLocation: "top",
        allowMultiple: Boolean(component.config?.multiple ?? true),
        acceptedTypes: String(component.config?.accept || ""),
        maxSize: Number(component.config?.maxSizeMb || 200) * 1024 * 1024,
      }
    case "consent":
      return { ...base, type: "boolean", labelTrue: "同意", labelFalse: "不同意" }
    default:
      return null
  }
}

function parseSurveyJsonImport(text: string): { title: string; description: string; components: FormComponent[] } {
  const raw = safeJSON(text, null)
  if (!raw || typeof raw !== "object") return { title: "", description: "", components: [] }
  const survey = raw as Record<string, unknown>
  const components: FormComponent[] = []
  const pages = Array.isArray(survey.pages) ? survey.pages : [{ elements: survey.elements }]
  pages.forEach((page, pageIndex) => {
    const pageRecord = ruleRecord(page) || {}
    const pageTitle = localizeSurveyText(pageRecord.title || pageRecord.name || (pages.length > 1 ? `第 ${pageIndex + 1} 页` : ""))
    if (pageTitle) {
      components.push({ id: surveyName(`section_${pageIndex + 1}`), type: "section", label: pageTitle, required: false, category: "公共组件" })
    }
    collectSurveyElements(pageRecord.elements, components)
  })
  return {
    title: localizeSurveyText(survey.title || survey.name || "导入的 SurveyJS 表单"),
    description: localizeSurveyText(survey.description || ""),
    components,
  }
}

function collectSurveyElements(elements: unknown, components: FormComponent[]) {
  if (!Array.isArray(elements)) return
  elements.forEach((element, index) => {
    const record = ruleRecord(element)
    if (!record) return
    const type = stringValue(record.type)
    if (type === "panel" || type === "paneldynamic") {
      components.push({
        id: surveyName(stringValue(record.name) || `panel_${index + 1}`),
        type: "section",
        label: localizeSurveyText(record.title || record.name || `分组 ${index + 1}`),
        required: false,
        category: "公共组件",
      })
      collectSurveyElements(record.elements || record.templateElements, components)
      return
    }
    const converted = surveyElementToComponent(record, index)
    if (converted) components.push(converted)
  })
}

function surveyElementToComponent(record: Record<string, unknown>, index: number): FormComponent | null {
  const sourceType = stringValue(record.type)
  const id = surveyName(stringValue(record.name) || `question_${index + 1}`)
  const label = localizeSurveyText(record.title || record.name || `题目 ${index + 1}`)
  const common = {
    id,
    label,
    required: Boolean(record.isRequired),
    category: "公共组件" as const,
    helpText: localizeSurveyText(record.description || ""),
    placeholder: localizeSurveyText(record.placeholder || ""),
    visibilityRules: surveyVisibleToRule(stringValue(record.visibleIf)),
    validationRules: surveyValidationToRule(record.validators),
    config: { surveyjs: { sourceType } },
  }

  switch (sourceType) {
    case "html":
      return { ...common, type: "static_text", helpText: localizeSurveyText(record.html || record.title || "") }
    case "comment":
      return { ...common, type: "textarea" }
    case "text":
      return { ...common, type: stringValue(record.inputType) === "date" ? "date" : stringValue(record.inputType) === "number" ? "number" : "text" }
    case "dropdown":
    case "radiogroup":
    case "buttongroup":
      return { ...common, type: "single_select", options: surveyOptionsToItems(record.choices) }
    case "checkbox":
      return { ...common, type: "multi_select", options: surveyOptionsToItems(record.choices) }
    case "rating":
      return { ...common, type: "rating", scale: Number(record.rateMax || 5) }
    case "matrix":
      return { ...common, type: "matrix", rows: surveyRowsToLabels(record.rows), columns: surveyRowsToLabels(record.columns) }
    case "matrixdynamic":
    case "matrixdropdown":
      return { ...common, type: "table", rows: ["记录 1"], columns: surveyMatrixColumns(record.columns), config: { ...(common.config || {}), addRows: sourceType === "matrixdynamic", addColumns: false } }
    case "file":
      return { ...common, type: "attachment", config: { ...(common.config || {}), accept: stringValue(record.acceptedTypes || ""), multiple: Boolean(record.allowMultiple) } }
    case "boolean":
      return { ...common, type: "consent" }
    case "expression":
      return { ...common, type: "computed", config: { ...(common.config || {}), expression: stringValue(record.expression), readonly: true } }
    default:
      return { ...common, type: "text" }
  }
}

function surveyOptionsToItems(value: unknown): OptionItem[] {
  if (!Array.isArray(value)) return []
  return value.map((item, index) => {
    if (typeof item === "string" || typeof item === "number") return { label: localizeSurveyText(item), value: String(item) || String(index + 1) }
    const record = ruleRecord(item) || {}
    return { label: localizeSurveyText(record.text || record.title || record.value || `选项 ${index + 1}`), value: stringValue(record.value || index + 1) }
  })
}

function surveyRowsToLabels(value: unknown): string[] {
  if (!Array.isArray(value)) return []
  return value.map((item, index) => {
    if (typeof item === "string" || typeof item === "number") return localizeSurveyText(item)
    const record = ruleRecord(item) || {}
    return localizeSurveyText(record.text || record.title || record.value || `项目 ${index + 1}`)
  })
}

function surveyMatrixColumns(value: unknown): string[] {
  const labels = surveyRowsToLabels(value)
  return labels.length ? labels : ["项目", "数值", "备注"]
}

function surveyValidationToRule(value: unknown): LogicRule {
  if (!Array.isArray(value)) return {}
  const rule: LogicRule = {}
  value.forEach((item) => {
    const record = ruleRecord(item)
    if (!record) return
    const type = stringValue(record.type)
    if (type === "regex") {
      rule.regex = stringValue(record.regex)
      rule.message = localizeSurveyText(record.text || record.message || "")
    }
    if (type === "numeric") {
      if (record.minValue !== undefined) rule.min = record.minValue
      if (record.maxValue !== undefined) rule.max = record.maxValue
      if (record.text || record.message) rule.message = localizeSurveyText(record.text || record.message)
    }
    if (type === "text") {
      if (record.minLength !== undefined) rule.min = record.minLength
      if (record.maxLength !== undefined) rule.max = record.maxLength
      if (record.text || record.message) rule.message = localizeSurveyText(record.text || record.message)
    }
  })
  return rule
}

function surveyVisibleToRule(value: string): LogicRule {
  const match = value.match(/\{([^}]+)\}\s*(=|!=|>|<|contains)\s*'?([^']*)'?/)
  if (!match) return {}
  const operatorMap: Record<string, ConditionOperator> = { "=": "equals", "!=": "not_equals", ">": "greater_than", "<": "less_than", contains: "contains" }
  return { when: { questionId: match[1], operator: operatorMap[match[2]] || "equals", value: match[3] || "" } }
}

function surveyChoices(options?: OptionItem[]) {
  const source = options?.length ? options : satisfactionOptions
  return source.map((item, index) => ({ value: item.value || String(index + 1), text: item.label || item.value }))
}

function surveyVisibleIf(rule?: LogicRule) {
  const condition = conditionFromVisibility(rule)
  if (!condition.questionId) return undefined
  const field = surveyName(condition.questionId)
  const value = condition.value.replace(/'/g, "\\'")
  switch (condition.operator) {
    case "not_equals":
      return `{${field}} != '${value}'`
    case "contains":
      return `{${field}} contains '${value}'`
    case "empty":
      return `{${field}} empty`
    case "not_empty":
      return `{${field}} notempty`
    case "greater_than":
      return `{${field}} > ${Number.isFinite(Number(condition.value)) ? condition.value : `'${value}'`}`
    case "less_than":
      return `{${field}} < ${Number.isFinite(Number(condition.value)) ? condition.value : `'${value}'`}`
    default:
      return `{${field}} = '${value}'`
  }
}

function surveyValidators(component: FormComponent) {
  const rule = validationFromRule(component.validationRules)
  const validators: Array<Record<string, unknown>> = []
  if (rule.regex.trim()) validators.push({ type: "regex", regex: rule.regex.trim(), text: rule.message || "格式不正确" })
  if (rule.min !== "" || rule.max !== "") {
    if (component.type === "number" || component.type === "rating") {
      validators.push({ type: "numeric", minValue: rule.min === "" ? undefined : Number(rule.min), maxValue: rule.max === "" ? undefined : Number(rule.max), text: rule.message || "数值超出范围" })
    } else {
      validators.push({ type: "text", minLength: rule.min === "" ? undefined : Number(rule.min), maxLength: rule.max === "" ? undefined : Number(rule.max), text: rule.message || "长度不符合要求" })
    }
  }
  return validators
}

function surveyName(value: string) {
  const normalized = value.trim().replace(/[^\w\u4e00-\u9fa5]+/g, "_").replace(/^_+|_+$/g, "")
  return normalized || "field"
}

function surveyValue(value: string, index: number) {
  const normalized = value.trim().replace(/[^\w\u4e00-\u9fa5]+/g, "_").replace(/^_+|_+$/g, "")
  return normalized || String(index + 1)
}

function localizeSurveyText(value: unknown): string {
  if (value === undefined || value === null) return ""
  const text = String(value).trim()
  if (!text) return ""
  const translations: Record<string, string> = {
    "Patient Satisfaction Survey": "患者满意度调查",
    "Outpatient Satisfaction Survey": "门诊满意度调查",
    "How likely are you to recommend us?": "您愿意向亲友推荐本院服务吗？",
    "How likely are you to recommend our service?": "您愿意向亲友推荐我们的服务吗？",
    "Overall satisfaction": "总体满意度",
    "Doctor communication": "医生沟通",
    "Nurse service": "护士服务",
    "Waiting time": "候诊时间",
    "Registration": "挂号缴费",
    "Environment": "环境设施",
    "Feedback": "意见与建议",
    "Please leave a comment": "请填写意见或建议",
    "Other": "其他",
    "None": "无",
    "Yes": "是",
    "No": "否",
    "Very dissatisfied": "很不满意",
    "Dissatisfied": "不满意",
    "Neutral": "一般",
    "Satisfied": "满意",
    "Very satisfied": "非常满意",
    "Poor": "差",
    "Fair": "一般",
    "Good": "好",
    "Excellent": "很好",
    "Name": "姓名",
    "Phone": "联系电话",
    "Mobile": "手机号",
    "Gender": "性别",
    "Age": "年龄",
    "Department": "科室",
    "Doctor": "医生",
    "Visit date": "就诊日期",
    "Diagnosis": "诊断",
    "Upload file": "上传附件",
  }
  return translations[text] || text
}

function FlowStep({ index, title, text }: { index: number; title: string; text: string }) {
  return <div className="rounded-lg border border-line bg-white p-3">
    <div className="flex items-center gap-2"><span className="grid h-6 w-6 place-items-center rounded-full bg-blue-50 text-xs font-semibold text-primary">{index}</span><span className="font-medium">{title}</span></div>
    <div className="mt-2 text-xs leading-5 text-muted">{text}</div>
  </div>
}

function ConditionEditor({ title, hint, components, currentId, value, onChange }: { title: string; hint: string; components: FormComponent[]; currentId: string; value: SimpleCondition; onChange: (value: SimpleCondition) => void }) {
  const sourceOptions = components.filter((item) => item.id !== currentId && !["section", "static_text"].includes(item.type))
  const needsValue = !["empty", "not_empty"].includes(value.operator)
  return <div className="grid gap-2 rounded-md border border-line bg-white p-3">
    <div><div className="text-xs font-semibold text-ink">{title}</div><div className="mt-1 text-xs text-muted">{hint}</div></div>
    <div className="grid gap-2">
      <select className="h-9 rounded-md border border-line px-2 text-sm" value={value.questionId} onChange={(event) => onChange({ ...value, questionId: event.target.value })}>
        <option value="">不配置条件</option>
        {sourceOptions.map((item) => <option key={item.id} value={item.id}>{item.label} · {item.id}</option>)}
      </select>
      {value.questionId && (
        <div className="grid grid-cols-[110px_minmax(0,1fr)] gap-2">
          <select className="h-9 rounded-md border border-line px-2 text-sm" value={value.operator} onChange={(event) => onChange({ ...value, operator: event.target.value as ConditionOperator })}>
            {conditionOperators.map((operator) => <option key={operator.id} value={operator.id}>{operator.label}</option>)}
          </select>
          {needsValue ? <input className="h-9 rounded-md border border-line px-2 text-sm" placeholder="匹配值，如 phone / 1 / 满意" value={value.value} onChange={(event) => onChange({ ...value, value: event.target.value })} /> : <div className="flex h-9 items-center rounded-md bg-gray-50 px-2 text-xs text-muted">无需填写匹配值</div>}
        </div>
      )}
    </div>
  </div>
}

function JumpRuleEditor({ components, currentId, value, onChange }: { components: FormComponent[]; currentId: string; value: JumpRuleVisual; onChange: (value: JumpRuleVisual) => void }) {
  return <div className="grid gap-2 rounded-md border border-line bg-white p-3">
    <div><div className="text-xs font-semibold text-ink">跳转逻辑</div><div className="mt-1 text-xs text-muted">当前题满足条件后，下一步跳到指定题。</div></div>
    <ConditionEditor title="触发条件" hint="选择当前题或其他题作为跳转判断条件。" components={components} currentId="" value={value.when} onChange={(when) => onChange({ ...value, when })} />
    <label className="grid gap-1">
      <span className="text-xs text-muted">跳转到</span>
      <select className="h-9 rounded-md border border-line px-2 text-sm" value={value.goto} onChange={(event) => onChange({ ...value, goto: event.target.value })}>
        <option value="">不跳转</option>
        {components.filter((item) => item.id !== currentId).map((item) => <option key={item.id} value={item.id}>{item.label} · {item.id}</option>)}
        <option value="__submit__">直接提交</option>
      </select>
    </label>
  </div>
}

function ValidationEditor({ value, onChange }: { value: ValidationVisual; onChange: (value: ValidationVisual) => void }) {
  return <div className="grid gap-2 rounded-md border border-line bg-white p-3">
    <div><div className="text-xs font-semibold text-ink">输入校验</div><div className="mt-1 text-xs text-muted">用于限制数字范围、文本格式，并给出患者可读的提示。</div></div>
    <div className="grid grid-cols-2 gap-2">
      <input className="h-9 rounded-md border border-line px-2 text-sm" placeholder="最小值/最短长度" value={value.min} onChange={(event) => onChange({ ...value, min: event.target.value })} />
      <input className="h-9 rounded-md border border-line px-2 text-sm" placeholder="最大值/最长长度" value={value.max} onChange={(event) => onChange({ ...value, max: event.target.value })} />
    </div>
    <input className="h-9 rounded-md border border-line px-2 text-sm" placeholder="正则表达式，如 ^1\\d{10}$" value={value.regex} onChange={(event) => onChange({ ...value, regex: event.target.value })} />
    <input className="h-9 rounded-md border border-line px-2 text-sm" placeholder="错误提示，如 请输入 11 位手机号" value={value.message} onChange={(event) => onChange({ ...value, message: event.target.value })} />
  </div>
}

function parseQuestionImport(text: string): FormComponent[] {
  return text.split(/\n+/).map((line) => line.trim()).filter(Boolean).map((line, index) => {
    const [id, label, type = "text", required = "false"] = line.split(/,|，/).map((item) => item.trim())
    return { id: id || `q_${index + 1}`, label: label || id || `题目 ${index + 1}`, type: type as ComponentKind, required: required === "true" || required === "必填", category: "公共组件" as const, binding: { kind: "none" as BindingKind } }
  })
}

function safeJSON(value: string, fallback: unknown) {
  try { return JSON.parse(value) } catch { return fallback }
}

function conditionFromVisibility(rule?: LogicRule): SimpleCondition {
  const when = ruleRecord(rule?.when) || ruleRecord(rule)
  return conditionFromRecord(when)
}

function conditionFromRecord(record?: Record<string, unknown>): SimpleCondition {
  if (!record) return emptyCondition()
  const questionId = stringValue(record.questionId || record.field || record.source)
  const operator = conditionOperators.some((item) => item.id === record.operator) ? record.operator as ConditionOperator : legacyOperator(record)
  const value = stringValue(record.value ?? record.equals ?? record.notEquals ?? record.contains ?? "")
  return { questionId, operator, value }
}

function visibilityToRule(condition: SimpleCondition): LogicRule {
  if (!condition.questionId) return {}
  return { when: condition }
}

function jumpFromRule(rule?: LogicRule): JumpRuleVisual {
  const when = ruleRecord(rule?.when) || ruleRecord(rule?.condition)
  return { when: conditionFromRecord(when), goto: stringValue(rule?.goto || rule?.target || "") }
}

function jumpToRule(value: JumpRuleVisual): LogicRule {
  const rule: LogicRule = {}
  if (value.when.questionId) rule.when = value.when
  if (value.goto) rule.goto = value.goto
  return rule
}

function validationFromRule(rule?: LogicRule): ValidationVisual {
  return {
    min: stringValue(rule?.min ?? rule?.minLength ?? ""),
    max: stringValue(rule?.max ?? rule?.maxLength ?? ""),
    regex: stringValue(rule?.regex ?? rule?.pattern ?? ""),
    message: stringValue(rule?.message ?? ""),
  }
}

function validationToRule(value: ValidationVisual): LogicRule {
  const rule: LogicRule = {}
  if (value.min !== "") rule.min = numericOrString(value.min)
  if (value.max !== "") rule.max = numericOrString(value.max)
  if (value.regex.trim()) rule.regex = value.regex.trim()
  if (value.message.trim()) rule.message = value.message.trim()
  return rule
}

function emptyCondition(): SimpleCondition {
  return { questionId: "", operator: "equals", value: "" }
}

function ruleRecord(value: unknown): Record<string, unknown> | undefined {
  return typeof value === "object" && value !== null && !Array.isArray(value) ? value as Record<string, unknown> : undefined
}

function stringValue(value: unknown): string {
  return value === undefined || value === null ? "" : String(value)
}

function legacyOperator(record: Record<string, unknown>): ConditionOperator {
  if ("notEquals" in record) return "not_equals"
  if ("contains" in record) return "contains"
  if ("empty" in record) return "empty"
  if ("notEmpty" in record) return "not_empty"
  if ("greaterThan" in record) return "greater_than"
  if ("lessThan" in record) return "less_than"
  return "equals"
}

function numericOrString(value: string) {
  const next = Number(value)
  return Number.isFinite(next) && value.trim() !== "" ? next : value
}
