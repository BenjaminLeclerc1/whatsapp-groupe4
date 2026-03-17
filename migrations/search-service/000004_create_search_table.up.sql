CREATE TABLE IF NOT EXISTS search_metadata (
    id SERIAL PRIMARY KEY,
    last_indexed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    total_messages_indexed INTEGER DEFAULT 0
);
-- We insert a seed row so we can update it later
INSERT INTO search_metadata (total_messages_indexed) VALUES (0);