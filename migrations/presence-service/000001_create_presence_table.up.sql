-- +migrate Up

-- 1. Presence Table
CREATE TABLE IF NOT EXISTS user_presence (
    user_id       UUID PRIMARY KEY,
    status        VARCHAR(20) NOT NULL DEFAULT 'offline', -- online, offline, typing
    last_seen     TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    chat_id       UUID, -- Used specifically for 'typing' status context
    updated_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Index for the Worker
-- The timeout worker will frequently scan for 'online' or 'typing' users to mark them offline.
CREATE INDEX idx_presence_status_activity ON user_presence(status, last_activity);

