package store

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) EnsureClinicalFactTables(ctx context.Context) error {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := ensureClinicalFactTables(ctx, db); err != nil {
		return err
	}
	return seedClinicalFacts(ctx, db)
}

func ensureClinicalFactTables(ctx context.Context, db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS patient_diagnoses (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  diagnosis_code VARCHAR(80) NULL,
  diagnosis_name VARCHAR(180) NOT NULL,
  diagnosis_type VARCHAR(60) NOT NULL DEFAULT 'primary',
  diagnosed_at DATETIME NULL,
  department_name VARCHAR(120) NULL,
  doctor_name VARCHAR(120) NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_patient_diagnoses_patient (patient_id),
  INDEX idx_patient_diagnoses_visit (visit_id),
  INDEX idx_patient_diagnoses_code (diagnosis_code)
)`,
		`CREATE TABLE IF NOT EXISTS patient_histories (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  history_type VARCHAR(60) NOT NULL,
  title VARCHAR(180) NOT NULL,
  content TEXT NULL,
  recorded_at DATETIME NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_patient_histories_patient (patient_id),
  INDEX idx_patient_histories_type (history_type)
)`,
		`CREATE TABLE IF NOT EXISTS medication_orders (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  order_no VARCHAR(120) NULL,
  prescription_no VARCHAR(120) NULL,
  drug_code VARCHAR(80) NULL,
  drug_name VARCHAR(180) NOT NULL,
  generic_name VARCHAR(180) NULL,
  specification VARCHAR(180) NULL,
  dosage VARCHAR(80) NULL,
  dosage_unit VARCHAR(40) NULL,
  frequency VARCHAR(80) NULL,
  route VARCHAR(80) NULL,
  start_at DATETIME NULL,
  end_at DATETIME NULL,
  days INT NULL,
  quantity DECIMAL(10,2) NULL,
  manufacturer VARCHAR(180) NULL,
  doctor_name VARCHAR(120) NULL,
  pharmacist_name VARCHAR(120) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  adverse_reaction TEXT NULL,
  compliance VARCHAR(60) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_medication_orders_patient (patient_id),
  INDEX idx_medication_orders_visit (visit_id),
  INDEX idx_medication_orders_drug (drug_code)
)`,
		`CREATE TABLE IF NOT EXISTS lab_reports (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  report_no VARCHAR(120) NOT NULL,
  report_name VARCHAR(180) NOT NULL,
  specimen VARCHAR(80) NULL,
  ordered_at DATETIME NULL,
  reported_at DATETIME NULL,
  department_name VARCHAR(120) NULL,
  doctor_name VARCHAR(120) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'reported',
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_lab_report_no (report_no),
  INDEX idx_lab_reports_patient (patient_id),
  INDEX idx_lab_reports_visit (visit_id)
)`,
		`CREATE TABLE IF NOT EXISTS lab_results (
  id CHAR(36) PRIMARY KEY,
  report_id CHAR(36) NOT NULL,
  item_code VARCHAR(80) NULL,
  item_name VARCHAR(180) NOT NULL,
  result_value VARCHAR(120) NULL,
  unit VARCHAR(60) NULL,
  reference_range VARCHAR(120) NULL,
  abnormal_flag VARCHAR(40) NULL,
  numeric_value DECIMAL(12,4) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_lab_results_report (report_id),
  INDEX idx_lab_results_item (item_code),
  INDEX idx_lab_results_abnormal (abnormal_flag)
)`,
		`CREATE TABLE IF NOT EXISTS exam_reports (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  exam_no VARCHAR(120) NOT NULL,
  exam_type VARCHAR(80) NULL,
  exam_name VARCHAR(180) NOT NULL,
  body_part VARCHAR(120) NULL,
  report_conclusion TEXT NULL,
  report_findings TEXT NULL,
  ordered_at DATETIME NULL,
  reported_at DATETIME NULL,
  department_name VARCHAR(120) NULL,
  doctor_name VARCHAR(120) NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_exam_report_no (exam_no),
  INDEX idx_exam_reports_patient (patient_id),
  INDEX idx_exam_reports_visit (visit_id)
)`,
		`CREATE TABLE IF NOT EXISTS surgery_records (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  operation_code VARCHAR(80) NULL,
  operation_name VARCHAR(180) NOT NULL,
  operation_date DATETIME NULL,
  surgeon_name VARCHAR(120) NULL,
  anesthesia_type VARCHAR(80) NULL,
  operation_level VARCHAR(60) NULL,
  wound_grade VARCHAR(60) NULL,
  outcome VARCHAR(120) NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_surgery_records_patient (patient_id),
  INDEX idx_surgery_records_visit (visit_id),
  INDEX idx_surgery_records_operation (operation_code)
)`,
		`CREATE TABLE IF NOT EXISTS followup_records (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  task_id VARCHAR(80) NULL,
  project_id VARCHAR(80) NULL,
  followup_type VARCHAR(80) NULL,
  channel VARCHAR(40) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'completed',
  summary TEXT NULL,
  satisfaction_score DECIMAL(5,2) NULL,
  risk_level VARCHAR(40) NULL,
  followed_at DATETIME NULL,
  operator_name VARCHAR(120) NULL,
  source_system VARCHAR(80) NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_followup_records_patient (patient_id),
  INDEX idx_followup_records_task (task_id),
  INDEX idx_followup_records_project (project_id)
)`,
		`CREATE TABLE IF NOT EXISTS interview_extracted_facts (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_id CHAR(36) NULL,
  interview_id CHAR(36) NULL,
  fact_type VARCHAR(80) NOT NULL,
  fact_key VARCHAR(120) NOT NULL,
  fact_label VARCHAR(180) NOT NULL,
  fact_value TEXT NULL,
  confidence DECIMAL(5,4) NULL,
  extracted_at DATETIME NULL,
  source_text TEXT NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_interview_facts_patient (patient_id),
  INDEX idx_interview_facts_key (fact_key),
  INDEX idx_interview_facts_interview (interview_id)
)`,
		`CREATE TABLE IF NOT EXISTS satisfaction_indicator_scores (
  id CHAR(36) PRIMARY KEY,
  project_id VARCHAR(80) NOT NULL,
  indicator_id CHAR(36) NOT NULL,
  patient_id CHAR(36) NULL,
  visit_id CHAR(36) NULL,
  department_name VARCHAR(120) NULL,
  doctor_name VARCHAR(120) NULL,
  nurse_name VARCHAR(120) NULL,
  disease_name VARCHAR(180) NULL,
  visit_type VARCHAR(60) NULL,
  score DECIMAL(10,2) NOT NULL DEFAULT 0,
  sample_count INT NOT NULL DEFAULT 0,
  score_period DATE NULL,
  source_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_satisfaction_scores_project (project_id),
  INDEX idx_satisfaction_scores_indicator (indicator_id),
  INDEX idx_satisfaction_scores_patient (patient_id),
  INDEX idx_satisfaction_scores_department (department_name),
  INDEX idx_satisfaction_scores_period (score_period)
)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func seedClinicalFacts(ctx context.Context, db *sql.DB) error {
	statements := []struct {
		sql  string
		args []interface{}
	}{
		{`INSERT INTO patient_diagnoses (id, patient_id, visit_id, diagnosis_code, diagnosis_name, diagnosis_type, diagnosed_at, department_name, doctor_name, source_system) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE diagnosis_name = VALUES(diagnosis_name)`,
			[]interface{}{"DX-P001-1", "P001", "V001", "I10", "高血压", "primary", "2026-05-10 09:30:00", "心内科", "王医生", "HIS"}},
		{`INSERT INTO patient_histories (id, patient_id, history_type, title, content, recorded_at, source_system) VALUES (?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE content = VALUES(content)`,
			[]interface{}{"HX-P001-1", "P001", "past", "既往史", "高血压病史 5 年，规律服药。", "2026-05-10 09:40:00", "EMR"}},
		{`INSERT INTO medication_orders (id, patient_id, visit_id, order_no, drug_code, drug_name, generic_name, specification, dosage, dosage_unit, frequency, route, start_at, days, quantity, doctor_name, status, compliance) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE drug_name = VALUES(drug_name), compliance = VALUES(compliance)`,
			[]interface{}{"MED-P001-1", "P001", "V001", "ORD20260510001", "YP-AML", "苯磺酸氨氯地平片", "氨氯地平", "5mg*28片", "5", "mg", "qd", "口服", "2026-05-10 10:00:00", 28, 28, "王医生", "active", "good"}},
		{`INSERT INTO lab_reports (id, patient_id, visit_id, report_no, report_name, specimen, reported_at, department_name, doctor_name, status, source_system) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE report_name = VALUES(report_name)`,
			[]interface{}{"LAB-P001-1", "P001", "V001", "LAB20260510001", "肝肾功能", "血清", "2026-05-10 14:20:00", "检验科", "检验医生", "reported", "LIS"}},
		{`INSERT INTO lab_results (id, report_id, item_code, item_name, result_value, unit, reference_range, abnormal_flag, numeric_value) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE result_value = VALUES(result_value)`,
			[]interface{}{"LAR-P001-1", "LAB-P001-1", "CREA", "肌酐", "72", "umol/L", "57-97", "normal", 72}},
		{`INSERT INTO exam_reports (id, patient_id, visit_id, exam_no, exam_type, exam_name, body_part, report_conclusion, reported_at, department_name, doctor_name, source_system) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE report_conclusion = VALUES(report_conclusion)`,
			[]interface{}{"EXAM-P001-1", "P001", "V001", "EX20260510001", "ECG", "十二导联心电图", "心脏", "窦性心律，未见明显急性缺血改变。", "2026-05-10 11:00:00", "功能科", "检查医生", "PACS"}},
		{`INSERT INTO followup_records (id, patient_id, visit_id, task_id, project_id, followup_type, channel, status, summary, satisfaction_score, risk_level, followed_at, operator_name, source_system) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE summary = VALUES(summary), satisfaction_score = VALUES(satisfaction_score)`,
			[]interface{}{"FUR-P001-1", "P001", "V001", "FT001", "SAT-OUTPATIENT", "满意度随访", "phone", "completed", "患者反馈候诊时间略长，用药说明清楚。", 86.0, "low", "2026-05-12 15:00:00", "随访员A", "followup"}},
		{`INSERT INTO interview_extracted_facts (id, patient_id, visit_id, interview_id, fact_type, fact_key, fact_label, fact_value, confidence, extracted_at, source_text) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE fact_value = VALUES(fact_value), confidence = VALUES(confidence)`,
			[]interface{}{"FACT-P001-1", "P001", "V001", "INT-P001-1", "experience", "waiting_time", "候诊时间", "候诊时间偏长", 0.9200, "2026-05-12 15:05:00", "等候时间有点久，其他还可以。"}},
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement.sql, statement.args...); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) Patient360(ctx context.Context, patientID string) (domain.Patient360, error) {
	patient, ok := s.Patient(patientID)
	if !ok {
		return domain.Patient360{}, ErrNotFound
	}
	diagnoses, err := s.PatientDiagnoses(ctx, patientID)
	if err != nil {
		return domain.Patient360{}, err
	}
	histories, err := s.PatientHistories(ctx, patientID)
	if err != nil {
		return domain.Patient360{}, err
	}
	medications, err := s.MedicationOrders(ctx, patientID)
	if err != nil {
		return domain.Patient360{}, err
	}
	labs, err := s.LabReports(ctx, patientID)
	if err != nil {
		return domain.Patient360{}, err
	}
	exams, err := s.ExamReports(ctx, patientID)
	if err != nil {
		return domain.Patient360{}, err
	}
	surgeries, err := s.SurgeryRecords(ctx, patientID)
	if err != nil {
		return domain.Patient360{}, err
	}
	followups, err := s.FollowupRecords(ctx, patientID)
	if err != nil {
		return domain.Patient360{}, err
	}
	facts, err := s.InterviewExtractedFacts(ctx, patientID)
	if err != nil {
		return domain.Patient360{}, err
	}
	return domain.Patient360{
		Patient:         patient,
		Visits:          s.Visits(patientID),
		MedicalRecords:  s.MedicalRecords(patientID),
		Diagnoses:       diagnoses,
		Histories:       histories,
		Medications:     medications,
		LabReports:      labs,
		ExamReports:     exams,
		Surgeries:       surgeries,
		FollowupRecords: followups,
		InterviewFacts:  facts,
	}, nil
}

func (s *Store) PatientDiagnoses(ctx context.Context, patientID string) ([]domain.PatientDiagnosis, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return []domain.PatientDiagnosis{}, nil
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, patient_id, COALESCE(visit_id, ''), COALESCE(diagnosis_code, ''), diagnosis_name, diagnosis_type, COALESCE(DATE_FORMAT(diagnosed_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(department_name, ''), COALESCE(doctor_name, ''), COALESCE(source_system, ''), created_at, updated_at FROM patient_diagnoses WHERE patient_id = ? ORDER BY diagnosed_at DESC, created_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.PatientDiagnosis
	for rows.Next() {
		var item domain.PatientDiagnosis
		if err := rows.Scan(&item.ID, &item.PatientID, &item.VisitID, &item.DiagnosisCode, &item.DiagnosisName, &item.DiagnosisType, &item.DiagnosedAt, &item.DepartmentName, &item.DoctorName, &item.SourceSystem, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) PatientHistories(ctx context.Context, patientID string) ([]domain.PatientHistory, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return []domain.PatientHistory{}, nil
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, patient_id, history_type, title, COALESCE(content, ''), COALESCE(DATE_FORMAT(recorded_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(source_system, ''), created_at, updated_at FROM patient_histories WHERE patient_id = ? ORDER BY recorded_at DESC, created_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.PatientHistory
	for rows.Next() {
		var item domain.PatientHistory
		if err := rows.Scan(&item.ID, &item.PatientID, &item.HistoryType, &item.Title, &item.Content, &item.RecordedAt, &item.SourceSystem, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) MedicationOrders(ctx context.Context, patientID string) ([]domain.MedicationOrder, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return []domain.MedicationOrder{}, nil
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, patient_id, COALESCE(visit_id, ''), COALESCE(order_no, ''), COALESCE(prescription_no, ''), COALESCE(drug_code, ''), drug_name, COALESCE(generic_name, ''), COALESCE(specification, ''), COALESCE(dosage, ''), COALESCE(dosage_unit, ''), COALESCE(frequency, ''), COALESCE(route, ''), COALESCE(DATE_FORMAT(start_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(DATE_FORMAT(end_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(days, 0), COALESCE(quantity, 0), COALESCE(manufacturer, ''), COALESCE(doctor_name, ''), COALESCE(pharmacist_name, ''), status, COALESCE(adverse_reaction, ''), COALESCE(compliance, ''), created_at, updated_at FROM medication_orders WHERE patient_id = ? ORDER BY start_at DESC, created_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.MedicationOrder
	for rows.Next() {
		var item domain.MedicationOrder
		if err := rows.Scan(&item.ID, &item.PatientID, &item.VisitID, &item.OrderNo, &item.PrescriptionNo, &item.DrugCode, &item.DrugName, &item.GenericName, &item.Specification, &item.Dosage, &item.DosageUnit, &item.Frequency, &item.Route, &item.StartAt, &item.EndAt, &item.Days, &item.Quantity, &item.Manufacturer, &item.DoctorName, &item.PharmacistName, &item.Status, &item.AdverseReaction, &item.Compliance, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) LabReports(ctx context.Context, patientID string) ([]domain.LabReport, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return []domain.LabReport{}, nil
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, patient_id, COALESCE(visit_id, ''), report_no, report_name, COALESCE(specimen, ''), COALESCE(DATE_FORMAT(ordered_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(DATE_FORMAT(reported_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(department_name, ''), COALESCE(doctor_name, ''), status, COALESCE(source_system, ''), created_at, updated_at FROM lab_reports WHERE patient_id = ? ORDER BY reported_at DESC, created_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.LabReport
	for rows.Next() {
		var item domain.LabReport
		if err := rows.Scan(&item.ID, &item.PatientID, &item.VisitID, &item.ReportNo, &item.ReportName, &item.Specimen, &item.OrderedAt, &item.ReportedAt, &item.DepartmentName, &item.DoctorName, &item.Status, &item.SourceSystem, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		results, err := labResults(ctx, db, item.ID)
		if err != nil {
			return nil, err
		}
		item.Results = results
		items = append(items, item)
	}
	return items, rows.Err()
}

func labResults(ctx context.Context, db *sql.DB, reportID string) ([]domain.LabResult, error) {
	rows, err := db.QueryContext(ctx, `SELECT id, report_id, COALESCE(item_code, ''), item_name, COALESCE(result_value, ''), COALESCE(unit, ''), COALESCE(reference_range, ''), COALESCE(abnormal_flag, ''), COALESCE(numeric_value, 0), created_at FROM lab_results WHERE report_id = ? ORDER BY item_name`, reportID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.LabResult
	for rows.Next() {
		var item domain.LabResult
		if err := rows.Scan(&item.ID, &item.ReportID, &item.ItemCode, &item.ItemName, &item.ResultValue, &item.Unit, &item.ReferenceRange, &item.AbnormalFlag, &item.NumericValue, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ExamReports(ctx context.Context, patientID string) ([]domain.ExamReport, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return []domain.ExamReport{}, nil
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, patient_id, COALESCE(visit_id, ''), exam_no, COALESCE(exam_type, ''), exam_name, COALESCE(body_part, ''), COALESCE(report_conclusion, ''), COALESCE(report_findings, ''), COALESCE(DATE_FORMAT(ordered_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(DATE_FORMAT(reported_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(department_name, ''), COALESCE(doctor_name, ''), COALESCE(source_system, ''), created_at, updated_at FROM exam_reports WHERE patient_id = ? ORDER BY reported_at DESC, created_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.ExamReport
	for rows.Next() {
		var item domain.ExamReport
		if err := rows.Scan(&item.ID, &item.PatientID, &item.VisitID, &item.ExamNo, &item.ExamType, &item.ExamName, &item.BodyPart, &item.ReportConclusion, &item.ReportFindings, &item.OrderedAt, &item.ReportedAt, &item.DepartmentName, &item.DoctorName, &item.SourceSystem, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SurgeryRecords(ctx context.Context, patientID string) ([]domain.SurgeryRecord, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return []domain.SurgeryRecord{}, nil
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, patient_id, COALESCE(visit_id, ''), COALESCE(operation_code, ''), operation_name, COALESCE(DATE_FORMAT(operation_date, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(surgeon_name, ''), COALESCE(anesthesia_type, ''), COALESCE(operation_level, ''), COALESCE(wound_grade, ''), COALESCE(outcome, ''), COALESCE(source_system, ''), created_at, updated_at FROM surgery_records WHERE patient_id = ? ORDER BY operation_date DESC, created_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.SurgeryRecord
	for rows.Next() {
		var item domain.SurgeryRecord
		if err := rows.Scan(&item.ID, &item.PatientID, &item.VisitID, &item.OperationCode, &item.OperationName, &item.OperationDate, &item.SurgeonName, &item.AnesthesiaType, &item.OperationLevel, &item.WoundGrade, &item.Outcome, &item.SourceSystem, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) FollowupRecords(ctx context.Context, patientID string) ([]domain.FollowupRecord, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return []domain.FollowupRecord{}, nil
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, patient_id, COALESCE(visit_id, ''), COALESCE(task_id, ''), COALESCE(project_id, ''), COALESCE(followup_type, ''), COALESCE(channel, ''), status, COALESCE(summary, ''), COALESCE(satisfaction_score, 0), COALESCE(risk_level, ''), COALESCE(DATE_FORMAT(followed_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(operator_name, ''), COALESCE(source_system, ''), created_at, updated_at FROM followup_records WHERE patient_id = ? ORDER BY followed_at DESC, created_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.FollowupRecord
	for rows.Next() {
		var item domain.FollowupRecord
		if err := rows.Scan(&item.ID, &item.PatientID, &item.VisitID, &item.TaskID, &item.ProjectID, &item.FollowupType, &item.Channel, &item.Status, &item.Summary, &item.SatisfactionScore, &item.RiskLevel, &item.FollowedAt, &item.OperatorName, &item.SourceSystem, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) InterviewExtractedFacts(ctx context.Context, patientID string) ([]domain.InterviewExtractedFact, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return []domain.InterviewExtractedFact{}, nil
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, patient_id, COALESCE(visit_id, ''), COALESCE(interview_id, ''), fact_type, fact_key, fact_label, COALESCE(fact_value, ''), COALESCE(confidence, 0), COALESCE(DATE_FORMAT(extracted_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(source_text, ''), created_at FROM interview_extracted_facts WHERE patient_id = ? ORDER BY extracted_at DESC, created_at DESC`, patientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.InterviewExtractedFact
	for rows.Next() {
		var item domain.InterviewExtractedFact
		if err := rows.Scan(&item.ID, &item.PatientID, &item.VisitID, &item.InterviewID, &item.FactType, &item.FactKey, &item.FactLabel, &item.FactValue, &item.Confidence, &item.ExtractedAt, &item.SourceText, &item.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SatisfactionIndicatorScores(ctx context.Context, projectID string) ([]domain.SatisfactionIndicatorScore, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return []domain.SatisfactionIndicatorScore{}, nil
	}
	defer db.Close()
	query := `SELECT id, project_id, indicator_id, COALESCE(patient_id, ''), COALESCE(visit_id, ''), COALESCE(department_name, ''), COALESCE(doctor_name, ''), COALESCE(nurse_name, ''), COALESCE(disease_name, ''), COALESCE(visit_type, ''), score, sample_count, COALESCE(DATE_FORMAT(score_period, '%Y-%m-%d'), ''), COALESCE(CAST(source_json AS CHAR), '{}'), created_at, updated_at FROM satisfaction_indicator_scores`
	args := []interface{}{}
	if projectID != "" {
		query += ` WHERE project_id = ?`
		args = append(args, projectID)
	}
	query += ` ORDER BY score_period DESC, department_name, indicator_id`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.SatisfactionIndicatorScore
	for rows.Next() {
		var item domain.SatisfactionIndicatorScore
		var raw string
		if err := rows.Scan(&item.ID, &item.ProjectID, &item.IndicatorID, &item.PatientID, &item.VisitID, &item.DepartmentName, &item.DoctorName, &item.NurseName, &item.DiseaseName, &item.VisitType, &item.Score, &item.SampleCount, &item.ScorePeriod, &raw, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(raw), &item.Source)
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) UpsertPatientDiagnosis(ctx context.Context, item domain.PatientDiagnosis) (domain.PatientDiagnosis, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return item, false, err
	}
	defer db.Close()
	created := item.ID == ""
	if item.ID == "" {
		item.ID = deterministicClinicalID("DX", item.PatientID, firstNonEmptyStore(item.VisitID, "-"), firstNonEmptyStore(item.DiagnosisCode, item.DiagnosisName))
	}
	if item.DiagnosisType == "" {
		item.DiagnosisType = "primary"
	}
	_, err = db.ExecContext(ctx, `INSERT INTO patient_diagnoses (id, patient_id, visit_id, diagnosis_code, diagnosis_name, diagnosis_type, diagnosed_at, department_name, doctor_name, source_system)
VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''))
ON DUPLICATE KEY UPDATE diagnosis_code=VALUES(diagnosis_code), diagnosis_name=VALUES(diagnosis_name), diagnosis_type=VALUES(diagnosis_type), diagnosed_at=VALUES(diagnosed_at), department_name=VALUES(department_name), doctor_name=VALUES(doctor_name), source_system=VALUES(source_system)`,
		item.ID, item.PatientID, item.VisitID, item.DiagnosisCode, item.DiagnosisName, item.DiagnosisType, item.DiagnosedAt, item.DepartmentName, item.DoctorName, item.SourceSystem)
	return item, created, err
}

func (s *Store) UpsertPatientHistory(ctx context.Context, item domain.PatientHistory) (domain.PatientHistory, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return item, false, err
	}
	defer db.Close()
	created := item.ID == ""
	if item.ID == "" {
		item.ID = deterministicClinicalID("HX", item.PatientID, item.HistoryType, firstNonEmptyStore(item.Title, item.Content))
	}
	if item.HistoryType == "" {
		item.HistoryType = "past"
	}
	if item.Title == "" {
		item.Title = "既往史"
	}
	_, err = db.ExecContext(ctx, `INSERT INTO patient_histories (id, patient_id, history_type, title, content, recorded_at, source_system)
VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''))
ON DUPLICATE KEY UPDATE history_type=VALUES(history_type), title=VALUES(title), content=VALUES(content), recorded_at=VALUES(recorded_at), source_system=VALUES(source_system)`,
		item.ID, item.PatientID, item.HistoryType, item.Title, item.Content, item.RecordedAt, item.SourceSystem)
	return item, created, err
}

func (s *Store) UpsertMedicationOrder(ctx context.Context, item domain.MedicationOrder) (domain.MedicationOrder, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return item, false, err
	}
	defer db.Close()
	created := item.ID == ""
	if item.ID == "" {
		item.ID = deterministicClinicalID("MED", item.PatientID, firstNonEmptyStore(item.VisitID, "-"), firstNonEmptyStore(item.OrderNo, item.PrescriptionNo, item.DrugCode, item.DrugName))
	}
	if item.Status == "" {
		item.Status = "active"
	}
	_, err = db.ExecContext(ctx, `INSERT INTO medication_orders (id, patient_id, visit_id, order_no, prescription_no, drug_code, drug_name, generic_name, specification, dosage, dosage_unit, frequency, route, start_at, end_at, days, quantity, manufacturer, doctor_name, pharmacist_name, status, adverse_reaction, compliance)
VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''))
ON DUPLICATE KEY UPDATE drug_name=VALUES(drug_name), generic_name=VALUES(generic_name), specification=VALUES(specification), dosage=VALUES(dosage), dosage_unit=VALUES(dosage_unit), frequency=VALUES(frequency), route=VALUES(route), start_at=VALUES(start_at), end_at=VALUES(end_at), days=VALUES(days), quantity=VALUES(quantity), manufacturer=VALUES(manufacturer), doctor_name=VALUES(doctor_name), pharmacist_name=VALUES(pharmacist_name), status=VALUES(status), adverse_reaction=VALUES(adverse_reaction), compliance=VALUES(compliance)`,
		item.ID, item.PatientID, item.VisitID, item.OrderNo, item.PrescriptionNo, item.DrugCode, item.DrugName, item.GenericName, item.Specification, item.Dosage, item.DosageUnit, item.Frequency, item.Route, item.StartAt, item.EndAt, item.Days, item.Quantity, item.Manufacturer, item.DoctorName, item.PharmacistName, item.Status, item.AdverseReaction, item.Compliance)
	return item, created, err
}

func (s *Store) UpsertLabReport(ctx context.Context, item domain.LabReport) (domain.LabReport, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return item, false, err
	}
	defer db.Close()
	created := item.ID == ""
	if item.ID == "" {
		item.ID = deterministicClinicalID("LAB", item.PatientID, firstNonEmptyStore(item.VisitID, "-"), firstNonEmptyStore(item.ReportNo, item.ReportName))
	}
	if item.ReportNo == "" {
		item.ReportNo = item.ID
	}
	if item.Status == "" {
		item.Status = "reported"
	}
	_, err = db.ExecContext(ctx, `INSERT INTO lab_reports (id, patient_id, visit_id, report_no, report_name, specimen, ordered_at, reported_at, department_name, doctor_name, status, source_system)
VALUES (?, ?, NULLIF(?, ''), ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''))
ON DUPLICATE KEY UPDATE patient_id=VALUES(patient_id), visit_id=VALUES(visit_id), report_name=VALUES(report_name), specimen=VALUES(specimen), ordered_at=VALUES(ordered_at), reported_at=VALUES(reported_at), department_name=VALUES(department_name), doctor_name=VALUES(doctor_name), status=VALUES(status), source_system=VALUES(source_system)`,
		item.ID, item.PatientID, item.VisitID, item.ReportNo, item.ReportName, item.Specimen, item.OrderedAt, item.ReportedAt, item.DepartmentName, item.DoctorName, item.Status, item.SourceSystem)
	if err != nil {
		return item, created, err
	}
	if err := db.QueryRowContext(ctx, `SELECT id FROM lab_reports WHERE report_no = ?`, item.ReportNo).Scan(&item.ID); err != nil {
		return item, created, err
	}
	return item, created, nil
}

func (s *Store) UpsertLabResult(ctx context.Context, item domain.LabResult) (domain.LabResult, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return item, false, err
	}
	defer db.Close()
	created := item.ID == ""
	if item.ID == "" {
		item.ID = deterministicClinicalID("LAR", item.ReportID, firstNonEmptyStore(item.ItemCode, item.ItemName))
	}
	_, err = db.ExecContext(ctx, `INSERT INTO lab_results (id, report_id, item_code, item_name, result_value, unit, reference_range, abnormal_flag, numeric_value)
VALUES (?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?)
ON DUPLICATE KEY UPDATE item_code=VALUES(item_code), item_name=VALUES(item_name), result_value=VALUES(result_value), unit=VALUES(unit), reference_range=VALUES(reference_range), abnormal_flag=VALUES(abnormal_flag), numeric_value=VALUES(numeric_value)`,
		item.ID, item.ReportID, item.ItemCode, item.ItemName, item.ResultValue, item.Unit, item.ReferenceRange, item.AbnormalFlag, item.NumericValue)
	return item, created, err
}

func (s *Store) UpsertExamReport(ctx context.Context, item domain.ExamReport) (domain.ExamReport, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return item, false, err
	}
	defer db.Close()
	created := item.ID == ""
	if item.ID == "" {
		item.ID = deterministicClinicalID("EXAM", item.PatientID, firstNonEmptyStore(item.VisitID, "-"), firstNonEmptyStore(item.ExamNo, item.ExamName))
	}
	if item.ExamNo == "" {
		item.ExamNo = item.ID
	}
	_, err = db.ExecContext(ctx, `INSERT INTO exam_reports (id, patient_id, visit_id, exam_no, exam_type, exam_name, body_part, report_conclusion, report_findings, ordered_at, reported_at, department_name, doctor_name, source_system)
VALUES (?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''))
ON DUPLICATE KEY UPDATE patient_id=VALUES(patient_id), visit_id=VALUES(visit_id), exam_type=VALUES(exam_type), exam_name=VALUES(exam_name), body_part=VALUES(body_part), report_conclusion=VALUES(report_conclusion), report_findings=VALUES(report_findings), ordered_at=VALUES(ordered_at), reported_at=VALUES(reported_at), department_name=VALUES(department_name), doctor_name=VALUES(doctor_name), source_system=VALUES(source_system)`,
		item.ID, item.PatientID, item.VisitID, item.ExamNo, item.ExamType, item.ExamName, item.BodyPart, item.ReportConclusion, item.ReportFindings, item.OrderedAt, item.ReportedAt, item.DepartmentName, item.DoctorName, item.SourceSystem)
	if err != nil {
		return item, created, err
	}
	if err := db.QueryRowContext(ctx, `SELECT id FROM exam_reports WHERE exam_no = ?`, item.ExamNo).Scan(&item.ID); err != nil {
		return item, created, err
	}
	return item, created, nil
}

func (s *Store) UpsertSurgeryRecord(ctx context.Context, item domain.SurgeryRecord) (domain.SurgeryRecord, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return item, false, err
	}
	defer db.Close()
	created := item.ID == ""
	if item.ID == "" {
		item.ID = deterministicClinicalID("SURG", item.PatientID, firstNonEmptyStore(item.VisitID, "-"), firstNonEmptyStore(item.OperationCode, item.OperationName))
	}
	_, err = db.ExecContext(ctx, `INSERT INTO surgery_records (id, patient_id, visit_id, operation_code, operation_name, operation_date, surgeon_name, anesthesia_type, operation_level, wound_grade, outcome, source_system)
VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''))
ON DUPLICATE KEY UPDATE operation_code=VALUES(operation_code), operation_name=VALUES(operation_name), operation_date=VALUES(operation_date), surgeon_name=VALUES(surgeon_name), anesthesia_type=VALUES(anesthesia_type), operation_level=VALUES(operation_level), wound_grade=VALUES(wound_grade), outcome=VALUES(outcome), source_system=VALUES(source_system)`,
		item.ID, item.PatientID, item.VisitID, item.OperationCode, item.OperationName, item.OperationDate, item.SurgeonName, item.AnesthesiaType, item.OperationLevel, item.WoundGrade, item.Outcome, item.SourceSystem)
	return item, created, err
}

func (s *Store) UpsertFollowupRecord(ctx context.Context, item domain.FollowupRecord) (domain.FollowupRecord, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return item, false, err
	}
	defer db.Close()
	created := item.ID == ""
	if item.ID == "" {
		item.ID = deterministicClinicalID("FUR", item.PatientID, firstNonEmptyStore(item.VisitID, "-"), firstNonEmptyStore(item.TaskID, item.FollowedAt, item.Summary))
	}
	if item.Status == "" {
		item.Status = "completed"
	}
	_, err = db.ExecContext(ctx, `INSERT INTO followup_records (id, patient_id, visit_id, task_id, project_id, followup_type, channel, status, summary, satisfaction_score, risk_level, followed_at, operator_name, source_system)
VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''))
ON DUPLICATE KEY UPDATE project_id=VALUES(project_id), followup_type=VALUES(followup_type), channel=VALUES(channel), status=VALUES(status), summary=VALUES(summary), satisfaction_score=VALUES(satisfaction_score), risk_level=VALUES(risk_level), followed_at=VALUES(followed_at), operator_name=VALUES(operator_name), source_system=VALUES(source_system)`,
		item.ID, item.PatientID, item.VisitID, item.TaskID, item.ProjectID, item.FollowupType, item.Channel, item.Status, item.Summary, item.SatisfactionScore, item.RiskLevel, item.FollowedAt, item.OperatorName, item.SourceSystem)
	return item, created, err
}

func (s *Store) UpsertInterviewExtractedFact(ctx context.Context, item domain.InterviewExtractedFact) (domain.InterviewExtractedFact, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return item, false, err
	}
	defer db.Close()
	created := item.ID == ""
	if item.ID == "" {
		item.ID = deterministicClinicalID("FACT", item.PatientID, firstNonEmptyStore(item.VisitID, "-"), firstNonEmptyStore(item.InterviewID, "-"), item.FactType, item.FactKey, item.FactValue)
	}
	_, err = db.ExecContext(ctx, `INSERT INTO interview_extracted_facts (id, patient_id, visit_id, interview_id, fact_type, fact_key, fact_label, fact_value, confidence, extracted_at, source_text)
VALUES (?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''))
ON DUPLICATE KEY UPDATE fact_label=VALUES(fact_label), fact_value=VALUES(fact_value), confidence=VALUES(confidence), extracted_at=VALUES(extracted_at), source_text=VALUES(source_text)`,
		item.ID, item.PatientID, item.VisitID, item.InterviewID, item.FactType, item.FactKey, item.FactLabel, item.FactValue, item.Confidence, item.ExtractedAt, item.SourceText)
	return item, created, err
}

func deterministicClinicalID(prefix string, parts ...string) string {
	seed := ""
	for _, part := range parts {
		if part != "" {
			seed += "|" + part
		}
	}
	if seed == "" {
		return uuid.NewString()
	}
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(prefix+seed)).String()
}
