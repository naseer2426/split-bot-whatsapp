CREATE TABLE IF NOT EXISTS polls (
    id SERIAL PRIMARY KEY,
    message_keys TEXT[] NOT NULL DEFAULT '{}',
    options_meta JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE IF NOT EXISTS votes (
    id SERIAL PRIMARY KEY,
    poll_id INTEGER NOT NULL REFERENCES polls (id) ON DELETE CASCADE,
    user_id VARCHAR NOT NULL,
    votes TEXT[] NOT NULL DEFAULT '{}'
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_votes_poll_id_user_id ON votes (poll_id, user_id);

CREATE INDEX IF NOT EXISTS idx_votes_poll_id ON votes (poll_id);
CREATE INDEX IF NOT EXISTS idx_votes_user_id ON votes (user_id);
