package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"reporter/internal/domain"
)

func (s *Store) FormLibrary() []domain.FormLibraryItem {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]domain.FormLibraryItem, len(s.formLibrary))
	copy(items, s.formLibrary)
	return items
}

func (s *Store) LoadFormLibraryFromSQL(ctx context.Context, driver, dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return nil
	}
	if strings.TrimSpace(driver) == "" {
		driver = "mysql"
	}
	db, err := sql.Open(driver, dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := ensureFormLibraryTable(ctx, db); err != nil {
		return err
	}
	if err := seedFormLibrary(ctx, db); err != nil {
		return err
	}
	items, err := queryFormLibrary(ctx, db)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.formLibrary = items
	return nil
}

func ensureFormLibraryTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS form_library_items (
  id VARCHAR(120) PRIMARY KEY,
  kind ENUM('template','common','atom') NOT NULL,
  label VARCHAR(180) NOT NULL,
  hint TEXT NULL,
  scenario VARCHAR(40) NULL,
  components_json JSON NOT NULL,
  sort_order INT NOT NULL DEFAULT 0,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
)`)
	return err
}

func seedFormLibrary(ctx context.Context, db *sql.DB) error {
	for _, item := range DefaultFormLibrary() {
		components, err := json.Marshal(item.Components)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `
INSERT IGNORE INTO form_library_items (id, kind, label, hint, scenario, components_json, sort_order)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			item.ID, item.Kind, item.Label, item.Hint, item.Scenario, string(components), item.SortOrder,
		); err != nil {
			return err
		}
	}
	return nil
}

func queryFormLibrary(ctx context.Context, db *sql.DB) ([]domain.FormLibraryItem, error) {
	rows, err := db.QueryContext(ctx, `
SELECT id, kind, label, COALESCE(hint, ''), COALESCE(scenario, ''), components_json, sort_order
FROM form_library_items
WHERE enabled = TRUE
ORDER BY sort_order, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.FormLibraryItem{}
	for rows.Next() {
		var item domain.FormLibraryItem
		var raw string
		if err := rows.Scan(&item.ID, &item.Kind, &item.Label, &item.Hint, &item.Scenario, &raw, &item.SortOrder); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &item.Components); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func DefaultFormLibrary() []domain.FormLibraryItem {
	satisfaction := []map[string]interface{}{{"label": "很不满意", "value": "1"}, {"label": "不满意", "value": "2"}, {"label": "一般", "value": "3"}, {"label": "满意", "value": "4"}, {"label": "非常满意", "value": "5"}}
	patient := []map[string]interface{}{
		{"id": "patient_section", "type": "section", "label": "患者基础信息", "required": false, "category": "公共组件"},
		{"id": "patient_name", "type": "text", "label": "患者姓名", "required": true, "category": "公共组件", "binding": map[string]interface{}{"kind": "http", "dataSourceId": "patients-api", "operation": "GET /patients/:patientId", "valuePath": "$.name"}},
		{"id": "patient_gender", "type": "single_select", "label": "性别", "required": false, "category": "公共组件", "options": []map[string]interface{}{{"label": "男", "value": "male"}, {"label": "女", "value": "female"}, {"label": "其他", "value": "other"}}, "binding": map[string]interface{}{"kind": "hl7", "dataSourceId": "hl7-adt", "operation": "PID-8", "valuePath": "PID.8"}},
		{"id": "patient_age", "type": "number", "label": "年龄", "required": false, "category": "公共组件", "binding": map[string]interface{}{"kind": "http", "dataSourceId": "patients-api", "operation": "GET /patients/:patientId", "valuePath": "$.age"}},
		{"id": "patient_phone", "type": "text", "label": "联系电话", "required": false, "category": "公共组件", "binding": map[string]interface{}{"kind": "http", "dataSourceId": "patients-api", "operation": "GET /patients/:patientId", "valuePath": "$.phone"}},
	}
	visit := []map[string]interface{}{
		{"id": "visit_section", "type": "section", "label": "就诊信息", "required": false, "category": "公共组件"},
		{"id": "visit_date", "type": "date", "label": "就诊日期", "required": true, "category": "公共组件", "binding": map[string]interface{}{"kind": "hl7", "dataSourceId": "hl7-adt", "operation": "PV1-44", "valuePath": "PV1.44"}},
		{"id": "department", "type": "remote_options", "label": "就诊科室", "required": true, "category": "公共组件", "binding": map[string]interface{}{"kind": "grpc", "dataSourceId": "dept-grpc", "operation": "DepartmentService/ListDepartments", "labelPath": "$.name", "valuePath": "$.code"}},
		{"id": "diagnosis", "type": "remote_options", "label": "诊断", "required": false, "category": "公共组件", "binding": map[string]interface{}{"kind": "mysql", "dataSourceId": "survey-dict", "operation": "select label, value from diagnosis_dict where keyword like :keyword", "labelPath": "$.label", "valuePath": "$.value"}},
	}
	follow := []map[string]interface{}{
		{"id": "follow_section", "type": "section", "label": "随访记录", "required": false, "category": "公共组件"},
		{"id": "follow_date", "type": "date", "label": "随访日期", "required": true, "category": "公共组件"},
		{"id": "follow_method", "type": "single_select", "label": "随访方式", "required": true, "category": "公共组件", "options": []map[string]interface{}{{"label": "电话", "value": "phone"}, {"label": "门诊", "value": "clinic"}, {"label": "线上", "value": "online"}, {"label": "上门", "value": "home"}}},
		{"id": "symptoms", "type": "multi_select", "label": "当前症状", "required": false, "category": "公共组件", "binding": map[string]interface{}{"kind": "mysql", "dataSourceId": "survey-dict", "operation": "select label, value from symptom_dict where disease_code = :diseaseCode", "labelPath": "$.label", "valuePath": "$.value"}},
		{"id": "medication_adherence", "type": "likert", "label": "用药依从性", "required": false, "category": "公共组件", "options": satisfaction},
	}
	satisfactionComponents := []map[string]interface{}{
		{"id": "satisfaction_section", "type": "section", "label": "满意度评价", "required": false, "category": "公共组件"},
		{"id": "overall_satisfaction", "type": "likert", "label": "总体满意度", "required": true, "category": "公共组件", "options": satisfaction, "binding": map[string]interface{}{"kind": "mysql", "dataSourceId": "survey-dict", "operation": "select label, value from survey_options where group_code = 'satisfaction'", "labelPath": "$.label", "valuePath": "$.value"}},
		{"id": "service_matrix", "type": "matrix", "label": "分项满意度", "required": true, "category": "公共组件", "rows": []string{"挂号缴费流程", "候诊时间", "医生沟通", "护士服务", "检查检验指引", "院内环境"}, "columns": []string{"很不满意", "不满意", "一般", "满意", "非常满意"}},
		{"id": "recommend_score", "type": "rating", "label": "推荐意愿", "required": true, "category": "公共组件", "scale": 10},
		{"id": "feedback", "type": "textarea", "label": "意见与建议", "required": false, "category": "公共组件"},
	}
	postOp := []map[string]interface{}{
		{"id": "post_op_section", "type": "section", "label": "术后跟踪", "required": false, "category": "公共组件"},
		{"id": "surgery_date", "type": "date", "label": "手术日期", "required": true, "category": "公共组件", "binding": map[string]interface{}{"kind": "hl7", "dataSourceId": "hl7-adt", "operation": "PR1-5", "valuePath": "PR1.5"}},
		{"id": "procedure_name", "type": "text", "label": "手术名称", "required": true, "category": "公共组件", "binding": map[string]interface{}{"kind": "hl7", "dataSourceId": "hl7-adt", "operation": "PR1-3", "valuePath": "PR1.3"}},
		{"id": "pain_score", "type": "rating", "label": "疼痛评分", "required": true, "category": "公共组件", "scale": 10},
		{"id": "image_followup", "type": "remote_options", "label": "相关影像检查", "required": false, "category": "公共组件", "binding": map[string]interface{}{"kind": "dicom", "dataSourceId": "dicom-pacs", "operation": "QIDO-RS /studies?PatientID=:patientId", "labelPath": "$.StudyDescription", "valuePath": "$.StudyInstanceUID"}},
	}
	return []domain.FormLibraryItem{
		{ID: "atom-text", Kind: "atom", Label: "单行文本", Hint: "姓名、编号、短文本", Components: []map[string]interface{}{{"id": "text", "type": "text", "label": "单行文本", "required": false, "category": "原子组件"}}, SortOrder: 10},
		{ID: "atom-number", Kind: "atom", Label: "数字", Hint: "年龄、评分、次数", Components: []map[string]interface{}{{"id": "number", "type": "number", "label": "数字", "required": false, "category": "原子组件"}}, SortOrder: 11},
		{ID: "atom-date", Kind: "atom", Label: "日期", Hint: "就诊、随访、手术日期", Components: []map[string]interface{}{{"id": "date", "type": "date", "label": "日期", "required": false, "category": "原子组件"}}, SortOrder: 12},
		{ID: "atom-rating", Kind: "atom", Label: "评分", Hint: "星级、NPS、疼痛评分", Components: []map[string]interface{}{{"id": "rating", "type": "rating", "label": "评分", "required": false, "category": "原子组件", "scale": 5}}, SortOrder: 13},
		{ID: "patient-basic", Kind: "common", Label: "患者基础信息", Hint: "姓名、性别、年龄、手机号，可从主索引/API/HL7 ADT 回填", Components: patient, SortOrder: 100},
		{ID: "visit-info", Kind: "common", Label: "就诊信息", Hint: "科室、医生、就诊日期、诊断，支持 HIS/API/gRPC/HL7", Components: visit, SortOrder: 101},
		{ID: "follow-up", Kind: "common", Label: "随访", Hint: "随访方式、时间、症状、用药依从性", Components: follow, SortOrder: 102},
		{ID: "post-op", Kind: "common", Label: "术后跟踪", Hint: "手术信息、疼痛评分、影像检查", Components: postOp, SortOrder: 103},
		{ID: "satisfaction", Kind: "common", Label: "满意度", Hint: "总体满意、分项矩阵、推荐意愿、原因和建议", Components: satisfactionComponents, SortOrder: 104},
		{ID: "outpatient-satisfaction", Kind: "template", Label: "患者就诊满意度调查", Hint: "由患者基础信息、就诊信息、满意度公共组件组合而成", Scenario: "调查", Components: append(append([]map[string]interface{}{}, patient...), append(visit, satisfactionComponents...)...), SortOrder: 200},
		{ID: "discharge-follow-up", Kind: "template", Label: "出院后随访问卷", Hint: "出院患者基础信息、随访方式、症状、用药依从性和复诊提醒", Scenario: "随访", Components: append(append([]map[string]interface{}{}, patient...), follow...), SortOrder: 201},
		{ID: "post-op-follow-up", Kind: "template", Label: "术后随访问卷", Hint: "由患者基础信息、术后跟踪、随访公共组件组合而成", Scenario: "术后", Components: append(append([]map[string]interface{}{}, patient...), append(postOp, follow...)...), SortOrder: 202},
		{ID: "hypertension-follow-up", Kind: "template", Label: "高血压慢病随访", Hint: "血压、用药、症状、生活方式和复诊计划", Scenario: "慢病", Components: append(append([]map[string]interface{}{}, patient...), append(follow, []map[string]interface{}{
			{"id": "bp_section", "type": "section", "label": "血压与生活方式", "required": false, "category": "公共组件"},
			{"id": "systolic_bp", "type": "number", "label": "收缩压 mmHg", "required": true, "category": "公共组件"},
			{"id": "diastolic_bp", "type": "number", "label": "舒张压 mmHg", "required": true, "category": "公共组件"},
			{"id": "bp_control", "type": "likert", "label": "血压控制情况", "required": false, "category": "公共组件", "options": []map[string]interface{}{{"label": "很差", "value": "1"}, {"label": "偏差", "value": "2"}, {"label": "一般", "value": "3"}, {"label": "较好", "value": "4"}, {"label": "很好", "value": "5"}}},
			{"id": "lifestyle", "type": "multi_select", "label": "生活方式干预", "required": false, "category": "公共组件", "options": []map[string]interface{}{{"label": "限盐", "value": "salt"}, {"label": "规律运动", "value": "exercise"}, {"label": "控制体重", "value": "weight"}, {"label": "戒烟限酒", "value": "smoke_alcohol"}}},
			{"id": "adverse_reaction", "type": "textarea", "label": "药物不良反应", "required": false, "category": "公共组件"},
		}...)...), SortOrder: 203},
		{ID: "diabetes-management", Kind: "template", Label: "糖尿病管理随访", Hint: "血糖、低血糖事件、饮食运动、足部和用药依从性", Scenario: "慢病", Components: append(append([]map[string]interface{}{}, patient...), append(follow, []map[string]interface{}{
			{"id": "glucose_section", "type": "section", "label": "血糖管理", "required": false, "category": "公共组件"},
			{"id": "fasting_glucose", "type": "number", "label": "空腹血糖 mmol/L", "required": true, "category": "公共组件"},
			{"id": "postprandial_glucose", "type": "number", "label": "餐后 2 小时血糖 mmol/L", "required": false, "category": "公共组件"},
			{"id": "hypoglycemia", "type": "single_select", "label": "近期低血糖事件", "required": true, "category": "公共组件", "options": []map[string]interface{}{{"label": "无", "value": "none"}, {"label": "1 次", "value": "once"}, {"label": "2 次及以上", "value": "multiple"}}},
			{"id": "diet_exercise", "type": "matrix", "label": "饮食与运动执行情况", "required": false, "category": "公共组件", "rows": []string{"控制主食", "规律运动", "监测血糖", "足部护理"}, "columns": []string{"未执行", "偶尔", "基本做到", "完全做到"}},
			{"id": "foot_problem", "type": "textarea", "label": "足部异常或其他问题", "required": false, "category": "公共组件"},
		}...)...), SortOrder: 204},
		{ID: "physical-exam-review", Kind: "template", Label: "体检异常复查登记", Hint: "体检异常项、影像/检验关联、复查建议和结果跟踪", Scenario: "体检", Components: append(append([]map[string]interface{}{}, patient...), []map[string]interface{}{
			{"id": "exam_section", "type": "section", "label": "体检异常信息", "required": false, "category": "公共组件"},
			{"id": "exam_date", "type": "date", "label": "体检日期", "required": true, "category": "公共组件"},
			{"id": "abnormal_items", "type": "multi_select", "label": "异常项目", "required": true, "category": "公共组件", "binding": map[string]interface{}{"kind": "http", "dataSourceId": "patients-api", "operation": "GET /exam/:examId/abnormal-items", "labelPath": "$.name", "valuePath": "$.code"}},
			{"id": "related_image", "type": "remote_options", "label": "相关影像", "required": false, "category": "公共组件", "binding": map[string]interface{}{"kind": "dicom", "dataSourceId": "dicom-pacs", "operation": "QIDO-RS /studies?PatientID=:patientId", "labelPath": "$.StudyDescription", "valuePath": "$.StudyInstanceUID"}},
			{"id": "review_advice", "type": "textarea", "label": "复查建议", "required": true, "category": "公共组件"},
			{"id": "review_date", "type": "date", "label": "计划复查日期", "required": false, "category": "公共组件"},
			{"id": "review_result", "type": "textarea", "label": "复查结果", "required": false, "category": "公共组件"},
		}...), SortOrder: 205},
	}
}
