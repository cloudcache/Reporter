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
  report_type VARCHAR(60) NOT NULL DEFAULT 'custom',
  name VARCHAR(180) NOT NULL,
  description TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
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
	return seedReports(ctx, db)
}

func seedReports(ctx context.Context, db *sql.DB) error {
	reports := []domain.Report{
		{ID: "RP001", Type: "followup", Name: "随访完成情况月报", Description: "从随访记录聚合随访提交量、完成量和完成率"},
		{ID: "RP002", Type: "satisfaction", Name: "满意度分析报告", Description: "从满意度答卷、访谈表单和指标体系聚合科室、指标、渠道和低分原因"},
		{ID: "RP003", Type: "complaint", Name: "评价投诉分析报告", Description: "从评价投诉台账聚合投诉、表扬、处理状态和责任科室"},
	}
	for _, report := range reports {
		if _, err := db.ExecContext(ctx, `
INSERT INTO reports (id, report_type, name, description)
VALUES (?, ?, ?, ?)
ON DUPLICATE KEY UPDATE report_type = VALUES(report_type), name = VALUES(name), description = VALUES(description)`,
			report.ID, report.Type, report.Name, report.Description); err != nil {
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
VALUES (?, ?, ?, ?, CAST(? AS JSON), JSON_OBJECT(), NULL)
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

func (s *Store) QueryReportData(ctx context.Context, reportID string, projectID string) (map[string]interface{}, error) {
	return s.queryDBReport(ctx, reportID, projectID)
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
	rows, err := db.QueryContext(ctx, `SELECT id, report_type, name, COALESCE(description, ''), created_at, updated_at FROM reports ORDER BY FIELD(report_type, 'satisfaction', 'complaint', 'followup', 'custom'), created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	reports := []domain.Report{}
	for rows.Next() {
		var report domain.Report
		if err := rows.Scan(&report.ID, &report.Type, &report.Name, &report.Description, &report.CreatedAt, &report.UpdatedAt); err != nil {
			return nil, err
		}
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
	err = db.QueryRowContext(ctx, `SELECT id, report_type, name, COALESCE(description, ''), created_at, updated_at FROM reports WHERE id = ?`, id).
		Scan(&report.ID, &report.Type, &report.Name, &report.Description, &report.CreatedAt, &report.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Report{}, ErrNotFound
	}
	if err != nil {
		return domain.Report{}, err
	}
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
	if _, err := db.ExecContext(ctx, `INSERT INTO reports (id, report_type, name, description) VALUES (?, ?, ?, ?)`, report.ID, report.Type, report.Name, report.Description); err != nil {
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
  report_type = COALESCE(NULLIF(?, ''), report_type),
  name = COALESCE(NULLIF(?, ''), name),
  description = COALESCE(NULLIF(?, ''), description)
WHERE id = ?`, patch.Type, patch.Name, patch.Description, id); err != nil {
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
	if _, err := db.ExecContext(ctx, `INSERT INTO report_widgets (id, report_id, widget_type, title, query_json, vis_spec_json, data_source_id) VALUES (?, ?, ?, ?, CAST(? AS JSON), JSON_OBJECT(), NULL)`,
		widget.ID, widget.ReportID, widget.Type, widget.Title, string(raw)); err != nil {
		return domain.ReportWidget{}, err
	}
	return widget, nil
}

func (s *Store) queryDBReport(ctx context.Context, reportID string, projectID string) (map[string]interface{}, error) {
	report, err := s.dbReport(ctx, reportID)
	if err != nil {
		return nil, err
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
