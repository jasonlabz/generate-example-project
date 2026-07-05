package bootstrap

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gorm.io/gorm"

	"github.com/jasonlabz/potato/gormx"

	"github.com/jasonlabz/generate-example-project/global/resource"
)

// ── 常量 ──

const migrationTableSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version VARCHAR(255) PRIMARY KEY,
	applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`

const (
	hdrVersion  = "-- @version " // 版本号声明，如 -- @version 20240701_001
	hdrBaseline = "-- @baseline" // 基线标记（仅基线文件使用）
)

// ── 数据结构 ──

// migFile 单个迁移文件的元信息。
// version 来自文件头部的 -- @version，是迁移追踪的唯一标识。
// name 是文件名，仅用于日志显示。
type migFile struct {
	name     string
	version  string
	baseline bool
}

// ── 公开入口 ──

// ensureDB 检查目标数据库是否存在，不存在则创建。
// 使用 database/sql 直连管理库（postgres / mysql / master），不依赖 GORM。
func ensureDB(ctx context.Context) {
	cfg := GetConfig().DataSource
	if !cfg.Enable {
		return
	}

	dsn := buildAdminDSN(cfg)
	if dsn == "" {
		resource.Logger.Warn(ctx, "[ensureDB] 无法构建管理连接 DSN，跳过")
		return
	}

	raw, err := sql.Open(cfg.DBType, dsn)
	if err != nil {
		resource.Logger.Errorf(ctx, "[ensureDB] 连接服务器失败: %v", err)
		return
	}
	defer raw.Close()

	ok, err := dbExists(raw, cfg.DBType, cfg.Database)
	if err != nil {
		resource.Logger.Errorf(ctx, "[ensureDB] 查询数据库失败: %v", err)
		return
	}
	if ok {
		return
	}

	if _, err := raw.Exec(createDBSQL(cfg.DBType, cfg.Database)); err != nil {
		resource.Logger.Errorf(ctx, "[ensureDB] 创建数据库失败: %v", err)
		return
	}
	resource.Logger.Infof(ctx, "[ensureDB] 数据库 %s 已创建", cfg.Database)
}

// runMigrations 执行表结构迁移（仅 DDL，不含种子数据）。
//
// 版本号统一来自文件头部的 -- @version，不依赖文件名。
//
// 策略：
//   - 新库（schema_migrations 为空）→ 执行基线 → 跳过版本 ≤ 基线的增量 → 执行剩余增量
//   - 已有库 → 只执行版本 > 最新已应用版本的增量
func runMigrations(ctx context.Context) {
	cfg := GetConfig().DataSource
	if !cfg.Enable {
		return
	}

	db := gormx.DefaultMaster()
	if err := db.Exec(migrationTableSQL).Error; err != nil {
		resource.Logger.Errorf(ctx, "[migrate] 创建追踪表失败: %v", err)
		return
	}

	files := loadMigrations(ctx, "conf/migrations")
	if len(files) == 0 {
		return
	}

	// 拆分基线 / 增量
	var baseline *migFile
	var incs []*migFile
	for i := range files {
		if files[i].baseline {
			baseline = &files[i]
		} else {
			incs = append(incs, &files[i])
		}
	}

	latest := latestVersion(db)

	if latest == "" {
		if baseline == nil {
			resource.Logger.Error(ctx, "[migrate] 缺少基线文件（头部需包含 -- @baseline）")
			return
		}
		resource.Logger.Infof(ctx, "[migrate] 执行基线 %s (版本 %s)", baseline.name, baseline.version)
		if err := execFile(db, baseline); err != nil {
			resource.Logger.Errorf(ctx, "[migrate] 基线失败: %v", err)
			return
		}
		latest = baseline.version
	}

	for _, mf := range incs {
		if mf.version <= latest {
			continue
		}
		done, err := isApplied(db, mf.version)
		if err != nil {
			resource.Logger.Errorf(ctx, "[migrate] 查询状态失败 %s: %v", mf.name, err)
			return
		}
		if done {
			continue
		}
		resource.Logger.Infof(ctx, "[migrate] 执行 %s (版本 %s)", mf.name, mf.version)
		if err := execFile(db, mf); err != nil {
			resource.Logger.Errorf(ctx, "[migrate] 迁移失败 %s: %v", mf.name, err)
			return
		}
	}
}

// runSeed 在所有 DDL 迁移完成后执行种子数据。
//
// 种子数据不记录版本号，每次启动都执行。
// INSERT 必须使用 ON CONFLICT ... DO NOTHING 保证幂等。
// 执行失败不阻塞启动。
func runSeed(ctx context.Context) {
	cfg := GetConfig().DataSource
	if !cfg.Enable {
		return
	}

	names := listSQLFiles("conf/seed")
	if len(names) == 0 {
		return
	}

	db := gormx.DefaultMaster()
	for _, name := range names {
		path := filepath.Join("conf", "seed", name)
		content, err := os.ReadFile(path)
		if err != nil {
			resource.Logger.Errorf(ctx, "[seed] 读取 %s 失败: %v", name, err)
			continue
		}
		if err := db.Exec(string(content)).Error; err != nil {
			resource.Logger.Warnf(ctx, "[seed] %s 执行失败(已跳过): %v", name, err)
			continue
		}
		resource.Logger.Infof(ctx, "[seed] %s 已执行", name)
	}
}

// ── 文件加载与解析 ──

// loadMigrations 扫描目录、解析头部、按版本号排序。
// 缺少 -- @version 的文件会被跳过并告警。
func loadMigrations(ctx context.Context, dir string) []migFile {
	names := listSQLFiles(dir)
	files := make([]migFile, 0, len(names))

	for _, name := range names {
		mf, err := parseMeta(filepath.Join(dir, name), name)
		if err != nil {
			resource.Logger.Warnf(ctx, "[migrate] 跳过 %s: %v", name, err)
			continue
		}
		files = append(files, mf)
	}

	sort.Slice(files, func(i, j int) bool {
		return files[i].version < files[j].version
	})
	return files
}

// parseMeta 读取 SQL 文件头部，提取 -- @version 和 -- @baseline。
// version 是必需字段，缺失则返回 error。
func parseMeta(path, name string) (migFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return migFile{}, fmt.Errorf("无法打开: %w", err)
	}
	defer f.Close()

	mf := migFile{name: name}

	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if mf.version == "" && strings.HasPrefix(line, hdrVersion) {
			mf.version = strings.TrimSpace(strings.TrimPrefix(line, hdrVersion))
		}
		if !mf.baseline && line == hdrBaseline {
			mf.baseline = true
		}
		// 两个标记都找到了就可以提前退出
		if mf.version != "" && mf.baseline {
			break
		}
		// 遇到非注释非空行说明头部结束（可选优化）
		if mf.version != "" && line != "" && !strings.HasPrefix(line, "--") {
			break
		}
	}

	if mf.version == "" {
		return migFile{}, fmt.Errorf("缺少 %s 声明", hdrVersion)
	}
	return mf, nil
}

// listSQLFiles 返回目录下所有 .sql 文件名（仅文件名，不含路径），按名称排序。
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

// execFile 在事务中执行迁移文件并记录版本号。
func execFile(db *gorm.DB, mf *migFile) error {
	path := filepath.Join("conf", "migrations", mf.name)
	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件: %w", err)
	}

	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(string(content)).Error; err != nil {
			return fmt.Errorf("执行SQL: %w", err)
		}
		if err := tx.Exec(
			`INSERT INTO schema_migrations (version) VALUES (?)`, mf.version,
		).Error; err != nil {
			return fmt.Errorf("记录版本: %w", err)
		}
		return nil
	})
}

// ── schema_migrations 查询 ──

func latestVersion(db *gorm.DB) string {
	var v string
	if err := db.Raw(
		`SELECT COALESCE(MAX(version), '') FROM schema_migrations`,
	).Scan(&v).Error; err != nil {
		return ""
	}
	return v
}

func isApplied(db *gorm.DB, version string) (bool, error) {
	var n int64
	err := db.Raw(
		`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version,
	).Scan(&n).Error
	return n > 0, err
}

// ── ensureDB 辅助 ──

func buildAdminDSN(cfg DataSource) string {
	switch cfg.DBType {
	case string(gormx.DatabaseTypePostgres):
		return buildPGDSN(cfg, "postgres")
	case string(gormx.DatabaseTypeMySQL):
		return buildMySQLDSN(cfg, "")
	case string(gormx.DatabaseTypeSqlserver):
		return buildMSSQLDSN(cfg, "master")
	}
	return ""
}

func buildPGDSN(cfg DataSource, db string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "user=%s password=%s host=%s port=%d dbname=%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, db)
	for _, a := range cfg.Args {
		fmt.Fprintf(&b, " %s=%s", a.Name, a.Value)
	}
	return b.String()
}

func buildMySQLDSN(cfg DataSource, db string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%s@tcp(%s:%d)/%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, db)
	if len(cfg.Args) > 0 {
		b.WriteByte('?')
		for i, a := range cfg.Args {
			if i > 0 {
				b.WriteByte('&')
			}
			fmt.Fprintf(&b, "%s=%s", a.Name, a.Value)
		}
	}
	return b.String()
}

func buildMSSQLDSN(cfg DataSource, db string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "user id=%s;password=%s;server=%s;port=%d;database=%s",
		cfg.Username, cfg.Password, cfg.Host, cfg.Port, db)
	for _, a := range cfg.Args {
		fmt.Fprintf(&b, ";%s=%s", a.Name, a.Value)
	}
	return b.String()
}

func dbExists(raw *sql.DB, dbType, dbName string) (bool, error) {
	q := dbExistsQuery(dbType)
	if q == "" {
		return false, nil
	}
	var n int
	err := raw.QueryRow(q, dbName).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func dbExistsQuery(dbType string) string {
	switch dbType {
	case string(gormx.DatabaseTypePostgres):
		return `SELECT 1 FROM pg_database WHERE datname = $1`
	case string(gormx.DatabaseTypeMySQL):
		return `SELECT 1 FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?`
	case string(gormx.DatabaseTypeSqlserver):
		return `SELECT 1 FROM sys.databases WHERE name = @p1`
	}
	return ""
}

func createDBSQL(dbType, dbName string) string {
	switch dbType {
	case string(gormx.DatabaseTypePostgres):
		return fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)
	case string(gormx.DatabaseTypeMySQL):
		return fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4", dbName)
	case string(gormx.DatabaseTypeSqlserver):
		return fmt.Sprintf("CREATE DATABASE [%s]", dbName)
	}
	return ""
}
