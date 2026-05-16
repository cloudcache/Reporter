INSERT INTO sip_endpoints (id, name, wss_url, domain, proxy, config_json) VALUES
('SIP001', '院内 WebRTC SIP 网关', 'wss://pbx.example.local/ws', 'call.example.local', 'sip:pbx.example.local;transport=wss', '{"enabled":false,"webrtc":true,"transport":"udp","bindHost":"0.0.0.0","trunkUri":"sip:{phone}@carrier.example.local"}')
ON DUPLICATE KEY UPDATE name = VALUES(name), wss_url = VALUES(wss_url), domain = VALUES(domain), proxy = VALUES(proxy), config_json = VALUES(config_json);

INSERT INTO agent_seats (id, user_id, name, extension, sip_uri, status, skills_json) VALUES
('SEAT001', NULL, '默认随访坐席', '8001', 'sip:8001@call.example.local', 'available', '["followup","survey"]')
ON DUPLICATE KEY UPDATE name = VALUES(name), extension = VALUES(extension), sip_uri = VALUES(sip_uri), status = VALUES(status), skills_json = VALUES(skills_json);

INSERT INTO model_providers (id, name, kind, mode, endpoint, model, credential_ref, config_json) VALUES
('LLM001', '院内大模型网关', 'openai-compatible', 'offline', 'https://llm.example.local/v1', 'medical-call-analyzer', 'secret://llm/primary', '{"supports_audio":true,"supports_json_schema":true,"audio_analysis":true}'),
('LLM002', '实时语音识别与表单回填', 'realtime-asr', 'realtime', 'wss://llm.example.local/realtime', 'medical-realtime-asr', 'secret://llm/realtime', '{"partial_transcript":true,"form_autofill":true}')
ON DUPLICATE KEY UPDATE name = VALUES(name), kind = VALUES(kind), mode = VALUES(mode), endpoint = VALUES(endpoint), model = VALUES(model), credential_ref = VALUES(credential_ref), config_json = VALUES(config_json);
