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

func (s *Store) RolesStrict(ctx context.Context) ([]domain.Role, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	roles, err := loadRoles(ctx, db)
	if err != nil {
		return nil, err
	}
	return mapRolesToSlice(roles), nil
}

func (s *Store) CreateRoleStrict(ctx context.Context, role domain.Role) (domain.Role, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Role{}, err
	}
	defer db.Close()
	if strings.TrimSpace(role.ID) == "" {
		role.ID = uuid.NewString()
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Role{}, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO roles (id, name, description) VALUES (?, ?, ?)`, role.ID, role.Name, nullableString(role.Description)); err == nil {
		err = replaceRolePermissions(ctx, tx, role.ID, role.Permissions)
	}
	if err != nil {
		_ = tx.Rollback()
		return domain.Role{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Role{}, err
	}
	saved, ok, err := s.RoleStrict(ctx, role.ID)
	if err != nil {
		return domain.Role{}, err
	}
	if !ok {
		return domain.Role{}, ErrNotFound
	}
	return saved, nil
}

func (s *Store) RoleStrict(ctx context.Context, id string) (domain.Role, bool, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Role{}, false, err
	}
	defer db.Close()
	roles, err := loadRoles(ctx, db)
	if err != nil {
		return domain.Role{}, false, err
	}
	role, ok := roles[id]
	return role, ok, nil
}

func (s *Store) UpdateRolePermissionsStrict(ctx context.Context, id string, permissions []string) (domain.Role, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.Role{}, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Role{}, err
	}
	var roleID string
	if err = tx.QueryRowContext(ctx, `SELECT id FROM roles WHERE id = ?`, id).Scan(&roleID); err != nil {
		_ = tx.Rollback()
		if err == sql.ErrNoRows {
			return domain.Role{}, ErrNotFound
		}
		return domain.Role{}, err
	}
	if err = replaceRolePermissions(ctx, tx, id, permissions); err != nil {
		_ = tx.Rollback()
		return domain.Role{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.Role{}, err
	}
	role, ok, err := s.RoleStrict(ctx, id)
	if err != nil {
		return domain.Role{}, err
	}
	if !ok {
		return domain.Role{}, ErrNotFound
	}
	return role, nil
}

func replaceRolePermissions(ctx context.Context, tx *sql.Tx, roleID string, permissions []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM role_permissions WHERE role_id = ?`, roleID); err != nil {
		return err
	}
	for _, permission := range permissions {
		resource, action := splitPermission(permission)
		if resource == "" || action == "" {
			continue
		}
		permissionID := uuid.NewString()
		_, err := tx.ExecContext(ctx, `
INSERT INTO permissions (id, resource, action) VALUES (?, ?, ?)
ON DUPLICATE KEY UPDATE id = id`, permissionID, resource, action)
		if err != nil {
			return err
		}
		var savedID string
		if err := tx.QueryRowContext(ctx, `SELECT id FROM permissions WHERE resource = ? AND action = ?`, resource, action).Scan(&savedID); err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `INSERT IGNORE INTO role_permissions (role_id, permission_id) VALUES (?, ?)`, roleID, savedID); err != nil {
			return err
		}
	}
	return nil
}

func splitPermission(permission string) (string, string) {
	permission = strings.TrimSpace(permission)
	if permission == "" {
		return "", ""
	}
	parts := strings.Split(permission, ":")
	if len(parts) < 2 {
		return permission, "*"
	}
	return strings.Join(parts[:len(parts)-1], ":"), parts[len(parts)-1]
}

func mapRolesToSlice(roles map[string]domain.Role) []domain.Role {
	items := make([]domain.Role, 0, len(roles))
	for _, role := range roles {
		items = append(items, role)
	}
	return items
}

func nullableJSON(raw []byte) interface{} {
	if string(raw) == "null" {
		return nil
	}
	return string(raw)
}

func (s *Store) SaveAuditStrict(ctx context.Context, log domain.AuditLog) (domain.AuditLog, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return domain.AuditLog{}, err
	}
	defer db.Close()
	if strings.TrimSpace(log.ID) == "" {
		log.ID = uuid.NewString()
	}
	beforeJSON, err := json.Marshal(log.Before)
	if err != nil {
		return domain.AuditLog{}, err
	}
	afterJSON, err := json.Marshal(log.After)
	if err != nil {
		return domain.AuditLog{}, err
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO audit_logs (id, actor_id, action, resource, before_json, after_json, ip, user_agent, trace_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, nullableString(log.ActorID), log.Action, log.Resource, nullableJSON(beforeJSON), nullableJSON(afterJSON), nullableString(log.IP), nullableString(log.UserAgent), nullableString(log.TraceID)); err != nil {
		return domain.AuditLog{}, err
	}
	log.CreatedAt = time.Now().UTC()
	return log, nil
}

func (s *Store) AuditLogsStrict(ctx context.Context) ([]domain.AuditLog, error) {
	db, err := s.openConfiguredDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()
	rows, err := db.QueryContext(ctx, `SELECT id, COALESCE(actor_id, ''), action, resource, COALESCE(CAST(before_json AS CHAR), 'null'), COALESCE(CAST(after_json AS CHAR), 'null'), COALESCE(ip, ''), COALESCE(user_agent, ''), COALESCE(trace_id, ''), created_at FROM audit_logs ORDER BY created_at DESC LIMIT 500`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.AuditLog{}
	for rows.Next() {
		var item domain.AuditLog
		var beforeRaw, afterRaw string
		if err := rows.Scan(&item.ID, &item.ActorID, &item.Action, &item.Resource, &beforeRaw, &afterRaw, &item.IP, &item.UserAgent, &item.TraceID, &item.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal([]byte(beforeRaw), &item.Before)
		_ = json.Unmarshal([]byte(afterRaw), &item.After)
		items = append(items, item)
	}
	return items, rows.Err()
}
