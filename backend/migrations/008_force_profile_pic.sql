ALTER TABLE conversations ADD COLUMN IF NOT EXISTS profile_picture_url TEXT;

CREATE TABLE IF NOT EXISTS conversation_profile_pics (
    conversation_id BIGINT REFERENCES conversations(id),
    profile_picture_url TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (conversation_id)
);
