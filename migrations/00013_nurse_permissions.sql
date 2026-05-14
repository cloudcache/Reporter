-- +goose Up
INSERT INTO permissions (id, resource, action, description) VALUES
(UUID(), '/api/v1/patients', 'read', '查看患者档案'),
(UUID(), '/api/v1/followup', 'read', '查看随访任务'),
(UUID(), '/api/v1/followup', 'update', '执行随访任务'),
(UUID(), '/api/v1/forms', 'read', '查看表单问卷'),
(UUID(), '/api/v1/complaints', 'read', '查看评价投诉'),
(UUID(), '/api/v1/complaints', 'create', '新建评价投诉')
ON DUPLICATE KEY UPDATE description = VALUES(description);

DELETE rp
FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE rp.role_id = 'nurse';

INSERT INTO role_permissions (role_id, permission_id)
SELECT 'nurse', id
FROM permissions
WHERE (resource = '/api/v1/patients' AND action = 'read')
   OR (resource = '/api/v1/followup' AND action IN ('read', 'update'))
   OR (resource = '/api/v1/forms' AND action = 'read')
   OR (resource = '/api/v1/complaints' AND action IN ('read', 'create'));

-- +goose Down
DELETE rp
FROM role_permissions rp
JOIN permissions p ON p.id = rp.permission_id
WHERE rp.role_id = 'nurse'
  AND ((resource = '/api/v1/patients' AND action = 'read')
    OR (resource = '/api/v1/followup' AND action IN ('read', 'update'))
    OR (resource = '/api/v1/forms' AND action = 'read')
    OR (resource = '/api/v1/complaints' AND action IN ('read', 'create')));
