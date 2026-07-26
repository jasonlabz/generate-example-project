package bootstrap

import (
	"fmt"
	"strings"

	"gorm.io/gorm"

	"github.com/jasonlabz/potato/gormx"
)

// migrateDialect 收敛迁移相关的方言差异：加锁、DDL 幂等判断。
// 注册到 dialectRegistry 的方言都必须支持迁移；建库能力是可选的，见 dbCreator。
type migrateDialect interface {
	// Locker 返回该方言的分布式锁实现，用于保证多实例并发启动时只有一个实例执行迁移。
	Locker(db *gorm.DB, lockKey string) migrateLocker
	// IsIdempotentSkippable 判断迁移执行失败的 err 是否属于"对象已存在"类幂等错误，
	// 仅作为分布式锁之外的兜底，不覆盖其余真实失败。
	IsIdempotentSkippable(err error) bool
}

// dbCreator 是可选的建库能力：连接管理库判断目标库是否存在、不存在则创建。
// 达梦等要求业务库由 DBA 预先手工初始化的方言不实现该接口，ensureDB 会跳过自动建库。
type dbCreator interface {
	// AdminDatabase 连接服务器（而非业务库）时使用的管理库名；返回空字符串表示不需要指定库（如 MySQL）。
	AdminDatabase() string
	// DBExistsQuery 判断目标库是否存在的查询语句，参数为库名。
	DBExistsQuery() string
	// CreateDatabaseSQL 返回建库语句。
	CreateDatabaseSQL(dbName string) string
}

var dialectRegistry = map[gormx.DatabaseType]migrateDialect{
	gormx.DatabaseTypePostgres:  postgresDialect{},
	gormx.DatabaseTypeMySQL:     mysqlDialect{},
	gormx.DatabaseTypeSqlserver: sqlserverDialect{},
	gormx.DatabaseTypeDM:        dmDialect{},
}

// lookupDialect 按数据库类型查找迁移方言实现；未注册的类型（如 oracle/sqlite）返回 ok=false。
func lookupDialect(dbType string) (migrateDialect, bool) {
	d, ok := dialectRegistry[gormx.DatabaseType(dbType)]
	return d, ok
}

// lookupDBCreator 按数据库类型查找建库能力；方言未注册或不支持自动建库都返回 ok=false。
func lookupDBCreator(dbType string) (dbCreator, bool) {
	d, ok := lookupDialect(dbType)
	if !ok {
		return nil, false
	}
	creator, ok := d.(dbCreator)
	return creator, ok
}

// ── PostgreSQL ──

type postgresDialect struct{}

func (postgresDialect) AdminDatabase() string { return "postgres" }

func (postgresDialect) DBExistsQuery() string {
	return `SELECT 1 FROM pg_database WHERE datname = ?`
}

func (postgresDialect) CreateDatabaseSQL(dbName string) string {
	return fmt.Sprintf(`CREATE DATABASE "%s"`, dbName)
}

func (postgresDialect) Locker(db *gorm.DB, lockKey string) migrateLocker {
	return newAdvisoryLocker(db, lockKey, postgresLockDialect{})
}

func (postgresDialect) IsIdempotentSkippable(err error) bool {
	if err == nil {
		return false
	}
	// 42P07 duplicate_table / 42701 duplicate_column / 42710 duplicate_object
	msg := err.Error()
	for _, code := range []string{"42P07", "42701", "42710"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}

// ── MySQL ──

type mysqlDialect struct{}

func (mysqlDialect) AdminDatabase() string { return "" }

func (mysqlDialect) DBExistsQuery() string {
	return `SELECT 1 FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = ?`
}

func (mysqlDialect) CreateDatabaseSQL(dbName string) string {
	return fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4", dbName)
}

func (mysqlDialect) Locker(db *gorm.DB, lockKey string) migrateLocker {
	return newAdvisoryLocker(db, lockKey, mysqlLockDialect{})
}

func (mysqlDialect) IsIdempotentSkippable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// 1050 table already exists / 1060 duplicate column name / 1061 duplicate key name
	for _, code := range []string{"1050", "1060", "1061"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}

// ── SQL Server ──

type sqlserverDialect struct{}

func (sqlserverDialect) AdminDatabase() string { return "master" }

func (sqlserverDialect) DBExistsQuery() string {
	return `SELECT 1 FROM sys.databases WHERE name = ?`
}

func (sqlserverDialect) CreateDatabaseSQL(dbName string) string {
	return fmt.Sprintf("CREATE DATABASE [%s]", dbName)
}

func (sqlserverDialect) Locker(db *gorm.DB, lockKey string) migrateLocker {
	return newAdvisoryLocker(db, lockKey, sqlserverLockDialect{})
}

func (sqlserverDialect) IsIdempotentSkippable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// 2714 对象已存在 / 1801 数据库已存在 / 21002 列名重复
	for _, code := range []string{"2714", "1801", "21002"} {
		if strings.Contains(msg, code) {
			return true
		}
	}
	return false
}

// ── 达梦（DM） ──

// dmDialect 只支持迁移，不支持自动建库：达梦实例通常由 DBA 用 dminit 预先初始化，
// 业务侵入式建库不是常规运维方式，因此不实现 dbCreator，ensureDB 会提示手动建库。
//
// 达梦没有确认可用的会话级咨询锁原语（DBMS_LOCK 兼容包是否内置取决于实例配置），
// 分布式锁改用不依赖方言特性的 tableLocker（基于表的乐观锁）。
type dmDialect struct{}

func (dmDialect) Locker(db *gorm.DB, lockKey string) migrateLocker {
	return newTableLocker(db, lockKey)
}

func (dmDialect) IsIdempotentSkippable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	// -2005 表或视图已存在；达梦错误信息也可能直接包含中英文提示
	for _, kw := range []string{"-2005", "already exists", "已存在"} {
		if strings.Contains(msg, kw) {
			return true
		}
	}
	return false
}
