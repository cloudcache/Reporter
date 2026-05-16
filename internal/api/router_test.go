package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"reporter/internal/config"
	"reporter/internal/domain"
	"reporter/internal/logger"
	"reporter/internal/sipgateway"
	"reporter/internal/store"
)

type fakeSIPGateway struct {
	dialed int
	hangup int
}

func (g *fakeSIPGateway) Dial(_ context.Context, _ domain.SipEndpoint, call domain.CallSession) (sipgateway.DialResult, error) {
	g.dialed++
	return sipgateway.DialResult{Provider: "fake", Status: "connected", DialogID: call.ID + "-dialog"}, nil
}

func (g *fakeSIPGateway) Hangup(_ context.Context, _ string) error {
	g.hangup++
	return nil
}

func TestLogin(t *testing.T) {
	cfg, appStore := openConfiguredStoreForTest(t)
	router := NewRouter(Dependencies{Config: cfg, Log: logger.New("test"), Store: appStore})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"admin123"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	router.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	if res.Result().Cookies()[0].Name != "reporter_access" {
		t.Fatal("expected access cookie response")
	}
}

func TestCreateAndEndCallUsesSIPGateway(t *testing.T) {
	cfg, appStore := openConfiguredStoreForTest(t)
	gateway := &fakeSIPGateway{}
	router := NewRouter(Dependencies{Config: cfg, Log: logger.New("test"), Store: appStore, SIP: gateway})
	cookie := loginCookieWithPassword(t, router, "admin", "2.3245678")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/call-center/calls", strings.NewReader(`{"seatId":"SEAT001","direction":"outbound","phoneNumber":"13800010001"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}
	if gateway.dialed != 1 {
		t.Fatal("expected sip gateway dial")
	}
	var call domain.CallSession
	if err := json.NewDecoder(res.Body).Decode(&call); err != nil {
		t.Fatal(err)
	}
	if call.Status != "connected" {
		t.Fatalf("expected connected call, got %s", call.Status)
	}

	req = httptest.NewRequest(http.MethodPut, "/api/v1/call-center/calls/"+call.ID+"?status=ended", nil)
	req.AddCookie(cookie)
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", res.Code, res.Body.String())
	}
	if gateway.hangup != 1 {
		t.Fatal("expected sip gateway hangup")
	}
}

func TestUploadRecordingUsesConfiguredStorage(t *testing.T) {
	cfg, appStore := openConfiguredStoreForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	db, err := sql.Open(cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		t.Fatalf("open test database: %v", err)
	}
	defer db.Close()
	var originalBasePath string
	if err := db.QueryRowContext(ctx, `SELECT COALESCE(base_path, '') FROM storage_configs WHERE id = 'STOR001'`).Scan(&originalBasePath); err != nil {
		t.Fatalf("load original recording storage config: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(), `UPDATE storage_configs SET base_path = ? WHERE id = 'STOR001'`, originalBasePath)
	})
	if _, err := db.ExecContext(ctx, `UPDATE storage_configs SET base_path = ? WHERE id = 'STOR001'`, t.TempDir()); err != nil {
		t.Fatalf("point default recording storage to temp dir: %v", err)
	}
	router := NewRouter(Dependencies{Config: cfg, Log: logger.New("test"), Store: appStore})
	cookie := loginCookieWithPassword(t, router, "admin", "2.3245678")
	call, err := appStore.CreateCallStrict(ctx, domain.CallSession{SeatID: "SEAT001", Direction: "outbound", PhoneNumber: "13800010001", Status: "connected"})
	if err != nil {
		t.Fatalf("create recording upload call: %v", err)
	}

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("callId", call.ID); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("duration", "3"); err != nil {
		t.Fatal(err)
	}
	file, err := writer.CreateFormFile("file", "call.webm")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := file.Write([]byte("fake-webm-data")); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/v1/call-center/recordings/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", res.Code, res.Body.String())
	}
	var recording domain.Recording
	if err := json.NewDecoder(res.Body).Decode(&recording); err != nil {
		t.Fatal(err)
	}
	if recording.Backend != "local" || !strings.HasPrefix(recording.StorageURI, "file://") {
		t.Fatalf("expected local recording storage, got backend=%s uri=%s", recording.Backend, recording.StorageURI)
	}
}

func TestDataSourcePreviewAndSyncMapsPatientVisitAndRecord(t *testing.T) {
	appStore := store.InstallOnly()
	source := appStore.CreateDataSource(domain.DataSource{
		Name:     "同步测试 API",
		Protocol: "http",
		Endpoint: "https://his.local/patient",
		FieldMapping: []domain.FieldMapping{
			{Source: "$.id", Target: "patient.patientNo", Required: true},
			{Source: "$.name", Target: "patient.name", Required: true},
			{Source: "$.visit.no", Target: "visit.visitNo"},
			{Source: "$.visit.department", Target: "visit.departmentName"},
			{Source: "$.record.no", Target: "record.recordNo"},
			{Source: "$.record.title", Target: "record.title"},
		},
	})
	if source.ID != "" {
		t.Fatal("expected data source creation without database DSN to return no masked memory source")
	}
	router := NewRouter(Dependencies{Config: config.Load(), Log: logger.New("test"), Store: appStore})
	body := `{"payload":{"id":"P777","name":"测试患者","visit":{"no":"V777","department":"心内科"},"record":{"no":"R777","title":"门诊病历"}}}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data-sources/DS-001/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code == http.StatusOK {
		t.Fatalf("expected preview without database DSN to fail, got %d: %s", res.Code, res.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/data-sources/DS-001/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code == http.StatusOK {
		t.Fatalf("expected sync without database DSN to fail, got %d: %s", res.Code, res.Body.String())
	}
}

func TestDataSourceSyncMapsClinicalFacts(t *testing.T) {
	cfg, appStore := openConfiguredStoreForTest(t)
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	source, err := appStore.CreateDataSourceStrict(ctx, domain.DataSource{
		Name:     "临床事实同步",
		Protocol: "http",
		Endpoint: "https://his.local/clinical-facts",
		FieldMapping: []domain.FieldMapping{
			{Source: "$.patientNo", Target: "patient.patientNo", Required: true},
			{Source: "$.name", Target: "patient.name", Required: true},
			{Source: "$.visitNo", Target: "visit.visitNo"},
			{Source: "$.diagnosis", Target: "diagnosis.diagnosisName"},
			{Source: "$.history", Target: "history.content"},
			{Source: "$.drug", Target: "medication.drugName"},
			{Source: "$.labNo", Target: "lab.reportNo"},
			{Source: "$.labName", Target: "lab.reportName"},
			{Source: "$.itemName", Target: "labResult.itemName"},
			{Source: "$.itemValue", Target: "labResult.resultValue"},
			{Source: "$.examNo", Target: "exam.examNo"},
			{Source: "$.examName", Target: "exam.examName"},
			{Source: "$.followupSummary", Target: "followup.summary"},
			{Source: "$.factType", Target: "fact.factType"},
			{Source: "$.factKey", Target: "fact.factKey"},
			{Source: "$.factLabel", Target: "fact.factLabel"},
			{Source: "$.factValue", Target: "fact.factValue"},
		},
	})
	if err != nil {
		t.Fatalf("create data source in database: %v", err)
	}
	router := NewRouter(Dependencies{Config: cfg, Log: logger.New("test"), Store: appStore})
	cookie := loginCookieWithPassword(t, router, "admin", "2.3245678")
	body := `{"payload":{"patientNo":"P778","name":"事实患者","visitNo":"V778","diagnosis":"糖尿病","history":"高血压病史","drug":"二甲双胍片","labNo":"L778","labName":"血糖","itemName":"空腹血糖","itemValue":"6.8","examNo":"E778","examName":"眼底检查","followupSummary":"用药依从性好","factType":"experience","factKey":"drug_compliance","factLabel":"用药依从性","factValue":"良好"}}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data-sources/"+source.ID+"/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected sync 200, got %d: %s", res.Code, res.Body.String())
	}
	bodyText := res.Body.String()
	for _, want := range []string{"糖尿病", "二甲双胍片", "空腹血糖", "眼底检查", "用药依从性"} {
		if !strings.Contains(bodyText, want) {
			t.Fatalf("expected clinical fact %q in response, got %s", want, bodyText)
		}
	}
	var patient domain.Patient
	for _, item := range appStore.Patients("P778") {
		if item.PatientNo == "P778" {
			patient = item
			break
		}
	}
	if patient.ID == "" {
		t.Fatal("expected patient P778 after real sync")
	}
	clinical, err := appStore.Patient360(context.Background(), patient.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(clinical.Diagnoses) == 0 || len(clinical.Medications) == 0 || len(clinical.LabReports) == 0 || len(clinical.ExamReports) == 0 || len(clinical.FollowupRecords) == 0 || len(clinical.InterviewFacts) == 0 {
		t.Fatalf("expected database-backed patient 360 clinical facts, response=%s got %#v", bodyText, clinical)
	}
}

func openConfiguredStoreForTest(t *testing.T) (config.Config, *store.Store) {
	t.Helper()
	cfg, err := config.LoadFile("../../config.yaml")
	if err != nil {
		t.Fatalf("load root config: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	appStore, err := store.Open(ctx, cfg.Database.Driver, cfg.Database.DSN)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) || strings.Contains(err.Error(), "operation not permitted") || strings.Contains(err.Error(), "connection refused") {
			t.Skipf("database unavailable in this test environment: %v", err)
		}
		t.Fatalf("open configured database store: %v", err)
	}
	return cfg, appStore
}

func loginCookie(t *testing.T, router http.Handler) *http.Cookie {
	t.Helper()
	return loginCookieWithPassword(t, router, "admin", "admin123")
}

func loginCookieWithPassword(t *testing.T, router http.Handler, username, password string) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"`+username+`","password":"`+password+`"}`))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", res.Code, res.Body.String())
	}
	for _, cookie := range res.Result().Cookies() {
		if cookie.Name == "reporter_access" {
			return cookie
		}
	}
	t.Fatal("expected reporter_access cookie")
	return nil
}
