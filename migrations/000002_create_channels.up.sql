CREATE TABLE IF NOT EXISTS chats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    is_group BOOLEAN DEFAULT FALSE,
    owner_id UUID NOT NULL,
    max_members INTEGER DEFAULT 1000,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chat_participants (
    chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,
    joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (chat_id, user_id)
);

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sender_id UUID NOT NULL,
    chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    status VARCHAR(50) DEFAULT 'sent',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);




-- -- 1. Chats must exist first
-- CREATE TABLE IF NOT EXISTS chats (
--     id UUID PRIMARY KEY, -- Remove DEFAULT if you generate IDs in Go
--     name VARCHAR(255),
--     owner_id UUID NOT NULL, -- The user who created it
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
-- );

-- -- 2. Participants link Users to Chats
-- CREATE TABLE IF NOT EXISTS chat_participants (
--     chat_id UUID REFERENCES chats(id) ON DELETE CASCADE,
--     user_id UUID NOT NULL, 
--     joined_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
--     PRIMARY KEY (chat_id, user_id)
-- );

-- -- 3. Messages depend on both
-- CREATE TABLE IF NOT EXISTS messages (
--     id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
--     sender_id UUID NOT NULL,
--     chat_id UUID NOT NULL,
--     content TEXT NOT NULL,
--     status VARCHAR(20) DEFAULT 'sent',
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
--     -- Add this constraint only if chats are on the same shard!
--     CONSTRAINT fk_chat FOREIGN KEY(chat_id) REFERENCES chats(id) ON DELETE CASCADE
-- );