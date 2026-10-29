-- Existing users retain null credentials until credentials are explicitly assigned.
ALTER TABLE users ADD COLUMN username varchar(100);
ALTER TABLE users ADD COLUMN password_hash bytea;
CREATE UNIQUE INDEX uq_users_username ON users(lower(username)) WHERE username IS NOT NULL;
ALTER TABLE users ADD CONSTRAINT ck_user_credentials CHECK (
 (username IS NULL AND password_hash IS NULL) OR
 (username IS NOT NULL AND length(trim(username)) > 0 AND password_hash IS NOT NULL AND octet_length(password_hash)=60)
);
