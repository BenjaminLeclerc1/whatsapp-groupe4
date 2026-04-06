-- +migrate Up
CREATE TABLE IF NOT EXISTS messages (
    id          UUID PRIMARY KEY,           -- Using UUID for the string ID
    sender_id   UUID NOT NULL,              -- References your User/Account table
    chat_id     UUID NOT NULL,              -- References your Chat/Conversation table
    content     TEXT NOT NULL,
    status      VARCHAR(20) DEFAULT 'sent', -- e.g., 'sent', 'delivered', 'read'
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexing for performance
CREATE INDEX idx_messages_chat_id ON messages(chat_id);
CREATE INDEX idx_messages_created_at ON messages(created_at);

