-- +goose Up
ALTER TABLE reports
  ADD COLUMN report_type VARCHAR(60) NOT NULL DEFAULT 'custom' AFTER id;

INSERT INTO reports (id, report_type, name, description)
VALUES
  ('RP001', 'followup', '随访完成情况月报', '从随访记录聚合随访提交量、完成量和完成率'),
  ('RP002', 'satisfaction', '满意度分析报告', '从满意度答卷、访谈表单和指标体系聚合科室、指标、渠道和低分原因'),
  ('RP003', 'complaint', '评价投诉分析报告', '从评价投诉台账聚合投诉、表扬、处理状态和责任科室')
ON DUPLICATE KEY UPDATE report_type = VALUES(report_type), name = VALUES(name), description = VALUES(description);

INSERT INTO report_widgets (id, report_id, widget_type, title, query_json, vis_spec_json, data_source_id)
VALUES
  ('RW001', 'RP001', 'bar', '月度随访完成率', '{"source":"followup_records"}', '{}', NULL),
  ('RW002', 'RP001', 'table', '随访月度明细', '{"source":"followup_records"}', '{}', NULL),
  ('RW003', 'RP002', 'bar', '科室满意度', '{"source":"survey_submissions"}', '{}', NULL),
  ('RW004', 'RP002', 'table', '满意度指标明细', '{"source":"satisfaction_indicator_scores"}', '{}', NULL),
  ('RW005', 'RP003', 'bar', '责任科室投诉评价', '{"source":"evaluation_complaints"}', '{}', NULL)
ON DUPLICATE KEY UPDATE title = VALUES(title), query_json = VALUES(query_json);

-- +goose Down
DELETE FROM report_widgets WHERE id IN ('RW001', 'RW002', 'RW003', 'RW004', 'RW005');
DELETE FROM reports WHERE id IN ('RP001', 'RP002', 'RP003');
ALTER TABLE reports DROP COLUMN report_type;
