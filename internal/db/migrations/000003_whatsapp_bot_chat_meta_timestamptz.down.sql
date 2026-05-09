ALTER TABLE whatsapp_bot_chat_meta
    ALTER COLUMN created_at TYPE timestamp without time zone USING created_at AT TIME ZONE 'Etc/GMT-8',
    ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP,
    ALTER COLUMN updated_at TYPE timestamp without time zone USING updated_at AT TIME ZONE 'Etc/GMT-8',
    ALTER COLUMN updated_at SET DEFAULT CURRENT_TIMESTAMP;
