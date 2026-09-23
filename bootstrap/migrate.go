package bootstrap

import (
	"context"
	"errors"
	"fmt"

	"github.com/jasonlabz/potato/gormx"
	"github.com/jasonlabz/potato/gormx/migrate"

	"github.com/jasonlabz/generate-example-project/common/resource"
)

// migrationLockKey 固定字符串，标识"本项目的迁移锁"，避免与其他咨询锁资源冲突。
const migrationLockKey = "generate-example-project:schema-migrations"

// ensureDB 检查目标数据库是否存在，不存在则创建。
// 建库逻辑在 potato/gormx/migrate.EnsureDatabase，这里只做配置转换与日志。
func ensureDB(ctx context.Context) {
	cfg := GetConfig().DataSource
	if !cfg.Enable {
		return
	}

	conn := gormx.Connection{
		DBType:   gormx.DatabaseType(cfg.DBType),
		Host:     cfg.Host,
		Port:     cfg.Port,
		Username: cfg.Username,
		Password: cfg.Password,
	}
	for _, a := range cfg.Args {
		conn.Args = append(conn.Args, gormx.ARG{Name: a.Name, Value: a.Value})
	}

	created, err := migrate.EnsureDatabase(ctx, conn, cfg.Database, resource.Logger)
	switch {
	case errors.Is(err, migrate.ErrAutoCreateUnsupported):
		resource.Logger.Warnf(ctx,
			"[ensureDB] 数据库类型 %s 不支持自动创建，请手动创建 %s 库后重启",
			cfg.DBType, cfg.Database)
	case err != nil:
		resource.Logger.Errorf(ctx, "[ensureDB] %v", err)
	case created:
		resource.Logger.Infof(ctx, "[ensureDB] 数据库 %s 已创建", cfg.Database)
	}
}

// runMigrations 执行 DDL 迁移和种子数据（先 DDL 后 seed）。
// 迁移实现已下沉到 potato/gormx/migrate，DDL 失败维持 panic 语义（启动失败）。
func runMigrations(ctx context.Context) {
	cfg := GetConfig().DataSource
	if !cfg.Enable {
		return
	}

	err := migrate.Run(ctx, gormx.DefaultMaster(), migrate.Config{
		DBType:  gormx.DatabaseType(cfg.DBType),
		DDLDir:  "conf/migrations",
		SeedDir: "conf/seed",
		LockKey: migrationLockKey,
		Logger:  resource.Logger,
	})
	if err != nil {
		panic(fmt.Errorf("%w", err))
	}
}
