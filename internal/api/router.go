package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"reporter/internal/auth"
	"reporter/internal/config"
	"reporter/internal/datamapping"
	"reporter/internal/domain"
	installer "reporter/internal/install"
	"reporter/internal/rbac"
	"reporter/internal/recordingstorage"
	"reporter/internal/sipgateway"
	"reporter/internal/store"
)

type Dependencies struct {
	Config config.Config
	Log    zerolog.Logger
	Store  *store.MemoryStore
	SIP    sipgateway.Gateway
}

type Server struct {
	cfg           config.Config
	log           zerolog.Logger
	store         *store.MemoryStore
	authz         *rbac.Authorizer
	sip           sipgateway.Gateway
	authMu        sync.Mutex
	loginFailures map[string]int
	refreshHits   map[string][]time.Time
	captchas      map[string]captchaChallenge
}

type captchaChallenge struct {
	Answer    string
	ExpiresAt time.Time
}

func NewRouter(deps Dependencies) http.Handler {
	authz, err := rbac.New()
	if err != nil {
		deps.Log.Fatal().Err(err).Msg("rbac init failed")
	}
	for _, role := range deps.Store.Roles() {
		if err := authz.SetRolePermissions(role.ID, role.Permissions); err != nil {
			deps.Log.Fatal().Err(err).Msg("rbac policy load failed")
		}
	}
	for _, user := range deps.Store.Users() {
		for _, role := range user.Roles {
			_ = authz.AddRoleForUser(user.ID, role)
		}
	}

	s := &Server{
		cfg:           deps.Config,
		log:           deps.Log,
		store:         deps.Store,
		authz:         authz,
		sip:           deps.SIP,
		loginFailures: map[string]int{},
		refreshHits:   map[string][]time.Time{},
		captchas:      map[string]captchaChallenge{},
	}
	if s.sip == nil {
		s.sip = sipgateway.NewDiagoGateway(nil)
	}
	r := chi.NewRouter()
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:4321", "http://127.0.0.1:4321"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(s.trace)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/install/status", s.installStatus)
		r.Post("/install/test-db", s.installTestDB)
		r.Post("/install", s.installRun)
		r.Post("/auth/login", s.login)
		r.Get("/auth/captcha", s.captcha)
		r.Post("/auth/refresh", s.refresh)
		r.Post("/auth/logout", s.logout)
		r.Get("/public/survey/{token}", s.publicSurvey)
		r.Post("/public/survey/{token}/verify", s.verifyPublicSurveyPatient)
		r.Post("/public/survey/{token}/submissions", s.createPublicSurveySubmission)
		r.Get("/public/survey/{token}/events", s.publicSurveyEvents)
		r.Post("/public/survey/{token}/interviews", s.createPublicSurveyInterview)
		r.Get("/public/survey-interviews/{id}/events", s.publicSurveyInterviewEvents)

		r.Group(func(r chi.Router) {
			r.Use(s.requireAuth)
			r.Get("/auth/me", s.me)
			r.Put("/auth/me", s.updateMe)
			r.Put("/auth/password", s.changePassword)
			r.Get("/users", s.withPermission("/api/v1/users", "read", s.listUsers))
			r.Post("/users", s.withPermission("/api/v1/users", "create", s.createUser))
			r.Get("/users/{id}", s.withPermission("/api/v1/users", "read", s.getUser))
			r.Put("/users/{id}", s.withPermission("/api/v1/users", "update", s.updateUser))
			r.Delete("/users/{id}", s.withPermission("/api/v1/users", "delete", s.deleteUser))
			r.Get("/patients", s.withPermission("/api/v1/patients", "read", s.listPatients))
			r.Post("/patients", s.withPermission("/api/v1/patients", "create", s.createPatient))
			r.Get("/patients/{id}", s.withPermission("/api/v1/patients", "read", s.getPatient))
			r.Put("/patients/{id}", s.withPermission("/api/v1/patients", "update", s.updatePatient))
			r.Get("/patients/{id}/360", s.withPermission("/api/v1/patients", "read", s.getPatient360))
			r.Get("/patients/{id}/visits", s.withPermission("/api/v1/patients", "read", s.listPatientVisits))
			r.Get("/patients/{id}/medical-records", s.withPermission("/api/v1/patients", "read", s.listPatientMedicalRecords))
			r.Get("/patients/{id}/diagnoses", s.withPermission("/api/v1/patients", "read", s.listPatientDiagnoses))
			r.Get("/patients/{id}/histories", s.withPermission("/api/v1/patients", "read", s.listPatientHistories))
			r.Get("/patients/{id}/medications", s.withPermission("/api/v1/patients", "read", s.listPatientMedications))
			r.Get("/patients/{id}/labs", s.withPermission("/api/v1/patients", "read", s.listPatientLabs))
			r.Get("/patients/{id}/exams", s.withPermission("/api/v1/patients", "read", s.listPatientExams))
			r.Get("/patients/{id}/surgeries", s.withPermission("/api/v1/patients", "read", s.listPatientSurgeries))
			r.Get("/patients/{id}/followup-records", s.withPermission("/api/v1/patients", "read", s.listPatientFollowupRecords))
			r.Get("/patients/{id}/interview-facts", s.withPermission("/api/v1/patients", "read", s.listPatientInterviewFacts))
			r.Get("/patient-tags", s.withPermission("/api/v1/patients", "read", s.listPatientTags))
			r.Post("/patient-tags", s.withPermission("/api/v1/patients", "update", s.upsertPatientTag))
			r.Get("/patient-groups", s.withPermission("/api/v1/patients", "read", s.listPatientGroups))
			r.Post("/patient-groups", s.withPermission("/api/v1/patients", "update", s.upsertPatientGroup))
			r.Put("/patient-groups/{id}", s.withPermission("/api/v1/patients", "update", s.upsertPatientGroup))
			r.Put("/patient-groups/{id}/members", s.withPermission("/api/v1/patients", "update", s.assignPatientGroupMembers))
			r.Get("/datasets", s.withPermission("/api/v1/datasets", "read", s.listDatasets))
			r.Post("/datasets", s.withPermission("/api/v1/datasets", "create", s.createDataset))
			r.Get("/datasets/{id}", s.withPermission("/api/v1/datasets", "read", s.getDataset))
			r.Put("/datasets/{id}", s.withPermission("/api/v1/datasets", "update", s.updateDataset))
			r.Delete("/datasets/{id}", s.withPermission("/api/v1/datasets", "delete", s.deleteDataset))
			r.Get("/roles", s.withPermission("/api/v1/roles", "read", s.listRoles))
			r.Post("/roles", s.withPermission("/api/v1/roles", "create", s.createRole))
			r.Put("/roles/{id}/permissions", s.withPermission("/api/v1/roles", "update", s.updateRolePermissions))

			r.Get("/forms", s.withPermission("/api/v1/forms", "read", s.listForms))
			r.Get("/form-library", s.withPermission("/api/v1/forms", "read", s.formLibrary))
			r.Post("/form-library", s.withPermission("/api/v1/forms", "update", s.upsertFormLibraryItem))
			r.Put("/form-library/{id}", s.withPermission("/api/v1/forms", "update", s.upsertFormLibraryItem))
			r.Delete("/form-library/{id}", s.withPermission("/api/v1/forms", "update", s.deleteFormLibraryItem))
			r.Post("/forms", s.withPermission("/api/v1/forms", "create", s.createForm))
			r.Post("/forms/{id}/versions", s.withPermission("/api/v1/forms", "update", s.createFormVersion))
			r.Post("/forms/{id}/publish", s.withPermission("/api/v1/forms", "publish", s.publishForm))
			r.Post("/forms/{id}/submissions", s.withPermission("/api/v1/forms", "submit", s.createSubmission))
			r.Get("/forms/{id}/submissions", s.withPermission("/api/v1/forms", "read", s.listSubmissions))
			r.Get("/submissions/{id}", s.withPermission("/api/v1/forms", "read", s.getSubmission))

			r.Get("/data-sources", s.withPermission("/api/v1/data-sources", "read", s.listDataSources))
			r.Post("/data-sources", s.withPermission("/api/v1/data-sources", "create", s.createDataSource))
			r.Get("/data-sources/{id}", s.withPermission("/api/v1/data-sources", "read", s.getDataSource))
			r.Put("/data-sources/{id}", s.withPermission("/api/v1/data-sources", "update", s.updateDataSource))
			r.Delete("/data-sources/{id}", s.withPermission("/api/v1/data-sources", "delete", s.deleteDataSource))
			r.Post("/data-sources/{id}/test", s.withPermission("/api/v1/data-sources", "test", s.testDataSource))
			r.Post("/data-sources/{id}/preview", s.withPermission("/api/v1/data-sources", "preview", s.previewDataSource))
			r.Post("/data-sources/{id}/sync", s.withPermission("/api/v1/data-sources", "update", s.syncDataSource))
			r.Get("/integration-channels", s.withPermission("/api/v1/system", "read", s.listIntegrationChannels))
			r.Post("/integration-channels", s.withPermission("/api/v1/system", "update", s.upsertIntegrationChannel))
			r.Get("/satisfaction/projects", s.withPermission("/api/v1/forms", "read", s.listSatisfactionProjects))
			r.Post("/satisfaction/projects", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionProject))
			r.Put("/satisfaction/projects/{id}", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionProject))
			r.Get("/satisfaction/submissions", s.withPermission("/api/v1/forms", "read", s.listSurveySubmissions))
			r.Get("/satisfaction/submissions/{id}", s.withPermission("/api/v1/forms", "read", s.getSurveySubmission))
			r.Put("/satisfaction/submissions/{id}/quality", s.withPermission("/api/v1/forms", "update", s.updateSurveySubmissionQuality))
			r.Get("/satisfaction/stats", s.withPermission("/api/v1/forms", "read", s.satisfactionStats))
			r.Get("/satisfaction/indicators", s.withPermission("/api/v1/forms", "read", s.listSatisfactionIndicators))
			r.Post("/satisfaction/indicators", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionIndicator))
			r.Put("/satisfaction/indicators/{id}", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionIndicator))
			r.Get("/satisfaction/indicator-questions", s.withPermission("/api/v1/forms", "read", s.listSatisfactionIndicatorQuestions))
			r.Post("/satisfaction/indicator-questions", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionIndicatorQuestion))
			r.Put("/satisfaction/indicator-questions/{id}", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionIndicatorQuestion))
			r.Get("/satisfaction/indicator-scores", s.withPermission("/api/v1/forms", "read", s.listSatisfactionIndicatorScores))
			r.Get("/satisfaction/cleaning-rules", s.withPermission("/api/v1/forms", "read", s.listSatisfactionCleaningRules))
			r.Post("/satisfaction/cleaning-rules", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionCleaningRule))
			r.Put("/satisfaction/cleaning-rules/{id}", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionCleaningRule))
			r.Get("/satisfaction/issues", s.withPermission("/api/v1/forms", "read", s.listSatisfactionIssues))
			r.Post("/satisfaction/issues", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionIssue))
			r.Put("/satisfaction/issues/{id}", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionIssue))
			r.Get("/satisfaction/issues/{id}/events", s.withPermission("/api/v1/forms", "read", s.listSatisfactionIssueEvents))
			r.Post("/satisfaction/issues/{id}/events", s.withPermission("/api/v1/forms", "update", s.addSatisfactionIssueEvent))
			r.Get("/satisfaction/submissions/{id}/audit-logs", s.withPermission("/api/v1/forms", "read", s.listSurveySubmissionAuditLogs))
			r.Post("/satisfaction/issues/generate", s.withPermission("/api/v1/forms", "update", s.generateSatisfactionIssues))
			r.Get("/survey-share-links", s.withPermission("/api/v1/forms", "read", s.listSurveyShareLinks))
			r.Post("/survey-share-links", s.withPermission("/api/v1/forms", "update", s.createSurveyShareLink))

			r.Get("/reports", s.withPermission("/api/v1/reports", "read", s.listReports))
			r.Post("/reports", s.withPermission("/api/v1/reports", "create", s.createReport))
			r.Get("/reports/{id}", s.withPermission("/api/v1/reports", "read", s.getReport))
			r.Put("/reports/{id}", s.withPermission("/api/v1/reports", "update", s.updateReport))
			r.Post("/reports/{id}/query", s.withPermission("/api/v1/reports", "query", s.queryReport))
			r.Post("/reports/{id}/widgets", s.withPermission("/api/v1/reports", "update", s.addReportWidget))
			r.Get("/evaluation-complaints", s.withPermission("/api/v1/complaints", "read", s.listEvaluationComplaints))
			r.Post("/evaluation-complaints", s.withPermission("/api/v1/complaints", "create", s.createEvaluationComplaint))
			r.Get("/evaluation-complaints/stats", s.withPermission("/api/v1/complaints", "read", s.evaluationComplaintStats))
			r.Get("/evaluation-complaints/{id}", s.withPermission("/api/v1/complaints", "read", s.getEvaluationComplaint))
			r.Put("/evaluation-complaints/{id}", s.withPermission("/api/v1/complaints", "update", s.updateEvaluationComplaint))
			r.Delete("/evaluation-complaints/{id}", s.withPermission("/api/v1/complaints", "delete", s.deleteEvaluationComplaint))
			r.Get("/followup/plans", s.withPermission("/api/v1/followup", "read", s.listFollowupPlans))
			r.Post("/followup/plans", s.withPermission("/api/v1/followup", "create", s.createFollowupPlan))
			r.Put("/followup/plans/{id}", s.withPermission("/api/v1/followup", "update", s.updateFollowupPlan))
			r.Post("/followup/plans/{id}/generate", s.withPermission("/api/v1/followup", "create", s.generateFollowupTasks))
			r.Get("/followup/tasks", s.withPermission("/api/v1/followup", "read", s.listFollowupTasks))
			r.Post("/followup/tasks", s.withPermission("/api/v1/followup", "create", s.createFollowupTask))
			r.Put("/followup/tasks/{id}", s.withPermission("/api/v1/followup", "update", s.updateFollowupTask))
			r.Get("/departments", s.withPermission("/api/v1/system", "read", s.listDepartments))
			r.Get("/dictionaries", s.withPermission("/api/v1/system", "read", s.listDictionaries))
			r.Post("/dictionaries", s.withPermission("/api/v1/system", "create", s.createDictionary))
			r.Put("/dictionaries/{id}", s.withPermission("/api/v1/system", "update", s.updateDictionary))
			r.Get("/call-center/seats", s.withPermission("/api/v1/call-center", "read", s.listSeats))
			r.Post("/call-center/seats", s.withPermission("/api/v1/call-center", "create", s.createSeat))
			r.Get("/call-center/seats/{id}", s.withPermission("/api/v1/call-center", "read", s.getSeat))
			r.Put("/call-center/seats/{id}", s.withPermission("/api/v1/call-center", "update", s.updateSeat))
			r.Delete("/call-center/seats/{id}", s.withPermission("/api/v1/call-center", "delete", s.deleteSeat))
			r.Get("/call-center/sip-endpoints", s.withPermission("/api/v1/call-center", "read", s.listSipEndpoints))
			r.Post("/call-center/sip-endpoints", s.withPermission("/api/v1/call-center", "create", s.createSipEndpoint))
			r.Get("/call-center/sip-endpoints/{id}", s.withPermission("/api/v1/call-center", "read", s.getSipEndpoint))
			r.Put("/call-center/sip-endpoints/{id}", s.withPermission("/api/v1/call-center", "update", s.updateSipEndpoint))
			r.Delete("/call-center/sip-endpoints/{id}", s.withPermission("/api/v1/call-center", "delete", s.deleteSipEndpoint))
			r.Get("/call-center/storage-configs", s.withPermission("/api/v1/call-center", "read", s.listStorageConfigs))
			r.Post("/call-center/storage-configs", s.withPermission("/api/v1/call-center", "create", s.createStorageConfig))
			r.Get("/call-center/storage-configs/{id}", s.withPermission("/api/v1/call-center", "read", s.getStorageConfig))
			r.Put("/call-center/storage-configs/{id}", s.withPermission("/api/v1/call-center", "update", s.updateStorageConfig))
			r.Delete("/call-center/storage-configs/{id}", s.withPermission("/api/v1/call-center", "delete", s.deleteStorageConfig))
			r.Get("/call-center/recording-configs", s.withPermission("/api/v1/call-center", "read", s.listRecordingConfigs))
			r.Post("/call-center/recording-configs", s.withPermission("/api/v1/call-center", "create", s.createRecordingConfig))
			r.Get("/call-center/recording-configs/{id}", s.withPermission("/api/v1/call-center", "read", s.getRecordingConfig))
			r.Put("/call-center/recording-configs/{id}", s.withPermission("/api/v1/call-center", "update", s.updateRecordingConfig))
			r.Delete("/call-center/recording-configs/{id}", s.withPermission("/api/v1/call-center", "delete", s.deleteRecordingConfig))
			r.Get("/call-center/calls", s.withPermission("/api/v1/call-center", "read", s.listCalls))
			r.Post("/call-center/calls", s.withPermission("/api/v1/call-center", "create", s.createCall))
			r.Put("/call-center/calls/{id}", s.withPermission("/api/v1/call-center", "update", s.updateCall))
			r.Get("/call-center/recordings", s.withPermission("/api/v1/call-center", "read", s.listRecordings))
			r.Post("/call-center/recordings", s.withPermission("/api/v1/call-center", "create", s.createRecording))
			r.Post("/call-center/recordings/upload", s.withPermission("/api/v1/call-center", "create", s.uploadRecording))
			r.Get("/call-center/model-providers", s.withPermission("/api/v1/call-center", "read", s.listModelProviders))
			r.Post("/call-center/model-providers", s.withPermission("/api/v1/call-center", "create", s.createModelProvider))
			r.Get("/call-center/model-providers/{id}", s.withPermission("/api/v1/call-center", "read", s.getModelProvider))
			r.Put("/call-center/model-providers/{id}", s.withPermission("/api/v1/call-center", "update", s.updateModelProvider))
			r.Delete("/call-center/model-providers/{id}", s.withPermission("/api/v1/call-center", "delete", s.deleteModelProvider))
			r.Get("/call-center/analyses", s.withPermission("/api/v1/call-center", "read", s.listAnalyses))
			r.Get("/call-center/realtime-assist", s.withPermission("/api/v1/call-center", "read", s.listRealtimeAssistSessions))
			r.Post("/call-center/realtime-assist", s.withPermission("/api/v1/call-center", "create", s.createRealtimeAssistSession))
			r.Post("/call-center/realtime-assist/{id}/transcript", s.withPermission("/api/v1/call-center", "update", s.addRealtimeTranscript))
			r.Get("/call-center/offline-analysis-jobs", s.withPermission("/api/v1/call-center", "read", s.listOfflineAnalysisJobs))
			r.Post("/call-center/offline-analysis-jobs", s.withPermission("/api/v1/call-center", "create", s.createOfflineAnalysisJob))
			r.Get("/call-center/interviews", s.withPermission("/api/v1/call-center", "read", s.listInterviews))
			r.Post("/call-center/interviews", s.withPermission("/api/v1/call-center", "create", s.createInterview))
			r.Get("/audit-logs", s.withPermission("/api/v1/audit-logs", "read", s.listAuditLogs))
		})
	})
	return r
}

func (s *Server) trace(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceID := r.Header.Get("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.NewString()
		}
		w.Header().Set("X-Trace-Id", traceID)
		next.ServeHTTP(w, r.WithContext(withTraceID(r.Context(), traceID)))
	})
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		CaptchaID     string `json:"captchaId"`
		CaptchaAnswer string `json:"captchaAnswer"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Username = strings.TrimSpace(req.Username)
	key := loginFailureKey(r, req.Username)
	if s.captchaRequired(key) && !s.verifyCaptcha(req.CaptchaID, req.CaptchaAnswer) {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"message": "请输入正确验证码", "captchaRequired": true})
		return
	}
	user, ok := s.store.UserByUsername(req.Username)
	if !ok || !auth.VerifyPassword(req.Password, user.PasswordHash) {
		failures := s.recordLoginFailure(key)
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{"message": "用户名或密码错误", "captchaRequired": failures >= 2})
		return
	}
	access, err := auth.IssueToken(s.cfg.Auth.JWTSecret, auth.Claims{Subject: user.ID, Username: user.Username, Roles: user.Roles}, s.cfg.Auth.AccessTokenTTL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	refresh, err := auth.IssueToken(s.cfg.Auth.JWTSecret, auth.Claims{Subject: user.ID, Username: user.Username, Roles: user.Roles}, s.cfg.Auth.RefreshTokenTTL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.resetLoginFailure(key)
	setAuthCookie(w, "reporter_access", access, "/api/v1", s.cfg.Auth.AccessTokenTTL)
	http.SetCookie(w, &http.Cookie{Name: "reporter_refresh", Value: refresh, Path: "/api/v1/auth", HttpOnly: true, SameSite: http.SameSiteLaxMode, MaxAge: int(s.cfg.Auth.RefreshTokenTTL.Seconds())})
	s.audit(r, user.ID, "auth.login", "auth", nil, map[string]string{"username": user.Username})
	writeJSON(w, http.StatusOK, map[string]interface{}{"user": user, "accessToken": access})
}

func (s *Server) captcha(w http.ResponseWriter, r *http.Request) {
	username := strings.TrimSpace(r.URL.Query().Get("username"))
	key := loginFailureKey(r, username)
	if !s.captchaRequired(key) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"required": false})
		return
	}
	left := rand.Intn(8) + 2
	right := rand.Intn(8) + 2
	id := uuid.NewString()
	s.authMu.Lock()
	s.captchas[id] = captchaChallenge{Answer: strconv.Itoa(left + right), ExpiresAt: time.Now().Add(5 * time.Minute)}
	s.authMu.Unlock()
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"required":  true,
		"captchaId": id,
		"question":  strconv.Itoa(left) + " + " + strconv.Itoa(right) + " = ?",
	})
}

func (s *Server) installStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, installer.CurrentStatus(s.cfg))
}

func (s *Server) installTestDB(w http.ResponseWriter, r *http.Request) {
	var req installer.DatabaseRequest
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := installer.TestDatabase(r.Context(), req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"ok": false, "message": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "message": "数据库连接成功"})
}

func (s *Server) installRun(w http.ResponseWriter, r *http.Request) {
	var req installer.Request
	if !decodeJSON(w, r, &req) {
		return
	}
	result, err := installer.Run(r.Context(), s.cfg, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]interface{}{"installed": false, "message": err.Error()})
		return
	}
	if err := s.reloadIdentityFromSQL(r.Context(), firstNonEmpty(req.Database.Driver, "mysql"), installer.BuildDSN(req.Database)); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]interface{}{"installed": true, "message": "安装已完成，但加载管理员账户失败，请重启服务后再登录：" + err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) reloadIdentityFromSQL(ctx context.Context, driver, dsn string) error {
	if err := s.store.LoadIdentityFromSQL(ctx, driver, dsn); err != nil {
		return err
	}
	authz, err := rbac.New()
	if err != nil {
		return err
	}
	for _, role := range s.store.Roles() {
		if err := authz.SetRolePermissions(role.ID, role.Permissions); err != nil {
			return err
		}
	}
	for _, user := range s.store.Users() {
		for _, role := range user.Roles {
			_ = authz.AddRoleForUser(user.ID, role)
		}
	}
	s.authz = authz
	return nil
}

func (s *Server) refresh(w http.ResponseWriter, r *http.Request) {
	if !s.allowTokenRefresh(tokenRefreshKey(r)) {
		http.Error(w, "refresh too frequent", http.StatusTooManyRequests)
		return
	}
	cookie, err := r.Cookie("reporter_refresh")
	if err != nil {
		http.Error(w, "refresh token required", http.StatusUnauthorized)
		return
	}
	claims, err := auth.ParseToken(s.cfg.Auth.JWTSecret, cookie.Value)
	if err != nil {
		http.Error(w, "invalid refresh token", http.StatusUnauthorized)
		return
	}
	access, err := auth.IssueToken(s.cfg.Auth.JWTSecret, claims, s.cfg.Auth.AccessTokenTTL)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	setAuthCookie(w, "reporter_access", access, "/api/v1", s.cfg.Auth.AccessTokenTTL)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "accessToken": access})
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	clearAuthCookie(w, "reporter_access", "/api/v1")
	clearAuthCookie(w, "reporter_refresh", "/api/v1/auth")
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) me(w http.ResponseWriter, r *http.Request) {
	user, _ := currentUser(r)
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) updateMe(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	var req struct {
		DisplayName string `json:"displayName"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.DisplayName = strings.TrimSpace(req.DisplayName)
	if req.DisplayName == "" {
		http.Error(w, "displayName required", http.StatusBadRequest)
		return
	}
	before := user
	updated, err := s.store.UpdateUser(user.ID, domain.User{
		Username:    user.Username,
		DisplayName: req.DisplayName,
		Roles:       user.Roles,
	})
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, user.ID, "auth.profile.update", "/api/v1/auth/me", before, updated)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) changePassword(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r)
	if !ok {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	var req struct {
		OldPassword string `json:"oldPassword"`
		NewPassword string `json:"newPassword"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !auth.VerifyPassword(req.OldPassword, user.PasswordHash) {
		http.Error(w, "原密码不正确", http.StatusBadRequest)
		return
	}
	if len(req.NewPassword) < 8 {
		http.Error(w, "新密码至少 8 位", http.StatusBadRequest)
		return
	}
	hash, err := auth.HashPassword(req.NewPassword)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	updated, err := s.store.UpdateUser(user.ID, domain.User{
		Username:     user.Username,
		DisplayName:  user.DisplayName,
		Roles:        user.Roles,
		PasswordHash: hash,
	})
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, user.ID, "auth.password.change", "/api/v1/auth/password", nil, map[string]string{"username": updated.Username})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listUsers(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Users())
}

func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username    string   `json:"username"`
		DisplayName string   `json:"displayName"`
		Password    string   `json:"password"`
		Roles       []string `json:"roles"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	user := s.store.CreateUser(domain.User{Username: req.Username, DisplayName: req.DisplayName, PasswordHash: hash, Roles: req.Roles})
	for _, role := range user.Roles {
		_ = s.authz.AddRoleForUser(user.ID, role)
	}
	s.audit(r, actorID(r), "user.create", "/api/v1/users", nil, user)
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) getUser(w http.ResponseWriter, r *http.Request) {
	user, ok := s.store.UserByID(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) updateUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.UserByID(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var req struct {
		Username    string   `json:"username"`
		DisplayName string   `json:"displayName"`
		Password    string   `json:"password"`
		Roles       []string `json:"roles"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	patch := domain.User{Username: req.Username, DisplayName: req.DisplayName, Roles: req.Roles}
	if req.Password != "" {
		hash, err := auth.HashPassword(req.Password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		patch.PasswordHash = hash
	}
	user, err := s.store.UpdateUser(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	for _, role := range before.Roles {
		_ = s.authz.DeleteRoleForUser(id, role)
	}
	for _, role := range user.Roles {
		_ = s.authz.AddRoleForUser(user.ID, role)
	}
	s.audit(r, actorID(r), "user.update", "/api/v1/users/"+user.ID, before, user)
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) deleteUser(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.UserByID(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	user, err := s.store.DeleteUser(id)
	if err != nil {
		statusError(w, err)
		return
	}
	for _, role := range before.Roles {
		_ = s.authz.DeleteRoleForUser(id, role)
	}
	s.audit(r, actorID(r), "user.delete", "/api/v1/users/"+user.ID, before, nil)
	writeJSON(w, http.StatusOK, user)
}

func (s *Server) listPatients(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Patients(r.URL.Query().Get("q")))
}

func (s *Server) createPatient(w http.ResponseWriter, r *http.Request) {
	var patient domain.Patient
	if !decodeJSON(w, r, &patient) {
		return
	}
	patient = s.store.CreatePatient(patient)
	s.audit(r, actorID(r), "patient.create", "/api/v1/patients/"+patient.ID, nil, patient)
	writeJSON(w, http.StatusCreated, patient)
}

func (s *Server) getPatient(w http.ResponseWriter, r *http.Request) {
	patient, ok := s.store.Patient(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, patient)
}

func (s *Server) updatePatient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.Patient(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch domain.Patient
	if !decodeJSON(w, r, &patch) {
		return
	}
	patient, err := s.store.UpdatePatient(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "patient.update", "/api/v1/patients/"+patient.ID, before, patient)
	writeJSON(w, http.StatusOK, patient)
}

func (s *Server) listPatientVisits(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Visits(chi.URLParam(r, "id")))
}

func (s *Server) listPatientMedicalRecords(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.MedicalRecords(chi.URLParam(r, "id")))
}

func (s *Server) getPatient360(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.Patient360(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) listPatientDiagnoses(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.PatientDiagnoses(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listPatientHistories(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.PatientHistories(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listPatientMedications(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.MedicationOrders(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listPatientLabs(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.LabReports(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listPatientExams(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ExamReports(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listPatientSurgeries(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SurgeryRecords(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listPatientFollowupRecords(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.FollowupRecords(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listPatientInterviewFacts(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.InterviewExtractedFacts(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listPatientTags(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.PatientTags(r.Context())
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) upsertPatientTag(w http.ResponseWriter, r *http.Request) {
	var item domain.PatientTag
	if !decodeJSON(w, r, &item) {
		return
	}
	saved, err := s.store.UpsertPatientTag(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "patient-tag.upsert", "/api/v1/patient-tags/"+saved.ID, nil, saved)
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) listPatientGroups(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.PatientGroups(r.Context())
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) upsertPatientGroup(w http.ResponseWriter, r *http.Request) {
	var item domain.PatientGroup
	if !decodeJSON(w, r, &item) {
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		item.ID = id
	}
	saved, err := s.store.UpsertPatientGroup(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "patient-group.upsert", "/api/v1/patient-groups/"+saved.ID, nil, saved)
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) assignPatientGroupMembers(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PatientIDs []string `json:"patientIds"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.store.AssignPatientGroupMembers(r.Context(), id, req.PatientIDs, actorID(r)); err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "patient-group.members.assign", "/api/v1/patient-groups/"+id+"/members", nil, req)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) listDatasets(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Datasets(r.URL.Query().Get("q")))
}

func (s *Server) createDataset(w http.ResponseWriter, r *http.Request) {
	var dataset domain.Dataset
	if !decodeJSON(w, r, &dataset) {
		return
	}
	dataset = s.store.CreateDataset(dataset)
	s.audit(r, actorID(r), "dataset.create", "/api/v1/datasets/"+dataset.ID, nil, dataset)
	writeJSON(w, http.StatusCreated, dataset)
}

func (s *Server) getDataset(w http.ResponseWriter, r *http.Request) {
	dataset, ok := s.store.Dataset(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, dataset)
}

func (s *Server) updateDataset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.Dataset(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch domain.Dataset
	if !decodeJSON(w, r, &patch) {
		return
	}
	dataset, err := s.store.UpdateDataset(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "dataset.update", "/api/v1/datasets/"+dataset.ID, before, dataset)
	writeJSON(w, http.StatusOK, dataset)
}

func (s *Server) deleteDataset(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.Dataset(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	deleted, err := s.store.DeleteDataset(id)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "dataset.delete", "/api/v1/datasets/"+deleted.ID, before, nil)
	writeJSON(w, http.StatusOK, deleted)
}

func (s *Server) listRoles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Roles())
}

func (s *Server) createRole(w http.ResponseWriter, r *http.Request) {
	var role domain.Role
	if !decodeJSON(w, r, &role) {
		return
	}
	role = s.store.CreateRole(role)
	_ = s.authz.SetRolePermissions(role.ID, role.Permissions)
	s.audit(r, actorID(r), "role.create", "/api/v1/roles", nil, role)
	writeJSON(w, http.StatusCreated, role)
}

func (s *Server) updateRolePermissions(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Permissions []string `json:"permissions"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	role, err := s.store.UpdateRolePermissions(chi.URLParam(r, "id"), req.Permissions)
	if err != nil {
		statusError(w, err)
		return
	}
	_ = s.authz.SetRolePermissions(role.ID, role.Permissions)
	s.audit(r, actorID(r), "role.permissions.update", "/api/v1/roles/"+role.ID, nil, role)
	writeJSON(w, http.StatusOK, role)
}

func (s *Server) listForms(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Forms())
}

func (s *Server) formLibrary(w http.ResponseWriter, r *http.Request) {
	items := s.store.FormLibrary()
	response := map[string][]domain.FormLibraryItem{
		"templates":        {},
		"commonComponents": {},
		"atomicComponents": {},
	}
	for _, item := range items {
		switch item.Kind {
		case "template":
			response["templates"] = append(response["templates"], item)
		case "common":
			response["commonComponents"] = append(response["commonComponents"], item)
		case "atom":
			response["atomicComponents"] = append(response["atomicComponents"], item)
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) upsertFormLibraryItem(w http.ResponseWriter, r *http.Request) {
	var item domain.FormLibraryItem
	if !decodeJSON(w, r, &item) {
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		item.ID = id
	}
	saved := s.store.UpsertFormLibraryItem(item)
	s.audit(r, actorID(r), "form-library.upsert", "/api/v1/form-library/"+saved.ID, nil, saved)
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) deleteFormLibraryItem(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.DeleteFormLibraryItem(chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "form-library.delete", "/api/v1/form-library/"+item.ID, item, nil)
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createForm(w http.ResponseWriter, r *http.Request) {
	var form domain.Form
	if !decodeJSON(w, r, &form) {
		return
	}
	form = s.store.CreateForm(form)
	s.audit(r, actorID(r), "form.create", "/api/v1/forms/"+form.ID, nil, form)
	writeJSON(w, http.StatusCreated, form)
}

func (s *Server) createFormVersion(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Schema []domain.FormComponent `json:"schema"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	version, err := s.store.CreateFormVersion(chi.URLParam(r, "id"), actorID(r), req.Schema)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "form.version.create", "/api/v1/forms/"+version.FormID, nil, version)
	writeJSON(w, http.StatusCreated, version)
}

func (s *Server) publishForm(w http.ResponseWriter, r *http.Request) {
	form, err := s.store.PublishForm(chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "form.publish", "/api/v1/forms/"+form.ID, nil, form)
	writeJSON(w, http.StatusOK, form)
}

func (s *Server) createSubmission(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data map[string]interface{} `json:"data"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	submission, err := s.store.CreateSubmission(domain.Submission{FormID: chi.URLParam(r, "id"), SubmitterID: actorID(r), Data: req.Data})
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "submission.create", "/api/v1/submissions/"+submission.ID, nil, submission)
	writeJSON(w, http.StatusCreated, submission)
}

func (s *Server) listSubmissions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.SubmissionsByForm(chi.URLParam(r, "id")))
}

func (s *Server) getSubmission(w http.ResponseWriter, r *http.Request) {
	submission, ok := s.store.Submission(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, submission)
}

func (s *Server) listDataSources(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.DataSources())
}

func (s *Server) createDataSource(w http.ResponseWriter, r *http.Request) {
	var source domain.DataSource
	if !decodeJSON(w, r, &source) {
		return
	}
	source = s.store.CreateDataSource(source)
	s.audit(r, actorID(r), "data-source.create", "/api/v1/data-sources/"+source.ID, nil, source)
	writeJSON(w, http.StatusCreated, source)
}

func (s *Server) getDataSource(w http.ResponseWriter, r *http.Request) {
	source, ok := s.store.DataSource(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (s *Server) updateDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.DataSource(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch domain.DataSource
	if !decodeJSON(w, r, &patch) {
		return
	}
	source, err := s.store.UpdateDataSource(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "data-source.update", "/api/v1/data-sources/"+source.ID, before, source)
	writeJSON(w, http.StatusOK, source)
}

func (s *Server) deleteDataSource(w http.ResponseWriter, r *http.Request) {
	source, err := s.store.DeleteDataSource(chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "data-source.delete", "/api/v1/data-sources/"+source.ID, source, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": source.ID})
}

func (s *Server) testDataSource(w http.ResponseWriter, r *http.Request) {
	source, ok := s.store.DataSource(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "protocol": source.Protocol, "message": "connector contract validated"})
}

func (s *Server) previewDataSource(w http.ResponseWriter, r *http.Request) {
	source, ok := s.store.DataSource(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	var req domain.DataSourceSyncRequest
	if r.Body != nil && r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	preview, err := datamapping.Preview(source, req.Payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, preview)
}

func (s *Server) syncDataSource(w http.ResponseWriter, r *http.Request) {
	source, ok := s.store.DataSource(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	var req domain.DataSourceSyncRequest
	if r.Body != nil && r.ContentLength != 0 && !decodeJSON(w, r, &req) {
		return
	}
	records, err := datamapping.Transform(source, req.Payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result := domain.DataSourceSyncResult{Rows: records}
	for _, record := range records {
		patientFields := record.Entities["patient"]
		var patient domain.Patient
		if len(patientFields) > 0 {
			patient = datamapping.ApplyPatientFields(patientFields, domain.Patient{})
			if patient.PatientNo != "" {
				patient.SourceRefs = map[string]interface{}{"dataSourceId": source.ID, "protocol": source.Protocol}
				if req.DryRun {
					result.Patients = append(result.Patients, patient)
				} else {
					saved, created := s.store.UpsertPatientByNo(patient)
					if created {
						result.Created++
					} else {
						result.Updated++
					}
					patient = saved
					result.Patients = append(result.Patients, saved)
				}
			}
		}

		visitFields := record.Entities["visit"]
		if len(visitFields) > 0 {
			visit := datamapping.ApplyVisitFields(visitFields, domain.ClinicalVisit{PatientID: patient.ID})
			if visit.PatientID == "" {
				visit.PatientID = patient.ID
			}
			if visit.VisitNo != "" {
				visit.SourceRefs = map[string]interface{}{"dataSourceId": source.ID, "protocol": source.Protocol}
				if req.DryRun {
					result.Visits = append(result.Visits, visit)
				} else {
					saved, created := s.store.UpsertVisitByNo(visit)
					if created {
						result.Created++
					} else {
						result.Updated++
					}
					result.Visits = append(result.Visits, saved)
				}
			}
		}

		recordFields := record.Entities["record"]
		if len(recordFields) > 0 {
			medicalRecord := datamapping.ApplyMedicalRecordFields(recordFields, domain.MedicalRecord{PatientID: patient.ID})
			if medicalRecord.PatientID == "" {
				medicalRecord.PatientID = patient.ID
			}
			if medicalRecord.RecordNo == "" && medicalRecord.StudyUID != "" {
				medicalRecord.RecordNo = medicalRecord.StudyUID
			}
			if medicalRecord.RecordType == "" {
				medicalRecord.RecordType = "external"
			}
			if medicalRecord.Title == "" {
				medicalRecord.Title = firstNonEmpty(medicalRecord.StudyDesc, medicalRecord.DiagnosisName, "外部同步病历")
			}
			if medicalRecord.RecordNo != "" {
				medicalRecord.SourceRefs = map[string]interface{}{"dataSourceId": source.ID, "protocol": source.Protocol}
				if req.DryRun {
					result.MedicalRecords = append(result.MedicalRecords, medicalRecord)
				} else {
					saved, created := s.store.UpsertMedicalRecordByNo(medicalRecord)
					if created {
						result.Created++
					} else {
						result.Updated++
					}
					result.MedicalRecords = append(result.MedicalRecords, saved)
				}
			}
		}
	}
	s.audit(r, actorID(r), "data-source.sync", "/api/v1/data-sources/"+source.ID, nil, result)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) listIntegrationChannels(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.IntegrationChannels(r.Context())
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) upsertIntegrationChannel(w http.ResponseWriter, r *http.Request) {
	var item domain.IntegrationChannel
	if !decodeJSON(w, r, &item) {
		return
	}
	saved, err := s.store.UpsertIntegrationChannel(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "integration-channel.upsert", "/api/v1/integration-channels/"+saved.ID, nil, saved)
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) listSurveyShareLinks(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SurveyShareLinks(r.Context())
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createSurveyShareLink(w http.ResponseWriter, r *http.Request) {
	var item domain.SurveyShareLink
	if !decodeJSON(w, r, &item) {
		return
	}
	created, err := s.store.CreateSurveyShareLink(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "survey-share.create", "/api/v1/survey-share-links/"+created.ID, nil, created)
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) listSatisfactionProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SatisfactionProjects(r.Context())
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) upsertSatisfactionProject(w http.ResponseWriter, r *http.Request) {
	var item domain.SatisfactionProject
	if !decodeJSON(w, r, &item) {
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		item.ID = id
	}
	saved, err := s.store.UpsertSatisfactionProject(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "satisfaction-project.upsert", "/api/v1/satisfaction/projects/"+saved.ID, nil, saved)
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) listSurveySubmissions(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SurveySubmissions(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getSurveySubmission(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.SurveySubmission(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) updateSurveySubmissionQuality(w http.ResponseWriter, r *http.Request) {
	var req struct {
		QualityStatus string `json:"qualityStatus"`
		QualityReason string `json:"qualityReason"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	item, err := s.store.UpdateSurveySubmissionQuality(r.Context(), chi.URLParam(r, "id"), req.QualityStatus, req.QualityReason)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "survey-submission.quality", "/api/v1/satisfaction/submissions/"+item.ID, nil, item)
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) satisfactionStats(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SurveySubmissions(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	stats := map[string]interface{}{
		"total":             len(items),
		"valid":             0,
		"pending":           0,
		"suspicious":        0,
		"byChannel":         map[string]int{},
		"scoreAverage":      0,
		"departmentRanking": map[string]map[string]float64{},
		"indicatorScores":   map[string]map[string]float64{},
		"lowReasons":        map[string]int{},
	}
	var scoreSum float64
	var scoreCount int
	for _, item := range items {
		switch item.QualityStatus {
		case "valid":
			stats["valid"] = stats["valid"].(int) + 1
		case "suspicious":
			stats["suspicious"] = stats["suspicious"].(int) + 1
		default:
			stats["pending"] = stats["pending"].(int) + 1
		}
		byChannel := stats["byChannel"].(map[string]int)
		byChannel[item.Channel]++
		for _, key := range []string{"overall_satisfaction", "recommend_score", "service_matrix"} {
			if score := numericAnswer(item.Answers[key]); score != nil {
				scoreSum += *score
				scoreCount++
			}
		}
		addDepartmentScore(stats["departmentRanking"].(map[string]map[string]float64), item)
		addIndicatorScores(stats["indicatorScores"].(map[string]map[string]float64), item)
		addLowReasons(stats["lowReasons"].(map[string]int), item)
	}
	if scoreCount > 0 {
		stats["scoreAverage"] = scoreSum / float64(scoreCount)
	}
	stats["departmentRanking"] = averageBuckets(stats["departmentRanking"].(map[string]map[string]float64))
	stats["indicatorScores"] = averageBuckets(stats["indicatorScores"].(map[string]map[string]float64))
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) listSatisfactionIndicators(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SatisfactionIndicators(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) upsertSatisfactionIndicator(w http.ResponseWriter, r *http.Request) {
	var item domain.SatisfactionIndicator
	if !decodeJSON(w, r, &item) {
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		item.ID = id
	}
	saved, err := s.store.UpsertSatisfactionIndicator(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) listSatisfactionIndicatorQuestions(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SatisfactionIndicatorQuestions(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) upsertSatisfactionIndicatorQuestion(w http.ResponseWriter, r *http.Request) {
	var item domain.SatisfactionIndicatorQuestion
	if !decodeJSON(w, r, &item) {
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		item.ID = id
	}
	saved, err := s.store.UpsertSatisfactionIndicatorQuestion(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) listSatisfactionIndicatorScores(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SatisfactionIndicatorScores(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listSatisfactionCleaningRules(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SatisfactionCleaningRules(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) upsertSatisfactionCleaningRule(w http.ResponseWriter, r *http.Request) {
	var item domain.SatisfactionCleaningRule
	if !decodeJSON(w, r, &item) {
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		item.ID = id
	}
	saved, err := s.store.UpsertSatisfactionCleaningRule(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) listSatisfactionIssues(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SatisfactionIssues(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) upsertSatisfactionIssue(w http.ResponseWriter, r *http.Request) {
	var item domain.SatisfactionIssue
	if !decodeJSON(w, r, &item) {
		return
	}
	if id := chi.URLParam(r, "id"); id != "" {
		item.ID = id
	}
	saved, err := s.store.UpsertSatisfactionIssue(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) listSatisfactionIssueEvents(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SatisfactionIssueEvents(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) addSatisfactionIssueEvent(w http.ResponseWriter, r *http.Request) {
	var item domain.SatisfactionIssueEvent
	if !decodeJSON(w, r, &item) {
		return
	}
	item.IssueID = chi.URLParam(r, "id")
	item.ActorID = actorID(r)
	saved, err := s.store.AddSatisfactionIssueEvent(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, saved)
}

func (s *Server) listSurveySubmissionAuditLogs(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SurveySubmissionAuditLogs(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) generateSatisfactionIssues(w http.ResponseWriter, r *http.Request) {
	projectID := r.URL.Query().Get("projectId")
	items, err := s.store.SurveySubmissions(r.Context(), projectID)
	if err != nil {
		statusError(w, err)
		return
	}
	created := []domain.SatisfactionIssue{}
	for _, item := range items {
		if low := numericAnswer(item.Answers["overall_satisfaction"]); low != nil && *low <= 3 {
			issue, err := s.store.UpsertSatisfactionIssue(r.Context(), domain.SatisfactionIssue{
				ProjectID:             item.ProjectID,
				SubmissionID:          item.ID,
				Title:                 "低满意度答卷需整改",
				Source:                "low_score",
				ResponsibleDepartment: firstNonEmpty(stringAnswer(item.Answers["department"]), "待分派"),
				Severity:              "high",
				Suggestion:            "结合答卷详情核查服务环节，形成整改措施并复评。",
				Status:                "open",
			})
			if err == nil {
				created = append(created, issue)
			}
		}
	}
	writeJSON(w, http.StatusOK, created)
}

func (s *Server) publicSurvey(w http.ResponseWriter, r *http.Request) {
	share, err := s.store.SurveyShareByToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		statusError(w, err)
		return
	}
	var template domain.FormLibraryItem
	for _, item := range s.store.FormLibrary() {
		if item.ID == share.FormTemplateID {
			template = item
			break
		}
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"share": share, "template": template, "requiresVerification": surveyRequiresVerification(share)})
}

func (s *Server) verifyPublicSurveyPatient(w http.ResponseWriter, r *http.Request) {
	share, err := s.store.SurveyShareByToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		statusError(w, err)
		return
	}
	if !surveyRequiresVerification(share) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"verified": true})
		return
	}
	var req struct {
		Identifier string `json:"identifier"`
		Phone      string `json:"phone"`
		OpenID     string `json:"openId"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	patient, visit, ok := s.findSurveyPatient(req.Identifier, req.Phone)
	if !ok {
		statusError(w, store.ErrNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"verified": true,
		"patient":  patient,
		"visit":    visit,
		"values":   surveyAutoFillValues(patient, visit),
	})
}

func (s *Server) createPublicSurveyInterview(w http.ResponseWriter, r *http.Request) {
	share, err := s.store.SurveyShareByToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		statusError(w, err)
		return
	}
	var req struct {
		PatientID string `json:"patientId"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	interview, err := s.store.CreateSurveyInterview(r.Context(), share.ID, req.PatientID)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, interview)
}

func (s *Server) createPublicSurveySubmission(w http.ResponseWriter, r *http.Request) {
	share, err := s.store.SurveyShareByToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		statusError(w, err)
		return
	}
	var req struct {
		PatientID       string                 `json:"patientId"`
		VisitID         string                 `json:"visitId"`
		Answers         map[string]interface{} `json:"answers"`
		StartedAt       string                 `json:"startedAt"`
		DurationSeconds int                    `json:"durationSeconds"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	template := s.formTemplateByID(share.FormTemplateID)
	submission, err := s.store.CreateSurveySubmission(r.Context(), domain.SurveySubmission{
		ProjectID:       share.ProjectID,
		ShareID:         share.ID,
		FormTemplateID:  share.FormTemplateID,
		Channel:         share.Channel,
		PatientID:       req.PatientID,
		VisitID:         req.VisitID,
		Anonymous:       !surveyRequiresVerification(share),
		Status:          "submitted",
		StartedAt:       req.StartedAt,
		DurationSeconds: req.DurationSeconds,
		IPAddress:       clientIP(r),
		UserAgent:       r.UserAgent(),
		Answers:         req.Answers,
	}, template.Components)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, submission)
}

func (s *Server) publicSurveyEvents(w http.ResponseWriter, r *http.Request) {
	share, err := s.store.SurveyShareByToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		statusError(w, err)
		return
	}
	s.streamSurveyTemplate(w, share.FormTemplateID)
}

func (s *Server) formTemplateByID(id string) domain.FormLibraryItem {
	for _, item := range s.store.FormLibrary() {
		if item.ID == id {
			return item
		}
	}
	return domain.FormLibraryItem{}
}

func (s *Server) publicSurveyInterviewEvents(w http.ResponseWriter, r *http.Request) {
	s.streamSurveyTemplate(w, "")
}

func surveyRequiresVerification(share domain.SurveyShareLink) bool {
	if value, ok := share.Config["allowAnonymous"].(bool); ok {
		return !value
	}
	if value, ok := share.Config["requiresVerification"].(bool); ok {
		return value
	}
	return share.Channel == "wechat" || share.Channel == "sms" || share.Channel == "qq"
}

func (s *Server) findSurveyPatient(identifier, phone string) (domain.Patient, domain.ClinicalVisit, bool) {
	identifier = strings.TrimSpace(strings.ToLower(identifier))
	phone = strings.TrimSpace(phone)
	if identifier == "" || phone == "" {
		return domain.Patient{}, domain.ClinicalVisit{}, false
	}
	for _, patient := range s.store.Patients("") {
		if strings.TrimSpace(patient.Phone) != phone {
			continue
		}
		if strings.EqualFold(patient.ID, identifier) ||
			strings.EqualFold(patient.PatientNo, identifier) ||
			strings.EqualFold(patient.MedicalRecordNo, identifier) {
			return patient, firstVisit(s.store.Visits(patient.ID)), true
		}
		for _, visit := range s.store.Visits(patient.ID) {
			if strings.EqualFold(visit.VisitNo, identifier) || strings.EqualFold(visit.ID, identifier) {
				return patient, visit, true
			}
		}
	}
	return domain.Patient{}, domain.ClinicalVisit{}, false
}

func firstVisit(visits []domain.ClinicalVisit) domain.ClinicalVisit {
	if len(visits) == 0 {
		return domain.ClinicalVisit{}
	}
	return visits[0]
}

func surveyAutoFillValues(patient domain.Patient, visit domain.ClinicalVisit) map[string]interface{} {
	return map[string]interface{}{
		"patient_id":          patient.ID,
		"patient_no":          patient.PatientNo,
		"patient_name":        patient.Name,
		"patient_gender":      normalizeSurveyGender(patient.Gender),
		"patient_age":         patient.Age,
		"patient_phone":       patient.Phone,
		"blood_type":          patient.BloodType,
		"visit_id":            visit.ID,
		"visit_no":            visit.VisitNo,
		"visit_date":          dateOnlyString(firstNonEmpty(visit.VisitAt, patient.LastVisitAt)),
		"discharge_date":      dateOnlyString(visit.DischargeAt),
		"department":          firstNonEmpty(visit.DepartmentName, visit.DepartmentCode),
		"doctor_name":         visit.AttendingDoctor,
		"diagnosis":           firstNonEmpty(visit.DiagnosisName, patient.Diagnosis),
		"discharge_diagnosis": firstNonEmpty(visit.DiagnosisName, patient.Diagnosis),
	}
}

func numericAnswer(value interface{}) *float64 {
	switch next := value.(type) {
	case float64:
		return &next
	case int:
		parsed := float64(next)
		return &parsed
	case string:
		parsed, err := strconv.ParseFloat(next, 64)
		if err == nil {
			return &parsed
		}
	case map[string]interface{}:
		var sum float64
		var count int
		for _, item := range next {
			if score := numericAnswer(item); score != nil {
				sum += *score
				count++
			} else if text, ok := item.(string); ok {
				mapped := satisfactionTextScore(text)
				if mapped > 0 {
					sum += mapped
					count++
				}
			}
		}
		if count > 0 {
			avg := sum / float64(count)
			return &avg
		}
	}
	return nil
}

func satisfactionTextScore(value string) float64 {
	switch strings.TrimSpace(value) {
	case "很不满意":
		return 1
	case "不满意":
		return 2
	case "一般":
		return 3
	case "满意":
		return 4
	case "非常满意":
		return 5
	default:
		return 0
	}
}

func addDepartmentScore(buckets map[string]map[string]float64, item domain.SurveySubmission) {
	department := firstNonEmpty(stringAnswer(item.Answers["department"]), "未填写科室")
	score := numericAnswer(item.Answers["overall_satisfaction"])
	if score == nil {
		return
	}
	addBucket(buckets, department, *score)
}

func addIndicatorScores(buckets map[string]map[string]float64, item domain.SurveySubmission) {
	labels := map[string]string{"overall_satisfaction": "总体满意度", "recommend_score": "推荐意愿", "service_matrix": "分项满意度"}
	for key, label := range labels {
		if score := numericAnswer(item.Answers[key]); score != nil {
			addBucket(buckets, label, *score)
		}
	}
}

func addLowReasons(reasons map[string]int, item domain.SurveySubmission) {
	if score := numericAnswer(item.Answers["overall_satisfaction"]); score != nil && *score <= 3 {
		reasons["总体满意度低"]++
	}
	if score := numericAnswer(item.Answers["recommend_score"]); score != nil && *score <= 6 {
		reasons["推荐意愿低"]++
	}
	for _, word := range []string{"等待", "排队", "态度", "缴费", "环境", "停车", "检查", "药房", "护士", "医生"} {
		if strings.Contains(stringAnswer(item.Answers["feedback"]), word) {
			reasons[word]++
		}
	}
}

func addBucket(buckets map[string]map[string]float64, key string, score float64) {
	if buckets[key] == nil {
		buckets[key] = map[string]float64{"sum": 0, "count": 0}
	}
	buckets[key]["sum"] += score
	buckets[key]["count"]++
}

func averageBuckets(buckets map[string]map[string]float64) []map[string]interface{} {
	items := []map[string]interface{}{}
	for key, item := range buckets {
		if item["count"] == 0 {
			continue
		}
		items = append(items, map[string]interface{}{"name": key, "score": item["sum"] / item["count"], "count": int(item["count"])})
	}
	return items
}

func stringAnswer(value interface{}) string {
	if value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func normalizeSurveyGender(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "男", "m", "male":
		return "male"
	case "女", "f", "female":
		return "female"
	default:
		return value
	}
}

func dateOnlyString(value string) string {
	if len(value) >= 10 {
		return value[:10]
	}
	return value
}

func (s *Server) streamSurveyTemplate(w http.ResponseWriter, formTemplateID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	components := []map[string]interface{}{}
	for _, item := range s.store.FormLibrary() {
		if item.Kind == "template" && (formTemplateID == "" || item.ID == formTemplateID) {
			components = item.Components
			break
		}
	}
	for _, component := range components {
		raw, _ := json.Marshal(component)
		_, _ = w.Write([]byte("event: form_component\n"))
		_, _ = w.Write([]byte("data: " + string(raw) + "\n\n"))
		flusher.Flush()
		time.Sleep(150 * time.Millisecond)
	}
	_, _ = w.Write([]byte("event: done\ndata: {}\n\n"))
	flusher.Flush()
}

func (s *Server) listReports(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ReportDefinitions(r.Context())
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createReport(w http.ResponseWriter, r *http.Request) {
	var report domain.Report
	if !decodeJSON(w, r, &report) {
		return
	}
	report, err := s.store.CreateReportDefinition(r.Context(), report)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "report.create", "/api/v1/reports/"+report.ID, nil, report)
	writeJSON(w, http.StatusCreated, report)
}

func (s *Server) getReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.store.ReportDefinition(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) updateReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, err := s.store.ReportDefinition(r.Context(), id)
	if err != nil {
		statusError(w, err)
		return
	}
	var patch domain.Report
	if !decodeJSON(w, r, &patch) {
		return
	}
	report, err := s.store.UpdateReportDefinition(r.Context(), id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "report.update", "/api/v1/reports/"+report.ID, before, report)
	writeJSON(w, http.StatusOK, report)
}

func (s *Server) queryReport(w http.ResponseWriter, r *http.Request) {
	result, err := s.store.QueryReportData(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) addReportWidget(w http.ResponseWriter, r *http.Request) {
	var widget domain.ReportWidget
	if !decodeJSON(w, r, &widget) {
		return
	}
	widget, err := s.store.AddReportDefinitionWidget(r.Context(), chi.URLParam(r, "id"), widget)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "report.widget.create", "/api/v1/reports/"+widget.ReportID, nil, widget)
	writeJSON(w, http.StatusCreated, widget)
}

func (s *Server) listEvaluationComplaints(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.EvaluationComplaints(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("kind"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createEvaluationComplaint(w http.ResponseWriter, r *http.Request) {
	var item domain.EvaluationComplaint
	if !decodeJSON(w, r, &item) {
		return
	}
	item.CreatedBy = actorID(r)
	created, err := s.store.CreateEvaluationComplaint(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "evaluation-complaint.create", "/api/v1/evaluation-complaints/"+created.ID, nil, created)
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) getEvaluationComplaint(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.EvaluationComplaint(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) updateEvaluationComplaint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, err := s.store.EvaluationComplaint(r.Context(), id)
	if err != nil {
		statusError(w, err)
		return
	}
	var patch domain.EvaluationComplaint
	if !decodeJSON(w, r, &patch) {
		return
	}
	updated, err := s.store.UpdateEvaluationComplaint(r.Context(), id, patch, actorID(r))
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "evaluation-complaint.update", "/api/v1/evaluation-complaints/"+updated.ID, before, updated)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) deleteEvaluationComplaint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, err := s.store.EvaluationComplaint(r.Context(), id)
	if err != nil {
		statusError(w, err)
		return
	}
	if err := s.store.DeleteEvaluationComplaint(r.Context(), id, actorID(r)); err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "evaluation-complaint.delete", "/api/v1/evaluation-complaints/"+id, before, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) evaluationComplaintStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.store.EvaluationComplaintStats(r.Context())
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) listSeats(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Seats())
}

func (s *Server) listFollowupPlans(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.FollowupPlans())
}

func (s *Server) createFollowupPlan(w http.ResponseWriter, r *http.Request) {
	var plan domain.FollowupPlan
	if !decodeJSON(w, r, &plan) {
		return
	}
	plan = s.store.CreateFollowupPlan(plan)
	s.audit(r, actorID(r), "followup-plan.create", "/api/v1/followup/plans/"+plan.ID, nil, plan)
	writeJSON(w, http.StatusCreated, plan)
}

func (s *Server) updateFollowupPlan(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, _ := s.store.FollowupPlanByID(id)
	var patch domain.FollowupPlan
	if !decodeJSON(w, r, &patch) {
		return
	}
	plan, err := s.store.UpdateFollowupPlan(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "followup-plan.update", "/api/v1/followup/plans/"+plan.ID, before, plan)
	writeJSON(w, http.StatusOK, plan)
}

func (s *Server) generateFollowupTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.GenerateFollowupTasks(chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "followup-task.generate", "/api/v1/followup/plans/"+chi.URLParam(r, "id"), nil, tasks)
	writeJSON(w, http.StatusCreated, tasks)
}

func (s *Server) listFollowupTasks(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.FollowupTasks(r.URL.Query().Get("status"), r.URL.Query().Get("assigneeId")))
}

func (s *Server) createFollowupTask(w http.ResponseWriter, r *http.Request) {
	var task domain.FollowupTask
	if !decodeJSON(w, r, &task) {
		return
	}
	task = s.store.CreateFollowupTask(task)
	s.audit(r, actorID(r), "followup-task.create", "/api/v1/followup/tasks/"+task.ID, nil, task)
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) updateFollowupTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var patch domain.FollowupTask
	if !decodeJSON(w, r, &patch) {
		return
	}
	task, err := s.store.UpdateFollowupTask(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "followup-task.update", "/api/v1/followup/tasks/"+task.ID, nil, task)
	writeJSON(w, http.StatusOK, task)
}

func (s *Server) listDepartments(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Departments())
}

func (s *Server) listDictionaries(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Dictionaries())
}

func (s *Server) createDictionary(w http.ResponseWriter, r *http.Request) {
	var item domain.Dictionary
	if !decodeJSON(w, r, &item) {
		return
	}
	item = s.store.CreateDictionary(item)
	s.audit(r, actorID(r), "dictionary.create", "/api/v1/dictionaries/"+item.ID, nil, item)
	writeJSON(w, http.StatusCreated, item)
}

func (s *Server) updateDictionary(w http.ResponseWriter, r *http.Request) {
	var patch domain.Dictionary
	if !decodeJSON(w, r, &patch) {
		return
	}
	item, err := s.store.UpdateDictionary(chi.URLParam(r, "id"), patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "dictionary.update", "/api/v1/dictionaries/"+item.ID, nil, item)
	writeJSON(w, http.StatusOK, item)
}

func (s *Server) createSeat(w http.ResponseWriter, r *http.Request) {
	var seat domain.AgentSeat
	if !decodeJSON(w, r, &seat) {
		return
	}
	seat = s.store.CreateSeat(seat)
	s.audit(r, actorID(r), "seat.create", "/api/v1/call-center/seats/"+seat.ID, nil, seat)
	writeJSON(w, http.StatusCreated, seat)
}

func (s *Server) getSeat(w http.ResponseWriter, r *http.Request) {
	seat, ok := s.store.Seat(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, seat)
}

func (s *Server) updateSeat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.Seat(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch domain.AgentSeat
	if !decodeJSON(w, r, &patch) {
		return
	}
	seat, err := s.store.UpdateSeat(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "seat.update", "/api/v1/call-center/seats/"+seat.ID, before, seat)
	writeJSON(w, http.StatusOK, seat)
}

func (s *Server) deleteSeat(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.Seat(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	seat, err := s.store.DeleteSeat(id)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "seat.delete", "/api/v1/call-center/seats/"+seat.ID, before, nil)
	writeJSON(w, http.StatusOK, seat)
}

func (s *Server) listSipEndpoints(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.SipEndpoints())
}

func (s *Server) createSipEndpoint(w http.ResponseWriter, r *http.Request) {
	var endpoint domain.SipEndpoint
	if !decodeJSON(w, r, &endpoint) {
		return
	}
	endpoint = s.store.CreateSipEndpoint(endpoint)
	s.audit(r, actorID(r), "sip_endpoint.create", "/api/v1/call-center/sip-endpoints/"+endpoint.ID, nil, endpoint)
	writeJSON(w, http.StatusCreated, endpoint)
}

func (s *Server) getSipEndpoint(w http.ResponseWriter, r *http.Request) {
	endpoint, ok := s.store.SipEndpoint(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, endpoint)
}

func (s *Server) updateSipEndpoint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.SipEndpoint(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch domain.SipEndpoint
	if !decodeJSON(w, r, &patch) {
		return
	}
	endpoint, err := s.store.UpdateSipEndpoint(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "sip_endpoint.update", "/api/v1/call-center/sip-endpoints/"+endpoint.ID, before, endpoint)
	writeJSON(w, http.StatusOK, endpoint)
}

func (s *Server) deleteSipEndpoint(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.SipEndpoint(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	endpoint, err := s.store.DeleteSipEndpoint(id)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "sip_endpoint.delete", "/api/v1/call-center/sip-endpoints/"+endpoint.ID, before, nil)
	writeJSON(w, http.StatusOK, endpoint)
}

func (s *Server) listStorageConfigs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.StorageConfigs())
}

func (s *Server) createStorageConfig(w http.ResponseWriter, r *http.Request) {
	var config domain.StorageConfig
	if !decodeJSON(w, r, &config) {
		return
	}
	config = s.store.CreateStorageConfig(config)
	s.audit(r, actorID(r), "storage_config.create", "/api/v1/call-center/storage-configs/"+config.ID, nil, config)
	writeJSON(w, http.StatusCreated, config)
}

func (s *Server) getStorageConfig(w http.ResponseWriter, r *http.Request) {
	config, ok := s.store.StorageConfig(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (s *Server) updateStorageConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.StorageConfig(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch domain.StorageConfig
	if !decodeJSON(w, r, &patch) {
		return
	}
	config, err := s.store.UpdateStorageConfig(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "storage_config.update", "/api/v1/call-center/storage-configs/"+config.ID, before, config)
	writeJSON(w, http.StatusOK, config)
}

func (s *Server) deleteStorageConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.StorageConfig(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	config, err := s.store.DeleteStorageConfig(id)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "storage_config.delete", "/api/v1/call-center/storage-configs/"+config.ID, before, nil)
	writeJSON(w, http.StatusOK, config)
}

func (s *Server) listRecordingConfigs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.RecordingConfigs())
}

func (s *Server) createRecordingConfig(w http.ResponseWriter, r *http.Request) {
	var config domain.RecordingConfig
	if !decodeJSON(w, r, &config) {
		return
	}
	config = s.store.CreateRecordingConfig(config)
	s.audit(r, actorID(r), "recording_config.create", "/api/v1/call-center/recording-configs/"+config.ID, nil, config)
	writeJSON(w, http.StatusCreated, config)
}

func (s *Server) getRecordingConfig(w http.ResponseWriter, r *http.Request) {
	config, ok := s.store.RecordingConfig(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, config)
}

func (s *Server) updateRecordingConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.RecordingConfig(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch domain.RecordingConfig
	if !decodeJSON(w, r, &patch) {
		return
	}
	config, err := s.store.UpdateRecordingConfig(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "recording_config.update", "/api/v1/call-center/recording-configs/"+config.ID, before, config)
	writeJSON(w, http.StatusOK, config)
}

func (s *Server) deleteRecordingConfig(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.RecordingConfig(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	config, err := s.store.DeleteRecordingConfig(id)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "recording_config.delete", "/api/v1/call-center/recording-configs/"+config.ID, before, nil)
	writeJSON(w, http.StatusOK, config)
}

func (s *Server) listCalls(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Calls())
}

func (s *Server) createCall(w http.ResponseWriter, r *http.Request) {
	var call domain.CallSession
	if !decodeJSON(w, r, &call) {
		return
	}
	call = s.store.CreateCall(call)
	if endpoint, ok := s.store.DefaultSipEndpoint(); ok {
		result, err := s.sip.Dial(r.Context(), endpoint, call)
		switch {
		case errors.Is(err, sipgateway.ErrDisabled):
			call, _ = s.store.UpdateCall(call.ID, domain.CallSession{Status: "connected"})
		case err != nil:
			call, _ = s.store.UpdateCall(call.ID, domain.CallSession{Status: "failed"})
			s.audit(r, actorID(r), "call.gateway.failed", "/api/v1/call-center/calls/"+call.ID, nil, map[string]string{"error": err.Error()})
		default:
			call, _ = s.store.UpdateCall(call.ID, domain.CallSession{Status: result.Status})
			s.audit(r, actorID(r), "call.gateway.dial", "/api/v1/call-center/calls/"+call.ID, nil, result)
		}
	}
	s.audit(r, actorID(r), "call.create", "/api/v1/call-center/calls/"+call.ID, nil, call)
	writeJSON(w, http.StatusCreated, call)
}

func (s *Server) updateCall(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	status := r.URL.Query().Get("status")
	if status == "ended" || status == "recorded" || status == "failed" {
		_ = s.sip.Hangup(r.Context(), id)
	}
	call, err := s.store.UpdateCall(id, domain.CallSession{Status: status, EndedAt: time.Now().UTC()})
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "call.update", "/api/v1/call-center/calls/"+call.ID, nil, call)
	writeJSON(w, http.StatusOK, call)
}

func (s *Server) listRecordings(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Recordings())
}

func (s *Server) createRecording(w http.ResponseWriter, r *http.Request) {
	var recording domain.Recording
	if !decodeJSON(w, r, &recording) {
		return
	}
	recording = s.store.CreateRecording(recording)
	s.audit(r, actorID(r), "recording.create", "/api/v1/call-center/recordings/"+recording.ID, nil, recording)
	writeJSON(w, http.StatusCreated, recording)
}

func (s *Server) uploadRecording(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 200<<20)
	if err := r.ParseMultipartForm(200 << 20); err != nil {
		http.Error(w, "invalid multipart recording upload", http.StatusBadRequest)
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "file is required", http.StatusBadRequest)
		return
	}
	defer file.Close()

	callID := strings.TrimSpace(r.FormValue("callId"))
	if callID == "" {
		http.Error(w, "callId is required", http.StatusBadRequest)
		return
	}
	recordingConfig, ok := s.store.DefaultRecordingConfig()
	if !ok {
		http.Error(w, "recording config is required", http.StatusBadRequest)
		return
	}
	storageConfig, ok := s.store.StorageConfig(recordingConfig.StorageConfigID)
	if !ok {
		http.Error(w, "recording storage config is required", http.StatusBadRequest)
		return
	}
	storageBackend := recordingstorage.NewFromStorageConfig(storageConfig)
	result, err := storageBackend.Save(r.Context(), recordingstorage.Request{
		CallID:       callID,
		OriginalName: header.Filename,
		MimeType:     header.Header.Get("Content-Type"),
		Size:         header.Size,
		Reader:       file,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	recording := s.store.CreateRecording(domain.Recording{
		CallID:     callID,
		StorageURI: result.URI,
		Duration:   parseOptionalInt(r.FormValue("duration")),
		Filename:   result.Filename,
		MimeType:   result.MimeType,
		SizeBytes:  result.SizeBytes,
		Source:     firstNonEmpty(r.FormValue("source"), "browser_media_recorder"),
		Backend:    result.Backend,
		ObjectName: result.ObjectName,
		Status:     "ready",
	})
	s.audit(r, actorID(r), "recording.upload", "/api/v1/call-center/recordings/"+recording.ID, nil, recording)
	writeJSON(w, http.StatusCreated, recording)
}

func (s *Server) listModelProviders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.ModelProviders())
}

func (s *Server) createModelProvider(w http.ResponseWriter, r *http.Request) {
	var provider domain.ModelProvider
	if !decodeJSON(w, r, &provider) {
		return
	}
	provider = s.store.CreateModelProvider(provider)
	s.audit(r, actorID(r), "model_provider.create", "/api/v1/call-center/model-providers/"+provider.ID, nil, provider)
	writeJSON(w, http.StatusCreated, provider)
}

func (s *Server) getModelProvider(w http.ResponseWriter, r *http.Request) {
	provider, ok := s.store.ModelProvider(chi.URLParam(r, "id"))
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, provider)
}

func (s *Server) updateModelProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.ModelProvider(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch domain.ModelProvider
	if !decodeJSON(w, r, &patch) {
		return
	}
	provider, err := s.store.UpdateModelProvider(id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "model_provider.update", "/api/v1/call-center/model-providers/"+provider.ID, before, provider)
	writeJSON(w, http.StatusOK, provider)
}

func (s *Server) deleteModelProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok := s.store.ModelProvider(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	provider, err := s.store.DeleteModelProvider(id)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "model_provider.delete", "/api/v1/call-center/model-providers/"+provider.ID, before, nil)
	writeJSON(w, http.StatusOK, provider)
}

func (s *Server) listAnalyses(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Analyses())
}

func (s *Server) listRealtimeAssistSessions(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.RealtimeAssistSessions())
}

func (s *Server) createRealtimeAssistSession(w http.ResponseWriter, r *http.Request) {
	var session domain.RealtimeAssistSession
	if !decodeJSON(w, r, &session) {
		return
	}
	session = s.store.CreateRealtimeAssistSession(session)
	s.audit(r, actorID(r), "realtime_assist.create", "/api/v1/call-center/realtime-assist/"+session.ID, nil, session)
	writeJSON(w, http.StatusCreated, session)
}

func (s *Server) addRealtimeTranscript(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Speaker   string                 `json:"speaker"`
		Text      string                 `json:"text"`
		IsFinal   bool                   `json:"isFinal"`
		FormPatch map[string]interface{} `json:"formPatch"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	session, err := s.store.AddRealtimeTranscript(chi.URLParam(r, "id"), domain.RealtimeTranscript{Speaker: req.Speaker, Text: req.Text, IsFinal: req.IsFinal}, req.FormPatch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "realtime_assist.transcript", "/api/v1/call-center/realtime-assist/"+session.ID, nil, session)
	writeJSON(w, http.StatusOK, session)
}

func (s *Server) listOfflineAnalysisJobs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.OfflineAnalysisJobs())
}

func (s *Server) createOfflineAnalysisJob(w http.ResponseWriter, r *http.Request) {
	var job domain.OfflineAnalysisJob
	if !decodeJSON(w, r, &job) {
		return
	}
	job = s.store.CreateOfflineAnalysisJob(job)
	s.audit(r, actorID(r), "offline_analysis.create", "/api/v1/call-center/offline-analysis-jobs/"+job.ID, nil, job)
	writeJSON(w, http.StatusCreated, job)
}

func (s *Server) listInterviews(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.Interviews())
}

func (s *Server) createInterview(w http.ResponseWriter, r *http.Request) {
	var interview domain.InterviewSession
	if !decodeJSON(w, r, &interview) {
		return
	}
	interview = s.store.CreateInterview(interview)
	s.audit(r, actorID(r), "interview.create", "/api/v1/call-center/interviews/"+interview.ID, nil, interview)
	writeJSON(w, http.StatusCreated, interview)
}

func (s *Server) listAuditLogs(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.store.AuditLogs())
}

func writeJSON(w http.ResponseWriter, status int, value interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, value interface{}) bool {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return false
	}
	return true
}

func statusError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func parseOptionalInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}
	return parsed
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func (s *Server) audit(r *http.Request, actor, action, resource string, before, after interface{}) {
	s.store.SaveAudit(domain.AuditLog{
		ActorID:   actor,
		Action:    action,
		Resource:  resource,
		Before:    before,
		After:     after,
		IP:        r.RemoteAddr,
		UserAgent: r.UserAgent(),
		TraceID:   traceID(r.Context()),
	})
}

func bearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	cookie, err := r.Cookie("reporter_access")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func setAuthCookie(w http.ResponseWriter, name, value, path string, ttl time.Duration) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     path,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(ttl.Seconds()),
	})
}

func clearAuthCookie(w http.ResponseWriter, name, path string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Path:     path,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func loginFailureKey(r *http.Request, username string) string {
	return strings.ToLower(strings.TrimSpace(username)) + "|" + clientIP(r)
}

func tokenRefreshKey(r *http.Request) string {
	return clientIP(r) + "|" + r.UserAgent()
}

func clientIP(r *http.Request) string {
	if forwarded := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); forwarded != "" {
		return strings.TrimSpace(strings.Split(forwarded, ",")[0])
	}
	return strings.Split(r.RemoteAddr, ":")[0]
}

func (s *Server) captchaRequired(key string) bool {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	return s.loginFailures[key] >= 2
}

func (s *Server) recordLoginFailure(key string) int {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	s.loginFailures[key]++
	return s.loginFailures[key]
}

func (s *Server) resetLoginFailure(key string) {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	delete(s.loginFailures, key)
}

func (s *Server) allowTokenRefresh(key string) bool {
	s.authMu.Lock()
	defer s.authMu.Unlock()
	now := time.Now()
	windowStart := now.Add(-30 * time.Second)
	hits := s.refreshHits[key]
	next := hits[:0]
	for _, hit := range hits {
		if hit.After(windowStart) {
			next = append(next, hit)
		}
	}
	if len(next) >= 6 {
		s.refreshHits[key] = next
		return false
	}
	next = append(next, now)
	s.refreshHits[key] = next
	return true
}

func (s *Server) verifyCaptcha(id, answer string) bool {
	id = strings.TrimSpace(id)
	answer = strings.TrimSpace(answer)
	if id == "" || answer == "" {
		return false
	}
	s.authMu.Lock()
	defer s.authMu.Unlock()
	challenge, ok := s.captchas[id]
	if !ok || time.Now().After(challenge.ExpiresAt) {
		delete(s.captchas, id)
		return false
	}
	delete(s.captchas, id)
	return challenge.Answer == answer
}
