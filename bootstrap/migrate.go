package bootstrap

import (
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

// 迁移记录表 DDL
const migrationTableSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version VARCHAR(255) PRIMARY KEY,
	applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`

// 基线文件名前缀，保证在所有增量文件之前排序
const baselinePrefix = "00000000_000"

var bgCtx = context.Background()

// ensureDB 检查数据库是否存在，不存在则创建
// 放在 initDB 之前调用，使用 database/sql 直连，不依赖 GORM
func ensureDB() {
	dbConf := GetConfig().DataSource
	if !dbConf.Enable {
		return
	}

	adminDSN := buildAdminDSN(dbConf)
	if adminDSN == "" {
		resource.Logger.Warn(bgCtx, "[ensureDB] 无法构建管理连接 DSN，跳过数据库检查")
		return
	}

	rawDB, err := sql.Open(string(dbConf.DBType), adminDSN)
	if err != nil {
		resource.Logger.Errorf(bgCtx, "[ensureDB] 连接数据库服务器失败: %v", err)
		return
	}
	defer rawDB.Close()

	exists, err := checkDB(rawDB, dbConf.DBType, dbConf.Database)
	if err != nil {
		resource.Logger.Errorf(bgCtx, "[ensureDB] 检查数据库存在性失败: %v", err)
		return
	}
	if exists {
		return
	}

	sqlStr := createDBSQL(dbConf.DBType, dbConf.Database)
	if _, err := rawDB.Exec(sqlStr); err != nil {
		resource.Logger.Errorf(bgCtx, "[ensureDB] 创建数据库失败: %v", err)
		return
	}
	resource.Logger.Infof(bgCtx, "[ensureDB] 数据库 %s 创建成功", dbConf.Database)
}

// runMigrations 自动执行数据库迁移
//
// 策略：
//   - 首次启动（schema_migrations 为空）→ 先执行基线完整 SQL，再追增量文件
//   - 后续启动（已有记录）→ 只执行最新版本之后的增量文件
//
// 放在 initDB 之后调用，依赖已初始化的 GORM 连接
func runMigrations() {
	dbConf := GetConfig().DataSource
	if !dbConf.Enable {
		return
	}

	db := gormx.DefaultMaster()

	// 1. 确保迁移记录表存在
	if err := db.Exec(migrationTableSQL).Error; err != nil {
		resource.Logger.Errorf(bgCtx, "[migrate] 创建迁移记录表失败: %v", err)
		return
	}

	// 2. 收集并排序迁移文件
	files := scanMigrations()
	if len(files) == 0 {
		return
	}

	// 3. 获取当前最新已应用版本
	latest := latestVersion(db)
	hasRecords := latest != ""

	// 4. 定位基线文件
	baselineFile, incrementalFiles := splitBaseline(files)

	if !hasRecords {
		// 首次启动：先跑基线
		if baselineFile == "" {
			resource.Logger.Error(bgCtx, "[migrate] 新数据库缺少基线文件 (00000000_000_baseline.sql)")
			return
		}
		resource.Logger.Infof(bgCtx, "[migrate] 首次启动，执行基线: %s", baselineFile)
		if !execFile(db, baselineFile) {
			return
		}
		latest = baselineFile
	}

	// 5. 执行 latest 之后的增量文件
	for _, fname := range incrementalFiles {
		if fname <= latest {
			continue // 已通过基线或之前的迁移覆盖
		}
		applied, err := isApplied(db, fname)
		if err != nil {
			resource.Logger.Errorf(bgCtx, "[migrate] 检查迁移状态失败 %s: %v", fname, err)
			return
		}
		if applied {
			continue
		}
		resource.Logger.Infof(bgCtx, "[migrate] 执行增量: %s", fname)
		if !execFile(db, fname) {
			return
		}
	}
}

// execFile 在事务中执行单个迁移文件并记录版本
func execFile(db *gorm.DB, fname string) bool {
	content, err := os.ReadFile(filepath.Join("conf", "migrations", fname))
	if err != nil {
		resource.Logger.Errorf(bgCtx, "[migrate] 读取文件失败 %s: %v", fname, err)
		return false
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(string(content)).Error; err != nil {
			return err
		}
		return tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, fname).Error
	})
	if err != nil {
		resource.Logger.Errorf(bgCtx, "[migrate] 迁移失败 %s: %v", fname, err)
		return false
	}
	resource.Logger.Infof(bgCtx, "[migrate] 迁移完成: %s", fname)
	return true
}

// scanMigrations 扫描 conf/migrations/ 下所有 .sql 文件并排序
func scanMigrations() []string {
	migrationsDir := filepath.Join("conf", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return nil
	}

	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)
	return files
}

// splitBaseline 将文件列表拆分为 [基线文件, 增量文件...]
// 基线文件以 00000000_000 为前缀，约定只有一个
func splitBaseline(files []string) (baseline string, incremental []string) {
	for _, f := range files {
		if strings.HasPrefix(f, baselinePrefix) {
			baseline = f
		} else {
			incremental = append(incremental, f)
		}
	}
	return
}

// latestVersion 查询 schema_migrations 中最大的版本号
// 空表返回 ""
func latestVersion(db *gorm.DB) string {
	var version string
	err := db.Raw(`SELECT COALESCE(MAX(version), '') FROM schema_migrations`).Scan(&version).Error
	if err != nil || version == "" {
		return ""
	}
	return version
}

// isApplied 检查迁移文件是否已执行
func isApplied(db *gorm.DB, version string) (bool, error) {
	var count int64
	err := db.Raw(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version).
		Scan(&count).Error
	return count > 0, err
}

// ── ensureDB 辅助函数 ──

// buildAdminDSN 根据配置构建连接管理库的 DSN（不连目标业务库）
func buildAdminDSN(dbConf DataSource) string {
	switch dbConf.DBType {
	case string(gormx.DatabaseTypePostgres):
		return buildPGDSN(dbConf, "postgres")
	case string(gormx.DatabaseTypeMySQL):
		return buildMySQLDSN(dbConf, "")
	case string(gormx.DatabaseTypeSqlserver):
		return buildMSSQLDSN(dbConf, "master")
	default:
		return ""
	}
}

func buildPGDSN(dbConf DataSource, dbName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "user=%s password=%s host=%s port=%d dbname=%s",
		dbConf.Username, dbConf.Password, dbConf.Host, dbConf.Port, dbName)
	for _, arg := range dbConf.Args {
		fmt.Fprintf(&b, " %s=%s", arg.Name, arg.Value)
	}
	return b.String()
}

func buildMySQLDSN(dbConf DataSource, dbName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s:%s@tcp(%s:%d)/%s",
		dbConf.Username, dbConf.Password, dbConf.Host, dbConf.Port, dbName)
	if len(dbConf.Args) > 0 {
		b.WriteByte('?')
		for i, arg := range dbConf.Args {
			if i > 0 {
				b.WriteByte('&')
			}
			fmt.Fprintf(&b, "%s=%s", arg.Name, arg.Value)
		}
	}
	return b.String()
}

func buildMSSQLDSN(dbConf DataSource, dbName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "user id=%s;password=%s;server=%s;port=%d;database=%s",
		dbConf.Username, dbConf.Password, dbConf.Host, dbConf.Port, dbName)
	for _, arg := range dbConf.Args {
		fmt.Fprintf(&b, ";%s=%s", arg.Name, arg.Value)
	}
	return b.String()
}

// checkDB 查询目标数据库是否存在
func checkDB(db *sql.DB, dbType, dbName string) (bool, error) {
	var query string
	switch dbType {
	case string(gormx.DatabaseTypePostgres):
		query = `SELECT 1 FROM pg_database WHERE datname = $1`
	case string(gormx.DatabaseTypeMySQL):
		query = `SELECT 1 FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?`
	case string(gormx.DatabaseTypeSqlserver):
		query = `SELECT 1 FROM sys.databases WHERE name = @p1`
	default:
		return false, nil
	}

	var found int
	err := db.QueryRow(query, dbName).Scan(&found)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

// createDBSQL 生成建库 SQL
func createDBSQL(dbType, dbName string) string {
	switch dbType {
	case string(gormx.DatabaseTypePostgres):
		return fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)
	case string(gormx.DatabaseTypeMySQL):
		return fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4", dbName)
	case string(gormx.DatabaseTypeSqlserver):
		return fmt.Sprintf("CREATE DATABASE [%s]", dbName)
	default:
		return ""
	}
}
