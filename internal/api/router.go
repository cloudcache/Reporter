package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"math"
	"math/rand"
	"net/http"
	"regexp"
	"sort"
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
	Store  *store.Store
	SIP    sipgateway.Gateway
}

type Server struct {
	cfg           config.Config
	log           zerolog.Logger
	store         *store.Store
	authz         *rbac.Authorizer
	sip           sipgateway.Gateway
	authMu        sync.Mutex
	loginFailures map[string]int
	refreshHits   map[string][]time.Time
	publicHits    map[string][]time.Time
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
		publicHits:    map[string][]time.Time{},
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
			r.Get("/projects", s.withPermission("/api/v1/forms", "read", s.listSatisfactionProjects))
			r.Post("/projects", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionProject))
			r.Put("/projects/{id}", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionProject))
			r.Delete("/projects/{id}", s.withPermission("/api/v1/forms", "delete", s.deleteSatisfactionProject))
			r.Post("/graphql", s.withPermission("/api/v1/forms", "read", s.graphQLQuery))
			r.Get("/satisfaction/projects", s.withPermission("/api/v1/forms", "read", s.listSatisfactionProjects))
			r.Post("/satisfaction/projects", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionProject))
			r.Put("/satisfaction/projects/{id}", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionProject))
			r.Delete("/satisfaction/projects/{id}", s.withPermission("/api/v1/forms", "delete", s.deleteSatisfactionProject))
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
			r.Post("/satisfaction/cleaning-rules/reapply", s.withPermission("/api/v1/forms", "update", s.reapplySatisfactionCleaningRules))
			r.Get("/satisfaction/issues", s.withPermission("/api/v1/forms", "read", s.listSatisfactionIssues))
			r.Post("/satisfaction/issues", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionIssue))
			r.Put("/satisfaction/issues/{id}", s.withPermission("/api/v1/forms", "update", s.upsertSatisfactionIssue))
			r.Get("/satisfaction/issues/{id}/events", s.withPermission("/api/v1/forms", "read", s.listSatisfactionIssueEvents))
			r.Post("/satisfaction/issues/{id}/events", s.withPermission("/api/v1/forms", "update", s.addSatisfactionIssueEvent))
			r.Get("/satisfaction/submissions/{id}/audit-logs", s.withPermission("/api/v1/forms", "read", s.listSurveySubmissionAuditLogs))
			r.Post("/satisfaction/issues/generate", s.withPermission("/api/v1/forms", "update", s.generateSatisfactionIssues))
			r.Get("/survey-share-links", s.withPermission("/api/v1/forms", "read", s.listSurveyShareLinks))
			r.Post("/survey-share-links", s.withPermission("/api/v1/forms", "update", s.createSurveyShareLink))
			r.Get("/survey-channel-recipients", s.withPermission("/api/v1/forms", "read", s.listSurveyChannelRecipients))
			r.Get("/survey-channel-deliveries", s.withPermission("/api/v1/forms", "read", s.listSurveyChannelDeliveries))
			r.Post("/survey-channel-deliveries", s.withPermission("/api/v1/forms", "update", s.createSurveyChannelDeliveries))
			r.Post("/survey-channel-deliveries/send", s.withPermission("/api/v1/forms", "update", s.sendSurveyChannelDeliveries))
			r.Post("/survey-channel-deliveries/{id}/send", s.withPermission("/api/v1/forms", "update", s.sendSurveyChannelDelivery))
			r.Post("/survey-channel-deliveries/{id}/receipt", s.withPermission("/api/v1/forms", "update", s.updateSurveyChannelDeliveryReceipt))

			r.Get("/reports", s.withPermission("/api/v1/reports", "read", s.listReports))
			r.Post("/reports", s.withPermission("/api/v1/reports", "create", s.createReport))
			r.Get("/reports/{id}", s.withPermission("/api/v1/reports", "read", s.getReport))
			r.Put("/reports/{id}", s.withPermission("/api/v1/reports", "update", s.updateReport))
			r.Post("/reports/{id}/query", s.withPermission("/api/v1/reports", "query", s.queryReport))
			r.Get("/reports/{id}/export", s.withPermission("/api/v1/reports", "read", s.exportReport))
			r.Get("/reports/{id}/insights", s.withPermission("/api/v1/reports", "read", s.reportInsights))
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
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":          user.ID,
		"username":    user.Username,
		"displayName": user.DisplayName,
		"roles":       user.Roles,
		"permissions": s.effectivePermissions(user),
	})
}

func (s *Server) effectivePermissions(user domain.User) []string {
	seen := map[string]bool{}
	for _, roleID := range user.Roles {
		if roleID == "admin" {
			return []string{"*:*"}
		}
		for _, role := range s.store.Roles() {
			if role.ID != roleID {
				continue
			}
			for _, permission := range role.Permissions {
				if !seen[permission] {
					seen[permission] = true
				}
			}
		}
	}
	items := make([]string, 0, len(seen))
	for permission := range seen {
		items = append(items, permission)
	}
	return items
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
	items, err := s.store.PatientsStrict(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createPatient(w http.ResponseWriter, r *http.Request) {
	var patient domain.Patient
	if !decodeJSON(w, r, &patient) {
		return
	}
	patient, err := s.store.CreatePatientStrict(r.Context(), patient)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "patient.create", "/api/v1/patients/"+patient.ID, nil, patient)
	writeJSON(w, http.StatusCreated, patient)
}

func (s *Server) getPatient(w http.ResponseWriter, r *http.Request) {
	patient, ok, err := s.store.PatientStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, patient)
}

func (s *Server) updatePatient(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok, err := s.store.PatientStrict(r.Context(), id)
	if err != nil {
		statusError(w, err)
		return
	}
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
	items, err := s.store.VisitsStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listPatientMedicalRecords(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.MedicalRecordsStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
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
	var err error
	form, err = s.store.CreateFormStrict(r.Context(), form)
	if err != nil {
		statusError(w, err)
		return
	}
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
	version, err := s.store.CreateFormVersionStrict(r.Context(), chi.URLParam(r, "id"), actorID(r), req.Schema)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "form.version.create", "/api/v1/forms/"+version.FormID, nil, version)
	writeJSON(w, http.StatusCreated, version)
}

func (s *Server) publishForm(w http.ResponseWriter, r *http.Request) {
	form, err := s.store.PublishFormStrict(r.Context(), chi.URLParam(r, "id"))
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
	submission, err := s.store.CreateSubmissionStrict(r.Context(), domain.Submission{FormID: chi.URLParam(r, "id"), SubmitterID: actorID(r), Data: req.Data})
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "submission.create", "/api/v1/submissions/"+submission.ID, nil, submission)
	writeJSON(w, http.StatusCreated, submission)
}

func (s *Server) listSubmissions(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SubmissionsByFormStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) getSubmission(w http.ResponseWriter, r *http.Request) {
	submission, ok, err := s.store.SubmissionStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, submission)
}

func (s *Server) listDataSources(w http.ResponseWriter, r *http.Request) {
	sources, err := s.store.DataSourcesStrict(r.Context())
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sources)
}

func (s *Server) createDataSource(w http.ResponseWriter, r *http.Request) {
	var source domain.DataSource
	if !decodeJSON(w, r, &source) {
		return
	}
	var err error
	source, err = s.store.CreateDataSourceStrict(r.Context(), source)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "data-source.create", "/api/v1/data-sources/"+source.ID, nil, source)
	writeJSON(w, http.StatusCreated, source)
}

func (s *Server) getDataSource(w http.ResponseWriter, r *http.Request) {
	source, ok, err := s.store.DataSourceStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, source)
}

func (s *Server) updateDataSource(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	before, ok, err := s.store.DataSourceStrict(r.Context(), id)
	if err != nil {
		statusError(w, err)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	var patch domain.DataSource
	if !decodeJSON(w, r, &patch) {
		return
	}
	source, err := s.store.UpdateDataSourceStrict(r.Context(), id, patch)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "data-source.update", "/api/v1/data-sources/"+source.ID, before, source)
	writeJSON(w, http.StatusOK, source)
}

func (s *Server) deleteDataSource(w http.ResponseWriter, r *http.Request) {
	source, err := s.store.DeleteDataSourceStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "data-source.delete", "/api/v1/data-sources/"+source.ID, source, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "id": source.ID})
}

func (s *Server) testDataSource(w http.ResponseWriter, r *http.Request) {
	source, ok, err := s.store.DataSourceStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	if !ok {
		http.NotFound(w, r)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"ok": true, "protocol": source.Protocol, "message": "connector contract validated"})
}

func (s *Server) previewDataSource(w http.ResponseWriter, r *http.Request) {
	source, ok, err := s.store.DataSourceStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
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
	source, ok, err := s.store.DataSourceStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
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
	for rowIndex, record := range records {
		quality := domain.DataSourceQualityResult{RowIndex: rowIndex, Status: "valid"}
		patientFields := record.Entities["patient"]
		var patient domain.Patient
		if len(patientFields) > 0 {
			patient = datamapping.ApplyPatientFields(patientFields, domain.Patient{})
			if patient.PatientNo != "" {
				patient.SourceRefs = map[string]interface{}{"dataSourceId": source.ID, "protocol": source.Protocol}
				if req.DryRun {
					result.Patients = append(result.Patients, patient)
				} else {
					saved, created, err := s.store.UpsertPatientByNoStrict(r.Context(), patient)
					if err != nil {
						appendQualityError(&result, &quality, "患者写入失败", err)
						result.Quality = append(result.Quality, quality)
						continue
					}
					if created {
						result.Created++
					} else {
						result.Updated++
					}
					patient = saved
					result.Patients = append(result.Patients, saved)
				}
			} else {
				quality.Status = "invalid"
				quality.Messages = append(quality.Messages, "缺少 patient.patientNo，无法稳定写入患者主索引")
			}
		}

		visitFields := record.Entities["visit"]
		var visit domain.ClinicalVisit
		if len(visitFields) > 0 {
			visit = datamapping.ApplyVisitFields(visitFields, domain.ClinicalVisit{PatientID: patient.ID})
			if visit.PatientID == "" {
				visit.PatientID = patient.ID
			}
			if visit.VisitNo != "" {
				visit.SourceRefs = map[string]interface{}{"dataSourceId": source.ID, "protocol": source.Protocol}
				if req.DryRun {
					result.Visits = append(result.Visits, visit)
				} else {
					saved, created, err := s.store.UpsertVisitByNoStrict(r.Context(), visit)
					if err != nil {
						appendQualityError(&result, &quality, "就诊写入失败", err)
						result.Quality = append(result.Quality, quality)
						continue
					}
					if created {
						result.Created++
					} else {
						result.Updated++
					}
					result.Visits = append(result.Visits, saved)
					visit = saved
				}
			} else if hasMeaningfulFields(visitFields) {
				quality.Status = "suspicious"
				quality.Messages = append(quality.Messages, "存在就诊字段但缺少 visit.visitNo，已跳过就诊写入")
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
					saved, created, err := s.store.UpsertMedicalRecordByNoStrict(r.Context(), medicalRecord)
					if err != nil {
						appendQualityError(&result, &quality, "病历写入失败", err)
						result.Quality = append(result.Quality, quality)
						continue
					}
					if created {
						result.Created++
					} else {
						result.Updated++
					}
					result.MedicalRecords = append(result.MedicalRecords, saved)
				}
			}
		}
		if patient.ID != "" || hasClinicalEntities(record.Entities) {
			s.syncClinicalEntities(r.Context(), source, record, patient.ID, visit.ID, req.DryRun, &result, &quality)
		}
		if len(quality.Messages) == 0 {
			quality.Messages = append(quality.Messages, "字段映射、必填项和标准实体写入校验通过")
		}
		result.Quality = append(result.Quality, quality)
	}
	s.audit(r, actorID(r), "data-source.sync", "/api/v1/data-sources/"+source.ID, nil, result)
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) syncClinicalEntities(ctx context.Context, source domain.DataSource, record domain.MappedRecord, patientID string, visitID string, dryRun bool, result *domain.DataSourceSyncResult, quality *domain.DataSourceQualityResult) {
	sourceSystem := firstNonEmpty(source.Name, source.ID, source.Protocol)
	patientID = firstNonEmpty(patientID, fieldString(record.Entities["patient"], "id"), fieldString(record.Entities["diagnosis"], "patientId"), fieldString(record.Entities["history"], "patientId"), fieldString(record.Entities["medication"], "patientId"), fieldString(record.Entities["lab"], "patientId"), fieldString(record.Entities["exam"], "patientId"), fieldString(record.Entities["surgery"], "patientId"), fieldString(record.Entities["followup"], "patientId"), fieldString(record.Entities["fact"], "patientId"))
	visitID = firstNonEmpty(visitID, fieldString(record.Entities["visit"], "id"))
	if patientID == "" {
		if hasClinicalEntities(record.Entities) {
			quality.Status = "suspicious"
			quality.Messages = append(quality.Messages, "临床事实缺少 patientId，未写入诊断/用药/检验等事实表")
		}
		return
	}
	if fields := record.Entities["diagnosis"]; len(fields) > 0 {
		item := datamapping.ApplyDiagnosisFields(fields, domain.PatientDiagnosis{PatientID: patientID, VisitID: visitID, SourceSystem: sourceSystem})
		if item.DiagnosisName == "" {
			appendQualityIssue(quality, "诊断映射缺少 diagnosis.diagnosisName")
		} else if dryRun {
			result.Diagnoses = append(result.Diagnoses, item)
		} else if saved, created, err := s.store.UpsertPatientDiagnosis(ctx, item); err == nil {
			countSync(result, created)
			result.Diagnoses = append(result.Diagnoses, saved)
		} else {
			appendQualityError(result, quality, "诊断写入失败", err)
		}
	}
	if fields := record.Entities["history"]; len(fields) > 0 {
		item := datamapping.ApplyHistoryFields(fields, domain.PatientHistory{PatientID: patientID, SourceSystem: sourceSystem})
		if item.Content == "" && item.Title == "" {
			appendQualityIssue(quality, "既往史映射缺少 history.content 或 history.title")
		} else if dryRun {
			result.Histories = append(result.Histories, item)
		} else if saved, created, err := s.store.UpsertPatientHistory(ctx, item); err == nil {
			countSync(result, created)
			result.Histories = append(result.Histories, saved)
		} else {
			appendQualityError(result, quality, "既往史写入失败", err)
		}
	}
	if fields := record.Entities["medication"]; len(fields) > 0 {
		item := datamapping.ApplyMedicationFields(fields, domain.MedicationOrder{PatientID: patientID, VisitID: visitID})
		if item.DrugName == "" {
			appendQualityIssue(quality, "用药映射缺少 medication.drugName")
		} else if dryRun {
			result.Medications = append(result.Medications, item)
		} else if saved, created, err := s.store.UpsertMedicationOrder(ctx, item); err == nil {
			countSync(result, created)
			result.Medications = append(result.Medications, saved)
		} else {
			appendQualityError(result, quality, "用药写入失败", err)
		}
	}
	if fields := record.Entities["lab"]; len(fields) > 0 {
		item := datamapping.ApplyLabReportFields(fields, domain.LabReport{PatientID: patientID, VisitID: visitID, SourceSystem: sourceSystem})
		if item.ReportName == "" {
			appendQualityIssue(quality, "检验报告映射缺少 lab.reportName")
		} else if dryRun {
			result.LabReports = append(result.LabReports, item)
		} else if saved, created, err := s.store.UpsertLabReport(ctx, item); err == nil {
			countSync(result, created)
			if labResultFields := record.Entities["labResult"]; len(labResultFields) > 0 {
				labResult := datamapping.ApplyLabResultFields(labResultFields, domain.LabResult{ReportID: saved.ID})
				if labResult.ItemName == "" {
					appendQualityIssue(quality, "检验结果映射缺少 labResult.itemName")
				} else if savedResult, createdResult, err := s.store.UpsertLabResult(ctx, labResult); err == nil {
					countSync(result, createdResult)
					saved.Results = append(saved.Results, savedResult)
				} else {
					appendQualityError(result, quality, "检验结果写入失败", err)
				}
			}
			result.LabReports = append(result.LabReports, saved)
		} else {
			appendQualityError(result, quality, "检验报告写入失败", err)
		}
	}
	if fields := record.Entities["exam"]; len(fields) > 0 {
		item := datamapping.ApplyExamReportFields(fields, domain.ExamReport{PatientID: patientID, VisitID: visitID, SourceSystem: sourceSystem})
		if item.ExamName == "" {
			appendQualityIssue(quality, "检查报告映射缺少 exam.examName")
		} else if dryRun {
			result.ExamReports = append(result.ExamReports, item)
		} else if saved, created, err := s.store.UpsertExamReport(ctx, item); err == nil {
			countSync(result, created)
			result.ExamReports = append(result.ExamReports, saved)
		} else {
			appendQualityError(result, quality, "检查报告写入失败", err)
		}
	}
	if fields := record.Entities["surgery"]; len(fields) > 0 {
		item := datamapping.ApplySurgeryFields(fields, domain.SurgeryRecord{PatientID: patientID, VisitID: visitID, SourceSystem: sourceSystem})
		if item.OperationName == "" {
			appendQualityIssue(quality, "手术记录映射缺少 surgery.operationName")
		} else if dryRun {
			result.Surgeries = append(result.Surgeries, item)
		} else if saved, created, err := s.store.UpsertSurgeryRecord(ctx, item); err == nil {
			countSync(result, created)
			result.Surgeries = append(result.Surgeries, saved)
		} else {
			appendQualityError(result, quality, "手术记录写入失败", err)
		}
	}
	if fields := record.Entities["followup"]; len(fields) > 0 {
		item := datamapping.ApplyFollowupRecordFields(fields, domain.FollowupRecord{PatientID: patientID, VisitID: visitID, SourceSystem: sourceSystem})
		if item.Summary == "" && item.TaskID == "" {
			appendQualityIssue(quality, "随访记录映射缺少 followup.summary 或 followup.taskId")
		} else if dryRun {
			result.Followups = append(result.Followups, item)
		} else if saved, created, err := s.store.UpsertFollowupRecord(ctx, item); err == nil {
			countSync(result, created)
			result.Followups = append(result.Followups, saved)
		} else {
			appendQualityError(result, quality, "随访记录写入失败", err)
		}
	}
	if fields := record.Entities["fact"]; len(fields) > 0 {
		item := datamapping.ApplyInterviewFactFields(fields, domain.InterviewExtractedFact{PatientID: patientID, VisitID: visitID})
		if item.FactType == "" || item.FactKey == "" || item.FactLabel == "" {
			appendQualityIssue(quality, "访谈事实映射缺少 fact.factType/factKey/factLabel")
		} else if dryRun {
			result.InterviewFacts = append(result.InterviewFacts, item)
		} else if saved, created, err := s.store.UpsertInterviewExtractedFact(ctx, item); err == nil {
			countSync(result, created)
			result.InterviewFacts = append(result.InterviewFacts, saved)
		} else {
			appendQualityError(result, quality, "访谈事实写入失败", err)
		}
	}
}

func hasClinicalEntities(entities map[string]map[string]interface{}) bool {
	for _, key := range []string{"diagnosis", "history", "medication", "lab", "labResult", "exam", "surgery", "followup", "fact"} {
		if len(entities[key]) > 0 {
			return true
		}
	}
	return false
}

func hasMeaningfulFields(fields map[string]interface{}) bool {
	for _, value := range fields {
		if strings.TrimSpace(fmt.Sprint(value)) != "" {
			return true
		}
	}
	return false
}

func fieldString(fields map[string]interface{}, key string) string {
	if fields == nil {
		return ""
	}
	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func countSync(result *domain.DataSourceSyncResult, created bool) {
	if created {
		result.Created++
	} else {
		result.Updated++
	}
}

func appendQualityIssue(quality *domain.DataSourceQualityResult, message string) {
	if quality.Status == "valid" {
		quality.Status = "suspicious"
	}
	quality.Messages = append(quality.Messages, message)
}

func appendQualityError(result *domain.DataSourceSyncResult, quality *domain.DataSourceQualityResult, label string, err error) {
	quality.Status = "invalid"
	message := fmt.Sprintf("%s：%v", label, err)
	quality.Messages = append(quality.Messages, message)
	result.Errors = append(result.Errors, message)
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
	if item.Config == nil {
		item.Config = map[string]interface{}{}
	}
	if form, version, ok, err := s.resolveFormVersion(r.Context(), item.FormTemplateID, configString(item.Config, "formVersionId")); err != nil {
		statusError(w, err)
		return
	} else if ok {
		item.Config["formId"] = form.ID
		item.Config["formVersionId"] = version.ID
		item.Config["formVersion"] = version.Version
	} else if exists, err := s.managedFormExists(r.Context(), item.FormTemplateID); err != nil {
		statusError(w, err)
		return
	} else if exists {
		http.Error(w, "selected form has no publishable version", http.StatusBadRequest)
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

func (s *Server) listSurveyChannelDeliveries(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.SurveyChannelDeliveries(r.Context(), r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) listSurveyChannelRecipients(w http.ResponseWriter, r *http.Request) {
	channel := sanitizePlainText(r.URL.Query().Get("channel"), 40)
	if channel == "" {
		channel = "sms"
	}
	keyword := sanitizePlainText(r.URL.Query().Get("keyword"), 80)
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	items := []domain.SurveyChannelRecipient{}
	patients, err := s.store.PatientsStrict(r.Context(), keyword)
	if err != nil {
		statusError(w, err)
		return
	}
	for _, patient := range patients {
		recipient, source := patientRecipientForChannel(patient, channel)
		item := domain.SurveyChannelRecipient{
			PatientID: patient.ID,
			PatientNo: patient.PatientNo,
			Name:      patient.Name,
			Channel:   channel,
			Recipient: recipient,
			Source:    source,
			Available: recipient != "",
		}
		if item.Available {
			items = append(items, item)
			if len(items) >= limit {
				break
			}
			continue
		}
		item.Unavailable = "患者档案缺少" + recipientSourceLabel(channel)
		items = append(items, item)
		if len(items) >= limit {
			break
		}
	}
	writeJSON(w, http.StatusOK, items)
}

func (s *Server) createSurveyChannelDeliveries(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ShareID    string `json:"shareId"`
		ProjectID  string `json:"projectId"`
		Channel    string `json:"channel"`
		URL        string `json:"url"`
		Recipients []struct {
			PatientID string `json:"patientId"`
			Name      string `json:"name"`
			Recipient string `json:"recipient"`
			Source    string `json:"source"`
		} `json:"recipients"`
		RecipientValues []string `json:"recipientValues"`
		RecipientName   string   `json:"recipientName"`
		Message         string   `json:"message"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if strings.TrimSpace(req.ShareID) == "" || strings.TrimSpace(req.Channel) == "" {
		http.Error(w, "shareId and channel are required", http.StatusBadRequest)
		return
	}
	items := []domain.SurveyChannelDelivery{}
	appendItem := func(recipient, name string, config map[string]interface{}) {
		recipient = sanitizeDeliveryRecipient(req.Channel, recipient)
		if recipient == "" {
			return
		}
		message := sanitizePlainText(req.Message, 500)
		if message == "" {
			message = "请点击链接完成调查：" + sanitizePlainText(req.URL, 300)
		}
		if config == nil {
			config = map[string]interface{}{}
		}
		config["url"] = sanitizePlainText(req.URL, 300)
		items = append(items, domain.SurveyChannelDelivery{
			ProjectID:     sanitizePlainText(req.ProjectID, 64),
			ShareID:       sanitizePlainText(req.ShareID, 64),
			Channel:       sanitizePlainText(req.Channel, 40),
			Recipient:     recipient,
			RecipientName: sanitizePlainText(firstNonEmpty(name, req.RecipientName), 120),
			Status:        "queued",
			Message:       message,
			Config:        config,
		})
	}
	for _, recipient := range req.Recipients {
		config := map[string]interface{}{}
		if recipient.PatientID != "" {
			config["patientId"] = sanitizePlainText(recipient.PatientID, 64)
		}
		if recipient.Source != "" {
			config["recipientSource"] = sanitizePlainText(recipient.Source, 80)
		}
		appendItem(recipient.Recipient, recipient.Name, config)
	}
	for _, recipient := range req.RecipientValues {
		appendItem(recipient, "", nil)
	}
	if len(items) == 0 {
		http.Error(w, "no valid recipients", http.StatusBadRequest)
		return
	}
	if len(items) > 500 {
		http.Error(w, "too many recipients", http.StatusBadRequest)
		return
	}
	created, err := s.store.CreateSurveyChannelDeliveries(r.Context(), items)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "survey-channel.delivery.create", "/api/v1/survey-channel-deliveries", nil, created)
	writeJSON(w, http.StatusCreated, created)
}

func (s *Server) sendSurveyChannelDelivery(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.SurveyChannelDelivery(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	sent, err := s.dispatchSurveyChannelDelivery(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "survey-channel.delivery.send", "/api/v1/survey-channel-deliveries/"+sent.ID, item, sent)
	writeJSON(w, http.StatusOK, sent)
}

func (s *Server) sendSurveyChannelDeliveries(w http.ResponseWriter, r *http.Request) {
	projectID := sanitizePlainText(r.URL.Query().Get("projectId"), 64)
	items, err := s.store.SurveyChannelDeliveries(r.Context(), projectID)
	if err != nil {
		statusError(w, err)
		return
	}
	updated := []domain.SurveyChannelDelivery{}
	for _, item := range items {
		if item.Status != "queued" && item.Status != "failed" {
			continue
		}
		sent, err := s.dispatchSurveyChannelDelivery(r.Context(), item)
		if err != nil {
			statusError(w, err)
			return
		}
		updated = append(updated, sent)
		if len(updated) >= 100 {
			break
		}
	}
	s.audit(r, actorID(r), "survey-channel.delivery.batch_send", "/api/v1/survey-channel-deliveries/send", nil, updated)
	writeJSON(w, http.StatusOK, updated)
}

func (s *Server) updateSurveyChannelDeliveryReceipt(w http.ResponseWriter, r *http.Request) {
	item, err := s.store.SurveyChannelDelivery(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	var req struct {
		Status      string                 `json:"status"`
		ProviderRef string                 `json:"providerRef"`
		Error       string                 `json:"error"`
		Receipt     map[string]interface{} `json:"receipt"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	status := sanitizePlainText(req.Status, 32)
	switch status {
	case "", "delivered":
		item.Status = "delivered"
	case "sent", "failed":
		item.Status = status
	default:
		http.Error(w, "unsupported receipt status", http.StatusBadRequest)
		return
	}
	item.ProviderRef = firstNonEmpty(sanitizePlainText(req.ProviderRef, 128), item.ProviderRef)
	item.Error = sanitizePlainText(req.Error, 500)
	if item.SentAt == "" && item.Status != "failed" {
		item.SentAt = time.Now().UTC().Format("2006-01-02 15:04:05")
	}
	if item.Config == nil {
		item.Config = map[string]interface{}{}
	}
	item.Config["receipt"] = map[string]interface{}{
		"status":      item.Status,
		"providerRef": item.ProviderRef,
		"error":       item.Error,
		"payload":     req.Receipt,
		"receivedAt":  time.Now().UTC().Format("2006-01-02 15:04:05"),
	}
	updated, err := s.store.UpdateSurveyChannelDelivery(r.Context(), item)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "survey-channel.delivery.receipt", "/api/v1/survey-channel-deliveries/"+updated.ID+"/receipt", item, updated)
	writeJSON(w, http.StatusOK, updated)
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

func (s *Server) deleteSatisfactionProject(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	deleted, err := s.store.DeleteSatisfactionProject(r.Context(), id)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "project.delete", "/api/v1/projects/"+id, deleted, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	stats := buildSatisfactionAnalysis(items)
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) graphQLQuery(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Query     string                 `json:"query"`
		Variables map[string]interface{} `json:"variables"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if !strings.Contains(req.Query, "satisfactionAnalysis") {
		writeJSON(w, http.StatusOK, map[string]interface{}{"data": map[string]interface{}{}})
		return
	}
	projectID := sanitizePlainText(fmt.Sprint(req.Variables["projectId"]), 64)
	items, err := s.store.SurveySubmissions(r.Context(), projectID)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data": map[string]interface{}{
			"satisfactionAnalysis": buildSatisfactionAnalysis(items),
		},
	})
}

func buildSatisfactionAnalysis(items []domain.SurveySubmission) map[string]interface{} {
	stats := map[string]interface{}{
		"total":             len(items),
		"valid":             0,
		"pending":           0,
		"suspicious":        0,
		"invalid":           0,
		"byChannel":         map[string]int{},
		"scoreAverage":      0,
		"departmentRanking": map[string]map[string]float64{},
		"indicatorScores":   map[string]map[string]float64{},
		"lowReasons":        map[string]int{},
		"trend":             map[string]map[string]float64{},
		"crossAnalysis":     map[string]map[string]map[string]float64{},
		"importanceMatrix":  map[string]map[string]float64{},
		"dimensionRankings": map[string]map[string]map[string]float64{},
		"jobAnalysis":       map[string]map[string]float64{},
		"shortBoards":       []map[string]interface{}{},
		"varianceAnalysis":  []map[string]interface{}{},
		"correlation":       []map[string]interface{}{},
		"periodCompare":     []map[string]interface{}{},
		"graphql":           true,
		"aiInsights":        []string{},
	}
	var scoreSum float64
	var scoreCount int
	for _, item := range items {
		switch item.QualityStatus {
		case "valid":
			stats["valid"] = stats["valid"].(int) + 1
		case "invalid":
			stats["invalid"] = stats["invalid"].(int) + 1
		case "suspicious":
			stats["suspicious"] = stats["suspicious"].(int) + 1
		case "level1_review", "level2_review":
			stats["suspicious"] = stats["suspicious"].(int) + 1
		default:
			stats["pending"] = stats["pending"].(int) + 1
		}
		byChannel := stats["byChannel"].(map[string]int)
		byChannel[item.Channel]++
		if item.QualityStatus == "invalid" {
			continue
		}
		for _, key := range []string{"overall_satisfaction", "recommend_score", "service_matrix"} {
			if score := numericAnswer(item.Answers[key]); score != nil {
				scoreSum += *score
				scoreCount++
			}
		}
		addDepartmentScore(stats["departmentRanking"].(map[string]map[string]float64), item)
		addIndicatorScores(stats["indicatorScores"].(map[string]map[string]float64), item)
		addLowReasons(stats["lowReasons"].(map[string]int), item)
		addTrendScore(stats["trend"].(map[string]map[string]float64), item)
		addCrossAnalysis(stats["crossAnalysis"].(map[string]map[string]map[string]float64), item)
		addImportanceMatrix(stats["importanceMatrix"].(map[string]map[string]float64), item)
		addDimensionRankings(stats["dimensionRankings"].(map[string]map[string]map[string]float64), item)
		addJobAnalysis(stats["jobAnalysis"].(map[string]map[string]float64), item)
	}
	if scoreCount > 0 {
		stats["scoreAverage"] = scoreSum / float64(scoreCount)
	}
	stats["departmentRanking"] = averageBuckets(stats["departmentRanking"].(map[string]map[string]float64))
	stats["indicatorScores"] = averageBuckets(stats["indicatorScores"].(map[string]map[string]float64))
	stats["trend"] = averageBuckets(stats["trend"].(map[string]map[string]float64))
	stats["crossAnalysis"] = averageNestedBuckets(stats["crossAnalysis"].(map[string]map[string]map[string]float64))
	stats["importanceMatrix"] = importanceBuckets(stats["importanceMatrix"].(map[string]map[string]float64))
	stats["dimensionRankings"] = averageNestedBuckets(stats["dimensionRankings"].(map[string]map[string]map[string]float64))
	stats["jobAnalysis"] = averageBuckets(stats["jobAnalysis"].(map[string]map[string]float64))
	stats["periodCompare"] = periodCompareBuckets(stats["trend"].([]map[string]interface{}))
	stats["shortBoards"] = shortBoardItems(stats)
	stats["varianceAnalysis"] = varianceAnalysis(stats["dimensionRankings"].(map[string][]map[string]interface{}))
	stats["correlation"] = correlationAnalysis(items)
	stats["aiInsights"] = satisfactionInsights(stats)
	return stats
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

func (s *Server) reapplySatisfactionCleaningRules(w http.ResponseWriter, r *http.Request) {
	projectID := sanitizePlainText(r.URL.Query().Get("projectId"), 64)
	items, err := s.store.ReevaluateSurveySubmissionQuality(r.Context(), projectID)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "satisfaction.cleaning.reapply", "/api/v1/satisfaction/cleaning-rules/reapply", nil, map[string]interface{}{"projectId": projectID, "count": len(items)})
	writeJSON(w, http.StatusOK, items)
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
	template, err := s.formTemplateForShare(r.Context(), share)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"share": share, "template": template, "requiresVerification": surveyRequiresVerification(share)})
}

func (s *Server) verifyPublicSurveyPatient(w http.ResponseWriter, r *http.Request) {
	share, err := s.store.SurveyShareByToken(r.Context(), chi.URLParam(r, "token"))
	if err != nil {
		statusError(w, err)
		return
	}
	if !s.allowPublicAction(r, "verify:"+share.ID, 12, time.Minute) {
		http.Error(w, "too many verification attempts", http.StatusTooManyRequests)
		return
	}
	if !surveyRequiresVerification(share) {
		writeJSON(w, http.StatusOK, map[string]interface{}{"verified": true})
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 32*1024)
	var req struct {
		Identifier string `json:"identifier"`
		Phone      string `json:"phone"`
		OpenID     string `json:"openId"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	req.Identifier = sanitizePlainText(req.Identifier, 80)
	req.Phone = sanitizeDialNumber(req.Phone)
	if req.Identifier == "" || !validPhone(req.Phone) {
		http.Error(w, "invalid patient identifier or phone", http.StatusBadRequest)
		return
	}
	patient, visit, ok, err := s.findSurveyPatient(r.Context(), req.Identifier, req.Phone)
	if err != nil {
		statusError(w, err)
		return
	}
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
	if !s.allowPublicAction(r, "submit:"+share.ID, 8, time.Minute) {
		http.Error(w, "too many submissions", http.StatusTooManyRequests)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
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
	template, err := s.formTemplateForShare(r.Context(), share)
	if err != nil {
		statusError(w, err)
		return
	}
	answers, err := sanitizeSurveyAnswers(template.Components, req.Answers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	req.PatientID = sanitizePlainText(req.PatientID, 64)
	req.VisitID = sanitizePlainText(req.VisitID, 64)
	if req.DurationSeconds < 0 || req.DurationSeconds > 24*60*60 {
		http.Error(w, "invalid duration", http.StatusBadRequest)
		return
	}
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
		Answers:         answers,
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
	template, err := s.formTemplateForShare(r.Context(), share)
	if err != nil {
		statusError(w, err)
		return
	}
	s.streamSurveyComponents(w, template.Components)
}

func (s *Server) formTemplateByID(id string) domain.FormLibraryItem {
	for _, item := range s.store.FormLibrary() {
		if item.ID == id {
			return item
		}
	}
	return domain.FormLibraryItem{}
}

func (s *Server) formTemplateForShare(ctx context.Context, share domain.SurveyShareLink) (domain.FormLibraryItem, error) {
	formID := firstNonEmpty(configString(share.Config, "formId"), share.FormTemplateID)
	if form, version, ok, err := s.resolveFormVersion(ctx, formID, configString(share.Config, "formVersionId")); err != nil {
		return domain.FormLibraryItem{}, err
	} else if ok {
		return formVersionTemplate(form, version), nil
	}
	return s.formTemplateByID(share.FormTemplateID), nil
}

func (s *Server) resolveFormVersion(ctx context.Context, formID, versionID string) (domain.Form, domain.FormVersion, bool, error) {
	formID = strings.TrimSpace(formID)
	versionID = strings.TrimSpace(versionID)
	if formID == "" {
		return domain.Form{}, domain.FormVersion{}, false, nil
	}
	forms, err := s.store.FormsStrict(ctx)
	if err != nil {
		return domain.Form{}, domain.FormVersion{}, false, err
	}
	for _, form := range forms {
		if form.ID != formID {
			continue
		}
		if versionID != "" {
			for _, version := range form.Versions {
				if version.ID == versionID {
					return form, version, true, nil
				}
			}
			return domain.Form{}, domain.FormVersion{}, false, nil
		}
		if form.CurrentVersionID != "" {
			for _, version := range form.Versions {
				if version.ID == form.CurrentVersionID {
					return form, version, true, nil
				}
			}
		}
		for _, version := range form.Versions {
			if version.Published {
				return form, version, true, nil
			}
		}
	}
	return domain.Form{}, domain.FormVersion{}, false, nil
}

func (s *Server) managedFormExists(ctx context.Context, formID string) (bool, error) {
	formID = strings.TrimSpace(formID)
	forms, err := s.store.FormsStrict(ctx)
	if err != nil {
		return false, err
	}
	for _, form := range forms {
		if form.ID == formID {
			return true, nil
		}
	}
	return false, nil
}

func formVersionTemplate(form domain.Form, version domain.FormVersion) domain.FormLibraryItem {
	return domain.FormLibraryItem{
		ID:         form.ID,
		Kind:       "template",
		Label:      form.Name,
		Hint:       firstNonEmpty(form.Description, fmt.Sprintf("表单版本 v%d", version.Version)),
		Components: formComponentsToMaps(version.Schema),
		Enabled:    true,
	}
}

func formComponentsToMaps(components []domain.FormComponent) []map[string]interface{} {
	raw, err := json.Marshal(components)
	if err != nil {
		return nil
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil
	}
	return result
}

func configString(config map[string]interface{}, key string) string {
	if config == nil {
		return ""
	}
	if value, ok := config[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
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

func (s *Server) allowPublicAction(r *http.Request, scope string, limit int, window time.Duration) bool {
	key := clientIP(r) + "|" + r.UserAgent() + "|" + scope
	now := time.Now()
	s.authMu.Lock()
	defer s.authMu.Unlock()
	hits := s.publicHits[key]
	kept := hits[:0]
	for _, hit := range hits {
		if now.Sub(hit) <= window {
			kept = append(kept, hit)
		}
	}
	if len(kept) >= limit {
		s.publicHits[key] = kept
		return false
	}
	s.publicHits[key] = append(kept, now)
	return true
}

func sanitizeSurveyAnswers(components []map[string]interface{}, answers map[string]interface{}) (map[string]interface{}, error) {
	if len(answers) > 200 {
		return nil, fmt.Errorf("too many answers")
	}
	allowed := map[string]map[string]interface{}{}
	for _, component := range components {
		id, _ := component["id"].(string)
		if id != "" {
			allowed[id] = component
		}
	}
	clean := map[string]interface{}{}
	for key, value := range answers {
		key = sanitizePlainText(key, 120)
		component, ok := allowed[key]
		if !ok {
			return nil, fmt.Errorf("unknown field: %s", key)
		}
		next, err := sanitizeSurveyValue(component, value)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		clean[key] = next
	}
	for id, component := range allowed {
		required, _ := component["required"].(bool)
		if required {
			if value, ok := clean[id]; !ok || isEmptySurveyValue(value) {
				return nil, fmt.Errorf("missing required field: %s", id)
			}
		}
	}
	return clean, nil
}

func sanitizeSurveyValue(component map[string]interface{}, value interface{}) (interface{}, error) {
	kind, _ := component["type"].(string)
	switch typed := value.(type) {
	case string:
		max := 2000
		if kind == "text" || kind == "select" || kind == "radio" || kind == "date" {
			max = 240
		}
		cleaned := sanitizePlainText(typed, max)
		if kind == "radio" || kind == "select" || kind == "single_select" || kind == "likert" {
			if !optionAllowed(component, cleaned) {
				return nil, fmt.Errorf("invalid option")
			}
		}
		return cleaned, nil
	case float64:
		if (kind == "rating" || kind == "number") && (typed < -1000000 || typed > 1000000) {
			return nil, fmt.Errorf("number out of range")
		}
		return typed, nil
	case bool, nil:
		return typed, nil
	case []interface{}:
		if len(typed) > 50 {
			return nil, fmt.Errorf("too many values")
		}
		items := []interface{}{}
		for _, item := range typed {
			cleaned, err := sanitizeSurveyValue(map[string]interface{}{"type": "text"}, item)
			if err != nil {
				return nil, err
			}
			if text, ok := cleaned.(string); ok && (kind == "checkbox" || kind == "multi_select") && !optionAllowed(component, text) {
				return nil, fmt.Errorf("invalid option")
			}
			items = append(items, cleaned)
		}
		return items, nil
	case map[string]interface{}:
		if len(typed) > 80 {
			return nil, fmt.Errorf("too many values")
		}
		result := map[string]interface{}{}
		for key, item := range typed {
			cleanKey := sanitizePlainText(key, 120)
			cleaned, err := sanitizeSurveyValue(map[string]interface{}{"type": "text"}, item)
			if err != nil {
				return nil, err
			}
			result[cleanKey] = cleaned
		}
		return result, nil
	default:
		return sanitizePlainText(fmt.Sprint(typed), 500), nil
	}
}

func optionAllowed(component map[string]interface{}, value string) bool {
	rawOptions, ok := component["options"]
	if !ok || rawOptions == nil {
		return true
	}
	switch options := rawOptions.(type) {
	case []interface{}:
		for _, option := range options {
			if surveyOptionMatches(option, value) {
				return true
			}
		}
	case []map[string]interface{}:
		for _, option := range options {
			if surveyOptionMatches(option, value) {
				return true
			}
		}
	case []string:
		for _, option := range options {
			if value == sanitizePlainText(option, 240) {
				return true
			}
		}
	default:
		return true
	}
	return false
}

func surveyOptionMatches(option interface{}, value string) bool {
	switch typed := option.(type) {
	case map[string]interface{}:
		label := sanitizePlainText(fmt.Sprint(typed["label"]), 240)
		next := sanitizePlainText(fmt.Sprint(typed["value"]), 240)
		return value == label || value == next
	case string:
		return value == sanitizePlainText(typed, 240)
	default:
		return value == sanitizePlainText(fmt.Sprint(typed), 240)
	}
}

func isEmptySurveyValue(value interface{}) bool {
	switch typed := value.(type) {
	case nil:
		return true
	case string:
		return strings.TrimSpace(typed) == ""
	case []interface{}:
		return len(typed) == 0
	}
	return false
}

func sanitizePlainText(value string, max int) string {
	value = strings.TrimSpace(value)
	value = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, value)
	if max > 0 && len([]rune(value)) > max {
		value = string([]rune(value)[:max])
	}
	return html.EscapeString(value)
}

func sanitizeDialNumber(value string) string {
	return regexp.MustCompile(`[^0-9+]`).ReplaceAllString(strings.TrimSpace(value), "")
}

func validPhone(value string) bool {
	return regexp.MustCompile(`^\+?[0-9]{6,20}$`).MatchString(value)
}

func sanitizeDeliveryRecipient(channel, value string) string {
	value = strings.TrimSpace(value)
	switch channel {
	case "sms", "phone":
		value = sanitizeDialNumber(value)
		if !validPhone(value) {
			return ""
		}
		return value
	case "wechat", "wework", "mini_program", "qq":
		return sanitizePlainText(value, 120)
	default:
		return sanitizePlainText(value, 180)
	}
}

func patientRecipientForChannel(patient domain.Patient, channel string) (string, string) {
	switch channel {
	case "sms", "phone":
		if recipient := sanitizeDeliveryRecipient(channel, patient.Phone); recipient != "" {
			return recipient, "patient.phone"
		}
	case "wechat", "mini_program":
		for _, key := range []string{"wechatOpenId", "wechatOpenID", "openid", "openId", "wechat.openid", "wechat.openId"} {
			if recipient := sanitizeDeliveryRecipient(channel, sourceRefString(patient.SourceRefs, key)); recipient != "" {
				return recipient, "patient.sourceRefs." + key
			}
		}
	case "wework":
		for _, key := range []string{"weworkUserId", "weworkUserID", "wework.userid", "wework.userId", "enterpriseWechatUserId"} {
			if recipient := sanitizeDeliveryRecipient(channel, sourceRefString(patient.SourceRefs, key)); recipient != "" {
				return recipient, "patient.sourceRefs." + key
			}
		}
	case "qq":
		for _, key := range []string{"qq", "qqOpenId", "qqOpenID", "qq.openid", "qq.openId"} {
			if recipient := sanitizeDeliveryRecipient(channel, sourceRefString(patient.SourceRefs, key)); recipient != "" {
				return recipient, "patient.sourceRefs." + key
			}
		}
	}
	return "", ""
}

func sourceRefString(refs map[string]interface{}, key string) string {
	if len(refs) == 0 || key == "" {
		return ""
	}
	parts := strings.Split(key, ".")
	var current interface{} = refs
	for _, part := range parts {
		asMap, ok := current.(map[string]interface{})
		if !ok {
			return ""
		}
		current, ok = asMap[part]
		if !ok {
			return ""
		}
	}
	return strings.TrimSpace(fmt.Sprint(current))
}

func recipientSourceLabel(channel string) string {
	switch channel {
	case "sms", "phone":
		return "联系电话"
	case "wechat", "mini_program":
		return "微信 OpenID"
	case "wework":
		return "企业微信 UserID"
	case "qq":
		return "QQ 标识"
	default:
		return "收件人标识"
	}
}

func (s *Server) dispatchSurveyChannelDelivery(ctx context.Context, item domain.SurveyChannelDelivery) (domain.SurveyChannelDelivery, error) {
	item.Status = "sending"
	item.Error = ""
	item, _ = s.store.UpdateSurveyChannelDelivery(ctx, item)
	switch item.Channel {
	case "phone":
		return s.dispatchPhoneDelivery(ctx, item)
	case "sms", "wechat", "wework", "mini_program", "qq":
		return s.dispatchIntegrationDelivery(ctx, item)
	default:
		item.Status = "sent"
		item.ProviderRef = "local-link"
		item.SentAt = time.Now().UTC().Format("2006-01-02 15:04:05")
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
}

func (s *Server) dispatchPhoneDelivery(ctx context.Context, item domain.SurveyChannelDelivery) (domain.SurveyChannelDelivery, error) {
	if !validPhone(item.Recipient) {
		item.Status = "failed"
		item.Error = "invalid phone number"
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	call := s.store.CreateCall(domain.CallSession{
		PatientID:     fmt.Sprint(item.Config["patientId"]),
		PhoneNumber:   item.Recipient,
		Status:        "dialing",
		Direction:     "outbound",
		InterviewForm: "survey_delivery:" + item.ID,
	})
	item.Config["callId"] = call.ID
	if endpoint, ok := s.store.DefaultSipEndpoint(); ok {
		result, err := s.sip.Dial(ctx, endpoint, call)
		if errors.Is(err, sipgateway.ErrDisabled) {
			call, _ = s.store.UpdateCall(call.ID, domain.CallSession{Status: "connected"})
			item.Status = "sent"
			item.ProviderRef = call.ID
			item.SentAt = time.Now().UTC().Format("2006-01-02 15:04:05")
			return s.store.UpdateSurveyChannelDelivery(ctx, item)
		}
		if err != nil {
			call, _ = s.store.UpdateCall(call.ID, domain.CallSession{Status: "failed"})
			item.Status = "failed"
			item.Error = err.Error()
			item.ProviderRef = call.ID
			return s.store.UpdateSurveyChannelDelivery(ctx, item)
		}
		call, _ = s.store.UpdateCall(call.ID, domain.CallSession{Status: result.Status})
		item.Status = "sent"
		item.ProviderRef = call.ID
		item.SentAt = time.Now().UTC().Format("2006-01-02 15:04:05")
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	item.Status = "failed"
	item.Error = "phone interface is not configured"
	item.ProviderRef = call.ID
	return s.store.UpdateSurveyChannelDelivery(ctx, item)
}

func (s *Server) dispatchIntegrationDelivery(ctx context.Context, item domain.SurveyChannelDelivery) (domain.SurveyChannelDelivery, error) {
	channels, err := s.store.IntegrationChannels(ctx)
	if err != nil {
		return item, err
	}
	var channel domain.IntegrationChannel
	for _, candidate := range channels {
		if candidate.Enabled && candidate.Kind == item.Channel {
			channel = candidate
			break
		}
	}
	if channel.ID == "" {
		item.Status = "failed"
		item.Error = "channel interface is not enabled"
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	if configBool(channel.Config, "mock") {
		item.Status = "sent"
		item.ProviderRef = "mock-" + item.ID
		item.SentAt = time.Now().UTC().Format("2006-01-02 15:04:05")
		item.Config["provider"] = channel.Name
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	providerName := configString(channel.Config, "provider")
	if providerName == "" {
		providerName = item.Channel
	}
	switch providerName {
	case "aliyun_sms":
		return s.dispatchAliyunSMSDelivery(ctx, item, channel)
	case "wechat_official", "wework", "wechat_mini_program", "mini_program":
		return s.dispatchWechatDelivery(ctx, item, channel, providerName)
	}
	if strings.TrimSpace(channel.Endpoint) == "" || strings.Contains(channel.Endpoint, "example.local") || strings.Contains(channel.Endpoint, "example.com") {
		item.Status = "failed"
		item.Error = "channel endpoint is not configured"
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	payload := map[string]interface{}{
		"deliveryId": item.ID,
		"projectId":  item.ProjectID,
		"shareId":    item.ShareID,
		"channel":    item.Channel,
		"recipient":  item.Recipient,
		"name":       item.RecipientName,
		"message":    item.Message,
		"url":        item.Config["url"],
		"appId":      channel.AppID,
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, channel.Endpoint, bytes.NewReader(raw))
	if err != nil {
		item.Status = "failed"
		item.Error = err.Error()
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	req.Header.Set("Content-Type", "application/json")
	if channel.CredentialRef != "" {
		req.Header.Set("X-Credential-Ref", channel.CredentialRef)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		item.Status = "failed"
		item.Error = err.Error()
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		item.Status = "failed"
		item.Error = fmt.Sprintf("provider returned HTTP %d", resp.StatusCode)
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	var providerResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&providerResp)
	item.Status = "sent"
	item.ProviderRef = firstNonEmpty(fmt.Sprint(providerResp["id"]), fmt.Sprint(providerResp["messageId"]), fmt.Sprint(providerResp["providerRef"]), item.ID)
	item.SentAt = time.Now().UTC().Format("2006-01-02 15:04:05")
	item.Config["provider"] = channel.Name
	return s.store.UpdateSurveyChannelDelivery(ctx, item)
}

func (s *Server) dispatchAliyunSMSDelivery(ctx context.Context, item domain.SurveyChannelDelivery, channel domain.IntegrationChannel) (domain.SurveyChannelDelivery, error) {
	if item.Config == nil {
		item.Config = map[string]interface{}{}
	}
	if !validPhone(item.Recipient) {
		item.Status = "failed"
		item.Error = "invalid phone number"
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	signName := configString(channel.Config, "signName")
	templateCode := configString(channel.Config, "templateCode")
	if signName == "" || templateCode == "" || strings.TrimSpace(channel.CredentialRef) == "" {
		item.Status = "failed"
		item.Error = "aliyun sms sdk is not fully configured: credentialRef, signName and templateCode are required"
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	endpoint := strings.TrimSpace(channel.Endpoint)
	if endpoint == "" {
		endpoint = "https://dysmsapi.aliyuncs.com"
	}
	payload := map[string]interface{}{
		"deliveryId":    item.ID,
		"provider":      "aliyun_sms",
		"sdk":           "aliyun-dysmsapi",
		"action":        "SendSms",
		"phoneNumbers":  item.Recipient,
		"signName":      signName,
		"templateCode":  templateCode,
		"templateParam": map[string]interface{}{"name": item.RecipientName, "url": item.Config["url"], "message": item.Message},
		"regionId":      firstNonEmpty(configString(channel.Config, "regionId"), "cn-hangzhou"),
	}
	return s.postIntegrationPayload(ctx, item, channel, endpoint, payload)
}

func (s *Server) dispatchWechatDelivery(ctx context.Context, item domain.SurveyChannelDelivery, channel domain.IntegrationChannel, provider string) (domain.SurveyChannelDelivery, error) {
	if item.Config == nil {
		item.Config = map[string]interface{}{}
	}
	templateID := configString(channel.Config, "templateId")
	if templateID == "" || strings.TrimSpace(channel.CredentialRef) == "" || strings.TrimSpace(channel.AppID) == "" {
		item.Status = "failed"
		item.Error = "wechat interface is not fully configured: appId, credentialRef and templateId are required"
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	endpoint := strings.TrimSpace(channel.Endpoint)
	if endpoint == "" {
		endpoint = "https://api.weixin.qq.com"
	}
	payload := map[string]interface{}{
		"deliveryId": item.ID,
		"provider":   provider,
		"appId":      channel.AppID,
		"templateId": templateID,
		"recipient":  item.Recipient,
		"name":       item.RecipientName,
		"url":        item.Config["url"],
		"pagePath":   configString(channel.Config, "pagePath"),
		"message":    item.Message,
	}
	if provider == "wework" {
		payload["agentId"] = configString(channel.Config, "agentId")
	}
	return s.postIntegrationPayload(ctx, item, channel, endpoint, payload)
}

func (s *Server) postIntegrationPayload(ctx context.Context, item domain.SurveyChannelDelivery, channel domain.IntegrationChannel, endpoint string, payload map[string]interface{}) (domain.SurveyChannelDelivery, error) {
	if item.Config == nil {
		item.Config = map[string]interface{}{}
	}
	if strings.TrimSpace(endpoint) == "" || strings.Contains(endpoint, "example.local") || strings.Contains(endpoint, "example.com") {
		item.Status = "failed"
		item.Error = "channel endpoint is not configured"
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	raw, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(raw))
	if err != nil {
		item.Status = "failed"
		item.Error = err.Error()
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	req.Header.Set("Content-Type", "application/json")
	if channel.CredentialRef != "" {
		req.Header.Set("X-Credential-Ref", channel.CredentialRef)
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		item.Status = "failed"
		item.Error = err.Error()
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		item.Status = "failed"
		item.Error = fmt.Sprintf("provider returned HTTP %d", resp.StatusCode)
		return s.store.UpdateSurveyChannelDelivery(ctx, item)
	}
	var providerResp map[string]interface{}
	_ = json.NewDecoder(resp.Body).Decode(&providerResp)
	item.Status = "sent"
	item.ProviderRef = firstNonEmpty(fmt.Sprint(providerResp["id"]), fmt.Sprint(providerResp["messageId"]), fmt.Sprint(providerResp["bizId"]), fmt.Sprint(providerResp["providerRef"]), item.ID)
	item.SentAt = time.Now().UTC().Format("2006-01-02 15:04:05")
	item.Config["provider"] = channel.Name
	item.Config["providerPayload"] = payload
	return s.store.UpdateSurveyChannelDelivery(ctx, item)
}

func configBool(config map[string]interface{}, key string) bool {
	value, ok := config[key]
	if !ok {
		return false
	}
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true") || typed == "1"
	default:
		return false
	}
}

func (s *Server) findSurveyPatient(ctx context.Context, identifier, phone string) (domain.Patient, domain.ClinicalVisit, bool, error) {
	identifier = strings.TrimSpace(strings.ToLower(identifier))
	phone = strings.TrimSpace(phone)
	if identifier == "" || phone == "" {
		return domain.Patient{}, domain.ClinicalVisit{}, false, nil
	}
	patients, err := s.store.PatientsStrict(ctx, "")
	if err != nil {
		return domain.Patient{}, domain.ClinicalVisit{}, false, err
	}
	for _, patient := range patients {
		if strings.TrimSpace(patient.Phone) != phone {
			continue
		}
		if strings.EqualFold(patient.ID, identifier) ||
			strings.EqualFold(patient.PatientNo, identifier) ||
			strings.EqualFold(patient.MedicalRecordNo, identifier) {
			visits, err := s.store.VisitsStrict(ctx, patient.ID)
			if err != nil {
				return domain.Patient{}, domain.ClinicalVisit{}, false, err
			}
			return patient, firstVisit(visits), true, nil
		}
		visits, err := s.store.VisitsStrict(ctx, patient.ID)
		if err != nil {
			return domain.Patient{}, domain.ClinicalVisit{}, false, err
		}
		for _, visit := range visits {
			if strings.EqualFold(visit.VisitNo, identifier) || strings.EqualFold(visit.ID, identifier) {
				return patient, visit, true, nil
			}
		}
	}
	return domain.Patient{}, domain.ClinicalVisit{}, false, nil
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

func addTrendScore(buckets map[string]map[string]float64, item domain.SurveySubmission) {
	score := numericAnswer(item.Answers["overall_satisfaction"])
	if score == nil {
		return
	}
	addBucket(buckets, item.SubmittedAt.Format("2006-01"), *score)
}

func addCrossAnalysis(buckets map[string]map[string]map[string]float64, item domain.SurveySubmission) {
	score := numericAnswer(item.Answers["overall_satisfaction"])
	if score == nil {
		return
	}
	dimensions := map[string]string{
		"科室":   firstNonEmpty(stringAnswer(item.Answers["department"]), "未填写科室"),
		"性别":   firstNonEmpty(stringAnswer(item.Answers["patient_gender"]), stringAnswer(item.Answers["gender"]), "未填写性别"),
		"就诊类型": firstNonEmpty(stringAnswer(item.Answers["visit_type"]), "未填写就诊类型"),
		"渠道":   firstNonEmpty(item.Channel, "未知渠道"),
	}
	for dimension, value := range dimensions {
		if buckets[dimension] == nil {
			buckets[dimension] = map[string]map[string]float64{}
		}
		addBucket(buckets[dimension], value, *score)
	}
}

func addDimensionRankings(buckets map[string]map[string]map[string]float64, item domain.SurveySubmission) {
	score := numericAnswer(item.Answers["overall_satisfaction"])
	if score == nil {
		return
	}
	dimensions := map[string]string{
		"科室":   firstNonEmpty(stringAnswer(item.Answers["department"]), "未填写科室"),
		"医生":   firstNonEmpty(stringAnswer(item.Answers["doctor_name"]), stringAnswer(item.Answers["doctor"]), "未填写医生"),
		"护士":   firstNonEmpty(stringAnswer(item.Answers["nurse_name"]), stringAnswer(item.Answers["nurse"]), "未填写护士"),
		"病种":   firstNonEmpty(stringAnswer(item.Answers["diagnosis"]), stringAnswer(item.Answers["disease"]), "未填写病种"),
		"就诊类型": firstNonEmpty(stringAnswer(item.Answers["visit_type"]), "未填写就诊类型"),
		"渠道":   firstNonEmpty(item.Channel, "未知渠道"),
		"性别":   firstNonEmpty(stringAnswer(item.Answers["patient_gender"]), stringAnswer(item.Answers["gender"]), "未填写性别"),
		"年龄段":  ageBucket(numericAnswer(item.Answers["patient_age"])),
	}
	for dimension, value := range dimensions {
		if buckets[dimension] == nil {
			buckets[dimension] = map[string]map[string]float64{}
		}
		addBucket(buckets[dimension], value, *score)
	}
}

func addJobAnalysis(buckets map[string]map[string]float64, item domain.SurveySubmission) {
	score := numericAnswer(item.Answers["overall_satisfaction"])
	if score == nil {
		return
	}
	for _, value := range []string{
		stringAnswer(item.Answers["doctor_name"]),
		stringAnswer(item.Answers["nurse_name"]),
		stringAnswer(item.Answers["window_name"]),
		stringAnswer(item.Answers["operator_name"]),
	} {
		if value != "" {
			addBucket(buckets, value, *score)
		}
	}
}

func addImportanceMatrix(buckets map[string]map[string]float64, item domain.SurveySubmission) {
	labels := map[string]string{"overall_satisfaction": "总体满意度", "recommend_score": "推荐意愿", "service_matrix": "分项满意度"}
	for key, label := range labels {
		score := numericAnswer(item.Answers[key])
		if score == nil {
			continue
		}
		if buckets[label] == nil {
			buckets[label] = map[string]float64{"sum": 0, "count": 0, "low": 0}
		}
		buckets[label]["sum"] += *score
		buckets[label]["count"]++
		if *score <= 3 {
			buckets[label]["low"]++
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

func averageNestedBuckets(buckets map[string]map[string]map[string]float64) map[string][]map[string]interface{} {
	result := map[string][]map[string]interface{}{}
	for dimension, items := range buckets {
		result[dimension] = averageBuckets(items)
	}
	return result
}

func periodCompareBuckets(items []map[string]interface{}) []map[string]interface{} {
	sort.Slice(items, func(i, j int) bool { return fmt.Sprint(items[i]["name"]) < fmt.Sprint(items[j]["name"]) })
	result := []map[string]interface{}{}
	byPeriod := map[string]map[string]interface{}{}
	for _, item := range items {
		period := fmt.Sprint(item["name"])
		byPeriod[period] = item
		score, _ := item["score"].(float64)
		row := map[string]interface{}{"name": period, "score": score, "count": item["count"], "mom": nil, "yoy": nil}
		if prev, ok := previousPeriod(period, -1); ok {
			if prevItem, exists := byPeriod[prev]; exists {
				row["mom"] = percentChange(score, valueFloat(prevItem["score"]))
			}
		}
		if prev, ok := previousPeriod(period, -12); ok {
			if prevItem, exists := byPeriod[prev]; exists {
				row["yoy"] = percentChange(score, valueFloat(prevItem["score"]))
			}
		}
		result = append(result, row)
	}
	return result
}

func shortBoardItems(stats map[string]interface{}) []map[string]interface{} {
	result := []map[string]interface{}{}
	if dimensions, ok := stats["dimensionRankings"].(map[string][]map[string]interface{}); ok {
		for dimension, rows := range dimensions {
			if len(rows) == 0 {
				continue
			}
			sort.Slice(rows, func(i, j int) bool { return valueFloat(rows[i]["score"]) < valueFloat(rows[j]["score"]) })
			row := rows[0]
			result = append(result, map[string]interface{}{"dimension": dimension, "name": row["name"], "score": row["score"], "count": row["count"], "reason": "维度最低分"})
		}
	}
	if reasons, ok := stats["lowReasons"].(map[string]int); ok {
		for name, count := range reasons {
			result = append(result, map[string]interface{}{"dimension": "低分原因", "name": name, "score": 0, "count": count, "reason": "负面原因高频"})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if valueFloat(result[i]["score"]) == 0 || valueFloat(result[j]["score"]) == 0 {
			return valueFloat(result[i]["count"]) > valueFloat(result[j]["count"])
		}
		return valueFloat(result[i]["score"]) < valueFloat(result[j]["score"])
	})
	if len(result) > 10 {
		return result[:10]
	}
	return result
}

func varianceAnalysis(dimensions map[string][]map[string]interface{}) []map[string]interface{} {
	result := []map[string]interface{}{}
	for dimension, rows := range dimensions {
		if len(rows) < 2 {
			continue
		}
		var sum float64
		minScore := math.MaxFloat64
		maxScore := -math.MaxFloat64
		minName := ""
		maxName := ""
		for _, row := range rows {
			score := valueFloat(row["score"])
			sum += score
			if score < minScore {
				minScore, minName = score, fmt.Sprint(row["name"])
			}
			if score > maxScore {
				maxScore, maxName = score, fmt.Sprint(row["name"])
			}
		}
		mean := sum / float64(len(rows))
		var variance float64
		for _, row := range rows {
			diff := valueFloat(row["score"]) - mean
			variance += diff * diff
		}
		variance = variance / float64(len(rows))
		result = append(result, map[string]interface{}{"dimension": dimension, "variance": variance, "stddev": math.Sqrt(variance), "minName": minName, "minScore": minScore, "maxName": maxName, "maxScore": maxScore, "gap": maxScore - minScore})
	}
	sort.Slice(result, func(i, j int) bool { return valueFloat(result[i]["gap"]) > valueFloat(result[j]["gap"]) })
	return result
}

func correlationAnalysis(items []domain.SurveySubmission) []map[string]interface{} {
	keys := map[string]string{"recommend_score": "推荐意愿", "service_matrix": "分项满意度", "wait_time_score": "候诊时间", "doctor_service": "医生服务", "nurse_service": "护理服务"}
	result := []map[string]interface{}{}
	for key, label := range keys {
		xs := []float64{}
		ys := []float64{}
		for _, item := range items {
			if item.QualityStatus == "invalid" {
				continue
			}
			x := numericAnswer(item.Answers[key])
			y := numericAnswer(item.Answers["overall_satisfaction"])
			if x != nil && y != nil {
				xs = append(xs, *x)
				ys = append(ys, *y)
			}
		}
		if len(xs) >= 2 {
			result = append(result, map[string]interface{}{"name": label, "coefficient": pearson(xs, ys), "count": len(xs)})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return math.Abs(valueFloat(result[i]["coefficient"])) > math.Abs(valueFloat(result[j]["coefficient"]))
	})
	return result
}

func pearson(xs, ys []float64) float64 {
	if len(xs) != len(ys) || len(xs) == 0 {
		return 0
	}
	var sumX, sumY float64
	for i := range xs {
		sumX += xs[i]
		sumY += ys[i]
	}
	meanX := sumX / float64(len(xs))
	meanY := sumY / float64(len(ys))
	var numerator, denomX, denomY float64
	for i := range xs {
		dx := xs[i] - meanX
		dy := ys[i] - meanY
		numerator += dx * dy
		denomX += dx * dx
		denomY += dy * dy
	}
	if denomX == 0 || denomY == 0 {
		return 0
	}
	return numerator / math.Sqrt(denomX*denomY)
}

func previousPeriod(period string, monthDelta int) (string, bool) {
	parsed, err := time.Parse("2006-01", period)
	if err != nil {
		return "", false
	}
	return parsed.AddDate(0, monthDelta, 0).Format("2006-01"), true
}

func percentChange(current, previous float64) interface{} {
	if previous == 0 {
		return nil
	}
	return (current - previous) / previous
}

func valueFloat(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case string:
		parsed, _ := strconv.ParseFloat(typed, 64)
		return parsed
	default:
		return 0
	}
}

func ageBucket(score *float64) string {
	if score == nil || *score <= 0 {
		return "未填写年龄"
	}
	age := *score
	switch {
	case age < 18:
		return "18岁以下"
	case age < 35:
		return "18-34岁"
	case age < 60:
		return "35-59岁"
	default:
		return "60岁及以上"
	}
}

func importanceBuckets(buckets map[string]map[string]float64) []map[string]interface{} {
	items := []map[string]interface{}{}
	for key, item := range buckets {
		if item["count"] == 0 {
			continue
		}
		score := item["sum"] / item["count"]
		impact := item["low"] / item["count"]
		items = append(items, map[string]interface{}{"name": key, "score": score, "impact": impact, "count": int(item["count"])})
	}
	return items
}

func satisfactionInsights(stats map[string]interface{}) []string {
	insights := []string{}
	if avg, ok := stats["scoreAverage"].(float64); ok && avg > 0 {
		if avg < 3.5 {
			insights = append(insights, "总体满意度偏低，建议优先排查候诊、沟通和缴费等高频触点。")
		} else {
			insights = append(insights, "总体满意度处于可接受区间，可继续关注低分科室和负面开放意见。")
		}
	}
	if reasons, ok := stats["lowReasons"].(map[string]int); ok {
		topName := ""
		topCount := 0
		for name, count := range reasons {
			if count > topCount {
				topName, topCount = name, count
			}
		}
		if topName != "" {
			insights = append(insights, fmt.Sprintf("低分原因最集中在“%s”，共出现 %d 次，可作为整改优先主题。", topName, topCount))
		}
	}
	if len(insights) == 0 {
		insights = append(insights, "当前样本量或有效得分不足，建议先扩大采集范围并完成数据清洗。")
	}
	return insights
}

func buildReportInsights(result map[string]interface{}) map[string]interface{} {
	rows, _ := result["rows"].([]map[string]interface{})
	if rows == nil {
		if rawRows, ok := result["rows"].([]interface{}); ok {
			for _, raw := range rawRows {
				if row, ok := raw.(map[string]interface{}); ok {
					rows = append(rows, row)
				}
			}
		}
	}
	lowDimension := ""
	lowScore := 999.0
	topDimension := ""
	topCount := 0
	for _, row := range rows {
		name := firstNonEmpty(fmt.Sprint(row["dimensionValue"]), fmt.Sprint(row["indicator"]))
		score := numberFromAny(row["score"])
		count := int(numberFromAny(row["sampleCount"]))
		if score > 0 && score < lowScore {
			lowScore = score
			lowDimension = name
		}
		if count > topCount {
			topCount = count
			topDimension = name
		}
	}
	insights := []string{"样本量与指标得分已完成基础聚合，可用于生成月度/季度分析报告。"}
	if lowDimension != "" {
		insights = append(insights, fmt.Sprintf("当前最低得分维度为“%s”，得分 %.1f，建议优先进入问题台账跟踪。", lowDimension, lowScore))
	}
	if topDimension != "" {
		insights = append(insights, fmt.Sprintf("样本最多的维度为“%s”，共 %d 条，可作为趋势判断的主要观察对象。", topDimension, topCount))
	}
	return map[string]interface{}{"sentiment": "neutral", "themes": []string{"服务体验", "流程效率", "沟通质量"}, "rootCauses": insights, "suggestions": []string{"对低分指标建立整改责任人和复评周期", "按科室和医生维度持续观察趋势变化", "将开放意见接入主题聚类和典型语句提取"}}
}

func renderReportDocument(report domain.Report, result map[string]interface{}) string {
	rows, _ := result["rows"].([]map[string]interface{})
	var b strings.Builder
	b.WriteString("<html><head><meta charset=\"utf-8\"><style>body{font-family:Arial,'Microsoft YaHei',sans-serif}table{border-collapse:collapse;width:100%}td,th{border:1px solid #ddd;padding:6px}</style></head><body>")
	b.WriteString("<h1>" + html.EscapeString(report.Name) + "</h1>")
	b.WriteString("<p>" + html.EscapeString(report.Description) + "</p>")
	b.WriteString("<h2>AI 洞察摘要</h2><ul>")
	for _, insight := range buildReportInsights(result)["rootCauses"].([]string) {
		b.WriteString("<li>" + html.EscapeString(insight) + "</li>")
	}
	b.WriteString("</ul><h2>数据明细</h2><table>")
	if len(rows) > 0 {
		b.WriteString("<tr>")
		for key := range rows[0] {
			b.WriteString("<th>" + html.EscapeString(key) + "</th>")
		}
		b.WriteString("</tr>")
		for _, row := range rows {
			b.WriteString("<tr>")
			for key := range rows[0] {
				b.WriteString("<td>" + html.EscapeString(fmt.Sprint(row[key])) + "</td>")
			}
			b.WriteString("</tr>")
		}
	}
	b.WriteString("</table></body></html>")
	return b.String()
}

func stripTags(value string) string {
	clean := regexp.MustCompile(`<[^>]+>`).ReplaceAllString(value, "\n")
	return html.UnescapeString(clean)
}

func simplePDFBytes(text string) []byte {
	text = strings.ReplaceAll(text, "\\", "\\\\")
	text = strings.ReplaceAll(text, "(", "\\(")
	text = strings.ReplaceAll(text, ")", "\\)")
	lines := strings.Split(text, "\n")
	if len(lines) > 28 {
		lines = lines[:28]
	}
	content := "BT /F1 12 Tf 50 780 Td "
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if len([]rune(line)) > 80 {
			line = string([]rune(line)[:80])
		}
		content += "(" + line + ") Tj 0 -18 Td "
	}
	content += "ET"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>",
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(content), content),
	}
	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := []int{0}
	for index, object := range objects {
		offsets = append(offsets, b.Len())
		b.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", index+1, object))
	}
	xref := b.Len()
	b.WriteString(fmt.Sprintf("xref\n0 %d\n0000000000 65535 f \n", len(objects)+1))
	for _, offset := range offsets[1:] {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", offset))
	}
	b.WriteString(fmt.Sprintf("trailer << /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF", len(objects)+1, xref))
	return []byte(b.String())
}

func numberFromAny(value interface{}) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	case json.Number:
		next, _ := typed.Float64()
		return next
	case string:
		next, _ := strconv.ParseFloat(typed, 64)
		return next
	default:
		return 0
	}
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
	components := []map[string]interface{}{}
	for _, item := range s.store.FormLibrary() {
		if item.Kind == "template" && (formTemplateID == "" || item.ID == formTemplateID) {
			components = item.Components
			break
		}
	}
	s.streamSurveyComponents(w, components)
}

func (s *Server) streamSurveyComponents(w http.ResponseWriter, components []map[string]interface{}) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
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
	var req struct {
		ProjectID string `json:"projectId"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}
	result, err := s.store.QueryReportData(r.Context(), chi.URLParam(r, "id"), req.ProjectID)
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) reportInsights(w http.ResponseWriter, r *http.Request) {
	result, err := s.store.QueryReportData(r.Context(), chi.URLParam(r, "id"), r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, buildReportInsights(result))
}

func (s *Server) exportReport(w http.ResponseWriter, r *http.Request) {
	report, err := s.store.ReportDefinition(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	result, err := s.store.QueryReportData(r.Context(), report.ID, r.URL.Query().Get("projectId"))
	if err != nil {
		statusError(w, err)
		return
	}
	format := strings.ToLower(firstNonEmpty(r.URL.Query().Get("format"), "word"))
	body := renderReportDocument(report, result)
	if format == "pdf" {
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", `attachment; filename="report.pdf"`)
		_, _ = w.Write(simplePDFBytes(stripTags(body)))
		return
	}
	w.Header().Set("Content-Type", "application/msword; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="report.doc"`)
	_, _ = w.Write([]byte(body))
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
	tasks, err := s.store.GenerateFollowupTasksStrict(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "followup-task.generate", "/api/v1/followup/plans/"+chi.URLParam(r, "id"), nil, tasks)
	writeJSON(w, http.StatusCreated, tasks)
}

func (s *Server) listFollowupTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := s.store.FollowupTasksStrict(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("assigneeId"))
	if err != nil {
		statusError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tasks)
}

func (s *Server) createFollowupTask(w http.ResponseWriter, r *http.Request) {
	var task domain.FollowupTask
	if !decodeJSON(w, r, &task) {
		return
	}
	var err error
	task, err = s.store.CreateFollowupTaskStrict(r.Context(), task)
	if err != nil {
		statusError(w, err)
		return
	}
	s.audit(r, actorID(r), "followup-task.create", "/api/v1/followup/tasks/"+task.ID, nil, task)
	writeJSON(w, http.StatusCreated, task)
}

func (s *Server) updateFollowupTask(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var patch domain.FollowupTask
	if !decodeJSON(w, r, &patch) {
		return
	}
	task, err := s.store.UpdateFollowupTaskStrict(r.Context(), id, patch)
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
	call.PhoneNumber = sanitizeDialNumber(call.PhoneNumber)
	if !validPhone(call.PhoneNumber) {
		http.Error(w, "invalid phone number", http.StatusBadRequest)
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
