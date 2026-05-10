-- votes: TEXT[] (flat hashes) -> JSONB map { "<poll_stanza_id>": ["<hex>", ...] }
-- Existing non-empty arrays are not convertible without stanza keys; rows are reset to {}.
ALTER TABLE votes
    ALTER COLUMN votes DROP DEFAULT;

ALTER TABLE votes
    ALTER COLUMN votes TYPE jsonb USING '{}'::jsonb;

ALTER TABLE votes
    ALTER COLUMN votes SET DEFAULT '{}'::jsonb;
