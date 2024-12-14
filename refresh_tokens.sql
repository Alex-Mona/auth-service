CREATE TABLE refresh_tokens (
    id SERIAL PRIMARY KEY,
    user_id UUID NOT NULL,
    refresh_token_hash TEXT NOT NULL,
    client_ip TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL
);