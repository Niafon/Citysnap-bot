CREATE TABLE photo_reactions (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    photo_id      UUID REFERENCES daily_photos(id) ON DELETE CASCADE,
    user_id       UUID REFERENCES users(id) ON DELETE CASCADE,
    reaction_type VARCHAR(16) NOT NULL,
    created_at    TIMESTAMPTZ DEFAULT now(),
    UNIQUE(photo_id, user_id)
);
