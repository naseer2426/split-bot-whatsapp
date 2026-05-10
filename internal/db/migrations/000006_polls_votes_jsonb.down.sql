ALTER TABLE votes
    ALTER COLUMN votes DROP DEFAULT;

ALTER TABLE votes
    ALTER COLUMN votes TYPE text[] USING '{}'::text[];

ALTER TABLE votes
    ALTER COLUMN votes SET DEFAULT '{}'::text[];
