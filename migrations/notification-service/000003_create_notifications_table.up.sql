-- +migrate Up

-- 1. Notifications Table
CREATE TABLE IF NOT EXISTS notifications (
    id          UUID PRIMARY KEY,
    user_id     UUID NOT NULL,           -- The recipient
    message_id  UUID,                    -- Optional: link to the specific message
    sender_id   UUID NOT NULL,           -- Who triggered the notification
    content     TEXT NOT NULL,
    chat_id     UUID,                    -- Context (which group or DM)
    type        VARCHAR(50) DEFAULT 'message',
    is_read     BOOLEAN DEFAULT FALSE,   -- Renamed from 'read' as READ is often a reserved word
    created_at  TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- 2. Indexes for fast lookups
-- Essential for your getUnreadCount and getNotificationsByUser endpoints
CREATE INDEX idx_notifications_user_id ON notifications(user_id);

-- Composite index to speed up "get all unread for user" queries
CREATE INDEX idx_notifications_user_unread ON notifications(user_id) WHERE is_read = FALSE;

