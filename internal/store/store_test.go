package store

import (
	"testing"

	"reporter/internal/domain"
)

func TestFormPublishAndSubmission(t *testing.T) {
	s := NewTestStore()
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
	s := NewTestStore()
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
	s := NewTestStore()
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
	s := NewTestStore()
	results := s.Datasets("高血压")
	if len(results) != 1 || results[0].ID != "DS001" {
		t.Fatal("expected dataset search to find DS001")
	}
	updated, err := s.UpdateDataset("DS001", domain.Dataset{Status: "archived", Owner: "科研办"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Status != "archived" || updated.Owner != "科研办" || updated.RecordCount != 0 {
		t.Fatal("dataset update did not persist")
	}
	deleted, err := s.DeleteDataset("DS001")
	if err != nil {
		t.Fatal(err)
	}
	if deleted.ID != "DS001" {
		t.Fatal("deleted dataset mismatch")
	}
	if _, ok := s.Dataset("DS001"); ok {
		t.Fatal("expected dataset to be deleted")
	}
}

func TestReportQueryAndWidget(t *testing.T) {
	s := NewTestStore()
	if _, err := s.QueryReport("RP001"); err == nil {
		t.Fatal("expected report query without database DSN to fail")
	}
	if _, err := s.AddReportWidget("RP001", domain.ReportWidget{Type: "table", Title: "新增明细"}); err == nil {
		t.Fatal("expected report widget creation without database DSN to fail")
	}
}

func TestCallCenterSeedsAndCreate(t *testing.T) {
	s := NewTestStore()
	if len(s.Seats()) == 0 || len(s.SipEndpoints()) == 0 {
		t.Fatal("expected seeded seats and sip endpoints")
	}
	endpoint := s.CreateSipEndpoint(domain.SipEndpoint{Name: "备用网关", WSSURL: "wss://backup.local/ws", Domain: "backup.local", Proxy: "sip:backup.local;transport=wss", Config: map[string]interface{}{"recording": "mixmonitor"}})
	updatedEndpoint, err := s.UpdateSipEndpoint(endpoint.ID, domain.SipEndpoint{Name: "主备网关", WSSURL: "wss://backup.local/ws", Domain: "backup.local", Proxy: "sip:backup.local;transport=wss", Config: map[string]interface{}{"recording": "siprec"}})
	if err != nil || updatedEndpoint.Name != "主备网关" {
		t.Fatal("expected sip endpoint update")
	}
	if _, err := s.DeleteSipEndpoint(endpoint.ID); err != nil {
		t.Fatal(err)
	}
	storageConfig := s.CreateStorageConfig(domain.StorageConfig{Name: "MinIO 录音", Kind: "s3", Endpoint: "minio.local:9000", Bucket: "recordings", CredentialRef: "secret://storage/minio"})
	updatedStorageConfig, err := s.UpdateStorageConfig(storageConfig.ID, domain.StorageConfig{Name: "对象存储录音", Kind: "s3", Endpoint: "minio.local:9000", Bucket: "recordings", CredentialRef: "secret://storage/prod"})
	if err != nil || updatedStorageConfig.Name != "对象存储录音" {
		t.Fatal("expected storage config update")
	}
	recordingConfig := s.CreateRecordingConfig(domain.RecordingConfig{Name: "默认录音", Mode: "server", StorageConfigID: storageConfig.ID, Format: "wav", RetentionDays: 365, AutoStart: true, AutoStop: true})
	updatedRecordingConfig, err := s.UpdateRecordingConfig(recordingConfig.ID, domain.RecordingConfig{Name: "质控录音", Mode: "diago", StorageConfigID: storageConfig.ID, Format: "wav", RetentionDays: 180, AutoStart: true, AutoStop: true})
	if err != nil || updatedRecordingConfig.RetentionDays != 180 {
		t.Fatal("expected recording config update")
	}
	if _, err := s.DeleteRecordingConfig(recordingConfig.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.DeleteStorageConfig(storageConfig.ID); err != nil {
		t.Fatal(err)
	}
	provider := s.CreateModelProvider(domain.ModelProvider{Name: "OpenAI 兼容网关", Kind: "openai-compatible", Mode: "offline", Endpoint: "https://llm.local/v1", Model: "call-analyzer", CredentialRef: "secret://llm/test"})
	updatedProvider, err := s.UpdateModelProvider(provider.ID, domain.ModelProvider{Name: "院内模型", Kind: "openai-compatible", Mode: "both", Endpoint: "https://llm.local/v1", Model: "call-analyzer-v2", CredentialRef: "secret://llm/prod", Config: map[string]interface{}{"audio_analysis": true}})
	if err != nil || updatedProvider.Model != "call-analyzer-v2" || updatedProvider.CredentialRef == "" || updatedProvider.Mode != "both" {
		t.Fatal("expected model provider update")
	}
	if _, err := s.DeleteModelProvider(provider.ID); err != nil {
		t.Fatal(err)
	}
	seat := s.CreateSeat(domain.AgentSeat{Name: "新增坐席", UserID: "2", Extension: "8008", SipURI: "sip:8008@call.example.local", Skills: []string{"满意度"}})
	updatedSeat, err := s.UpdateSeat(seat.ID, domain.AgentSeat{Name: "质控坐席", UserID: "2", Extension: "8008", SipURI: "sip:8008@call.example.local", Status: "available", Skills: []string{"满意度", "质控"}})
	if err != nil {
		t.Fatal(err)
	}
	if updatedSeat.Name != "质控坐席" || len(updatedSeat.Skills) != 2 {
		t.Fatal("expected seat extension update to persist skills")
	}
	deletedSeat, err := s.DeleteSeat(seat.ID)
	if err != nil || deletedSeat.ID != seat.ID {
		t.Fatal("expected seat to be deleted")
	}
	call := s.CreateCall(domain.CallSession{SeatID: "SEAT001", PatientID: "P001", Direction: "outbound", PhoneNumber: "13800010001"})
	if call.ID == "" || call.Status != "dialing" {
		t.Fatal("expected call to be created")
	}
	ended, err := s.UpdateCall(call.ID, domain.CallSession{Status: "ended"})
	if err != nil || ended.Status != "ended" {
		t.Fatal("expected call to be ended")
	}
	recording := s.CreateRecording(domain.Recording{CallID: call.ID, StorageURI: "file://data/recordings/test.webm", Duration: 12, Filename: "test.webm", MimeType: "audio/webm", SizeBytes: 1024})
	if recording.ID == "" || recording.Status != "ready" || recording.Source != "browser" {
		t.Fatal("expected recording to be created")
	}
	updatedCalls := s.Calls()
	foundLinkedRecording := false
	for _, updatedCall := range updatedCalls {
		if updatedCall.ID == call.ID && updatedCall.RecordingID == recording.ID {
			foundLinkedRecording = true
		}
	}
	if !foundLinkedRecording {
		t.Fatal("expected recording to be linked to call")
	}
	realtime := s.CreateRealtimeAssistSession(domain.RealtimeAssistSession{CallID: call.ID, PatientID: "P001", FormID: "outpatient-satisfaction", ProviderID: "LLM002"})
	updatedRealtime, err := s.AddRealtimeTranscript(realtime.ID, domain.RealtimeTranscript{Speaker: "patient", Text: "医生解释很清楚", IsFinal: true}, map[string]interface{}{"doctor_communication": "满意"})
	if err != nil || updatedRealtime.FormDraft["doctor_communication"] != "满意" {
		t.Fatal("expected realtime transcript to update form draft")
	}
	job := s.CreateOfflineAnalysisJob(domain.OfflineAnalysisJob{CallID: call.ID, RecordingID: recording.ID, ProviderID: "LLM001"})
	if job.ID == "" || job.Status != "queued" {
		t.Fatal("expected offline analysis job to be queued")
	}
	interview := s.CreateInterview(domain.InterviewSession{PatientID: "P001", FormID: "outpatient-satisfaction", Mode: "chat_call"})
	if interview.ID == "" || interview.Status != "draft" {
		t.Fatal("expected interview to be created")
	}
}
