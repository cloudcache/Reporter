package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"

	_ "github.com/go-sql-driver/mysql"

	"reporter/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	mu       sync.RWMutex
	dbDriver string
	dbDSN    string
}

func (s *Store) ConfigureSQL(driver, dsn string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dbDriver = firstNonEmptyStore(driver, "mysql")
	s.dbDSN = strings.TrimSpace(dsn)
}

func (s *Store) LoadIdentityFromSQL(ctx context.Context, driver, dsn string) error {
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
	if err := db.PingContext(ctx); err != nil {
		return err
	}

	users, err := loadUsers(ctx, db)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}
	return nil
}

func loadRoles(ctx context.Context, db *sql.DB) (map[string]domain.Role, error) {
	roleRows, err := db.QueryContext(ctx, `SELECT id, name, COALESCE(description, '') FROM roles`)
	if err != nil {
		return nil, err
	}
	defer roleRows.Close()
	roles := map[string]domain.Role{}
	for roleRows.Next() {
		var role domain.Role
		if err := roleRows.Scan(&role.ID, &role.Name, &role.Description); err != nil {
			return nil, err
		}
		role.Permissions = []string{}
		roles[role.ID] = role
	}
	if err := roleRows.Err(); err != nil {
		return nil, err
	}

	permissionRows, err := db.QueryContext(ctx, `
SELECT rp.role_id, p.resource, p.action
FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
`)
	if err != nil {
		return nil, err
	}
	defer permissionRows.Close()
	for permissionRows.Next() {
		var roleID, resource, action string
		if err := permissionRows.Scan(&roleID, &resource, &action); err != nil {
			return nil, err
		}
		role := roles[roleID]
		role.Permissions = append(role.Permissions, resource+":"+action)
		roles[roleID] = role
	}
	return roles, permissionRows.Err()
}

func loadUsers(ctx context.Context, db *sql.DB) (map[string]domain.User, error) {
	return queryUsers(ctx, db, "")
}

func newEmptyStore() *Store {
	return &Store{}
}

func InstallOnly() *Store {
	return newEmptyStore()
}

func Open(ctx context.Context, driver, dsn string) (*Store, error) {
	if strings.TrimSpace(dsn) == "" {
		return nil, errors.New("database dsn required")
	}
	store := newEmptyStore()
	store.ConfigureSQL(driver, dsn)
	if err := store.LoadFormLibraryFromSQL(ctx, driver, dsn); err != nil {
		return nil, err
	}
	if err := store.LoadFollowupConfigFromSQL(ctx, driver, dsn); err != nil {
		return nil, err
	}
	if err := store.EnsureEvaluationComplaintTables(ctx); err != nil {
		return nil, err
	}
	if err := store.EnsurePatientGroupTables(ctx); err != nil {
		return nil, err
	}
	if err := store.EnsurePatientTables(ctx); err != nil {
		return nil, err
	}
	if err := store.EnsureSurveyChannelTables(ctx); err != nil {
		return nil, err
	}
	if err := store.EnsureClinicalFactTables(ctx); err != nil {
		return nil, err
	}
	if err := store.EnsureReportTables(ctx); err != nil {
		return nil, err
	}
	if err := store.EnsureFormEngineTables(ctx); err != nil {
		return nil, err
	}
	if err := store.EnsureCallCenterDefaults(ctx); err != nil {
		return nil, err
	}
	if err := store.EnsureIdentityOrgTables(ctx); err != nil {
		return nil, err
	}
	if err := store.LoadIdentityFromSQL(ctx, driver, dsn); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) Seats() []domain.AgentSeat {
	seats, _ := s.SeatsStrict(context.Background())
	return seats
}

func (s *Store) CreateSeat(seat domain.AgentSeat) domain.AgentSeat {
	saved, _ := s.CreateSeatStrict(context.Background(), seat)
	return saved
}

func (s *Store) Seat(id string) (domain.AgentSeat, bool) {
	seat, ok, _ := s.SeatStrict(context.Background(), id)
	return seat, ok
}

func (s *Store) UpdateSeat(id string, patch domain.AgentSeat) (domain.AgentSeat, error) {
	return s.UpdateSeatStrict(context.Background(), id, patch)
}

func (s *Store) DeleteSeat(id string) (domain.AgentSeat, error) {
	return s.DeleteSeatStrict(context.Background(), id)
}

func (s *Store) SipEndpoints() []domain.SipEndpoint {
	endpoints, _ := s.SipEndpointsStrict(context.Background())
	return endpoints
}

func (s *Store) DefaultSipEndpoint() (domain.SipEndpoint, bool) {
	endpoint, ok, _ := s.DefaultSipEndpointStrict(context.Background())
	return endpoint, ok
}

func sipConfigEnabled(config map[string]interface{}) bool {
	if config == nil {
		return false
	}
	switch value := config["enabled"].(type) {
	case bool:
		return value
	case string:
		value = strings.TrimSpace(strings.ToLower(value))
		return value == "true" || value == "1" || value == "yes"
	default:
		return false
	}
}

func (s *Store) SipEndpoint(id string) (domain.SipEndpoint, bool) {
	endpoint, ok, _ := s.SipEndpointStrict(context.Background(), id)
	return endpoint, ok
}

func (s *Store) CreateSipEndpoint(endpoint domain.SipEndpoint) domain.SipEndpoint {
	saved, _ := s.CreateSipEndpointStrict(context.Background(), endpoint)
	return saved
}

func (s *Store) UpdateSipEndpoint(id string, patch domain.SipEndpoint) (domain.SipEndpoint, error) {
	return s.UpdateSipEndpointStrict(context.Background(), id, patch)
}

func (s *Store) DeleteSipEndpoint(id string) (domain.SipEndpoint, error) {
	return s.DeleteSipEndpointStrict(context.Background(), id)
}

func (s *Store) StorageConfigs() []domain.StorageConfig {
	configs, _ := s.StorageConfigsStrict(context.Background())
	return configs
}

func (s *Store) StorageConfig(id string) (domain.StorageConfig, bool) {
	config, ok, _ := s.StorageConfigStrict(context.Background(), id)
	return config, ok
}

func (s *Store) CreateStorageConfig(config domain.StorageConfig) domain.StorageConfig {
	saved, _ := s.CreateStorageConfigStrict(context.Background(), config)
	return saved
}

func (s *Store) UpdateStorageConfig(id string, patch domain.StorageConfig) (domain.StorageConfig, error) {
	return s.UpdateStorageConfigStrict(context.Background(), id, patch)
}

func (s *Store) DeleteStorageConfig(id string) (domain.StorageConfig, error) {
	return s.DeleteStorageConfigStrict(context.Background(), id)
}

func (s *Store) RecordingConfigs() []domain.RecordingConfig {
	configs, _ := s.RecordingConfigsStrict(context.Background())
	return configs
}

func (s *Store) RecordingConfig(id string) (domain.RecordingConfig, bool) {
	config, ok, _ := s.RecordingConfigStrict(context.Background(), id)
	return config, ok
}

func (s *Store) DefaultRecordingConfig() (domain.RecordingConfig, bool) {
	config, ok, _ := s.DefaultRecordingConfigStrict(context.Background())
	return config, ok
}

func (s *Store) CreateRecordingConfig(config domain.RecordingConfig) domain.RecordingConfig {
	saved, _ := s.CreateRecordingConfigStrict(context.Background(), config)
	return saved
}

func (s *Store) UpdateRecordingConfig(id string, patch domain.RecordingConfig) (domain.RecordingConfig, error) {
	return s.UpdateRecordingConfigStrict(context.Background(), id, patch)
}

func (s *Store) DeleteRecordingConfig(id string) (domain.RecordingConfig, error) {
	return s.DeleteRecordingConfigStrict(context.Background(), id)
}

func (s *Store) Calls() []domain.CallSession {
	calls, _ := s.CallsStrict(context.Background())
	return calls
}

func (s *Store) CreateCall(call domain.CallSession) domain.CallSession {
	saved, _ := s.CreateCallStrict(context.Background(), call)
	return saved
}

func (s *Store) UpdateCall(id string, patch domain.CallSession) (domain.CallSession, error) {
	return s.UpdateCallStrict(context.Background(), id, patch)
}

func (s *Store) Recordings() []domain.Recording {
	recordings, _ := s.RecordingsStrict(context.Background())
	return recordings
}

func (s *Store) CreateRecording(recording domain.Recording) domain.Recording {
	saved, _ := s.CreateRecordingStrict(context.Background(), recording)
	return saved
}

func (s *Store) ModelProviders() []domain.ModelProvider {
	models, _ := s.ModelProvidersStrict(context.Background())
	return models
}

func (s *Store) ModelProvider(id string) (domain.ModelProvider, bool) {
	model, ok, _ := s.ModelProviderStrict(context.Background(), id)
	return model, ok
}

func (s *Store) CreateModelProvider(provider domain.ModelProvider) domain.ModelProvider {
	saved, _ := s.CreateModelProviderStrict(context.Background(), provider)
	return saved
}

func (s *Store) UpdateModelProvider(id string, patch domain.ModelProvider) (domain.ModelProvider, error) {
	return s.UpdateModelProviderStrict(context.Background(), id, patch)
}

func (s *Store) DeleteModelProvider(id string) (domain.ModelProvider, error) {
	return s.DeleteModelProviderStrict(context.Background(), id)
}

func (s *Store) Analyses() []domain.CallAnalysis {
	analyses, _ := s.AnalysesStrict(context.Background())
	return analyses
}

func (s *Store) RealtimeAssistSessions() []domain.RealtimeAssistSession {
	sessions, _ := s.RealtimeAssistSessionsStrict(context.Background())
	return sessions
}

func (s *Store) CreateRealtimeAssistSession(session domain.RealtimeAssistSession) domain.RealtimeAssistSession {
	saved, _ := s.CreateRealtimeAssistSessionStrict(context.Background(), session)
	return saved
}

func (s *Store) AddRealtimeTranscript(id string, transcript domain.RealtimeTranscript, formPatch map[string]interface{}) (domain.RealtimeAssistSession, error) {
	return s.AddRealtimeTranscriptStrict(context.Background(), id, transcript, formPatch)
}

func (s *Store) OfflineAnalysisJobs() []domain.OfflineAnalysisJob {
	jobs, _ := s.OfflineAnalysisJobsStrict(context.Background())
	return jobs
}

func (s *Store) CreateOfflineAnalysisJob(job domain.OfflineAnalysisJob) domain.OfflineAnalysisJob {
	saved, _ := s.CreateOfflineAnalysisJobStrict(context.Background(), job)
	return saved
}

func (s *Store) Interviews() []domain.InterviewSession {
	interviews, _ := s.InterviewsStrict(context.Background())
	return interviews
}

func (s *Store) CreateInterview(interview domain.InterviewSession) domain.InterviewSession {
	saved, _ := s.CreateInterviewStrict(context.Background(), interview)
	return saved
}

func (s *Store) UserByUsername(username string) (domain.User, bool) {
	if s.dbDSN != "" {
		user, ok, err := s.dbUserByUsername(context.Background(), username)
		if err != nil {
			return domain.User{}, false
		}
		return user, ok
	}
	return domain.User{}, false
}

func (s *Store) UserByID(id string) (domain.User, bool) {
	if s.dbDSN != "" {
		user, ok, err := s.dbUserByID(context.Background(), id)
		if err != nil {
			return domain.User{}, false
		}
		return user, ok
	}
	return domain.User{}, false
}

func (s *Store) Users() []domain.User {
	users, _ := s.UsersStrict(context.Background())
	return users
}

func (s *Store) UsersStrict(ctx context.Context) ([]domain.User, error) {
	if s.dbDSN != "" {
		return s.dbUsers(ctx)
	}
	return nil, errors.New("database dsn required")
}

func (s *Store) CreateUser(user domain.User) domain.User {
	created, _ := s.CreateUserStrict(context.Background(), user)
	return created
}

func (s *Store) CreateUserStrict(ctx context.Context, user domain.User) (domain.User, error) {
	if s.dbDSN != "" {
		return s.dbCreateUser(ctx, user)
	}
	return domain.User{}, errors.New("database dsn required")
}

func (s *Store) UpdateUser(id string, patch domain.User) (domain.User, error) {
	if s.dbDSN != "" {
		return s.dbUpdateUser(context.Background(), id, patch)
	}
	return domain.User{}, errors.New("database dsn required")
}

func (s *Store) DeleteUser(id string) (domain.User, error) {
	if s.dbDSN != "" {
		return s.dbDeleteUser(context.Background(), id)
	}
	return domain.User{}, errors.New("database dsn required")
}

func (s *Store) Patients(keyword string) []domain.Patient {
	patients, _ := s.PatientsStrict(context.Background(), keyword)
	return patients
}

func (s *Store) PatientsStrict(ctx context.Context, keyword string) ([]domain.Patient, error) {
	return s.dbPatients(ctx, keyword)
}

func (s *Store) Patient(id string) (domain.Patient, bool) {
	patient, ok, _ := s.PatientStrict(context.Background(), id)
	return patient, ok
}

func (s *Store) PatientStrict(ctx context.Context, id string) (domain.Patient, bool, error) {
	return s.dbPatient(ctx, id)
}

func (s *Store) CreatePatient(patient domain.Patient) domain.Patient {
	created, err := s.CreatePatientStrict(context.Background(), patient)
	if err != nil {
		return domain.Patient{}
	}
	return created
}

func (s *Store) CreatePatientStrict(ctx context.Context, patient domain.Patient) (domain.Patient, error) {
	return s.dbCreatePatient(ctx, patient)
}

func (s *Store) UpdatePatient(id string, patch domain.Patient) (domain.Patient, error) {
	return s.dbUpdatePatient(context.Background(), id, patch)
}

func (s *Store) UpsertPatientByNo(patient domain.Patient) (domain.Patient, bool) {
	saved, created, err := s.UpsertPatientByNoStrict(context.Background(), patient)
	if err != nil {
		return domain.Patient{}, false
	}
	return saved, created
}

func (s *Store) UpsertPatientByNoStrict(ctx context.Context, patient domain.Patient) (domain.Patient, bool, error) {
	return s.dbUpsertPatientByNo(ctx, patient)
}

func (s *Store) Visits(patientID string) []domain.ClinicalVisit {
	visits, _ := s.VisitsStrict(context.Background(), patientID)
	return visits
}

func (s *Store) VisitsStrict(ctx context.Context, patientID string) ([]domain.ClinicalVisit, error) {
	return s.dbVisits(ctx, patientID)
}

func (s *Store) UpsertVisitByNo(visit domain.ClinicalVisit) (domain.ClinicalVisit, bool) {
	saved, created, err := s.UpsertVisitByNoStrict(context.Background(), visit)
	if err != nil {
		return domain.ClinicalVisit{}, false
	}
	return saved, created
}

func (s *Store) UpsertVisitByNoStrict(ctx context.Context, visit domain.ClinicalVisit) (domain.ClinicalVisit, bool, error) {
	return s.dbUpsertVisitByNo(ctx, visit)
}

func (s *Store) MedicalRecords(patientID string) []domain.MedicalRecord {
	records, _ := s.MedicalRecordsStrict(context.Background(), patientID)
	return records
}

func (s *Store) MedicalRecordsStrict(ctx context.Context, patientID string) ([]domain.MedicalRecord, error) {
	return s.dbMedicalRecords(ctx, patientID)
}

func (s *Store) UpsertMedicalRecordByNo(record domain.MedicalRecord) (domain.MedicalRecord, bool) {
	saved, created, err := s.UpsertMedicalRecordByNoStrict(context.Background(), record)
	if err != nil {
		return domain.MedicalRecord{}, false
	}
	return saved, created
}

func (s *Store) UpsertMedicalRecordByNoStrict(ctx context.Context, record domain.MedicalRecord) (domain.MedicalRecord, bool, error) {
	return s.dbUpsertMedicalRecordByNo(ctx, record)
}

func (s *Store) Datasets(keyword string) []domain.Dataset {
	datasets, _ := s.DatasetsStrict(context.Background(), keyword)
	return datasets
}

func (s *Store) Dataset(id string) (domain.Dataset, bool) {
	dataset, ok, _ := s.DatasetStrict(context.Background(), id)
	return dataset, ok
}

func (s *Store) CreateDataset(dataset domain.Dataset) domain.Dataset {
	saved, _ := s.CreateDatasetStrict(context.Background(), dataset)
	return saved
}

func (s *Store) UpdateDataset(id string, patch domain.Dataset) (domain.Dataset, error) {
	return s.UpdateDatasetStrict(context.Background(), id, patch)
}

func (s *Store) DeleteDataset(id string) (domain.Dataset, error) {
	return s.DeleteDatasetStrict(context.Background(), id)
}

func (s *Store) Roles() []domain.Role {
	roles, _ := s.RolesStrict(context.Background())
	return roles
}

func (s *Store) CreateRole(role domain.Role) domain.Role {
	saved, _ := s.CreateRoleStrict(context.Background(), role)
	return saved
}

func (s *Store) UpdateRolePermissions(id string, permissions []string) (domain.Role, error) {
	return s.UpdateRolePermissionsStrict(context.Background(), id, permissions)
}

func (s *Store) SaveAudit(log domain.AuditLog) {
	_, _ = s.SaveAuditStrict(context.Background(), log)
}

func (s *Store) AuditLogs() []domain.AuditLog {
	logs, _ := s.AuditLogsStrict(context.Background())
	return logs
}

func (s *Store) Forms() []domain.Form {
	forms, err := s.FormsStrict(context.Background())
	if err != nil {
		return nil
	}
	return forms
}

func (s *Store) FormsStrict(ctx context.Context) ([]domain.Form, error) {
	return s.formsFromSQL(ctx)
}

func (s *Store) CreateForm(form domain.Form) domain.Form {
	saved, err := s.CreateFormStrict(context.Background(), form)
	if err != nil {
		return domain.Form{}
	}
	return saved
}

func (s *Store) CreateFormStrict(ctx context.Context, form domain.Form) (domain.Form, error) {
	return s.createFormInSQL(ctx, form)
}

func (s *Store) CreateFormVersion(formID, actor string, schema []domain.FormComponent) (domain.FormVersion, error) {
	return s.CreateFormVersionStrict(context.Background(), formID, actor, schema)
}

func (s *Store) CreateFormVersionStrict(ctx context.Context, formID, actor string, schema []domain.FormComponent) (domain.FormVersion, error) {
	return s.createFormVersionInSQL(ctx, formID, actor, schema)
}

func (s *Store) PublishForm(formID string) (domain.Form, error) {
	return s.PublishFormStrict(context.Background(), formID)
}

func (s *Store) PublishFormStrict(ctx context.Context, formID string) (domain.Form, error) {
	return s.publishFormInSQL(ctx, formID)
}

func (s *Store) CreateSubmission(submission domain.Submission) (domain.Submission, error) {
	return s.CreateSubmissionStrict(context.Background(), submission)
}

func (s *Store) CreateSubmissionStrict(ctx context.Context, submission domain.Submission) (domain.Submission, error) {
	return s.createSubmissionInSQL(ctx, submission)
}

func (s *Store) SubmissionsByForm(formID string) []domain.Submission {
	submissions, err := s.SubmissionsByFormStrict(context.Background(), formID)
	if err != nil {
		return nil
	}
	return submissions
}

func (s *Store) SubmissionsByFormStrict(ctx context.Context, formID string) ([]domain.Submission, error) {
	return s.submissionsByFormFromSQL(ctx, formID)
}

func (s *Store) Submission(id string) (domain.Submission, bool) {
	submission, ok, err := s.SubmissionStrict(context.Background(), id)
	if err != nil {
		return domain.Submission{}, false
	}
	return submission, ok
}

func (s *Store) SubmissionStrict(ctx context.Context, id string) (domain.Submission, bool, error) {
	return s.submissionFromSQL(ctx, id)
}

func (s *Store) DataSources() []domain.DataSource {
	sources, err := s.DataSourcesStrict(context.Background())
	if err != nil {
		return nil
	}
	return sources
}

func (s *Store) DataSourcesStrict(ctx context.Context) ([]domain.DataSource, error) {
	return s.dataSourcesFromSQL(ctx)
}

func (s *Store) CreateDataSource(source domain.DataSource) domain.DataSource {
	saved, err := s.CreateDataSourceStrict(context.Background(), source)
	if err != nil {
		return domain.DataSource{}
	}
	return saved
}

func (s *Store) CreateDataSourceStrict(ctx context.Context, source domain.DataSource) (domain.DataSource, error) {
	return s.createDataSourceInSQL(ctx, source)
}

func (s *Store) DataSource(id string) (domain.DataSource, bool) {
	source, ok, err := s.DataSourceStrict(context.Background(), id)
	if err != nil {
		return domain.DataSource{}, false
	}
	return source, ok
}

func (s *Store) DataSourceStrict(ctx context.Context, id string) (domain.DataSource, bool, error) {
	return s.dataSourceFromSQL(ctx, id)
}

func (s *Store) UpdateDataSource(id string, patch domain.DataSource) (domain.DataSource, error) {
	return s.UpdateDataSourceStrict(context.Background(), id, patch)
}

func (s *Store) UpdateDataSourceStrict(ctx context.Context, id string, patch domain.DataSource) (domain.DataSource, error) {
	return s.updateDataSourceInSQL(ctx, id, patch)
}

func (s *Store) DeleteDataSource(id string) (domain.DataSource, error) {
	return s.DeleteDataSourceStrict(context.Background(), id)
}

func (s *Store) DeleteDataSourceStrict(ctx context.Context, id string) (domain.DataSource, error) {
	return s.deleteDataSourceInSQL(ctx, id)
}

func (s *Store) Reports() []domain.Report {
	reports, err := s.ReportDefinitions(context.Background())
	if err != nil {
		return nil
	}
	return reports
}

func (s *Store) Report(id string) (domain.Report, bool) {
	report, err := s.ReportDefinition(context.Background(), id)
	if err != nil {
		return domain.Report{}, false
	}
	return report, true
}

func (s *Store) CreateReport(report domain.Report) domain.Report {
	saved, err := s.CreateReportDefinition(context.Background(), report)
	if err != nil {
		return domain.Report{}
	}
	return saved
}

func (s *Store) UpdateReport(id string, patch domain.Report) (domain.Report, error) {
	return s.UpdateReportDefinition(context.Background(), id, patch)
}

func (s *Store) AddReportWidget(reportID string, widget domain.ReportWidget) (domain.ReportWidget, error) {
	return s.AddReportDefinitionWidget(context.Background(), reportID, widget)
}

func (s *Store) QueryReport(reportID string) (map[string]interface{}, error) {
	return s.QueryReportData(context.Background(), reportID, "", domain.ReportQueryFilters{})
}
