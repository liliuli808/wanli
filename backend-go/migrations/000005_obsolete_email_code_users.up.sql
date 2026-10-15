-- 作废通过邮箱验证码创建的存量开发账号：将 hashed_password 置为不可验证哨兵，
-- bcrypt 比较恒失败，账号无法再登录，但保留用户行以维持外键完整性。
-- 用户需重新注册（预发布阶段无真实用户）。
UPDATE users
SET hashed_password = '!code-era-obsolete',
    updated_at = CURRENT_TIMESTAMP;
