-- +migrate Up

-- 1. Channels Table
CREATE TABLE IF NOT EXISTS channels (
    id           UUID PRIMARY KEY,
    name         VARCHAR(255) NOT NULL,
    description  TEXT,
    is_group     BOOLEAN DEFAULT FALSE,
    owner_id     UUID NOT NULL, -- References users(id)
    max_members  INTEGER DEFAULT 100,
    created_at   TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Participants Table (Junction table for Many-to-Many)
CREATE TABLE IF NOT EXISTS participants (
    chat_id      UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL, -- References users(id)
    joined_at    TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chat_id, user_id) -- Prevents a user from joining the same chat twice
);

-- 3. Messages Table (Updated to link to channels)
CREATE TABLE IF NOT EXISTS messages (
    id          UUID PRIMARY KEY,
    sender_id   UUID NOT NULL,
    chat_id     UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    status      VARCHAR(20) DEFAULT 'sent',
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_channels_owner_id ON channels(owner_id);
CREATE INDEX idx_participants_user_id ON participants(user_id);
CREATE INDEX idx_messages_chat_id_created_at ON messages(chat_id, created_at DESC);

