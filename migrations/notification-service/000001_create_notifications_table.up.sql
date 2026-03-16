CREATE TABLE IF NOT EXISTS notifications (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL,        -- Le destinataire
    message_id UUID,             -- Référence au message source (optionnel)
    sender_id UUID NOT NULL,     -- Qui a envoyé le message/déclenché la notification
    content TEXT NOT NULL,       -- Le texte affiché (ex: "X vous a envoyé un message")
    chat_id UUID,               -- Pour rediriger l'utilisateur vers la bonne discussion
    type VARCHAR(30),           -- "message", "call", "group_invite", etc.
    read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index pour charger rapidement les notifications d'un utilisateur spécifique
CREATE INDEX IF NOT EXISTS idx_notifications_user_id ON notifications(user_id);
-- Index pour trier par date (pour les listes de notifications récentes)
CREATE INDEX IF NOT EXISTS idx_notifications_created_at ON notifications(created_at);