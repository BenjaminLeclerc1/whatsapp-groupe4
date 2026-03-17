CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    telephone VARCHAR(20) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password TEXT, -- On stockera ici le hash du mot de passe
    role VARCHAR(20) DEFAULT 'user',
    status VARCHAR(50) DEFAULT 'offline',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index pour accélérer la recherche par téléphone (très fréquent sur WhatsApp)
CREATE INDEX IF NOT EXISTS idx_users_telephone ON users(telephone);