# 数据库迁移与种子数据规范

## 概述

系统启动时自动完成数据库创建、表结构迁移和种子数据填充。

```
MustInit()
  ├── ensureDB       → 数据库不存在则创建
  ├── initDB         → GORM 连接
  └── runMigrations  → 先执行 DDL，再执行 seed
```

---

## 一、目录结构

```
conf/
├── migrations/                     ← 表结构迁移（DDL）
│   ├── 00000000_000_baseline.sql   ← 基线：完整建表快照
│   └── YYYYMMDD_NNN_desc.sql       ← 增量：单次表结构变更
└── seed/                           ← 种子数据（seed）
    ├── 00000000_000_baseline.sql   ← 基线：完整种子数据
    └── YYYYMMDD_NNN_desc.sql       ← 增量：单次种子数据变更
```

---

## 二、文件类型与版本号

文件类型由所在目录决定，不在 SQL 文件中声明类型：

| 目录 | 类型 |
|------|------|
| `conf/migrations/` | `ddl` |
| `conf/seed/` | `seed` |

版本号用于排序、去重和基线比较：

1. DDL 和 seed 文件名都必须使用 `YYYYMMDD_NNN_desc.sql` 格式。
2. 普通文件优先使用头部 `-- @version <版本号>`，缺失时从文件名前缀提取
   `YYYYMMDD_NNN`。
3. baseline 必须在文件头部使用 `-- @version <版本号>` 声明其覆盖版本。
4. 无法解析版本号的 SQL 文件会被跳过并告警。

### 示例

```sql
-- @version 20240701_006
-- 头部声明版本号（推荐），同时文件名前缀也可作为兜底

ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255);
```

---

## 三、基线文件

### 约定

- 文件名**以 `00000000_000` 开头**的 baseline 有且仅有一个
- 内容是某个时间点的完整建表 SQL
- 基线版本由头部 `-- @version` 声明，代表"此快照已覆盖到该版本"
- `schema_migrations` 由运行时创建，不写入任何 baseline
- baseline 中的建表、索引和其他对象应使用 `IF NOT EXISTS`；不支持时使用
  `IF EXISTS` 或条件判断保证可重复执行

### 基线版本号的含义

基线的 `-- @version` 值告诉系统：**≤ 该版本的所有增量变更已包含在基线中**。

**示例**：假设有以下文件

```
00000000_000_baseline.sql       ← @version 20240701_005（基线，由文件名前缀识别）
20240701_001_add_a.sql          ← 版本 20240701_001（文件名解析）
20240701_003_add_b.sql          ← 版本 20240701_003（文件名解析）
20240701_006_add_c.sql          ← 版本 20240701_006（文件名解析）
20240801_001_add_d.sql          ← 版本 20240801_001（文件名解析）
```

**新库首次启动：**

```
执行 00000000_000_baseline.sql  ← 基线，完整建表
跳过 20240701_001               ← ≤ 20240701_005，已被基线覆盖
跳过 20240701_003               ← ≤ 20240701_005，已被基线覆盖
执行 20240701_006_add_c.sql     ← > 20240701_005
执行 20240801_001_add_d.sql     ← > 20240701_005
```

**已有库后续启动：** 读取 `type=ddl` 的最新版本，只执行版本更高且未记录的新 DDL 文件。

### 更新基线

增量文件太多时，重新生成基线：

1. 导出当前数据库完整建表 SQL
2. 另存为新的 `00000000_000_*.sql`，头部写上当前最新版本号
3. 删除旧基线，其他增量文件可保留（会被自动跳过）或删除

---

## 四、种子数据

- 放在 `conf/seed/` 目录，包含一个 `00000000_000_baseline.sql` 和按版本递增的
  `YYYYMMDD_NNN_desc.sql`
- 加载器与 DDL 共用扫描、解析、排序、事务和追踪逻辑，类型由目录决定
- 新项目部署或没有 `type=seed` 记录时，先执行 seed baseline，再执行版本更高的 seed 文件
- 已有 seed 记录时，读取 `type=seed` 的最新版本，只执行版本更高且未记录的 seed 文件
- SQL 必须幂等：默认值可用 `ON CONFLICT ... DO NOTHING`，需要同步内置定义时使用 `ON CONFLICT ... DO UPDATE`
- seed 执行失败只记录告警，不阻塞服务启动

```sql
-- conf/seed/20260903_001_default_roles.sql
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
| `version` | 文件中的原始版本号 |
| `type` | 执行类型：`ddl` 或 `seed`；与 `version` 共同构成唯一记录 |
| `applied_at` | 执行时间 |

---

## 六、日常操作

| 场景 | 做法 |
|------|------|
| 新增表 / 修改表结构 | 新建 `YYYYMMDD_NNN_desc.sql`，头部加 `-- @version YYYYMMDD_NNN` |
| 新增种子数据 | 在 `conf/seed/` 新建 `YYYYMMDD_NNN_desc.sql`，保证 SQL 可重复执行 |
| 更新基线 | 导出完整 DDL，替换 `00000000_000_*.sql` |
