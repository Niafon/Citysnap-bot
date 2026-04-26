CREATE TABLE matches (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user1_id   UUID REFERENCES users(id) ON DELETE CASCADE,
    user2_id   UUID REFERENCES users(id) ON DELETE CASCADE,
    is_active  BOOLEAN DEFAULT true,
    created_at TIMESTAMPTZ DEFAULT now()
);
