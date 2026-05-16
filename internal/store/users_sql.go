package store

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"

	"reporter/internal/domain"
)

func (s *Store) EnsureIdentityOrgTables(ctx context.Context) error {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.ExecContext(ctx, `
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
)`)
	return err
}

func (s *Store) dbUsers(ctx context.Context) ([]domain.User, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	users, err := queryUsers(ctx, db, "")
	if err != nil {
		return nil, err
	}
	return mapUsersToSlice(users), nil
}

func (s *Store) dbUserByID(ctx context.Context, id string) (domain.User, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.User{}, false, err
	}
	defer db.Close()
	users, err := queryUsers(ctx, db, "WHERE u.id = ?", id)
	if err != nil {
		return domain.User{}, false, err
	}
	user, ok := users[id]
	return user, ok, nil
}

func (s *Store) dbUserByUsername(ctx context.Context, username string) (domain.User, bool, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.User{}, false, err
	}
	defer db.Close()
	users, err := queryUsersByUsername(ctx, db, username)
	if err != nil {
		return domain.User{}, false, err
	}
	for _, user := range users {
		return user, true, nil
	}
	return domain.User{}, false, nil
}

func (s *Store) dbCreateUser(ctx context.Context, user domain.User) (domain.User, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	if user.ID == "" {
		user.ID = uuid.NewString()
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO users (id, username, display_name, password_hash, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
		user.ID, user.Username, user.DisplayName, user.PasswordHash, now, now); err != nil {
		return domain.User{}, err
	}
	if err := replaceUserRoles(ctx, tx, user.ID, user.Roles); err != nil {
		return domain.User{}, err
	}
	if err := replaceUserDepartments(ctx, tx, user.ID, user.DepartmentID, user.DepartmentIDs, user.ManagedDepartmentIDs); err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, err
	}
	created, ok, err := s.dbUserByID(ctx, user.ID)
	if err != nil || !ok {
		return user, err
	}
	return created, nil
}

func (s *Store) dbUpdateUser(ctx context.Context, id string, patch domain.User) (domain.User, error) {
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()
	if patch.PasswordHash != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET username = ?, display_name = ?, password_hash = ?, updated_at = ? WHERE id = ?`, patch.Username, patch.DisplayName, patch.PasswordHash, time.Now().UTC(), id); err != nil {
			return domain.User{}, err
		}
	} else {
		if _, err := tx.ExecContext(ctx, `UPDATE users SET username = ?, display_name = ?, updated_at = ? WHERE id = ?`, patch.Username, patch.DisplayName, time.Now().UTC(), id); err != nil {
			return domain.User{}, err
		}
	}
	if err := replaceUserRoles(ctx, tx, id, patch.Roles); err != nil {
		return domain.User{}, err
	}
	if err := replaceUserDepartments(ctx, tx, id, patch.DepartmentID, patch.DepartmentIDs, patch.ManagedDepartmentIDs); err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, err
	}
	user, ok, err := s.dbUserByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if !ok {
		return domain.User{}, ErrNotFound
	}
	return user, nil
}

func (s *Store) dbDeleteUser(ctx context.Context, id string) (domain.User, error) {
	before, ok, err := s.dbUserByID(ctx, id)
	if err != nil {
		return domain.User{}, err
	}
	if !ok {
		return domain.User{}, ErrNotFound
	}
	db, err := s.surveyDB(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer db.Close()
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return domain.User{}, err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `UPDATE agent_seats SET user_id = NULL WHERE user_id = ?`, id); err != nil {
		return domain.User{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id); err != nil {
		return domain.User{}, err
	}
	if err := tx.Commit(); err != nil {
		return domain.User{}, err
	}
	return before, nil
}

func queryUsers(ctx context.Context, db *sql.DB, where string, args ...interface{}) (map[string]domain.User, error) {
	rows, err := db.QueryContext(ctx, `
SELECT u.id, u.username, u.display_name, u.password_hash, u.created_at, u.updated_at
FROM users u `+where+`
ORDER BY u.created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := map[string]domain.User{}
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.Username, &user.DisplayName, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, err
		}
		user.Roles = []string{}
		user.DepartmentIDs = []string{}
		user.ManagedDepartmentIDs = []string{}
		users[user.ID] = user
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return users, nil
	}
	if err := attachUserRoles(ctx, db, users); err != nil {
		return nil, err
	}
	if err := attachUserDepartments(ctx, db, users); err != nil {
		return nil, err
	}
	return users, nil
}

func queryUsersByUsername(ctx context.Context, db *sql.DB, username string) (map[string]domain.User, error) {
	return queryUsers(ctx, db, "WHERE u.username = ?", username)
}

func attachUserRoles(ctx context.Context, db *sql.DB, users map[string]domain.User) error {
	rows, err := db.QueryContext(ctx, `SELECT user_id, role_id FROM user_roles ORDER BY role_id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, roleID string
		if err := rows.Scan(&userID, &roleID); err != nil {
			return err
		}
		user, ok := users[userID]
		if !ok {
			continue
		}
		user.Roles = append(user.Roles, roleID)
		users[userID] = user
	}
	return rows.Err()
}

func attachUserDepartments(ctx context.Context, db *sql.DB, users map[string]domain.User) error {
	rows, err := db.QueryContext(ctx, `
SELECT ud.user_id, ud.department_id, ud.relation_type, ud.is_primary, d.code, d.name
FROM user_departments ud
JOIN departments d ON d.id = ud.department_id
ORDER BY ud.is_primary DESC, d.code`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var userID, departmentID, relationType, code, name string
		var primary bool
		if err := rows.Scan(&userID, &departmentID, &relationType, &primary, &code, &name); err != nil {
			return err
		}
		user, ok := users[userID]
		if !ok {
			continue
		}
		switch relationType {
		case "manage":
			user.ManagedDepartmentIDs = appendUniqueString(user.ManagedDepartmentIDs, departmentID)
		default:
			user.DepartmentIDs = appendUniqueString(user.DepartmentIDs, departmentID)
			if primary || user.DepartmentID == "" {
				user.DepartmentID = departmentID
				user.DepartmentCode = code
				user.DepartmentName = name
			}
		}
		users[userID] = user
	}
	return rows.Err()
}

func replaceUserRoles(ctx context.Context, tx *sql.Tx, userID string, roles []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_roles WHERE user_id = ?`, userID); err != nil {
		return err
	}
	for _, roleID := range roles {
		if strings.TrimSpace(roleID) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)`, userID, roleID); err != nil {
			return err
		}
	}
	return nil
}

func replaceUserDepartments(ctx context.Context, tx *sql.Tx, userID string, primaryDepartmentID string, departmentIDs []string, managedDepartmentIDs []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM user_departments WHERE user_id = ?`, userID); err != nil {
		return err
	}
	if primaryDepartmentID != "" {
		departmentIDs = append([]string{primaryDepartmentID}, departmentIDs...)
	}
	for _, departmentID := range uniqueStrings(departmentIDs) {
		if strings.TrimSpace(departmentID) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_departments (user_id, department_id, relation_type, is_primary) VALUES (?, ?, 'member', ?)`,
			userID, departmentID, departmentID == primaryDepartmentID); err != nil {
			return err
		}
	}
	for _, departmentID := range uniqueStrings(managedDepartmentIDs) {
		if strings.TrimSpace(departmentID) == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO user_departments (user_id, department_id, relation_type, is_primary) VALUES (?, ?, 'manage', FALSE)`,
			userID, departmentID); err != nil {
			return err
		}
	}
	return nil
}

func mapUsersToSlice(users map[string]domain.User) []domain.User {
	items := make([]domain.User, 0, len(users))
	for _, user := range users {
		items = append(items, user)
	}
	return items
}

func appendUniqueString(items []string, item string) []string {
	item = strings.TrimSpace(item)
	if item == "" {
		return items
	}
	for _, existing := range items {
		if existing == item {
			return items
		}
	}
	return append(items, item)
}
