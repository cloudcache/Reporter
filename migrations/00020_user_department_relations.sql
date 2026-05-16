CREATE TABLE IF NOT EXISTS user_departments (
  user_id CHAR(36) NOT NULL,
  department_id VARCHAR(80) NOT NULL,
  relation_type ENUM('member','manage') NOT NULL DEFAULT 'member',
  is_primary BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (user_id, department_id, relation_type),
  INDEX idx_user_departments_user (user_id, relation_type),
  INDEX idx_user_departments_department (department_id, relation_type),
  CONSTRAINT fk_user_departments_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
  CONSTRAINT fk_user_departments_department FOREIGN KEY (department_id) REFERENCES departments(id)
);

INSERT INTO user_departments (user_id, department_id, relation_type, is_primary)
SELECT u.id, d.id, 'manage', FALSE
FROM users u
JOIN departments d
WHERE u.username = 'admin'
ON DUPLICATE KEY UPDATE is_primary = VALUES(is_primary);
