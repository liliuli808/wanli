ALTER TABLE users
    DROP COLUMN IF EXISTS bilibili_sessdata_enc,
    DROP COLUMN IF EXISTS bilibili_mid,
    DROP COLUMN IF EXISTS bilibili_uname,
    DROP COLUMN IF EXISTS bilibili_credential_updated_at;
