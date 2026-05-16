SET @report_schema := DATABASE();
SET @sql := IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = @report_schema AND table_name = 'reports' AND column_name = 'code') = 0, 'ALTER TABLE reports ADD COLUMN code VARCHAR(120) NULL AFTER id', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = @report_schema AND table_name = 'reports' AND column_name = 'category') = 0, 'ALTER TABLE reports ADD COLUMN category VARCHAR(80) NULL AFTER report_type', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = @report_schema AND table_name = 'reports' AND column_name = 'subject_type') = 0, 'ALTER TABLE reports ADD COLUMN subject_type VARCHAR(80) NULL AFTER category', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = @report_schema AND table_name = 'reports' AND column_name = 'default_dimension') = 0, 'ALTER TABLE reports ADD COLUMN default_dimension VARCHAR(80) NULL AFTER subject_type', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.columns WHERE table_schema = @report_schema AND table_name = 'reports' AND column_name = 'default_filters_json') = 0, 'ALTER TABLE reports ADD COLUMN default_filters_json JSON NULL AFTER default_dimension', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
SET @sql := IF((SELECT COUNT(*) FROM information_schema.statistics WHERE table_schema = @report_schema AND table_name = 'reports' AND index_name = 'uk_reports_code') = 0, 'ALTER TABLE reports ADD UNIQUE KEY uk_reports_code (code)', 'SELECT 1');
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

CREATE TABLE IF NOT EXISTS report_query_logs (
  id CHAR(36) PRIMARY KEY,
  report_id CHAR(36) NOT NULL,
  user_id CHAR(36) NULL,
  project_id CHAR(36) NULL,
  filters_json JSON NULL,
  result_count INT NOT NULL DEFAULT 0,
  duration_ms INT NOT NULL DEFAULT 0,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_report_query_logs_report (report_id),
  INDEX idx_report_query_logs_project (project_id)
);

CREATE TABLE IF NOT EXISTS report_export_jobs (
  id CHAR(36) PRIMARY KEY,
  report_id CHAR(36) NOT NULL,
  project_id CHAR(36) NULL,
  export_type ENUM('excel','image','pdf','word') NOT NULL DEFAULT 'excel',
  filters_json JSON NULL,
  status ENUM('pending','running','success','failed') NOT NULL DEFAULT 'pending',
  file_path VARCHAR(500) NULL,
  error_message TEXT NULL,
  created_by CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  finished_at TIMESTAMP NULL,
  INDEX idx_report_export_jobs_report (report_id),
  INDEX idx_report_export_jobs_status (status)
);

CREATE TABLE IF NOT EXISTS report_export_files (
  job_id CHAR(36) PRIMARY KEY,
  file_name VARCHAR(240) NOT NULL,
  mime_type VARCHAR(120) NOT NULL,
  content LONGBLOB NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_report_export_files_job FOREIGN KEY (job_id) REFERENCES report_export_jobs(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS praise_records (
  id CHAR(36) PRIMARY KEY,
  project_id CHAR(36) NULL,
  praise_date DATE NOT NULL,
  praise_type VARCHAR(80) NULL,
  praise_method VARCHAR(80) NULL,
  department_id VARCHAR(120) NULL,
  department_name VARCHAR(180) NULL,
  staff_id VARCHAR(120) NULL,
  staff_name VARCHAR(120) NULL,
  patient_id CHAR(36) NULL,
  patient_name VARCHAR(120) NULL,
  quantity INT NOT NULL DEFAULT 1,
  reward_amount DECIMAL(12,2) NOT NULL DEFAULT 0,
  content TEXT NULL,
  remark TEXT NULL,
  status ENUM('draft','confirmed','archived','deleted') NOT NULL DEFAULT 'confirmed',
  created_by CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_praise_records_project (project_id),
  INDEX idx_praise_records_date (praise_date),
  INDEX idx_praise_records_department (department_name),
  INDEX idx_praise_records_staff (staff_name),
  INDEX idx_praise_records_status (status)
);

INSERT INTO reports (id, code, report_type, category, subject_type, default_dimension, default_filters_json, name, description)
VALUES
('RPT_DEPT_SAT', 'department_satisfaction', 'satisfaction', '满意度专题', 'patient', 'department', '{}', '科室满意度统计', '按科室统计评价人数、有效样本、平均满意度和排名'),
('RPT_DEPT_QUESTION', 'department_question_satisfaction', 'satisfaction', '满意度专题', 'patient', 'department_question', '{}', '科室问题满意度分析', '按科室和题目交叉统计各档人数、评价人数和满意度'),
('RPT_QUESTION_OPTIONS', 'question_option_distribution', 'satisfaction', '满意度专题', 'patient', 'question', '{}', '题目满意度分析', '按题目统计各选项人数、总人数和满意度'),
('RPT_LOW_REASON', 'low_score_reason', 'satisfaction', '满意度专题', 'patient', 'reason', '{}', '不满意原因统计', '按低分、多选原因和开放反馈统计问题原因 TopN'),
('RPT_COMMENTS', 'comments_suggestions', 'satisfaction', '满意度专题', 'patient', 'comment', '{}', '意见与建议统计', '开放题意见建议、关联科室、患者和处理状态列表'),
('RPT_TREND', 'satisfaction_trend', 'satisfaction', '满意度专题', 'patient', 'month', '{}', '周期满意度分析', '按月统计满意度趋势、样本量和有效率'),
('RPT_STAFF', 'staff_department_satisfaction', 'satisfaction', '员工与协作科室', 'staff', 'department', '{}', '院内员工/协作科室测评', '支持员工、协作科室和职能科室满意度统计'),
('RPT_PRAISE', 'praise_statistics', 'complaint', '评价投诉', 'praise', 'department', '{}', '好人好事表扬统计', '按科室、人员、表扬方式统计表扬数量和奖励金额')
ON DUPLICATE KEY UPDATE
  code = VALUES(code),
  report_type = VALUES(report_type),
  category = VALUES(category),
  subject_type = VALUES(subject_type),
  default_dimension = VALUES(default_dimension),
  default_filters_json = VALUES(default_filters_json),
  name = VALUES(name),
  description = VALUES(description);

INSERT INTO form_library_items (id, kind, label, hint, scenario, components_json, sort_order, enabled)
VALUES
('traditional-inpatient-satisfaction', 'template', '住院患者满意度调查', '对标传统行风系统住院患者满意度，预置住院环节、科室环境、医生护士、出院指导等题目', '调查',
'[
  {"id":"patient_section","type":"section","label":"患者基础信息","required":false,"category":"公共组件"},
  {"id":"patient_name","type":"text","label":"患者姓名","required":false,"category":"公共组件"},
  {"id":"patient_phone","type":"text","label":"联系电话","required":false,"category":"公共组件"},
  {"id":"visit_section","type":"section","label":"住院信息","required":false,"category":"公共组件"},
  {"id":"department","type":"remote_options","label":"住院科室","required":true,"category":"公共组件","binding":{"kind":"mysql","dataSourceId":"survey-dict","operation":"select label, value from department_dict","labelPath":"$.label","valuePath":"$.value"}},
  {"id":"discharge_date","type":"date","label":"出院日期","required":false,"category":"公共组件"},
  {"id":"satisfaction_section","type":"section","label":"住院满意度","required":false,"category":"公共组件"},
  {"id":"inpatient_matrix","type":"matrix","label":"住院服务评价","required":true,"category":"公共组件","rows":["每次用药时，医务人员是否告知药品名称","护士是否用您听得懂的方式解释问题","医生是否尊重您","院内路标和指示是否明确","夜间病房附近是否安静","药房服务是否满意","出院时是否清楚健康注意事项"],"columns":["很不满意","不满意","一般","满意","非常满意"]},
  {"id":"overall_satisfaction","type":"likert","label":"总体满意度","required":true,"category":"公共组件","options":[{"label":"很不满意","value":"1"},{"label":"不满意","value":"2"},{"label":"一般","value":"3"},{"label":"满意","value":"4"},{"label":"非常满意","value":"5"}]},
  {"id":"problem_reasons","type":"multi_select","label":"不满意原因","required":false,"category":"公共组件","options":[{"label":"等待时间长","value":"wait_time"},{"label":"解释沟通不足","value":"communication"},{"label":"环境设施","value":"environment"},{"label":"费用问题","value":"billing"},{"label":"服务态度","value":"attitude"}]},
  {"id":"feedback","type":"textarea","label":"意见与建议","required":false,"category":"公共组件"}
]', 193, TRUE),
('traditional-function-dept-satisfaction', 'template', '职能科室满意度测评', '用于院内员工对职能科室、协作科室进行服务态度、流程、效率和反馈评价', '调查',
'[
  {"id":"staff_section","type":"section","label":"测评对象","required":false,"category":"公共组件"},
  {"id":"source_department","type":"text","label":"评价人所在科室","required":false,"category":"公共组件"},
  {"id":"target_department","type":"remote_options","label":"被评价科室","required":true,"category":"公共组件","binding":{"kind":"mysql","dataSourceId":"survey-dict","operation":"select label, value from department_dict","labelPath":"$.label","valuePath":"$.value"}},
  {"id":"function_dept_matrix","type":"matrix","label":"职能科室问题满意度","required":true,"category":"公共组件","rows":["工作态度和服务意识","工作流程顺畅程度","业务水平和能力","工作效率","问题反馈是否重视并给予反馈","工作纪律和精神风貌"],"columns":["不满意","一般","基本满意","满意","很满意"]},
  {"id":"overall_satisfaction","type":"likert","label":"总体满意度","required":true,"category":"公共组件","options":[{"label":"不满意","value":"1"},{"label":"一般","value":"2"},{"label":"基本满意","value":"3"},{"label":"满意","value":"4"},{"label":"很满意","value":"5"}]},
  {"id":"feedback","type":"textarea","label":"意见与建议","required":false,"category":"公共组件"}
]', 194, TRUE),
('traditional-praise-registration', 'template', '好人好事表扬登记', '用于登记表扬日期、表扬方式、人员科室、患者姓名、奖励金额和备注', '调查',
'[
  {"id":"praise_section","type":"section","label":"表扬登记","required":false,"category":"公共组件"},
  {"id":"praise_date","type":"date","label":"表扬日期","required":true,"category":"公共组件"},
  {"id":"department_name","type":"remote_options","label":"科室名称","required":true,"category":"公共组件","binding":{"kind":"mysql","dataSourceId":"survey-dict","operation":"select label, value from department_dict","labelPath":"$.label","valuePath":"$.value"}},
  {"id":"staff_name","type":"text","label":"医护人员姓名","required":true,"category":"公共组件"},
  {"id":"patient_name","type":"text","label":"患者姓名","required":false,"category":"公共组件"},
  {"id":"praise_method","type":"single_select","label":"表扬方式","required":true,"category":"公共组件","options":[{"label":"电话表扬","value":"phone"},{"label":"锦旗","value":"banner"},{"label":"感谢信","value":"letter"},{"label":"微信","value":"wechat"},{"label":"现场","value":"onsite"}]},
  {"id":"quantity","type":"number","label":"数量","required":false,"category":"公共组件"},
  {"id":"reward_amount","type":"number","label":"退红包金额/奖励金额","required":false,"category":"公共组件"},
  {"id":"remark","type":"textarea","label":"备注","required":false,"category":"公共组件"}
]', 195, TRUE)
ON DUPLICATE KEY UPDATE label = VALUES(label), hint = VALUES(hint), scenario = VALUES(scenario), components_json = VALUES(components_json), sort_order = VALUES(sort_order), enabled = VALUES(enabled);
