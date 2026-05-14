package domain

import "time"

type Role struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	DisplayName  string    `json:"displayName"`
	PasswordHash string    `json:"-"`
	Roles        []string  `json:"roles"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt,omitempty"`
}

type Patient struct {
	ID               string                 `json:"id"`
	PatientNo        string                 `json:"patientNo"`
	MedicalRecordNo  string                 `json:"medicalRecordNo,omitempty"`
	Name             string                 `json:"name"`
	Gender           string                 `json:"gender"`
	BirthDate        string                 `json:"birthDate,omitempty"`
	Age              int                    `json:"age"`
	IDCardNo         string                 `json:"idCardNo,omitempty"`
	Phone            string                 `json:"phone"`
	Address          string                 `json:"address,omitempty"`
	Nationality      string                 `json:"nationality,omitempty"`
	Ethnicity        string                 `json:"ethnicity,omitempty"`
	MaritalStatus    string                 `json:"maritalStatus,omitempty"`
	InsuranceType    string                 `json:"insuranceType,omitempty"`
	BloodType        string                 `json:"bloodType,omitempty"`
	Allergies        []string               `json:"allergies,omitempty"`
	EmergencyContact string                 `json:"emergencyContact,omitempty"`
	EmergencyPhone   string                 `json:"emergencyPhone,omitempty"`
	Diagnosis        string                 `json:"diagnosis"`
	Status           string                 `json:"status"`
	LastVisitAt      string                 `json:"lastVisitAt"`
	SourceRefs       map[string]interface{} `json:"sourceRefs,omitempty"`
	CreatedAt        time.Time              `json:"createdAt"`
	UpdatedAt        time.Time              `json:"updatedAt"`
}

type ClinicalVisit struct {
	ID              string                 `json:"id"`
	PatientID       string                 `json:"patientId"`
	VisitNo         string                 `json:"visitNo"`
	VisitType       string                 `json:"visitType"`
	DepartmentCode  string                 `json:"departmentCode,omitempty"`
	DepartmentName  string                 `json:"departmentName,omitempty"`
	Ward            string                 `json:"ward,omitempty"`
	BedNo           string                 `json:"bedNo,omitempty"`
	AttendingDoctor string                 `json:"attendingDoctor,omitempty"`
	VisitAt         string                 `json:"visitAt,omitempty"`
	DischargeAt     string                 `json:"dischargeAt,omitempty"`
	DiagnosisCode   string                 `json:"diagnosisCode,omitempty"`
	DiagnosisName   string                 `json:"diagnosisName,omitempty"`
	Status          string                 `json:"status"`
	SourceRefs      map[string]interface{} `json:"sourceRefs,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type MedicalRecord struct {
	ID             string                 `json:"id"`
	PatientID      string                 `json:"patientId"`
	VisitID        string                 `json:"visitId,omitempty"`
	RecordNo       string                 `json:"recordNo"`
	RecordType     string                 `json:"recordType"`
	Title          string                 `json:"title"`
	Summary        string                 `json:"summary,omitempty"`
	ChiefComplaint string                 `json:"chiefComplaint,omitempty"`
	PresentIllness string                 `json:"presentIllness,omitempty"`
	DiagnosisCode  string                 `json:"diagnosisCode,omitempty"`
	DiagnosisName  string                 `json:"diagnosisName,omitempty"`
	ProcedureName  string                 `json:"procedureName,omitempty"`
	StudyUID       string                 `json:"studyUid,omitempty"`
	StudyDesc      string                 `json:"studyDesc,omitempty"`
	RecordedAt     string                 `json:"recordedAt,omitempty"`
	SourceRefs     map[string]interface{} `json:"sourceRefs,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type PatientDiagnosis struct {
	ID             string    `json:"id"`
	PatientID      string    `json:"patientId"`
	VisitID        string    `json:"visitId,omitempty"`
	DiagnosisCode  string    `json:"diagnosisCode,omitempty"`
	DiagnosisName  string    `json:"diagnosisName"`
	DiagnosisType  string    `json:"diagnosisType"`
	DiagnosedAt    string    `json:"diagnosedAt,omitempty"`
	DepartmentName string    `json:"departmentName,omitempty"`
	DoctorName     string    `json:"doctorName,omitempty"`
	SourceSystem   string    `json:"sourceSystem,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type PatientHistory struct {
	ID           string    `json:"id"`
	PatientID    string    `json:"patientId"`
	HistoryType  string    `json:"historyType"`
	Title        string    `json:"title"`
	Content      string    `json:"content,omitempty"`
	RecordedAt   string    `json:"recordedAt,omitempty"`
	SourceSystem string    `json:"sourceSystem,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type MedicationOrder struct {
	ID              string    `json:"id"`
	PatientID       string    `json:"patientId"`
	VisitID         string    `json:"visitId,omitempty"`
	OrderNo         string    `json:"orderNo,omitempty"`
	PrescriptionNo  string    `json:"prescriptionNo,omitempty"`
	DrugCode        string    `json:"drugCode,omitempty"`
	DrugName        string    `json:"drugName"`
	GenericName     string    `json:"genericName,omitempty"`
	Specification   string    `json:"specification,omitempty"`
	Dosage          string    `json:"dosage,omitempty"`
	DosageUnit      string    `json:"dosageUnit,omitempty"`
	Frequency       string    `json:"frequency,omitempty"`
	Route           string    `json:"route,omitempty"`
	StartAt         string    `json:"startAt,omitempty"`
	EndAt           string    `json:"endAt,omitempty"`
	Days            int       `json:"days,omitempty"`
	Quantity        float64   `json:"quantity,omitempty"`
	Manufacturer    string    `json:"manufacturer,omitempty"`
	DoctorName      string    `json:"doctorName,omitempty"`
	PharmacistName  string    `json:"pharmacistName,omitempty"`
	Status          string    `json:"status"`
	AdverseReaction string    `json:"adverseReaction,omitempty"`
	Compliance      string    `json:"compliance,omitempty"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type LabReport struct {
	ID             string      `json:"id"`
	PatientID      string      `json:"patientId"`
	VisitID        string      `json:"visitId,omitempty"`
	ReportNo       string      `json:"reportNo"`
	ReportName     string      `json:"reportName"`
	Specimen       string      `json:"specimen,omitempty"`
	OrderedAt      string      `json:"orderedAt,omitempty"`
	ReportedAt     string      `json:"reportedAt,omitempty"`
	DepartmentName string      `json:"departmentName,omitempty"`
	DoctorName     string      `json:"doctorName,omitempty"`
	Status         string      `json:"status"`
	SourceSystem   string      `json:"sourceSystem,omitempty"`
	Results        []LabResult `json:"results,omitempty"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
}

type LabResult struct {
	ID             string    `json:"id"`
	ReportID       string    `json:"reportId"`
	ItemCode       string    `json:"itemCode,omitempty"`
	ItemName       string    `json:"itemName"`
	ResultValue    string    `json:"resultValue,omitempty"`
	Unit           string    `json:"unit,omitempty"`
	ReferenceRange string    `json:"referenceRange,omitempty"`
	AbnormalFlag   string    `json:"abnormalFlag,omitempty"`
	NumericValue   float64   `json:"numericValue,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
}

type ExamReport struct {
	ID               string    `json:"id"`
	PatientID        string    `json:"patientId"`
	VisitID          string    `json:"visitId,omitempty"`
	ExamNo           string    `json:"examNo"`
	ExamType         string    `json:"examType,omitempty"`
	ExamName         string    `json:"examName"`
	BodyPart         string    `json:"bodyPart,omitempty"`
	ReportConclusion string    `json:"reportConclusion,omitempty"`
	ReportFindings   string    `json:"reportFindings,omitempty"`
	OrderedAt        string    `json:"orderedAt,omitempty"`
	ReportedAt       string    `json:"reportedAt,omitempty"`
	DepartmentName   string    `json:"departmentName,omitempty"`
	DoctorName       string    `json:"doctorName,omitempty"`
	SourceSystem     string    `json:"sourceSystem,omitempty"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type SurgeryRecord struct {
	ID             string    `json:"id"`
	PatientID      string    `json:"patientId"`
	VisitID        string    `json:"visitId,omitempty"`
	OperationCode  string    `json:"operationCode,omitempty"`
	OperationName  string    `json:"operationName"`
	OperationDate  string    `json:"operationDate,omitempty"`
	SurgeonName    string    `json:"surgeonName,omitempty"`
	AnesthesiaType string    `json:"anesthesiaType,omitempty"`
	OperationLevel string    `json:"operationLevel,omitempty"`
	WoundGrade     string    `json:"woundGrade,omitempty"`
	Outcome        string    `json:"outcome,omitempty"`
	SourceSystem   string    `json:"sourceSystem,omitempty"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type FollowupRecord struct {
	ID                string    `json:"id"`
	PatientID         string    `json:"patientId"`
	VisitID           string    `json:"visitId,omitempty"`
	TaskID            string    `json:"taskId,omitempty"`
	ProjectID         string    `json:"projectId,omitempty"`
	FollowupType      string    `json:"followupType,omitempty"`
	Channel           string    `json:"channel,omitempty"`
	Status            string    `json:"status"`
	Summary           string    `json:"summary,omitempty"`
	SatisfactionScore float64   `json:"satisfactionScore,omitempty"`
	RiskLevel         string    `json:"riskLevel,omitempty"`
	FollowedAt        string    `json:"followedAt,omitempty"`
	OperatorName      string    `json:"operatorName,omitempty"`
	SourceSystem      string    `json:"sourceSystem,omitempty"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type InterviewExtractedFact struct {
	ID          string    `json:"id"`
	PatientID   string    `json:"patientId"`
	VisitID     string    `json:"visitId,omitempty"`
	InterviewID string    `json:"interviewId,omitempty"`
	FactType    string    `json:"factType"`
	FactKey     string    `json:"factKey"`
	FactLabel   string    `json:"factLabel"`
	FactValue   string    `json:"factValue,omitempty"`
	Confidence  float64   `json:"confidence,omitempty"`
	ExtractedAt string    `json:"extractedAt,omitempty"`
	SourceText  string    `json:"sourceText,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SatisfactionIndicatorScore struct {
	ID             string                 `json:"id"`
	ProjectID      string                 `json:"projectId"`
	IndicatorID    string                 `json:"indicatorId"`
	PatientID      string                 `json:"patientId,omitempty"`
	VisitID        string                 `json:"visitId,omitempty"`
	DepartmentName string                 `json:"departmentName,omitempty"`
	DoctorName     string                 `json:"doctorName,omitempty"`
	NurseName      string                 `json:"nurseName,omitempty"`
	DiseaseName    string                 `json:"diseaseName,omitempty"`
	VisitType      string                 `json:"visitType,omitempty"`
	Score          float64                `json:"score"`
	SampleCount    int                    `json:"sampleCount"`
	ScorePeriod    string                 `json:"scorePeriod,omitempty"`
	Source         map[string]interface{} `json:"source,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type Patient360 struct {
	Patient         Patient                      `json:"patient"`
	Visits          []ClinicalVisit              `json:"visits"`
	MedicalRecords  []MedicalRecord              `json:"medicalRecords"`
	Diagnoses       []PatientDiagnosis           `json:"diagnoses"`
	Histories       []PatientHistory             `json:"histories"`
	Medications     []MedicationOrder            `json:"medications"`
	LabReports      []LabReport                  `json:"labReports"`
	ExamReports     []ExamReport                 `json:"examReports"`
	Surgeries       []SurgeryRecord              `json:"surgeries"`
	FollowupRecords []FollowupRecord             `json:"followupRecords"`
	InterviewFacts  []InterviewExtractedFact     `json:"interviewFacts"`
	IndicatorScores []SatisfactionIndicatorScore `json:"indicatorScores,omitempty"`
}

type Dataset struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Owner       string    `json:"owner"`
	RecordCount int       `json:"recordCount"`
	FormCount   int       `json:"formCount"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type AgentSeat struct {
	ID          string    `json:"id"`
	UserID      string    `json:"userId,omitempty"`
	Username    string    `json:"username,omitempty"`
	UserDisplay string    `json:"userDisplay,omitempty"`
	Name        string    `json:"name"`
	Extension   string    `json:"extension"`
	SipURI      string    `json:"sipUri"`
	Status      string    `json:"status"`
	Skills      []string  `json:"skills"`
	CurrentCall string    `json:"currentCall,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SipEndpoint struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	WSSURL    string                 `json:"wssUrl"`
	Domain    string                 `json:"domain"`
	Proxy     string                 `json:"proxy"`
	Config    map[string]interface{} `json:"config,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

type StorageConfig struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Kind          string                 `json:"kind"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	Bucket        string                 `json:"bucket,omitempty"`
	BasePath      string                 `json:"basePath,omitempty"`
	BaseURI       string                 `json:"baseUri,omitempty"`
	CredentialRef string                 `json:"credentialRef,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

type RecordingConfig struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Mode            string                 `json:"mode"`
	StorageConfigID string                 `json:"storageConfigId"`
	Format          string                 `json:"format"`
	RetentionDays   int                    `json:"retentionDays"`
	AutoStart       bool                   `json:"autoStart"`
	AutoStop        bool                   `json:"autoStop"`
	Config          map[string]interface{} `json:"config,omitempty"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
}

type CallSession struct {
	ID            string    `json:"id"`
	SeatID        string    `json:"seatId"`
	PatientID     string    `json:"patientId,omitempty"`
	Direction     string    `json:"direction"`
	PhoneNumber   string    `json:"phoneNumber"`
	Status        string    `json:"status"`
	StartedAt     time.Time `json:"startedAt"`
	EndedAt       time.Time `json:"endedAt,omitempty"`
	RecordingID   string    `json:"recordingId,omitempty"`
	TranscriptID  string    `json:"transcriptId,omitempty"`
	AnalysisID    string    `json:"analysisId,omitempty"`
	InterviewForm string    `json:"interviewForm,omitempty"`
}

type Recording struct {
	ID         string    `json:"id"`
	CallID     string    `json:"callId"`
	StorageURI string    `json:"storageUri"`
	Duration   int       `json:"duration"`
	Filename   string    `json:"filename,omitempty"`
	MimeType   string    `json:"mimeType,omitempty"`
	SizeBytes  int64     `json:"sizeBytes,omitempty"`
	Source     string    `json:"source,omitempty"`
	Backend    string    `json:"backend,omitempty"`
	ObjectName string    `json:"objectName,omitempty"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ModelProvider struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Kind          string                 `json:"kind"`
	Mode          string                 `json:"mode"`
	Endpoint      string                 `json:"endpoint"`
	Model         string                 `json:"model"`
	CredentialRef string                 `json:"credentialRef,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

type RealtimeAssistSession struct {
	ID             string                 `json:"id"`
	CallID         string                 `json:"callId"`
	PatientID      string                 `json:"patientId,omitempty"`
	FormID         string                 `json:"formId"`
	ProviderID     string                 `json:"providerId"`
	Status         string                 `json:"status"`
	Transcript     []RealtimeTranscript   `json:"transcript"`
	FormDraft      map[string]interface{} `json:"formDraft,omitempty"`
	LastSuggestion string                 `json:"lastSuggestion,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type RealtimeTranscript struct {
	Speaker   string    `json:"speaker"`
	Text      string    `json:"text"`
	IsFinal   bool      `json:"isFinal"`
	CreatedAt time.Time `json:"createdAt"`
}

type OfflineAnalysisJob struct {
	ID          string                 `json:"id"`
	CallID      string                 `json:"callId"`
	RecordingID string                 `json:"recordingId"`
	ProviderID  string                 `json:"providerId"`
	Status      string                 `json:"status"`
	Result      map[string]interface{} `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	CreatedAt   time.Time              `json:"createdAt"`
	UpdatedAt   time.Time              `json:"updatedAt"`
}

type CallAnalysis struct {
	ID                string                 `json:"id"`
	CallID            string                 `json:"callId"`
	ProviderID        string                 `json:"providerId"`
	PatientEmotion    string                 `json:"patientEmotion"`
	TrueSatisfaction  float64                `json:"trueSatisfaction"`
	RiskLevel         string                 `json:"riskLevel"`
	PatientStatus     string                 `json:"patientStatus"`
	Summary           string                 `json:"summary"`
	ExtractedFormData map[string]interface{} `json:"extractedFormData,omitempty"`
	CreatedAt         time.Time              `json:"createdAt"`
}

type InterviewSession struct {
	ID        string                 `json:"id"`
	PatientID string                 `json:"patientId"`
	FormID    string                 `json:"formId"`
	CallID    string                 `json:"callId,omitempty"`
	Mode      string                 `json:"mode"`
	Status    string                 `json:"status"`
	Messages  []InterviewMessage     `json:"messages"`
	FormDraft map[string]interface{} `json:"formDraft,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

type InterviewMessage struct {
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type AuditLog struct {
	ID        string      `json:"id"`
	ActorID   string      `json:"actorId"`
	Action    string      `json:"action"`
	Resource  string      `json:"resource"`
	Before    interface{} `json:"before,omitempty"`
	After     interface{} `json:"after,omitempty"`
	IP        string      `json:"ip"`
	UserAgent string      `json:"userAgent"`
	TraceID   string      `json:"traceId"`
	CreatedAt time.Time   `json:"createdAt"`
}

type DataBinding struct {
	Kind         string                 `json:"kind"`
	DataSourceID string                 `json:"dataSourceId,omitempty"`
	Operation    string                 `json:"operation,omitempty"`
	Params       map[string]interface{} `json:"params,omitempty"`
}

type FormComponent struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Label    string                 `json:"label"`
	Required bool                   `json:"required"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Binding  *DataBinding           `json:"binding,omitempty"`
	Children []FormComponent        `json:"children,omitempty"`
}

type FormLibraryItem struct {
	ID         string                   `json:"id"`
	Kind       string                   `json:"kind"`
	Label      string                   `json:"label"`
	Hint       string                   `json:"hint"`
	Scenario   string                   `json:"scenario,omitempty"`
	Components []map[string]interface{} `json:"components"`
	SortOrder  int                      `json:"sortOrder"`
	Enabled    bool                     `json:"enabled"`
}

type Department struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Dictionary struct {
	ID          string            `json:"id"`
	Code        string            `json:"code"`
	Name        string            `json:"name"`
	Category    string            `json:"category"`
	Description string            `json:"description,omitempty"`
	Items       []DictionaryEntry `json:"items"`
	CreatedAt   time.Time         `json:"createdAt"`
	UpdatedAt   time.Time         `json:"updatedAt"`
}

type FollowupPlan struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Scenario       string                 `json:"scenario"`
	DiseaseCode    string                 `json:"diseaseCode,omitempty"`
	DepartmentID   string                 `json:"departmentId,omitempty"`
	FormTemplateID string                 `json:"formTemplateId"`
	TriggerType    string                 `json:"triggerType"`
	TriggerOffset  int                    `json:"triggerOffset"`
	Channel        string                 `json:"channel"`
	AssigneeRole   string                 `json:"assigneeRole"`
	Status         string                 `json:"status"`
	Rules          map[string]interface{} `json:"rules,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type FollowupTask struct {
	ID             string                 `json:"id"`
	PlanID         string                 `json:"planId,omitempty"`
	PatientID      string                 `json:"patientId"`
	PatientName    string                 `json:"patientName,omitempty"`
	PatientPhone   string                 `json:"patientPhone,omitempty"`
	VisitID        string                 `json:"visitId,omitempty"`
	FormID         string                 `json:"formId,omitempty"`
	FormTemplateID string                 `json:"formTemplateId,omitempty"`
	AssigneeID     string                 `json:"assigneeId,omitempty"`
	AssigneeName   string                 `json:"assigneeName,omitempty"`
	Role           string                 `json:"role,omitempty"`
	Channel        string                 `json:"channel"`
	Status         string                 `json:"status"`
	Priority       string                 `json:"priority"`
	DueAt          string                 `json:"dueAt"`
	Result         map[string]interface{} `json:"result,omitempty"`
	LastEvent      string                 `json:"lastEvent,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type Form struct {
	ID               string        `json:"id"`
	Name             string        `json:"name"`
	Description      string        `json:"description"`
	Status           string        `json:"status"`
	CurrentVersionID string        `json:"currentVersionId,omitempty"`
	CreatedAt        time.Time     `json:"createdAt"`
	UpdatedAt        time.Time     `json:"updatedAt"`
	Versions         []FormVersion `json:"versions,omitempty"`
}

type FormVersion struct {
	ID        string          `json:"id"`
	FormID    string          `json:"formId"`
	Version   int             `json:"version"`
	Schema    []FormComponent `json:"schema"`
	CreatedBy string          `json:"createdBy"`
	CreatedAt time.Time       `json:"createdAt"`
	Published bool            `json:"published"`
}

type Submission struct {
	ID            string                 `json:"id"`
	FormID        string                 `json:"formId"`
	FormVersionID string                 `json:"formVersionId"`
	SubmitterID   string                 `json:"submitterId"`
	Status        string                 `json:"status"`
	Data          map[string]interface{} `json:"data"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

type DataSource struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Protocol     string                 `json:"protocol"`
	Endpoint     string                 `json:"endpoint"`
	Config       map[string]interface{} `json:"config,omitempty"`
	Dictionaries []DictionaryMapping    `json:"dictionaries,omitempty"`
	FieldMapping []FieldMapping         `json:"fieldMapping,omitempty"`
	CreatedAt    time.Time              `json:"createdAt"`
	UpdatedAt    time.Time              `json:"updatedAt"`
}

type DictionaryMapping struct {
	Name       string            `json:"name"`
	KeyField   string            `json:"keyField"`
	LabelField string            `json:"labelField"`
	ValueField string            `json:"valueField"`
	Entries    []DictionaryEntry `json:"entries,omitempty"`
}

type DictionaryEntry struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type FieldMapping struct {
	Source     string      `json:"source"`
	Target     string      `json:"target"`
	Entity     string      `json:"entity,omitempty"`
	Dictionary string      `json:"dictionary,omitempty"`
	Required   bool        `json:"required,omitempty"`
	Default    interface{} `json:"default,omitempty"`
	Type       string      `json:"type,omitempty"`
}

type DataSourceSyncRequest struct {
	Payload interface{}            `json:"payload,omitempty"`
	Params  map[string]interface{} `json:"params,omitempty"`
	DryRun  bool                   `json:"dryRun,omitempty"`
}

type DataSourceSyncResult struct {
	Rows           []MappedRecord  `json:"rows"`
	Patients       []Patient       `json:"patients"`
	Visits         []ClinicalVisit `json:"visits"`
	MedicalRecords []MedicalRecord `json:"medicalRecords"`
	Created        int             `json:"created"`
	Updated        int             `json:"updated"`
	Errors         []string        `json:"errors,omitempty"`
}

type MappedRecord struct {
	Raw      map[string]interface{}            `json:"raw,omitempty"`
	Entities map[string]map[string]interface{} `json:"entities"`
}

type Report struct {
	ID          string         `json:"id"`
	Type        string         `json:"type,omitempty"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Widgets     []ReportWidget `json:"widgets,omitempty"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
}

type ReportWidget struct {
	ID         string                 `json:"id"`
	ReportID   string                 `json:"reportId"`
	Type       string                 `json:"type"`
	Title      string                 `json:"title"`
	Query      map[string]interface{} `json:"query,omitempty"`
	VisSpec    map[string]interface{} `json:"visSpec,omitempty"`
	DataSource string                 `json:"dataSource,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
}

type EvaluationComplaint struct {
	ID                    string                 `json:"id"`
	Source                string                 `json:"source"`
	Kind                  string                 `json:"kind"`
	PatientID             string                 `json:"patientId,omitempty"`
	PatientName           string                 `json:"patientName,omitempty"`
	PatientPhone          string                 `json:"patientPhone,omitempty"`
	VisitID               string                 `json:"visitId,omitempty"`
	Channel               string                 `json:"channel,omitempty"`
	Title                 string                 `json:"title"`
	Content               string                 `json:"content"`
	Rating                int                    `json:"rating,omitempty"`
	Category              string                 `json:"category,omitempty"`
	Authenticity          string                 `json:"authenticity"`
	Status                string                 `json:"status"`
	ResponsibleDepartment string                 `json:"responsibleDepartment,omitempty"`
	ResponsiblePerson     string                 `json:"responsiblePerson,omitempty"`
	AuditOpinion          string                 `json:"auditOpinion,omitempty"`
	HandlingOpinion       string                 `json:"handlingOpinion,omitempty"`
	RectificationMeasures string                 `json:"rectificationMeasures,omitempty"`
	TrackingOpinion       string                 `json:"trackingOpinion,omitempty"`
	RawPayload            map[string]interface{} `json:"rawPayload,omitempty"`
	CreatedBy             string                 `json:"createdBy,omitempty"`
	ArchivedAt            *time.Time             `json:"archivedAt,omitempty"`
	CreatedAt             time.Time              `json:"createdAt"`
	UpdatedAt             time.Time              `json:"updatedAt"`
}

type PatientTag struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Color       string    `json:"color"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type PatientGroup struct {
	ID             string                 `json:"id"`
	Name           string                 `json:"name"`
	Category       string                 `json:"category"`
	Mode           string                 `json:"mode"`
	AssignmentMode string                 `json:"assignmentMode"`
	FollowupPlanID string                 `json:"followupPlanId,omitempty"`
	Rules          map[string]interface{} `json:"rules,omitempty"`
	Permissions    map[string]interface{} `json:"permissions,omitempty"`
	MemberCount    int                    `json:"memberCount"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type IntegrationChannel struct {
	ID            string                 `json:"id"`
	Kind          string                 `json:"kind"`
	Name          string                 `json:"name"`
	Endpoint      string                 `json:"endpoint,omitempty"`
	AppID         string                 `json:"appId,omitempty"`
	CredentialRef string                 `json:"credentialRef,omitempty"`
	Config        map[string]interface{} `json:"config,omitempty"`
	Enabled       bool                   `json:"enabled"`
	CreatedAt     time.Time              `json:"createdAt"`
	UpdatedAt     time.Time              `json:"updatedAt"`
}

type SurveyShareLink struct {
	ID             string                 `json:"id"`
	ProjectID      string                 `json:"projectId,omitempty"`
	FormTemplateID string                 `json:"formTemplateId"`
	Title          string                 `json:"title"`
	Channel        string                 `json:"channel"`
	Token          string                 `json:"token"`
	URL            string                 `json:"url"`
	ExpiresAt      *time.Time             `json:"expiresAt,omitempty"`
	Config         map[string]interface{} `json:"config,omitempty"`
	CreatedAt      time.Time              `json:"createdAt"`
	UpdatedAt      time.Time              `json:"updatedAt"`
}

type SatisfactionProject struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	TargetType           string                 `json:"targetType"`
	FormTemplateID       string                 `json:"formTemplateId"`
	StartDate            string                 `json:"startDate,omitempty"`
	EndDate              string                 `json:"endDate,omitempty"`
	TargetSampleSize     int                    `json:"targetSampleSize"`
	ActualSampleSize     int                    `json:"actualSampleSize"`
	Anonymous            bool                   `json:"anonymous"`
	RequiresVerification bool                   `json:"requiresVerification"`
	Status               string                 `json:"status"`
	Config               map[string]interface{} `json:"config,omitempty"`
	CreatedAt            time.Time              `json:"createdAt"`
	UpdatedAt            time.Time              `json:"updatedAt"`
}

type SurveySubmission struct {
	ID              string                   `json:"id"`
	ProjectID       string                   `json:"projectId,omitempty"`
	ShareID         string                   `json:"shareId,omitempty"`
	FormTemplateID  string                   `json:"formTemplateId"`
	Channel         string                   `json:"channel"`
	PatientID       string                   `json:"patientId,omitempty"`
	VisitID         string                   `json:"visitId,omitempty"`
	Anonymous       bool                     `json:"anonymous"`
	Status          string                   `json:"status"`
	QualityStatus   string                   `json:"qualityStatus"`
	QualityReason   string                   `json:"qualityReason,omitempty"`
	StartedAt       string                   `json:"startedAt,omitempty"`
	SubmittedAt     time.Time                `json:"submittedAt"`
	DurationSeconds int                      `json:"durationSeconds"`
	IPAddress       string                   `json:"ipAddress,omitempty"`
	UserAgent       string                   `json:"userAgent,omitempty"`
	Answers         map[string]interface{}   `json:"answers,omitempty"`
	AnswerItems     []SurveySubmissionAnswer `json:"answerItems,omitempty"`
	CreatedAt       time.Time                `json:"createdAt"`
	UpdatedAt       time.Time                `json:"updatedAt"`
}

type SurveySubmissionAnswer struct {
	ID            string      `json:"id"`
	SubmissionID  string      `json:"submissionId"`
	QuestionID    string      `json:"questionId"`
	QuestionLabel string      `json:"questionLabel"`
	QuestionType  string      `json:"questionType"`
	Answer        interface{} `json:"answer"`
	Score         *float64    `json:"score,omitempty"`
	CreatedAt     time.Time   `json:"createdAt"`
}

type SatisfactionIndicator struct {
	ID                string    `json:"id"`
	ProjectID         string    `json:"projectId,omitempty"`
	TargetType        string    `json:"targetType"`
	Level             int       `json:"level"`
	ParentID          string    `json:"parentId,omitempty"`
	Name              string    `json:"name"`
	ServiceStage      string    `json:"serviceStage,omitempty"`
	ServiceNode       string    `json:"serviceNode,omitempty"`
	QuestionID        string    `json:"questionId,omitempty"`
	Weight            float64   `json:"weight"`
	IncludeTotalScore bool      `json:"includeTotalScore"`
	NationalDimension string    `json:"nationalDimension,omitempty"`
	IncludeNational   bool      `json:"includeNational"`
	Enabled           bool      `json:"enabled"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type SatisfactionIndicatorQuestion struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"projectId,omitempty"`
	IndicatorID    string    `json:"indicatorId"`
	FormTemplateID string    `json:"formTemplateId"`
	QuestionID     string    `json:"questionId"`
	QuestionLabel  string    `json:"questionLabel,omitempty"`
	ScoreDirection string    `json:"scoreDirection"`
	Weight         float64   `json:"weight"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type SatisfactionCleaningRule struct {
	ID        string                 `json:"id"`
	ProjectID string                 `json:"projectId,omitempty"`
	Name      string                 `json:"name"`
	RuleType  string                 `json:"ruleType"`
	Enabled   bool                   `json:"enabled"`
	Config    map[string]interface{} `json:"config,omitempty"`
	Action    string                 `json:"action"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}

type SurveySubmissionAuditLog struct {
	ID           string    `json:"id"`
	SubmissionID string    `json:"submissionId"`
	ProjectID    string    `json:"projectId,omitempty"`
	Action       string    `json:"action"`
	FromStatus   string    `json:"fromStatus,omitempty"`
	ToStatus     string    `json:"toStatus,omitempty"`
	Reason       string    `json:"reason,omitempty"`
	ActorID      string    `json:"actorId,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
}

type SatisfactionIssue struct {
	ID                    string    `json:"id"`
	ProjectID             string    `json:"projectId,omitempty"`
	SubmissionID          string    `json:"submissionId,omitempty"`
	IndicatorID           string    `json:"indicatorId,omitempty"`
	Title                 string    `json:"title"`
	Source                string    `json:"source"`
	ResponsibleDepartment string    `json:"responsibleDepartment,omitempty"`
	ResponsiblePerson     string    `json:"responsiblePerson,omitempty"`
	Severity              string    `json:"severity"`
	Suggestion            string    `json:"suggestion,omitempty"`
	Measure               string    `json:"measure,omitempty"`
	MaterialURLs          []string  `json:"materialUrls,omitempty"`
	VerificationResult    string    `json:"verificationResult,omitempty"`
	Status                string    `json:"status"`
	DueDate               string    `json:"dueDate,omitempty"`
	ClosedAt              string    `json:"closedAt,omitempty"`
	CreatedAt             time.Time `json:"createdAt"`
	UpdatedAt             time.Time `json:"updatedAt"`
}

type SatisfactionIssueEvent struct {
	ID          string    `json:"id"`
	IssueID     string    `json:"issueId"`
	Action      string    `json:"action"`
	FromStatus  string    `json:"fromStatus,omitempty"`
	ToStatus    string    `json:"toStatus,omitempty"`
	Content     string    `json:"content,omitempty"`
	Attachments []string  `json:"attachments,omitempty"`
	ActorID     string    `json:"actorId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
}

type SurveyInterview struct {
	ID        string                 `json:"id"`
	ShareID   string                 `json:"shareId"`
	PatientID string                 `json:"patientId,omitempty"`
	Status    string                 `json:"status"`
	Answers   map[string]interface{} `json:"answers,omitempty"`
	CreatedAt time.Time              `json:"createdAt"`
	UpdatedAt time.Time              `json:"updatedAt"`
}
