package store

import (
	"testing"

	"reporter/internal/domain"
)

func TestFormPublishAndSubmission(t *testing.T) {
	s := InstallOnly()
	form := s.CreateForm(domain.Form{Name: "Intake"})
	if form.ID != "" {
		t.Fatal("expected form creation without database DSN to return no masked memory form")
	}
	if _, err := s.CreateFormVersion("FORM-001", "1", []domain.FormComponent{{ID: "name", Type: "text", Label: "Name", Required: true}}); err == nil {
		t.Fatal("expected form version creation without database DSN to fail")
	}
	if _, err := s.PublishForm("FORM-001"); err == nil {
		t.Fatal("expected form publishing without database DSN to fail")
	}
	if _, err := s.CreateSubmission(domain.Submission{FormID: "FORM-001", SubmitterID: "1", Data: map[string]interface{}{"name": "Ada"}}); err == nil {
		t.Fatal("expected submission creation without database DSN to fail")
	}
	if submissions := s.SubmissionsByForm("FORM-001"); len(submissions) != 0 {
		t.Fatal("expected submissions without database DSN to return no masked memory submissions")
	}
}

func TestDataSourceUpdateAndDelete(t *testing.T) {
	s := InstallOnly()
	source := s.CreateDataSource(domain.DataSource{Name: "HIS", Protocol: "http", Endpoint: "https://his.local"})
	if source.ID != "" {
		t.Fatal("expected data source creation without database DSN to return no masked memory source")
	}
	if _, err := s.UpdateDataSource("DS-001", domain.DataSource{Name: "HIS API", Config: map[string]interface{}{"timeoutMs": 2000}}); err == nil {
		t.Fatal("expected data source update without database DSN to fail")
	}
	if _, err := s.DeleteDataSource("DS-001"); err == nil {
		t.Fatal("expected data source delete without database DSN to fail")
	}
	if _, ok := s.DataSource("DS-001"); ok {
		t.Fatal("expected data source lookup without database DSN to return no masked memory source")
	}
}

func TestPatientSearchAndUpdate(t *testing.T) {
	s := InstallOnly()
	if _, err := s.PatientsStrict(t.Context(), "张三"); err == nil {
		t.Fatal("expected patient search without database DSN to fail")
	}
	if _, err := s.UpdatePatient("P001", domain.Patient{Status: "follow_up", Diagnosis: "高血压复诊"}); err == nil {
		t.Fatal("expected patient update without database DSN to fail")
	}
	if patient, ok := s.Patient("P001"); ok || patient.ID != "" {
		t.Fatal("expected patient lookup without database DSN to return no masked memory record")
	}
}

func TestDatasetSearchAndUpdate(t *testing.T) {
	s := InstallOnly()
	results := s.Datasets("高血压")
	if len(results) != 0 {
		t.Fatal("expected datasets without database DSN to return no masked records")
	}
	if _, ok := s.Dataset("DS001"); ok {
		t.Fatal("expected dataset lookup without database DSN to return no masked record")
	}
	if created := s.CreateDataset(domain.Dataset{Name: "高血压随访研究"}); created.ID != "" {
		t.Fatal("expected dataset creation without database DSN to return no masked record")
	}
	if _, err := s.UpdateDataset("DS001", domain.Dataset{Status: "archived", Owner: "科研办"}); err == nil {
		t.Fatal("expected dataset update without database DSN to fail")
	}
	if _, err := s.DeleteDataset("DS001"); err == nil {
		t.Fatal("expected dataset delete without database DSN to fail")
	}
}

func TestReportQueryAndWidget(t *testing.T) {
	s := InstallOnly()
	if _, err := s.QueryReport("RP001"); err == nil {
		t.Fatal("expected report query without database DSN to fail")
	}
	if _, err := s.AddReportWidget("RP001", domain.ReportWidget{Type: "table", Title: "新增明细"}); err == nil {
		t.Fatal("expected report widget creation without database DSN to fail")
	}
}

func TestCallCenterSeedsAndCreate(t *testing.T) {
	s := InstallOnly()
	if len(s.Seats()) != 0 || len(s.SipEndpoints()) != 0 {
		t.Fatal("expected call center lists without database DSN to return no masked records")
	}
	if endpoint, ok := s.DefaultSipEndpoint(); ok || endpoint.ID != "" {
		t.Fatal("expected default SIP endpoint without database DSN to return no masked record")
	}
	endpoint := s.CreateSipEndpoint(domain.SipEndpoint{Name: "备用网关", WSSURL: "wss://backup.local/ws", Domain: "backup.local", Proxy: "sip:backup.local;transport=wss", Config: map[string]interface{}{"recording": "mixmonitor"}})
	if endpoint.ID != "" {
		t.Fatal("expected SIP endpoint creation without database DSN to return no masked record")
	}
	if _, err := s.UpdateSipEndpoint("SIP001", domain.SipEndpoint{Name: "主备网关"}); err == nil {
		t.Fatal("expected SIP endpoint update without database DSN to fail")
	}
	if _, err := s.DeleteSipEndpoint("SIP001"); err == nil {
		t.Fatal("expected SIP endpoint delete without database DSN to fail")
	}
	storageConfig := s.CreateStorageConfig(domain.StorageConfig{Name: "MinIO 录音", Kind: "s3", Endpoint: "minio.local:9000", Bucket: "recordings", CredentialRef: "secret://storage/minio"})
	if storageConfig.ID != "" {
		t.Fatal("expected storage config creation without database DSN to return no masked record")
	}
	provider := s.CreateModelProvider(domain.ModelProvider{Name: "OpenAI 兼容网关", Kind: "openai-compatible", Mode: "offline", Endpoint: "https://llm.local/v1", Model: "call-analyzer", CredentialRef: "secret://llm/test"})
	if provider.ID != "" {
		t.Fatal("expected model provider creation without database DSN to return no masked record")
	}
	seat := s.CreateSeat(domain.AgentSeat{Name: "新增坐席", UserID: "2", Extension: "8008", SipURI: "sip:8008@call.example.local", Skills: []string{"满意度"}})
	if seat.ID != "" {
		t.Fatal("expected seat creation without database DSN to return no masked record")
	}
	call := s.CreateCall(domain.CallSession{SeatID: "SEAT001", PatientID: "P001", Direction: "outbound", PhoneNumber: "13800010001"})
	if call.ID != "" {
		t.Fatal("expected call creation without database DSN to return no masked record")
	}
	recording := s.CreateRecording(domain.Recording{CallID: call.ID, StorageURI: "file://data/recordings/test.webm", Duration: 12, Filename: "test.webm", MimeType: "audio/webm", SizeBytes: 1024})
	if recording.ID != "" {
		t.Fatal("expected recording creation without database DSN to return no masked record")
	}
	realtime := s.CreateRealtimeAssistSession(domain.RealtimeAssistSession{CallID: call.ID, PatientID: "P001", FormID: "outpatient-satisfaction", ProviderID: "LLM002"})
	if realtime.ID != "" {
		t.Fatal("expected realtime assist creation without database DSN to return no masked record")
	}
	job := s.CreateOfflineAnalysisJob(domain.OfflineAnalysisJob{CallID: call.ID, RecordingID: recording.ID, ProviderID: "LLM001"})
	if job.ID != "" {
		t.Fatal("expected offline analysis job creation without database DSN to return no masked record")
	}
	interview := s.CreateInterview(domain.InterviewSession{PatientID: "P001", FormID: "outpatient-satisfaction", Mode: "chat_call"})
	if interview.ID != "" {
		t.Fatal("expected interview creation without database DSN to return no masked record")
	}
}
