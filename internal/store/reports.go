package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) EnsureReportTables(ctx context.Context) error {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS reports (
  id CHAR(36) PRIMARY KEY,
  code VARCHAR(120) NULL,
  report_type VARCHAR(60) NOT NULL DEFAULT 'custom',
  category VARCHAR(80) NULL,
  subject_type VARCHAR(80) NULL,
  default_dimension VARCHAR(80) NULL,
  default_filters_json JSON NULL,
  name VARCHAR(180) NOT NULL,
  description TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_reports_code (code)
)`,
		`CREATE TABLE IF NOT EXISTS report_widgets (
  id CHAR(36) PRIMARY KEY,
  report_id CHAR(36) NOT NULL,
  widget_type VARCHAR(60) NOT NULL,
  title VARCHAR(180) NOT NULL,
  query_json JSON NULL,
  vis_spec_json JSON NULL,
  data_source_id CHAR(36) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_report_widgets_report (report_id)
)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	if err := ensureColumn(ctx, db, "reports", "report_type", "VARCHAR(60) NOT NULL DEFAULT 'custom' AFTER id"); err != nil {
		return err
	}
	reportColumns := []struct{ name, ddl string }{
		{"code", "VARCHAR(120) NULL AFTER id"},
		{"category", "VARCHAR(80) NULL AFTER report_type"},
		{"subject_type", "VARCHAR(80) NULL AFTER category"},
		{"default_dimension", "VARCHAR(80) NULL AFTER subject_type"},
		{"default_filters_json", "JSON NULL AFTER default_dimension"},
	}
	for _, column := range reportColumns {
		if err := ensureColumn(ctx, db, "reports", column.name, column.ddl); err != nil {
			return err
		}
	}
	extraStatements := []string{
		`CREATE TABLE IF NOT EXISTS report_query_logs (
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
)`,
		`CREATE TABLE IF NOT EXISTS report_export_jobs (
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
)`,
		`CREATE TABLE IF NOT EXISTS report_export_files (
  job_id CHAR(36) PRIMARY KEY,
  file_name VARCHAR(240) NOT NULL,
  mime_type VARCHAR(120) NOT NULL,
  content LONGBLOB NOT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT fk_report_export_files_job FOREIGN KEY (job_id) REFERENCES report_export_jobs(id) ON DELETE CASCADE
)`,
		`CREATE TABLE IF NOT EXISTS praise_records (
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
)`,
	}
	for _, statement := range extraStatements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return seedReports(ctx, db)
}

func seedReports(ctx context.Context, db *sql.DB) error {
	reports := []domain.Report{
		{ID: "RP001", Code: "followup_monthly", Type: "followup", Category: "随访分析", SubjectType: "followup", DefaultDimension: "month", Name: "随访完成情况月报", Description: "从随访记录聚合随访提交量、完成量和完成率"},
		{ID: "RP002", Code: "satisfaction_overview", Type: "satisfaction", Category: "满意度专题", SubjectType: "patient", DefaultDimension: "department", Name: "满意度分析报告", Description: "从满意度答卷、访谈表单和指标体系聚合科室、指标、渠道和低分原因"},
		{ID: "RP003", Code: "complaint_overview", Type: "complaint", Category: "评价投诉", SubjectType: "complaint", DefaultDimension: "department", Name: "评价投诉分析报告", Description: "从评价投诉台账聚合投诉、表扬、处理状态和责任科室"},
		{ID: "RPT_DEPT_SAT", Code: "department_satisfaction", Type: "satisfaction", Category: "满意度专题", SubjectType: "patient", DefaultDimension: "department", Name: "科室满意度统计", Description: "按科室统计评价人数、有效样本、平均满意度和排名"},
		{ID: "RPT_DEPT_QUESTION", Code: "department_question_satisfaction", Type: "satisfaction", Category: "满意度专题", SubjectType: "patient", DefaultDimension: "department_question", Name: "科室问题满意度分析", Description: "按科室和题目交叉统计各档人数、评价人数和满意度"},
		{ID: "RPT_QUESTION_OPTIONS", Code: "question_option_distribution", Type: "satisfaction", Category: "满意度专题", SubjectType: "patient", DefaultDimension: "question", Name: "题目满意度分析", Description: "按题目统计各选项人数、总人数和满意度"},
		{ID: "RPT_LOW_REASON", Code: "low_score_reason", Type: "satisfaction", Category: "满意度专题", SubjectType: "patient", DefaultDimension: "reason", Name: "不满意原因统计", Description: "按低分、多选原因和开放反馈统计问题原因 TopN"},
		{ID: "RPT_COMMENTS", Code: "comments_suggestions", Type: "satisfaction", Category: "满意度专题", SubjectType: "patient", DefaultDimension: "comment", Name: "意见与建议统计", Description: "开放题意见建议、关联科室、患者和处理状态列表"},
		{ID: "RPT_TREND", Code: "satisfaction_trend", Type: "satisfaction", Category: "满意度专题", SubjectType: "patient", DefaultDimension: "month", Name: "周期满意度分析", Description: "按月统计满意度趋势、样本量和有效率"},
		{ID: "RPT_STAFF", Code: "staff_department_satisfaction", Type: "satisfaction", Category: "员工与协作科室", SubjectType: "staff", DefaultDimension: "department", Name: "院内员工/协作科室测评", Description: "支持员工、协作科室和职能科室满意度统计"},
		{ID: "RPT_PRAISE", Code: "praise_statistics", Type: "complaint", Category: "评价投诉", SubjectType: "praise", DefaultDimension: "department", Name: "好人好事表扬统计", Description: "按科室、人员、表扬方式统计表扬数量和奖励金额"},
	}
	for _, report := range reports {
		filters, _ := json.Marshal(report.DefaultFilters)
		if _, err := db.ExecContext(ctx, `
INSERT INTO reports (id, code, report_type, category, subject_type, default_dimension, default_filters_json, name, description)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE code = VALUES(code), report_type = VALUES(report_type), category = VALUES(category), subject_type = VALUES(subject_type), default_dimension = VALUES(default_dimension), default_filters_json = VALUES(default_filters_json), name = VALUES(name), description = VALUES(description)`,
			report.ID, nullableString(report.Code), report.Type, nullableString(report.Category), nullableString(report.SubjectType), nullableString(report.DefaultDimension), string(filters), report.Name, report.Description); err != nil {
			return err
		}
	}
	widgets := []domain.ReportWidget{
		{ID: "RW001", ReportID: "RP001", Type: "bar", Title: "月度随访完成率", DataSource: "followup_records"},
		{ID: "RW002", ReportID: "RP001", Type: "table", Title: "随访月度明细", DataSource: "followup_records"},
		{ID: "RW003", ReportID: "RP002", Type: "bar", Title: "科室满意度", DataSource: "survey_submissions"},
		{ID: "RW004", ReportID: "RP002", Type: "table", Title: "满意度指标明细", DataSource: "satisfaction_indicator_scores"},
		{ID: "RW005", ReportID: "RP003", Type: "bar", Title: "责任科室投诉评价", DataSource: "evaluation_complaints"},
	}
	for _, widget := range widgets {
		raw, _ := json.Marshal(map[string]string{"source": widget.DataSource})
		if _, err := db.ExecContext(ctx, `
INSERT INTO report_widgets (id, report_id, widget_type, title, query_json, vis_spec_json, data_source_id)
VALUES (?, ?, ?, ?, ?, '{}', NULL)
ON DUPLICATE KEY UPDATE title = VALUES(title), query_json = VALUES(query_json)`,
			widget.ID, widget.ReportID, widget.Type, widget.Title, string(raw)); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ReportDefinitions(ctx context.Context) ([]domain.Report, error) {
	return s.dbReports(ctx)
}

func (s *Store) ReportDefinition(ctx context.Context, id string) (domain.Report, error) {
	return s.dbReport(ctx, id)
}

func (s *Store) CreateReportDefinition(ctx context.Context, report domain.Report) (domain.Report, error) {
	return s.createDBReport(ctx, report)
}

func (s *Store) UpdateReportDefinition(ctx context.Context, id string, patch domain.Report) (domain.Report, error) {
	return s.updateDBReport(ctx, id, patch)
}

func (s *Store) AddReportDefinitionWidget(ctx context.Context, reportID string, widget domain.ReportWidget) (domain.ReportWidget, error) {
	return s.addDBReportWidget(ctx, reportID, widget)
}

func (s *Store) QueryReportData(ctx context.Context, reportID string, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	return s.queryDBReport(ctx, reportID, projectID, filters)
}

func (s *Store) ReportSubmissionDrilldown(ctx context.Context, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureSurveyChannelTables(ctx); err != nil {
		return nil, err
	}
	allowedDepartments := reportAllowedDepartmentsCSV(filters)
	rows, err := db.QueryContext(ctx, `
SELECT ss.id,
       COALESCE(sp.name, '') AS project_name,
       COALESCE(p.name, '') AS patient_name,
       COALESCE(p.phone, '') AS patient_phone,
       COALESCE(cv.visit_no, '') AS visit_no,
       COALESCE(cv.department_name, '') AS department,
       COALESCE(cv.attending_doctor, '') AS doctor,
       ss.channel,
       ss.quality_status,
       ss.duration_seconds,
       DATE_FORMAT(ss.submitted_at, '%Y-%m-%d %H:%i:%s') AS submitted_at
FROM survey_submissions ss
LEFT JOIN satisfaction_projects sp ON sp.id = ss.project_id
LEFT JOIN patients p ON p.id = ss.patient_id
LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
WHERE ss.status <> 'deleted'
  AND (? = '' OR ss.project_id = ?)
  AND (? = '' OR DATE(ss.submitted_at) >= ?)
  AND (? = '' OR DATE(ss.submitted_at) <= ?)
  AND (? = '' OR COALESCE(cv.department_name, '') LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR FIND_IN_SET(COALESCE(cv.department_name, ''), ?) > 0)
  AND (? = '' OR COALESCE(cv.attending_doctor, '') LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR ss.channel = ?)
  AND (? = '' OR EXISTS (SELECT 1 FROM survey_submission_answers sa WHERE sa.submission_id = ss.id AND sa.question_id = ?))
ORDER BY ss.submitted_at DESC
LIMIT 300`, projectID, projectID, filters.DateFrom, filters.DateFrom, filters.DateTo, filters.DateTo, filters.Department, filters.Department, allowedDepartments, allowedDepartments, filters.Doctor, filters.Doctor, filters.Channel, filters.Channel, filters.QuestionID, filters.QuestionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var id, projectName, patientName, patientPhone, visitNo, department, doctor, channel, qualityStatus, submittedAt string
		var duration int
		if err := rows.Scan(&id, &projectName, &patientName, &patientPhone, &visitNo, &department, &doctor, &channel, &qualityStatus, &duration, &submittedAt); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"submissionId": id, "projectName": projectName, "patientName": patientName, "patientPhone": patientPhone, "visitNo": visitNo, "department": department, "doctor": doctor, "channel": channel, "qualityStatus": qualityStatus, "durationSeconds": duration, "submittedAt": submittedAt})
	}
	return map[string]interface{}{"dimensions": []string{"submissionId"}, "measures": []string{"durationSeconds"}, "rows": resultRows}, rows.Err()
}

func (s *Store) ReportExportJobs(ctx context.Context, reportID string) ([]domain.ReportExportJob, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, report_id, COALESCE(project_id, ''), export_type, COALESCE(CAST(filters_json AS CHAR), '{}'), status, COALESCE(file_path, ''), COALESCE(error_message, ''), COALESCE(created_by, ''), created_at, finished_at
FROM report_export_jobs
WHERE (? = '' OR report_id = ?)
ORDER BY created_at DESC
LIMIT 200`, reportID, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	jobs := []domain.ReportExportJob{}
	for rows.Next() {
		var job domain.ReportExportJob
		var filtersRaw string
		var finishedAt sql.NullTime
		if err := rows.Scan(&job.ID, &job.ReportID, &job.ProjectID, &job.ExportType, &filtersRaw, &job.Status, &job.FilePath, &job.ErrorMessage, &job.CreatedBy, &job.CreatedAt, &finishedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(filtersRaw), &job.Filters)
		if finishedAt.Valid {
			job.FinishedAt = &finishedAt.Time
		}
		jobs = append(jobs, job)
	}
	return jobs, rows.Err()
}

func (s *Store) CreateReportExportJob(ctx context.Context, job domain.ReportExportJob) (domain.ReportExportJob, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.ReportExportJob{}, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return domain.ReportExportJob{}, err
	}
	report, err := s.dbReport(ctx, job.ReportID)
	if err != nil {
		return domain.ReportExportJob{}, err
	}
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	job.ReportID = report.ID
	job.ExportType = firstNonEmptyStore(job.ExportType, "excel")
	job.Status = firstNonEmptyStore(job.Status, "pending")
	job.CreatedAt = time.Now().UTC()
	filtersRaw, _ := json.Marshal(job.Filters)
	_, err = db.ExecContext(ctx, `
INSERT INTO report_export_jobs (id, report_id, project_id, export_type, filters_json, status, file_path, error_message, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		job.ID, job.ReportID, nullableString(job.ProjectID), job.ExportType, string(filtersRaw), job.Status, nullableString(job.FilePath), nullableString(job.ErrorMessage), nullableString(job.CreatedBy))
	if err != nil {
		return domain.ReportExportJob{}, err
	}
	return job, nil
}

func (s *Store) CompleteReportExportJob(ctx context.Context, jobID string, fileName string, mimeType string, content []byte) (domain.ReportExportJob, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.ReportExportJob{}, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return domain.ReportExportJob{}, err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.ReportExportJob{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO report_export_files (job_id, file_name, mime_type, content)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE file_name = VALUES(file_name), mime_type = VALUES(mime_type), content = VALUES(content)`,
		jobID, fileName, mimeType, content); err != nil {
		return domain.ReportExportJob{}, err
	}
	filePath := "db://report_export_files/" + jobID
	if _, err := tx.ExecContext(ctx, `UPDATE report_export_jobs SET status = 'success', file_path = ?, error_message = NULL, finished_at = NOW() WHERE id = ?`, filePath, jobID); err != nil {
		return domain.ReportExportJob{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.ReportExportJob{}, err
	}
	return s.ReportExportJob(ctx, jobID)
}

func (s *Store) FailReportExportJob(ctx context.Context, jobID string, message string) error {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `UPDATE report_export_jobs SET status = 'failed', error_message = ?, finished_at = NOW() WHERE id = ?`, message, jobID)
	return err
}

func (s *Store) ReportExportJob(ctx context.Context, jobID string) (domain.ReportExportJob, error) {
	items, err := s.ReportExportJobs(ctx, "")
	if err != nil {
		return domain.ReportExportJob{}, err
	}
	for _, item := range items {
		if item.ID == jobID {
			return item, nil
		}
	}
	return domain.ReportExportJob{}, ErrNotFound
}

func (s *Store) ReportExportFile(ctx context.Context, jobID string) (string, string, []byte, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return "", "", nil, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return "", "", nil, err
	}
	var fileName, mimeType string
	var content []byte
	err = db.QueryRowContext(ctx, `SELECT file_name, mime_type, content FROM report_export_files WHERE job_id = ?`, jobID).Scan(&fileName, &mimeType, &content)
	if errors.Is(err, sql.ErrNoRows) {
		return "", "", nil, ErrNotFound
	}
	return fileName, mimeType, content, err
}

func (s *Store) PraiseRecords(ctx context.Context, projectID string, keyword string) ([]domain.PraiseRecord, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return nil, err
	}
	like := "%" + strings.TrimSpace(keyword) + "%"
	rows, err := db.QueryContext(ctx, `
SELECT id, COALESCE(project_id, ''), DATE_FORMAT(praise_date, '%Y-%m-%d'), COALESCE(praise_type, ''), COALESCE(praise_method, ''), COALESCE(department_id, ''), COALESCE(department_name, ''), COALESCE(staff_id, ''), COALESCE(staff_name, ''), COALESCE(patient_id, ''), COALESCE(patient_name, ''), quantity, reward_amount, COALESCE(content, ''), COALESCE(remark, ''), status, COALESCE(created_by, ''), created_at, updated_at
FROM praise_records
WHERE status <> 'deleted'
  AND (? = '' OR project_id = ?)
  AND (? = '' OR department_name LIKE ? OR staff_name LIKE ? OR patient_name LIKE ? OR content LIKE ?)
ORDER BY praise_date DESC, created_at DESC
LIMIT 300`, projectID, projectID, keyword, like, like, like, like)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.PraiseRecord{}
	for rows.Next() {
		item, err := scanPraiseRecord(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreatePraiseRecord(ctx context.Context, item domain.PraiseRecord) (domain.PraiseRecord, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.PraiseRecord{}, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return domain.PraiseRecord{}, err
	}
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	item.PraiseDate = firstNonEmptyStore(item.PraiseDate, time.Now().Format("2006-01-02"))
	item.Status = firstNonEmptyStore(item.Status, "confirmed")
	if item.Quantity <= 0 {
		item.Quantity = 1
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO praise_records (id, project_id, praise_date, praise_type, praise_method, department_id, department_name, staff_id, staff_name, patient_id, patient_name, quantity, reward_amount, content, remark, status, created_by)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, nullableString(item.ProjectID), item.PraiseDate, nullableString(item.PraiseType), nullableString(item.PraiseMethod), nullableString(item.DepartmentID), nullableString(item.DepartmentName), nullableString(item.StaffID), nullableString(item.StaffName), nullableString(item.PatientID), nullableString(item.PatientName), item.Quantity, item.RewardAmount, nullableString(item.Content), nullableString(item.Remark), item.Status, nullableString(item.CreatedBy))
	if err != nil {
		return domain.PraiseRecord{}, err
	}
	return s.PraiseRecord(ctx, item.ID)
}

func (s *Store) PraiseRecord(ctx context.Context, id string) (domain.PraiseRecord, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.PraiseRecord{}, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return domain.PraiseRecord{}, err
	}
	row := db.QueryRowContext(ctx, `
SELECT id, COALESCE(project_id, ''), DATE_FORMAT(praise_date, '%Y-%m-%d'), COALESCE(praise_type, ''), COALESCE(praise_method, ''), COALESCE(department_id, ''), COALESCE(department_name, ''), COALESCE(staff_id, ''), COALESCE(staff_name, ''), COALESCE(patient_id, ''), COALESCE(patient_name, ''), quantity, reward_amount, COALESCE(content, ''), COALESCE(remark, ''), status, COALESCE(created_by, ''), created_at, updated_at
FROM praise_records
WHERE id = ? AND status <> 'deleted'`, id)
	item, err := scanPraiseRecord(row)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.PraiseRecord{}, ErrNotFound
	}
	return item, err
}

func (s *Store) UpdatePraiseRecord(ctx context.Context, id string, patch domain.PraiseRecord) (domain.PraiseRecord, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.PraiseRecord{}, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return domain.PraiseRecord{}, err
	}
	current, err := s.PraiseRecord(ctx, id)
	if err != nil {
		return domain.PraiseRecord{}, err
	}
	if patch.PraiseDate != "" {
		current.PraiseDate = patch.PraiseDate
	}
	if patch.PraiseType != "" {
		current.PraiseType = patch.PraiseType
	}
	if patch.PraiseMethod != "" {
		current.PraiseMethod = patch.PraiseMethod
	}
	if patch.DepartmentID != "" {
		current.DepartmentID = patch.DepartmentID
	}
	if patch.DepartmentName != "" {
		current.DepartmentName = patch.DepartmentName
	}
	if patch.StaffID != "" {
		current.StaffID = patch.StaffID
	}
	if patch.StaffName != "" {
		current.StaffName = patch.StaffName
	}
	if patch.PatientID != "" {
		current.PatientID = patch.PatientID
	}
	if patch.PatientName != "" {
		current.PatientName = patch.PatientName
	}
	if patch.Quantity > 0 {
		current.Quantity = patch.Quantity
	}
	if patch.RewardAmount >= 0 {
		current.RewardAmount = patch.RewardAmount
	}
	if patch.Content != "" {
		current.Content = patch.Content
	}
	if patch.Remark != "" {
		current.Remark = patch.Remark
	}
	if patch.Status != "" {
		current.Status = patch.Status
	}
	_, err = db.ExecContext(ctx, `
UPDATE praise_records SET praise_date = ?, praise_type = ?, praise_method = ?, department_id = ?, department_name = ?, staff_id = ?, staff_name = ?, patient_id = ?, patient_name = ?, quantity = ?, reward_amount = ?, content = ?, remark = ?, status = ? WHERE id = ?`,
		current.PraiseDate, nullableString(current.PraiseType), nullableString(current.PraiseMethod), nullableString(current.DepartmentID), nullableString(current.DepartmentName), nullableString(current.StaffID), nullableString(current.StaffName), nullableString(current.PatientID), nullableString(current.PatientName), current.Quantity, current.RewardAmount, nullableString(current.Content), nullableString(current.Remark), current.Status, id)
	if err != nil {
		return domain.PraiseRecord{}, err
	}
	return s.PraiseRecord(ctx, id)
}

func (s *Store) DeletePraiseRecord(ctx context.Context, id string) error {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return err
	}
	result, err := db.ExecContext(ctx, `UPDATE praise_records SET status = 'deleted' WHERE id = ?`, id)
	if err != nil {
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}

type praiseRecordScanner interface {
	Scan(dest ...interface{}) error
}

func scanPraiseRecord(scanner praiseRecordScanner) (domain.PraiseRecord, error) {
	var item domain.PraiseRecord
	if err := scanner.Scan(
		&item.ID,
		&item.ProjectID,
		&item.PraiseDate,
		&item.PraiseType,
		&item.PraiseMethod,
		&item.DepartmentID,
		&item.DepartmentName,
		&item.StaffID,
		&item.StaffName,
		&item.PatientID,
		&item.PatientName,
		&item.Quantity,
		&item.RewardAmount,
		&item.Content,
		&item.Remark,
		&item.Status,
		&item.CreatedBy,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return domain.PraiseRecord{}, err
	}
	return item, nil
}

func reportAllowedDepartmentsCSV(filters domain.ReportQueryFilters) string {
	return strings.Join(uniqueStrings(filters.AllowedDepartments), ",")
}

func (s *Store) dbReports(ctx context.Context) ([]domain.Report, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return nil, err
	}
	rows, err := db.QueryContext(ctx, `SELECT id, COALESCE(code, ''), report_type, COALESCE(category, ''), COALESCE(subject_type, ''), COALESCE(default_dimension, ''), COALESCE(CAST(default_filters_json AS CHAR), '{}'), name, COALESCE(description, ''), created_at, updated_at FROM reports ORDER BY FIELD(report_type, 'satisfaction', 'complaint', 'followup', 'custom'), category, created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := []domain.Report{}
	for rows.Next() {
		var report domain.Report
		var filtersRaw string
		if err := rows.Scan(&report.ID, &report.Code, &report.Type, &report.Category, &report.SubjectType, &report.DefaultDimension, &filtersRaw, &report.Name, &report.Description, &report.CreatedAt, &report.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(filtersRaw), &report.DefaultFilters)
		widgets, err := dbReportWidgets(ctx, db, report.ID)
		if err != nil {
			return nil, err
		}
		report.Widgets = widgets
		reports = append(reports, report)
	}
	return reports, rows.Err()
}

func (s *Store) dbReport(ctx context.Context, id string) (domain.Report, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.Report{}, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return domain.Report{}, err
	}
	var report domain.Report
	var filtersRaw string
	err = db.QueryRowContext(ctx, `SELECT id, COALESCE(code, ''), report_type, COALESCE(category, ''), COALESCE(subject_type, ''), COALESCE(default_dimension, ''), COALESCE(CAST(default_filters_json AS CHAR), '{}'), name, COALESCE(description, ''), created_at, updated_at FROM reports WHERE id = ? OR code = ?`, id, id).
		Scan(&report.ID, &report.Code, &report.Type, &report.Category, &report.SubjectType, &report.DefaultDimension, &filtersRaw, &report.Name, &report.Description, &report.CreatedAt, &report.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Report{}, ErrNotFound
	}
	if err != nil {
		return domain.Report{}, err
	}
	_ = json.Unmarshal([]byte(filtersRaw), &report.DefaultFilters)
	widgets, err := dbReportWidgets(ctx, db, report.ID)
	if err != nil {
		return domain.Report{}, err
	}
	report.Widgets = widgets
	return report, nil
}

func dbReportWidgets(ctx context.Context, db *sql.DB, reportID string) ([]domain.ReportWidget, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, report_id, widget_type, title, COALESCE(CAST(query_json AS CHAR), '{}'), COALESCE(CAST(vis_spec_json AS CHAR), '{}'), COALESCE(data_source_id, ''), created_at FROM report_widgets WHERE report_id = ? ORDER BY created_at`, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	widgets := []domain.ReportWidget{}
	for rows.Next() {
		var widget domain.ReportWidget
		var queryRaw, visRaw string
		if err := rows.Scan(&widget.ID, &widget.ReportID, &widget.Type, &widget.Title, &queryRaw, &visRaw, &widget.DataSource, &widget.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(queryRaw), &widget.Query)
		_ = json.Unmarshal([]byte(visRaw), &widget.VisSpec)
		if source, ok := widget.Query["source"].(string); ok && widget.DataSource == "" {
			widget.DataSource = source
		}
		widgets = append(widgets, widget)
	}
	return widgets, rows.Err()
}

func (s *Store) createDBReport(ctx context.Context, report domain.Report) (domain.Report, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.Report{}, err
	}
	defer db.Close()
	if err := s.EnsureReportTables(ctx); err != nil {
		return domain.Report{}, err
	}
	if report.ID == "" {
		report.ID = uuid.NewString()
	}
	report.Type = firstNonEmptyStore(report.Type, "custom")
	filters, _ := json.Marshal(report.DefaultFilters)
	if _, err := db.ExecContext(ctx, `INSERT INTO reports (id, code, report_type, category, subject_type, default_dimension, default_filters_json, name, description) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		report.ID, nullableString(report.Code), report.Type, nullableString(report.Category), nullableString(report.SubjectType), nullableString(report.DefaultDimension), string(filters), report.Name, report.Description); err != nil {
		return domain.Report{}, err
	}
	return s.dbReport(ctx, report.ID)
}

func (s *Store) updateDBReport(ctx context.Context, id string, patch domain.Report) (domain.Report, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.Report{}, err
	}
	defer db.Close()
	if _, err := s.dbReport(ctx, id); err != nil {
		return domain.Report{}, err
	}
	if _, err := db.ExecContext(ctx, `
UPDATE reports SET
  code = COALESCE(NULLIF(?, ''), code),
  report_type = COALESCE(NULLIF(?, ''), report_type),
  category = COALESCE(NULLIF(?, ''), category),
  subject_type = COALESCE(NULLIF(?, ''), subject_type),
  default_dimension = COALESCE(NULLIF(?, ''), default_dimension),
  name = COALESCE(NULLIF(?, ''), name),
  description = COALESCE(NULLIF(?, ''), description)
WHERE id = ?`, patch.Code, patch.Type, patch.Category, patch.SubjectType, patch.DefaultDimension, patch.Name, patch.Description, id); err != nil {
		return domain.Report{}, err
	}
	return s.dbReport(ctx, id)
}

func (s *Store) addDBReportWidget(ctx context.Context, reportID string, widget domain.ReportWidget) (domain.ReportWidget, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.ReportWidget{}, err
	}
	defer db.Close()
	if _, err := s.dbReport(ctx, reportID); err != nil {
		return domain.ReportWidget{}, err
	}
	if widget.ID == "" {
		widget.ID = uuid.NewString()
	}
	widget.ReportID = reportID
	widget.CreatedAt = time.Now().UTC()
	widget.Type = firstNonEmptyStore(widget.Type, "table")
	raw, _ := json.Marshal(map[string]string{"source": widget.DataSource})
	if _, err := db.ExecContext(ctx, `INSERT INTO report_widgets (id, report_id, widget_type, title, query_json, vis_spec_json, data_source_id) VALUES (?, ?, ?, ?, ?, '{}', NULL)`,
		widget.ID, widget.ReportID, widget.Type, widget.Title, string(raw)); err != nil {
		return domain.ReportWidget{}, err
	}
	return widget, nil
}

func (s *Store) queryDBReport(ctx context.Context, reportID string, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	report, err := s.dbReport(ctx, reportID)
	if err != nil {
		return nil, err
	}
	switch report.Code {
	case "department_satisfaction":
		return s.queryDepartmentSatisfactionReport(ctx, projectID, filters)
	case "department_question_satisfaction":
		return s.queryDepartmentQuestionReport(ctx, projectID, filters)
	case "question_option_distribution":
		return s.queryQuestionOptionReport(ctx, projectID, filters)
	case "low_score_reason":
		return s.queryLowScoreReasonReport(ctx, projectID, filters)
	case "comments_suggestions":
		return s.queryCommentsReport(ctx, projectID, filters)
	case "satisfaction_trend":
		return s.querySatisfactionTrendReport(ctx, projectID, filters)
	case "staff_department_satisfaction":
		return s.queryDepartmentSatisfactionReport(ctx, projectID, filters)
	case "praise_statistics":
		return s.queryPraiseReport(ctx, projectID, filters)
	}
	switch report.Type {
	case "satisfaction":
		return s.querySatisfactionReport(ctx, projectID)
	case "complaint":
		return s.queryComplaintReport(ctx)
	default:
		return s.queryFollowupReport(ctx, projectID)
	}
}

func (s *Store) querySatisfactionReport(ctx context.Context, projectID string) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `
SELECT dimension_type, dimension_value, indicator_name,
       ROUND(COALESCE(AVG(score_value), 0), 2) AS score,
       COUNT(DISTINCT submission_id) AS sample_count,
       ROUND(SUM(CASE WHEN quality_status = 'valid' THEN 1 ELSE 0 END) * 100 / NULLIF(COUNT(DISTINCT submission_id), 0), 2) AS valid_rate
FROM (
  SELECT '指标' AS dimension_type,
         COALESCE(ss.project_id, '') AS project_id,
         COALESCE(si.name, siq.question_label, sa.question_label, '未绑定指标') AS dimension_value,
         COALESCE(si.name, siq.question_label, sa.question_label, '未绑定指标') AS indicator_name,
         ss.id AS submission_id,
         ss.quality_status,
         sa.score * COALESCE(siq.weight, 1) AS score_value
  FROM survey_submissions ss
  JOIN survey_submission_answers sa ON sa.submission_id = ss.id AND sa.score IS NOT NULL
  LEFT JOIN satisfaction_indicator_questions siq ON siq.form_template_id = ss.form_template_id AND siq.question_id = sa.question_id
  LEFT JOIN satisfaction_indicators si ON si.id = siq.indicator_id
  WHERE ss.status <> 'deleted'

  UNION ALL

  SELECT '科室' AS dimension_type,
         COALESCE(ss.project_id, '') AS project_id,
         COALESCE(cv.department_name, sis.department_name, '未绑定科室') AS dimension_value,
         COALESCE(si.name, siq.question_label, sa.question_label, sis.indicator_id, '综合满意度') AS indicator_name,
         ss.id AS submission_id,
         ss.quality_status,
         COALESCE(sa.score * COALESCE(siq.weight, 1), sis.score) AS score_value
  FROM survey_submissions ss
  LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
  LEFT JOIN survey_submission_answers sa ON sa.submission_id = ss.id AND sa.score IS NOT NULL
  LEFT JOIN satisfaction_indicator_questions siq ON siq.form_template_id = ss.form_template_id AND siq.question_id = sa.question_id
  LEFT JOIN satisfaction_indicators si ON si.id = siq.indicator_id
  LEFT JOIN satisfaction_indicator_scores sis ON sis.project_id = ss.project_id AND (sis.patient_id = ss.patient_id OR sis.visit_id = ss.visit_id)
  WHERE ss.status <> 'deleted' AND COALESCE(sa.score, sis.score) IS NOT NULL

  UNION ALL

  SELECT '医生' AS dimension_type,
         COALESCE(ss.project_id, '') AS project_id,
         COALESCE(cv.attending_doctor, pd.doctor_name, sis.doctor_name, '未绑定医生') AS dimension_value,
         COALESCE(si.name, siq.question_label, sa.question_label, sis.indicator_id, '综合满意度') AS indicator_name,
         ss.id AS submission_id,
         ss.quality_status,
         COALESCE(sa.score * COALESCE(siq.weight, 1), sis.score) AS score_value
  FROM survey_submissions ss
  LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
  LEFT JOIN patient_diagnoses pd ON pd.visit_id = ss.visit_id
  LEFT JOIN survey_submission_answers sa ON sa.submission_id = ss.id AND sa.score IS NOT NULL
  LEFT JOIN satisfaction_indicator_questions siq ON siq.form_template_id = ss.form_template_id AND siq.question_id = sa.question_id
  LEFT JOIN satisfaction_indicators si ON si.id = siq.indicator_id
  LEFT JOIN satisfaction_indicator_scores sis ON sis.project_id = ss.project_id AND (sis.patient_id = ss.patient_id OR sis.visit_id = ss.visit_id)
  WHERE ss.status <> 'deleted' AND COALESCE(sa.score, sis.score) IS NOT NULL

  UNION ALL

  SELECT '病种' AS dimension_type,
         COALESCE(ss.project_id, '') AS project_id,
         COALESCE(pd.diagnosis_name, cv.diagnosis_name, sis.disease_name, '未绑定病种') AS dimension_value,
         COALESCE(si.name, siq.question_label, sa.question_label, sis.indicator_id, '综合满意度') AS indicator_name,
         ss.id AS submission_id,
         ss.quality_status,
         COALESCE(sa.score * COALESCE(siq.weight, 1), sis.score) AS score_value
  FROM survey_submissions ss
  LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
  LEFT JOIN patient_diagnoses pd ON pd.visit_id = ss.visit_id
  LEFT JOIN survey_submission_answers sa ON sa.submission_id = ss.id AND sa.score IS NOT NULL
  LEFT JOIN satisfaction_indicator_questions siq ON siq.form_template_id = ss.form_template_id AND siq.question_id = sa.question_id
  LEFT JOIN satisfaction_indicators si ON si.id = siq.indicator_id
  LEFT JOIN satisfaction_indicator_scores sis ON sis.project_id = ss.project_id AND (sis.patient_id = ss.patient_id OR sis.visit_id = ss.visit_id)
  WHERE ss.status <> 'deleted' AND COALESCE(sa.score, sis.score) IS NOT NULL

  UNION ALL

  SELECT '就诊类型' AS dimension_type,
         COALESCE(ss.project_id, '') AS project_id,
         COALESCE(cv.visit_type, sis.visit_type, '未绑定就诊类型') AS dimension_value,
         COALESCE(si.name, siq.question_label, sa.question_label, sis.indicator_id, '综合满意度') AS indicator_name,
         ss.id AS submission_id,
         ss.quality_status,
         COALESCE(sa.score * COALESCE(siq.weight, 1), sis.score) AS score_value
  FROM survey_submissions ss
  LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
  LEFT JOIN survey_submission_answers sa ON sa.submission_id = ss.id AND sa.score IS NOT NULL
  LEFT JOIN satisfaction_indicator_questions siq ON siq.form_template_id = ss.form_template_id AND siq.question_id = sa.question_id
  LEFT JOIN satisfaction_indicators si ON si.id = siq.indicator_id
  LEFT JOIN satisfaction_indicator_scores sis ON sis.project_id = ss.project_id AND (sis.patient_id = ss.patient_id OR sis.visit_id = ss.visit_id)
  WHERE ss.status <> 'deleted' AND COALESCE(sa.score, sis.score) IS NOT NULL
) report_source
WHERE (? = '' OR project_id = ?)
GROUP BY dimension_type, dimension_value, indicator_name
ORDER BY FIELD(dimension_type, '指标', '科室', '医生', '病种', '就诊类型'), score DESC, sample_count DESC`, projectID, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var dimensionType, dimensionValue, indicatorName string
		var score, validRate float64
		var sampleCount int
		if err := rows.Scan(&dimensionType, &dimensionValue, &indicatorName, &score, &sampleCount, &validRate); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"dimensionType": dimensionType, "dimensionValue": dimensionValue, "indicator": indicatorName, "score": score, "sampleCount": sampleCount, "validRate": validRate})
	}
	return map[string]interface{}{"dimensions": []string{"dimensionValue", "dimensionType", "indicator"}, "measures": []string{"score", "sampleCount", "validRate"}, "rows": resultRows}, rows.Err()
}

func (s *Store) queryDepartmentSatisfactionReport(ctx context.Context, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	allowedDepartments := reportAllowedDepartmentsCSV(filters)
	rows, err := db.QueryContext(ctx, `
SELECT COALESCE(cv.department_name, '未绑定科室') AS department,
       COUNT(DISTINCT ss.id) AS sample_count,
       SUM(CASE WHEN ss.quality_status = 'valid' THEN 1 ELSE 0 END) AS valid_count,
       ROUND(AVG(CASE WHEN sa.score <= 5 THEN sa.score * 20 WHEN sa.score <= 10 THEN sa.score * 10 ELSE sa.score END), 2) AS satisfaction,
       ROUND(SUM(CASE WHEN ss.quality_status = 'valid' THEN 1 ELSE 0 END) * 100 / NULLIF(COUNT(DISTINCT ss.id), 0), 2) AS valid_rate
FROM survey_submissions ss
JOIN survey_submission_answers sa ON sa.submission_id = ss.id AND sa.score IS NOT NULL
LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
WHERE ss.status <> 'deleted'
  AND (? = '' OR ss.project_id = ?)
  AND (? = '' OR DATE(ss.submitted_at) >= ?)
  AND (? = '' OR DATE(ss.submitted_at) <= ?)
  AND (? = '' OR COALESCE(cv.department_name, '') LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR FIND_IN_SET(COALESCE(cv.department_name, ''), ?) > 0)
  AND (? = '' OR ss.channel = ?)
GROUP BY COALESCE(cv.department_name, '未绑定科室')
ORDER BY satisfaction DESC, sample_count DESC`, projectID, projectID, filters.DateFrom, filters.DateFrom, filters.DateTo, filters.DateTo, filters.Department, filters.Department, allowedDepartments, allowedDepartments, filters.Channel, filters.Channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	rank := 1
	for rows.Next() {
		var department string
		var sampleCount, validCount int
		var satisfaction, validRate float64
		if err := rows.Scan(&department, &sampleCount, &validCount, &satisfaction, &validRate); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"rank": rank, "department": department, "sampleCount": sampleCount, "validCount": validCount, "satisfaction": satisfaction, "validRate": validRate})
		rank++
	}
	return map[string]interface{}{"dimensions": []string{"department"}, "measures": []string{"satisfaction", "sampleCount", "validRate"}, "rows": resultRows}, rows.Err()
}

func (s *Store) queryDepartmentQuestionReport(ctx context.Context, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	allowedDepartments := reportAllowedDepartmentsCSV(filters)
	rows, err := db.QueryContext(ctx, `
SELECT COALESCE(cv.department_name, '未绑定科室') AS department,
       sa.question_label,
       SUM(CASE WHEN ROUND(sa.score) <= 1 THEN 1 ELSE 0 END) AS very_dissatisfied,
       SUM(CASE WHEN ROUND(sa.score) = 2 THEN 1 ELSE 0 END) AS dissatisfied,
       SUM(CASE WHEN ROUND(sa.score) = 3 THEN 1 ELSE 0 END) AS neutral,
       SUM(CASE WHEN ROUND(sa.score) = 4 THEN 1 ELSE 0 END) AS satisfied,
       SUM(CASE WHEN ROUND(sa.score) >= 5 THEN 1 ELSE 0 END) AS very_satisfied,
       COUNT(DISTINCT ss.id) AS sample_count,
       ROUND(AVG(CASE WHEN sa.score <= 5 THEN sa.score * 20 WHEN sa.score <= 10 THEN sa.score * 10 ELSE sa.score END), 2) AS satisfaction
FROM survey_submissions ss
JOIN survey_submission_answers sa ON sa.submission_id = ss.id AND sa.score IS NOT NULL
LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
WHERE ss.status <> 'deleted'
  AND (? = '' OR ss.project_id = ?)
  AND (? = '' OR DATE(ss.submitted_at) >= ?)
  AND (? = '' OR DATE(ss.submitted_at) <= ?)
  AND (? = '' OR COALESCE(cv.department_name, '') LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR FIND_IN_SET(COALESCE(cv.department_name, ''), ?) > 0)
  AND (? = '' OR ss.channel = ?)
  AND (? = '' OR sa.question_id = ?)
GROUP BY COALESCE(cv.department_name, '未绑定科室'), sa.question_label
ORDER BY department, satisfaction DESC, sample_count DESC`, projectID, projectID, filters.DateFrom, filters.DateFrom, filters.DateTo, filters.DateTo, filters.Department, filters.Department, allowedDepartments, allowedDepartments, filters.Channel, filters.Channel, filters.QuestionID, filters.QuestionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var department, question string
		var veryDissatisfied, dissatisfied, neutral, satisfied, verySatisfied, sampleCount int
		var satisfaction float64
		if err := rows.Scan(&department, &question, &veryDissatisfied, &dissatisfied, &neutral, &satisfied, &verySatisfied, &sampleCount, &satisfaction); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"department": department, "question": question, "veryDissatisfied": veryDissatisfied, "dissatisfied": dissatisfied, "neutral": neutral, "satisfied": satisfied, "verySatisfied": verySatisfied, "sampleCount": sampleCount, "satisfaction": satisfaction})
	}
	return map[string]interface{}{"dimensions": []string{"department", "question"}, "measures": []string{"veryDissatisfied", "dissatisfied", "neutral", "satisfied", "verySatisfied", "sampleCount", "satisfaction"}, "rows": resultRows}, rows.Err()
}

func (s *Store) queryQuestionOptionReport(ctx context.Context, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	allowedDepartments := reportAllowedDepartmentsCSV(filters)
	rows, err := db.QueryContext(ctx, `
SELECT sa.question_label,
       SUM(CASE WHEN ROUND(sa.score) <= 1 THEN 1 ELSE 0 END) AS one_score,
       SUM(CASE WHEN ROUND(sa.score) = 2 THEN 1 ELSE 0 END) AS two_score,
       SUM(CASE WHEN ROUND(sa.score) = 3 THEN 1 ELSE 0 END) AS three_score,
       SUM(CASE WHEN ROUND(sa.score) = 4 THEN 1 ELSE 0 END) AS four_score,
       SUM(CASE WHEN ROUND(sa.score) >= 5 THEN 1 ELSE 0 END) AS five_score,
       COUNT(DISTINCT ss.id) AS sample_count,
       ROUND(AVG(CASE WHEN sa.score <= 5 THEN sa.score * 20 WHEN sa.score <= 10 THEN sa.score * 10 ELSE sa.score END), 2) AS satisfaction
FROM survey_submissions ss
JOIN survey_submission_answers sa ON sa.submission_id = ss.id AND sa.score IS NOT NULL
LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
WHERE ss.status <> 'deleted'
  AND (? = '' OR ss.project_id = ?)
  AND (? = '' OR DATE(ss.submitted_at) >= ?)
  AND (? = '' OR DATE(ss.submitted_at) <= ?)
  AND (? = '' OR COALESCE(cv.department_name, '') LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR FIND_IN_SET(COALESCE(cv.department_name, ''), ?) > 0)
  AND (? = '' OR ss.channel = ?)
  AND (? = '' OR sa.question_id = ?)
GROUP BY sa.question_label
ORDER BY satisfaction DESC, sample_count DESC`, projectID, projectID, filters.DateFrom, filters.DateFrom, filters.DateTo, filters.DateTo, filters.Department, filters.Department, allowedDepartments, allowedDepartments, filters.Channel, filters.Channel, filters.QuestionID, filters.QuestionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var question string
		var one, two, three, four, five, sampleCount int
		var satisfaction float64
		if err := rows.Scan(&question, &one, &two, &three, &four, &five, &sampleCount, &satisfaction); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"question": question, "oneScore": one, "twoScore": two, "threeScore": three, "fourScore": four, "fiveScore": five, "sampleCount": sampleCount, "satisfaction": satisfaction})
	}
	return map[string]interface{}{"dimensions": []string{"question"}, "measures": []string{"oneScore", "twoScore", "threeScore", "fourScore", "fiveScore", "sampleCount", "satisfaction"}, "rows": resultRows}, rows.Err()
}

func (s *Store) queryLowScoreReasonReport(ctx context.Context, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	allowedDepartments := reportAllowedDepartmentsCSV(filters)
	rows, err := db.QueryContext(ctx, `
SELECT COALESCE(sa.question_label, sa.question_id, '低分原因') AS reason_field,
       TRIM(BOTH '"' FROM COALESCE(CAST(sa.answer_json AS CHAR), '未填写')) AS reason_value,
       COUNT(*) AS count_value
FROM survey_submissions ss
JOIN survey_submission_answers sa ON sa.submission_id = ss.id
LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
WHERE ss.status <> 'deleted'
  AND (? = '' OR ss.project_id = ?)
  AND (? = '' OR DATE(ss.submitted_at) >= ?)
  AND (? = '' OR DATE(ss.submitted_at) <= ?)
  AND (? = '' OR COALESCE(cv.department_name, '') LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR FIND_IN_SET(COALESCE(cv.department_name, ''), ?) > 0)
  AND (? = '' OR ss.channel = ?)
  AND (sa.question_id LIKE '%reason%' OR sa.question_id LIKE '%problem%' OR sa.question_label LIKE '%原因%' OR sa.question_label LIKE '%问题%')
GROUP BY reason_field, reason_value
ORDER BY count_value DESC`, projectID, projectID, filters.DateFrom, filters.DateFrom, filters.DateTo, filters.DateTo, filters.Department, filters.Department, allowedDepartments, allowedDepartments, filters.Channel, filters.Channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var field, value string
		var count int
		if err := rows.Scan(&field, &value, &count); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"reasonField": field, "reason": value, "count": count})
	}
	return map[string]interface{}{"dimensions": []string{"reason"}, "measures": []string{"count"}, "rows": resultRows}, rows.Err()
}

func (s *Store) queryCommentsReport(ctx context.Context, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	allowedDepartments := reportAllowedDepartmentsCSV(filters)
	rows, err := db.QueryContext(ctx, `
SELECT ss.id, COALESCE(cv.department_name, '未绑定科室') AS department,
       COALESCE(p.name, '') AS patient_name,
       sa.question_label,
       TRIM(BOTH '"' FROM COALESCE(CAST(sa.answer_json AS CHAR), '')) AS content,
       ss.channel,
       DATE_FORMAT(ss.submitted_at, '%Y-%m-%d %H:%i:%s') AS submitted_at
FROM survey_submissions ss
JOIN survey_submission_answers sa ON sa.submission_id = ss.id
LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
LEFT JOIN patients p ON p.id = ss.patient_id
WHERE ss.status <> 'deleted'
  AND (? = '' OR ss.project_id = ?)
  AND (? = '' OR DATE(ss.submitted_at) >= ?)
  AND (? = '' OR DATE(ss.submitted_at) <= ?)
  AND (? = '' OR COALESCE(cv.department_name, '') LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR FIND_IN_SET(COALESCE(cv.department_name, ''), ?) > 0)
  AND (? = '' OR ss.channel = ?)
  AND sa.score IS NULL
  AND TRIM(BOTH '"' FROM COALESCE(CAST(sa.answer_json AS CHAR), '')) <> ''
ORDER BY ss.submitted_at DESC
LIMIT 300`, projectID, projectID, filters.DateFrom, filters.DateFrom, filters.DateTo, filters.DateTo, filters.Department, filters.Department, allowedDepartments, allowedDepartments, filters.Channel, filters.Channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var id, department, patientName, question, content, channel, submittedAt string
		if err := rows.Scan(&id, &department, &patientName, &question, &content, &channel, &submittedAt); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"submissionId": id, "department": department, "patientName": patientName, "question": question, "content": content, "channel": channel, "submittedAt": submittedAt})
	}
	return map[string]interface{}{"dimensions": []string{"department", "question"}, "measures": []string{"content"}, "rows": resultRows}, rows.Err()
}

func (s *Store) querySatisfactionTrendReport(ctx context.Context, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	allowedDepartments := reportAllowedDepartmentsCSV(filters)
	rows, err := db.QueryContext(ctx, `
SELECT DATE_FORMAT(ss.submitted_at, '%Y-%m') AS month,
       COUNT(DISTINCT ss.id) AS sample_count,
       ROUND(AVG(CASE WHEN sa.score <= 5 THEN sa.score * 20 WHEN sa.score <= 10 THEN sa.score * 10 ELSE sa.score END), 2) AS satisfaction,
       ROUND(SUM(CASE WHEN ss.quality_status = 'valid' THEN 1 ELSE 0 END) * 100 / NULLIF(COUNT(DISTINCT ss.id), 0), 2) AS valid_rate
FROM survey_submissions ss
JOIN survey_submission_answers sa ON sa.submission_id = ss.id AND sa.score IS NOT NULL
LEFT JOIN clinical_visits cv ON cv.id = ss.visit_id
WHERE ss.status <> 'deleted'
  AND (? = '' OR ss.project_id = ?)
  AND (? = '' OR DATE(ss.submitted_at) >= ?)
  AND (? = '' OR DATE(ss.submitted_at) <= ?)
  AND (? = '' OR COALESCE(cv.department_name, '') LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR FIND_IN_SET(COALESCE(cv.department_name, ''), ?) > 0)
  AND (? = '' OR ss.channel = ?)
GROUP BY DATE_FORMAT(ss.submitted_at, '%Y-%m')
ORDER BY month`, projectID, projectID, filters.DateFrom, filters.DateFrom, filters.DateTo, filters.DateTo, filters.Department, filters.Department, allowedDepartments, allowedDepartments, filters.Channel, filters.Channel)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var month string
		var sampleCount int
		var satisfaction, validRate float64
		if err := rows.Scan(&month, &sampleCount, &satisfaction, &validRate); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"month": month, "sampleCount": sampleCount, "satisfaction": satisfaction, "validRate": validRate})
	}
	return map[string]interface{}{"dimensions": []string{"month"}, "measures": []string{"satisfaction", "sampleCount", "validRate"}, "rows": resultRows}, rows.Err()
}

func (s *Store) queryPraiseReport(ctx context.Context, projectID string, filters domain.ReportQueryFilters) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	allowedDepartments := reportAllowedDepartmentsCSV(filters)
	rows, err := db.QueryContext(ctx, `
SELECT COALESCE(department_name, '未绑定科室') AS department,
       COALESCE(staff_name, '未绑定人员') AS staff,
       COALESCE(praise_method, '未填写') AS praise_method,
       COUNT(*) AS praise_count,
       COALESCE(SUM(quantity), 0) AS quantity,
       COALESCE(SUM(reward_amount), 0) AS reward_amount
FROM praise_records
WHERE status <> 'deleted'
  AND (? = '' OR project_id = ?)
  AND (? = '' OR praise_date >= ?)
  AND (? = '' OR praise_date <= ?)
  AND (? = '' OR COALESCE(department_name, '') LIKE CONCAT('%', ?, '%'))
  AND (? = '' OR FIND_IN_SET(COALESCE(department_name, ''), ?) > 0)
GROUP BY COALESCE(department_name, '未绑定科室'), COALESCE(staff_name, '未绑定人员'), COALESCE(praise_method, '未填写')
ORDER BY praise_count DESC, reward_amount DESC`, projectID, projectID, filters.DateFrom, filters.DateFrom, filters.DateTo, filters.DateTo, filters.Department, filters.Department, allowedDepartments, allowedDepartments)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var department, staff, method string
		var praiseCount, quantity int
		var rewardAmount float64
		if err := rows.Scan(&department, &staff, &method, &praiseCount, &quantity, &rewardAmount); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"department": department, "staff": staff, "praiseMethod": method, "praiseCount": praiseCount, "quantity": quantity, "rewardAmount": rewardAmount})
	}
	return map[string]interface{}{"dimensions": []string{"department", "staff", "praiseMethod"}, "measures": []string{"praiseCount", "quantity", "rewardAmount"}, "rows": resultRows}, rows.Err()
}

func (s *Store) queryComplaintReport(ctx context.Context) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `
SELECT COALESCE(responsible_department, '未分配') AS department,
       COUNT(*) AS total,
       SUM(CASE WHEN kind = 'complaint' THEN 1 ELSE 0 END) AS complaints,
       SUM(CASE WHEN kind = 'praise' THEN 1 ELSE 0 END) AS praises,
       SUM(CASE WHEN status = 'archived' THEN 1 ELSE 0 END) AS archived
FROM evaluation_complaints
WHERE status <> 'deleted'
GROUP BY COALESCE(responsible_department, '未分配')
ORDER BY total DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var department string
		var total, complaints, praises, archived int
		if err := rows.Scan(&department, &total, &complaints, &praises, &archived); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"department": department, "total": total, "complaints": complaints, "praises": praises, "archived": archived})
	}
	return map[string]interface{}{"dimensions": []string{"department"}, "measures": []string{"total", "complaints", "praises", "archived"}, "rows": resultRows}, rows.Err()
}

func (s *Store) queryFollowupReport(ctx context.Context, projectID string) (map[string]interface{}, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `
SELECT COALESCE(DATE_FORMAT(followed_at, '%Y-%m'), DATE_FORMAT(created_at, '%Y-%m')) AS month,
       COUNT(*) AS submissions,
       SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) AS completed,
       ROUND(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) * 100 / NULLIF(COUNT(*), 0), 2) AS completion_rate
FROM followup_records
WHERE (? = '' OR project_id = ?)
GROUP BY COALESCE(DATE_FORMAT(followed_at, '%Y-%m'), DATE_FORMAT(created_at, '%Y-%m'))
ORDER BY month`, projectID, projectID)
	if err != nil {
		if strings.Contains(err.Error(), "followup_records") {
			return map[string]interface{}{"dimensions": []string{"month"}, "measures": []string{"submissions", "completed", "completionRate"}, "rows": []map[string]interface{}{}}, nil
		}
		return nil, err
	}
	defer rows.Close()
	resultRows := []map[string]interface{}{}
	for rows.Next() {
		var month string
		var submissions, completed int
		var completionRate float64
		if err := rows.Scan(&month, &submissions, &completed, &completionRate); err != nil {
			return nil, err
		}
		resultRows = append(resultRows, map[string]interface{}{"month": month, "submissions": submissions, "completed": completed, "completionRate": completionRate})
	}
	return map[string]interface{}{"dimensions": []string{"month"}, "measures": []string{"submissions", "completed", "completionRate"}, "rows": resultRows}, rows.Err()
}
