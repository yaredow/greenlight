CREATE TABLE refresh_tokens (
    hash BYTEA PRIMARY KEY,
    user_id bigint NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    expires_at timestamp(0) with time zone NOT NULL,
    created_at timestamp(0) with time zone NOT NULL DEFAULT NOW(),
    revoked_at timestamp(0) with time zone,
    family_id uuid NOT NULL,
    replaced_by_hash bytea
);

CREATE INDEX refresh_tokens_user_id_idx ON refresh_tokens (user_id);

CREATE INDEX refresh_tokens_expires_at_idx ON refresh_tokens (expires_at);

CREATE INDEX refresh_tokens_family_id_idx ON refresh_tokens (family_id);

