INSERT INTO form_library_items (id, kind, label, hint, scenario, components_json, sort_order, enabled)
VALUES
(
  'surveyjs-outpatient-satisfaction',
  'template',
  'SurveyJS 门诊满意度模板',
  '面向公开链接、微信和短信渠道的标准调查结构，支持矩阵、NPS、条件题和附件扩展',
  '调查',
  CAST('[
    {"id":"patient_section","type":"section","label":"患者基础信息","required":false,"category":"公共组件"},
    {"id":"patient_name","type":"text","label":"患者姓名","required":true,"category":"公共组件"},
    {"id":"patient_phone","type":"text","label":"联系电话","required":false,"category":"公共组件"},
    {"id":"visit_section","type":"section","label":"就诊信息","required":false,"category":"公共组件"},
    {"id":"visit_date","type":"date","label":"就诊日期","required":true,"category":"公共组件"},
    {"id":"department","type":"remote_options","label":"就诊科室","required":true,"category":"公共组件","binding":{"kind":"mysql","dataSourceId":"survey-dict","operation":"select label, value from department_dict","labelPath":"$.label","valuePath":"$.value"}},
    {"id":"satisfaction_section","type":"section","label":"满意度评价","required":false,"category":"公共组件"},
    {"id":"overall_satisfaction","type":"likert","label":"总体满意度","required":true,"category":"公共组件","options":[{"label":"很不满意","value":"1"},{"label":"不满意","value":"2"},{"label":"一般","value":"3"},{"label":"满意","value":"4"},{"label":"非常满意","value":"5"}]},
    {"id":"service_matrix","type":"matrix","label":"分项满意度","required":true,"category":"公共组件","rows":["挂号缴费流程","候诊时间","医生沟通","护士服务","检查检验指引","院内环境"],"columns":["很不满意","不满意","一般","满意","非常满意"]},
    {"id":"recommend_score","type":"rating","label":"推荐意愿","required":true,"category":"公共组件","scale":10},
    {"id":"problem_reasons","type":"multi_select","label":"不满意原因","required":false,"category":"公共组件","options":[{"label":"等待时间","value":"wait_time"},{"label":"沟通解释","value":"communication"},{"label":"流程指引","value":"guidance"},{"label":"费用体验","value":"billing"},{"label":"环境设施","value":"environment"}],"visibilityRules":{"when":{"questionId":"overall_satisfaction","operator":"less_than","value":"4"}}},
    {"id":"feedback","type":"textarea","label":"意见与建议","required":false,"category":"公共组件"},
    {"id":"surveyjs_attachment","type":"attachment","label":"补充材料","required":false,"category":"公共组件","config":{"accept":"image/*,audio/*,application/pdf","maxSizeMb":50,"multiple":true}}
  ]' AS JSON),
  190,
  TRUE
),
(
  'surveyjs-nps',
  'template',
  'SurveyJS NPS 推荐度调查',
  '推荐意愿、原因追问、开放建议，适合快速满意度或体验净推荐值采集',
  '调查',
  CAST('[
    {"id":"nps_section","type":"section","label":"推荐意愿","required":false,"category":"公共组件"},
    {"id":"recommend_score","type":"rating","label":"您愿意向亲友推荐本院服务吗？","required":true,"category":"公共组件","scale":10,"helpText":"0 表示完全不推荐，10 表示非常愿意推荐。"},
    {"id":"low_score_reason","type":"multi_select","label":"影响您推荐的主要原因","required":false,"category":"公共组件","options":[{"label":"等待时间","value":"wait_time"},{"label":"沟通解释","value":"communication"},{"label":"流程指引","value":"guidance"},{"label":"费用体验","value":"billing"},{"label":"环境设施","value":"environment"}],"visibilityRules":{"when":{"questionId":"recommend_score","operator":"less_than","value":"7"}}},
    {"id":"nps_feedback","type":"textarea","label":"还有哪些改进建议？","required":false,"category":"公共组件"}
  ]' AS JSON),
  191,
  TRUE
),
(
  'surveyjs-registration-table',
  'template',
  'SurveyJS 多维登记表',
  '包含动态明细表、计算字段和附件，适合预约、登记、会务和宣传报名',
  '调查',
  CAST('[
    {"id":"register_section","type":"section","label":"登记信息","required":false,"category":"公共组件"},
    {"id":"contact_name","type":"text","label":"联系人","required":true,"category":"公共组件"},
    {"id":"contact_phone","type":"text","label":"联系电话","required":true,"category":"公共组件","validationRules":{"regex":"^1\\\\d{10}$","message":"请输入 11 位手机号"}},
    {"id":"items_table","type":"table","label":"报名/预约明细","required":false,"category":"公共组件","rows":["记录 1"],"columns":["项目","人数","备注"],"config":{"addRows":true,"addColumns":false}},
    {"id":"estimated_total","type":"computed","label":"预计人数","required":false,"category":"公共组件","config":{"expression":"sum(items_table.人数)","precision":0,"readonly":true}}
  ]' AS JSON),
  192,
  TRUE
)
ON DUPLICATE KEY UPDATE
  kind = VALUES(kind),
  label = VALUES(label),
  hint = VALUES(hint),
  scenario = VALUES(scenario),
  components_json = VALUES(components_json),
  sort_order = VALUES(sort_order),
  enabled = VALUES(enabled);
