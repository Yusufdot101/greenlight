CREATE TABLE IF NOT EXISTS tokens (
    hash bytea NOT NULL,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE,
    expiry TIMESTAMP(0) with time zone NOT NULL,
    scope text NOT NULL
);
