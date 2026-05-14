package store

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"reporter/internal/auth"
	"reporter/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	mu           sync.RWMutex
	dbDriver     string
	dbDSN        string
	users        map[string]domain.User
	roles        map[string]domain.Role
	patients     map[string]domain.Patient
	visits       map[string]domain.ClinicalVisit
	records      map[string]domain.MedicalRecord
	datasets     map[string]domain.Dataset
	departments  map[string]domain.Department
	dictionaries map[string]domain.Dictionary
	followPlans  map[string]domain.FollowupPlan
	followTasks  map[string]domain.FollowupTask
	forms        map[string]domain.Form
	formLibrary  []domain.FormLibraryItem
	submissions  map[string]domain.Submission
	dataSources  map[string]domain.DataSource
	reports      map[string]domain.Report
	seats        map[string]domain.AgentSeat
	sip          map[string]domain.SipEndpoint
	storageCfg   map[string]domain.StorageConfig
	recordingCfg map[string]domain.RecordingConfig
	calls        map[string]domain.CallSession
	recordings   map[string]domain.Recording
	models       map[string]domain.ModelProvider
	realtime     map[string]domain.RealtimeAssistSession
	offlineJobs  map[string]domain.OfflineAnalysisJob
	analyses     map[string]domain.CallAnalysis
	interviews   map[string]domain.InterviewSession
	auditLogs    []domain.AuditLog
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

	roles, err := loadRoles(ctx, db)
	if err != nil {
		return err
	}
	users, err := loadUsers(ctx, db)
	if err != nil {
		return err
	}
	if len(users) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.roles = roles
	s.users = users
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
	userRows, err := db.QueryContext(ctx, `SELECT id, username, display_name, password_hash, created_at, updated_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer userRows.Close()
	users := map[string]domain.User{}
	for userRows.Next() {
		var user domain.User
		if err := userRows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		user.Roles = []string{}
		users[user.ID] = user
	}
	if err := userRows.Err(); err != nil {
		return nil, err
	}

	roleRows, err := db.QueryContext(ctx, `SELECT user_id, role_id FROM user_roles`)
	if err != nil {
		return nil, err
	}
	defer roleRows.Close()
	for roleRows.Next() {
		var userID, roleID string
		if err := roleRows.Scan(&userID, &roleID); err != nil {
			return nil, err
		}
		user := users[userID]
		user.Roles = append(user.Roles, roleID)
		users[userID] = user
	}
	return users, roleRows.Err()
}

func newEmptyStore() *Store {
	return &Store{
		users:        map[string]domain.User{},
		roles:        map[string]domain.Role{},
		patients:     map[string]domain.Patient{},
		visits:       map[string]domain.ClinicalVisit{},
		records:      map[string]domain.MedicalRecord{},
		datasets:     map[string]domain.Dataset{},
		departments:  map[string]domain.Department{},
		dictionaries: map[string]domain.Dictionary{},
		followPlans:  map[string]domain.FollowupPlan{},
		followTasks:  map[string]domain.FollowupTask{},
		forms:        map[string]domain.Form{},
		formLibrary:  []domain.FormLibraryItem{},
		submissions:  map[string]domain.Submission{},
		dataSources:  map[string]domain.DataSource{},
		reports:      map[string]domain.Report{},
		seats:        map[string]domain.AgentSeat{},
		sip:          map[string]domain.SipEndpoint{},
		storageCfg:   map[string]domain.StorageConfig{},
		recordingCfg: map[string]domain.RecordingConfig{},
		calls:        map[string]domain.CallSession{},
		recordings:   map[string]domain.Recording{},
		models:       map[string]domain.ModelProvider{},
		realtime:     map[string]domain.RealtimeAssistSession{},
		offlineJobs:  map[string]domain.OfflineAnalysisJob{},
		analyses:     map[string]domain.CallAnalysis{},
		interviews:   map[string]domain.InterviewSession{},
		auditLogs:    []domain.AuditLog{},
	}
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
	if err := store.LoadIdentityFromSQL(ctx, driver, dsn); err != nil {
		return nil, err
	}
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
	return store, nil
}

func NewTestStore() *Store {
	adminHash, _ := auth.HashPassword("admin123")
	userHash, _ := auth.HashPassword("user123")
	now := time.Now().UTC()
	store := newEmptyStore()
	store.formLibrary = DefaultFormLibrary()
	store.roles["admin"] = domain.Role{ID: "admin", Name: "系统管理员", Description: "拥有平台全部管理权限", Permissions: []string{"*:*"}}
	store.roles["doctor"] = domain.Role{ID: "doctor", Name: "医生", Description: "查看患者档案、制定随访方案、处理异常结果", Permissions: []string{"/api/v1/patients:read", "/api/v1/forms:read", "/api/v1/followup:*", "/api/v1/reports:read", "/api/v1/complaints:read", "/api/v1/complaints:update"}}
	store.roles["nurse"] = domain.Role{ID: "nurse", Name: "护士", Description: "执行护理随访、查看授权患者档案和处理随访记录", Permissions: []string{"/api/v1/patients:read", "/api/v1/followup:read", "/api/v1/followup:update", "/api/v1/forms:read", "/api/v1/complaints:read", "/api/v1/complaints:create"}}
	store.roles["analyst"] = domain.Role{ID: "analyst", Name: "数据分析员", Description: "可管理表单、报表并查看数据源", Permissions: []string{"/api/v1/forms:*", "/api/v1/reports:*", "/api/v1/data-sources:read", "/api/v1/complaints:read"}}
	store.roles["agent"] = domain.Role{ID: "agent", Name: "随访员/调查员", Description: "可查看患者并执行电话随访、问卷调查", Permissions: []string{"/api/v1/patients:read", "/api/v1/followup:read", "/api/v1/followup:update", "/api/v1/call-center:read", "/api/v1/call-center:create", "/api/v1/call-center:update", "/api/v1/complaints:read", "/api/v1/complaints:create"}}
	store.users["1"] = domain.User{ID: "1", Username: "admin", DisplayName: "管理员", PasswordHash: adminHash, Roles: []string{"admin"}, CreatedAt: now, UpdatedAt: now}
	store.users["2"] = domain.User{ID: "2", Username: "user", DisplayName: "普通用户", PasswordHash: userHash, Roles: []string{"analyst", "agent"}, CreatedAt: now, UpdatedAt: now}
	store.patients["P001"] = domain.Patient{ID: "P001", PatientNo: "MZ20260501001", Name: "张三", Gender: "男", Age: 58, Phone: "13800010001", Diagnosis: "高血压", Status: "active", LastVisitAt: "2026-05-10", CreatedAt: now, UpdatedAt: now}
	store.patients["P002"] = domain.Patient{ID: "P002", PatientNo: "MZ20260502008", Name: "李四", Gender: "女", Age: 63, Phone: "13800010002", Diagnosis: "2型糖尿病", Status: "follow_up", LastVisitAt: "2026-05-11", CreatedAt: now, UpdatedAt: now}
	store.patients["P003"] = domain.Patient{ID: "P003", PatientNo: "ZY20260503012", Name: "王五", Gender: "男", Age: 46, Phone: "13800010003", Diagnosis: "术后恢复", Status: "inactive", LastVisitAt: "2026-05-12", CreatedAt: now, UpdatedAt: now}
	store.visits["V001"] = domain.ClinicalVisit{ID: "V001", PatientID: "P001", VisitNo: "MZ20260501001", VisitType: "outpatient", DepartmentCode: "CARD", DepartmentName: "心内科", AttendingDoctor: "王医生", VisitAt: "2026-05-10 09:30", DiagnosisCode: "I10", DiagnosisName: "高血压", Status: "active", CreatedAt: now, UpdatedAt: now}
	store.records["MR001"] = domain.MedicalRecord{ID: "MR001", PatientID: "P001", VisitID: "V001", RecordNo: "MR20260501001", RecordType: "outpatient_note", Title: "门诊病历", ChiefComplaint: "头晕 3 天", DiagnosisCode: "I10", DiagnosisName: "高血压", RecordedAt: "2026-05-10 10:00", CreatedAt: now, UpdatedAt: now}
	store.datasets["DS001"] = domain.Dataset{ID: "DS001", Name: "高血压随访研究", Description: "高血压患者长期随访数据采集", Owner: "心内科", RecordCount: 1250, FormCount: 5, Status: "active", CreatedAt: now, UpdatedAt: now}
	store.datasets["DS002"] = domain.Dataset{ID: "DS002", Name: "糖尿病管理研究", Description: "2型糖尿病患者血糖管理跟踪", Owner: "内分泌科", RecordCount: 890, FormCount: 4, Status: "active", CreatedAt: now, UpdatedAt: now}
	store.datasets["DS003"] = domain.Dataset{ID: "DS003", Name: "心血管疾病筛查", Description: "心血管疾病高危人群筛查数据", Owner: "体检中心", RecordCount: 2100, FormCount: 6, Status: "archived", CreatedAt: now, UpdatedAt: now}
	store.departments["DEPT-CARD"] = domain.Department{ID: "DEPT-CARD", Code: "CARD", Name: "心内科", Kind: "clinical", Status: "active", CreatedAt: now, UpdatedAt: now}
	store.departments["DEPT-ENDO"] = domain.Department{ID: "DEPT-ENDO", Code: "ENDO", Name: "内分泌科", Kind: "clinical", Status: "active", CreatedAt: now, UpdatedAt: now}
	for _, dictionary := range DefaultDictionaries() {
		dictionary.CreatedAt = now
		dictionary.UpdatedAt = now
		store.dictionaries[dictionary.ID] = dictionary
	}
	store.followPlans["PLAN-HTN"] = domain.FollowupPlan{ID: "PLAN-HTN", Name: "高血压慢病随访", Scenario: "慢病", DiseaseCode: "I10", DepartmentID: "DEPT-CARD", FormTemplateID: "hypertension-follow-up", TriggerType: "定期", TriggerOffset: 30, Channel: "phone", AssigneeRole: "agent", Status: "active", Rules: map[string]interface{}{"ageMin": 45, "diagnosis": "高血压"}, CreatedAt: now, UpdatedAt: now}
	store.followPlans["PLAN-DISCHARGE"] = domain.FollowupPlan{ID: "PLAN-DISCHARGE", Name: "出院后 7 日随访", Scenario: "随访", FormTemplateID: "discharge-follow-up", TriggerType: "出院后", TriggerOffset: 7, Channel: "phone", AssigneeRole: "nurse", Status: "active", CreatedAt: now, UpdatedAt: now}
	store.followTasks["TASK-001"] = domain.FollowupTask{ID: "TASK-001", PlanID: "PLAN-HTN", PatientID: "P001", PatientName: "张三", PatientPhone: "13800010001", FormTemplateID: "hypertension-follow-up", AssigneeID: "1", AssigneeName: "管理员", Role: "agent", Channel: "phone", Status: "pending", Priority: "high", DueAt: "2026-05-15", LastEvent: "系统按高血压方案生成任务", CreatedAt: now, UpdatedAt: now}
	store.reports["RP001"] = domain.Report{
		ID:          "RP001",
		Name:        "随访完成情况月报",
		Description: "按月统计随访提交量、完成量和完成率",
		Widgets: []domain.ReportWidget{
			{ID: "RW001", ReportID: "RP001", Type: "bar", Title: "月度随访提交量", DataSource: "survey-dict", CreatedAt: now},
			{ID: "RW002", ReportID: "RP001", Type: "table", Title: "随访明细", DataSource: "survey-dict", CreatedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.reports["RP002"] = domain.Report{
		ID:          "RP002",
		Name:        "患者满意度分析",
		Description: "按科室统计满意度、推荐意愿和反馈量",
		Widgets: []domain.ReportWidget{
			{ID: "RW003", ReportID: "RP002", Type: "bar", Title: "科室满意度", DataSource: "survey-dict", CreatedAt: now},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.seats["SEAT001"] = domain.AgentSeat{ID: "SEAT001", UserID: "1", Name: "随访坐席 A", Extension: "8001", SipURI: "sip:8001@call.example.local", Status: "available", Skills: []string{"随访", "满意度"}, CreatedAt: now, UpdatedAt: now}
	store.seats["SEAT002"] = domain.AgentSeat{ID: "SEAT002", UserID: "2", Name: "质控坐席 B", Extension: "8002", SipURI: "sip:8002@call.example.local", Status: "busy", Skills: []string{"术后", "慢病"}, CurrentCall: "CALL001", CreatedAt: now, UpdatedAt: now}
	store.sip["SIP001"] = domain.SipEndpoint{ID: "SIP001", Name: "院内 WebRTC SIP 网关", WSSURL: "wss://pbx.example.local/ws", Domain: "call.example.local", Proxy: "sip:pbx.example.local;transport=wss", Config: map[string]interface{}{"enabled": false, "webrtc": true, "transport": "udp", "bindHost": "0.0.0.0", "trunkUri": "sip:{phone}@carrier.example.local"}, CreatedAt: now, UpdatedAt: now}
	store.storageCfg["STOR001"] = domain.StorageConfig{ID: "STOR001", Name: "本地录音存储", Kind: "local", BasePath: "data/recordings", Config: map[string]interface{}{"pathStrategy": "yyyy/mm/dd"}, CreatedAt: now, UpdatedAt: now}
	store.recordingCfg["REC-CFG-001"] = domain.RecordingConfig{ID: "REC-CFG-001", Name: "默认通话录音策略", Mode: "server", StorageConfigID: "STOR001", Format: "wav", RetentionDays: 365, AutoStart: true, AutoStop: true, Config: map[string]interface{}{"source": "pbx_or_diago"}, CreatedAt: now, UpdatedAt: now}
	store.calls["CALL001"] = domain.CallSession{ID: "CALL001", SeatID: "SEAT002", PatientID: "P001", Direction: "outbound", PhoneNumber: "13800010001", Status: "recorded", StartedAt: now.Add(-18 * time.Minute), EndedAt: now.Add(-8 * time.Minute), RecordingID: "REC001", AnalysisID: "AN001", InterviewForm: "outpatient-satisfaction"}
	store.recordings["REC001"] = domain.Recording{ID: "REC001", CallID: "CALL001", StorageURI: "s3://reporter-recordings/2026/05/CALL001.wav", Duration: 602, Filename: "CALL001.wav", MimeType: "audio/wav", SizeBytes: 1248000, Source: "pbx_siprec", Backend: "s3", ObjectName: "2026/05/CALL001.wav", Status: "ready", CreatedAt: now.Add(-8 * time.Minute)}
	store.models["LLM001"] = domain.ModelProvider{ID: "LLM001", Name: "院内大模型网关", Kind: "openai-compatible", Mode: "offline", Endpoint: "https://llm.example.local/v1", Model: "medical-call-analyzer", CredentialRef: "secret://llm/primary", Config: map[string]interface{}{"supports_audio": true, "supports_json_schema": true, "audio_analysis": true}, CreatedAt: now, UpdatedAt: now}
	store.models["LLM002"] = domain.ModelProvider{ID: "LLM002", Name: "实时语音识别与表单回填", Kind: "realtime-asr", Mode: "realtime", Endpoint: "wss://llm.example.local/realtime", Model: "medical-realtime-asr", CredentialRef: "secret://llm/realtime", Config: map[string]interface{}{"partial_transcript": true, "form_autofill": true}, CreatedAt: now, UpdatedAt: now}
	store.realtime["RT001"] = domain.RealtimeAssistSession{ID: "RT001", CallID: "CALL001", PatientID: "P001", FormID: "outpatient-satisfaction", ProviderID: "LLM002", Status: "active", Transcript: []domain.RealtimeTranscript{{Speaker: "patient", Text: "候诊时间有点久，但是医生解释得很清楚。", IsFinal: true, CreatedAt: now.Add(-10 * time.Minute)}}, FormDraft: map[string]interface{}{"waiting_time_feedback": "候诊时间较久", "doctor_communication": "满意"}, LastSuggestion: "可追问候诊时间具体区间。", CreatedAt: now.Add(-12 * time.Minute), UpdatedAt: now.Add(-10 * time.Minute)}
	store.offlineJobs["JOB001"] = domain.OfflineAnalysisJob{ID: "JOB001", CallID: "CALL001", RecordingID: "REC001", ProviderID: "LLM001", Status: "completed", Result: map[string]interface{}{"emotion": "焦虑但配合", "true_satisfaction": 3.8}, CreatedAt: now.Add(-8 * time.Minute), UpdatedAt: now.Add(-7 * time.Minute)}
	store.analyses["AN001"] = domain.CallAnalysis{ID: "AN001", CallID: "CALL001", ProviderID: "LLM001", PatientEmotion: "焦虑但配合", TrueSatisfaction: 3.8, RiskLevel: "medium", PatientStatus: "需要二次随访", Summary: "患者对候诊时间不满，认可医生沟通，建议 48 小时内回访确认用药。", ExtractedFormData: map[string]interface{}{"overall_satisfaction": 4, "problem_reasons": []string{"等待时间长"}}, CreatedAt: now.Add(-7 * time.Minute)}
	store.interviews["INT001"] = domain.InterviewSession{ID: "INT001", PatientID: "P002", FormID: "diabetes-management", Mode: "chat_call", Status: "draft", Messages: []domain.InterviewMessage{{Role: "assistant", Content: "您好，我想了解一下您最近一周的血糖和用药情况。", CreatedAt: now}}, FormDraft: map[string]interface{}{"follow_method": "phone"}, CreatedAt: now, UpdatedAt: now}
	store.dataSources["patients-api"] = domain.DataSource{
		ID:       "patients-api",
		Name:     "患者主索引 API",
		Protocol: "http",
		Endpoint: "https://his.example.local/api/patients",
		Config:   map[string]interface{}{"method": "GET", "timeoutMs": 3000},
		FieldMapping: []domain.FieldMapping{
			{Source: "$.id", Target: "patient.patientNo", Required: true},
			{Source: "$.name", Target: "patient.name", Required: true},
			{Source: "$.gender", Target: "patient.gender", Dictionary: "通用性别"},
			{Source: "$.phone", Target: "patient.phone"},
			{Source: "$.age", Target: "patient.age", Type: "int"},
			{Source: "$.visit.visitNo", Target: "visit.visitNo"},
			{Source: "$.visit.departmentName", Target: "visit.departmentName"},
			{Source: "$.visit.diagnosisName", Target: "visit.diagnosisName"},
		},
		Dictionaries: []domain.DictionaryMapping{
			{Name: "通用性别", Entries: []domain.DictionaryEntry{{Key: "M", Label: "男", Value: "男"}, {Key: "F", Label: "女", Value: "女"}, {Key: "1", Label: "男", Value: "男"}, {Key: "2", Label: "女", Value: "女"}}},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.dataSources["survey-dict"] = domain.DataSource{
		ID:       "survey-dict",
		Name:     "满意度字典库",
		Protocol: "mysql",
		Endpoint: "mysql://survey-reader@db.local:3306/reporter",
		Config:   map[string]interface{}{"queryTemplate": "select label, value from survey_options where group_code = :group"},
		Dictionaries: []domain.DictionaryMapping{
			{Name: "满意度选项", KeyField: "group_code", LabelField: "label", ValueField: "value"},
		},
		FieldMapping: []domain.FieldMapping{
			{Source: "label", Target: "option_label", Dictionary: "满意度选项"},
			{Source: "value", Target: "option_value", Dictionary: "满意度选项"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.dataSources["hl7-adt"] = domain.DataSource{
		ID:       "hl7-adt",
		Name:     "HL7 ADT 入院登记",
		Protocol: "hl7",
		Endpoint: "mllp://his.example.local:2575",
		Config:   map[string]interface{}{"segments": []string{"PID", "PV1", "PR1"}},
		Dictionaries: []domain.DictionaryMapping{
			{Name: "HL7 性别", KeyField: "PID.8", LabelField: "display", ValueField: "code", Entries: []domain.DictionaryEntry{{Key: "M", Label: "男", Value: "男"}, {Key: "F", Label: "女", Value: "女"}, {Key: "O", Label: "其他", Value: "其他"}}},
		},
		FieldMapping: []domain.FieldMapping{
			{Source: "PID.3", Target: "patient.patientNo", Required: true},
			{Source: "PID.5.1", Target: "patient.name", Required: true},
			{Source: "PID.7", Target: "patient.birthDate"},
			{Source: "PID.8", Target: "patient.gender", Dictionary: "HL7 性别"},
			{Source: "PID.11", Target: "patient.address"},
			{Source: "PID.13", Target: "patient.phone"},
			{Source: "PV1.19", Target: "visit.visitNo"},
			{Source: "PV1.10", Target: "visit.departmentCode"},
			{Source: "PV1.44", Target: "visit.visitAt"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	store.dataSources["dicom-pacs"] = domain.DataSource{
		ID:       "dicom-pacs",
		Name:     "DICOM/PACS 检查影像",
		Protocol: "dicom",
		Endpoint: "https://pacs.example.local/dicom-web",
		Config:   map[string]interface{}{"query": "QIDO-RS", "timeoutMs": 5000},
		FieldMapping: []domain.FieldMapping{
			{Source: "0010,0020", Target: "patient.patientNo", Required: true},
			{Source: "0010,0010", Target: "patient.name"},
			{Source: "0008,0050", Target: "record.recordNo"},
			{Source: "0008,1030", Target: "record.studyDesc"},
			{Source: "0020,000D", Target: "record.studyUid"},
			{Source: "0008,0020", Target: "record.recordedAt"},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	return store
}

func (s *Store) Seats() []domain.AgentSeat {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seats := make([]domain.AgentSeat, 0, len(s.seats))
	for _, seat := range s.seats {
		seats = append(seats, s.enrichSeatLocked(seat))
	}
	return seats
}

func (s *Store) enrichSeatLocked(seat domain.AgentSeat) domain.AgentSeat {
	if user, ok := s.users[seat.UserID]; ok {
		seat.Username = user.Username
		seat.UserDisplay = user.DisplayName
	}
	return seat
}

func (s *Store) CreateSeat(seat domain.AgentSeat) domain.AgentSeat {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if seat.ID == "" {
		seat.ID = uuid.NewString()
	}
	if seat.Status == "" {
		seat.Status = "offline"
	}
	seat.CreatedAt = now
	seat.UpdatedAt = now
	s.seats[seat.ID] = seat
	return seat
}

func (s *Store) Seat(id string) (domain.AgentSeat, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	seat, ok := s.seats[id]
	return s.enrichSeatLocked(seat), ok
}

func (s *Store) UpdateSeat(id string, patch domain.AgentSeat) (domain.AgentSeat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seat, ok := s.seats[id]
	if !ok {
		return domain.AgentSeat{}, ErrNotFound
	}
	seat.UserID = patch.UserID
	seat.Name = patch.Name
	seat.Extension = patch.Extension
	seat.SipURI = patch.SipURI
	seat.Status = patch.Status
	seat.Skills = patch.Skills
	seat.CurrentCall = patch.CurrentCall
	if seat.Status == "" {
		seat.Status = "offline"
	}
	seat.UpdatedAt = time.Now().UTC()
	s.seats[id] = seat
	return seat, nil
}

func (s *Store) DeleteSeat(id string) (domain.AgentSeat, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	seat, ok := s.seats[id]
	if !ok {
		return domain.AgentSeat{}, ErrNotFound
	}
	delete(s.seats, id)
	return seat, nil
}

func (s *Store) SipEndpoints() []domain.SipEndpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	endpoints := make([]domain.SipEndpoint, 0, len(s.sip))
	for _, endpoint := range s.sip {
		endpoints = append(endpoints, endpoint)
	}
	return endpoints
}

func (s *Store) DefaultSipEndpoint() (domain.SipEndpoint, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, endpoint := range s.sip {
		if sipConfigEnabled(endpoint.Config) {
			return endpoint, true
		}
	}
	if endpoint, ok := s.sip["SIP001"]; ok {
		return endpoint, true
	}
	for _, endpoint := range s.sip {
		return endpoint, true
	}
	return domain.SipEndpoint{}, false
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
	s.mu.RLock()
	defer s.mu.RUnlock()
	endpoint, ok := s.sip[id]
	return endpoint, ok
}

func (s *Store) CreateSipEndpoint(endpoint domain.SipEndpoint) domain.SipEndpoint {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if endpoint.ID == "" {
		endpoint.ID = uuid.NewString()
	}
	endpoint.CreatedAt = now
	endpoint.UpdatedAt = now
	s.sip[endpoint.ID] = endpoint
	return endpoint
}

func (s *Store) UpdateSipEndpoint(id string, patch domain.SipEndpoint) (domain.SipEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	endpoint, ok := s.sip[id]
	if !ok {
		return domain.SipEndpoint{}, ErrNotFound
	}
	endpoint.Name = patch.Name
	endpoint.WSSURL = patch.WSSURL
	endpoint.Domain = patch.Domain
	endpoint.Proxy = patch.Proxy
	endpoint.Config = patch.Config
	endpoint.UpdatedAt = time.Now().UTC()
	s.sip[id] = endpoint
	return endpoint, nil
}

func (s *Store) DeleteSipEndpoint(id string) (domain.SipEndpoint, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	endpoint, ok := s.sip[id]
	if !ok {
		return domain.SipEndpoint{}, ErrNotFound
	}
	delete(s.sip, id)
	return endpoint, nil
}

func (s *Store) StorageConfigs() []domain.StorageConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	configs := make([]domain.StorageConfig, 0, len(s.storageCfg))
	for _, config := range s.storageCfg {
		configs = append(configs, config)
	}
	return configs
}

func (s *Store) StorageConfig(id string) (domain.StorageConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	config, ok := s.storageCfg[id]
	return config, ok
}

func (s *Store) CreateStorageConfig(config domain.StorageConfig) domain.StorageConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if config.ID == "" {
		config.ID = uuid.NewString()
	}
	config.CreatedAt = now
	config.UpdatedAt = now
	s.storageCfg[config.ID] = config
	return config
}

func (s *Store) UpdateStorageConfig(id string, patch domain.StorageConfig) (domain.StorageConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, ok := s.storageCfg[id]
	if !ok {
		return domain.StorageConfig{}, ErrNotFound
	}
	config.Name = patch.Name
	config.Kind = patch.Kind
	config.Endpoint = patch.Endpoint
	config.Bucket = patch.Bucket
	config.BasePath = patch.BasePath
	config.BaseURI = patch.BaseURI
	config.CredentialRef = patch.CredentialRef
	config.Config = patch.Config
	config.UpdatedAt = time.Now().UTC()
	s.storageCfg[id] = config
	return config, nil
}

func (s *Store) DeleteStorageConfig(id string) (domain.StorageConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, ok := s.storageCfg[id]
	if !ok {
		return domain.StorageConfig{}, ErrNotFound
	}
	delete(s.storageCfg, id)
	return config, nil
}

func (s *Store) RecordingConfigs() []domain.RecordingConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	configs := make([]domain.RecordingConfig, 0, len(s.recordingCfg))
	for _, config := range s.recordingCfg {
		configs = append(configs, config)
	}
	return configs
}

func (s *Store) RecordingConfig(id string) (domain.RecordingConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	config, ok := s.recordingCfg[id]
	return config, ok
}

func (s *Store) DefaultRecordingConfig() (domain.RecordingConfig, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if config, ok := s.recordingCfg["REC-CFG-001"]; ok {
		return config, true
	}
	for _, config := range s.recordingCfg {
		return config, true
	}
	return domain.RecordingConfig{}, false
}

func (s *Store) CreateRecordingConfig(config domain.RecordingConfig) domain.RecordingConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if config.ID == "" {
		config.ID = uuid.NewString()
	}
	config.CreatedAt = now
	config.UpdatedAt = now
	s.recordingCfg[config.ID] = config
	return config
}

func (s *Store) UpdateRecordingConfig(id string, patch domain.RecordingConfig) (domain.RecordingConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, ok := s.recordingCfg[id]
	if !ok {
		return domain.RecordingConfig{}, ErrNotFound
	}
	config.Name = patch.Name
	config.Mode = patch.Mode
	config.StorageConfigID = patch.StorageConfigID
	config.Format = patch.Format
	config.RetentionDays = patch.RetentionDays
	config.AutoStart = patch.AutoStart
	config.AutoStop = patch.AutoStop
	config.Config = patch.Config
	config.UpdatedAt = time.Now().UTC()
	s.recordingCfg[id] = config
	return config, nil
}

func (s *Store) DeleteRecordingConfig(id string) (domain.RecordingConfig, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	config, ok := s.recordingCfg[id]
	if !ok {
		return domain.RecordingConfig{}, ErrNotFound
	}
	delete(s.recordingCfg, id)
	return config, nil
}

func (s *Store) Calls() []domain.CallSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	calls := make([]domain.CallSession, 0, len(s.calls))
	for _, call := range s.calls {
		calls = append(calls, call)
	}
	return calls
}

func (s *Store) CreateCall(call domain.CallSession) domain.CallSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	if call.ID == "" {
		call.ID = uuid.NewString()
	}
	if call.Status == "" {
		call.Status = "dialing"
	}
	call.StartedAt = time.Now().UTC()
	s.calls[call.ID] = call
	return call
}

func (s *Store) UpdateCall(id string, patch domain.CallSession) (domain.CallSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	call, ok := s.calls[id]
	if !ok {
		return domain.CallSession{}, ErrNotFound
	}
	if patch.Status != "" {
		call.Status = patch.Status
	}
	if patch.RecordingID != "" {
		call.RecordingID = patch.RecordingID
	}
	if patch.TranscriptID != "" {
		call.TranscriptID = patch.TranscriptID
	}
	if patch.AnalysisID != "" {
		call.AnalysisID = patch.AnalysisID
	}
	if !patch.EndedAt.IsZero() {
		call.EndedAt = patch.EndedAt
	}
	s.calls[id] = call
	return call, nil
}

func (s *Store) Recordings() []domain.Recording {
	s.mu.RLock()
	defer s.mu.RUnlock()
	recordings := make([]domain.Recording, 0, len(s.recordings))
	for _, recording := range s.recordings {
		recordings = append(recordings, recording)
	}
	return recordings
}

func (s *Store) CreateRecording(recording domain.Recording) domain.Recording {
	s.mu.Lock()
	defer s.mu.Unlock()
	if recording.ID == "" {
		recording.ID = uuid.NewString()
	}
	if recording.Status == "" {
		recording.Status = "ready"
	}
	if recording.Source == "" {
		recording.Source = "browser"
	}
	recording.CreatedAt = time.Now().UTC()
	s.recordings[recording.ID] = recording
	if call, ok := s.calls[recording.CallID]; ok {
		call.RecordingID = recording.ID
		if call.Status == "" || call.Status == "dialing" || call.Status == "connected" {
			call.Status = "recorded"
		}
		s.calls[recording.CallID] = call
	}
	return recording
}

func (s *Store) ModelProviders() []domain.ModelProvider {
	s.mu.RLock()
	defer s.mu.RUnlock()
	models := make([]domain.ModelProvider, 0, len(s.models))
	for _, model := range s.models {
		models = append(models, model)
	}
	return models
}

func (s *Store) ModelProvider(id string) (domain.ModelProvider, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	model, ok := s.models[id]
	return model, ok
}

func (s *Store) CreateModelProvider(provider domain.ModelProvider) domain.ModelProvider {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if provider.ID == "" {
		provider.ID = uuid.NewString()
	}
	provider.CreatedAt = now
	provider.UpdatedAt = now
	s.models[provider.ID] = provider
	return provider
}

func (s *Store) UpdateModelProvider(id string, patch domain.ModelProvider) (domain.ModelProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	provider, ok := s.models[id]
	if !ok {
		return domain.ModelProvider{}, ErrNotFound
	}
	provider.Name = patch.Name
	provider.Kind = patch.Kind
	provider.Mode = patch.Mode
	provider.Endpoint = patch.Endpoint
	provider.Model = patch.Model
	provider.CredentialRef = patch.CredentialRef
	provider.Config = patch.Config
	provider.UpdatedAt = time.Now().UTC()
	s.models[id] = provider
	return provider, nil
}

func (s *Store) DeleteModelProvider(id string) (domain.ModelProvider, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	provider, ok := s.models[id]
	if !ok {
		return domain.ModelProvider{}, ErrNotFound
	}
	delete(s.models, id)
	return provider, nil
}

func (s *Store) Analyses() []domain.CallAnalysis {
	s.mu.RLock()
	defer s.mu.RUnlock()
	analyses := make([]domain.CallAnalysis, 0, len(s.analyses))
	for _, analysis := range s.analyses {
		analyses = append(analyses, analysis)
	}
	return analyses
}

func (s *Store) RealtimeAssistSessions() []domain.RealtimeAssistSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sessions := make([]domain.RealtimeAssistSession, 0, len(s.realtime))
	for _, session := range s.realtime {
		sessions = append(sessions, session)
	}
	return sessions
}

func (s *Store) CreateRealtimeAssistSession(session domain.RealtimeAssistSession) domain.RealtimeAssistSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if session.ID == "" {
		session.ID = uuid.NewString()
	}
	if session.Status == "" {
		session.Status = "active"
	}
	session.CreatedAt = now
	session.UpdatedAt = now
	s.realtime[session.ID] = session
	return session
}

func (s *Store) AddRealtimeTranscript(id string, transcript domain.RealtimeTranscript, formPatch map[string]interface{}) (domain.RealtimeAssistSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.realtime[id]
	if !ok {
		return domain.RealtimeAssistSession{}, ErrNotFound
	}
	transcript.CreatedAt = time.Now().UTC()
	session.Transcript = append(session.Transcript, transcript)
	if session.FormDraft == nil {
		session.FormDraft = map[string]interface{}{}
	}
	for key, value := range formPatch {
		session.FormDraft[key] = value
	}
	if transcript.Text != "" {
		session.LastSuggestion = "已根据实时识别更新表单草稿"
	}
	session.UpdatedAt = time.Now().UTC()
	s.realtime[id] = session
	return session, nil
}

func (s *Store) OfflineAnalysisJobs() []domain.OfflineAnalysisJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	jobs := make([]domain.OfflineAnalysisJob, 0, len(s.offlineJobs))
	for _, job := range s.offlineJobs {
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *Store) CreateOfflineAnalysisJob(job domain.OfflineAnalysisJob) domain.OfflineAnalysisJob {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if job.ID == "" {
		job.ID = uuid.NewString()
	}
	if job.Status == "" {
		job.Status = "queued"
	}
	job.CreatedAt = now
	job.UpdatedAt = now
	s.offlineJobs[job.ID] = job
	return job
}

func (s *Store) Interviews() []domain.InterviewSession {
	s.mu.RLock()
	defer s.mu.RUnlock()
	interviews := make([]domain.InterviewSession, 0, len(s.interviews))
	for _, interview := range s.interviews {
		interviews = append(interviews, interview)
	}
	return interviews
}

func (s *Store) CreateInterview(interview domain.InterviewSession) domain.InterviewSession {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if interview.ID == "" {
		interview.ID = uuid.NewString()
	}
	if interview.Status == "" {
		interview.Status = "draft"
	}
	interview.CreatedAt = now
	interview.UpdatedAt = now
	s.interviews[interview.ID] = interview
	return interview
}

func (s *Store) UserByUsername(username string) (domain.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, user := range s.users {
		if user.Username == username {
			return user, true
		}
	}
	return domain.User{}, false
}

func (s *Store) UserByID(id string) (domain.User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	user, ok := s.users[id]
	return user, ok
}

func (s *Store) Users() []domain.User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	users := make([]domain.User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users
}

func (s *Store) CreateUser(user domain.User) domain.User {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	user.ID = uuid.NewString()
	user.CreatedAt = now
	user.UpdatedAt = now
	s.users[user.ID] = user
	return user
}

func (s *Store) UpdateUser(id string, patch domain.User) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	user.Username = patch.Username
	user.DisplayName = patch.DisplayName
	if len(patch.Roles) > 0 {
		user.Roles = patch.Roles
	}
	if patch.PasswordHash != "" {
		user.PasswordHash = patch.PasswordHash
	}
	user.UpdatedAt = time.Now().UTC()
	s.users[id] = user
	return user, nil
}

func (s *Store) DeleteUser(id string) (domain.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	user, ok := s.users[id]
	if !ok {
		return domain.User{}, ErrNotFound
	}
	delete(s.users, id)
	for seatID, seat := range s.seats {
		if seat.UserID == id {
			seat.UserID = ""
			s.seats[seatID] = seat
		}
	}
	return user, nil
}

func (s *Store) Patients(keyword string) []domain.Patient {
	if patients, err := s.dbPatients(context.Background(), keyword); err == nil {
		return patients
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	patients := make([]domain.Patient, 0, len(s.patients))
	for _, patient := range s.patients {
		if keyword == "" ||
			strings.Contains(strings.ToLower(patient.ID), keyword) ||
			strings.Contains(strings.ToLower(patient.PatientNo), keyword) ||
			strings.Contains(strings.ToLower(patient.Name), keyword) ||
			strings.Contains(strings.ToLower(patient.Phone), keyword) ||
			strings.Contains(strings.ToLower(patient.Diagnosis), keyword) {
			patients = append(patients, patient)
		}
	}
	return patients
}

func (s *Store) Patient(id string) (domain.Patient, bool) {
	if patient, ok, err := s.dbPatient(context.Background(), id); err == nil {
		return patient, ok
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	patient, ok := s.patients[id]
	return patient, ok
}

func (s *Store) CreatePatient(patient domain.Patient) domain.Patient {
	if created, err := s.dbCreatePatient(context.Background(), patient); err == nil {
		return created
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if patient.ID == "" {
		patient.ID = uuid.NewString()
	}
	if patient.Status == "" {
		patient.Status = "active"
	}
	patient.CreatedAt = now
	patient.UpdatedAt = now
	s.patients[patient.ID] = patient
	return patient
}

func (s *Store) UpdatePatient(id string, patch domain.Patient) (domain.Patient, error) {
	if updated, err := s.dbUpdatePatient(context.Background(), id, patch); err == nil || err == ErrNotFound {
		return updated, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	patient, ok := s.patients[id]
	if !ok {
		return domain.Patient{}, ErrNotFound
	}
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
	patient.UpdatedAt = time.Now().UTC()
	s.patients[id] = patient
	return patient, nil
}

func (s *Store) UpsertPatientByNo(patient domain.Patient) (domain.Patient, bool) {
	if saved, created, err := s.dbUpsertPatientByNo(context.Background(), patient); err == nil {
		return saved, created
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for id, existing := range s.patients {
		if existing.PatientNo != "" && existing.PatientNo == patient.PatientNo {
			patient.ID = id
			patient.CreatedAt = existing.CreatedAt
			patient.UpdatedAt = now
			if patient.Status == "" {
				patient.Status = existing.Status
			}
			s.patients[id] = patient
			return patient, false
		}
	}
	if patient.ID == "" {
		patient.ID = uuid.NewString()
	}
	if patient.Status == "" {
		patient.Status = "active"
	}
	patient.CreatedAt = now
	patient.UpdatedAt = now
	s.patients[patient.ID] = patient
	return patient, true
}

func (s *Store) Visits(patientID string) []domain.ClinicalVisit {
	if visits, err := s.dbVisits(context.Background(), patientID); err == nil {
		return visits
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	visits := make([]domain.ClinicalVisit, 0, len(s.visits))
	for _, visit := range s.visits {
		if patientID == "" || visit.PatientID == patientID {
			visits = append(visits, visit)
		}
	}
	return visits
}

func (s *Store) UpsertVisitByNo(visit domain.ClinicalVisit) (domain.ClinicalVisit, bool) {
	if saved, created, err := s.dbUpsertVisitByNo(context.Background(), visit); err == nil {
		return saved, created
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for id, existing := range s.visits {
		if existing.VisitNo != "" && existing.VisitNo == visit.VisitNo {
			visit.ID = id
			visit.CreatedAt = existing.CreatedAt
			visit.UpdatedAt = now
			if visit.Status == "" {
				visit.Status = existing.Status
			}
			s.visits[id] = visit
			return visit, false
		}
	}
	if visit.ID == "" {
		visit.ID = uuid.NewString()
	}
	if visit.Status == "" {
		visit.Status = "active"
	}
	visit.CreatedAt = now
	visit.UpdatedAt = now
	s.visits[visit.ID] = visit
	return visit, true
}

func (s *Store) MedicalRecords(patientID string) []domain.MedicalRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()
	records := make([]domain.MedicalRecord, 0, len(s.records))
	for _, record := range s.records {
		if patientID == "" || record.PatientID == patientID {
			records = append(records, record)
		}
	}
	return records
}

func (s *Store) UpsertMedicalRecordByNo(record domain.MedicalRecord) (domain.MedicalRecord, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	for id, existing := range s.records {
		if existing.RecordNo != "" && existing.RecordNo == record.RecordNo {
			record.ID = id
			record.CreatedAt = existing.CreatedAt
			record.UpdatedAt = now
			s.records[id] = record
			return record, false
		}
	}
	if record.ID == "" {
		record.ID = uuid.NewString()
	}
	record.CreatedAt = now
	record.UpdatedAt = now
	s.records[record.ID] = record
	return record, true
}

func (s *Store) Datasets(keyword string) []domain.Dataset {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keyword = strings.TrimSpace(strings.ToLower(keyword))
	datasets := make([]domain.Dataset, 0, len(s.datasets))
	for _, dataset := range s.datasets {
		if keyword == "" ||
			strings.Contains(strings.ToLower(dataset.ID), keyword) ||
			strings.Contains(strings.ToLower(dataset.Name), keyword) ||
			strings.Contains(strings.ToLower(dataset.Description), keyword) ||
			strings.Contains(strings.ToLower(dataset.Owner), keyword) {
			datasets = append(datasets, dataset)
		}
	}
	return datasets
}

func (s *Store) Dataset(id string) (domain.Dataset, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dataset, ok := s.datasets[id]
	return dataset, ok
}

func (s *Store) CreateDataset(dataset domain.Dataset) domain.Dataset {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	if dataset.ID == "" {
		dataset.ID = uuid.NewString()
	}
	if dataset.Status == "" {
		dataset.Status = "active"
	}
	dataset.CreatedAt = now
	dataset.UpdatedAt = now
	s.datasets[dataset.ID] = dataset
	return dataset
}

func (s *Store) UpdateDataset(id string, patch domain.Dataset) (domain.Dataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dataset, ok := s.datasets[id]
	if !ok {
		return domain.Dataset{}, ErrNotFound
	}
	dataset.Name = patch.Name
	dataset.Description = patch.Description
	dataset.Owner = patch.Owner
	dataset.RecordCount = patch.RecordCount
	dataset.FormCount = patch.FormCount
	dataset.Status = patch.Status
	if dataset.Status == "" {
		dataset.Status = "active"
	}
	dataset.UpdatedAt = time.Now().UTC()
	s.datasets[id] = dataset
	return dataset, nil
}

func (s *Store) DeleteDataset(id string) (domain.Dataset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dataset, ok := s.datasets[id]
	if !ok {
		return domain.Dataset{}, ErrNotFound
	}
	delete(s.datasets, id)
	return dataset, nil
}

func (s *Store) Roles() []domain.Role {
	s.mu.RLock()
	defer s.mu.RUnlock()
	roles := make([]domain.Role, 0, len(s.roles))
	for _, role := range s.roles {
		roles = append(roles, role)
	}
	return roles
}

func (s *Store) CreateRole(role domain.Role) domain.Role {
	s.mu.Lock()
	defer s.mu.Unlock()
	if role.ID == "" {
		role.ID = uuid.NewString()
	}
	s.roles[role.ID] = role
	return role
}

func (s *Store) UpdateRolePermissions(id string, permissions []string) (domain.Role, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	role, ok := s.roles[id]
	if !ok {
		return domain.Role{}, ErrNotFound
	}
	role.Permissions = permissions
	s.roles[id] = role
	return role, nil
}

func (s *Store) SaveAudit(log domain.AuditLog) {
	s.mu.Lock()
	defer s.mu.Unlock()
	log.ID = uuid.NewString()
	log.CreatedAt = time.Now().UTC()
	s.auditLogs = append(s.auditLogs, log)
}

func (s *Store) AuditLogs() []domain.AuditLog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	logs := make([]domain.AuditLog, len(s.auditLogs))
	copy(logs, s.auditLogs)
	return logs
}

func (s *Store) Forms() []domain.Form {
	s.mu.RLock()
	defer s.mu.RUnlock()
	forms := make([]domain.Form, 0, len(s.forms))
	for _, form := range s.forms {
		forms = append(forms, form)
	}
	return forms
}

func (s *Store) CreateForm(form domain.Form) domain.Form {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	form.ID = uuid.NewString()
	form.Status = "draft"
	form.CreatedAt = now
	form.UpdatedAt = now
	s.forms[form.ID] = form
	return form
}

func (s *Store) CreateFormVersion(formID, actor string, schema []domain.FormComponent) (domain.FormVersion, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	form, ok := s.forms[formID]
	if !ok {
		return domain.FormVersion{}, ErrNotFound
	}
	version := domain.FormVersion{
		ID:        uuid.NewString(),
		FormID:    formID,
		Version:   len(form.Versions) + 1,
		Schema:    schema,
		CreatedBy: actor,
		CreatedAt: time.Now().UTC(),
	}
	form.Versions = append(form.Versions, version)
	form.UpdatedAt = version.CreatedAt
	s.forms[formID] = form
	return version, nil
}

func (s *Store) PublishForm(formID string) (domain.Form, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	form, ok := s.forms[formID]
	if !ok {
		return domain.Form{}, ErrNotFound
	}
	if len(form.Versions) == 0 {
		return domain.Form{}, errors.New("form has no version")
	}
	for i := range form.Versions {
		form.Versions[i].Published = false
	}
	form.Versions[len(form.Versions)-1].Published = true
	form.CurrentVersionID = form.Versions[len(form.Versions)-1].ID
	form.Status = "published"
	form.UpdatedAt = time.Now().UTC()
	s.forms[formID] = form
	return form, nil
}

func (s *Store) CreateSubmission(submission domain.Submission) (domain.Submission, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	form, ok := s.forms[submission.FormID]
	if !ok {
		return domain.Submission{}, ErrNotFound
	}
	if submission.FormVersionID == "" {
		submission.FormVersionID = form.CurrentVersionID
	}
	now := time.Now().UTC()
	submission.ID = uuid.NewString()
	submission.Status = "submitted"
	submission.CreatedAt = now
	submission.UpdatedAt = now
	s.submissions[submission.ID] = submission
	return submission, nil
}

func (s *Store) SubmissionsByForm(formID string) []domain.Submission {
	s.mu.RLock()
	defer s.mu.RUnlock()
	submissions := []domain.Submission{}
	for _, submission := range s.submissions {
		if submission.FormID == formID {
			submissions = append(submissions, submission)
		}
	}
	return submissions
}

func (s *Store) Submission(id string) (domain.Submission, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	submission, ok := s.submissions[id]
	return submission, ok
}

func (s *Store) DataSources() []domain.DataSource {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sources := make([]domain.DataSource, 0, len(s.dataSources))
	for _, source := range s.dataSources {
		sources = append(sources, source)
	}
	return sources
}

func (s *Store) CreateDataSource(source domain.DataSource) domain.DataSource {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	source.ID = uuid.NewString()
	source.CreatedAt = now
	source.UpdatedAt = now
	s.dataSources[source.ID] = source
	return source
}

func (s *Store) DataSource(id string) (domain.DataSource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	source, ok := s.dataSources[id]
	return source, ok
}

func (s *Store) UpdateDataSource(id string, patch domain.DataSource) (domain.DataSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	source, ok := s.dataSources[id]
	if !ok {
		return domain.DataSource{}, ErrNotFound
	}
	if patch.Name != "" {
		source.Name = patch.Name
	}
	if patch.Protocol != "" {
		source.Protocol = patch.Protocol
	}
	if patch.Endpoint != "" {
		source.Endpoint = patch.Endpoint
	}
	if patch.Config != nil {
		source.Config = patch.Config
	}
	if patch.Dictionaries != nil {
		source.Dictionaries = patch.Dictionaries
	}
	if patch.FieldMapping != nil {
		source.FieldMapping = patch.FieldMapping
	}
	source.UpdatedAt = time.Now().UTC()
	s.dataSources[id] = source
	return source, nil
}

func (s *Store) DeleteDataSource(id string) (domain.DataSource, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	source, ok := s.dataSources[id]
	if !ok {
		return domain.DataSource{}, ErrNotFound
	}
	delete(s.dataSources, id)
	return source, nil
}

func (s *Store) Reports() []domain.Report {
	s.mu.RLock()
	defer s.mu.RUnlock()
	reports := make([]domain.Report, 0, len(s.reports))
	for _, report := range s.reports {
		reports = append(reports, report)
	}
	return reports
}

func (s *Store) Report(id string) (domain.Report, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	report, ok := s.reports[id]
	return report, ok
}

func (s *Store) CreateReport(report domain.Report) domain.Report {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now().UTC()
	report.ID = uuid.NewString()
	report.CreatedAt = now
	report.UpdatedAt = now
	s.reports[report.ID] = report
	return report
}

func (s *Store) UpdateReport(id string, patch domain.Report) (domain.Report, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report, ok := s.reports[id]
	if !ok {
		return domain.Report{}, ErrNotFound
	}
	if patch.Name != "" {
		report.Name = patch.Name
	}
	if patch.Description != "" {
		report.Description = patch.Description
	}
	report.UpdatedAt = time.Now().UTC()
	s.reports[id] = report
	return report, nil
}

func (s *Store) AddReportWidget(reportID string, widget domain.ReportWidget) (domain.ReportWidget, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	report, ok := s.reports[reportID]
	if !ok {
		return domain.ReportWidget{}, ErrNotFound
	}
	widget.ID = uuid.NewString()
	widget.ReportID = reportID
	widget.CreatedAt = time.Now().UTC()
	report.Widgets = append(report.Widgets, widget)
	report.UpdatedAt = widget.CreatedAt
	s.reports[reportID] = report
	return widget, nil
}

func (s *Store) QueryReport(reportID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.reports[reportID]; !ok {
		return nil, ErrNotFound
	}
	if reportID == "RP002" {
		return map[string]interface{}{
			"dimensions": []string{"department"},
			"measures":   []string{"satisfaction", "recommendScore", "feedbackCount"},
			"rows": []map[string]interface{}{
				{"department": "心内科", "satisfaction": 4.6, "recommendScore": 9.1, "feedbackCount": 82},
				{"department": "内分泌科", "satisfaction": 4.3, "recommendScore": 8.7, "feedbackCount": 64},
				{"department": "体检中心", "satisfaction": 4.1, "recommendScore": 8.2, "feedbackCount": 51},
			},
		}, nil
	}
	return map[string]interface{}{
		"dimensions": []string{"month"},
		"measures":   []string{"submissions", "completed", "completionRate"},
		"rows": []map[string]interface{}{
			{"month": "2026-01", "submissions": 126, "completed": 112, "completionRate": 88.9},
			{"month": "2026-02", "submissions": 142, "completed": 130, "completionRate": 91.5},
			{"month": "2026-03", "submissions": 168, "completed": 151, "completionRate": 89.9},
			{"month": "2026-04", "submissions": 184, "completed": 167, "completionRate": 90.8},
			{"month": "2026-05", "submissions": 201, "completed": 188, "completionRate": 93.5},
		},
	}, nil
}
