-- We use UUID for IDs to match your User and Message services
CREATE TABLE IF NOT EXISTS chats (
    id UUID PRIMARY KEY,
    name VARCHAR(255),           -- Name for group chats, null for private
    type VARCHAR(20) NOT NULL,   -- 'private' (2 people) or 'group' (many)
    participants UUID[] NOT NULL, -- Array of User IDs
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Crucial for performance: This index allows fast searching 
-- for "Give me all chats where User X is a participant"
CREATE INDEX IF NOT EXISTS idx_chats_participants ON chats USING GIN (participants);