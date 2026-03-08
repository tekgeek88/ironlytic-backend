DROP INDEX IF EXISTS sessions_active_lookup_idx;
DROP INDEX IF EXISTS sessions_expires_at_idx;
DROP INDEX IF EXISTS sessions_user_id_idx;
DROP INDEX IF EXISTS sessions_token_hash_uq;

DROP TABLE IF EXISTS sessions;