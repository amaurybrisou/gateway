CREATE TABLE refresh_tokens (
    user_id UUID REFERENCES "user" (id) ON DELETE CASCADE,
    refresh_token TEXT NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW() NOT NULL,
    PRIMARY KEY (user_id, refresh_token)
);