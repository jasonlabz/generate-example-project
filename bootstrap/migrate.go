package bootstrap

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jasonlabz/generate-example-project/common/resource"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/jasonlabz/potato/gormx"
)

// ── 常量 ──

const migrationTableSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version VARCHAR(255) NOT NULL,
	type VARCHAR(16) NOT NULL DEFAULT 'ddl' CHECK (type IN ('ddl', 'seed')),
	applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
	PRIMARY KEY (version, type)
)`

const (
	baselinePrefix = "00000000_000" // 基线文件名前缀，有且仅有一个
	versionPrefix  = "-- @version " // 推荐版本声明，也支持 --@version
	versionPrefix2 = "--@version "
)

type migrationType string

const (
	migrationTypeDDL  migrationType = "ddl"
	migrationTypeSeed migrationType = "seed"
)

// ── 数据结构 ──

// migFile 迁移文件元信息。
// 版本优先取头部 -- @version，普通文件缺失时从文件名前缀提取。
// baseline 由文件名是否以 00000000_000 开头决定。
type migFile struct {
	name     string
	path     string
	version  string
	kind     migrationType
	baseline bool
}

// ── 公开入口 ──

// ensureDB 检查目标数据库是否存在，不存在则创建。
// 通过 gormx.InitConfig 临时连接管理库，用完 Close，不污染全局连接池。
func ensureDB(ctx context.Context) {
	cfg := GetConfig().DataSource
	if !cfg.Enable || cfg.DBType == string(gormx.DatabaseTypeSQLite) {
		return
	}

	creator, ok := lookupDBCreator(cfg.DBType)
	if !ok {
		resource.Logger.Warnf(ctx,
			"[ensureDB] 数据库类型 %s 不支持自动创建，请手动创建 %s 库后重启",
			cfg.DBType, cfg.Database)
		return
	}

	adminCfg := toGormxConfig(cfg, creator)
	adminDB, err := gormx.InitConfig(adminCfg)
	if err != nil {
		resource.Logger.Errorf(ctx, "[ensureDB] 连接服务器失败: %v", err)
		return
	}
	defer func() {
		if err := gormx.Close(adminCfg.DBName); err != nil {
			resource.Logger.Errorf(ctx, "[ensureDB] 关闭管理员数据库连接失败: %v", err)
		}
	}()

	if dbExists(adminDB, creator, cfg.Database) {
		return
	}

	if err := adminDB.Exec(creator.CreateDatabaseSQL(cfg.Database)).Error; err != nil {
		resource.Logger.Errorf(ctx, "[ensureDB] 创建数据库失败: %v", err)
		return
	}
	resource.Logger.Infof(ctx, "[ensureDB] 数据库 %s 已创建", cfg.Database)
}

// runMigrations 执行 DDL 迁移和种子数据。
//
// DDL 和 seed 版本号优先取头部 -- @version，普通文件缺失时从文件名前缀取
// YYYYMMDD_NNN；baseline 必须显式声明覆盖版本。
// 基线由文件名是否以 00000000_000 开头判定，有且仅有一个。
//
// 策略：
//   - 新库 → 执行基线 → 跳过版本 ≤ 基线版本的增量 → 执行剩余增量
//   - 已有库 → 只执行版本 > 最新已应用版本的增量
//   - seed → 在全部 DDL 完成后按同样的版本规则执行
func runMigrations(ctx context.Context) {
	cfg := GetConfig().DataSource
	if !cfg.Enable {
		return
	}
	db := withErrorLogger(gormx.DefaultMaster())

	unlock, err := acquireMigrationLock(ctx, db, cfg.DBType)
	if err != nil {
		panic(fmt.Errorf("[migrate] 获取迁移锁失败: %v", err))
	}
	defer unlock()

	if err = db.Exec(migrationTableSQL).Error; err != nil {
		panic(fmt.Errorf("[migrate] 创建追踪表失败: %v", err))
	}

	ddlFiles := loadMigrationFiles(ctx, "conf/migrations", migrationTypeDDL)
	seedFiles := loadMigrationFiles(ctx, "conf/seed", migrationTypeSeed)
	if len(ddlFiles) == 0 && len(seedFiles) == 0 {
		return
	}

	if err = runMigrationFiles(ctx, db, ddlFiles); err != nil {
		panic(err)
	}

	if err = runMigrationFiles(ctx, db, seedFiles); err != nil {
		resource.Logger.Warnf(ctx, "[seed] 迁移失败(已跳过): %v", err)
	}
}

// ── 文件加载与解析 ──

// loadMigrationFiles 扫描目录、按目录类型解析版本号并排序。
// seed 文件必须使用 YYYYMMDD_NNN_desc.sql 命名；无法解析版本号的文件会被跳过并告警。
func loadMigrationFiles(ctx context.Context, dir string, kind migrationType) []migFile {
	names := listSQLFiles(dir)
	files := make([]migFile, 0, len(names))

	for _, name := range names {
		path := filepath.Join(dir, name)
		ver := resolveVersion(path, name)
		if ver == "" {
			resource.Logger.Warnf(ctx, "[migrate] 跳过 %s: 无法解析版本号", name)
			continue
		}
		files = append(files, migFile{
			name:     name,
			path:     path,
			version:  ver,
			kind:     kind,
			baseline: strings.HasPrefix(name, baselinePrefix),
		})
	}

	sort.Slice(files, func(i, j int) bool {
		if files[i].version != files[j].version {
			return files[i].version < files[j].version
		}
		return files[i].name < files[j].name
	})
	return files
}

// resolveVersion 解析迁移版本号。DDL 和 seed 使用同一套规则。
func resolveVersion(path, name string) string {
	filenameVersion := extractNameVersion(name)
	if filenameVersion == "" {
		return ""
	}
	if version := parseHeaderVersion(path); isMigrationVersion(version) {
		return version
	}
	if strings.HasPrefix(name, baselinePrefix) {
		return ""
	}
	return filenameVersion
}

// parseHeaderVersion 读取 SQL 文件前若干行，查找 -- @version xxx 或 --@version xxx。
func parseHeaderVersion(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	// 仅读取头部声明，关闭失败不会影响已提取的值。
	defer func() {
		_ = f.Close()
	}()

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if value, ok := cutVersion(line, versionPrefix); ok {
			return strings.TrimSpace(value)
		}
		if value, ok := cutVersion(line, versionPrefix2); ok {
			return strings.TrimSpace(value)
		}
		// 遇到非注释非空行说明头部结束
		if line != "" && !strings.HasPrefix(line, "--") {
			break
		}
	}
	return ""
}

func cutVersion(line, prefix string) (string, bool) {
	if strings.HasPrefix(line, prefix) {
		return line[len(prefix):], true
	}
	return "", false
}

// extractNameVersion 从标准文件名提取版本号 YYYYMMDD_NNN。
// 例如 "20240701_001_add_email.sql" → "20240701_001"。
func extractNameVersion(name string) string {
	if !strings.HasSuffix(name, ".sql") {
		return ""
	}
	base := strings.TrimSuffix(name, ".sql")
	parts := strings.SplitN(base, "_", 3)
	if len(parts) == 3 &&
		len(parts[0]) == 8 &&
		len(parts[1]) == 3 &&
		parts[2] != "" &&
		isDigits(parts[0]) &&
		isDigits(parts[1]) {
		return parts[0] + "_" + parts[1]
	}
	return ""
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func isMigrationVersion(value string) bool {
	parts := strings.Split(value, "_")
	return len(parts) == 2 &&
		len(parts[0]) == 8 &&
		len(parts[1]) == 3 &&
		isDigits(parts[0]) &&
		isDigits(parts[1])
}

// listSQLFiles 返回目录下所有 .sql 文件名（不含路径），按名称排序。
func listSQLFiles(dir string) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	return names
}

// ── 迁移执行 ──

func runMigrationFiles(ctx context.Context, db *gorm.DB, files []migFile) error {
	if len(files) == 0 {
		return nil
	}

	kind := files[0].kind
	var baseline *migFile
	for i := range files {
		if !files[i].baseline {
			continue
		}
		if baseline != nil {
			return fmt.Errorf("[%s] 存在多个 baseline 文件", kind)
		}
		baseline = &files[i]
	}

	latest, err := latestVersion(db, kind)
	if err != nil {
		return fmt.Errorf("[%s] 查询最新版本失败: %w", kind, err)
	}

	if latest == "" {
		if baseline == nil {
			return fmt.Errorf("[migrate] 缺少基线文件（文件名需以 00000000_000 开头）")
		}
		resource.Logger.Infof(ctx, "[migrate] 执行基线 %s (版本 %s)", baseline.name, baseline.version)
		if err := execFile(db, baseline); err != nil {
			return fmt.Errorf("[migrate] 基线失败: %w", err)
		}
		latest = baseline.version
	}

	for i := range files {
		mf := &files[i]
		if mf.kind != kind || mf.baseline || mf.version <= latest {
			continue
		}
		done, err := isApplied(db, mf.version, mf.kind)
		if err != nil {
			return fmt.Errorf("[migrate] 查询状态失败 %s: %w", mf.name, err)
		}
		if done {
			continue
		}
		resource.Logger.Infof(ctx, "[migrate] 执行 %s (版本 %s)", mf.name, mf.version)
		if err := execFile(db, mf); err != nil {
			return fmt.Errorf("[%s] 迁移失败 %s: %w", mf.kind, mf.name, err)
		}
		latest = mf.version
	}
	return nil
}

// execFile 在事务中执行迁移文件并记录版本号。
//
// 分布式锁已保证同一时刻只有一个实例执行迁移；这里的幂等兜底只覆盖锁保护之外的场景
// （例如历史遗留、手工误操作导致的结构已存在），命中"对象已存在"类错误时记录警告后
// 视为已应用，其余错误仍然中断迁移。
func execFile(db *gorm.DB, mf *migFile) error {
	content, err := os.ReadFile(mf.path)
	if err != nil {
		return fmt.Errorf("读取文件: %w", err)
	}

	dialect, _ := lookupDialect(GetConfig().DataSource.DBType)

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(string(content)).Error; err != nil {
			if mf.kind == migrationTypeSeed || dialect == nil || !dialect.IsIdempotentSkippable(err) {
				return fmt.Errorf("执行SQL: %w", err)
			}
			resource.Logger.Warnf(context.Background(),
				"[migrate] %s 执行报重复对象错误，视为已应用: %v", mf.name, err)
		}
		if err := tx.Exec(
			`INSERT INTO schema_migrations (version, type) VALUES (?, ?)`,
			mf.version, mf.kind,
		).Error; err != nil {
			return fmt.Errorf("记录版本: %w", err)
		}
		return nil
	})
}

// ── schema_migrations 查询 ──

func latestVersion(db *gorm.DB, kind migrationType) (string, error) {
	var v string
	err := db.Raw(
		`SELECT COALESCE(MAX(version), '') FROM schema_migrations WHERE type = ?`,
		kind,
	).Scan(&v).Error
	return v, err
}

func isApplied(db *gorm.DB, version string, kind migrationType) (bool, error) {
	var n int64
	err := db.Raw(
		`SELECT COUNT(1) FROM schema_migrations WHERE version = ? AND type = ?`,
		version, kind,
	).Scan(&n).Error
	return n > 0, err
}

func withErrorLogger(db *gorm.DB) *gorm.DB {
	return db.Session(&gorm.Session{Logger: db.Logger.LogMode(logger.Error)})
}

// ── ensureDB 辅助 ──

// toGormxConfig 将 DataSource 转为 gormx.Config，数据库名替换为管理库名。
// DBName 使用固定值避免与业务连接冲突，用完即 Close。
func toGormxConfig(cfg DataSource, creator dbCreator) *gormx.Config {
	args := make([]gormx.ARG, len(cfg.Args))
	for i, a := range cfg.Args {
		args[i] = gormx.ARG{Name: a.Name, Value: a.Value}
	}
	return &gormx.Config{
		DBName: "__ensure_db__",
		Connection: gormx.Connection{
			DBType:   gormx.DatabaseType(cfg.DBType),
			Host:     cfg.Host,
			Port:     cfg.Port,
			Username: cfg.Username,
			Password: cfg.Password,
			Database: creator.AdminDatabase(),
			Args:     args,
		},
		LogMode: gormx.LogModeError,
		Logger:  gormx.LoggerAdapter(resource.Logger.WithCallerSkip(3)),
	}
}

// dbExists 通过 GORM 查询目标数据库是否存在。
func dbExists(db *gorm.DB, creator dbCreator, dbName string) bool {
	q := creator.DBExistsQuery()
	if q == "" {
		return false
	}
	var n int
	if err := db.Raw(q, dbName).Scan(&n).Error; err != nil {
		return false
	}
	return n > 0
}
