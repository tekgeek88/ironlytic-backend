DROP INDEX IF EXISTS users_email_lower_uq;
DROP TABLE IF EXISTS users;

-- Keeping pgcrypto is usually fine; drop if you want a pristine down:
-- DROP EXTENSION IF EXISTS pgcrypto;