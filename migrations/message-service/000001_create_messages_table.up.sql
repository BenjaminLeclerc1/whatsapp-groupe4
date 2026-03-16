-- Extension pour la génération automatique d'UUID si nécessaire
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id UUID NOT NULL,          -- Identifiant unique de la conversation
    sender_id UUID NOT NULL,        -- L'utilisateur qui envoie (référé via UserID dans ton middleware)
    receiver_id UUID NOT NULL,      -- Destinataire (User ou Groupe)
    content TEXT NOT NULL,          -- Le corps du message
    message_type VARCHAR(20) DEFAULT 'text', -- text, image, audio, video
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index pour charger rapidement les messages d'un chat par ordre chronologique
CREATE INDEX IF NOT EXISTS idx_messages_chat_id_created_at ON messages(chat_id, created_at);

-- Index pour faciliter la recherche des messages envoyés par un utilisateur
CREATE INDEX IF NOT EXISTS idx_messages_sender_id ON messages(sender_id);