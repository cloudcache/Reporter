package install

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/google/uuid"

	"reporter/internal/auth"
	"reporter/internal/config"
	"reporter/internal/store"
)

const (
	DefaultLockPath = "install/install.lock"
	DefaultSQLPath  = "install/init.sql"
	DefaultCfgPath  = "config.yaml"
)

type DatabaseRequest struct {
	Driver   string `json:"driver"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Database string `json:"database"`
	Username string `json:"username"`
	Password string `json:"password"`
	Charset  string `json:"charset"`
	Loc      string `json:"loc"`
	DSN      string `json:"dsn"`
}

type AdminRequest struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	Password    string `json:"password"`
}

type Request struct {
	Database DatabaseRequest `json:"database"`
	Admin    AdminRequest    `json:"admin"`
}

type Status struct {
	Installed    bool   `json:"installed"`
	LockPath     string `json:"lockPath"`
	ConfigPath   string `json:"configPath"`
	DatabaseDSN  bool   `json:"databaseDsn"`
	LockCreateAt string `json:"lockCreatedAt,omitempty"`
}

type Result struct {
	Installed bool   `json:"installed"`
	Message   string `json:"message"`
	Next      string `json:"next"`
}

func CurrentStatus(cfg config.Config) Status {
	status := Status{
		Installed:   false,
		LockPath:    DefaultLockPath,
		ConfigPath:  DefaultCfgPath,
		DatabaseDSN: cfg.Database.DSN != "",
	}
	info, err := os.Stat(DefaultLockPath)
	if err == nil {
		status.Installed = true
		status.LockCreateAt = info.ModTime().Format(time.RFC3339)
	}
	return status
}

func TestDatabase(ctx context.Context, req DatabaseRequest) error {
	db, err := sql.Open(driver(req), BuildDSN(req))
	if err != nil {
		return err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	return db.PingContext(ctx)
}

func Run(ctx context.Context, cfg config.Config, req Request) (Result, error) {
	if CurrentStatus(cfg).Installed {
		return Result{}, errors.New("系统已安装，不能重复执行安装")
	}
	if err := validate(req); err != nil {
		return Result{}, err
	}
	dsn := BuildDSN(req.Database)
	db, err := sql.Open(driver(req.Database), dsn)
	if err != nil {
		return Result{}, err
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return Result{}, err
	}
	if err := executeInitSQL(ctx, db, DefaultSQLPath); err != nil {
		return Result{}, err
	}
	if err := createAdmin(ctx, db, req.Admin); err != nil {
		return Result{}, err
	}
	if err := seedFormLibrary(ctx, db); err != nil {
		return Result{}, err
	}
	cfg.Database.Driver = driver(req.Database)
	cfg.Database.DSN = dsn
	if err := writeConfig(DefaultCfgPath, cfg); err != nil {
		return Result{}, err
	}
	if err := writeLock(req); err != nil {
		return Result{}, err
	}
	return Result{Installed: true, Message: "安装完成", Next: "/login"}, nil
}

func BuildDSN(req DatabaseRequest) string {
	if strings.TrimSpace(req.DSN) != "" {
		return strings.TrimSpace(req.DSN)
	}
	port := req.Port
	if port == 0 {
		port = 3306
	}
	charset := firstNonEmpty(req.Charset, "utf8mb4")
	loc := firstNonEmpty(req.Loc, "Local")
	params := url.Values{}
	params.Set("parseTime", "true")
	params.Set("charset", charset)
	params.Set("loc", loc)
	address := net.JoinHostPort(strings.TrimSpace(req.Host), strconv.Itoa(port))
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?%s", req.Username, req.Password, address, req.Database, params.Encode())
}

func driver(req DatabaseRequest) string {
	if req.Driver == "" {
		return "mysql"
	}
	return req.Driver
}

func validate(req Request) error {
	if driver(req.Database) != "mysql" {
		return errors.New("当前安装器仅支持 MySQL")
	}
	if strings.TrimSpace(req.Database.DSN) == "" {
		if strings.TrimSpace(req.Database.Host) == "" || strings.TrimSpace(req.Database.Database) == "" || strings.TrimSpace(req.Database.Username) == "" {
			return errors.New("数据库主机、库名和用户名不能为空")
		}
	}
	if strings.TrimSpace(req.Admin.Username) == "" || strings.TrimSpace(req.Admin.DisplayName) == "" {
		return errors.New("管理员账号和姓名不能为空")
	}
	if len(req.Admin.Password) < 8 {
		return errors.New("管理员密码至少 8 位")
	}
	return nil
}

func executeInitSQL(ctx context.Context, db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, statement := range splitSQL(string(content)) {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("执行 SQL 失败: %w\n%s", err, statement)
		}
	}
	return nil
}

func splitSQL(content string) []string {
	lines := []string{}
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || trimmed == "" {
			continue
		}
		lines = append(lines, line)
	}
	parts := strings.Split(strings.Join(lines, "\n"), ";")
	statements := make([]string, 0, len(parts))
	for _, part := range parts {
		statement := strings.TrimSpace(part)
		if statement != "" {
			statements = append(statements, statement)
		}
	}
	return statements
}

func createAdmin(ctx context.Context, db *sql.DB, admin AdminRequest) error {
	passwordHash, err := auth.HashPassword(admin.Password)
	if err != nil {
		return err
	}
	adminID := uuid.NewString()
	permissionID := uuid.NewString()
	_, err = db.ExecContext(ctx, `
INSERT INTO permissions (id, resource, action, description)
VALUES (?, '*', '*', '全部权限')
ON DUPLICATE KEY UPDATE description = VALUES(description)
`, permissionID)
	if err != nil {
		return err
	}
	var existingPermissionID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM permissions WHERE resource='*' AND action='*' LIMIT 1`).Scan(&existingPermissionID); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO roles (id, name, description) VALUES
('admin', '系统管理员', '拥有平台全部管理权限'),
('doctor', '医生', '查看患者档案、制定随访方案、处理异常结果'),
('nurse', '护士', '维护护理随访、宣教和患者基础信息'),
('analyst', '数据分析员', '可管理表单、报表并查看数据源'),
('agent', '随访员/调查员', '可查看患者并执行电话随访、问卷调查')
ON DUPLICATE KEY UPDATE name = VALUES(name), description = VALUES(description)
`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO role_permissions (role_id, permission_id)
VALUES ('admin', ?)
ON DUPLICATE KEY UPDATE permission_id = VALUES(permission_id)
`, existingPermissionID); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO permissions (id, resource, action, description) VALUES
(UUID(), '/api/v1/patients', 'read', '查看患者档案'),
(UUID(), '/api/v1/forms', 'read', '查看表单问卷'),
(UUID(), '/api/v1/forms', 'update', '维护表单问卷'),
(UUID(), '/api/v1/forms', '*', '管理表单问卷'),
(UUID(), '/api/v1/followup', 'read', '查看随访任务'),
(UUID(), '/api/v1/followup', 'update', '执行随访任务'),
(UUID(), '/api/v1/followup', '*', '管理随访任务'),
(UUID(), '/api/v1/reports', 'read', '查看分析报表'),
(UUID(), '/api/v1/reports', '*', '管理分析报表'),
(UUID(), '/api/v1/data-sources', 'read', '查看数据源'),
(UUID(), '/api/v1/complaints', 'read', '查看评价投诉'),
(UUID(), '/api/v1/complaints', 'create', '新建评价投诉'),
(UUID(), '/api/v1/complaints', 'update', '处理评价投诉'),
(UUID(), '/api/v1/call-center', 'read', '查看呼叫中心'),
(UUID(), '/api/v1/call-center', 'create', '新建呼叫任务'),
(UUID(), '/api/v1/call-center', 'update', '维护呼叫中心')
ON DUPLICATE KEY UPDATE description = VALUES(description)
`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO role_permissions (role_id, permission_id)
SELECT role_id, id FROM (
  SELECT 'doctor' role_id, '/api/v1/patients' resource, 'read' action UNION ALL
  SELECT 'doctor', '/api/v1/forms', 'read' UNION ALL
  SELECT 'doctor', '/api/v1/followup', '*' UNION ALL
  SELECT 'doctor', '/api/v1/reports', 'read' UNION ALL
  SELECT 'doctor', '/api/v1/complaints', 'read' UNION ALL
  SELECT 'doctor', '/api/v1/complaints', 'update' UNION ALL
  SELECT 'nurse', '/api/v1/patients', 'read' UNION ALL
  SELECT 'nurse', '/api/v1/followup', 'read' UNION ALL
  SELECT 'nurse', '/api/v1/followup', 'update' UNION ALL
  SELECT 'nurse', '/api/v1/forms', 'read' UNION ALL
  SELECT 'nurse', '/api/v1/complaints', 'read' UNION ALL
  SELECT 'nurse', '/api/v1/complaints', 'create' UNION ALL
  SELECT 'analyst', '/api/v1/forms', '*' UNION ALL
  SELECT 'analyst', '/api/v1/reports', '*' UNION ALL
  SELECT 'analyst', '/api/v1/data-sources', 'read' UNION ALL
  SELECT 'analyst', '/api/v1/complaints', 'read' UNION ALL
  SELECT 'agent', '/api/v1/patients', 'read' UNION ALL
  SELECT 'agent', '/api/v1/followup', 'read' UNION ALL
  SELECT 'agent', '/api/v1/followup', 'update' UNION ALL
  SELECT 'agent', '/api/v1/call-center', 'read' UNION ALL
  SELECT 'agent', '/api/v1/call-center', 'create' UNION ALL
  SELECT 'agent', '/api/v1/call-center', 'update' UNION ALL
  SELECT 'agent', '/api/v1/complaints', 'read' UNION ALL
  SELECT 'agent', '/api/v1/complaints', 'create'
) wanted
JOIN permissions p ON p.resource = wanted.resource AND p.action = wanted.action
ON DUPLICATE KEY UPDATE permission_id = VALUES(permission_id)
`); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `
INSERT INTO users (id, username, display_name, password_hash)
VALUES (?, ?, ?, ?)
`, adminID, admin.Username, admin.DisplayName, passwordHash); err != nil {
		return err
	}
	_, err = db.ExecContext(ctx, `
INSERT INTO user_roles (user_id, role_id)
VALUES (?, 'admin')
`, adminID)
	return err
}

func seedFormLibrary(ctx context.Context, db *sql.DB) error {
	for _, item := range store.DefaultFormLibrary() {
		components, err := json.Marshal(item.Components)
		if err != nil {
			return err
		}
		if _, err := db.ExecContext(ctx, `
INSERT INTO form_library_items (id, kind, label, hint, scenario, components_json, sort_order, enabled)
VALUES (?, ?, ?, ?, ?, ?, ?, TRUE)
ON DUPLICATE KEY UPDATE
  kind = VALUES(kind),
  label = VALUES(label),
  hint = VALUES(hint),
  scenario = VALUES(scenario),
  components_json = VALUES(components_json),
  sort_order = VALUES(sort_order),
  enabled = VALUES(enabled)
`, item.ID, item.Kind, item.Label, item.Hint, item.Scenario, string(components), item.SortOrder); err != nil {
			return err
		}
	}
	return nil
}

func writeConfig(path string, cfg config.Config) error {
	content := fmt.Sprintf(`environment: %s
businessConfigDB: %t

log:
  level: %s

http:
  addr: %q
  readHeaderTimeout: %s
  shutdownTimeout: %s

auth:
  jwtSecret: %s
  accessTokenTTL: %s
  refreshTokenTTL: %s

database:
  driver: %s
  dsn: %q
  maxOpenConns: %d
  maxIdleConns: %d
  connMaxLifetime: %s

redis:
  enabled: %t
  addr: %q
  username: %q
  password: %q
  db: %d
  ttl: %s
`,
		cfg.Environment,
		cfg.BusinessConfigDB,
		cfg.Log.Level,
		cfg.HTTP.Addr,
		cfg.HTTP.ReadHeaderTimeout,
		cfg.HTTP.ShutdownTimeout,
		cfg.Auth.JWTSecret,
		cfg.Auth.AccessTokenTTL,
		cfg.Auth.RefreshTokenTTL,
		cfg.Database.Driver,
		cfg.Database.DSN,
		cfg.Database.MaxOpenConns,
		cfg.Database.MaxIdleConns,
		cfg.Database.ConnMaxLifetime,
		cfg.Redis.Enabled,
		cfg.Redis.Addr,
		cfg.Redis.Username,
		cfg.Redis.Password,
		cfg.Redis.DB,
		cfg.Redis.TTL,
	)
	return os.WriteFile(path, []byte(content), 0o600)
}

func writeLock(req Request) error {
	if err := os.MkdirAll(filepath.Dir(DefaultLockPath), 0o755); err != nil {
		return err
	}
	content, err := json.MarshalIndent(map[string]interface{}{
		"installedAt":   time.Now().Format(time.RFC3339),
		"database":      req.Database.Database,
		"adminUsername": req.Admin.Username,
		"version":       "1.0.0",
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(DefaultLockPath, content, 0o600)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
