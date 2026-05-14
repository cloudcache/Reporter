package api

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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
	router := NewRouter(Dependencies{Config: config.Load(), Log: logger.New("test"), Store: store.NewMemoryStore()})
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
	gateway := &fakeSIPGateway{}
	router := NewRouter(Dependencies{Config: config.Load(), Log: logger.New("test"), Store: store.NewMemoryStore(), SIP: gateway})
	cookie := loginCookie(t, router)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/call-center/calls", strings.NewReader(`{"seatId":"SEAT001","patientId":"P001","direction":"outbound","phoneNumber":"13800010001"}`))
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
	appStore := store.NewMemoryStore()
	if _, err := appStore.UpdateStorageConfig("STOR001", domain.StorageConfig{Name: "测试本地存储", Kind: "local", BasePath: t.TempDir()}); err != nil {
		t.Fatal(err)
	}
	router := NewRouter(Dependencies{Config: config.Load(), Log: logger.New("test"), Store: appStore})
	cookie := loginCookie(t, router)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("callId", "CALL001"); err != nil {
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
	appStore := store.NewMemoryStore()
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
	router := NewRouter(Dependencies{Config: config.Load(), Log: logger.New("test"), Store: appStore})
	cookie := loginCookie(t, router)
	body := `{"payload":{"id":"P777","name":"测试患者","visit":{"no":"V777","department":"心内科"},"record":{"no":"R777","title":"门诊病历"}}}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/data-sources/"+source.ID+"/preview", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected preview 200, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "patient.patientNo") {
		t.Fatalf("expected mapped preview columns, got %s", res.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/v1/data-sources/"+source.ID+"/sync", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(cookie)
	res = httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("expected sync 200, got %d: %s", res.Code, res.Body.String())
	}
	if !strings.Contains(res.Body.String(), "测试患者") || !strings.Contains(res.Body.String(), "门诊病历") {
		t.Fatalf("expected synced patient and record, got %s", res.Body.String())
	}
}

func loginCookie(t *testing.T, router http.Handler) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"username":"admin","password":"admin123"}`))
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
