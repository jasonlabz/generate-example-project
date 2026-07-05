-- @version 20240701_001
-- 增量迁移示例：给 users 表增加 email 字段

-- ALTER TABLE users ADD COLUMN IF NOT EXISTS email VARCHAR(255);

-- 注意：种子数据（INSERT）请放在 conf/seed/ 目录下，不要放在这里
