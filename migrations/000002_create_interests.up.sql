CREATE TABLE user_interests (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID REFERENCES users(id) ON DELETE CASCADE,
    category   VARCHAR(32) NOT NULL,
    value      VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT now(),
    UNIQUE(user_id, category, value)
);
