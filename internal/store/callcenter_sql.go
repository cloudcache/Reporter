package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
	"reporter/internal/domain"
)

func (s *Store) EnsureCallCenterDefaults(ctx context.Context) error {
	db, err := s.openConfiguredDB()
	if err != nil {
		return err
	}
	defer db.Close()
	defaults := []string{
		`INSERT INTO sip_endpoints (id, name, wss_url, domain, proxy, config_json) VALUES ('SIP001', '院内 WebRTC SIP 网关', 'wss://pbx.example.local/ws', 'call.example.local', 'sip:pbx.example.local;transport=wss', '{"enabled":false,"webrtc":true,"transport":"udp","bindHost":"0.0.0.0","trunkUri":"sip:{phone}@carrier.example.local"}') ON DUPLICATE KEY UPDATE name = VALUES(name), wss_url = VALUES(wss_url), domain = VALUES(domain), proxy = VALUES(proxy), config_json = VALUES(config_json)`,
		`INSERT INTO agent_seats (id, user_id, name, extension, sip_uri, status, skills_json) VALUES ('SEAT001', NULL, '默认随访坐席', '8001', 'sip:8001@call.example.local', 'available', '["followup","survey"]') ON DUPLICATE KEY UPDATE name = VALUES(name), extension = VALUES(extension), sip_uri = VALUES(sip_uri), status = VALUES(status), skills_json = VALUES(skills_json)`,
		`INSERT INTO model_providers (id, name, kind, mode, endpoint, model, credential_ref, config_json) VALUES ('LLM001', '院内大模型网关', 'openai-compatible', 'offline', 'https://llm.example.local/v1', 'medical-call-analyzer', 'secret://llm/primary', '{"supports_audio":true,"supports_json_schema":true,"audio_analysis":true}') ON DUPLICATE KEY UPDATE name = VALUES(name), kind = VALUES(kind), mode = VALUES(mode), endpoint = VALUES(endpoint), model = VALUES(model), credential_ref = VALUES(credential_ref), config_json = VALUES(config_json)`,
		`INSERT INTO model_providers (id, name, kind, mode, endpoint, model, credential_ref, config_json) VALUES ('LLM002', '实时语音识别与表单回填', 'realtime-asr', 'realtime', 'wss://llm.example.local/realtime', 'medical-realtime-asr', 'secret://llm/realtime', '{"partial_transcript":true,"form_autofill":true}') ON DUPLICATE KEY UPDATE name = VALUES(name), kind = VALUES(kind), mode = VALUES(mode), endpoint = VALUES(endpoint), model = VALUES(model), credential_ref = VALUES(credential_ref), config_json = VALUES(config_json)`,
	}
	for _, stmt := range defaults {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) SeatsStrict(ctx context.Context) ([]domain.AgentSeat, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT s.id, COALESCE(s.user_id, ''), COALESCE(u.username, ''), COALESCE(u.display_name, ''), s.name, s.extension, s.sip_uri, s.status, COALESCE(CAST(s.skills_json AS CHAR), '[]'), COALESCE(s.current_call_id, ''), s.created_at, s.updated_at FROM agent_seats s LEFT JOIN users u ON u.id = s.user_id ORDER BY s.created_at DESC, s.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.AgentSeat{}
	for rows.Next() {
		item, err := scanSeat(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SeatStrict(ctx context.Context, id string) (domain.AgentSeat, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.AgentSeat{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT s.id, COALESCE(s.user_id, ''), COALESCE(u.username, ''), COALESCE(u.display_name, ''), s.name, s.extension, s.sip_uri, s.status, COALESCE(CAST(s.skills_json AS CHAR), '[]'), COALESCE(s.current_call_id, ''), s.created_at, s.updated_at FROM agent_seats s LEFT JOIN users u ON u.id = s.user_id WHERE s.id = ?`, id)
	item, err := scanSeat(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.AgentSeat{}, false, nil
		}
		return domain.AgentSeat{}, false, err
	}
	return item, true, nil
}

func (s *Store) CreateSeatStrict(ctx context.Context, item domain.AgentSeat) (domain.AgentSeat, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.AgentSeat{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = "offline"
	}
	skillsJSON, err := json.Marshal(item.Skills)
	if err != nil {
		return domain.AgentSeat{}, err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO agent_seats (id, user_id, name, extension, sip_uri, status, skills_json, current_call_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, nullableString(item.UserID), item.Name, item.Extension, item.SipURI, item.Status, string(skillsJSON), nullableString(item.CurrentCall))
	if err != nil {
		return domain.AgentSeat{}, err
	}
	saved, _, err := s.SeatStrict(ctx, item.ID)
	return saved, err
}

func (s *Store) UpdateSeatStrict(ctx context.Context, id string, patch domain.AgentSeat) (domain.AgentSeat, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.AgentSeat{}, err
	}
	defer db.Close()
	if strings.TrimSpace(patch.Status) == "" {
		patch.Status = "offline"
	}
	skillsJSON, err := json.Marshal(patch.Skills)
	if err != nil {
		return domain.AgentSeat{}, err
	}
	result, err := db.ExecContext(ctx, `UPDATE agent_seats SET user_id = ?, name = ?, extension = ?, sip_uri = ?, status = ?, skills_json = ?, current_call_id = ? WHERE id = ?`, nullableString(patch.UserID), patch.Name, patch.Extension, patch.SipURI, patch.Status, string(skillsJSON), nullableString(patch.CurrentCall), id)
	if err != nil {
		return domain.AgentSeat{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return domain.AgentSeat{}, ErrNotFound
	}
	saved, _, err := s.SeatStrict(ctx, id)
	return saved, err
}

func (s *Store) DeleteSeatStrict(ctx context.Context, id string) (domain.AgentSeat, error) {
	before, ok, err := s.SeatStrict(ctx, id)
	if err != nil {
		return domain.AgentSeat{}, err
	}
	if !ok {
		return domain.AgentSeat{}, ErrNotFound
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.AgentSeat{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DELETE FROM agent_seats WHERE id = ?`, id); err != nil {
		return domain.AgentSeat{}, err
	}
	return before, nil
}

func (s *Store) SipEndpointsStrict(ctx context.Context) ([]domain.SipEndpoint, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, name, wss_url, domain, COALESCE(proxy, ''), COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM sip_endpoints ORDER BY created_at DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.SipEndpoint{}
	for rows.Next() {
		item, err := scanSipEndpoint(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) SipEndpointStrict(ctx context.Context, id string) (domain.SipEndpoint, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.SipEndpoint{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT id, name, wss_url, domain, COALESCE(proxy, ''), COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM sip_endpoints WHERE id = ?`, id)
	item, err := scanSipEndpoint(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.SipEndpoint{}, false, nil
		}
		return domain.SipEndpoint{}, false, err
	}
	return item, true, nil
}

func (s *Store) DefaultSipEndpointStrict(ctx context.Context) (domain.SipEndpoint, bool, error) {
	items, err := s.SipEndpointsStrict(ctx)
	if err != nil {
		return domain.SipEndpoint{}, false, err
	}
	for _, item := range items {
		if sipConfigEnabled(item.Config) {
			return item, true, nil
		}
	}
	for _, item := range items {
		if item.ID == "SIP001" {
			return item, true, nil
		}
	}
	if len(items) > 0 {
		return items[0], true, nil
	}
	return domain.SipEndpoint{}, false, nil
}

func (s *Store) CreateSipEndpointStrict(ctx context.Context, item domain.SipEndpoint) (domain.SipEndpoint, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.SipEndpoint{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	configJSON, err := json.Marshal(nonNilMap(item.Config))
	if err != nil {
		return domain.SipEndpoint{}, err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO sip_endpoints (id, name, wss_url, domain, proxy, config_json) VALUES (?, ?, ?, ?, ?, ?)`, item.ID, item.Name, item.WSSURL, item.Domain, nullableString(item.Proxy), string(configJSON))
	if err != nil {
		return domain.SipEndpoint{}, err
	}
	saved, _, err := s.SipEndpointStrict(ctx, item.ID)
	return saved, err
}

func (s *Store) UpdateSipEndpointStrict(ctx context.Context, id string, patch domain.SipEndpoint) (domain.SipEndpoint, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.SipEndpoint{}, err
	}
	defer db.Close()
	configJSON, err := json.Marshal(nonNilMap(patch.Config))
	if err != nil {
		return domain.SipEndpoint{}, err
	}
	result, err := db.ExecContext(ctx, `UPDATE sip_endpoints SET name = ?, wss_url = ?, domain = ?, proxy = ?, config_json = ? WHERE id = ?`, patch.Name, patch.WSSURL, patch.Domain, nullableString(patch.Proxy), string(configJSON), id)
	if err != nil {
		return domain.SipEndpoint{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return domain.SipEndpoint{}, ErrNotFound
	}
	saved, _, err := s.SipEndpointStrict(ctx, id)
	return saved, err
}

func (s *Store) DeleteSipEndpointStrict(ctx context.Context, id string) (domain.SipEndpoint, error) {
	before, ok, err := s.SipEndpointStrict(ctx, id)
	if err != nil {
		return domain.SipEndpoint{}, err
	}
	if !ok {
		return domain.SipEndpoint{}, ErrNotFound
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.SipEndpoint{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DELETE FROM sip_endpoints WHERE id = ?`, id); err != nil {
		return domain.SipEndpoint{}, err
	}
	return before, nil
}

func (s *Store) CallsStrict(ctx context.Context) ([]domain.CallSession, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, seat_id, COALESCE(patient_id, ''), direction, phone_number, status, started_at, ended_at, COALESCE(recording_id, ''), COALESCE(transcript_id, ''), COALESCE(analysis_id, ''), COALESCE(interview_form, '') FROM call_sessions ORDER BY started_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.CallSession{}
	for rows.Next() {
		item, err := scanCallSession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateCallStrict(ctx context.Context, item domain.CallSession) (domain.CallSession, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.CallSession{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.SeatID) == "" {
		seatID, err := defaultSeatID(ctx, db)
		if err != nil {
			return domain.CallSession{}, err
		}
		item.SeatID = seatID
	}
	if strings.TrimSpace(item.Direction) == "" {
		item.Direction = "outbound"
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = "dialing"
	}
	_, err = db.ExecContext(ctx, `INSERT INTO call_sessions (id, seat_id, patient_id, direction, phone_number, status, interview_form) VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, item.SeatID, nullableString(item.PatientID), item.Direction, item.PhoneNumber, item.Status, nullableString(item.InterviewForm))
	if err != nil {
		return domain.CallSession{}, err
	}
	return s.callStrict(ctx, item.ID)
}

func (s *Store) UpdateCallStrict(ctx context.Context, id string, patch domain.CallSession) (domain.CallSession, error) {
	current, err := s.callStrict(ctx, id)
	if err != nil {
		return domain.CallSession{}, err
	}
	if patch.Status != "" {
		current.Status = patch.Status
	}
	if patch.RecordingID != "" {
		current.RecordingID = patch.RecordingID
	}
	if patch.TranscriptID != "" {
		current.TranscriptID = patch.TranscriptID
	}
	if patch.AnalysisID != "" {
		current.AnalysisID = patch.AnalysisID
	}
	if !patch.EndedAt.IsZero() {
		current.EndedAt = patch.EndedAt
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.CallSession{}, err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `UPDATE call_sessions SET status = ?, ended_at = ?, recording_id = ?, transcript_id = ?, analysis_id = ? WHERE id = ?`, current.Status, nullableTime(current.EndedAt), nullableString(current.RecordingID), nullableString(current.TranscriptID), nullableString(current.AnalysisID), id)
	if err != nil {
		return domain.CallSession{}, err
	}
	return s.callStrict(ctx, id)
}

func (s *Store) callStrict(ctx context.Context, id string) (domain.CallSession, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.CallSession{}, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT id, seat_id, COALESCE(patient_id, ''), direction, phone_number, status, started_at, ended_at, COALESCE(recording_id, ''), COALESCE(transcript_id, ''), COALESCE(analysis_id, ''), COALESCE(interview_form, '') FROM call_sessions WHERE id = ?`, id)
	item, err := scanCallSession(row)
	if err != nil {
		return domain.CallSession{}, err
	}
	return item, nil
}

func (s *Store) RecordingsStrict(ctx context.Context) ([]domain.Recording, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, call_id, storage_uri, duration, COALESCE(filename, ''), COALESCE(mime_type, ''), size_bytes, source, backend, COALESCE(object_name, ''), status, created_at FROM recordings ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Recording{}
	for rows.Next() {
		item, err := scanRecording(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateRecordingStrict(ctx context.Context, item domain.Recording) (domain.Recording, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Recording{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = "ready"
	}
	if strings.TrimSpace(item.Source) == "" {
		item.Source = "browser"
	}
	if strings.TrimSpace(item.Backend) == "" {
		item.Backend = "local"
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Recording{}, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO recordings (id, call_id, storage_uri, duration, filename, mime_type, size_bytes, source, backend, object_name, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.CallID, item.StorageURI, item.Duration, nullableString(item.Filename), nullableString(item.MimeType), item.SizeBytes, item.Source, item.Backend, nullableString(item.ObjectName), item.Status)
	if err == nil {
		_, err = tx.ExecContext(ctx, `UPDATE call_sessions SET recording_id = ?, status = CASE WHEN status IN ('dialing','connected','recording') THEN 'recorded' ELSE status END WHERE id = ?`, item.ID, item.CallID)
	}
	if err != nil {
		_ = tx.Rollback()
		return domain.Recording{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Recording{}, err
	}
	return item, nil
}

func (s *Store) ModelProvidersStrict(ctx context.Context) ([]domain.ModelProvider, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, name, kind, mode, endpoint, model, COALESCE(credential_ref, ''), COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM model_providers ORDER BY created_at DESC, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.ModelProvider{}
	for rows.Next() {
		item, err := scanModelProvider(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) ModelProviderStrict(ctx context.Context, id string) (domain.ModelProvider, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.ModelProvider{}, false, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT id, name, kind, mode, endpoint, model, COALESCE(credential_ref, ''), COALESCE(CAST(config_json AS CHAR), '{}'), created_at, updated_at FROM model_providers WHERE id = ?`, id)
	item, err := scanModelProvider(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.ModelProvider{}, false, nil
		}
		return domain.ModelProvider{}, false, err
	}
	return item, true, nil
}

func (s *Store) CreateModelProviderStrict(ctx context.Context, item domain.ModelProvider) (domain.ModelProvider, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.ModelProvider{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Mode) == "" {
		item.Mode = "offline"
	}
	configJSON, err := json.Marshal(nonNilMap(item.Config))
	if err != nil {
		return domain.ModelProvider{}, err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO model_providers (id, name, kind, mode, endpoint, model, credential_ref, config_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.Name, item.Kind, item.Mode, item.Endpoint, item.Model, nullableString(item.CredentialRef), string(configJSON))
	if err != nil {
		return domain.ModelProvider{}, err
	}
	saved, _, err := s.ModelProviderStrict(ctx, item.ID)
	return saved, err
}

func (s *Store) UpdateModelProviderStrict(ctx context.Context, id string, patch domain.ModelProvider) (domain.ModelProvider, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.ModelProvider{}, err
	}
	defer db.Close()
	if strings.TrimSpace(patch.Mode) == "" {
		patch.Mode = "offline"
	}
	configJSON, err := json.Marshal(nonNilMap(patch.Config))
	if err != nil {
		return domain.ModelProvider{}, err
	}
	result, err := db.ExecContext(ctx, `UPDATE model_providers SET name = ?, kind = ?, mode = ?, endpoint = ?, model = ?, credential_ref = ?, config_json = ? WHERE id = ?`, patch.Name, patch.Kind, patch.Mode, patch.Endpoint, patch.Model, nullableString(patch.CredentialRef), string(configJSON), id)
	if err != nil {
		return domain.ModelProvider{}, err
	}
	if affected, _ := result.RowsAffected(); affected == 0 {
		return domain.ModelProvider{}, ErrNotFound
	}
	saved, _, err := s.ModelProviderStrict(ctx, id)
	return saved, err
}

func (s *Store) DeleteModelProviderStrict(ctx context.Context, id string) (domain.ModelProvider, error) {
	before, ok, err := s.ModelProviderStrict(ctx, id)
	if err != nil {
		return domain.ModelProvider{}, err
	}
	if !ok {
		return domain.ModelProvider{}, ErrNotFound
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.ModelProvider{}, err
	}
	defer db.Close()
	if _, err := db.ExecContext(ctx, `DELETE FROM model_providers WHERE id = ?`, id); err != nil {
		return domain.ModelProvider{}, err
	}
	return before, nil
}

func (s *Store) AnalysesStrict(ctx context.Context) ([]domain.CallAnalysis, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, call_id, provider_id, COALESCE(patient_emotion, ''), COALESCE(true_satisfaction, 0), COALESCE(risk_level, ''), COALESCE(patient_status, ''), COALESCE(summary, ''), COALESCE(CAST(extracted_form_data AS CHAR), '{}'), created_at FROM call_analyses ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.CallAnalysis{}
	for rows.Next() {
		item, err := scanCallAnalysis(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) RealtimeAssistSessionsStrict(ctx context.Context) ([]domain.RealtimeAssistSession, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, call_id, COALESCE(patient_id, ''), form_id, provider_id, status, COALESCE(CAST(transcript_json AS CHAR), '[]'), COALESCE(CAST(form_draft_json AS CHAR), '{}'), COALESCE(last_suggestion, ''), created_at, updated_at FROM realtime_assist_sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.RealtimeAssistSession{}
	for rows.Next() {
		item, err := scanRealtimeSession(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateRealtimeAssistSessionStrict(ctx context.Context, item domain.RealtimeAssistSession) (domain.RealtimeAssistSession, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = "active"
	}
	transcriptJSON, err := json.Marshal(item.Transcript)
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	draftJSON, err := json.Marshal(nonNilMap(item.FormDraft))
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO realtime_assist_sessions (id, call_id, patient_id, form_id, provider_id, status, transcript_json, form_draft_json, last_suggestion) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.CallID, nullableString(item.PatientID), item.FormID, item.ProviderID, item.Status, string(transcriptJSON), string(draftJSON), nullableString(item.LastSuggestion))
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	return s.realtimeAssistSessionStrict(ctx, item.ID)
}

func (s *Store) AddRealtimeTranscriptStrict(ctx context.Context, id string, transcript domain.RealtimeTranscript, formPatch map[string]interface{}) (domain.RealtimeAssistSession, error) {
	current, err := s.realtimeAssistSessionStrict(ctx, id)
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	transcript.CreatedAt = time.Now().UTC()
	current.Transcript = append(current.Transcript, transcript)
	if current.FormDraft == nil {
		current.FormDraft = map[string]interface{}{}
	}
	for key, value := range formPatch {
		current.FormDraft[key] = value
	}
	if transcript.Text != "" {
		current.LastSuggestion = "已根据实时识别更新表单草稿"
	}
	transcriptJSON, err := json.Marshal(current.Transcript)
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	draftJSON, err := json.Marshal(nonNilMap(current.FormDraft))
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `UPDATE realtime_assist_sessions SET transcript_json = ?, form_draft_json = ?, last_suggestion = ? WHERE id = ?`, string(transcriptJSON), string(draftJSON), nullableString(current.LastSuggestion), id)
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	return s.realtimeAssistSessionStrict(ctx, id)
}

func (s *Store) OfflineAnalysisJobsStrict(ctx context.Context) ([]domain.OfflineAnalysisJob, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, call_id, recording_id, provider_id, status, COALESCE(CAST(result_json AS CHAR), '{}'), COALESCE(error, ''), created_at, updated_at FROM offline_analysis_jobs ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.OfflineAnalysisJob{}
	for rows.Next() {
		item, err := scanOfflineJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateOfflineAnalysisJobStrict(ctx context.Context, item domain.OfflineAnalysisJob) (domain.OfflineAnalysisJob, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.OfflineAnalysisJob{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = "queued"
	}
	resultJSON, err := json.Marshal(nonNilMap(item.Result))
	if err != nil {
		return domain.OfflineAnalysisJob{}, err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO offline_analysis_jobs (id, call_id, recording_id, provider_id, status, result_json, error) VALUES (?, ?, ?, ?, ?, ?, ?)`, item.ID, item.CallID, item.RecordingID, item.ProviderID, item.Status, string(resultJSON), nullableString(item.Error))
	if err != nil {
		return domain.OfflineAnalysisJob{}, err
	}
	items, err := s.OfflineAnalysisJobsStrict(ctx)
	if err != nil {
		return domain.OfflineAnalysisJob{}, err
	}
	for _, saved := range items {
		if saved.ID == item.ID {
			return saved, nil
		}
	}
	return item, nil
}

func (s *Store) InterviewsStrict(ctx context.Context) ([]domain.InterviewSession, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, patient_id, form_id, COALESCE(call_id, ''), mode, status, COALESCE(CAST(messages_json AS CHAR), '[]'), COALESCE(CAST(form_draft_json AS CHAR), '{}'), created_at, updated_at FROM interview_sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.InterviewSession{}
	for rows.Next() {
		item, err := scanInterview(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Store) CreateInterviewStrict(ctx context.Context, item domain.InterviewSession) (domain.InterviewSession, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.InterviewSession{}, err
	}
	defer db.Close()
	if strings.TrimSpace(item.ID) == "" {
		item.ID = uuid.NewString()
	}
	if strings.TrimSpace(item.Mode) == "" {
		item.Mode = "chat"
	}
	if strings.TrimSpace(item.Status) == "" {
		item.Status = "draft"
	}
	messagesJSON, err := json.Marshal(item.Messages)
	if err != nil {
		return domain.InterviewSession{}, err
	}
	draftJSON, err := json.Marshal(nonNilMap(item.FormDraft))
	if err != nil {
		return domain.InterviewSession{}, err
	}
	_, err = db.ExecContext(ctx, `INSERT INTO interview_sessions (id, patient_id, form_id, call_id, mode, status, messages_json, form_draft_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, item.ID, item.PatientID, item.FormID, nullableString(item.CallID), item.Mode, item.Status, string(messagesJSON), string(draftJSON))
	if err != nil {
		return domain.InterviewSession{}, err
	}
	items, err := s.InterviewsStrict(ctx)
	if err != nil {
		return domain.InterviewSession{}, err
	}
	for _, saved := range items {
		if saved.ID == item.ID {
			return saved, nil
		}
	}
	return item, nil
}

func (s *Store) realtimeAssistSessionStrict(ctx context.Context, id string) (domain.RealtimeAssistSession, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	defer db.Close()
	row := db.QueryRowContext(ctx, `SELECT id, call_id, COALESCE(patient_id, ''), form_id, provider_id, status, COALESCE(CAST(transcript_json AS CHAR), '[]'), COALESCE(CAST(form_draft_json AS CHAR), '{}'), COALESCE(last_suggestion, ''), created_at, updated_at FROM realtime_assist_sessions WHERE id = ?`, id)
	return scanRealtimeSession(row)
}

func defaultSeatID(ctx context.Context, db *sql.DB) (string, error) {
	var id string
	err := db.QueryRowContext(ctx, `SELECT id FROM agent_seats ORDER BY CASE WHEN status = 'available' THEN 0 ELSE 1 END, created_at LIMIT 1`).Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		}
		return "", err
	}
	return id, nil
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSeat(scanner rowScanner) (domain.AgentSeat, error) {
	var item domain.AgentSeat
	var skillsRaw string
	err := scanner.Scan(&item.ID, &item.UserID, &item.Username, &item.UserDisplay, &item.Name, &item.Extension, &item.SipURI, &item.Status, &skillsRaw, &item.CurrentCall, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.AgentSeat{}, err
	}
	_ = json.Unmarshal([]byte(skillsRaw), &item.Skills)
	return item, nil
}

func scanSipEndpoint(scanner rowScanner) (domain.SipEndpoint, error) {
	var item domain.SipEndpoint
	var configRaw string
	err := scanner.Scan(&item.ID, &item.Name, &item.WSSURL, &item.Domain, &item.Proxy, &configRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.SipEndpoint{}, err
	}
	_ = json.Unmarshal([]byte(configRaw), &item.Config)
	return item, nil
}

func scanCallSession(scanner rowScanner) (domain.CallSession, error) {
	var item domain.CallSession
	var ended sql.NullTime
	err := scanner.Scan(&item.ID, &item.SeatID, &item.PatientID, &item.Direction, &item.PhoneNumber, &item.Status, &item.StartedAt, &ended, &item.RecordingID, &item.TranscriptID, &item.AnalysisID, &item.InterviewForm)
	if err != nil {
		return domain.CallSession{}, err
	}
	if ended.Valid {
		item.EndedAt = ended.Time
	}
	return item, nil
}

func scanRecording(scanner rowScanner) (domain.Recording, error) {
	var item domain.Recording
	err := scanner.Scan(&item.ID, &item.CallID, &item.StorageURI, &item.Duration, &item.Filename, &item.MimeType, &item.SizeBytes, &item.Source, &item.Backend, &item.ObjectName, &item.Status, &item.CreatedAt)
	if err != nil {
		return domain.Recording{}, err
	}
	return item, nil
}

func scanModelProvider(scanner rowScanner) (domain.ModelProvider, error) {
	var item domain.ModelProvider
	var configRaw string
	err := scanner.Scan(&item.ID, &item.Name, &item.Kind, &item.Mode, &item.Endpoint, &item.Model, &item.CredentialRef, &configRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.ModelProvider{}, err
	}
	_ = json.Unmarshal([]byte(configRaw), &item.Config)
	return item, nil
}

func scanCallAnalysis(scanner rowScanner) (domain.CallAnalysis, error) {
	var item domain.CallAnalysis
	var extractedRaw string
	err := scanner.Scan(&item.ID, &item.CallID, &item.ProviderID, &item.PatientEmotion, &item.TrueSatisfaction, &item.RiskLevel, &item.PatientStatus, &item.Summary, &extractedRaw, &item.CreatedAt)
	if err != nil {
		return domain.CallAnalysis{}, err
	}
	_ = json.Unmarshal([]byte(extractedRaw), &item.ExtractedFormData)
	return item, nil
}

func scanRealtimeSession(scanner rowScanner) (domain.RealtimeAssistSession, error) {
	var item domain.RealtimeAssistSession
	var transcriptRaw, draftRaw string
	err := scanner.Scan(&item.ID, &item.CallID, &item.PatientID, &item.FormID, &item.ProviderID, &item.Status, &transcriptRaw, &draftRaw, &item.LastSuggestion, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.RealtimeAssistSession{}, err
	}
	_ = json.Unmarshal([]byte(transcriptRaw), &item.Transcript)
	_ = json.Unmarshal([]byte(draftRaw), &item.FormDraft)
	return item, nil
}

func scanOfflineJob(scanner rowScanner) (domain.OfflineAnalysisJob, error) {
	var item domain.OfflineAnalysisJob
	var resultRaw string
	err := scanner.Scan(&item.ID, &item.CallID, &item.RecordingID, &item.ProviderID, &item.Status, &resultRaw, &item.Error, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.OfflineAnalysisJob{}, err
	}
	_ = json.Unmarshal([]byte(resultRaw), &item.Result)
	return item, nil
}

func scanInterview(scanner rowScanner) (domain.InterviewSession, error) {
	var item domain.InterviewSession
	var messagesRaw, draftRaw string
	err := scanner.Scan(&item.ID, &item.PatientID, &item.FormID, &item.CallID, &item.Mode, &item.Status, &messagesRaw, &draftRaw, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return domain.InterviewSession{}, err
	}
	_ = json.Unmarshal([]byte(messagesRaw), &item.Messages)
	_ = json.Unmarshal([]byte(draftRaw), &item.FormDraft)
	return item, nil
}

func nullableTime(value time.Time) interface{} {
	if value.IsZero() {
		return nil
	}
	return value
}
