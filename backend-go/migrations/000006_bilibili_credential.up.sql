-- 用户级 B 站登录态：仅存储加密后的 SESSDATA 与展示用的账号信息。
ALTER TABLE users
    ADD COLUMN bilibili_sessdata_enc TEXT,
    ADD COLUMN bilibili_mid VARCHAR(32),
    ADD COLUMN bilibili_uname VARCHAR(100),
    ADD COLUMN bilibili_credential_updated_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN users.bilibili_sessdata_enc IS 'AES-GCM 加密后的 B 站 SESSDATA，仅后端可解密';
