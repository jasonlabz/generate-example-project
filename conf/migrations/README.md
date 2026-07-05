# 数据库迁移与种子数据规范

## 概述

系统启动时自动完成数据库创建、表结构迁移和种子数据填充。

```
MustInit()
  ├── ensureDB       → 数据库不存在则创建
  ├── initDB         → GORM 连接
  ├── runMigrations  → DDL 迁移（仅执行一次）
  └── runSeed        → 种子数据（每次启动都执行）
```

---

## 一、目录结构

```
conf/
├── migrations/          ← 表结构迁移（仅 DDL）
│   ├── baseline.sql     ← 基线：某个版本的完整建表快照
│   └── *.sql            ← 增量：单次表结构变更
└── seed/                ← 种子数据（仅 INSERT）
    └── *.sql            ← 按文件名排序执行
```

---

## 二、文件头部标记

### version 是头部的 `-- @version` 值，与文件名无关

```sql
-- @version YYYYMMDD_NNN    ← 必填，版本号。格式：日期_NNN
-- @baseline                 ← 仅基线文件需要，标识此文件是完整建表快照
```

| 标记 | 用途 | 示例 |
|------|------|------|
| `-- @version <版本号>` | **迁移文件必填**。版本号格式 `YYYYMMDD_NNN`，用于排序、去重、基线比较 | `-- @version 20240701_005` |
| `-- @baseline` | 仅基线文件使用，声明此文件是完整建表快照 | `-- @baseline` |

> **缺少 `-- @version` 的迁移文件会在启动时被跳过并告警。文件名只是标签，不参与版本判断。**

### 基线文件示例

```sql
-- @version 20240701_005
-- @baseline
-- 截至 20240701_005 的完整表结构

CREATE TABLE IF NOT EXISTS users (
    id         BIGSERIAL PRIMARY KEY,
    username   VARCHAR(100) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

### 增量文件示例

```sql
-- @version 20240701_006

ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255);
```

---

## 三、基线机制

### 为什么需要基线

项目运行一段时间后增量文件增多。基线是某个时间点的"完整建表快照"，新库直接执行基线，无需逐条重放历史。

### 基线版本号的作用

基线头部的 `-- @version` 的值表示"此快照已包含 ≤ 该版本的所有变更"。

**示例**：假设有以下文件，按 `@version` 排序：

```
baseline.sql               -- @version 20240701_005  @baseline
add_column_a.sql            -- @version 20240701_001
add_column_b.sql            -- @version 20240701_003
add_column_c.sql            -- @version 20240701_006
add_table_d.sql             -- @version 20240801_001
```

**新库首次启动：**

```
执行 baseline.sql           ← 版本 20240701_005，完整建表
跳过 20240701_001           ← ≤ 基线版本，已被覆盖
跳过 20240701_003           ← ≤ 基线版本，已被覆盖
执行 add_column_c.sql       ← 20240701_006 > 基线版本
执行 add_table_d.sql        ← 20240801_001 > 基线版本
```

**已有库后续启动：**

```
跳过 baseline.sql           ← 已在 schema_migrations
跳过 add_column_a/b/c.sql   ← 已执行
执行 新文件.sql             ← 仅执行新的
```

### 更新基线

增量文件堆积较多时，重新生成基线：

1. 导出当前数据库完整建表 SQL
2. 保存为新基线文件，头部写上当前最新版本号
3. 旧基线可保留（系统自动跳过）或删除

---

## 四、种子数据

### 约定

- 放在 `conf/seed/` 目录
- **每次启动都在 DDL 迁移之后执行**
- 不写入 `schema_migrations`（每次都会跑）
- INSERT 必须用 `ON CONFLICT ... DO NOTHING` 保证幂等
- 执行失败不阻塞启动

### 示例

```sql
-- conf/seed/001_default_roles.sql

INSERT INTO roles (code, name, description) VALUES
    ('R_SUPER', '超级管理员', '拥有所有权限'),
    ('R_ADMIN', '管理员', '拥有管理权限')
ON CONFLICT (code) DO NOTHING;
```

---

## 五、追踪表

系统自动维护 `schema_migrations`：

| 字段 | 说明 |
|------|------|
| `version` | 来自文件头 `-- @version` 的值 |
| `applied_at` | 执行时间 |

DDL 迁移执行后写入此表，下次启动查表跳过已执行的版本。

---

## 六、日常操作

| 场景 | 做法 |
|------|------|
| 新增表 | 新建 `.sql` 文件，头部写 `-- @version YYYYMMDD_NNN` |
| 修改表结构 | 同上，写 ALTER TABLE |
| 新增种子数据 | 在 `conf/seed/` 新增 `.sql` 文件 |
| 迁移文件太多 | 导出完整 DDL → 保存为带 `-- @baseline` 的新基线 |
