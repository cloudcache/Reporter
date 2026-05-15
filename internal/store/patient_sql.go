package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) EnsurePatientTables(ctx context.Context) error {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	statements := []string{
		`CREATE TABLE IF NOT EXISTS patients (
  id CHAR(36) PRIMARY KEY,
  patient_no VARCHAR(80) NOT NULL UNIQUE,
  medical_record_no VARCHAR(80) NULL,
  name VARCHAR(120) NOT NULL,
  gender VARCHAR(20) NULL,
  birth_date DATE NULL,
  age INT NULL,
  id_card_no VARCHAR(80) NULL,
  phone VARCHAR(40) NULL,
  address TEXT NULL,
  nationality VARCHAR(80) NULL,
  ethnicity VARCHAR(80) NULL,
  marital_status VARCHAR(40) NULL,
  insurance_type VARCHAR(80) NULL,
  blood_type VARCHAR(20) NULL,
  allergies_json JSON NULL,
  emergency_contact VARCHAR(120) NULL,
  emergency_phone VARCHAR(40) NULL,
  diagnosis VARCHAR(240) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  last_visit_at DATE NULL,
  source_refs_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_patients_name (name),
  INDEX idx_patients_phone (phone),
  INDEX idx_patients_record_no (medical_record_no)
)`,
		`CREATE TABLE IF NOT EXISTS clinical_visits (
  id CHAR(36) PRIMARY KEY,
  patient_id CHAR(36) NOT NULL,
  visit_no VARCHAR(100) NOT NULL UNIQUE,
  visit_type VARCHAR(60) NULL,
  department_code VARCHAR(80) NULL,
  department_name VARCHAR(120) NULL,
  ward VARCHAR(120) NULL,
  bed_no VARCHAR(40) NULL,
  attending_doctor VARCHAR(120) NULL,
  visit_at DATETIME NULL,
  discharge_at DATETIME NULL,
  diagnosis_code VARCHAR(80) NULL,
  diagnosis_name VARCHAR(240) NULL,
  status VARCHAR(40) NOT NULL DEFAULT 'active',
  source_refs_json JSON NULL,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_clinical_visits_patient (patient_id),
  INDEX idx_clinical_visits_dept (department_name),
  INDEX idx_clinical_visits_doctor (attending_doctor)
)`,
	}
	for _, statement := range statements {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) dbPatients(ctx context.Context, keyword string) ([]domain.Patient, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	keyword = strings.TrimSpace(keyword)
	like := "%" + keyword + "%"
	query := `SELECT id, patient_no, COALESCE(medical_record_no, ''), name, COALESCE(gender, ''),
COALESCE(DATE_FORMAT(birth_date, '%Y-%m-%d'), ''), COALESCE(age, 0), COALESCE(id_card_no, ''),
COALESCE(phone, ''), COALESCE(address, ''), COALESCE(nationality, ''), COALESCE(ethnicity, ''),
COALESCE(marital_status, ''), COALESCE(insurance_type, ''), COALESCE(blood_type, ''),
COALESCE(CAST(allergies_json AS CHAR), '[]'), COALESCE(emergency_contact, ''), COALESCE(emergency_phone, ''),
COALESCE(diagnosis, ''), status, COALESCE(DATE_FORMAT(last_visit_at, '%Y-%m-%d'), ''),
COALESCE(CAST(source_refs_json AS CHAR), '{}'), created_at, updated_at
FROM patients`
	args := []interface{}{}
	if keyword != "" {
		query += ` WHERE id LIKE ? OR patient_no LIKE ? OR medical_record_no LIKE ? OR name LIKE ? OR phone LIKE ? OR diagnosis LIKE ?`
		args = append(args, like, like, like, like, like, like)
	}
	query += ` ORDER BY updated_at DESC, created_at DESC LIMIT 500`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.Patient
	for rows.Next() {
		item, err := scanPatient(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) dbPatient(ctx context.Context, id string) (domain.Patient, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.Patient{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT id, patient_no, COALESCE(medical_record_no, ''), name, COALESCE(gender, ''),
COALESCE(DATE_FORMAT(birth_date, '%Y-%m-%d'), ''), COALESCE(age, 0), COALESCE(id_card_no, ''),
COALESCE(phone, ''), COALESCE(address, ''), COALESCE(nationality, ''), COALESCE(ethnicity, ''),
COALESCE(marital_status, ''), COALESCE(insurance_type, ''), COALESCE(blood_type, ''),
COALESCE(CAST(allergies_json AS CHAR), '[]'), COALESCE(emergency_contact, ''), COALESCE(emergency_phone, ''),
COALESCE(diagnosis, ''), status, COALESCE(DATE_FORMAT(last_visit_at, '%Y-%m-%d'), ''),
COALESCE(CAST(source_refs_json AS CHAR), '{}'), created_at, updated_at
FROM patients WHERE id = ? OR patient_no = ? OR medical_record_no = ? LIMIT 1`, id, id, id)
	item, err := scanPatient(row)
	if err == sql.ErrNoRows {
		return domain.Patient{}, false, nil
	}
	if err != nil {
		return domain.Patient{}, false, err
	}
	return item, true, nil
}

func (s *Store) dbCreatePatient(ctx context.Context, patient domain.Patient) (domain.Patient, error) {
	if patient.ID == "" {
		patient.ID = uuid.NewString()
	}
	if patient.PatientNo == "" {
		patient.PatientNo = patient.ID
	}
	if patient.Name == "" {
		patient.Name = patient.PatientNo
	}
	if patient.Status == "" {
		patient.Status = "active"
	}
	saved, _, err := s.dbUpsertPatientByNo(ctx, patient)
	return saved, err
}

func (s *Store) dbUpdatePatient(ctx context.Context, id string, patch domain.Patient) (domain.Patient, error) {
	existing, ok, err := s.dbPatient(ctx, id)
	if err != nil {
		return domain.Patient{}, err
	}
	if !ok {
		return domain.Patient{}, ErrNotFound
	}
	merged := mergePatientPatch(existing, patch)
	saved, _, err := s.dbUpsertPatientByNo(ctx, merged)
	return saved, err
}

func (s *Store) dbUpsertPatientByNo(ctx context.Context, patient domain.Patient) (domain.Patient, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return patient, false, err
	}
	defer db.Close()
	created := false
	if patient.ID == "" {
		if patient.PatientNo != "" {
			var existingID string
			err := db.QueryRowContext(ctx, `SELECT id FROM patients WHERE patient_no = ?`, patient.PatientNo).Scan(&existingID)
			if err == nil {
				patient.ID = existingID
			} else if err != sql.ErrNoRows {
				return patient, false, err
			}
		}
		if patient.ID == "" {
			patient.ID = uuid.NewString()
			created = true
		}
	}
	if patient.PatientNo == "" {
		patient.PatientNo = patient.ID
	}
	if patient.Name == "" {
		patient.Name = patient.PatientNo
	}
	if patient.Status == "" {
		patient.Status = "active"
	}
	allergies, _ := json.Marshal(patient.Allergies)
	sourceRefs, _ := json.Marshal(patient.SourceRefs)
	_, err = db.ExecContext(ctx, `INSERT INTO patients (id, patient_no, medical_record_no, name, gender, birth_date, age, id_card_no, phone, address, nationality, ethnicity, marital_status, insurance_type, blood_type, allergies_json, emergency_contact, emergency_phone, diagnosis, status, last_visit_at, source_refs_json)
VALUES (?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?)
ON DUPLICATE KEY UPDATE medical_record_no=VALUES(medical_record_no), name=VALUES(name), gender=VALUES(gender), birth_date=VALUES(birth_date), age=VALUES(age), id_card_no=VALUES(id_card_no), phone=VALUES(phone), address=VALUES(address), nationality=VALUES(nationality), ethnicity=VALUES(ethnicity), marital_status=VALUES(marital_status), insurance_type=VALUES(insurance_type), blood_type=VALUES(blood_type), allergies_json=VALUES(allergies_json), emergency_contact=VALUES(emergency_contact), emergency_phone=VALUES(emergency_phone), diagnosis=VALUES(diagnosis), status=VALUES(status), last_visit_at=VALUES(last_visit_at), source_refs_json=VALUES(source_refs_json)`,
		patient.ID, patient.PatientNo, patient.MedicalRecordNo, patient.Name, patient.Gender, patient.BirthDate, patient.Age, patient.IDCardNo, patient.Phone, patient.Address, patient.Nationality, patient.Ethnicity, patient.MaritalStatus, patient.InsuranceType, patient.BloodType, string(defaultJSON(allergies, "[]")), patient.EmergencyContact, patient.EmergencyPhone, patient.Diagnosis, patient.Status, patient.LastVisitAt, string(defaultJSON(sourceRefs, "{}")))
	if err != nil {
		return patient, created, err
	}
	returned, ok, err := s.dbPatient(ctx, patient.ID)
	if err != nil || !ok {
		return patient, created, err
	}
	return returned, created, nil
}

func (s *Store) dbVisits(ctx context.Context, patientID string) ([]domain.ClinicalVisit, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT id, patient_id, visit_no, COALESCE(visit_type, ''), COALESCE(department_code, ''),
COALESCE(department_name, ''), COALESCE(ward, ''), COALESCE(bed_no, ''), COALESCE(attending_doctor, ''),
COALESCE(DATE_FORMAT(visit_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(DATE_FORMAT(discharge_at, '%Y-%m-%d %H:%i:%s'), ''),
COALESCE(diagnosis_code, ''), COALESCE(diagnosis_name, ''), status, COALESCE(CAST(source_refs_json AS CHAR), '{}'), created_at, updated_at
FROM clinical_visits`
	args := []interface{}{}
	if patientID != "" {
		query += ` WHERE patient_id = ?`
		args = append(args, patientID)
	}
	query += ` ORDER BY visit_at DESC, created_at DESC LIMIT 500`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.ClinicalVisit
	for rows.Next() {
		item, err := scanVisit(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) dbUpsertVisitByNo(ctx context.Context, visit domain.ClinicalVisit) (domain.ClinicalVisit, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return visit, false, err
	}
	defer db.Close()
	created := false
	if visit.ID == "" {
		if visit.VisitNo != "" {
			var existingID string
			err := db.QueryRowContext(ctx, `SELECT id FROM clinical_visits WHERE visit_no = ?`, visit.VisitNo).Scan(&existingID)
			if err == nil {
				visit.ID = existingID
			} else if err != sql.ErrNoRows {
				return visit, false, err
			}
		}
		if visit.ID == "" {
			visit.ID = uuid.NewString()
			created = true
		}
	}
	if visit.VisitNo == "" {
		visit.VisitNo = visit.ID
	}
	if visit.Status == "" {
		visit.Status = "active"
	}
	sourceRefs, _ := json.Marshal(visit.SourceRefs)
	_, err = db.ExecContext(ctx, `INSERT INTO clinical_visits (id, patient_id, visit_no, visit_type, department_code, department_name, ward, bed_no, attending_doctor, visit_at, discharge_at, diagnosis_code, diagnosis_name, status, source_refs_json)
VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?, ?)
ON DUPLICATE KEY UPDATE patient_id=VALUES(patient_id), visit_type=VALUES(visit_type), department_code=VALUES(department_code), department_name=VALUES(department_name), ward=VALUES(ward), bed_no=VALUES(bed_no), attending_doctor=VALUES(attending_doctor), visit_at=VALUES(visit_at), discharge_at=VALUES(discharge_at), diagnosis_code=VALUES(diagnosis_code), diagnosis_name=VALUES(diagnosis_name), status=VALUES(status), source_refs_json=VALUES(source_refs_json)`,
		visit.ID, visit.PatientID, visit.VisitNo, visit.VisitType, visit.DepartmentCode, visit.DepartmentName, visit.Ward, visit.BedNo, visit.AttendingDoctor, visit.VisitAt, visit.DischargeAt, visit.DiagnosisCode, visit.DiagnosisName, visit.Status, string(defaultJSON(sourceRefs, "{}")))
	if err != nil {
		return visit, created, err
	}
	returned, ok, err := s.dbVisit(ctx, visit.ID)
	if err != nil || !ok {
		return visit, created, err
	}
	return returned, created, nil
}

func (s *Store) dbMedicalRecords(ctx context.Context, patientID string) ([]domain.MedicalRecord, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	query := `SELECT id, patient_id, COALESCE(visit_id, ''), record_no, record_type, title, COALESCE(summary, ''),
COALESCE(chief_complaint, ''), COALESCE(present_illness, ''), COALESCE(diagnosis_code, ''),
COALESCE(diagnosis_name, ''), COALESCE(procedure_name, ''), COALESCE(study_uid, ''),
COALESCE(study_desc, ''), COALESCE(DATE_FORMAT(recorded_at, '%Y-%m-%d %H:%i:%s'), ''),
COALESCE(CAST(source_refs_json AS CHAR), '{}'), created_at, updated_at FROM medical_records`
	args := []interface{}{}
	if patientID != "" {
		query += ` WHERE patient_id = ?`
		args = append(args, patientID)
	}
	query += ` ORDER BY recorded_at DESC, created_at DESC LIMIT 500`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []domain.MedicalRecord
	for rows.Next() {
		item, err := scanMedicalRecord(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) dbMedicalRecord(ctx context.Context, id string) (domain.MedicalRecord, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.MedicalRecord{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT id, patient_id, COALESCE(visit_id, ''), record_no, record_type, title, COALESCE(summary, ''),
COALESCE(chief_complaint, ''), COALESCE(present_illness, ''), COALESCE(diagnosis_code, ''),
COALESCE(diagnosis_name, ''), COALESCE(procedure_name, ''), COALESCE(study_uid, ''),
COALESCE(study_desc, ''), COALESCE(DATE_FORMAT(recorded_at, '%Y-%m-%d %H:%i:%s'), ''),
COALESCE(CAST(source_refs_json AS CHAR), '{}'), created_at, updated_at FROM medical_records WHERE id = ? OR record_no = ? LIMIT 1`, id, id)
	item, err := scanMedicalRecord(row)
	if err == sql.ErrNoRows {
		return domain.MedicalRecord{}, false, nil
	}
	if err != nil {
		return domain.MedicalRecord{}, false, err
	}
	return item, true, nil
}

func (s *Store) dbUpsertMedicalRecordByNo(ctx context.Context, record domain.MedicalRecord) (domain.MedicalRecord, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return record, false, err
	}
	defer db.Close()
	created := false
	if record.ID == "" {
		if record.RecordNo != "" {
			var existingID string
			err := db.QueryRowContext(ctx, `SELECT id FROM medical_records WHERE record_no = ?`, record.RecordNo).Scan(&existingID)
			if err == nil {
				record.ID = existingID
			} else if err != sql.ErrNoRows {
				return record, false, err
			}
		}
		if record.ID == "" {
			record.ID = uuid.NewString()
			created = true
		}
	}
	if record.RecordNo == "" {
		record.RecordNo = record.ID
	}
	if record.RecordType == "" {
		record.RecordType = "external"
	}
	if record.Title == "" {
		record.Title = record.RecordNo
	}
	sourceRefs, _ := json.Marshal(record.SourceRefs)
	_, err = db.ExecContext(ctx, `INSERT INTO medical_records (id, patient_id, visit_id, record_no, record_type, title, summary, chief_complaint, present_illness, diagnosis_code, diagnosis_name, procedure_name, study_uid, study_desc, recorded_at, source_refs_json)
VALUES (?, ?, NULLIF(?, ''), ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), ?)
ON DUPLICATE KEY UPDATE patient_id=VALUES(patient_id), visit_id=VALUES(visit_id), record_type=VALUES(record_type), title=VALUES(title), summary=VALUES(summary), chief_complaint=VALUES(chief_complaint), present_illness=VALUES(present_illness), diagnosis_code=VALUES(diagnosis_code), diagnosis_name=VALUES(diagnosis_name), procedure_name=VALUES(procedure_name), study_uid=VALUES(study_uid), study_desc=VALUES(study_desc), recorded_at=VALUES(recorded_at), source_refs_json=VALUES(source_refs_json)`,
		record.ID, record.PatientID, record.VisitID, record.RecordNo, record.RecordType, record.Title, record.Summary, record.ChiefComplaint, record.PresentIllness, record.DiagnosisCode, record.DiagnosisName, record.ProcedureName, record.StudyUID, record.StudyDesc, record.RecordedAt, string(defaultJSON(sourceRefs, "{}")))
	if err != nil {
		return record, created, err
	}
	returned, ok, err := s.dbMedicalRecord(ctx, record.ID)
	if err != nil || !ok {
		return record, created, err
	}
	return returned, created, nil
}

func (s *Store) dbVisit(ctx context.Context, id string) (domain.ClinicalVisit, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.ClinicalVisit{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT id, patient_id, visit_no, COALESCE(visit_type, ''), COALESCE(department_code, ''),
COALESCE(department_name, ''), COALESCE(ward, ''), COALESCE(bed_no, ''), COALESCE(attending_doctor, ''),
COALESCE(DATE_FORMAT(visit_at, '%Y-%m-%d %H:%i:%s'), ''), COALESCE(DATE_FORMAT(discharge_at, '%Y-%m-%d %H:%i:%s'), ''),
COALESCE(diagnosis_code, ''), COALESCE(diagnosis_name, ''), status, COALESCE(CAST(source_refs_json AS CHAR), '{}'), created_at, updated_at
FROM clinical_visits WHERE id = ? OR visit_no = ? LIMIT 1`, id, id)
	item, err := scanVisit(row)
	if err == sql.ErrNoRows {
		return domain.ClinicalVisit{}, false, nil
	}
	if err != nil {
		return domain.ClinicalVisit{}, false, err
	}
	return item, true, nil
}

type patientScanner interface {
	Scan(dest ...interface{}) error
}

func scanPatient(row patientScanner) (domain.Patient, error) {
	var item domain.Patient
	var allergiesRaw, sourceRaw string
	if err := row.Scan(&item.ID, &item.PatientNo, &item.MedicalRecordNo, &item.Name, &item.Gender, &item.BirthDate, &item.Age, &item.IDCardNo, &item.Phone, &item.Address, &item.Nationality, &item.Ethnicity, &item.MaritalStatus, &item.InsuranceType, &item.BloodType, &allergiesRaw, &item.EmergencyContact, &item.EmergencyPhone, &item.Diagnosis, &item.Status, &item.LastVisitAt, &sourceRaw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	_ = json.Unmarshal([]byte(allergiesRaw), &item.Allergies)
	_ = json.Unmarshal([]byte(sourceRaw), &item.SourceRefs)
	return item, nil
}

func scanVisit(row patientScanner) (domain.ClinicalVisit, error) {
	var item domain.ClinicalVisit
	var sourceRaw string
	if err := row.Scan(&item.ID, &item.PatientID, &item.VisitNo, &item.VisitType, &item.DepartmentCode, &item.DepartmentName, &item.Ward, &item.BedNo, &item.AttendingDoctor, &item.VisitAt, &item.DischargeAt, &item.DiagnosisCode, &item.DiagnosisName, &item.Status, &sourceRaw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	_ = json.Unmarshal([]byte(sourceRaw), &item.SourceRefs)
	return item, nil
}

func scanMedicalRecord(row patientScanner) (domain.MedicalRecord, error) {
	var item domain.MedicalRecord
	var sourceRaw string
	if err := row.Scan(&item.ID, &item.PatientID, &item.VisitID, &item.RecordNo, &item.RecordType, &item.Title, &item.Summary, &item.ChiefComplaint, &item.PresentIllness, &item.DiagnosisCode, &item.DiagnosisName, &item.ProcedureName, &item.StudyUID, &item.StudyDesc, &item.RecordedAt, &sourceRaw, &item.CreatedAt, &item.UpdatedAt); err != nil {
		return item, err
	}
	_ = json.Unmarshal([]byte(sourceRaw), &item.SourceRefs)
	return item, nil
}

func mergePatientPatch(patient domain.Patient, patch domain.Patient) domain.Patient {
	if patch.PatientNo != "" {
		patient.PatientNo = patch.PatientNo
	}
	if patch.Name != "" {
		patient.Name = patch.Name
	}
	if patch.Gender != "" {
		patient.Gender = patch.Gender
	}
	if patch.Age != 0 {
		patient.Age = patch.Age
	}
	if patch.Phone != "" {
		patient.Phone = patch.Phone
	}
	if patch.Diagnosis != "" {
		patient.Diagnosis = patch.Diagnosis
	}
	if patch.Status != "" {
		patient.Status = patch.Status
	}
	if patch.LastVisitAt != "" {
		patient.LastVisitAt = patch.LastVisitAt
	}
	if patch.MedicalRecordNo != "" {
		patient.MedicalRecordNo = patch.MedicalRecordNo
	}
	if patch.BirthDate != "" {
		patient.BirthDate = patch.BirthDate
	}
	if patch.IDCardNo != "" {
		patient.IDCardNo = patch.IDCardNo
	}
	if patch.Address != "" {
		patient.Address = patch.Address
	}
	if patch.Nationality != "" {
		patient.Nationality = patch.Nationality
	}
	if patch.Ethnicity != "" {
		patient.Ethnicity = patch.Ethnicity
	}
	if patch.MaritalStatus != "" {
		patient.MaritalStatus = patch.MaritalStatus
	}
	if patch.InsuranceType != "" {
		patient.InsuranceType = patch.InsuranceType
	}
	if patch.BloodType != "" {
		patient.BloodType = patch.BloodType
	}
	if patch.Allergies != nil {
		patient.Allergies = patch.Allergies
	}
	if patch.EmergencyContact != "" {
		patient.EmergencyContact = patch.EmergencyContact
	}
	if patch.EmergencyPhone != "" {
		patient.EmergencyPhone = patch.EmergencyPhone
	}
	if patch.SourceRefs != nil {
		patient.SourceRefs = patch.SourceRefs
	}
	return patient
}

func defaultJSON(value []byte, fallback string) []byte {
	if len(value) == 0 || string(value) == "null" {
		return []byte(fallback)
	}
	return value
}
