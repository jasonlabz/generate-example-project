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

// migrationTableSQL 迁移记录表 DDL
const migrationTableSQL = `CREATE TABLE IF NOT EXISTS schema_migrations (
	version VARCHAR(255) PRIMARY KEY,
	applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`

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

// runMigrations 扫描 conf/migrations/ 下的 .sql 文件，按文件名顺序执行未应用的迁移
// 放在 initDB 之后调用，依赖已初始化的 GORM 连接
func runMigrations() {
	dbConf := GetConfig().DataSource
	if !dbConf.Enable {
		return
	}

	db := gormx.DefaultMaster()

	// 确保迁移记录表存在
	if err := db.Exec(migrationTableSQL).Error; err != nil {
		resource.Logger.Errorf(bgCtx, "[migrate] 创建迁移记录表失败: %v", err)
		return
	}

	// 扫描迁移文件目录
	migrationsDir := filepath.Join("conf", "migrations")
	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return // 目录不存在不报错
	}

	var sqlFiles []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			sqlFiles = append(sqlFiles, e.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, fname := range sqlFiles {
		applied, err := isApplied(db, fname)
		if err != nil {
			resource.Logger.Errorf(bgCtx, "[migrate] 检查迁移状态失败 %s: %v", fname, err)
			return
		}
		if applied {
			continue
		}

		content, err := os.ReadFile(filepath.Join(migrationsDir, fname))
		if err != nil {
			resource.Logger.Errorf(bgCtx, "[migrate] 读取文件失败 %s: %v", fname, err)
			return
		}

		resource.Logger.Infof(bgCtx, "[migrate] 执行迁移: %s", fname)

		// GORM 事务：自动处理不同数据库的占位符转换
		err = db.Transaction(func(tx *gorm.DB) error {
			if err := tx.Exec(string(content)).Error; err != nil {
				return err
			}
			return tx.Exec(`INSERT INTO schema_migrations (version) VALUES (?)`, fname).Error
		})
		if err != nil {
			resource.Logger.Errorf(bgCtx, "[migrate] 迁移失败 %s: %v", fname, err)
			return
		}
		resource.Logger.Infof(bgCtx, "[migrate] 迁移完成: %s", fname)
	}
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

// ── runMigrations 辅助函数 ──

// isApplied 检查迁移文件是否已执行（通过 GORM 自动转换占位符）
func isApplied(db *gorm.DB, version string) (bool, error) {
	var count int64
	err := db.Raw(`SELECT COUNT(1) FROM schema_migrations WHERE version = ?`, version).
		Scan(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
